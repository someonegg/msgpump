// Copyright 2024 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpeer

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/someonegg/msgpump/v2"
)

type Request = msgpump.Message
type Response = msgpump.Message
type Notify = msgpump.Message

type ResponseWriter func(ctx context.Context, resp Response) error

// Handler is the request processor.
//
// Processor is called serially, so it should return as soon as possible.
// You can process requests asynchronously if necessary, it is safe to read
// them after returning.
//
// See ParallelHandler too.
type Handler interface {
	Process(ctx context.Context, r Request, w ResponseWriter)
	// notify message
	OnNotify(ctx context.Context, n Notify)
}

type Peer struct {
	*msgpump.Pump
	h Handler

	locker sync.Mutex
	nrid   uint64
	resps  map[string]chan Response
}

// NewPeer will create the message-pump with rw and writeQueueSize.
//
// The write methods of msgpump.Pump like Output and Post should not be called.
func NewPeer(rw msgpump.MessageReadWriter, h Handler, writeQueueSize int) *Peer {
	p := &Peer{
		h:     h,
		resps: make(map[string]chan Response),
	}
	p.Pump = msgpump.NewPump(rw, p, writeQueueSize)
	return p
}

// Do will send the request and wait for a response.
func (p *Peer) Do(ctx context.Context, r Request) (Response, error) {
	respC := make(chan Response, 1)

	p.locker.Lock()
	p.nrid++
	rid := strconv.FormatUint(p.nrid, 16)
	p.resps[rid] = respC
	p.locker.Unlock()

	added := true
	defer func() {
		if added {
			p.locker.Lock()
			delete(p.resps, rid)
			p.locker.Unlock()
		}
	}()

	err := p.Pump.OutputMP(ctx, msgpump.MPMessage{requestHeader(rid), r})
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.Pump.StopD():
		return nil, msgpump.ErrPumpStopped
	case resp := <-respC:
		added = false
		return resp, nil
	}
}

// Notify will post the notify.
func (p *Peer) Notify(ctx context.Context, n Notify) error {
	h := []byte("N\n")
	return p.Pump.OutputMP(ctx, msgpump.MPMessage{h, n})
}

// Process implements the msgpump.Handler interface.
//
// The format of the message header is:
//
//	R,request-id\n    for request
//	P,request-id\n    for response
//	N\n               for notify
func (p *Peer) Process(ctx context.Context, m msgpump.Message) {
	const MaxHeader = 128

	var h []byte
	var r []byte
	for i := 0; i < len(m) && i < MaxHeader; i++ {
		if m[i] == '\n' {
			h = m[0:i]
			r = m[i+1:]
			break
		}
	}

	if len(h) == 0 {
		return
	}
	if len(h) == 1 && h[0] == 'N' {
		p.h.OnNotify(ctx, r)
		return
	}

	ss := strings.SplitN(string(h), ",", 3)

	switch ss[0] {
	case "R":
		rid := ss[1]
		p.h.Process(ctx, r,
			func(ctx context.Context, resp Response) error {
				return p.Pump.OutputMP(ctx, msgpump.MPMessage{responseHeader(rid), resp})
			})
	case "P":
		rid := ss[1]
		p.locker.Lock()
		respC := p.resps[rid]
		if respC != nil {
			select {
			case respC <- r:
			default:
			}
			delete(p.resps, rid)
		}
		p.locker.Unlock()
	}
}

func requestHeader(rid string) []byte {
	l := len(rid) + 3
	h := make([]byte, l)
	h[0] = 'R'
	h[1] = ','
	copy(h[2:], rid)
	h[l-1] = '\n'
	return h
}

func responseHeader(rid string) []byte {
	l := len(rid) + 3
	h := make([]byte, l)
	h[0] = 'P'
	h[1] = ','
	copy(h[2:], rid)
	h[l-1] = '\n'
	return h
}
