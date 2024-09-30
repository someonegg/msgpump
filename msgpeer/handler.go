// Copyright 2024 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpeer

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/someonegg/msgpump/v2"
)

type entry struct {
	ctx context.Context
	m   msgpump.Message
	w   ResponseWriter // not nil if request
}

type parallHandler struct {
	h        Handler
	idle     time.Duration
	panicLog func(interface{})
	entryC   chan entry
}

// ParallelHandler convert a handler to parallel mode, in which each call
// will be initiated from a different worker goroutine.
func ParallelHandler(h Handler, workerIdleTimeout time.Duration,
	workerPanicLog func(panicV interface{})) Handler {

	if workerPanicLog == nil {
		workerPanicLog = theWorkerPanicLogFunc
	}

	return &parallHandler{
		h:        h,
		idle:     workerIdleTimeout,
		panicLog: workerPanicLog,
		entryC:   make(chan entry),
	}
}

func (h *parallHandler) Process(ctx context.Context, r Request, w ResponseWriter) {
	h.parall(entry{ctx, r, w})
}

func (h *parallHandler) OnNotify(ctx context.Context, n Notify) {
	h.parall(entry{ctx, n, nil})
}

func (h *parallHandler) parall(e entry) {
	select {
	case <-e.ctx.Done():
	case h.entryC <- e:
	default:
		go h.work(e)
	}
}

// The default worker panic log function.
func theWorkerPanicLogFunc(v interface{}) {
	const size = 16 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	log.Print("worker panic: ", v, fmt.Sprintf("\n%s", buf))
}

func (h *parallHandler) work(e entry) {
	defer func() {
		if e := recover(); e != nil {
			h.panicLog(e)
		}
	}()

	h.handle(e)

	t := time.NewTimer(h.idle)

	for q := false; !q; {
		select {
		case e = <-h.entryC:
			h.handle(e)

			if !t.Stop() {
				<-t.C
			}
			t.Reset(h.idle)
		case <-t.C:
			q = true
		}
	}
}

func (h *parallHandler) handle(e entry) {
	if e.w != nil {
		h.h.Process(e.ctx, e.m, e.w)
	} else {
		h.h.OnNotify(e.ctx, e.m)
	}
}
