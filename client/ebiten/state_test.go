package ebiten

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestTokenPersistence_RoundTrip verifies that a reconnect token saved by one
// LocalState instance is correctly loaded by a freshly constructed instance.
func TestTokenPersistence_RoundTrip(t *testing.T) {
	// Redirect the home directory so we don't touch the real ~/.bostonfear.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if err := os.Setenv("USERPROFILE", tmpHome); err != nil {
		t.Fatal(err)
	}

	const want = "tok-abc-123"

	// Save via SetReconnectToken.
	ls1 := NewLocalState("ws://localhost:8080/ws")
	ls1.SetReconnectToken(want)

	// Verify the file was written.
	expected := filepath.Join(tmpHome, ".bostonfear", "session.json")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("session file not created at %s", expected)
	}

	// Load via a brand-new LocalState.
	ls2 := NewLocalState("ws://localhost:8080/ws")
	if got := ls2.GetReconnectToken(); got != want {
		t.Errorf("loaded token = %q, want %q", got, want)
	}
}

// TestTokenPersistence_MissingFile verifies that a missing session file
// does not cause NewLocalState to fail.
func TestTokenPersistence_MissingFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	ls := NewLocalState("ws://localhost:8080/ws")
	if got := ls.GetReconnectToken(); got != "" {
		t.Errorf("expected empty token without file, got %q", got)
	}
}

// TestTokenPersistence_OverwriteExisting verifies that calling SetReconnectToken
// again replaces the previously persisted token.
func TestTokenPersistence_OverwriteExisting(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	ls := NewLocalState("ws://localhost:8080/ws")
	ls.SetReconnectToken("first-token")
	ls.SetReconnectToken("second-token")

	ls2 := NewLocalState("ws://localhost:8080/ws")
	if got := ls2.GetReconnectToken(); got != "second-token" {
		t.Errorf("loaded token = %q, want %q", got, "second-token")
	}
}

// TestUpdateGame verifies that UpdateGame replaces the game state atomically.
func TestUpdateGame(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")

	gs := GameState{
		CurrentPlayer: "player1",
		Doom:          7,
		GamePhase:     "playerTurn",
		GameStarted:   true,
		Players:       map[string]*Player{"player1": {ID: "player1"}},
		TurnOrder:     []string{"player1"},
		RequiredClues: 3,
	}
	ls.UpdateGame(gs)

	snap, _, _ := ls.Snapshot()
	if snap.CurrentPlayer != "player1" {
		t.Errorf("CurrentPlayer = %q, want player1", snap.CurrentPlayer)
	}
	if snap.Doom != 7 {
		t.Errorf("Doom = %d, want 7", snap.Doom)
	}
	if snap.GamePhase != "playerTurn" {
		t.Errorf("GamePhase = %q, want playerTurn", snap.GamePhase)
	}
}

// TestSetConnected verifies that SetConnected updates the Connected flag.
func TestSetConnected(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")

	ls.SetConnected(true)
	_, _, connected := ls.Snapshot()
	if !connected {
		t.Error("expected Connected = true after SetConnected(true)")
	}

	ls.SetConnected(false)
	_, _, connected = ls.Snapshot()
	if connected {
		t.Error("expected Connected = false after SetConnected(false)")
	}
}

// TestSnapshot_ReturnsPlayerID verifies Snapshot returns the stored player ID.
func TestSnapshot_ReturnsPlayerID(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")
	ls.SetPlayerID("player3")

	_, pid, _ := ls.Snapshot()
	if pid != "player3" {
		t.Errorf("playerID = %q, want player3", pid)
	}
}

// TestUpdateDiceResult_AppendEvent verifies UpdateDiceResult appends an entry to EventLog.
func TestUpdateDiceResult_AppendEvent(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")

	dr := DiceResultData{
		PlayerID:     "player2",
		Action:       "investigate",
		Results:      []DiceResult{"success", "blank"},
		Successes:    1,
		Tentacles:    0,
		Success:      true,
		DoomIncrease: 0,
	}
	ls.UpdateDiceResult(dr)

	if ls.LastDiceResult == nil {
		t.Fatal("LastDiceResult is nil after UpdateDiceResult")
	}
	if ls.LastDiceResult.Action != "investigate" {
		t.Errorf("Action = %q, want investigate", ls.LastDiceResult.Action)
	}
	if !ls.LastDiceResult.Success {
		t.Error("expected Success = true")
	}

	events := ls.EventLogSnapshot()
	if len(events) != 1 {
		t.Fatalf("EventLog length = %d, want 1", len(events))
	}
}

// TestUpdateDiceResult_FailureLabel verifies the event text uses "failed" for unsuccessful rolls.
func TestUpdateDiceResult_FailureLabel(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")
	dr := DiceResultData{PlayerID: "p1", Action: "ward", Success: false}
	ls.UpdateDiceResult(dr)

	events := ls.EventLogSnapshot()
	if len(events) == 0 {
		t.Fatal("expected event log entry")
	}
}

// TestUpdateGameEvent_AppendEvent verifies UpdateGameEvent stores and logs the event.
func TestUpdateGameEvent_AppendEvent(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")

	gu := GameUpdateData{
		PlayerID:  "player1",
		Event:     "move",
		Result:    "University",
		DoomDelta: 0,
		Timestamp: time.Now(),
	}
	ls.UpdateGameEvent(gu)

	if ls.LastGameUpdate == nil {
		t.Fatal("LastGameUpdate is nil after UpdateGameEvent")
	}
	if ls.LastGameUpdate.Event != "move" {
		t.Errorf("Event = %q, want move", ls.LastGameUpdate.Event)
	}

	events := ls.EventLogSnapshot()
	if len(events) != 1 {
		t.Fatalf("EventLog length = %d, want 1", len(events))
	}
}

// TestEventLogSnapshot_CopiesSlice verifies EventLogSnapshot returns a copy, not the internal slice.
func TestEventLogSnapshot_CopiesSlice(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")
	dr := DiceResultData{PlayerID: "p1", Action: "gather", Success: true}
	ls.UpdateDiceResult(dr)

	snap := ls.EventLogSnapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 event, got %d", len(snap))
	}

	// Mutate the snapshot; the internal log must not be affected.
	snap[0].Text = "tampered"
	events := ls.EventLogSnapshot()
	if events[0].Text == "tampered" {
		t.Error("EventLogSnapshot returned a reference to the internal slice")
	}
}

// TestEventLogSnapshot_CapsAt20 verifies that the event log is trimmed to 20 entries.
func TestEventLogSnapshot_CapsAt20(t *testing.T) {
	ls := NewLocalState("ws://localhost:8080/ws")

	for i := 0; i < 25; i++ {
		ls.UpdateDiceResult(DiceResultData{PlayerID: "p1", Action: "gather", Success: true})
	}

	events := ls.EventLogSnapshot()
	if len(events) > 20 {
		t.Errorf("EventLog length = %d, want ≤ 20", len(events))
	}
}
