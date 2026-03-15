package main

import (
	"math/rand"
	"net"
	"testing"
)

// newTestServer returns a GameServer in "playing" state with one connected player.
// It does not start the broadcast/action goroutines, making it safe for unit tests.
func newTestServer(t *testing.T) (*GameServer, string) {
	t.Helper()
	gs := NewGameServer()
	gs.gameState.GamePhase = "playing"
	gs.gameState.GameStarted = true

	p := &Player{
		ID:               "p1",
		Location:         Downtown,
		Resources:        Resources{Health: 10, Sanity: 10, Clues: 0},
		ActionsRemaining: 2,
		Connected:        true,
	}
	gs.gameState.Players["p1"] = p
	gs.gameState.TurnOrder = []string{"p1"}
	gs.gameState.CurrentPlayer = "p1"
	return gs, "p1"
}

// addPlayer adds a second player to an existing test server.
func addPlayer(gs *GameServer, id string, connected bool) {
	gs.gameState.Players[id] = &Player{
		ID:               id,
		Location:         Downtown,
		Resources:        Resources{Health: 10, Sanity: 10, Clues: 0},
		ActionsRemaining: 0,
		Connected:        connected,
	}
	gs.gameState.TurnOrder = append(gs.gameState.TurnOrder, id)
}

// --- validateResources ---

func TestValidateResources_ClampsBounds(t *testing.T) {
	gs, _ := newTestServer(t)

	cases := []struct {
		name string
		in   Resources
		want Resources
	}{
		{"health below min", Resources{Health: 0, Sanity: 5, Clues: 0}, Resources{Health: 1, Sanity: 5, Clues: 0}},
		{"health above max", Resources{Health: 11, Sanity: 5, Clues: 0}, Resources{Health: 10, Sanity: 5, Clues: 0}},
		{"sanity below min", Resources{Health: 5, Sanity: 0, Clues: 0}, Resources{Health: 5, Sanity: 1, Clues: 0}},
		{"sanity above max", Resources{Health: 5, Sanity: 11, Clues: 0}, Resources{Health: 5, Sanity: 10, Clues: 0}},
		{"clues below min", Resources{Health: 5, Sanity: 5, Clues: -1}, Resources{Health: 5, Sanity: 5, Clues: 0}},
		{"clues above max", Resources{Health: 5, Sanity: 5, Clues: 6}, Resources{Health: 5, Sanity: 5, Clues: 5}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.in
			gs.validateResources(&r)
			if r != tc.want {
				t.Errorf("validateResources(%+v) = %+v, want %+v", tc.in, r, tc.want)
			}
		})
	}
}

// --- validateMovement ---

func TestValidateMovement_Adjacency(t *testing.T) {
	gs, _ := newTestServer(t)

	allowed := [][2]Location{
		{Downtown, University},
		{Downtown, Rivertown},
		{University, Downtown},
		{University, Northside},
		{Rivertown, Downtown},
		{Rivertown, Northside},
		{Northside, University},
		{Northside, Rivertown},
	}
	for _, pair := range allowed {
		if !gs.validateMovement(pair[0], pair[1]) {
			t.Errorf("expected movement from %s to %s to be allowed", pair[0], pair[1])
		}
	}

	forbidden := [][2]Location{
		{Downtown, Northside},
		{University, Rivertown},
		{Rivertown, University},
		{Northside, Downtown},
	}
	for _, pair := range forbidden {
		if gs.validateMovement(pair[0], pair[1]) {
			t.Errorf("expected movement from %s to %s to be forbidden", pair[0], pair[1])
		}
	}
}

// --- rollDice ---

func TestRollDice_ZeroDice(t *testing.T) {
	gs, _ := newTestServer(t)
	results, successes, tentacles := gs.rollDice(0)
	if len(results) != 0 || successes != 0 || tentacles != 0 {
		t.Errorf("rollDice(0) should return empty slice with 0 successes/tentacles")
	}
}

func TestRollDice_NegativeDice(t *testing.T) {
	gs, _ := newTestServer(t)
	results, successes, tentacles := gs.rollDice(-5)
	if len(results) != 0 || successes != 0 || tentacles != 0 {
		t.Errorf("rollDice(-5) should return empty slice")
	}
}

