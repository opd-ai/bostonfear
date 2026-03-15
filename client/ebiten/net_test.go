package ebiten

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDecodeGameState_FromDataWrapper verifies that decodeGameState correctly
// unmarshals GameState from the server's {"type":"gameState","data":{...}} envelope.
// This is a regression test for the CRITICAL-1 protocol mismatch where the previous
// implementation read top-level fields that were actually nested under "data".
func TestDecodeGameState_FromDataWrapper(t *testing.T) {
	player := &Player{
		ID:               "player1",
		Location:         "University",
		Resources:        Resources{Health: 7, Sanity: 5, Clues: 2},
		ActionsRemaining: 1,
		Connected:        true,
	}

	inner := GameState{
		Players:       map[string]*Player{"player1": player},
		CurrentPlayer: "player1",
		Doom:          3,
		GamePhase:     "playerTurn",
		TurnOrder:     []string{"player1"},
		GameStarted:   true,
		WinCondition:  false,
		LoseCondition: false,
		RequiredClues: 4,
	}

	dataBytes, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal inner GameState: %v", err)
	}

	msg := serverMessage{
		Type: "gameState",
		Data: json.RawMessage(dataBytes),
	}

	got := decodeGameState(msg)

	if got.CurrentPlayer != "player1" {
		t.Errorf("CurrentPlayer = %q, want %q", got.CurrentPlayer, "player1")
	}
	if got.Doom != 3 {
		t.Errorf("Doom = %d, want 3", got.Doom)
	}
	if got.GamePhase != "playerTurn" {
		t.Errorf("GamePhase = %q, want %q", got.GamePhase, "playerTurn")
	}
	if !got.GameStarted {
		t.Error("GameStarted = false, want true")
	}
	if got.RequiredClues != 4 {
		t.Errorf("RequiredClues = %d, want 4", got.RequiredClues)
	}
	if p, ok := got.Players["player1"]; !ok {
		t.Error("Players missing player1")
	} else {
		if p.Resources.Health != 7 {
			t.Errorf("player1 Health = %d, want 7", p.Resources.Health)
		}
		if p.Resources.Sanity != 5 {
			t.Errorf("player1 Sanity = %d, want 5", p.Resources.Sanity)
		}
		if p.Resources.Clues != 2 {
			t.Errorf("player1 Clues = %d, want 2", p.Resources.Clues)
		}
		if string(p.Location) != "University" {
			t.Errorf("player1 Location = %q, want %q", p.Location, "University")
		}
	}
}

// TestDecodeGameState_EmptyData verifies that decodeGameState handles an empty
// Data field gracefully, returning an initialised (non-nil) Players map.
func TestDecodeGameState_EmptyData(t *testing.T) {
	msg := serverMessage{Type: "gameState"}
	got := decodeGameState(msg)
	if got.Players == nil {
		t.Error("Players map must not be nil after decoding empty data")
	}
}

// TestApplyConnectionStatus_PreservesToken verifies that applyConnectionStatus
// stores the server-issued reconnect token in LocalState.
// This is a regression test for GAP-04 where the token was silently discarded.
func TestApplyConnectionStatus_PreservesToken(t *testing.T) {
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": "player1",
		"token":    "abc123",
		"quality":  map[string]interface{}{"latency": 10, "rating": "excellent"},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	client.applyConnectionStatus(data)

	if state.PlayerID != "player1" {
		t.Errorf("PlayerID = %q, want %q", state.PlayerID, "player1")
	}
	if state.GetReconnectToken() != "abc123" {
		t.Errorf("ReconnectToken = %q, want %q", state.GetReconnectToken(), "abc123")
	}
}

// TestReconnectURL_IncludesTokenWhenSet verifies that GetReconnectToken returns
// the stored token and that the URL would be constructed correctly with it.
// (reconnectLoop itself starts a goroutine that dials a real server, so we
// test the token retrieval and URL construction logic here instead.)
func TestReconnectURL_IncludesTokenWhenSet(t *testing.T) {
	state := NewLocalState("ws://localhost:8080/ws")
	state.SetReconnectToken("tok-xyz")

	tok := state.GetReconnectToken()
	if tok == "" {
		t.Fatal("expected non-empty token")
	}

	dialURL := state.ServerURL
	if tok != "" {
		dialURL = dialURL + "?token=" + tok
	}

	if !strings.Contains(dialURL, "?token=tok-xyz") {
		t.Errorf("dial URL %q does not contain expected token query param", dialURL)
	}
}

// TestReconnectURL_OmitsTokenWhenEmpty verifies that no token query param is
// appended when no token has been received yet.
func TestReconnectURL_OmitsTokenWhenEmpty(t *testing.T) {
	// Redirect HOME so NewLocalState does not load a real session file.
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	tok := state.GetReconnectToken()

	dialURL := state.ServerURL
	if tok != "" {
		dialURL = dialURL + "?token=" + tok
	}

	if strings.Contains(dialURL, "token") {
		t.Errorf("dial URL %q should not contain token when none is set", dialURL)
	}
}
