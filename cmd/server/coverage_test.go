package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- broadcastGameState: valid & corrupted state ---

func TestBroadcastGameState_QueuesMessage(t *testing.T) {
	gs, _ := newTestServer(t)

	gs.broadcastGameState()

	select {
	case msg := <-gs.broadcastCh:
		if !strings.Contains(string(msg), "gameState") {
			t.Errorf("expected gameState type in broadcast, got: %s", msg)
		}
	default:
		t.Error("expected message queued in broadcastCh after broadcastGameState")
	}
}

func TestBroadcastGameState_CorruptedStateRecovered(t *testing.T) {
	gs, _ := newTestServer(t)
	// Inject corruption that triggers recovery path
	gs.gameState.Doom = -5

	gs.broadcastGameState()

	// After broadcast, doom should be clamped by recovery
	gs.mutex.RLock()
	doom := gs.gameState.Doom
	gs.mutex.RUnlock()
	if doom < 0 {
		t.Errorf("doom should have been recovered from -5, got %d", doom)
	}

	// Drain any queued message
	select {
	case <-gs.broadcastCh:
	default:
	}
}

// --- calculateErrorRate ---

func TestCalculateErrorRate_ZeroMessages(t *testing.T) {
	gs, _ := newTestServer(t)
	if rate := gs.calculateErrorRate(); rate != 0 {
		t.Errorf("expected 0 error rate with no messages, got %.2f", rate)
	}
}

func TestCalculateErrorRate_NonZero(t *testing.T) {
	gs, _ := newTestServer(t)
	atomic.StoreInt64(&gs.errorCount, 5)
	atomic.StoreInt64(&gs.totalMessagesRecv, 100)

	rate := gs.calculateErrorRate()
	if rate < 4.9 || rate > 5.1 {
		t.Errorf("expected ~5%% error rate, got %.2f%%", rate)
	}
}

// --- assessConnectionQuality: packet loss degradation ---

func TestAssessConnectionQuality_PacketLossDegrades(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.initializeConnectionQuality(pid)

	gs.qualityMutex.Lock()
	gs.connectionQualities[pid].LatencyMs = 25    // excellent latency
	gs.connectionQualities[pid].PacketLoss = 0.10 // 10% packet loss
	gs.assessConnectionQuality(pid)
	got := gs.connectionQualities[pid].Quality
	gs.qualityMutex.Unlock()

	// excellent → good due to packet loss
	if got != "good" {
		t.Errorf("expected quality degraded to 'good' with packet loss, got %s", got)
	}
}

// --- handlePongMessage: missing player ID ---

func TestHandlePongMessage_UnknownPlayer(t *testing.T) {
	gs, _ := newTestServer(t)
	// Should return cleanly without panicking — unknown player has no quality entry
	done := make(chan struct{})
	go func() {
		gs.handlePongMessage(PingMessage{
			Type:     "pong",
			PlayerID: "nonexistent",
		}, time.Now())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("handlePongMessage hung on unknown player")
	}
}

// --- validateMovement: unknown source ---

func TestValidateMovement_UnknownSource(t *testing.T) {
	gs, _ := newTestServer(t)
	result := gs.validateMovement("Unknown", Downtown)
	if result {
		t.Error("expected false for movement from unknown location")
	}
}

// --- processAction: invalid action type ---

func TestProcessAction_InvalidActionType(t *testing.T) {
	gs, pid := newTestServer(t)
	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   "invalidAction",
	})
	if err == nil {
		t.Error("expected error for invalid action type")
	}
}

// --- processAction: Investigate doom integration ---

func TestProcessAction_InvestigateUpdatesState(t *testing.T) {
	gs, pid := newTestServer(t)
	initialDoom := gs.gameState.Doom

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionInvestigate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Doom must not decrease
	if gs.gameState.Doom < initialDoom {
		t.Error("doom should never decrease from investigate action")
	}
	// Player used one action
	if gs.gameState.Players[pid].ActionsRemaining != 1 {
		t.Errorf("expected 1 action remaining after investigate, got %d",
			gs.gameState.Players[pid].ActionsRemaining)
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
}

// --- getGameStatistics: with multiple players ---

func TestGetGameStatistics_MultiPlayer(t *testing.T) {
	gs, _ := newTestServer(t)
	addPlayer(gs, "p2", true)
	gs.gameState.Players["p2"].Resources.Clues = 3

	stats := gs.getGameStatistics()
	if total, ok := stats["totalClues"].(int); !ok || total < 3 {
		t.Errorf("expected totalClues >= 3, got %v", stats["totalClues"])
	}
}

// --- handleHealthCheck: unhealthy state ---

func TestHandleHealthCheck_UnhealthyOnHighCorruption(t *testing.T) {
	gs, _ := newTestServer(t)
	// Force many corruptions in the last 5 minutes
	for i := 0; i < 15; i++ {
		gs.validator.logCorruption(CorruptionEvent{
			ErrorType:   "TEST",
			Description: "forced test corruption",
			Timestamp:   time.Now(),
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	gs.handleHealthCheck(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for unhealthy state, got %d", w.Code)
	}
}

// --- collectGCMetrics: triggers GC path ---

func TestCollectGCMetrics_AfterGC(t *testing.T) {
	gs, _ := newTestServer(t)
	// Force a GC so NumGC > 0 and PauseNs paths are exercised
	for i := 0; i < 3; i++ {
		_ = make([]byte, 1024*1024)
	}
	m := gs.collectGCMetrics()
	// Just verify no panic and valid data
	if m.CPUFraction < 0 {
		t.Error("CPUFraction should not be negative")
	}
}

// --- getSystemAlerts: medium doom alert ---

func TestGetSystemAlerts_MediumDoom(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 8 // > 60% threshold but < 80%

	alerts := gs.getSystemAlerts()
	found := false
	for _, a := range alerts {
		if sev, ok := a["severity"].(string); ok && sev == "medium" {
			if msg, ok := a["message"].(string); ok && strings.Contains(msg, "doom") {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected medium severity doom alert at doom=8")
	}
}

// --- broadcastConnectionQuality: with quality data ---

func TestBroadcastConnectionQuality_NoPlayers(t *testing.T) {
	gs := NewGameServer()
	// No players, no qualities — should not panic
	gs.broadcastConnectionQuality()
	// Drain any messages
	select {
	case <-gs.broadcastCh:
	default:
	}
}

// --- trackConnection: disconnect doesn't increment total ---

func TestTrackConnection_DisconnectNoIncrement(t *testing.T) {
	gs, _ := newTestServer(t)
	before := gs.totalConnections
	gs.trackConnection("disconnect", "p1", 0)
	if gs.totalConnections != before {
		t.Error("totalConnections should not increment on disconnect")
	}
}
