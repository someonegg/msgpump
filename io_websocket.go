// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump

import (
	"errors"
	"io"
	"io/ioutil"
)

var (
	errWebsocketMessageType   = errors.New("websocket io: need text message")
	errWebsocketMessageFormat = errors.New("websocket io: wrong message format")
)

// WebsocketReader interface, see https://godoc.org/github.com/gorilla/websocket/#Conn.NextReader
type WebsocketReader interface {
	NextReader() (messageType int, r io.Reader, err error)
}

// WebsocketWriter interface, see https://godoc.org/github.com/gorilla/websocket/#Conn.NextWriter
type WebsocketWriter interface {
	NextWriter(messageType int) (io.WriteCloser, error)
}

// WebsocketConn interface, see https://godoc.org/github.com/gorilla/websocket/#Conn
type WebsocketConn interface {
	WebsocketReader
	WebsocketWriter
	io.Closer
}

// See https://godoc.org/github.com/gorilla/websocket#pkg-constants
const (
	TextMessage   = 1
	BinaryMessage = 2
	CloseMessage  = 8
	PingMessage   = 9
	PongMessage   = 10
)

// WebsocketMRW converts a WebsocketConn to a MessageReadWriter.
//
// The websocket message is text message, the format is:
//
//	Type=Message
func WebsocketMRW(c WebsocketConn) MessageReadWriter {
	return websocketMRW{c: c}
}

type websocketMRW struct {
	c WebsocketConn
}

func (rw websocketMRW) OnStop() {
	rw.c.Close()
}

func (rw websocketMRW) ReadMessage() (t string, m Message, Err error) {
	wst, wsr, err := rw.c.NextReader()
	if err != nil {
		Err = err
		return
	}

	if wst != TextMessage {
		Err = errWebsocketMessageType
		return
	}

	p, err := ioutil.ReadAll(wsr)
	if err != nil {
		Err = err
		return
	}

	for i := 0; i < len(p); i++ {
		if p[i] == '=' {
			t = string(p[0:i])
			m = p[i+1:]
			return
		}
	}

	Err = errWebsocketMessageFormat
	return
}

func (rw websocketMRW) WriteMessage(t string, m Message) error {
	wswc, err := rw.c.NextWriter(TextMessage)
	if err != nil {
		return err
	}

	wswc.Write([]byte(t))
	wswc.Write([]byte("="))
	wswc.Write(m)

	return wswc.Close()
}
