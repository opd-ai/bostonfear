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