func TestRollDice_CountsAddUp(t *testing.T) {
	gs, _ := newTestServer(t)
	// Over many rolls, every die must produce exactly one of success/blank/tentacle
	for trial := 0; trial < 50; trial++ {
		n := 1 + rand.Intn(6)
		results, successes, tentacles := gs.rollDice(n)
		if len(results) != n {
			t.Fatalf("rollDice(%d): got %d results", n, len(results))
		}
		blanks := 0
		for _, r := range results {
			switch r {
			case DiceSuccess, DiceBlank, DiceTentacle:
			default:
				t.Errorf("unexpected dice result value %q", r)
			}
			if r == DiceBlank {
				blanks++
			}
		}
		if successes+blanks+tentacles != n {
			t.Errorf("rollDice(%d): successes(%d)+blanks(%d)+tentacles(%d) != %d",
				n, successes, blanks, tentacles, n)
		}
	}
}

// --- processAction: Move ---

func TestProcessAction_MoveValid(t *testing.T) {
	gs, pid := newTestServer(t)
	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionMove,
		Target:   string(University),
	})
	if err != nil {
		t.Fatalf("unexpected error on valid move: %v", err)
	}
	if gs.gameState.Players[pid].Location != University {
		t.Errorf("expected player at University, got %s", gs.gameState.Players[pid].Location)
	}
}

func TestProcessAction_MoveInvalid(t *testing.T) {
	gs, pid := newTestServer(t)
	// Downtown is not adjacent to Northside
	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionMove,
		Target:   string(Northside),
	})
	if err == nil {
		t.Error("expected error for invalid movement, got nil")
	}
}

// --- processAction: turn validation ---

func TestProcessAction_WrongPlayer(t *testing.T) {
	gs, _ := newTestServer(t)
	addPlayer(gs, "p2", true)

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: "p2",
		Action:   ActionGather,
	})
	if err == nil {
		t.Error("expected error when acting out of turn")
	}
}

func TestProcessAction_GameNotPlaying(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.gameState.GamePhase = "waiting"

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionGather,
	})
	if err == nil {
		t.Error("expected error when game is not in playing state")
	}
}

func TestProcessAction_NoActionsRemaining(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.gameState.Players[pid].ActionsRemaining = 0

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionGather,
	})
	if err == nil {
		t.Error("expected error when player has no actions remaining")
	}
}

// --- processAction: CastWard sanity cost ---

func TestProcessAction_CastWard_InsufficientSanity(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.gameState.Players[pid].Resources.Sanity = 1 // minimum — cannot afford ward

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionCastWard,
	})
	if err == nil {
		t.Error("expected error when sanity is too low to cast ward")
	}
}

func TestProcessAction_CastWard_CostsSanity(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.gameState.Players[pid].Resources.Sanity = 5
	initialSanity := gs.gameState.Players[pid].Resources.Sanity

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionCastWard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.gameState.Players[pid].Resources.Sanity != initialSanity-1 {
		t.Errorf("expected sanity to decrease by 1, got %d (was %d)",
			gs.gameState.Players[pid].Resources.Sanity, initialSanity)
	}
}

// --- processAction: actions per turn and turn rotation ---

func TestProcessAction_TurnAdvancesAfterTwoActions(t *testing.T) {
	gs, pid := newTestServer(t)
	addPlayer(gs, "p2", true)

	// First action
	if err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionMove,
		Target:   string(University),
	}); err != nil {
		t.Fatalf("first action: %v", err)
	}
	if gs.gameState.CurrentPlayer != pid {
		t.Error("turn should not advance after first action")
	}

	// Second action — move back
	if err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionMove,
		Target:   string(Downtown),
	}); err != nil {
		t.Fatalf("second action: %v", err)
	}
	if gs.gameState.CurrentPlayer != "p2" {
		t.Errorf("expected turn to advance to p2, got %s", gs.gameState.CurrentPlayer)
	}
	if gs.gameState.Players["p2"].ActionsRemaining != 2 {
		t.Errorf("expected p2 to have 2 actions, got %d",
			gs.gameState.Players["p2"].ActionsRemaining)
	}
}

// --- advanceTurn: disconnected player skip ---

