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
	h := func(ctx context.Context, m Message) {
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
		test.Fatal("read count", count)
	}

	if err := pump.Error(); err != io.EOF {
		test.Fatal("read error", err)
	}
}

func TestPumpWrite(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wmax: 2}

	h := func(ctx context.Context, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	pump.Output(context.Background(), []byte("m1"))
	pump.Output(context.Background(), []byte("m2"))
	pump.Output(context.Background(), []byte("m3"))

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("write stop")
	}

	if rw.wcnt != rw.wmax {
		test.Fatal("write count")
	}

	if rw.b.Len() != 8 {
		test.Fatal("write format", string(rw.b.Bytes()))
	}

	if err := pump.Error(); err != io.ErrClosedPipe {
		test.Fatal("write error", err)
	}
}

func TestPumpWriteMP(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wmax: 2}

	h := func(ctx context.Context, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	pump.Output(context.Background(), []byte("m1"))
	pump.OutputMP(context.Background(), MPMessage{[]byte("m2"), []byte("m3")})
	pump.Output(context.Background(), []byte("m4"))

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("write stop")
	}

	if rw.wcnt != rw.wmax {
		test.Fatal("write count")
	}

	if rw.b.Len() != 10 {
		test.Fatal("write format", string(rw.b.Bytes()))
	}

	if err := pump.Error(); err != io.ErrClosedPipe {
		test.Fatal("write error", err)
	}
}

func TestPumpTryWriteAndStop(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wsus: make(chan bool)}

	h := func(ctx context.Context, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	ok := pump.TryOutput([]byte("m1"))
	if !ok {
		test.Fatal("try write")
	}
	pump.Output(context.Background(), []byte("m2"))
	ok = pump.TryOutputMP(MPMessage{[]byte("m3")})
	if ok {
		test.Fatal("try write")
	}

	pump.Stop()

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("pump stop")
	}

	if err := pump.Error(); err != nil {
		test.Fatal("pump error", err)
	}
}

func TestPumpWriteAndStop(test *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wsus: make(chan bool)}

	h := func(ctx context.Context, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 1)
	pump.Start(nil)

	err := pump.Output(context.Background(), []byte("m1"))
	if err != nil {
		test.Fatal("output", err)
	}
	pump.Output(context.Background(), []byte("m2"))
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	err = pump.Output(ctx, []byte("m3"))
	if err == nil {
		test.Fatal("try write")
	}

	pump.Stop()

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		test.Fatal("pump stop")
	}

	if err := pump.Error(); err != nil {
		test.Fatal("pump error", err)
	}
}
