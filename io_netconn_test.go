package msgpump

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

type mockNetConn struct {
	bytes.Buffer
}

func (c *mockNetConn) Close() error {
	return nil
}

func (c *mockNetConn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (c *mockNetConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (c *mockNetConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *mockNetConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *mockNetConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestNetconnRead(test *testing.T) {
	c := &mockNetConn{}
	rw := NetconnMRW(c)

	binary.Write(&c.Buffer, binary.BigEndian, int32(2))
	c.Buffer.WriteString("m1")

	m, err := rw.ReadMessage()
	if string(m) != "m1" || err != nil {
		test.Fatal("netconn io: read normal", err)
	}

	binary.Write(&c.Buffer, binary.BigEndian, int32(0))
	m, err = rw.ReadMessage()
	if err != errNetconnMessageLength {
		test.Fatal("netconn io: read wrong length", err)
	}

	binary.Write(&c.Buffer, binary.BigEndian, int32(5))
	m, err = rw.ReadMessage()
	if err != io.EOF {
		test.Fatal("netconn io: read wrong length", err)
	}
}

func TestNetconnWrite(test *testing.T) {
	c := &mockNetConn{}
	rw := NetconnMRW(c)

	err := rw.WriteMessage([]byte("m1"))
	if err != nil {
		test.Fatal(err)
	}

	m, err := rw.ReadMessage()
	if err != nil {
		test.Fatal(err)
	}

	if string(m) != "m1" {
		test.Fatal("netconn io: write wrong format")
	}
}

func TestNetconnWriteMP(test *testing.T) {
	c := &mockNetConn{}
	rw := NetconnMRW(c)

	err := rw.WriteMessageMP(MPMessage{[]byte("m1"), []byte("m2"), []byte("m3")})
	if err != nil {
		test.Fatal(err)
	}

	m, err := rw.ReadMessage()
	if err != nil {
		test.Fatal(err)
	}

	if string(m) != "m1m2m3" {
		test.Fatal("netconn io: write wrong format")
	}
}
