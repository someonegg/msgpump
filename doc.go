// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpump provides a message-pump facility.
//
// The message-pump will continuously receive, process and send messages after startup.
//
// The message is defined as variable-length byte array, they are distinguished
// by the message-type (a string).
//
// The transport layer is defined by the MessageReadWriter interface, there are
// two default implementations:
//   NetconnMRW over net.Conn
//   WebsocketMRW over websocket.Conn
//
// Here is a quick example, includes client and server.
//
// Client
//  type ClientPeer struct {
//  	pump *msgpump.Pump
//
//  	helloD syncx.DoneChan
//  }
//
//  func (p *ClientPeer) Start() {
//  	conn, err := net.Dial("tcp", TheAddr)
//  	if err != nil {
//  		log.Fatal(err)
//  	}
//
//  	mrw := msgpump.NetconnMRW(conn)
//
//  	p.pump = msgpump.NewPump(mrw, p, WriteQueueSize)
//  	p.pump.Start(nil)
//  }
//
//  func (p *ClientPeer) Hello(m msgpump.Message) {
//  	p.pump.Output("client-hello", m)
//  }
//
//  func (p *ClientPeer) Ask(m msgpump.Message) {
//  	p.pump.Output("client-ask", m)
//  }
//
//  func (p *ClientPeer) Bye(m msgpump.Message) {
//  	p.pump.Output("client-bye", m)
//  }
//
//  func (p *ClientPeer) Process(ctx context.Context, t string, m msgpump.Message) {
//  	log.Printf("client receive message: %v, %v", t, string(m))
//
//  	switch t {
//  	case "server-hello":
//  		p.OnServerHello(m)
//  	case "server-answer":
//  		p.OnServerAnswer(m)
//  	case "server-bye":
//  		p.OnServerBye(m)
//  	default:
//  		log.Print("unknown server message")
//  	}
//  }
//
//  func (p *ClientPeer) OnServerHello(m msgpump.Message) {
//  	p.helloD.SetDone()
//  }
//
//  func (p *ClientPeer) OnServerAnswer(m msgpump.Message) {
//  }
//
//  func (p *ClientPeer) OnServerBye(m msgpump.Message) {
//  	p.pump.Stop()
//  }
//
//  func (p *ClientPeer) WaitStop() {
//  	select {
//  	case <-p.pump.StopD():
//  	}
//  }
//
//  func client() {
//  	p := &ClientPeer{helloD: syncx.NewDoneChan()}
//
//  	p.Start()
//
//  	p.Hello([]byte("aaa"))
//
//  	<-p.helloD
//
//  	for i := 0; i < 3; i++ {
//  		p.Ask([]byte(fmt.Sprint("bbb", i)))
//  		time.Sleep(time.Millisecond)
//  	}
//
//  	p.Bye([]byte("ccc"))
//
//  	p.WaitStop()
//
//  	log.Printf("client stop, error: %v", p.pump.Error())
//  }
//
// Server
//  type ServerPeer struct {
//  	pump *msgpump.Pump
//  }
//
//  func (p *ServerPeer) Start(conn net.Conn) {
//  	mrw := msgpump.NetconnMRW(conn)
//
//  	p.pump = msgpump.NewPump(mrw, p, WriteQueueSize)
//  	p.pump.Start(nil)
//  }
//
//  func (p *ServerPeer) Hello(m msgpump.Message) {
//  	p.pump.Output("server-hello", m)
//  }
//
//  func (p *ServerPeer) Answer(m msgpump.Message) {
//  	p.pump.Output("server-answer", m)
//  }
//
//  func (p *ServerPeer) Bye(m msgpump.Message) {
//  	p.pump.Output("server-bye", m)
//  }
//
//  func (p *ServerPeer) Process(ctx context.Context, t string, m msgpump.Message) {
//  	log.Printf("server receive message: %v, %v", t, string(m))
//
//  	switch t {
//  	case "client-hello":
//  		p.OnClientHello(m)
//  	case "client-ask":
//  		p.OnClientAsk(m)
//  	case "client-bye":
//  		p.OnClientBye(m)
//  	default:
//  		log.Print("unknown client message")
//  	}
//  }
//
//  func (p *ServerPeer) OnClientHello(m msgpump.Message) {
//  	p.Hello(m)
//  }
//
//  func (p *ServerPeer) OnClientAsk(m msgpump.Message) {
//  	p.Answer(m)
//  }
//
//  func (p *ServerPeer) OnClientBye(m msgpump.Message) {
//  	p.Bye(m)
//  }
//
//  func (p *ServerPeer) WaitStop() {
//  	select {
//  	case <-p.pump.StopD():
//  	}
//  }
//
//  func server() {
//  	l, err := net.Listen("tcp", TheAddr)
//  	if err != nil {
//  		log.Fatal(err)
//  	}
//  	defer l.Close()
//
//  	for {
//  		conn, err := l.Accept()
//  		if err != nil {
//  			log.Fatal(err)
//  		}
//
//  		p := &ServerPeer{}
//  		p.Start(conn)
//  		go func() {
//  			p.WaitStop()
//  			log.Printf("sclient stop, error: %v", p.pump.Error())
//  		}()
//  	}
//  }
package msgpump
