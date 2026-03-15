// Package main — integration tests for WebSocket connection handling.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestHandleWebSocket_NewConnection verifies that a brand-new WebSocket connection
// reaches runMessageLoop without panicking (regression test for the unbalanced
// RUnlock that panicked on every live connection — AUDIT CRITICAL).
func TestHandleWebSocket_NewConnection(t *testing.T) {
	gs := NewGameServer()
	// Start background goroutines so broadcastGameState's non-blocking send
	// doesn't fill the channel and broadcastHandler drains it cleanly.
	go gs.broadcastHandler()
	go gs.actionHandler()
	defer close(gs.shutdownCh)

	srv := httptest.NewServer(http.HandlerFunc(gs.handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v (HTTP status: %v)", err, resp)
	}
	defer conn.Close()

	// The server sends a connectionStatus message immediately after registering
	// the player. Set a short read deadline so the test fails fast on regression.
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("did not receive connectionStatus within timeout: %v", err)
	}

	var status map[string]interface{}
	if jsonErr := json.Unmarshal(msg, &status); jsonErr != nil {
		t.Fatalf("connectionStatus is not valid JSON: %v", jsonErr)
	}
	if msgType, _ := status["type"].(string); msgType != "connectionStatus" {
		t.Errorf("expected message type 'connectionStatus', got %q", msgType)
	}
	if _, hasPlayer := status["playerId"]; !hasPlayer {
		t.Error("connectionStatus missing 'playerId' field")
	}
	if _, hasToken := status["token"]; !hasToken {
		t.Error("connectionStatus missing 'token' field")
	}
}

// TestHandleWebSocket_TokenReconnect verifies that a client supplying a valid
// reconnect token reclaims its original player slot (not a new player ID).
func TestHandleWebSocket_TokenReconnect(t *testing.T) {
	gs := NewGameServer()
	go gs.broadcastHandler()
	go gs.actionHandler()
	defer close(gs.shutdownCh)

	srv := httptest.NewServer(http.HandlerFunc(gs.handleWebSocket))
	defer srv.Close()

	baseURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// First connection — receive player ID and reconnect token.
	conn1, _, err := websocket.DefaultDialer.Dial(baseURL, nil)
	if err != nil {
		t.Fatalf("first dial failed: %v", err)
	}
	conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg1, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("first connectionStatus not received: %v", err)
	}
	var status1 map[string]interface{}
	json.Unmarshal(msg1, &status1)
	originalID, _ := status1["playerId"].(string)
	token, _ := status1["token"].(string)
	if token == "" {
		t.Fatal("no reconnect token in first connectionStatus")
	}
	conn1.Close()
	// Brief pause so the server processes the disconnect.
	time.Sleep(50 * time.Millisecond)

	// Second connection with the token — should reclaim the same player ID.
	reconnectURL := baseURL + "?token=" + token
	conn2, _, err := websocket.DefaultDialer.Dial(reconnectURL, nil)
	if err != nil {
		t.Fatalf("reconnect dial failed: %v", err)
	}
	defer conn2.Close()
	conn2.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg2, err := conn2.ReadMessage()
	if err != nil {
		t.Fatalf("reconnect connectionStatus not received: %v", err)
	}
	var status2 map[string]interface{}
	json.Unmarshal(msg2, &status2)
	reconnectedID, _ := status2["playerId"].(string)

	if reconnectedID != originalID {
		t.Errorf("expected reconnected player ID %q, got %q", originalID, reconnectedID)
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
	gs := NewGameServer()
	go gs.broadcastHandler()
	go gs.actionHandler()
	defer close(gs.shutdownCh)

	srv := httptest.NewServer(http.HandlerFunc(gs.handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// Player 1 connects — this starts the game (MinPlayers == 1).
	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("player 1 dial: %v", err)
	}
	defer c1.Close()
	c1.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c1.ReadMessage(); err != nil { // drain connectionStatus
		t.Fatalf("player 1 connectionStatus: %v", err)
	}
	// Drain the initial gameState broadcast.
	c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	c1.ReadMessage() //nolint:errcheck

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
	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("player 2 dial: %v", err)
	}
	defer c2.Close()
	c2.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c2.ReadMessage(); err != nil { // drain connectionStatus
		t.Fatalf("player 2 connectionStatus: %v", err)
	}

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
	gs := NewGameServer()
	go gs.broadcastHandler()
	go gs.actionHandler()
	defer close(gs.shutdownCh)

	srv := httptest.NewServer(http.HandlerFunc(gs.handleWebSocket))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// Connect players 1 and 2.
	for i := 0; i < 2; i++ {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("player %d dial: %v", i+1, err)
		}
		defer c.Close()
		c.SetReadDeadline(time.Now().Add(3 * time.Second))
		c.ReadMessage() //nolint:errcheck — drain connectionStatus
	}
	time.Sleep(50 * time.Millisecond)

	// Player 3 late-joins.
	c3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("player 3 dial: %v", err)
	}
	defer c3.Close()
	c3.SetReadDeadline(time.Now().Add(3 * time.Second))
	c3.ReadMessage() //nolint:errcheck — drain connectionStatus
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
