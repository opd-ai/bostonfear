// Package main — integration tests for WebSocket connection handling.
package serverengine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestHandleWebSocket_NewConnection verifies that a brand-new WebSocket connection
// reaches runMessageLoop without panicking (regression test for the unbalanced
// RUnlock that panicked on every live connection — AUDIT CRITICAL).
func TestHandleWebSocket_NewConnection(t *testing.T) {
	srv, cleanup := newIntegrationTestServer(t)
	defer cleanup()

	conn, playerID, token := srv.connectPlayer(t)
	defer conn.Close()

	// Verify we received valid connection info.
	if playerID == "" {
		t.Error("connectionStatus missing 'playerId' field")
	}
	if token == "" {
		t.Error("connectionStatus missing 'token' field")
	}
}

// TestHandleWebSocket_TokenReconnect verifies that a client supplying a valid
// reconnect token reclaims its original player slot (not a new player ID).
func TestHandleWebSocket_TokenReconnect(t *testing.T) {
	srv, cleanup := newIntegrationTestServer(t)
	defer cleanup()

	// First connection — receive player ID and reconnect token.
	conn1, originalID, token := srv.connectPlayer(t)
	if token == "" {
		t.Fatal("no reconnect token in first connectionStatus")
	}
	conn1.Close()
	// Brief pause so the server processes the disconnect.
	time.Sleep(50 * time.Millisecond)

	// Second connection with the token — should reclaim the same player ID.
	conn2, reconnectedID := srv.connectPlayerWithToken(t, token)
	defer conn2.Close()

	if reconnectedID != originalID {
		t.Errorf("expected reconnected player ID %q, got %q", originalID, reconnectedID)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		srv.GameServer.performanceMutex.RLock()
		session := srv.GameServer.playerSessions[originalID]
		reconnections := 0
		if session != nil {
			reconnections = session.Reconnections
		}
		srv.GameServer.performanceMutex.RUnlock()

		srv.GameServer.qualityMutex.RLock()
		_, qualityTracked := srv.GameServer.connectionQualities[originalID]
		srv.GameServer.qualityMutex.RUnlock()

		if reconnections >= 1 && qualityTracked {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	srv.GameServer.performanceMutex.RLock()
	session := srv.GameServer.playerSessions[originalID]
	gotReconnections := 0
	if session != nil {
		gotReconnections = session.Reconnections
	}
	srv.GameServer.performanceMutex.RUnlock()
	if gotReconnections < 1 {
		t.Errorf("expected reconnect session metric to increment, got %d", gotReconnections)
	}

	srv.GameServer.qualityMutex.RLock()
	_, qualityTracked := srv.GameServer.connectionQualities[originalID]
	srv.GameServer.qualityMutex.RUnlock()
	if !qualityTracked {
		t.Error("expected connection quality tracking restored after reconnect")
	}
}

// --- TestRescaleActDeck_LateJoin (Step 3 acceptance tests) ---
//
// TestRescaleActDeck_LateJoin verifies the documented win condition:
// "4 clues per investigator collectively" — when a second player joins a game
// that is already in progress, the Act deck win threshold must double from 4
// (1P) to 8 (2P).  This is the end-to-end regression test for GAP-19.
func TestRescaleActDeck_LateJoin(t *testing.T) {
	t.Parallel()
	srv, cleanup := newIntegrationTestServer(t)
	defer cleanup()
	gs := srv.GameServer

	// Player 1 connects — this starts the game (MinPlayers == 1).
	c1, _, _ := srv.connectPlayer(t)
	defer c1.Close()

	// Verify 1P threshold on the final act card (ActDeck[2]).
	gs.mutex.RLock()
	if len(gs.gameState.ActDeck) < 3 {
		gs.mutex.RUnlock()
		t.Fatalf("ActDeck has fewer than 3 cards after 1P game start")
	}
	thresh1P := gs.gameState.ActDeck[2].ClueThreshold
	gs.mutex.RUnlock()
	if thresh1P != 4 {
		t.Errorf("1P final act threshold = %d, want 4", thresh1P)
	}

	// Player 2 connects — this is a late join (game already in progress).
	c2, _, _ := srv.connectPlayer(t)
	defer c2.Close()

	// Allow the server goroutine to complete registerPlayer before we read state.
	time.Sleep(50 * time.Millisecond)

	gs.mutex.RLock()
	thresh2P := gs.gameState.ActDeck[2].ClueThreshold
	playerCount := len(gs.gameState.Players)
	gs.mutex.RUnlock()

	if playerCount != 2 {
		t.Fatalf("expected 2 players after late join, got %d", playerCount)
	}
	if thresh2P != 8 {
		t.Errorf("2P late-join final act threshold = %d, want 8 (4 clues × 2 players)", thresh2P)
	}
}

// TestRescaleActDeck_LateJoin_ThreePlayers verifies the threshold scales to 12
// when a third player joins after the game has started with two players.
func TestRescaleActDeck_LateJoin_ThreePlayers(t *testing.T) {
	t.Parallel()
	srv, cleanup := newIntegrationTestServer(t)
	defer cleanup()
	gs := srv.GameServer

	// Connect players 1 and 2.
	conns := make([]*websocket.Conn, 2)
	for i := 0; i < 2; i++ {
		c, _, _ := srv.connectPlayer(t)
		conns[i] = c
		defer c.Close()
	}
	time.Sleep(50 * time.Millisecond)

	// Player 3 late-joins.
	c3, _, _ := srv.connectPlayer(t)
	defer c3.Close()
	time.Sleep(50 * time.Millisecond)

	gs.mutex.RLock()
	thresh3P := gs.gameState.ActDeck[2].ClueThreshold
	playerCount := len(gs.gameState.Players)
	gs.mutex.RUnlock()

	if playerCount != 3 {
		t.Fatalf("expected 3 players, got %d", playerCount)
	}
	if thresh3P != 12 {
		t.Errorf("3P late-join final act threshold = %d, want 12 (4 clues × 3 players)", thresh3P)
	}
}

// TestHandleWebSocket_PregameActionsBeforeFirstTurn verifies that a freshly
// connected player can still run pregame setup actions before taking the first
// normal turn action in a live session.
func TestHandleWebSocket_PregameActionsBeforeFirstTurn(t *testing.T) {
	t.Parallel()

	srv, cleanup := newIntegrationTestServer(t)
	defer cleanup()

	conn, playerID, _ := srv.connectPlayer(t)
	defer conn.Close()

	selectAction := map[string]interface{}{
		"type":     "playerAction",
		"playerId": playerID,
		"action":   "selectInvestigator",
		"target":   "researcher",
	}
	if err := conn.WriteJSON(selectAction); err != nil {
		t.Fatalf("failed to send selectInvestigator action: %v", err)
	}

	difficultyAction := map[string]interface{}{
		"type":     "playerAction",
		"playerId": playerID,
		"action":   "setDifficulty",
		"target":   "hard",
	}
	if err := conn.WriteJSON(difficultyAction); err != nil {
		t.Fatalf("failed to send setDifficulty action: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		now := time.Now()
		if now.After(deadline) {
			break
		}
		if err := conn.SetReadDeadline(now.Add(200 * time.Millisecond)); err != nil {
			t.Fatalf("failed to set read deadline: %v", err)
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msgType, _ := msg["type"].(string); msgType != "gameState" {
			continue
		}

		srv.GameServer.mutex.RLock()
		p := srv.GameServer.gameState.Players[playerID]
		difficulty := srv.GameServer.gameState.Difficulty
		srv.GameServer.mutex.RUnlock()

		if p != nil && p.InvestigatorType == InvestigatorResearcher && difficulty == "hard" {
			return
		}
	}

	srv.GameServer.mutex.RLock()
	p := srv.GameServer.gameState.Players[playerID]
	difficulty := srv.GameServer.gameState.Difficulty
	srv.GameServer.mutex.RUnlock()

	if p == nil {
		t.Fatal("player not present after pregame actions")
	}
	if p.InvestigatorType != InvestigatorResearcher {
		t.Fatalf("investigator type = %q, want %q", p.InvestigatorType, InvestigatorResearcher)
	}
	if difficulty != "hard" {
		t.Fatalf("difficulty = %q, want hard", difficulty)
	}
}
