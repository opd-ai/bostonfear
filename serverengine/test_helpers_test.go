// Package main — shared test helpers for WebSocket integration tests.
// Consolidates common server setup and connection patterns to reduce duplication.
package serverengine

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// testServerWithCleanup represents a running test server with cleanup function.
type testServerWithCleanup struct {
	GameServer *GameServer
	HTTPServer *httptest.Server
	BaseURL    string
}

// newIntegrationTestServer creates a GameServer with running broadcast/action handlers
// and an httptest.Server ready for WebSocket connections. Call cleanup when done.
func newIntegrationTestServer(t testing.TB) (*testServerWithCleanup, func()) {
	t.Helper()
	gs := NewGameServer()
	go gs.broadcastHandler()
	go gs.actionHandler()

	srv := httptest.NewServer(gs.WebSocketHandler())
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	cleanup := func() {
		srv.Close()
		close(gs.shutdownCh)
	}

	return &testServerWithCleanup{
		GameServer: gs,
		HTTPServer: srv,
		BaseURL:    wsURL,
	}, cleanup
}

// connectPlayer dials a WebSocket to the test server, reads the initial connectionStatus
// and gameState messages, and returns the connection plus the assigned playerID and token.
func (s *testServerWithCleanup) connectPlayer(t testing.TB) (*websocket.Conn, string, string) {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(s.BaseURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	// First message: connectionStatus carrying the player ID and token.
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("no connectionStatus: %v", err)
	}
	var status map[string]interface{}
	if err := json.Unmarshal(raw, &status); err != nil {
		t.Fatalf("bad connectionStatus JSON: %v", err)
	}
	playerID, _ := status["playerId"].(string)
	token, _ := status["token"].(string)
	if playerID == "" {
		t.Fatal("connectionStatus has no playerId")
	}

	// Drain the immediate gameState broadcast so the caller starts from a clean slate.
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	conn.ReadMessage() //nolint:errcheck // intentional drain

	conn.SetReadDeadline(time.Time{}) // clear deadline for normal operation
	return conn, playerID, token
}

// connectPlayerWithToken dials using a reconnect token and returns the connection and playerID.
func (s *testServerWithCleanup) connectPlayerWithToken(t testing.TB, token string) (*websocket.Conn, string) {
	t.Helper()
	reconnectURL := s.BaseURL + "?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(reconnectURL, nil)
	if err != nil {
		t.Fatalf("reconnect dial failed: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("no connectionStatus on reconnect: %v", err)
	}
	var status map[string]interface{}
	if err := json.Unmarshal(raw, &status); err != nil {
		t.Fatalf("bad reconnect connectionStatus JSON: %v", err)
	}
	playerID, _ := status["playerId"].(string)

	// Drain gameState broadcast.
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	conn.ReadMessage() //nolint:errcheck

	conn.SetReadDeadline(time.Time{})
	return conn, playerID
}
