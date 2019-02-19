// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump_test

import (
	"context"
	"fmt"
	"github.com/someonegg/gox/syncx"
	"github.com/someonegg/msgpump"
	"log"
	"net"
	"testing"
	"time"
)

const TheAddr = "127.0.0.1:7000"
const WriteQueueSize = 100

type ClientPeer struct {
	pump *msgpump.Pump

	helloD syncx.DoneChan
}

func (p *ClientPeer) Start(conn net.Conn) {
	mrw := msgpump.NetconnMRW(conn)

	p.pump = msgpump.NewPump(mrw, p, WriteQueueSize)
	p.pump.Start(nil)
}

func (p *ClientPeer) Hello(m msgpump.Message) {
	p.pump.Output("client-hello", m)
}

func (p *ClientPeer) Ask(m msgpump.Message) {
	p.pump.Output("client-ask", m)
}

func (p *ClientPeer) Bye(m msgpump.Message) {
	p.pump.Output("client-bye", m)
}

func (p *ClientPeer) Process(ctx context.Context, t string, m msgpump.Message) {
	log.Printf("client receive message: %v, %v", t, string(m))

	switch t {
	case "server-hello":
		p.OnServerHello(m)
	case "server-answer":
		p.OnServerAnswer(m)
	case "server-bye":
		p.OnServerBye(m)
	default:
		log.Print("unknown server message")
	}
}

func (p *ClientPeer) OnServerHello(m msgpump.Message) {
	p.helloD.SetDone()
}

func (p *ClientPeer) OnServerAnswer(m msgpump.Message) {
}

func (p *ClientPeer) OnServerBye(m msgpump.Message) {
	p.pump.Stop()
}

func (p *ClientPeer) WaitStop() {
	select {
	case <-p.pump.StopD():
	}
}

func client() {
	conn, err := net.Dial("tcp", TheAddr)
	if err != nil {
		log.Fatal(err)
	}

	p := &ClientPeer{helloD: syncx.NewDoneChan()}
	p.Start(conn)

	p.Hello([]byte("aaa"))

	<-p.helloD

	for i := 0; i < 3; i++ {
		p.Ask([]byte(fmt.Sprint("bbb", i)))
		time.Sleep(time.Millisecond)
	}

	p.Bye([]byte("ccc"))

	p.WaitStop()

	log.Printf("client stop, error: %v", p.pump.Error())
}

type ServerPeer struct {
	pump *msgpump.Pump
}

func (p *ServerPeer) Start(conn net.Conn) {
	mrw := msgpump.NetconnMRW(conn)

	p.pump = msgpump.NewPump(mrw, p, WriteQueueSize)
	p.pump.Start(nil)
}

func (p *ServerPeer) Hello(m msgpump.Message) {
	p.pump.Output("server-hello", m)
}

func (p *ServerPeer) Answer(m msgpump.Message) {
	p.pump.Output("server-answer", m)
}

func (p *ServerPeer) Bye(m msgpump.Message) {
	p.pump.Output("server-bye", m)
}

func (p *ServerPeer) Process(ctx context.Context, t string, m msgpump.Message) {
	log.Printf("server receive message: %v, %v", t, string(m))

	switch t {
	case "client-hello":
		p.OnClientHello(m)
	case "client-ask":
		p.OnClientAsk(m)
	case "client-bye":
		p.OnClientBye(m)
	default:
		log.Print("unknown client message")
	}
}

func (p *ServerPeer) OnClientHello(m msgpump.Message) {
	p.Hello(m)
}

func (p *ServerPeer) OnClientAsk(m msgpump.Message) {
	p.Answer(m)
}

func (p *ServerPeer) OnClientBye(m msgpump.Message) {
	p.Bye(m)
}

func (p *ServerPeer) WaitStop() {
	select {
	case <-p.pump.StopD():
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

		p := &ServerPeer{}
		p.Start(conn)
		go func() {
			p.WaitStop()
			log.Printf("sclient stop, error: %v", p.pump.Error())
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
