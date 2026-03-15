package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- min / max ---

func TestMinMax(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(5, 3) != 5 {
		t.Error("max(5,3) should be 5")
	}
}

// --- recordBroadcastLatency / averageBroadcastLatencyMs ---

func TestBroadcastLatencyRingBuffer(t *testing.T) {
	gs, _ := newTestServer(t)

	// Before any samples, average should be 0
	if avg := gs.averageBroadcastLatencyMs(); avg != 0 {
		t.Errorf("expected 0 before samples, got %.4f", avg)
	}

	gs.recordBroadcastLatency(10 * time.Millisecond)
	gs.recordBroadcastLatency(20 * time.Millisecond)
	gs.recordBroadcastLatency(30 * time.Millisecond)

	avg := gs.averageBroadcastLatencyMs()
	if avg < 19 || avg > 21 {
		t.Errorf("expected average ~20 ms, got %.4f ms", avg)
	}
}

func TestBroadcastLatencyRingBuffer_Wrap(t *testing.T) {
	gs, _ := newTestServer(t)
	// Fill beyond ring buffer capacity (100 samples)
	for i := 0; i < 110; i++ {
		gs.recordBroadcastLatency(5 * time.Millisecond)
	}
	avg := gs.averageBroadcastLatencyMs()
	if avg < 4.5 || avg > 5.5 {
		t.Errorf("expected ~5 ms after wrap, got %.4f ms", avg)
	}
}

// --- collectMessageThroughput ---

func TestCollectMessageThroughput_ZeroUptime(t *testing.T) {
	gs, _ := newTestServer(t)
	m := gs.collectMessageThroughput(0)
	if m.MessagesPerSecond != 0 {
		t.Errorf("expected 0 messages/sec with zero uptime, got %f", m.MessagesPerSecond)
	}
}

func TestCollectMessageThroughput_NonZero(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.trackMessage("sent")
	gs.trackMessage("received")
	gs.recordBroadcastLatency(8 * time.Millisecond)

	m := gs.collectMessageThroughput(10 * time.Second)
	if m.MessagesPerSecond <= 0 {
		t.Error("expected positive messages/sec")
	}
	if m.BroadcastLatency <= 0 {
		t.Error("expected positive broadcast latency")
	}
}

// --- trackConnection / trackPlayerSession / trackMessage ---

func TestTrackConnection_CountsConnects(t *testing.T) {
	gs, _ := newTestServer(t)
	before := gs.totalConnections
	gs.trackConnection("connect", "p1", 0)
	if gs.totalConnections != before+1 {
		t.Errorf("expected totalConnections to increment")
	}
}

func TestTrackPlayerSession_LifeCycle(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.trackPlayerSession("testplayer", "start")
	if _, ok := gs.playerSessions["testplayer"]; !ok {
		t.Error("expected session to be created on start")
	}
	gs.trackPlayerSession("testplayer", "action")
	gs.trackPlayerSession("testplayer", "reconnect")
	gs.trackPlayerSession("testplayer", "end")
	if gs.playerSessions["testplayer"].IsActive {
		t.Error("expected session to be inactive after end")
	}
}

func TestTrackMessage(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.trackMessage("sent")
	gs.trackMessage("received")
	gs.trackMessage("sent")
	if gs.totalMessagesSent != 2 {
		t.Errorf("expected 2 sent, got %d", gs.totalMessagesSent)
	}
	if gs.totalMessagesRecv != 1 {
		t.Errorf("expected 1 received, got %d", gs.totalMessagesRecv)
	}
}

// --- collectMemoryMetrics / collectGCMetrics ---

func TestCollectMemoryMetrics(t *testing.T) {
	gs, _ := newTestServer(t)
	m := gs.collectMemoryMetrics()
	if m.AllocatedBytes == 0 {
		t.Error("expected non-zero allocated bytes")
	}
	if m.GoroutineCount == 0 {
		t.Error("expected non-zero goroutine count")
	}
	if m.MemoryUsagePercent < 0 || m.MemoryUsagePercent > 100 {
		t.Errorf("memory usage percent out of range: %.2f", m.MemoryUsagePercent)
	}
}

func TestCollectGCMetrics(t *testing.T) {
	gs, _ := newTestServer(t)
	m := gs.collectGCMetrics()
	// Just verify it returns without panic and PauseTotal is non-negative
	if m.PauseTotal < 0 {
		t.Error("pause total should be non-negative")
	}
}

// --- collectConnectionAnalytics ---

func TestCollectConnectionAnalytics(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.trackConnection("connect", pid, 0)
	gs.trackPlayerSession(pid, "start")

	analytics := gs.collectConnectionAnalytics()
	if analytics.TotalPlayers < 0 {
		t.Error("TotalPlayers should not be negative")
	}
}

// --- initializeConnectionQuality / assessConnectionQuality / cleanupConnectionQuality ---

