package msgpump

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
)

type mockMRW struct {
	rsus chan bool
	rcnt int
	rmax int

	wsus chan bool
	wcnt int
	wmax int

	// length:Message
	b bytes.Buffer
}

func (rw *mockMRW) OnStop() {
	if rw.rsus != nil {
		close(rw.rsus)
	}
	if rw.wsus != nil {
		close(rw.wsus)
	}
}

func (rw *mockMRW) ReadMessage() (m Message, err error) {
	if rw.rsus != nil {
		select {
		case <-rw.rsus:
		}
	}

	if rw.rmax > 0 && rw.rcnt >= rw.rmax {
		err = io.EOF
		return
	}

	rw.rcnt++
	return []byte(fmt.Sprint("m", rw.rcnt)), nil
}

func (rw *mockMRW) WriteMessage(m Message) error {
	if rw.wsus != nil {
		select {
		case <-rw.wsus:
		}
	}

	if rw.wmax > 0 && rw.wcnt >= rw.wmax {
		return io.ErrClosedPipe
	}

	rw.wcnt++
	l := m.Size()
	rw.b.WriteString(fmt.Sprint(l))
	rw.b.WriteString(":")
	rw.b.Write(m)
	return nil
}

func (rw *mockMRW) WriteMessageMP(m MPMessage) error {
	if rw.wsus != nil {
		select {
		case <-rw.wsus:
		}
	}

	if rw.wmax > 0 && rw.wcnt >= rw.wmax {
		return io.ErrClosedPipe
	}

	rw.wcnt++
	l := m.Size()
	rw.b.WriteString(fmt.Sprint(l))
	rw.b.WriteString(":")
	bufs := net.Buffers(m)
	bufs.WriteTo(&rw.b)
	return nil
}

func TestMessage_Size(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
		want int
	}{
		{"empty", Message{}, 0},
		{"normal", Message("hello"), 5},
		{"binary", Message{0x00, 0xff, 0x01}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.Size(); got != tt.want {
				t.Errorf("Message.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMPMessage_Size(t *testing.T) {
	tests := []struct {
		name string
		msg  MPMessage
		want int
	}{
		{"empty", MPMessage{}, 0},
		{"single", MPMessage{[]byte("hello")}, 5},
		{"multiple", MPMessage{[]byte("hello"), []byte("world")}, 10},
		{"with empty part", MPMessage{[]byte("a"), {}, []byte("b")}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.Size(); got != tt.want {
				t.Errorf("MPMessage.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}
