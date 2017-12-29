// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpump provides a message-pump facility.
//
// The message-pump is a facility which will receive and dispatch messages
// continuously after startup. It will also send messages continuously and
// asynchronously after startup.
//
// The message is defined as variable-length byte array, they are distinguished
// by the message-type (a string).
//
// Here is a quick example, includes client and server.
//
// Client
//  type Client struct {
//  	pump *msgpump.Pump
//
//  	helloD syncx.DoneChan
//  }
//
//  func (c *Client) Start() {
//  	conn, err := net.Dial("tcp", TheAddr)
//  	if err != nil {
//  		log.Fatal(err)
//  	}
//
//  	mrw := msgpump.NetconnMRW(conn)
//
//  	c.pump = msgpump.NewPump(mrw, c, WriteQueueSize)
//  	c.pump.Start(nil)
//  }
//
//  func (c *Client) Hello(m msgpump.Message) {
//  	c.pump.Output("client-hello", m)
//  }
//
//  func (c *Client) Ask(m msgpump.Message) {
//  	c.pump.Output("client-ask", m)
//  }
//
//  func (c *Client) Bye(m msgpump.Message) {
//  	c.pump.Output("client-bye", m)
//  }
//
//  func (c *Client) Process(ctx context.Context, t string, m msgpump.Message) {
//  	log.Printf("client receive message: %v, %v", t, string(m))
//
//  	switch t {
//  	case "server-hello":
//  		c.OnServerHello(m)
//  	case "server-answer":
//  		c.OnServerAnswer(m)
//  	case "server-bye":
//  		c.OnServerBye(m)
//  	default:
//  		log.Print("unknown server message")
//  	}
//  }
//
//  func (c *Client) OnServerHello(m msgpump.Message) {
//  	c.helloD.SetDone()
//  }
//
//  func (c *Client) OnServerAnswer(m msgpump.Message) {
//  }
//
//  func (c *Client) OnServerBye(m msgpump.Message) {
//  	c.pump.Stop()
//  }
//
//  func (c *Client) WaitStop() {
//  	select {
//  	case <-c.pump.StopD():
//  	}
//  }
//
//  func client() {
//  	c := &Client{helloD: syncx.NewDoneChan()}
//
//  	c.Start()
//
//  	c.Hello([]byte("aaa"))
//
//  	<-c.helloD
//
//  	for i := 0; i < 3; i++ {
//  		c.Ask([]byte(fmt.Sprint("bbb", i)))
//  		time.Sleep(time.Millisecond)
//  	}
//
//  	c.Bye([]byte("ccc"))
//
//  	c.WaitStop()
//
//  	log.Printf("client stop, error: %v", c.pump.Error())
//  }
//
// Server
//  type SClient struct {
//  	pump *msgpump.Pump
//  }
//
//  func (c *SClient) Start(conn net.Conn) {
//  	mrw := msgpump.NetconnMRW(conn)
//
//  	c.pump = msgpump.NewPump(mrw, c, WriteQueueSize)
//  	c.pump.Start(nil)
//  }
//
//  func (c *SClient) Hello(m msgpump.Message) {
//  	c.pump.Output("server-hello", m)
//  }
//
//  func (c *SClient) Answer(m msgpump.Message) {
//  	c.pump.Output("server-answer", m)
//  }
//
//  func (c *SClient) Bye(m msgpump.Message) {
//  	c.pump.Output("server-bye", m)
//  }
//
//  func (c *SClient) Process(ctx context.Context, t string, m msgpump.Message) {
//  	log.Printf("server receive message: %v, %v", t, string(m))
//
//  	switch t {
//  	case "client-hello":
//  		c.OnClientHello(m)
//  	case "client-ask":
//  		c.OnClientAsk(m)
//  	case "client-bye":
//  		c.OnClientBye(m)
//  	default:
//  		log.Print("unknown client message")
//  	}
//  }
//
//  func (c *SClient) OnClientHello(m msgpump.Message) {
//  	c.Hello(m)
//  }
//
//  func (c *SClient) OnClientAsk(m msgpump.Message) {
//  	c.Answer(m)
//  }
//
//  func (c *SClient) OnClientBye(m msgpump.Message) {
//  	c.Bye(m)
//  }
//
//  func (c *SClient) WaitStop() {
//  	select {
//  	case <-c.pump.StopD():
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
//  		c := &SClient{}
//  		c.Start(conn)
//  		go func() {
//  			c.WaitStop()
//  			log.Printf("sclient stop, error: %v", c.pump.Error())
//  		}()
//  	}
//  }
package msgpump
