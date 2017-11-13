package msgpump

import (
	"bytes"
	"fmt"
	"io"
)

type mockMRW struct {
	rsus chan bool
	rcnt int
	rmax int

	wsus chan bool
	wcnt int
	wmax int

	// length:Type:Message
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

func (rw *mockMRW) ReadMessage() (t string, m Message, err error) {
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
	return fmt.Sprint("t", rw.rcnt), []byte(fmt.Sprint("m", rw.rcnt)), nil
}

func (rw *mockMRW) WriteMessage(t string, m Message) error {
	if rw.wsus != nil {
		select {
		case <-rw.wsus:
		}
	}

	if rw.wmax > 0 && rw.wcnt >= rw.wmax {
		return io.ErrClosedPipe
	}

	rw.wcnt++
	l := len(t) + 1 + len(m)
	rw.b.WriteString(fmt.Sprint(l))
	rw.b.WriteString(":")
	rw.b.WriteString(t)
	rw.b.WriteString(":")
	rw.b.Write(m)
	return nil
}