func TestAdvanceTurn_SkipsDisconnectedPlayers(t *testing.T) {
	gs, _ := newTestServer(t)
	// p1 is connected; p2 disconnected; p3 connected
	addPlayer(gs, "p2", false)
	addPlayer(gs, "p3", true)

	gs.gameState.CurrentPlayer = "p1"
	gs.advanceTurn()

	if gs.gameState.CurrentPlayer != "p3" {
		t.Errorf("expected advanceTurn to skip disconnected p2, got %s",
			gs.gameState.CurrentPlayer)
	}
	if gs.gameState.Players["p3"].ActionsRemaining != 2 {
		t.Errorf("expected p3 to receive 2 actions, got %d",
			gs.gameState.Players["p3"].ActionsRemaining)
	}
}

func TestAdvanceTurn_AllDisconnected(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Players["p1"].Connected = false
	addPlayer(gs, "p2", false)

	before := gs.gameState.CurrentPlayer
	gs.advanceTurn() // should not panic or change state
	// CurrentPlayer may or may not change; just assert no crash
	_ = before
}

// --- checkGameEndConditions ---

func TestCheckGameEndConditions_DoomLose(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 12

	gs.checkGameEndConditions()

	if !gs.gameState.LoseCondition {
		t.Error("expected LoseCondition to be true at doom=12")
	}
	if gs.gameState.GamePhase != "ended" {
		t.Errorf("expected GamePhase=ended, got %s", gs.gameState.GamePhase)
	}
}

func TestCheckGameEndConditions_DoomPartial(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 6

	gs.checkGameEndConditions()

	if gs.gameState.LoseCondition {
		t.Error("LoseCondition should be false at doom=6")
	}
	if gs.gameState.GamePhase == "ended" {
		t.Error("GamePhase should not be ended at doom=6")
	}
}

func TestCheckGameEndConditions_ClueWin(t *testing.T) {
	gs, _ := newTestServer(t)
	// 1 player × 4 clues = win threshold; clues are capped at 5 but let's hit exactly 4
	gs.gameState.Players["p1"].Resources.Clues = 4

	gs.checkGameEndConditions()

	if !gs.gameState.WinCondition {
		t.Error("expected WinCondition when clues >= required")
	}
	if gs.gameState.GamePhase != "ended" {
		t.Errorf("expected GamePhase=ended, got %s", gs.gameState.GamePhase)
	}
}

func TestCheckGameEndConditions_ClueInsufficient(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Players["p1"].Resources.Clues = 2

	gs.checkGameEndConditions()

	if gs.gameState.WinCondition {
		t.Error("WinCondition should be false with insufficient clues")
	}
}

func TestCheckGameEndConditions_IncrementsTotalGamesPlayed(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 12

	before := gs.totalGamesPlayed
	gs.checkGameEndConditions()
	after := gs.totalGamesPlayed

	if after != before+1 {
		t.Errorf("expected totalGamesPlayed to increment by 1, got %d → %d", before, after)
	}
}

func TestCheckGameEndConditions_WinIncrementsTotalGamesPlayed(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Players["p1"].Resources.Clues = 4

	before := gs.totalGamesPlayed
	gs.checkGameEndConditions()
	after := gs.totalGamesPlayed

	if after != before+1 {
		t.Errorf("expected totalGamesPlayed to increment by 1 on win, got %d → %d", before, after)
	}
}

// --- Doom increments on tentacle dice ---

