// Copyright 2019 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpeer_test

import (
	"context"
	"fmt"
	"github.com/someonegg/gox/syncx"
	"github.com/someonegg/msgpump"
	"github.com/someonegg/msgpump/msgpeer"
	"log"
	"net"
	"testing"
	"time"
)

const TheAddr = "127.0.0.1:7000"
const WriteQueueSize = 100

type ClientPeer struct {
	peer *msgpeer.Peer
}

func (p *ClientPeer) Start(conn net.Conn) {
	mrw := msgpump.NetconnMRW(conn)

	p.peer = msgpeer.NewPeer(mrw, p, WriteQueueSize)
	p.peer.Start(nil)
}

func (p *ClientPeer) Hello(r []byte) {
	resp, err := p.peer.Do(context.Background(), "client-hello", r)
	log.Printf("client-hello response: %s, %v", resp, err)
}

func (p *ClientPeer) Ask(r []byte) {
	resp, err := p.peer.Do(context.Background(), "client-ask", r)
	log.Printf("client-ask response: %s, %v", resp, err)
}

func (p *ClientPeer) Bye(r []byte) {
	resp, err := p.peer.Do(context.Background(), "client-bye", r)
	log.Printf("client-bye response: %s, %v", resp, err)
	p.peer.Stop()
}

func (p *ClientPeer) Process(ctx context.Context, t string, r msgpeer.Request, w msgpeer.ResponseWriter) {
	log.Printf("client process request: %v, %s", t, r)

	switch t {
	case "server-ask":
		p.OnAsk(ctx, r, w)
	default:
		log.Print("unknown server request")
	}
}

func (p *ClientPeer) OnAsk(ctx context.Context, r []byte, w msgpeer.ResponseWriter) {
	w(ctx, r)
}

func (p *ClientPeer) OnNotify(ctx context.Context, t string, n msgpeer.Notify) {
	log.Printf("client receive notify: %v, %s", t, n)
}

func (p *ClientPeer) WaitStop() {
	select {
	case <-p.peer.StopD():
	}
}

func client() {
	conn, err := net.Dial("tcp", TheAddr)
	if err != nil {
		log.Fatal(err)
	}

	p := &ClientPeer{}
	p.Start(conn)

	p.Hello([]byte("aaa"))

	p.peer.Notify(context.Background(), "client-notify", []byte("nnn"))

	for i := 0; i < 3; i++ {
		p.Ask([]byte(fmt.Sprint("bbb", i)))
		time.Sleep(time.Millisecond)
	}

	p.Bye([]byte("ccc"))

	p.WaitStop()

	log.Printf("client peer stop, error: %v", p.peer.Error())
}

type ServerPeer struct {
	peer *msgpeer.Peer
}

func (p *ServerPeer) Start(conn net.Conn) {
	mrw := msgpump.NetconnMRW(conn)

	h := msgpeer.ParallelHandler(p, 5*time.Second, func(v interface{}) {
		log.Println("catch panic:", v)
	})
	p.peer = msgpeer.NewPeer(mrw, h, WriteQueueSize)
	p.peer.Start(nil)
}

func (p *ServerPeer) Ask(r []byte) {
	resp, err := p.peer.Do(context.Background(), "server-ask", r)
	log.Printf("server-ask response: %s, %v", resp, err)
}

func (p *ServerPeer) Process(ctx context.Context, t string, r msgpeer.Request, w msgpeer.ResponseWriter) {
	log.Printf("server process request: %v, %s", t, r)

	switch t {
	case "client-hello":
		p.OnHello(ctx, r, w)
	case "client-ask":
		p.OnAsk(ctx, r, w)
	case "client-bye":
		p.OnBye(ctx, r, w)
	default:
		log.Print("unknown client request")
	}
}

func (p *ServerPeer) OnHello(ctx context.Context, r []byte, w msgpeer.ResponseWriter) {
	w(ctx, []byte("AAA"))
}

func (p *ServerPeer) OnAsk(ctx context.Context, r []byte, w msgpeer.ResponseWriter) {
	w(ctx, r)

	// must use ParallelHandler
	p.Ask([]byte("BBB0"))
}

func (p *ServerPeer) OnBye(ctx context.Context, r []byte, w msgpeer.ResponseWriter) {
	w(ctx, []byte("CCC"))
	panic("bye")
}

func (p *ServerPeer) OnNotify(ctx context.Context, t string, n msgpeer.Notify) {
	log.Printf("server receive notify: %v, %s", t, n)
}

func (p *ServerPeer) WaitStop() {
	select {
	case <-p.peer.StopD():
	}
}

func server(listenD syncx.DoneChan) {
	l, err := net.Listen("tcp", TheAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	listenD.SetDone()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			p := &ServerPeer{}
			p.Start(conn)

			p.peer.Notify(context.Background(), "server-notify", []byte("NNN"))
			time.Sleep(time.Millisecond)

			p.Ask([]byte("BBB1"))

			p.WaitStop()

			log.Printf("server peer stop, error: %v", p.peer.Error())
		}()
	}
}

func TestExample(t *testing.T) {
	listenD := syncx.NewDoneChan()
	go server(listenD)
	<-listenD

	client()
	time.Sleep(50 * time.Millisecond)
}
