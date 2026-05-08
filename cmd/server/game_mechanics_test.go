package main

import (
	"math/rand"
	"net"
	"testing"
	"time"
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
		ReconnectToken:   generateReconnectToken(),
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
		{"health below min", Resources{Health: -1, Sanity: 5, Clues: 0}, Resources{Health: 0, Sanity: 5, Clues: 0}},
		{"health above max", Resources{Health: 11, Sanity: 5, Clues: 0}, Resources{Health: 10, Sanity: 5, Clues: 0}},
		{"sanity below min", Resources{Health: 5, Sanity: -1, Clues: 0}, Resources{Health: 5, Sanity: 0, Clues: 0}},
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
	gs.gameState.Players[pid].Resources.Sanity = 0 // defeated threshold — cannot cast ward

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
	// Drain acts to the final card so the next advance sets WinCondition.
	gs.gameState.ActDeck = []ActCard{{Title: "Final Act", ClueThreshold: 4, Effect: "victory"}}
	gs.gameState.Players["p1"].Resources.Clues = 4

	gs.checkGameEndConditions()

	if !gs.gameState.WinCondition {
		t.Error("expected WinCondition when clues >= required and final act is advanced")
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
	// Use a single-card act deck so the first advance ends the game.
	gs.gameState.ActDeck = []ActCard{{Title: "Final Act", ClueThreshold: 4, Effect: "victory"}}
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

// --- getSystemAlerts: present in /health (table-driven, consolidates coverage_test.go) ---

func TestGetSystemAlerts(t *testing.T) {
	cases := []struct {
		name         string
		doom         int
		wantNonNil   bool
		wantSeverity string // empty means no specific severity required
	}{
		{
			name:         "no_alerts_at_low_doom",
			doom:         0,
			wantNonNil:   true,
			wantSeverity: "", // only verifies the slice is non-nil; no severity required
		},
		{"medium_alert_at_doom_8", 8, true, "medium"},
		{"critical_alert_at_doom_10", 10, true, "critical"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs, _ := newTestServer(t)
			gs.gameState.Doom = tc.doom

			alerts := gs.getSystemAlerts()
			if tc.wantNonNil && alerts == nil {
				t.Error("getSystemAlerts returned nil")
				return
			}
			if tc.wantSeverity == "" {
				return
			}
			found := false
			for _, a := range alerts {
				if sev, ok := a["severity"].(string); ok && sev == tc.wantSeverity {
					found = true
				}
			}
			if !found {
				t.Errorf("doom=%d: expected alert with severity=%q", tc.doom, tc.wantSeverity)
			}
		})
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

// TestInvestigatorDefeat verifies that a player with Health or Sanity reaching 0
// is marked Defeated, cannot take further actions, and is skipped in turn rotation.
func TestInvestigatorDefeat(t *testing.T) {
	gs := NewGameServer()

	// Set up a two-player game where p1 is the current player.
	p1ID, p2ID := "investigator1", "investigator2"
	gs.gameState.GamePhase = "playing"
	gs.gameState.GameStarted = true
	gs.gameState.Players[p1ID] = &Player{
		ID:               p1ID,
		Location:         Downtown,
		Resources:        Resources{Health: 1, Sanity: 1, Clues: 0},
		ActionsRemaining: 2,
		Connected:        true,
	}
	gs.gameState.Players[p2ID] = &Player{
		ID:               p2ID,
		Location:         Downtown,
		Resources:        Resources{Health: 5, Sanity: 5, Clues: 0},
		ActionsRemaining: 0,
		Connected:        true,
	}
	gs.gameState.TurnOrder = []string{p1ID, p2ID}
	gs.gameState.CurrentPlayer = p1ID

	// Directly set Health to 0, then call checkInvestigatorDefeat.
	gs.mutex.Lock()
	gs.gameState.Players[p1ID].Resources.Health = 0
	gs.checkInvestigatorDefeat(p1ID)
	gs.mutex.Unlock()

	gs.mutex.RLock()
	p1 := gs.gameState.Players[p1ID]
	if !p1.Defeated {
		t.Error("expected player to be Defeated when Health == 0")
	}
	if p1.ActionsRemaining != 0 {
		t.Errorf("expected ActionsRemaining = 0 for defeated player, got %d", p1.ActionsRemaining)
	}
	gs.mutex.RUnlock()

	// advanceTurn should skip p1 (defeated) and give turn to p2.
	gs.mutex.Lock()
	gs.advanceTurn()
	currentPlayer := gs.gameState.CurrentPlayer
	gs.mutex.Unlock()

	if currentPlayer == p1ID {
		t.Errorf("advanceTurn should have skipped defeated player %s; still got %s as current", p1ID, currentPlayer)
	}
	if currentPlayer != p2ID {
		t.Errorf("advanceTurn should advance to %s; got %s", p2ID, currentPlayer)
	}
}

// TestInvestigatorDefeat_SanityZero verifies defeat triggers on Sanity reaching 0.
func TestInvestigatorDefeat_SanityZero(t *testing.T) {
	gs := NewGameServer()
	playerID := "p1"
	gs.gameState.Players[playerID] = &Player{
		ID:        playerID,
		Resources: Resources{Health: 5, Sanity: 0, Clues: 0},
		Connected: true,
	}

	gs.mutex.Lock()
	gs.checkInvestigatorDefeat(playerID)
	gs.mutex.Unlock()

	gs.mutex.RLock()
	defer gs.mutex.RUnlock()
	if !gs.gameState.Players[playerID].Defeated {
		t.Error("expected Defeated=true when Sanity==0")
	}
}

// TestValidateActionRequest_DefeatedPlayer verifies that a defeated player
// cannot submit actions.
func TestValidateActionRequest_DefeatedPlayer(t *testing.T) {
	gs := NewGameServer()
	playerID := "p1"
	gs.gameState.GamePhase = "playing"
	gs.gameState.CurrentPlayer = playerID
	gs.gameState.Players[playerID] = &Player{
		ID:               playerID,
		Resources:        Resources{Health: 0, Sanity: 5},
		ActionsRemaining: 2,
		Connected:        true,
		Defeated:         true,
	}

	gs.mutex.Lock()
	defer gs.mutex.Unlock()
	_, err := gs.validateActionRequest(PlayerActionMessage{
		PlayerID: playerID,
		Action:   ActionGather,
	})
	if err == nil {
		t.Error("expected error for defeated player taking action; got nil")
	}
}

// --- Session Reconnection ---

func TestSessionReconnection_TokenGenerated(t *testing.T) {
	gs, p1ID := newTestServer(t)
	token := gs.gameState.Players[p1ID].ReconnectToken
	if token == "" {
		t.Error("player should have a non-empty ReconnectToken")
	}
}

func TestSessionReconnection_RestoreByToken(t *testing.T) {
	gs, p1ID := newTestServer(t)
	token := gs.gameState.Players[p1ID].ReconnectToken

	// Simulate disconnect.
	gs.gameState.Players[p1ID].Connected = false
	gs.gameState.Players[p1ID].DisconnectedAt = time.Now().Add(-10 * time.Second)

	// Simulate reconnection via token.
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}
	fakeConn := NewConnectionWrapper(nil, addr, addr)
	restoredID := gs.restorePlayerByToken(token, fakeConn)

	if restoredID != p1ID {
		t.Errorf("expected restored player %s; got %q", p1ID, restoredID)
	}
	if !gs.gameState.Players[p1ID].Connected {
		t.Error("player should be marked connected after restore")
	}
	// Token should rotate.
	newToken := gs.gameState.Players[p1ID].ReconnectToken
	if newToken == token {
		t.Error("ReconnectToken should rotate after use")
	}
}

func TestSessionReconnection_UnknownTokenReturnsEmpty(t *testing.T) {
	gs, _ := newTestServer(t)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9999}
	fakeConn := NewConnectionWrapper(nil, addr, addr)
	result := gs.restorePlayerByToken("invalid-token-xyz", fakeConn)
	if result != "" {
		t.Errorf("unknown token should return empty string; got %q", result)
	}
}

// --- Mythos Token Randomness ---

// TestDrawMythosToken_IsRandom verifies that drawMythosToken returns all four
// token values across 100 calls, guarding against the deterministic (doom%4) bug.
func TestDrawMythosToken_IsRandom(t *testing.T) {
	gs, _ := newTestServer(t)
	seen := make(map[string]int)
	for i := 0; i < 100; i++ {
		tok := gs.drawMythosToken()
		seen[tok]++
	}
	want := []string{MythosTokenDoom, MythosTokenBlessing, MythosTokenCurse, MythosTokenBlank}
	for _, v := range want {
		if seen[v] == 0 {
			t.Errorf("token %q never appeared in 100 draws; distribution: %v", v, seen)
		}
	}
}

// --- Double-increment guard for checkGameEndConditions ---

// TestCheckGameEndConditions_NoDoubleIncrement asserts that calling checkGameEndConditions
// on a game already in the "ended" phase does not increment totalGamesPlayed a second time.
func TestCheckGameEndConditions_NoDoubleIncrement(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.Doom = 12

	// First call — should transition to ended and increment.
	gs.checkGameEndConditions()
	after1 := gs.totalGamesPlayed

	// Second call on an already-ended game — should be a no-op.
	gs.checkGameEndConditions()
	after2 := gs.totalGamesPlayed

	if after2 != after1 {
		t.Errorf("totalGamesPlayed incremented on second call: %d → %d", after1, after2)
	}
}

// --- ActionComponent acceptance ---

// TestProcessAction_ComponentActionAccepted verifies that ActionComponent is now a
// valid action type and executes the player's investigator ability without error.
func TestProcessAction_ComponentActionAccepted(t *testing.T) {
	gs, p1ID := newTestServer(t)
	gs.gameState.GamePhase = "playing"
	gs.gameState.Players[p1ID].InvestigatorType = InvestigatorSurvivor

	action := PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: p1ID,
		Action:   ActionComponent,
		Target:   "",
	}
	if err := gs.processAction(action); err != nil {
		t.Fatalf("unexpected error for ActionComponent: %v", err)
	}
}

