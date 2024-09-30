// Copyright 2023 someonegg. All rights reserved.
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

var errNetconnMessageLength = errors.New("netconn io: wrong message length")

type netbufconn struct {
	conn net.Conn
	*bufio.ReadWriter
}

func getNetbufConn(conn net.Conn) netbufconn {
	return netbufconn{
		conn:       conn,
		ReadWriter: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

func (c *netbufconn) Close() error {
	return c.conn.Close()
}

// NetconnMessageMaxLength is the maximum message length.
var NetconnMessageMaxLength = 16 * 1024 * 1024

// NetconnMRW converts a net.Conn to a MessageReadWriter.
//
// In the transport layer, message's layout is:
//
//	Length(4-bytes int, big-endian)Message
func NetconnMRW(c net.Conn) MessageReadWriter {
	return netconnMRW{c: getNetbufConn(c)}
}

type netconnMRW struct {
	c netbufconn
}

func (rw netconnMRW) OnStop() {
	rw.c.Close()
}

func (rw netconnMRW) ReadMessage() (m Message, Err error) {
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

	m = p
	return
}

func (rw netconnMRW) WriteMessage(m Message) error {
	l := m.Size()

	err := binary.Write(rw.c, binary.BigEndian, int32(l))
	if err != nil {
		return err
	}

	_, err = rw.c.Write(m)
	if err != nil {
		return err
	}

	return rw.c.Flush()
}

func (rw netconnMRW) WriteMessageMP(m MPMessage) error {
	l := m.Size()

	err := binary.Write(rw.c, binary.BigEndian, int32(l))
	if err != nil {
		return err
	}

	bufs := net.Buffers(m)
	_, err = bufs.WriteTo(rw.c)
	if err != nil {
		return err
	}

	return rw.c.Flush()
}
