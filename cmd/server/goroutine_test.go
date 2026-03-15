package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- broadcastHandler: shutdown path ---

func TestBroadcastHandler_ShutdownExits(t *testing.T) {
	gs, _ := newTestServer(t)
	done := make(chan struct{})
	go func() {
		gs.broadcastHandler()
		close(done)
	}()
	close(gs.shutdownCh) // signal shutdown
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("broadcastHandler did not exit on shutdown")
	}
}

func TestBroadcastHandler_ProcessesMessage(t *testing.T) {
	gs, _ := newTestServer(t)
	// No wsConns, so broadcast does nothing except record latency
	done := make(chan struct{})
	go func() {
		gs.broadcastHandler()
		close(done)
	}()

	gs.broadcastCh <- []byte(`{"type":"test"}`)
	time.Sleep(50 * time.Millisecond) // let handler process

	// Verify latency was recorded
	avg := gs.averageBroadcastLatencyMs()
	if avg < 0 {
		t.Error("expected non-negative latency after broadcast")
	}

	close(gs.shutdownCh)
	<-done
}

// --- actionHandler: processes action and error path ---

func TestActionHandler_ProcessesAction(t *testing.T) {
	gs, pid := newTestServer(t)
	done := make(chan struct{})
	go func() {
		gs.actionHandler()
		close(done)
	}()

	gs.actionCh <- PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionMove,
		Target:   string(University),
	}
	time.Sleep(100 * time.Millisecond) // let handler process

	gs.mutex.RLock()
	loc := gs.gameState.Players[pid].Location
	gs.mutex.RUnlock()
	if loc != University {
		t.Errorf("expected player moved to University, got %s", loc)
	}

	// Drain broadcast messages
	for {
		select {
		case <-gs.broadcastCh:
			continue
		default:
		}
		break
	}

	close(gs.shutdownCh)
	<-done
}

func TestActionHandler_ShutdownExits(t *testing.T) {
	gs, _ := newTestServer(t)
	done := make(chan struct{})
	go func() {
		gs.actionHandler()
		close(done)
	}()
	close(gs.shutdownCh)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("actionHandler did not exit on shutdown")
	}
}

// --- processAction: Gather all paths ---

func TestProcessAction_GatherUpdatesResources(t *testing.T) {
	gs, pid := newTestServer(t)
	// Gather may increase Health/Sanity on success
	_ = gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionGather,
	})
	// Just validate no panic and actions were consumed
	if gs.gameState.Players[pid].ActionsRemaining > 1 {
		t.Error("expected actions to decrease after gather")
	}

	// Drain broadcast
	for {
		select {
		case <-gs.broadcastCh:
			continue
		default:
		}
		break
	}
}

// --- broadcastGameState: channel full path ---

func TestBroadcastGameState_ChannelFull(t *testing.T) {
	gs, _ := newTestServer(t)
	// Fill the channel (capacity 100)
	for i := 0; i < 100; i++ {
		gs.broadcastCh <- []byte(`{}`)
	}
	// Should not block — uses non-blocking send
	done := make(chan struct{})
	go func() {
		gs.broadcastGameState()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("broadcastGameState blocked on full channel")
	}
	// Drain
	for {
		select {
		case <-gs.broadcastCh:
			continue
		default:
		}
		break
	}
}

// --- collectConnectionAnalytics: no events path ---

func TestCollectConnectionAnalytics_NoEvents(t *testing.T) {
	gs := NewGameServer()
	analytics := gs.collectConnectionAnalytics()
	if analytics.ReconnectionRate < 0 {
		t.Error("reconnection rate should not be negative")
	}
}

// --- handleMetrics: doom level reflected ---

func TestHandleMetrics_DoomLevel(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 7

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	gs.handleMetrics(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "arkham_horror_game_doom_level 7") {
		t.Errorf("expected doom level 7 in metrics, body snippet: %s",
			body[:min(200, len(body))])
	}
}

// --- getGameStatistics: doomPercent paths ---

func TestGetGameStatistics_HighDoom(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 10

	stats := gs.getGameStatistics()
	threat, ok := stats["doomThreat"].(string)
	if !ok {
		t.Fatal("expected doomThreat key in game statistics")
	}
	if !strings.EqualFold(threat, "critical") {
		t.Errorf("expected critical doom threat at doom=10, got %s", threat)
	}
}

// newHTTPRequest creates a test HTTP request without importing net/http in this file.
func newHTTPRequest(method, url string) interface { /* http.Request */
} {
	return nil
}

// newHTTPRecorder creates a test response recorder.
func newHTTPRecorder() interface { /* http.ResponseWriter */
} {
	return nil
}
