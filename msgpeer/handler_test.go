package msgpeer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestParallelHandler_Process(t *testing.T) {
	var processed int32

	h := &mockPeerHandler{
		processC: make(chan Request, 10),
	}

	ph := ParallelHandler(h, 100*time.Millisecond, nil)

	ph.Process(context.Background(), []byte("req1"), func(ctx context.Context, resp Response) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	select {
	case req := <-h.processC:
		if string(req) != "req1" {
			t.Errorf("received request = %q, want %q", req, "req1")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("request not processed")
	}
}

func TestParallelHandler_OnNotify(t *testing.T) {
	h := &mockPeerHandler{
		notifyC: make(chan Notify, 10),
	}

	ph := ParallelHandler(h, 100*time.Millisecond, nil)

	ph.OnNotify(context.Background(), []byte("notify1"))

	select {
	case n := <-h.notifyC:
		if string(n) != "notify1" {
			t.Errorf("received notify = %q, want %q", n, "notify1")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("notify not processed")
	}
}

func TestParallelHandler_PanicRecovery(t *testing.T) {
	var panicValue interface{}
	panicCalled := make(chan bool, 1)

	panicHandler := &panicMockHandler{}

	ph := ParallelHandler(panicHandler, 100*time.Millisecond, func(v interface{}) {
		panicValue = v
		select {
		case panicCalled <- true:
		default:
		}
	})

	ph.Process(context.Background(), []byte("req"), func(ctx context.Context, resp Response) error {
		return nil
	})

	select {
	case <-panicCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("panic log not called")
	}

	if panicValue != "handler panic" {
		t.Errorf("panic value = %v, want 'handler panic'", panicValue)
	}
}

type panicMockHandler struct{}

func (h *panicMockHandler) Process(ctx context.Context, r Request, w ResponseWriter) {
	panic("handler panic")
}

func (h *panicMockHandler) OnNotify(ctx context.Context, n Notify) {
	panic("handler panic")
}

func TestParallelHandler_ContextCanceled(t *testing.T) {
	h := &mockPeerHandler{
		processC: make(chan Request, 10),
	}

	ph := ParallelHandler(h, 100*time.Millisecond, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ph.Process(ctx, []byte("req"), func(ctx context.Context, resp Response) error {
		return nil
	})

	// Wait a short time to confirm the request is not processed
	time.Sleep(50 * time.Millisecond)

	select {
	case <-h.processC:
		t.Error("request should not be processed when context is canceled")
	default:
		// Expected: request is not processed
	}
}
