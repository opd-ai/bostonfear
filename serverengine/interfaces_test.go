package serverengine

import (
	"sync/atomic"
	"testing"
)

// mockBroadcaster records every payload passed to Broadcast for test assertions.
type mockBroadcaster struct {
	calls int64
	last  []byte
}

// Broadcast records the payload and increments the call counter.
func (m *mockBroadcaster) Broadcast(payload []byte) error {
	atomic.AddInt64(&m.calls, 1)
	m.last = payload
	return nil
}

// TestBroadcasterInterface verifies that the Broadcaster interface is satisfied by
// mockBroadcaster, enabling unit tests to inject a no-op broadcaster without
// spinning up real goroutines or WebSocket connections.
func TestBroadcasterInterface(t *testing.T) {
	var b Broadcaster = &mockBroadcaster{}
	payload := []byte(`{"type":"gameState"}`)
	if err := b.Broadcast(payload); err != nil {
		t.Fatalf("Broadcast() error = %v; want nil", err)
	}
	mb := b.(*mockBroadcaster)
	if atomic.LoadInt64(&mb.calls) != 1 {
		t.Errorf("Broadcast call count = %d; want 1", atomic.LoadInt64(&mb.calls))
	}
}

// TestChannelBroadcaster_Broadcast verifies the channelBroadcaster implementation
// delivers payloads to the underlying channel.
func TestChannelBroadcaster_Broadcast(t *testing.T) {
	ch := make(chan []byte, 10)
	b := &channelBroadcaster{ch: ch}

	payload := []byte(`{"type":"diceResult"}`)
	if err := b.Broadcast(payload); err != nil {
		t.Fatalf("Broadcast() error = %v; want nil", err)
	}
	select {
	case got := <-ch:
		if string(got) != string(payload) {
			t.Errorf("channel payload = %q; want %q", got, payload)
		}
	default:
		t.Fatal("expected payload on channel but channel was empty")
	}
}

// TestChannelBroadcaster_FullChannel verifies that errBroadcastFull is returned
// when the channel is full, matching the documented drop behaviour.
func TestChannelBroadcaster_FullChannel(t *testing.T) {
	ch := make(chan []byte, 1)
	ch <- []byte("existing") // fill the buffer
	b := &channelBroadcaster{ch: ch}

	err := b.Broadcast([]byte("new"))
	if err != errBroadcastFull {
		t.Errorf("Broadcast() error = %v; want errBroadcastFull", err)
	}
}

// TestGameServer_BroadcasterInjection verifies that NewGameServer injects a
// channelBroadcaster backed by broadcastCh into the broadcaster field.
func TestGameServer_BroadcasterInjection(t *testing.T) {
	gs := NewGameServer()
	if gs.broadcaster == nil {
		t.Fatal("gs.broadcaster is nil; expected a channelBroadcaster")
	}
	payload := []byte(`{"type":"gameState"}`)
	if err := gs.broadcaster.Broadcast(payload); err != nil {
		t.Fatalf("gs.broadcaster.Broadcast() error = %v; want nil", err)
	}
	select {
	case msg := <-gs.broadcastCh:
		if string(msg) != string(payload) {
			t.Errorf("broadcastCh payload = %q; want %q", msg, payload)
		}
	default:
		t.Fatal("expected payload on gs.broadcastCh but channel was empty")
	}
}
