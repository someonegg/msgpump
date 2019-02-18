// Copyright 2019 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpeer implements a synchronous request response model over message-pump.
// It considers the client and server peers, allowing each to send requests to the other.
package msgpeer

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/someonegg/msgpump"
)

type Request = msgpump.Message
type Response = msgpump.Message
type Notify = msgpump.Message

type ResponseWriter func(ctx context.Context, resp Response) error

// Handler is the request processor.
//
// Handler should return as soon as possible, it is valid to read the
// message or use the ResponseWriter after returning.
type Handler interface {
	Process(ctx context.Context, t string, r Request, w ResponseWriter)
	// notify message
	OnNotify(ctx context.Context, t string, n Notify)
	// unknown message
	OnUnknown(ctx context.Context, t string, m msgpump.Message)
}

type Peer struct {
	*msgpump.Pump
	h Handler

	locker sync.Mutex
	nrid   uint32
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
func (p *Peer) Do(ctx context.Context, t string, r Request) (Response, error) {
	respC := make(chan Response, 1)

	p.locker.Lock()
	rid := strconv.FormatUint(uint64(p.nrid), 16)
	p.nrid += 1
	p.resps[rid] = respC
	added := true
	p.locker.Unlock()

	defer func() {
		if added {
			p.locker.Lock()
			delete(p.resps, rid)
			p.locker.Unlock()
		}
	}()

	mt := "R," + t + "," + rid
	err := p.Pump.Post(ctx, mt, r)
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
func (p *Peer) Notify(ctx context.Context, t string, n Notify) error {
	mt := "N," + t
	return p.Pump.Post(ctx, mt, n)
}

// Process implements the msgpump.Handler interface.
//
// The format of the message-type string t is:
//   R,request-type,request-id  for request
//   P,request-type,request-id  for response
//   N,notify-type              for notify
func (p *Peer) Process(ctx context.Context, mt string, msg msgpump.Message) {
	ss := strings.SplitN(mt, ",", 4)
	switch ss[0] {
	case "R":
		t, rid := ss[1], ss[2]
		p.h.Process(ctx, t, msg,
			func(ctx context.Context, resp Response) error {
				mt := "P," + t + "," + rid
				return p.Pump.Post(ctx, mt, resp)
			})
	case "P":
		rid := ss[2]
		p.locker.Lock()
		respC := p.resps[rid]
		if respC != nil {
			select {
			case respC <- msg:
			default:
			}
			delete(p.resps, rid)
		}
		p.locker.Unlock()
	case "N":
		t := ss[1]
		p.h.OnNotify(ctx, t, msg)
	default:
		p.h.OnUnknown(ctx, mt, msg)
	}
}
