// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package msgpump

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var (
	errNetconnMessageLength = errors.New("netconn io: wrong message length")
	errNetconnMessageFormat = errors.New("netconn io: wrong message format")
)

type netbufconn struct {
	conn net.Conn
	*bufio.ReadWriter
}

func newNetbufConn(conn net.Conn) netbufconn {
	return netbufconn{
		conn:       conn,
		ReadWriter: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (c *netbufconn) Close() error {
	return c.conn.Close()
}

// NetconnMRW converts a Net.Conn to a MessageReadWriter.
//
// In the transport layer, message's layout is:
//   Length(4-bytes int, big-endian)Type=Message
type NetconnMRW struct {
	c netbufconn
}

func (rw NetconnMRW) OnStop() {
	rw.c.Close()
}

func (rw NetconnMRW) ReadMessage() (t string, m Message, Err error) {
	var _l int32
	err := binary.Read(rw.c, binary.BigEndian, &_l)
	if err != nil {
		Err = err
		return
	}
	l := int(_l)

	if l <= 0 || l > NetconnMessageMaxLength {
		Err = errNetconnMessageLength
		return
	}

	p := make([]byte, l)
	_, err = io.ReadFull(rw.c, p)
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

	Err = errNetconnMessageFormat
	return
}

func (rw NetconnMRW) WriteMessage(t string, m Message) error {
	l := len(t) + 1 + len(m)

	err := binary.Write(rw.c, binary.BigEndian, int32(l))
	if err != nil {
		return err
	}

	rw.c.WriteString(t)
	rw.c.WriteString("=")

	_, err = rw.c.Write(m)
	if err != nil {
		return err
	}

	return rw.c.Flush()
}

// NetconnMessageMaxLength is the maximum message length.
const NetconnMessageMaxLength = 32 * 1024 * 1024

// NetconnPump create a pump from a net.Conn.
func NetconnPump(conn net.Conn, h Handler, writeQueueSize int) *Pump {
	rw := NetconnMRW{c: newNetbufConn(conn)}
	return NewPump(rw, h, writeQueueSize, rw)
}