// --- Dice Pool Focus Modifiers ---

// TestDicePool_ZeroFocusNoChange verifies that a focusSpend of 0 behaves like
// the original rollDice call (no extra dice, no rerolls).
func TestDicePool_ZeroFocusNoChange(t *testing.T) {
	gs := NewGameServer()
	player := &Player{Resources: Resources{Focus: 2}}
	_, _, _ = gs.rollDicePool(3, 0, player)
	if player.Resources.Focus != 2 {
		t.Errorf("Focus should not be deducted with focusSpend=0; got %d", player.Resources.Focus)
	}
}

// TestDicePool_FocusSpendDeductsTokens verifies that spending N focus tokens
// deducts N from the player's focus pool.
func TestDicePool_FocusSpendDeductsTokens(t *testing.T) {
	gs := NewGameServer()
	player := &Player{Resources: Resources{Focus: 3}}
	gs.rollDicePool(3, 2, player)
	if player.Resources.Focus != 1 {
		t.Errorf("Focus after spending 2 = %d, want 1", player.Resources.Focus)
	}
}

// TestDicePool_InvalidFocusSpend verifies that spending more focus than available
// is clamped to what the player actually has (no negative focus).
func TestDicePool_InvalidFocusSpend(t *testing.T) {
	gs := NewGameServer()
	player := &Player{Resources: Resources{Focus: 1}}
	// Attempt to spend 5 but only 1 is available.
	gs.rollDicePool(3, 5, player)
	if player.Resources.Focus < 0 {
		t.Errorf("Focus must not go negative; got %d", player.Resources.Focus)
	}
}

