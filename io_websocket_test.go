package msgpump

import (
	"bytes"
	"io"
	"testing"
)

type bufferCloser struct {
	bytes.Buffer
}

func (bc *bufferCloser) Close() error {
	return nil
}

type mockWebsocketConn struct {
	rt int
	rb string

	wt int
	wb bufferCloser
}

func (c *mockWebsocketConn) NextReader() (int, io.Reader, error) {
	return c.rt, bytes.NewBufferString(c.rb), nil
}

func (c *mockWebsocketConn) NextWriter(messageType int) (io.WriteCloser, error) {
	c.wt = messageType
	return &c.wb, nil
}

func (c *mockWebsocketConn) Close() error {
	return nil
}

func TestWebsocketRead(test *testing.T) {
	c := &mockWebsocketConn{}
	rw := WebsocketMRW(c)

	c.rt = TextMessage
	c.rb = "t1=m1"
	t, m, err := rw.ReadMessage()
	if t != "t1" || string(m) != "m1" || err != nil {
		test.Fatal("websocket io: read normal")
	}

	c.rt = BinaryMessage
	c.rb = "t2=m2"
	t, m, err = rw.ReadMessage()
	if err != errWebsocketMessageType {
		test.Fatal("websocket io: read wrong type")
	}

	c.rt = TextMessage
	c.rb = "t3,m3"
	t, m, err = rw.ReadMessage()
	if err != errWebsocketMessageFormat {
		test.Fatal("websocket io: read wrong format")
	}
}

func TestWebsocketWrite(test *testing.T) {
	c := &mockWebsocketConn{}
	rw := WebsocketMRW(c)

	err := rw.WriteMessage("t1", []byte("m1"))
	if err != nil {
		test.Fatal(err)
	}

	if c.wt != TextMessage {
		test.Fatal("websocket io: write wrong type")
	}

	if c.wb.String() != "t1=m1" {
		test.Fatal("websocket io: write wrong format")
	}
}
