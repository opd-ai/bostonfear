package ebiten

import (
	"os"
	"path/filepath"
	"strings"
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")
	ls.SetPlayerID("player3")

	_, pid, _ := ls.Snapshot()
	if pid != "player3" {
		t.Errorf("playerID = %q, want player3", pid)
	}
}

// TestUpdateDiceResult_AppendEvent verifies UpdateDiceResult appends an entry to EventLog.
func TestUpdateDiceResult_AppendEvent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")
	dr := DiceResultData{PlayerID: "p1", Action: "ward", Success: false}
	ls.UpdateDiceResult(dr)

	events := ls.EventLogSnapshot()
	if len(events) == 0 {
		t.Fatal("expected event log entry")
	}
	if !strings.Contains(events[0].Text, "failed") {
		t.Errorf("event text %q does not contain expected label %q", events[0].Text, "failed")
	}
}

// TestUpdateGameEvent_AppendEvent verifies UpdateGameEvent stores and logs the event.
func TestUpdateGameEvent_AppendEvent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")

	for i := 0; i < 25; i++ {
		ls.UpdateDiceResult(DiceResultData{PlayerID: "p1", Action: "gather", Success: true})
	}

	events := ls.EventLogSnapshot()
	if len(events) > 20 {
		t.Errorf("EventLog length = %d, want ≤ 20", len(events))
	}
}

// TestConnectFormSnapshot_Defaults verifies the connect form starts with a
// host:port value derived from ServerURL.
func TestConnectFormSnapshot_Defaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")

	address, displayName := ls.ConnectFormSnapshot()
	if address != "localhost:8080" {
		t.Errorf("address = %q, want %q", address, "localhost:8080")
	}
	if displayName != "" {
		t.Errorf("displayName = %q, want empty", displayName)
	}
}

// TestSetConnectAddress_NormalizesURL verifies host:port input updates ServerURL
// with ws:// and /ws normalization.
func TestSetConnectAddress_NormalizesURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")

	ls.SetConnectAddress("example.org:9090")
	if ls.ServerURL != "ws://example.org:9090/ws" {
		t.Errorf("ServerURL = %q, want %q", ls.ServerURL, "ws://example.org:9090/ws")
	}

	ls.SetConnectAddress("wss://example.org/socket")
	if ls.ServerURL != "wss://example.org/socket/ws" {
		t.Errorf("ServerURL = %q, want %q", ls.ServerURL, "wss://example.org/socket/ws")
	}
}

// TestUXMetrics_FirstValidAction ensures the first valid action timestamp is
// captured once and time-to-first-valid-action is derived from session start.
func TestUXMetrics_FirstValidAction(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")

	base := time.Unix(1000, 0)
	ls.mu.Lock()
	ls.sessionStartedAt = base
	ls.mu.Unlock()

	ls.recordValidActionSentAt(base.Add(3 * time.Second))
	ls.recordValidActionSentAt(base.Add(5 * time.Second))

	metrics := ls.UXMetrics()
	if !metrics.HasFirstValidAction {
		t.Fatal("expected HasFirstValidAction = true")
	}
	if metrics.ValidActionsSent != 2 {
		t.Fatalf("ValidActionsSent = %d, want 2", metrics.ValidActionsSent)
	}
	if got, want := metrics.TimeToFirstValidAction, 3*time.Second; got != want {
		t.Fatalf("TimeToFirstValidAction = %v, want %v", got, want)
	}
}

// TestUXMetrics_InvalidRetryCounter verifies invalid retries and last reason are tracked.
func TestUXMetrics_InvalidRetryCounter(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ls := NewLocalState("ws://localhost:8080/ws")

	ls.RecordInvalidActionRetry("out-of-turn")
	ls.RecordInvalidActionRetry("trade-no-colocated-player")

	metrics := ls.UXMetrics()
	if got, want := metrics.InvalidActionRetries, 2; got != want {
		t.Fatalf("InvalidActionRetries = %d, want %d", got, want)
	}
	if got, want := metrics.LastInvalidReason, "trade-no-colocated-player"; got != want {
		t.Fatalf("LastInvalidReason = %q, want %q", got, want)
	}
}
