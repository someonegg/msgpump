// Copyright 2024 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpeer implements a synchronous request response model over message-pump.
// It considers the client and server peers, allowing each to send requests to the
// other concurrently.
//
// Here is a quick example, includes client and server.
//
// Server
//
//	const TheAddr = "127.0.0.1:7000"
//	const WriteQueueSize = 100
//
//	type ServerPeer struct {
//		peer *msgpeer.Peer
//	}
//
//	func (p *ServerPeer) Start(conn net.Conn) {
//		mrw := msgpump.NetconnMRW(conn)
//
//		h := msgpeer.ParallelHandler(p, 5*time.Second, func(v interface{}) {
//			log.Println("catch panic:", v)
//		})
//		p.peer = msgpeer.NewPeer(mrw, h, WriteQueueSize)
//		p.peer.Start(nil)
//	}
//
//	func (p *ServerPeer) Process(ctx context.Context, r msgpeer.Request, w msgpeer.ResponseWriter) {
//		log.Printf("server receive request: %s", r)
//		w(ctx, []byte("server processed"))
//	}
//
//	func (p *ServerPeer) OnNotify(ctx context.Context, n msgpeer.Notify) {
//		log.Printf("server receive notify: %s", n)
//	}
//
//	func (p *ServerPeer) Ask() {
//		resp, err := p.peer.Do(context.Background(), []byte("server-ask"))
//		log.Printf("server-ask response: %s, %v", resp, err)
//	}
//
//	func (p *ServerPeer) WaitStop() {
//		select {
//		case <-p.peer.StopD():
//		}
//	}
//
//	func server(listenD syncx.DoneChan) {
//		l, err := net.Listen("tcp", TheAddr)
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer l.Close()
//
//		listenD.SetDone()
//
//		for {
//			conn, err := l.Accept()
//			if err != nil {
//				log.Fatal(err)
//			}
//
//			go func() {
//				p := &ServerPeer{}
//				p.Start(conn)
//
//				p.peer.Notify(context.Background(), []byte("server started"))
//				time.Sleep(time.Millisecond)
//
//				p.Ask()
//				time.Sleep(time.Millisecond)
//
//				p.WaitStop()
//
//				p.Ask()
//
//				log.Printf("server peer stopped, error: %v", p.peer.Error())
//			}()
//		}
//	}
//
// Client
//
//	type ClientPeer struct {
//		peer *msgpeer.Peer
//	}
//
//	func (p *ClientPeer) Start(conn net.Conn) {
//		mrw := msgpump.NetconnMRW(conn)
//
//		p.peer = msgpeer.NewPeer(mrw, p, WriteQueueSize)
//		p.peer.Start(nil)
//	}
//
//	func (p *ClientPeer) Process(ctx context.Context, r msgpeer.Request, w msgpeer.ResponseWriter) {
//		log.Printf("client receive request: %s", r)
//		w(ctx, []byte("client processed"))
//	}
//
//	func (p *ClientPeer) OnNotify(ctx context.Context, n msgpeer.Notify) {
//		log.Printf("client receive notify: %s", n)
//	}
//
//	func (p *ClientPeer) Ask() {
//		resp, err := p.peer.Do(context.Background(), []byte("client-ask"))
//		log.Printf("client-ask response: %s, %v", resp, err)
//	}
//
//	func (p *ClientPeer) WaitStop() {
//		select {
//		case <-p.peer.StopD():
//		}
//	}
//
//	func client() {
//		conn, err := net.Dial("tcp", TheAddr)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		p := &ClientPeer{}
//		p.Start(conn)
//
//		p.peer.Notify(context.Background(), []byte("client started"))
//		time.Sleep(time.Millisecond)
//
//		p.Ask()
//		time.Sleep(time.Millisecond)
//
//		time.Sleep(10 * time.Millisecond)
//		p.peer.Stop()
//
//		p.WaitStop()
//
//		p.Ask()
//
//		log.Printf("client peer stopped, error: %v", p.peer.Error())
//	}
package msgpeer
