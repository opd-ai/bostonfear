package observability

import (
	"testing"
	"time"
)

type captureHook struct {
	count int
	last  Event
}

func (h *captureHook) Observe(evt Event) {
	h.count++
	h.last = evt
}

func TestNoopHookObserve(t *testing.T) {
	h := NoopHook{}
	h.Observe(Event{Name: "test", Timestamp: time.Now()})
}

func TestCaptureHookObserve(t *testing.T) {
	h := &captureHook{}
	evt := Event{Name: "connection.connect", Timestamp: time.Now(), Labels: map[string]string{"playerId": "p1"}}
	h.Observe(evt)

	if h.count != 1 {
		t.Fatalf("count = %d, want 1", h.count)
	}
	if h.last.Name != evt.Name {
		t.Fatalf("last.Name = %q, want %q", h.last.Name, evt.Name)
	}
	if h.last.Labels["playerId"] != "p1" {
		t.Fatalf("last labels missing playerId")
	}
}
