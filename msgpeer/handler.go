// Copyright 2019 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpeer

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/someonegg/msgpump"
)

type entry struct {
	ctx context.Context
	t   string
	m   msgpump.Message
	w   ResponseWriter // not nil if request
}

type asyncHandler struct {
	h        Handler
	idle     time.Duration
	panicLog func(interface{})
	entryC   chan entry
}

// AsyncHandler convert a handler to asynchronous mode, each call is initiated
// from a separate worker goroutine.
func AsyncHandler(h Handler, workerIdleTimeout time.Duration,
	workerPanicLog func(panicV interface{})) Handler {

	if workerPanicLog == nil {
		workerPanicLog = theWorkerPanicLogFunc
	}

	return &asyncHandler{
		h:        h,
		idle:     workerIdleTimeout,
		panicLog: workerPanicLog,
		entryC:   make(chan entry),
	}
}

func (h *asyncHandler) Process(ctx context.Context, t string, r Request, w ResponseWriter) {
	h.async(entry{ctx, t, r, w})
}

func (h *asyncHandler) OnNotify(ctx context.Context, t string, n Notify) {
	h.async(entry{ctx, t, n, nil})
}

func (h *asyncHandler) async(e entry) {
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

func (h *asyncHandler) work(e entry) {
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

func (h *asyncHandler) handle(e entry) {
	if e.w != nil {
		h.h.Process(e.ctx, e.t, e.m, e.w)
	} else {
		h.h.OnNotify(e.ctx, e.t, e.m)
	}
}