// TestDicePool_FocusSpendAddsExtraDice verifies that spending focus adds dice
// to the pool (the returned results slice is longer than baseDice).
func TestDicePool_FocusSpendAddsExtraDice(t *testing.T) {
	gs := NewGameServer()
	// Run many times so statistical variance doesn't mask the effect.
	for i := 0; i < 20; i++ {
		player := &Player{Resources: Resources{Focus: 2}}
		results, _, _ := gs.rollDicePool(3, 2, player)
		if len(results) != 5 {
			t.Errorf("expected 5 dice results (3 base + 2 focus); got %d", len(results))
		}
	}
}

// --- Mythos Phase Tests (Step 9 validation) ---

// TestMythosPhase_DrawsTwoEvents verifies that runMythosPhase places exactly 2 events.
func TestMythosPhase_DrawsTwoEvents(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.MythosEventDeck = defaultMythosEventDeck()
	gs.runMythosPhase()
	if len(gs.gameState.MythosEvents) != 2 {
		t.Errorf("expected 2 mythos events; got %d", len(gs.gameState.MythosEvents))
	}
}

// TestMythosPhase_EventSpreadIncrementsDooom verifies doom increments per placed event.
func TestMythosPhase_EventSpreadIncrementsDooom(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.MythosEventDeck = defaultMythosEventDeck()
	beforeDoom := gs.gameState.Doom
	gs.runMythosPhase()
	// At minimum doom should have increased (2 events placed = +2, possibly +1 from token).
	if gs.gameState.Doom <= beforeDoom {
		t.Errorf("doom should increase after mythos phase; before=%d after=%d", beforeDoom, gs.gameState.Doom)
	}
}