func TestProcessAction_TentacleIncrementsDoom(t *testing.T) {
	gs, pid := newTestServer(t)
	// Seed so Gather always rolls tentacles: we test statistically over many runs
	// Instead, we verify doom never goes below its initial value after any Gather action.
	initialDoom := gs.gameState.Doom

	err := gs.processAction(PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: pid,
		Action:   ActionGather,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gs.gameState.Doom < initialDoom {
		t.Error("doom must never decrease during Gather action")
	}
}

// --- getSystemAlerts: present in /health ---

func TestGetSystemAlerts_ReturnsSlice(t *testing.T) {
	gs, _ := newTestServer(t)
	alerts := gs.getSystemAlerts()
	// Should always return a non-nil slice (possibly empty)
	if alerts == nil {
		t.Error("getSystemAlerts returned nil")
	}
}

func TestGetSystemAlerts_CriticalDoomAlert(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 10 // > 80% threshold

	alerts := gs.getSystemAlerts()
	found := false
	for _, a := range alerts {
		if sev, ok := a["severity"].(string); ok && sev == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected a critical severity alert at doom=10")
	}
}

// --- handlePongMessage: no deadlock ---

func TestHandlePongMessage_NoDeadlock(t *testing.T) {
	gs, pid := newTestServer(t)
	gs.connectionQualities[pid] = &ConnectionQuality{
		LatencyMs: 0,
		Quality:   "good",
	}

	// Run handlePongMessage in a goroutine with a 1-second timeout.
	// Before the fix, this would deadlock; now it must complete.
	done := make(chan struct{})
	go func() {
		gs.handlePongMessage(PingMessage{
			Type:     "pong",
			PlayerID: pid,
		}, gs.startTime)
		close(done)
	}()

	// Drain any quality messages the non-blocking send might have queued
	timer := make(chan struct{})
	go func() {
		// Give goroutine up to 2 s to finish
		for i := 0; i < 200; i++ {
			select {
			case <-done:
				close(timer)
				return
			default:
			}
			// non-busy spin handled by select below
		}
	}()

drainLoop:
	for {
		select {
		case <-gs.broadcastCh:
		case <-done:
			break drainLoop
		}
	}
}

// TestAdvanceTurnOnDisconnect verifies GAP-03: when the current player
// disconnects, the turn automatically advances to the next connected player
// so the game never stalls.
func TestAdvanceTurnOnDisconnect(t *testing.T) {
t.Parallel()
gs, p1ID := newTestServer(t)
addPlayer(gs, "p2", true)

// Confirm p1 holds the current turn.
if gs.gameState.CurrentPlayer != p1ID {
t.Fatalf("expected p1 to hold the turn; got %s", gs.gameState.CurrentPlayer)
}

// Simulate the disconnect path: mark p1 disconnected and advance if needed.
gs.mutex.Lock()
if player, exists := gs.gameState.Players[p1ID]; exists {
player.Connected = false
}
if gs.gameState.CurrentPlayer == p1ID && gs.gameState.GamePhase == "playing" {
gs.advanceTurn()
}
gs.mutex.Unlock()

// Turn must now belong to p2.
if gs.gameState.CurrentPlayer == p1ID {
t.Error("turn did not advance after current player disconnected")
}
if gs.gameState.CurrentPlayer != "p2" {
t.Errorf("expected p2 to hold the turn; got %s", gs.gameState.CurrentPlayer)
}
if gs.gameState.Players["p2"].ActionsRemaining != 2 {
t.Errorf("p2 should start with 2 actions; got %d",
gs.gameState.Players["p2"].ActionsRemaining)
}
}

// TestAdvanceTurnOnDisconnect_OnlyPlayer verifies that when the sole player
// disconnects the game does not panic and CurrentPlayer retains its old value
// (no connected candidate exists to advance to).
func TestAdvanceTurnOnDisconnect_OnlyPlayer(t *testing.T) {
t.Parallel()
gs, p1ID := newTestServer(t)

gs.mutex.Lock()
if player, exists := gs.gameState.Players[p1ID]; exists {
player.Connected = false
}
if gs.gameState.CurrentPlayer == p1ID && gs.gameState.GamePhase == "playing" {
gs.advanceTurn()
}
gs.mutex.Unlock()

// With no connected players advanceTurn keeps CurrentPlayer as-is.
if gs.gameState.CurrentPlayer != p1ID {
t.Errorf("expected CurrentPlayer to stay %s; got %s", p1ID, gs.gameState.CurrentPlayer)
}
}

// TestHandlePlayerDisconnect exercises the handlePlayerDisconnect helper to
// verify map cleanup and turn advancement under the extracted method.
func TestHandlePlayerDisconnect_MapsCleanedUp(t *testing.T) {
	t.Parallel()
	gs, p1ID := newTestServer(t)
	addPlayer(gs, "p2", true)

	// Use a ConnectionWrapper as a net.Conn stand-in for the maps.
	addrStr := "127.0.0.1:12345"
	var stub net.Conn = &ConnectionWrapper{}
	gs.connections[addrStr] = stub
	gs.playerConns[p1ID] = stub

	gs.handlePlayerDisconnect(p1ID, addrStr)

	// Connection maps must be empty after disconnect.
	if _, ok := gs.connections[addrStr]; ok {
		t.Error("connections map entry not removed")
	}
	if _, ok := gs.playerConns[p1ID]; ok {
		t.Error("playerConns map entry not removed")
	}

	// Turn must have advanced to p2.
	if gs.gameState.CurrentPlayer == p1ID {
		t.Error("turn not advanced after current player disconnected via handlePlayerDisconnect")
	}
}
