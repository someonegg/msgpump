package msgpump

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestPumpRead(test *testing.T) {
	rw := &mockMRW{rmax: 2}

	count := 0
	h := func(ctx context.Context, t string, m Message) {
		count++
	}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("read stop")
	}

	if count != rw.rmax {
		test.Fatal("read count")
	}

	if pump.Error() != io.EOF {
		test.Fatal("read error")
	}
}

func TestPumpWrite(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wmax: 2}

	h := func(ctx context.Context, t string, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	pump.Output("t1", []byte("m1"))
	pump.Output("t2", []byte("m2"))
	pump.Output("t3", []byte("m3"))

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("write stop")
	}

	if rw.wcnt != rw.wmax {
		test.Fatal("write count")
	}

	if rw.b.Len() != 14 {
		test.Fatal("write format")
	}

	if pump.Error() != io.ErrClosedPipe {
		test.Fatal("write error")
	}
}

func TestPumpTryWriteAndStop(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wsus: make(chan bool)}

	h := func(ctx context.Context, t string, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	ok := pump.TryOutput("t1", []byte("m1"))
	if !ok {
		test.Fatal("try write")
	}
	pump.Output("t2", []byte("m2"))
	ok = pump.TryOutput("t3", []byte("m3"))
	if ok {
		test.Fatal("try write")
	}

	pump.Stop()

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("pump stop")
	}

	if pump.Error() != nil {
		test.Fatal("pump error")
	}
}

func TestPumpPostAndStop(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wsus: make(chan bool)}

	h := func(ctx context.Context, t string, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	err := pump.Post(context.Background(), "t1", []byte("m1"))
	if err != nil {
		test.Fatal("post")
	}
	pump.Output("t2", []byte("m2"))
	ctx, _ := context.WithTimeout(context.Background(), 0)
	err = pump.Post(ctx, "t3", []byte("m3"))
	if err == nil {
		test.Fatal("try write")
	}

	pump.Stop()

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("pump stop")
	}

	if pump.Error() != nil {
		test.Fatal("pump error")
	}
}
