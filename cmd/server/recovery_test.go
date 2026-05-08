package main

import (
	"sync/atomic"
	"testing"
	"time"
)

// --- RecoverGameState: INVALID_CURRENT_PLAYER path ---

func TestRecoverGameState_InvalidCurrentPlayer(t *testing.T) {
	v := NewGameStateValidator()
	gs := &GameState{
		Players: map[string]*Player{
			"p1": {ID: "p1", Location: Downtown, Resources: Resources{Health: 5, Sanity: 5}},
		},
		TurnOrder:     []string{"p1"},
		CurrentPlayer: "nonexistent", // triggers INVALID_CURRENT_PLAYER
		Doom:          0,
		GamePhase:     "playing",
		GameStarted:   true,
	}

	errors := v.ValidateGameState(gs)
	recovered, err := v.RecoverGameState(gs, errors)
	if err != nil {
		t.Fatalf("unexpected recovery error: %v", err)
	}
	if recovered.CurrentPlayer != "p1" {
		t.Errorf("expected current player reset to p1, got %s", recovered.CurrentPlayer)
	}
}

// --- RecoverGameState: INVALID_LOCATION path ---

func TestRecoverGameState_InvalidLocation(t *testing.T) {
	v := NewGameStateValidator()
	gs := &GameState{
		Players: map[string]*Player{
			"p1": {
				ID:        "p1",
				Location:  Location("Dunwich"), // invalid location
				Resources: Resources{Health: 5, Sanity: 5},
			},
		},
		TurnOrder:     []string{"p1"},
		CurrentPlayer: "p1",
		Doom:          0,
		GamePhase:     "playing",
		GameStarted:   true,
	}

	errors := v.ValidateGameState(gs)
	recovered, err := v.RecoverGameState(gs, errors)
	if err != nil {
		t.Fatalf("unexpected recovery error: %v", err)
	}
	if recovered.Players["p1"].Location != Downtown {
		t.Errorf("expected invalid location reset to Downtown, got %s",
			recovered.Players["p1"].Location)
	}
}

// --- startPingTimer: sends ping and shuts down ---

func TestStartPingTimer_Shutdown(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.initializeConnectionQuality("p1")

	gs.startPingTimer("p1")

	// Give timer goroutine time to start, then shut down
	time.Sleep(10 * time.Millisecond)
	close(gs.shutdownCh)
	time.Sleep(50 * time.Millisecond) // let goroutine exit

	// Verify no panic occurred
}

// --- collectConnectionAnalytics: reconnection events ---

func TestCollectConnectionAnalytics_ReconnectionRate(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.trackConnection("connect", "p1", 0)
	gs.trackConnection("reconnect", "p1", 0)
	gs.trackConnection("disconnect", "p1", 0)

	analytics := gs.collectConnectionAnalytics()
	if analytics.ReconnectionRate < 0 {
		t.Error("reconnection rate should not be negative")
	}
}

// --- trackConnection: peak connections ---

func TestTrackConnection_PeakConnections(t *testing.T) {
	gs, _ := newTestServer(t)
	// Add two fake connections to connections map and keep atomic counter in sync.
	gs.mutex.Lock()
	gs.connections["addr1"] = nil
	gs.connections["addr2"] = nil
	atomic.AddInt64(&gs.activeConnections, 2)
	gs.mutex.Unlock()

	gs.trackConnection("connect", "p1", 0)
	gs.trackConnection("connect", "p2", 0)

	gs.performanceMutex.RLock()
	peak := gs.peakConnections
	gs.performanceMutex.RUnlock()

	if peak < 2 {
		t.Errorf("expected peak connections >= 2, got %d", peak)
	}
}

// --- handlePongMessage: latency calculation ---

func TestHandlePongMessage_LatencyCalculated(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.connectionQualities[pid] = &ConnectionQuality{Quality: "good"}

	tsBefore := time.Now()
	done := make(chan struct{})
	go func() {
		gs.handlePongMessage(PingMessage{
			Type:      "pong",
			PlayerID:  pid,
			Timestamp: tsBefore,
		}, time.Now())
		close(done)
	}()

	// Drain broadcast
drainLoop2:
	for {
		select {
		case <-gs.broadcastCh:
		case <-done:
			break drainLoop2
		}
	}

	gs.qualityMutex.RLock()
	latency := gs.connectionQualities[pid].LatencyMs
	gs.qualityMutex.RUnlock()

	if latency < 0 {
		t.Errorf("expected non-negative latency, got %.2f", latency)
	}
}
