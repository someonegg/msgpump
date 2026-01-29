package msgpeer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/someonegg/msgpump/v2"
)

type mockPeerMRW struct {
	mu      sync.Mutex
	readC   chan msgpump.Message
	writeC  chan msgpump.Message
	closed  bool
	closedC chan struct{}
}

func newMockPeerMRW() *mockPeerMRW {
	return &mockPeerMRW{
		readC:   make(chan msgpump.Message, 10),
		writeC:  make(chan msgpump.Message, 10),
		closedC: make(chan struct{}),
	}
}

func (m *mockPeerMRW) ReadMessage() (msgpump.Message, error) {
	select {
	case msg := <-m.readC:
		return msg, nil
	case <-m.closedC:
		return nil, context.Canceled
	}
}

func (m *mockPeerMRW) WriteMessage(msg msgpump.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return context.Canceled
	}
	m.writeC <- msg
	return nil
}

func (m *mockPeerMRW) WriteMessageMP(msg msgpump.MPMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return context.Canceled
	}
	// Merge into a single message
	var combined []byte
	for _, part := range msg {
		combined = append(combined, part...)
	}
	m.writeC <- combined
	return nil
}

func (m *mockPeerMRW) OnStop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closedC)
	}
}

type mockPeerHandler struct {
	processC chan Request
	notifyC  chan Notify
}

func newMockPeerHandler() *mockPeerHandler {
	return &mockPeerHandler{
		processC: make(chan Request, 10),
		notifyC:  make(chan Notify, 10),
	}
}

func (h *mockPeerHandler) Process(ctx context.Context, r Request, w ResponseWriter) {
	h.processC <- r
	w(ctx, []byte("response"))
}

func (h *mockPeerHandler) OnNotify(ctx context.Context, n Notify) {
	h.notifyC <- n
}

func TestPeer_Notify(t *testing.T) {
	mrw := newMockPeerMRW()
	h := newMockPeerHandler()
	peer := NewPeer(mrw, h, 10)
	peer.Start(nil)

	err := peer.Notify(context.Background(), []byte("hello"))
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	select {
	case msg := <-mrw.writeC:
		// Verify message format: N\n + body
		expected := "N\nhello"
		if string(msg) != expected {
			t.Errorf("Notify message = %q, want %q", msg, expected)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Notify message not written")
	}

	peer.Stop()
	<-peer.StopD()
}

func TestPeer_Do(t *testing.T) {
	mrw := newMockPeerMRW()
	h := newMockPeerHandler()
	peer := NewPeer(mrw, h, 10)
	peer.Start(nil)

	// Start goroutine to simulate response
	go func() {
		select {
		case msg := <-mrw.writeC:
			// Parse request ID: R,<rid>\n<body>
			s := string(msg)
			if len(s) > 2 && s[0] == 'R' && s[1] == ',' {
				var rid string
				for i := 2; i < len(s); i++ {
					if s[i] == '\n' {
						rid = s[2:i]
						break
					}
				}
				// Send response: P,<rid>\n<response>
				resp := []byte("P," + rid + "\nresponse-data")
				mrw.readC <- resp
			}
		case <-time.After(100 * time.Millisecond):
		}
	}()

	resp, err := peer.Do(context.Background(), []byte("request-data"))
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}

	if string(resp) != "response-data" {
		t.Errorf("Do response = %q, want %q", resp, "response-data")
	}

	peer.Stop()
	<-peer.StopD()
}

func TestPeer_Do_Timeout(t *testing.T) {
	mrw := newMockPeerMRW()
	h := newMockPeerHandler()
	peer := NewPeer(mrw, h, 10)
	peer.Start(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := peer.Do(ctx, []byte("request"))
	if err != context.DeadlineExceeded {
		t.Errorf("Do with timeout: got %v, want DeadlineExceeded", err)
	}

	peer.Stop()
	<-peer.StopD()
}

func TestPeer_Process_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		message []byte
		wantReq bool
		wantNot bool
	}{
		{"empty", []byte{}, false, false},
		{"no newline", []byte("R,123"), false, false},
		{"notify", []byte("N\nhello"), false, true},
		{"request", []byte("R,1\nbody"), true, false},
		{"unknown type", []byte("X,1\nbody"), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mrw := newMockPeerMRW()
			h := newMockPeerHandler()
			peer := NewPeer(mrw, h, 10)

			// Call Process directly
			peer.Process(context.Background(), tt.message)

			gotReq := false
			gotNot := false

			select {
			case <-h.processC:
				gotReq = true
			default:
			}

			select {
			case <-h.notifyC:
				gotNot = true
			default:
			}

			if gotReq != tt.wantReq {
				t.Errorf("request received = %v, want %v", gotReq, tt.wantReq)
			}
			if gotNot != tt.wantNot {
				t.Errorf("notify received = %v, want %v", gotNot, tt.wantNot)
			}
		})
	}
}
