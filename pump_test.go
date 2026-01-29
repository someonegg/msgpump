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

func TestPump_Statistics(t *testing.T) {
	rw := &mockMRW{rmax: 3}

	h := func(ctx context.Context, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 10)
	pump.Start(nil)

	// Wait for read to complete
	<-pump.StopD()

	stat := pump.Statistics()

	if stat.ReadedCount != 3 {
		t.Errorf("ReadedCount = %d, want 3", stat.ReadedCount)
	}
	// m1, m2, m3 are 2 bytes each
	if stat.ReadedBytes != 6 {
		t.Errorf("ReadedBytes = %d, want 6", stat.ReadedBytes)
	}
}

func TestPump_Statistics_Write(t *testing.T) {
	rw := &mockMRW{rsus: make(chan bool), wmax: 3}

	h := func(ctx context.Context, m Message) {}

	pump := NewPump(rw, HandlerFunc(h), 10)
	pump.Start(nil)

	pump.Output(context.Background(), []byte("aa"))
	pump.Output(context.Background(), []byte("bbb"))
	pump.Output(context.Background(), []byte("c"))
	// The 4th message triggers wmax error
	pump.Output(context.Background(), []byte("d"))

	<-pump.StopD()

	stat := pump.Statistics()

	if stat.WrittenCount != 3 {
		t.Errorf("WrittenCount = %d, want 3", stat.WrittenCount)
	}
	if stat.WrittenBytes != 6 {
		t.Errorf("WrittenBytes = %d, want 6", stat.WrittenBytes)
	}
	if stat.OutputCount != 4 {
		t.Errorf("OutputCount = %d, want 4", stat.OutputCount)
	}
}

func TestPump_Stopped(t *testing.T) {
	rw := &mockMRW{rsus: make(chan bool)}

	pump := NewPump(rw, HandlerFunc(func(ctx context.Context, m Message) {}), 1)

	if pump.Stopped() {
		t.Error("Stopped() should be false before Start")
	}

	pump.Start(nil)

	if pump.Stopped() {
		t.Error("Stopped() should be false after Start")
	}

	pump.Stop()
	<-pump.StopD()

	if !pump.Stopped() {
		t.Error("Stopped() should be true after Stop")
	}
}

func TestPump_OutputAfterStop(t *testing.T) {
	rw := &mockMRW{rsus: make(chan bool)}

	pump := NewPump(rw, HandlerFunc(func(ctx context.Context, m Message) {}), 1)
	pump.Start(nil)
	pump.Stop()
	<-pump.StopD()

	// Fill the queue first to ensure subsequent calls can only select the stopD branch
	pump.TryOutput([]byte("fill"))

	err := pump.Output(context.Background(), []byte("test"))
	if err != ErrPumpStopped {
		t.Errorf("Output after stop: got %v, want ErrPumpStopped", err)
	}

	err = pump.OutputMP(context.Background(), MPMessage{[]byte("test")})
	if err != ErrPumpStopped {
		t.Errorf("OutputMP after stop: got %v, want ErrPumpStopped", err)
	}
}

func TestPump_ContextCancel(t *testing.T) {
	rw := &mockMRW{rsus: make(chan bool)}

	pump := NewPump(rw, HandlerFunc(func(ctx context.Context, m Message) {}), 1)

	ctx, cancel := context.WithCancel(context.Background())
	pump.Start(ctx)

	cancel()

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		t.Fatal("pump should stop when context is canceled")
	}

	if pump.Error() != nil {
		t.Errorf("Error() = %v, want nil", pump.Error())
	}
}

func TestPump_StopNotifier(t *testing.T) {
	rw := &mockMRW{rsus: make(chan bool)}

	pump := NewPump(rw, HandlerFunc(func(ctx context.Context, m Message) {}), 1)
	pump.Start(nil)
	pump.Stop()
	<-pump.StopD()

	// mockMRW implements StopNotifier, OnStop closes rsus
	select {
	case <-rw.rsus:
		// OnStop was called
	default:
		t.Error("StopNotifier.OnStop was not called")
	}
}

func TestPump_HandlerPanic(t *testing.T) {
	rw := &mockMRW{rmax: 2}

	var panicValue interface{}
	panicCalled := make(chan bool, 1)

	pump := NewPump(rw, HandlerFunc(func(ctx context.Context, m Message) {
		panic("test panic")
	}), 1)
	pump.SetPanicLogFunc(func(v interface{}) {
		panicValue = v
		select {
		case panicCalled <- true:
		default:
		}
	})
	pump.Start(nil)

	select {
	case <-pump.StopD():
	case <-time.After(1 * time.Second):
		t.Fatal("pump should stop after handler panic")
	}

	select {
	case <-panicCalled:
	default:
		t.Error("panicLogFunc was not called")
	}

	if panicValue != "test panic" {
		t.Errorf("panicValue = %v, want 'test panic'", panicValue)
	}
}
