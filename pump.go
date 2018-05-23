// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync/atomic"

	"github.com/someonegg/gox/syncx"
)

var (
	errUnknownPanic = errors.New("unknown panic")
)

type legalPanic struct {
	err error
}

// Handler is the message processor.
//
// Process should complete the message processing as soon as possible, and
// it is not valid to access the message after the Process call.
type Handler interface {
	Process(ctx context.Context, t string, m Message)
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as message handlers.  If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// PumperHandler object that calls f.
type HandlerFunc func(ctx context.Context, t string, m Message)

// Process calls f(ctx, t, m).
func (f HandlerFunc) Process(ctx context.Context, t string, m Message) {
	f(ctx, t, m)
}

type Statistics struct {
	// from MessageReadWriter
	ReadedCount int64
	ReadedBytes int64

	// to MessageReadWriter
	WrittenCount int64
	WrittenBytes int64

	// Output call
	OutputCount int64
}

// Pump represents a message-pump, it has a working loop which reads
// and writes messages parallelly and continuously.
//
// Pump supports concurrently access.
type Pump struct {
	err   error
	quitF context.CancelFunc
	stopD syncx.DoneChan

	rw MessageReadWriter
	h  Handler
	sn StopNotifier

	// read
	rerr error
	rD   syncx.DoneChan
	// write
	werr error
	wD   syncx.DoneChan
	wQ   chan msgEntry

	stat Statistics

	panicLogF func(interface{})
}

// NewPump allocates and returns a new Pump.
//
// If rw implementes the StopNotifier interface, it will be called when
// the working loop exiting.
func NewPump(rw MessageReadWriter, h Handler, writeQueueSize int) *Pump {
	sn, _ := rw.(StopNotifier)
	return &Pump{
		stopD: syncx.NewDoneChan(),

		rw: rw,
		h:  h,
		sn: sn,

		rD: syncx.NewDoneChan(),
		wD: syncx.NewDoneChan(),
		wQ: make(chan msgEntry, writeQueueSize),

		panicLogF: thePanicLogFunc,
	}
}

// The default panic log function.
func thePanicLogFunc(v interface{}) {
	const size = 16 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Print("pump panic: ", v, fmt.Sprintf("\n%s", buf))
}

// SetPanicLogFunc is optional.
func (p *Pump) SetPanicLogFunc(f func(panicV interface{})) {
	p.panicLogF = f
}

// Start will start the working loop.
func (p *Pump) Start(parent context.Context) {
	if parent == nil {
		parent = context.Background()
	}

	var ctx context.Context
	ctx, p.quitF = context.WithCancel(parent)

	go p.reading(ctx)
	go p.writing(ctx)
	go p.monitor(ctx)
}

func (p *Pump) monitor(ctx context.Context) {
	defer p.ending()

	select {
	case <-ctx.Done():
	case <-p.rD:
	case <-p.wD:
	}
}

func (p *Pump) ending() {
	if e := recover(); e != nil {
		legal := false
		switch v := e.(type) {
		case legalPanic:
			legal = true
			p.err = v.err
		case error:
			p.err = v
		default:
			p.err = errUnknownPanic
		}
		if !legal && p.panicLogF != nil {
			p.panicLogF(e)
		}
	}

	defer func() { recover() }()
	defer p.stopD.SetDone()

	// if ending from error.
	p.quitF()

	if p.sn != nil {
		p.sn.OnStop()
	}

	<-p.rD
	<-p.wD
}

func (p *Pump) reading(ctx context.Context) {
	defer func() {
		if e := recover(); e != nil {
			legal := false
			switch v := e.(type) {
			case legalPanic:
				legal = true
				p.err = v.err
			case error:
				p.rerr = v
			default:
				p.rerr = errUnknownPanic
			}
			if !legal && p.panicLogF != nil {
				p.panicLogF(e)
			}
		}

		p.rD.SetDone()
	}()

	for q := false; !q; {
		t, m := p.readMessage()

		p.h.Process(ctx, t, m)

		select {
		case <-ctx.Done():
			q = true
		default:
		}
	}
}

func (p *Pump) readMessage() (string, Message) {
	t, m, err := p.rw.ReadMessage()
	if err != nil {
		panic(legalPanic{err})
	}
	atomic.AddInt64(&p.stat.ReadedCount, 1)
	atomic.AddInt64(&p.stat.ReadedBytes, int64(len(m)))
	return t, m
}

type msgEntry struct {
	t string
	m Message
}

func (p *Pump) writing(ctx context.Context) {
	defer func() {
		if e := recover(); e != nil {
			legal := false
			switch v := e.(type) {
			case legalPanic:
				legal = true
				p.err = v.err
			case error:
				p.werr = v
			default:
				p.werr = errUnknownPanic
			}
			if !legal && p.panicLogF != nil {
				p.panicLogF(e)
			}
		}

		p.wD.SetDone()
	}()

	for q := false; !q; {
		select {
		case <-ctx.Done():
			q = true
		case e := <-p.wQ:
			p.writeMessage(e.t, e.m)
		}
	}
}

func (p *Pump) writeMessage(t string, m Message) {
	err := p.rw.WriteMessage(t, m)
	if err != nil {
		panic(legalPanic{err})
	}
	atomic.AddInt64(&p.stat.WrittenCount, 1)
	atomic.AddInt64(&p.stat.WrittenBytes, int64(len(m)))
}

// Stop requests to stop the pump, the working loop will stop asynchronously.
func (p *Pump) Stop() {
	p.quitF()
}

// StopD returns a done channel, it will be signaled when the pump is stopped.
func (p *Pump) StopD() syncx.DoneChanR {
	return p.stopD.R()
}

func (p *Pump) Stopped() bool {
	return p.stopD.R().Done()
}

// Error can only be called after pump stopped.
func (p *Pump) Error() error {
	if p.err != nil {
		return p.err
	}
	if p.rerr != nil {
		return p.rerr
	}
	return p.werr
}

// Output puts the message to the write queue.
func (p *Pump) Output(t string, m Message) {
	select {
	case p.wQ <- msgEntry{t, m}:
		atomic.AddInt64(&p.stat.OutputCount, 1)
	case <-p.stopD:
	}
}

// TryOutput tries to put the message to the write queue.
func (p *Pump) TryOutput(t string, m Message) bool {
	select {
	case p.wQ <- msgEntry{t, m}:
		atomic.AddInt64(&p.stat.OutputCount, 1)
		return true
	default:
		return false
	}
}

func (p *Pump) Statistics() Statistics {
	return Statistics{
		ReadedCount:  atomic.LoadInt64(&p.stat.ReadedCount),
		ReadedBytes:  atomic.LoadInt64(&p.stat.ReadedBytes),
		WrittenCount: atomic.LoadInt64(&p.stat.WrittenCount),
		WrittenBytes: atomic.LoadInt64(&p.stat.WrittenBytes),
		OutputCount:  atomic.LoadInt64(&p.stat.OutputCount),
	}
}

// UnderlyingMRW returns the internal message readwriter.
func (p *Pump) UnderlyingMRW() MessageReadWriter {
	return p.rw
}