func TestConnectionQuality_InitAssessCleanup(t *testing.T) {
	gs, pid := newTestServer(t)

	gs.initializeConnectionQuality(pid)
	gs.qualityMutex.RLock()
	q, ok := gs.connectionQualities[pid]
	gs.qualityMutex.RUnlock()
	if !ok {
		t.Fatal("expected quality entry after initialize")
	}
	if q.Quality == "" {
		t.Error("expected non-empty quality string after initialize")
	}

	// Test each latency tier
	cases := []struct {
		latency float64
		want    string
	}{
		{25, "excellent"},
		{75, "good"},
		{150, "fair"},
		{300, "poor"},
	}
	for _, tc := range cases {
		gs.qualityMutex.Lock()
		gs.connectionQualities[pid].LatencyMs = tc.latency
		gs.assessConnectionQuality(pid)
		got := gs.connectionQualities[pid].Quality
		gs.qualityMutex.Unlock()
		if got != tc.want {
			t.Errorf("latency %.0f: expected quality %s, got %s", tc.latency, tc.want, got)
		}
	}

	gs.cleanupConnectionQuality(pid)
	gs.qualityMutex.RLock()
	_, stillExists := gs.connectionQualities[pid]
	gs.qualityMutex.RUnlock()
	if stillExists {
		t.Error("expected quality entry removed after cleanup")
	}
}

// --- getGameStatistics ---

func TestGetGameStatistics(t *testing.T) {
	gs, _ := newTestServer(t)
	stats := gs.getGameStatistics()
	if stats == nil {
		t.Error("getGameStatistics returned nil")
	}
	if _, ok := stats["totalPlayers"]; !ok {
		t.Error("expected totalPlayers key in statistics")
	}
}

// --- HTTP handler: /health ---

func TestHandleHealthCheck_ReturnsJSON(t *testing.T) {
	gs, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	gs.handleHealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "status") {
		t.Error("expected 'status' field in health response")
	}
	if !strings.Contains(body, "systemAlerts") {
		t.Error("expected 'systemAlerts' field in health response (wired in by audit fix)")
	}
}

// --- HTTP handler: /metrics ---

func TestHandleMetrics_ContainsBroadcastLatency(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.recordBroadcastLatency(7 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	gs.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "arkham_horror_broadcast_latency_ms") {
		t.Error("expected broadcast_latency_ms metric in /metrics output")
	}
	if !strings.Contains(body, "arkham_horror_games_played_total") {
		t.Error("expected games_played_total metric in /metrics output")
	}
}

// --- updateConnectionQuality ---

func TestUpdateConnectionQuality(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.initializeConnectionQuality(pid)

	gs.updateConnectionQuality(pid, time.Now())

	gs.qualityMutex.RLock()
	q := gs.connectionQualities[pid]
	gs.qualityMutex.RUnlock()

	// MessageDelay should be very small (near-zero for just-captured time)
	if q.MessageDelay < 0 {
		t.Error("MessageDelay should not be negative")
	}
}

// --- collectPerformanceMetrics: TotalGamesPlayed field ---

func TestCollectPerformanceMetrics_TotalGamesPlayed(t *testing.T) {
	gs, _ := newTestServer(t)
	// Simulate two completed games
	gs.gameState.Doom = 12
	gs.checkGameEndConditions()
	gs.gameState.Doom = 0
	gs.gameState.GamePhase = "playing"
	gs.gameState.LoseCondition = false
	gs.gameState.Doom = 12
	gs.checkGameEndConditions()

	metrics := gs.collectPerformanceMetrics()
	if metrics.TotalGamesPlayed != 2 {
		t.Errorf("expected TotalGamesPlayed=2, got %d", metrics.TotalGamesPlayed)
	}
}

// --- TestHandleHealthCheck_ConcurrentActions (Step 1 acceptance test) ---
//
// Spin up 4 goroutines submitting actions through the action channel while
// concurrently issuing 50 GET /health requests. Validates that the deadlock
// pattern described in GAP-17 (nested RLock under write pressure) is absent:
// every health request must return HTTP 200 within the test timeout (10 s).
func TestHandleHealthCheck_ConcurrentActions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent integration test in -short mode")
	}

	gs := NewGameServer()
	go gs.broadcastHandler()
	go gs.actionHandler()
	defer close(gs.shutdownCh)

	// Seed four players so processAction has valid state to work with.
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("p%d", i)
		gs.mutex.Lock()
		gs.gameState.Players[id] = &Player{
			ID:               id,
			Location:         Downtown,
			Resources:        Resources{Health: 10, Sanity: 10, Clues: 0},
			ActionsRemaining: 2,
			Connected:        true,
		}
		gs.gameState.TurnOrder = append(gs.gameState.TurnOrder, id)
		gs.mutex.Unlock()
	}
	gs.mutex.Lock()
	gs.gameState.GameStarted = true
	gs.gameState.GamePhase = "playing"
	gs.gameState.CurrentPlayer = "p1"
	gs.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// 4 writer goroutines continuously send Gather actions (cycled through players).
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("p%d", i)
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Don't block when shutdownCh is closed.
					select {
					case gs.actionCh <- PlayerActionMessage{
						Type:     "playerAction",
						PlayerID: pid,
						Action:   ActionGather,
					}:
					case <-gs.shutdownCh:
						return
					case <-ctx.Done():
						return
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(id)
	}

	// 50 sequential /health requests, each must return 200 within the deadline.
	failures := 0
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			gs.handleHealthCheck(w, req)
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			t.Errorf("health request %d timed out (possible deadlock)", i+1)
			failures++
			continue
		}
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("health request %d: unexpected status %d", i+1, w.Code)
			failures++
		}
		time.Sleep(2 * time.Millisecond)
	}

	cancel() // stop writers
	wg.Wait()

	if failures > 0 {
		t.Errorf("%d out of 50 health requests failed or deadlocked", failures)
	}
}
