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
	rw := NetconnMRW{c: newNetbufConn(c)}

	binary.Write(&c.Buffer, binary.BigEndian, int32(5))
	c.Buffer.WriteString("t1=m1")

	t, m, err := rw.ReadMessage()
	if t != "t1" || string(m) != "m1" || err != nil {
		test.Fatal("netconn io: read normal")
	}

	binary.Write(&c.Buffer, binary.BigEndian, int32(0))
	t, m, err = rw.ReadMessage()
	if err != errNetconnMessageLength {
		test.Fatal("netconn io: read wrong length")
	}

	binary.Write(&c.Buffer, binary.BigEndian, int32(5))
	t, m, err = rw.ReadMessage()
	if err != io.EOF {
		test.Fatal("netconn io: read wrong length")
	}

	binary.Write(&c.Buffer, binary.BigEndian, int32(5))
	c.Buffer.WriteString("t1,m1")
	t, m, err = rw.ReadMessage()
	if err != errNetconnMessageFormat {
		test.Fatal("netconn io: read wrong format", t, m, err)
	}
}

func TestNetconnWrite(test *testing.T) {
	c := &mockNetConn{}
	rw := NetconnMRW{c: newNetbufConn(c)}

	err := rw.WriteMessage("t1", []byte("m1"))
	if err != nil {
		test.Fatal(err)
	}

	t, m, err := rw.ReadMessage()
	if err != nil {
		test.Fatal(err)
	}

	if t != "t1" || string(m) != "m1" {
		test.Fatal("websocket io: write wrong format")
	}
}
