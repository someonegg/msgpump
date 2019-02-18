// Copyright 2019 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpeer

import (
	"context"
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
	h      Handler
	idle   time.Duration
	entryC chan entry
}

// AsyncHandler convert a handler to asynchronous mode, then each call is initiated
// from a separate worker goroutine.
func AsyncHandler(h Handler, workerIdleTimeout time.Duration) Handler {
	return &asyncHandler{
		h:      h,
		idle:   workerIdleTimeout,
		entryC: make(chan entry),
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

func (h *asyncHandler) work(e entry) {
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