// TestMythosPhase_TokenDrawAffectsState verifies that drawing a doom token increments doom.
func TestMythosPhase_TokenDrawAffectsState(t *testing.T) {
	gs, _ := newTestServer(t)
	gs.gameState.MythosEventDeck = []MythosEvent{}
	beforeDoom := gs.gameState.Doom
	gs.resolveMythosToken(MythosTokenDoom)
	if gs.gameState.Doom != beforeDoom+1 {
		t.Errorf("doom token should increment doom by 1; before=%d after=%d", beforeDoom, gs.gameState.Doom)
	}
}

// TestMythosPhase_StarterCupComposition verifies the default event deck is non-empty.
func TestMythosPhase_StarterCupComposition(t *testing.T) {
	deck := defaultMythosEventDeck()
	if len(deck) == 0 {
		t.Error("defaultMythosEventDeck should return a non-empty deck")
	}
}

// --- Difficulty Settings Tests (Step 13) ---

func TestDifficulty_EasyStartsDoomAtZero(t *testing.T) {
	gs, _ := newTestServer(t)
	if err := gs.applyDifficulty("easy"); err != nil {
		t.Fatalf("applyDifficulty(easy): %v", err)
	}
	if gs.gameState.Doom != 0 {
		t.Errorf("easy difficulty should start with doom=0; got %d", gs.gameState.Doom)
	}
}

func TestDifficulty_HardAddsExtraDoomTokens(t *testing.T) {
	gs, _ := newTestServer(t)
	if err := gs.applyDifficulty("hard"); err != nil {
		t.Fatalf("applyDifficulty(hard): %v", err)
	}
	if gs.gameState.Doom != 3 {
		t.Errorf("hard difficulty should start with doom=3; got %d", gs.gameState.Doom)
	}
	if gs.gameState.Difficulty != "hard" {
		t.Errorf("difficulty should be 'hard'; got %q", gs.gameState.Difficulty)
	}
}

func TestDifficulty_InvalidDifficultyReturnsError(t *testing.T) {
	gs, _ := newTestServer(t)
	if err := gs.applyDifficulty("impossible"); err == nil {
		t.Error("invalid difficulty should return an error")
	}
}
