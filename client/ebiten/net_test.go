package ebiten

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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

// TestApplyDiceResult_UpdatesState verifies applyDiceResult unmarshals and stores a dice result.
func TestApplyDiceResult_UpdatesState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{
		"type":         "diceResult",
		"playerId":     "player1",
		"action":       "investigate",
		"results":      []string{"success", "tentacle"},
		"successes":    1,
		"tentacles":    1,
		"success":      false,
		"doomIncrease": 1,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	client.applyDiceResult(data)

	if state.LastDiceResult == nil {
		t.Fatal("LastDiceResult is nil after applyDiceResult")
	}
	if state.LastDiceResult.Action != "investigate" {
		t.Errorf("Action = %q, want investigate", state.LastDiceResult.Action)
	}
	if state.LastDiceResult.DoomIncrease != 1 {
		t.Errorf("DoomIncrease = %d, want 1", state.LastDiceResult.DoomIncrease)
	}
}

// TestApplyDiceResult_InvalidJSON logs and skips bad payloads without panicking.
func TestApplyDiceResult_InvalidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)
	client.applyDiceResult([]byte("{invalid"))
	if state.LastDiceResult != nil {
		t.Error("LastDiceResult should remain nil on bad JSON")
	}
}

// TestApplyGameUpdate_UpdatesState verifies applyGameUpdate stores a game-update event.
func TestApplyGameUpdate_UpdatesState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{
		"type":      "gameUpdate",
		"playerId":  "player2",
		"event":     "gather",
		"result":    "gained 1 clue",
		"doomDelta": 0,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	client.applyGameUpdate(data)

	if state.LastGameUpdate == nil {
		t.Fatal("LastGameUpdate is nil after applyGameUpdate")
	}
	if state.LastGameUpdate.Event != "gather" {
		t.Errorf("Event = %q, want gather", state.LastGameUpdate.Event)
	}
	if state.LastGameUpdate.Result != "gained 1 clue" {
		t.Errorf("Result = %q, want 'gained 1 clue'", state.LastGameUpdate.Result)
	}
}

// TestApplyGameUpdate_InvalidJSON logs and skips bad payloads.
func TestApplyGameUpdate_InvalidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)
	client.applyGameUpdate([]byte("{bad"))
	if state.LastGameUpdate != nil {
		t.Error("LastGameUpdate should remain nil on bad JSON")
	}
}

// TestRouteMessage_GameState verifies routeMessage dispatches gameState messages to UpdateGame.
func TestRouteMessage_GameState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	inner := GameState{
		CurrentPlayer: "player1",
		Doom:          4,
		GamePhase:     "playerTurn",
		Players:       map[string]*Player{},
		TurnOrder:     []string{"player1"},
	}
	innerData, _ := json.Marshal(inner)
	msg := map[string]interface{}{
		"type": "gameState",
		"data": json.RawMessage(innerData),
	}
	data, _ := json.Marshal(msg)

	client.routeMessage(data)

	snap, _, _ := state.Snapshot()
	if snap.CurrentPlayer != "player1" {
		t.Errorf("CurrentPlayer = %q, want player1", snap.CurrentPlayer)
	}
	if snap.Doom != 4 {
		t.Errorf("Doom = %d, want 4", snap.Doom)
	}
}

// TestRouteMessage_DiceResult verifies routeMessage dispatches diceResult messages.
func TestRouteMessage_DiceResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{
		"type":     "diceResult",
		"playerId": "p1",
		"action":   "ward",
		"success":  true,
	}
	data, _ := json.Marshal(payload)
	client.routeMessage(data)

	if state.LastDiceResult == nil {
		t.Fatal("LastDiceResult is nil after routing diceResult message")
	}
}

// TestRouteMessage_GameUpdate verifies routeMessage dispatches gameUpdate messages.
func TestRouteMessage_GameUpdate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{
		"type":      "gameUpdate",
		"playerId":  "p1",
		"event":     "move",
		"result":    "Rivertown",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(payload)
	client.routeMessage(data)

	if state.LastGameUpdate == nil {
		t.Fatal("LastGameUpdate is nil after routing gameUpdate message")
	}
}

// TestRouteMessage_ConnectionStatus verifies routeMessage dispatches connectionStatus messages.
func TestRouteMessage_ConnectionStatus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": "p2",
		"token":    "tok-route-test",
		"quality":  map[string]interface{}{"latency": 25, "rating": "good"},
	}
	data, _ := json.Marshal(payload)
	client.routeMessage(data)

	if state.PlayerID != "p2" {
		t.Errorf("PlayerID = %q, want p2", state.PlayerID)
	}
}

// TestRouteMessage_UnknownType verifies routeMessage silently ignores unknown message types.
func TestRouteMessage_UnknownType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	payload := map[string]interface{}{"type": "unknownFutureType", "data": "x"}
	data, _ := json.Marshal(payload)
	// Must not panic.
	client.routeMessage(data)
}

// TestRouteMessage_InvalidJSON verifies routeMessage handles malformed bytes gracefully.
func TestRouteMessage_InvalidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)
	client.routeMessage([]byte("{broken"))
}

// TestSendAction_DeliveredToChannel verifies SendAction enqueues the action for delivery.
func TestSendAction_DeliveredToChannel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	action := PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: "player1",
		Action:   "move",
		Target:   "University",
	}
	client.SendAction(action)

	select {
	case got := <-client.actionsCh:
		if got.Action != "move" {
			t.Errorf("Action = %q, want move", got.Action)
		}
		if got.Target != "University" {
			t.Errorf("Target = %q, want University", got.Target)
		}
	default:
		t.Error("action was not enqueued in the channel")
	}
}

// TestSendAction_DropWhenFull verifies SendAction does not block when the channel is full.
func TestSendAction_DropWhenFull(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	state := NewLocalState("ws://localhost:8080/ws")
	client := NewNetClient(state)

	// Fill the channel (capacity 16).
	for i := 0; i < 16; i++ {
		client.actionsCh <- PlayerActionMessage{Action: "gather"}
	}

	// This call must return immediately without blocking.
	done := make(chan struct{})
	go func() {
		client.SendAction(PlayerActionMessage{Action: "overflow"})
		close(done)
	}()

	select {
	case <-done:
		// Expected: dropped without blocking.
	case <-time.After(time.Second):
		t.Error("SendAction blocked when channel was full")
	}
}
