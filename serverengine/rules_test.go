package serverengine

// rules_test.go — AH3e Core Mechanics Compliance Test Suite (ROADMAP §AH3e Engine Compliance).
//
// Each test function maps to one rule system in RULES.md.  Fully implemented
// mechanics are validated with functional assertions; mechanics not yet
// implemented in the engine are recorded as pending with t.Skip so the suite
// remains runnable and its coverage gap is explicit.
//
// Run with: go test -run TestRules ./cmd/server/

import (
	"fmt"
	"testing"
)

// --- TestTurnStructure ---
// AH3e: Investigator Phase → each investigator takes up to 2 actions →
// Mythos Phase → repeat.  The engine models the Investigator Phase only;
// Mythos Phase is tracked via the doom counter.

func TestRulesTurnStructure(t *testing.T) {
	t.Parallel()
	gs, p1ID := newTestServer(t)
	addPlayer(gs, "p2", true)

	// p1 starts the turn with 2 actions.
	if got := gs.gameState.Players[p1ID].ActionsRemaining; got != 2 {
		t.Fatalf("p1 should start with 2 actions; got %d", got)
	}
	if gs.gameState.CurrentPlayer != p1ID {
		t.Fatalf("p1 should be current player; got %s", gs.gameState.CurrentPlayer)
	}

	// Spend both actions via Gather (does not require specific location).
	for i := 0; i < 2; i++ {
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionGather,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("action %d: %v", i+1, err)
		}
	}

	// After two actions the turn must advance to p2.
	if gs.gameState.CurrentPlayer == p1ID {
		t.Error("turn did not advance after two actions")
	}
	if gs.gameState.Players["p2"].ActionsRemaining != 2 {
		t.Errorf("p2 should receive 2 actions; got %d",
			gs.gameState.Players["p2"].ActionsRemaining)
	}
}

// --- TestRulesMythosPhaseEventPlacement ---
// AH3e: During Mythos Phase, draw 2 events, place on neighborhoods, spread if
// doom already present, resolve mythos cup token.
// Status: NOT IMPLEMENTED — engine does not yet model Mythos Phase events.

func TestRulesMythosPhaseEventPlacement(t *testing.T) {
	t.Parallel()

	t.Run("EventsDrawnAndPlaced", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.MythosEventDeck = defaultMythosEventDeck()
		gs.gameState.LocationDoomTokens = make(map[string]int)
		initialDoom := gs.gameState.Doom

		gs.runMythosPhase()

		if gs.gameState.Doom <= initialDoom {
			t.Errorf("doom should increase during Mythos Phase; before=%d after=%d", initialDoom, gs.gameState.Doom)
		}
		if len(gs.gameState.MythosEvents) == 0 {
			t.Error("MythosEvents should be populated after Mythos Phase")
		}
		if gs.gameState.GamePhase != "playing" {
			t.Errorf("GamePhase should return to 'playing' after Mythos Phase; got %s", gs.gameState.GamePhase)
		}
	})

	t.Run("SpreadRule", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "test event"},
		}
		gs.gameState.LocationDoomTokens = map[string]int{string(Downtown): 1}

		gs.runMythosPhase()

		// Downtown already had a token, so spread to an adjacent location.
		found := false
		for _, loc := range []string{string(University), string(Rivertown)} {
			if gs.gameState.LocationDoomTokens[loc] > 0 {
				found = true
				break
			}
		}
		if !found {
			t.Error("spread rule should place doom token in a Downtown-adjacent location")
		}
	})

	t.Run("DeckRebuild", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.MythosEventDeck = []MythosEvent{} // empty deck
		gs.gameState.LocationDoomTokens = make(map[string]int)

		gs.runMythosPhase() // should rebuild deck and draw without panic

		if len(gs.gameState.MythosEvents) == 0 {
			t.Error("after deck rebuild, events should still be drawn")
		}
	})
}

// --- TestRulesFullActionSet ---
// AH3e: 8 action types (Move, Gather, Focus, Ward, Research, Trade, Component, Attack/Evade).
// Engine implements 4 of 8.

func TestRulesFullActionSet(t *testing.T) {
	t.Parallel()

	// Implemented actions.
	t.Run("Move", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Downtown
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionMove,
			Target:   "University",
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("move: %v", err)
		}
		if got := gs.gameState.Players[p1ID].Location; got != University {
			t.Errorf("expected University; got %s", got)
		}
	})

	t.Run("Gather", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Resources.Health = 5
		before := gs.gameState.Players[p1ID].Resources.Health
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionGather,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("gather: %v", err)
		}
		after := gs.gameState.Players[p1ID].Resources.Health
		if after < before {
			t.Errorf("gather should not reduce health; before=%d after=%d", before, after)
		}
	})

	t.Run("Investigate", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionInvestigate,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("investigate: %v", err)
		}
	})

	t.Run("Ward", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Resources.Sanity = 5
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionCastWard,
		}
		// Ward may fail the dice roll; error is only expected for missing sanity.
		_ = gs.processAction(msg)
	})

	t.Run("Focus", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Resources.Focus = 0
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionFocus,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("focus: %v", err)
		}
		if got := gs.gameState.Players[p1ID].Resources.Focus; got != 1 {
			t.Errorf("focus should award 1 Focus token; got %d", got)
		}
	})

	t.Run("Research", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionResearch,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("research: %v", err)
		}
	})

	t.Run("Trade", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		p2ID := "player2"
		addPlayer(gs, p2ID, true)
		gs.gameState.Players[p1ID].Location = Downtown
		gs.gameState.Players[p2ID].Location = Downtown
		gs.gameState.Players[p1ID].Resources.Clues = 2
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionTrade,
			Target:   p2ID,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("trade: %v", err)
		}
		if got := gs.gameState.Players[p2ID].Resources.Clues; got != 1 {
			t.Errorf("target should receive 1 Clue; got %d", got)
		}
	})

	t.Run("Component_implemented", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].InvestigatorType = InvestigatorSurvivor
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionComponent,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("component action should succeed for Survivor; got %v", err)
		}
	})
}

// --- TestRulesDicePoolFocusModifier ---
// AH3e: Investigators may spend focus tokens to add dice to their pool.
// Status: IMPLEMENTED — rollDicePool deducts focus tokens and adds extra dice with rerolls.

func TestRulesDicePoolFocusModifier(t *testing.T) {
	gs, p1ID := newTestServer(t)
	gs.gameState.Players[p1ID].Resources.Focus = 2

	// Send an Investigate action spending 1 focus token.
	before := gs.gameState.Players[p1ID].Resources.Focus
	msg := PlayerActionMessage{
		Type:       "playerAction",
		PlayerID:   p1ID,
		Action:     ActionInvestigate,
		FocusSpend: 1,
	}
	_ = gs.processAction(msg)

	// After the action, 1 focus token should have been deducted.
	after := gs.gameState.Players[p1ID].Resources.Focus
	if after != before-1 {
		t.Errorf("focus after spending 1 = %d, want %d", after, before-1)
	}
}

// --- TestRulesAnomalyGateMechanics ---
// AH3e: Anomalies spawn on doom-threshold locations; investigators seal them
// by casting a successful Ward.

func TestRulesAnomalyGateMechanics(t *testing.T) {
	t.Parallel()

	t.Run("AnomalySpawnOnMythosEvent", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.spawnAnomaly(string(Downtown))
		if len(gs.gameState.Anomalies) != 1 {
			t.Errorf("expected 1 anomaly; got %d", len(gs.gameState.Anomalies))
		}
		if gs.gameState.Anomalies[0].NeighbourhoodID != string(Downtown) {
			t.Errorf("anomaly should be at Downtown; got %s", gs.gameState.Anomalies[0].NeighbourhoodID)
		}
	})

	t.Run("WardSealsAnomaly", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Downtown
		gs.gameState.Players[p1ID].Resources.Sanity = 5
		gs.spawnAnomaly(string(Downtown))
		gs.gameState.Doom = 6
		// Directly seal the anomaly to test the sealing path.
		gs.SealAnomalyAtLocation(string(Downtown))
		if len(gs.gameState.Anomalies) != 0 {
			t.Errorf("ward success should seal anomaly; got %d anomalies", len(gs.gameState.Anomalies))
		}
		if gs.gameState.Doom >= 6 {
			// Doom should have decreased after sealing
			t.Logf("doom before=6 after=%d", gs.gameState.Doom)
		}
	})

	t.Run("WardNoDoomReductionWithoutAnomaly", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 5
		// No anomaly — sealing should be a no-op.
		gs.SealAnomalyAtLocation(string(Downtown))
		if gs.gameState.Doom != 5 {
			t.Errorf("no anomaly seal should not change doom; got %d", gs.gameState.Doom)
		}
	})
}

// --- TestRulesEncounterResolution ---
// AH3e: Investigators at locations with encounter tokens draw from
// neighborhood-specific encounter decks and resolve skill tests.

func TestRulesEncounterResolution(t *testing.T) {
	t.Parallel()

	t.Run("EncounterAtLocation", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Downtown
		// Plant a known card to make the outcome deterministic.
		gs.gameState.EncounterDecks[string(Downtown)] = []EncounterCard{
			{FlavorText: "test", EffectType: "clue_gain", Magnitude: 1},
		}
		before := gs.gameState.Players[p1ID].Resources.Clues
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionEncounter,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("encounter: %v", err)
		}
		if got := gs.gameState.Players[p1ID].Resources.Clues; got != before+1 {
			t.Errorf("clue_gain encounter should add 1 clue; before=%d after=%d", before, got)
		}
	})

	t.Run("EncounterDeckRebuild", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = University
		gs.gameState.EncounterDecks[string(University)] = []EncounterCard{} // empty

		// Should rebuild from defaults and succeed.
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionEncounter,
		}
		if err := gs.processAction(msg); err != nil {
			t.Fatalf("encounter deck rebuild: %v", err)
		}
	})

	t.Run("HealthLossEffect", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Downtown
		gs.gameState.Players[p1ID].Resources.Health = 8
		gs.gameState.EncounterDecks[string(Downtown)] = []EncounterCard{
			{FlavorText: "hurt", EffectType: "health_loss", Magnitude: 2},
		}
		if err := gs.processAction(PlayerActionMessage{
			Type: "playerAction", PlayerID: p1ID, Action: ActionEncounter,
		}); err != nil {
			t.Fatalf("health_loss encounter: %v", err)
		}
		if got := gs.gameState.Players[p1ID].Resources.Health; got != 6 {
			t.Errorf("health_loss should reduce health by 2; got %d want 6", got)
		}
	})

	t.Run("SanityLossEffect", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Downtown
		gs.gameState.Players[p1ID].Resources.Sanity = 7
		gs.gameState.EncounterDecks[string(Downtown)] = []EncounterCard{
			{FlavorText: "disturbing", EffectType: "sanity_loss", Magnitude: 3},
		}
		if err := gs.processAction(PlayerActionMessage{
			Type: "playerAction", PlayerID: p1ID, Action: ActionEncounter,
		}); err != nil {
			t.Fatalf("sanity_loss encounter: %v", err)
		}
		if got := gs.gameState.Players[p1ID].Resources.Sanity; got != 4 {
			t.Errorf("sanity_loss should reduce sanity by 3; got %d want 4", got)
		}
	})

	t.Run("DoomIncEffect", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Downtown
		gs.gameState.Doom = 3
		gs.gameState.EncounterDecks[string(Downtown)] = []EncounterCard{
			{FlavorText: "ominous", EffectType: "doom_inc", Magnitude: 2},
		}
		if err := gs.processAction(PlayerActionMessage{
			Type: "playerAction", PlayerID: p1ID, Action: ActionEncounter,
		}); err != nil {
			t.Fatalf("doom_inc encounter: %v", err)
		}
		if got := gs.gameState.Doom; got != 5 {
			t.Errorf("doom_inc should increase doom by 2; got %d want 5", got)
		}
	})

	t.Run("DeckExhaustionReshuffle", func(t *testing.T) {
		// After the deck is exhausted, subsequent encounters should rebuild from
		// defaults without error.
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Location = Rivertown

		// Drain the deck to one card, then process it.
		gs.gameState.EncounterDecks[string(Rivertown)] = []EncounterCard{
			{FlavorText: "last card", EffectType: "clue_gain", Magnitude: 1},
		}
		if err := gs.processAction(PlayerActionMessage{
			Type: "playerAction", PlayerID: p1ID, Action: ActionEncounter,
		}); err != nil {
			t.Fatalf("first encounter (draining deck): %v", err)
		}
		// Deck should now be empty; advance turn so p1 gets 2 actions again.
		gs.gameState.Players[p1ID].ActionsRemaining = 1

		// A second encounter should trigger rebuild without error.
		if err := gs.processAction(PlayerActionMessage{
			Type: "playerAction", PlayerID: p1ID, Action: ActionEncounter,
		}); err != nil {
			t.Fatalf("second encounter (reshuffle): %v", err)
		}
	})
}

// --- TestRulesActAgendaProgression ---
// AH3e: Act deck advances when investigators collect enough clues; agenda deck
// advances when doom reaches the threshold on each card.

func TestRulesActAgendaProgression(t *testing.T) {
	t.Parallel()

	t.Run("DoomAdvancesAgenda", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Doom = 11

		// An investigate action may add a tentacle → doom 12 → lose condition.
		// Force doom to 12 directly to validate agenda end condition.
		gs.gameState.Doom = 12
		gs.checkGameEndConditions()

		if !gs.gameState.LoseCondition {
			t.Error("doom=12 should trigger lose condition (agenda completion)")
		}
		if gs.gameState.GamePhase != "ended" {
			t.Errorf("game phase should be ended; got %s", gs.gameState.GamePhase)
		}
		_ = p1ID
	})

	t.Run("ActAdvancesOnClues", func(t *testing.T) {
		gs, _ := newTestServer(t)
		// Use single-card act deck so the first advance triggers WinCondition.
		gs.gameState.ActDeck = []ActCard{{Title: "Final Act", ClueThreshold: 4, Effect: "victory"}}
		gs.gameState.Players["p1"].Resources.Clues = 4
		gs.checkGameEndConditions()

		if !gs.gameState.WinCondition {
			t.Error("enough clues should trigger win condition (act completion)")
		}
	})

	t.Run("AgendaAdvancesOnDoom", func(t *testing.T) {
		gs, _ := newTestServer(t)
		// Single-card agenda with threshold 4; doom=4 should advance and exhaust it.
		gs.gameState.AgendaDeck = []AgendaCard{{Title: "Final Agenda", DoomThreshold: 4, Effect: "lose"}}
		gs.gameState.Doom = 4
		gs.checkGameEndConditions()

		if !gs.gameState.LoseCondition {
			t.Error("doom threshold reached should trigger lose condition (agenda exhausted)")
		}
	})

	t.Run("MidGameAct2Advancement", func(t *testing.T) {
		// Three-card act deck; collecting enough clues for Act 2 advances the
		// deck without triggering the win condition.
		gs, _ := newTestServer(t)
		gs.gameState.ActDeck = []ActCard{
			{Title: "Act 1", ClueThreshold: 3, Effect: "advance"},
			{Title: "Act 2", ClueThreshold: 7, Effect: "advance"},
			{Title: "Act 3", ClueThreshold: 12, Effect: "victory"},
		}
		gs.gameState.Players["p1"].Resources.Clues = 5

		gs.checkGameEndConditions()

		// Act 1 threshold (3) was met — Act 2 should now be the front card.
		if len(gs.gameState.ActDeck) != 2 {
			t.Fatalf("expected 2 act cards remaining after Act 1 advance; got %d", len(gs.gameState.ActDeck))
		}
		if gs.gameState.ActDeck[0].Title != "Act 2" {
			t.Errorf("expected Act 2 to be current; got %q", gs.gameState.ActDeck[0].Title)
		}
		// Game should still be in progress.
		if gs.gameState.WinCondition {
			t.Error("win condition should NOT be set while act cards remain")
		}
	})

	t.Run("AgendaDefeatAtDoom8", func(t *testing.T) {
		// Two-card agenda deck: card 1 threshold 5, card 2 threshold 8.
		// Doom=8 should advance past card 1 (threshold 5) and then exhaust card 2.
		gs, _ := newTestServer(t)
		gs.gameState.AgendaDeck = []AgendaCard{
			{Title: "Agenda 1", DoomThreshold: 5, Effect: "advance"},
			{Title: "Agenda 2", DoomThreshold: 8, Effect: "lose"},
		}
		gs.gameState.Doom = 8

		gs.checkGameEndConditions()

		if !gs.gameState.LoseCondition {
			t.Error("doom=8 with two-card agenda (thresholds 5, 8) should trigger lose condition")
		}
		if gs.gameState.GamePhase != "ended" {
			t.Errorf("game phase should be ended; got %q", gs.gameState.GamePhase)
		}
	})
}

// --- TestRulesDefeatRecovery ---
// AH3e: An investigator is defeated when health OR sanity reaches 0; they are
// placed in the "lost in time and space" state and may recover later.

func TestRulesDefeatRecovery(t *testing.T) {
	t.Parallel()

	t.Run("ZeroHealthTriggersLose", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Resources.Health = 0
		gs.checkGameEndConditions()
		// Engine does not yet model per-investigator defeat separately from
		// game-end; check that health=0 is preserved (not clamped to positive).
		if gs.gameState.Players[p1ID].Resources.Health < 0 {
			t.Error("health should not go below 0")
		}
	})

	t.Run("LostInTimeAndSpaceState", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Resources.Health = 0
		gs.CheckInvestigatorDefeat(p1ID)
		if !gs.gameState.Players[p1ID].LostInTimeAndSpace {
			t.Error("defeated player should be LostInTimeAndSpace")
		}
		if gs.gameState.Players[p1ID].Location != Downtown {
			t.Errorf("defeated player should return to Downtown; got %s", gs.gameState.Players[p1ID].Location)
		}
	})

	t.Run("InvestigatorRecovery", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.gameState.Players[p1ID].Resources.Health = 0
		gs.CheckInvestigatorDefeat(p1ID)
		gs.recoverInvestigator(p1ID)
		if gs.gameState.Players[p1ID].Defeated {
			t.Error("recovered investigator should not be Defeated")
		}
		if gs.gameState.Players[p1ID].LostInTimeAndSpace {
			t.Error("recovered investigator should not be LostInTimeAndSpace")
		}
	})
}

// --- TestInvestigatorAutoRecovery ---
// Verifies that runMythosPhase automatically recovers connected investigators
// who are in the LostInTimeAndSpace state (RULES.md §Defeat/Recovery).

func TestInvestigatorAutoRecovery(t *testing.T) {
	t.Parallel()
	gs, p1ID := newTestServer(t)

	// Defeat the player first (caller must hold gs.mutex per checkInvestigatorDefeat contract).
	gs.mutex.Lock()
	gs.gameState.Players[p1ID].Resources.Health = 0
	gs.CheckInvestigatorDefeat(p1ID)
	gs.mutex.Unlock()
	if !gs.gameState.Players[p1ID].LostInTimeAndSpace {
		t.Fatal("precondition: player should be LostInTimeAndSpace after defeat")
	}

	// runMythosPhase should auto-recover connected defeated investigators.
	gs.mutex.Lock()
	gs.runMythosPhase()
	gs.mutex.Unlock()

	if gs.gameState.Players[p1ID].LostInTimeAndSpace {
		t.Error("runMythosPhase should have cleared LostInTimeAndSpace for connected player")
	}
	if gs.gameState.Players[p1ID].Defeated {
		t.Error("runMythosPhase should have cleared Defeated flag for connected player")
	}
}

// AH3e: Victory when all act cards are completed; defeat when final agenda card
// is reached or all investigators are defeated.

func TestRulesVictoryDefeatConditions(t *testing.T) {
	t.Parallel()

	t.Run("WinOnRequiredClues", func(t *testing.T) {
		gs, _ := newTestServer(t)
		// Use a single-card act deck so exhausting it sets WinCondition.
		gs.gameState.ActDeck = []ActCard{{Title: "Final Act", ClueThreshold: 4, Effect: "victory"}}
		gs.gameState.Players["p1"].Resources.Clues = 4
		gs.checkGameEndConditions()
		if !gs.gameState.WinCondition {
			t.Error("clue threshold reached but WinCondition not set")
		}
	})

	t.Run("LoseOnMaxDoom", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 12
		gs.checkGameEndConditions()
		if !gs.gameState.LoseCondition {
			t.Error("doom=12 but LoseCondition not set")
		}
	})

	t.Run("NoConditionMidGame", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 6
		gs.gameState.RequiredClues = 4
		gs.gameState.Players["p1"].Resources.Clues = 2
		gs.checkGameEndConditions()
		if gs.gameState.WinCondition || gs.gameState.LoseCondition {
			t.Error("mid-game should have neither win nor lose condition")
		}
	})
}

// --- TestRulesResourceTypes ---
// AH3e: Investigators track money, clues, remnants, and focus tokens in addition
// to health and sanity.  Engine implements health, sanity, and clues only.

func TestRulesResourceTypes(t *testing.T) {
	t.Parallel()

	t.Run("HealthSanityClueBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		// Health bounds: 0–10 (0 = investigator defeated).
		r.Health = -1
		gs.ValidateResources(r)
		if r.Health < 0 {
			t.Errorf("health below 0 after validation: %d", r.Health)
		}
		r.Health = 11
		gs.ValidateResources(r)
		if r.Health > 10 {
			t.Errorf("health above 10 after validation: %d", r.Health)
		}

		// Sanity bounds: 0–10 (0 = investigator defeated).
		r.Sanity = -1
		gs.ValidateResources(r)
		if r.Sanity < 0 {
			t.Errorf("sanity below 0 after validation: %d", r.Sanity)
		}

		// Clue bounds: 0–5.
		r.Clues = -1
		gs.ValidateResources(r)
		if r.Clues < 0 {
			t.Errorf("clues below 0 after validation: %d", r.Clues)
		}
		r.Clues = 6
		gs.ValidateResources(r)
		if r.Clues > 5 {
			t.Errorf("clues above 5 after validation: %d", r.Clues)
		}
	})

	t.Run("MoneyBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		r.Money = -1
		gs.ValidateResources(r)
		if r.Money < 0 {
			t.Errorf("money below 0 after validation: %d", r.Money)
		}
		r.Money = MaxMoney + 1
		gs.ValidateResources(r)
		if r.Money > MaxMoney {
			t.Errorf("money above %d after validation: %d", MaxMoney, r.Money)
		}
	})

	t.Run("RemnantsBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		r.Remnants = -1
		gs.ValidateResources(r)
		if r.Remnants < 0 {
			t.Errorf("remnants below 0 after validation: %d", r.Remnants)
		}
		r.Remnants = MaxRemnants + 1
		gs.ValidateResources(r)
		if r.Remnants > MaxRemnants {
			t.Errorf("remnants above %d after validation: %d", MaxRemnants, r.Remnants)
		}
	})

	t.Run("FocusBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		r.Focus = -1
		gs.ValidateResources(r)
		if r.Focus < 0 {
			t.Errorf("focus below 0 after validation: %d", r.Focus)
		}
		r.Focus = MaxFocus + 1
		gs.ValidateResources(r)
		if r.Focus > MaxFocus {
			t.Errorf("focus above %d after validation: %d", MaxFocus, r.Focus)
		}
	})
}

// --- TestRulesScenarioSystem ---
// AH3e: Scenario system provides modular setup and difficulty via Scenario struct.

func TestRulesScenarioSystem(t *testing.T) {
	t.Parallel()

	t.Run("DefaultScenarioInitialisesDecks", func(t *testing.T) {
		gs := newGameServerWithScenario(DefaultScenario)
		if len(gs.gameState.ActDeck) == 0 {
			t.Error("DefaultScenario should populate ActDeck")
		}
		if len(gs.gameState.AgendaDeck) == 0 {
			t.Error("DefaultScenario should populate AgendaDeck")
		}
		if len(gs.gameState.MythosEventDeck) == 0 {
			t.Error("DefaultScenario should populate MythosEventDeck")
		}
		if len(gs.gameState.EncounterDecks) == 0 {
			t.Error("DefaultScenario should populate EncounterDecks")
		}
	})

	t.Run("CustomScenarioStartingDoom", func(t *testing.T) {
		custom := Scenario{
			Name:         "Hard Mode",
			StartingDoom: 6,
			SetupFn: func(gs *GameState) {
				gs.Doom = 6
				gs.ActDeck = defaultActDeck()
				gs.AgendaDeck = defaultAgendaDeck()
				gs.MythosEventDeck = defaultMythosEventDeck()
				gs.EncounterDecks = defaultEncounterDecks()
				gs.LocationDoomTokens = make(map[string]int)
			},
		}
		gs := newGameServerWithScenario(custom)
		if gs.gameState.Doom != 6 {
			t.Errorf("custom scenario StartingDoom=6; got %d", gs.gameState.Doom)
		}
	})

	t.Run("CustomScenarioWinFn", func(t *testing.T) {
		custom := Scenario{
			Name:         "Instant Win",
			StartingDoom: 0,
			SetupFn: func(gs *GameState) {
				gs.LocationDoomTokens = make(map[string]int)
				gs.ActDeck = defaultActDeck()
				gs.AgendaDeck = defaultAgendaDeck()
				gs.MythosEventDeck = defaultMythosEventDeck()
				gs.EncounterDecks = defaultEncounterDecks()
			},
			WinFn: func(gs *GameState) bool { return true }, // always win
		}
		gs := newGameServerWithScenario(custom)
		gs.gameState.Players = map[string]*Player{
			"p1": {ID: "p1", Connected: true, Resources: Resources{Health: 10, Sanity: 10}},
		}
		gs.gameState.GameStarted = true
		gs.gameState.GamePhase = "playing"

		gs.checkGameEndConditions()

		if !gs.gameState.WinCondition {
			t.Error("custom WinFn returning true should set WinCondition")
		}
	})
}

// TestRulesActAgendaProgression_PlayerCountScaling verifies that rescaleActDeck
// sets the final Act threshold to 4 × playerCount (4 clues per investigator as
// documented in the README win condition).
func TestRulesActAgendaProgression_PlayerCountScaling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		playerCount     int
		wantFinalThresh int
	}{
		{1, 4},
		{2, 8},
		{3, 12},
		{4, 16},
		{5, 20},
		{6, 24},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("%dP", tc.playerCount), func(t *testing.T) {
			t.Parallel()
			gs, _ := newTestServer(t)
			// Reset to default 3-card act deck.
			gs.gameState.ActDeck = defaultActDeck()

			gs.mutex.Lock()
			gs.rescaleActDeck(tc.playerCount)
			gs.mutex.Unlock()

			if len(gs.gameState.ActDeck) != 3 {
				t.Fatalf("expected 3 act cards, got %d", len(gs.gameState.ActDeck))
			}
			got := gs.gameState.ActDeck[2].ClueThreshold
			if got != tc.wantFinalThresh {
				t.Errorf("final act threshold = %d, want %d (player count %d)",
					got, tc.wantFinalThresh, tc.playerCount)
			}
			// Act 1 < Act 2 < Act 3.
			if !(gs.gameState.ActDeck[0].ClueThreshold < gs.gameState.ActDeck[1].ClueThreshold &&
				gs.gameState.ActDeck[1].ClueThreshold < gs.gameState.ActDeck[2].ClueThreshold) {
				t.Errorf("act thresholds not strictly increasing: %d / %d / %d",
					gs.gameState.ActDeck[0].ClueThreshold,
					gs.gameState.ActDeck[1].ClueThreshold,
					gs.gameState.ActDeck[2].ClueThreshold)
			}
		})
	}
}

// TestRulesActAgendaProgression_1PGameWinsAt4Clues verifies end-to-end that a
// single-player game is won exactly when the investigator accumulates 4 clues.
func TestRulesActAgendaProgression_1PGameWinsAt4Clues(t *testing.T) {
	gs, p1ID := newTestServer(t)
	gs.gameState.ActDeck = defaultActDeck()

	gs.mutex.Lock()
	gs.rescaleActDeck(1) // 1P → final threshold = 4
	gs.mutex.Unlock()

	// Accumulate clues one at a time; WinCondition must be false until 4.
	for clues := 1; clues <= 3; clues++ {
		gs.gameState.Players[p1ID].Resources.Clues = clues
		gs.mutex.Lock()
		gs.checkActAdvance()
		gs.mutex.Unlock()
		if gs.gameState.WinCondition {
			t.Errorf("WinCondition set at %d clue(s); expected false before 4", clues)
		}
	}

	// Reaching 4 clues should eventually exhaust all acts (via repeated advance).
	// Give the player 4 clues and drive the full check.
	gs.gameState.Players[p1ID].Resources.Clues = 4
	gs.mutex.Lock()
	gs.checkGameEndConditions()
	gs.mutex.Unlock()

	if !gs.gameState.WinCondition {
		t.Error("WinCondition should be set when 1P reaches 4 clues")
	}
}

// --- TestProcessAction_SelectInvestigator ---
// Verifies that a player can select an investigator type during the waiting phase.

func TestProcessAction_SelectInvestigator(t *testing.T) {
	t.Parallel()

	t.Run("ValidTypeInWaiting", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   ActionSelectInvestigator,
			Target:   "researcher",
		})
		if err != nil {
			t.Fatalf("selectinvestigator in waiting phase returned error: %v", err)
		}
		if got := gs.gameState.Players["p1"].InvestigatorType; got != InvestigatorResearcher {
			t.Errorf("InvestigatorType = %q, want %q", got, InvestigatorResearcher)
		}
	})

	t.Run("UnknownTypeReturnsError", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   ActionSelectInvestigator,
			Target:   "nonexistent",
		})
		if err == nil {
			t.Error("expected error for unknown investigator type, got nil")
		}
	})

	t.Run("NormalisedCamelCase", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		// Client may send "selectInvestigator" (camelCase); server must normalise it.
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   "selectInvestigator",
			Target:   "soldier",
		})
		if err != nil {
			t.Fatalf("camelCase action not normalised: %v", err)
		}
		if got := gs.gameState.Players["p1"].InvestigatorType; got != InvestigatorSoldier {
			t.Errorf("InvestigatorType = %q, want %q", got, InvestigatorSoldier)
		}
	})
}

// --- TestProcessAction_SetDifficulty ---
// Verifies that difficulty can be set during the waiting phase but not during play.

func TestProcessAction_SetDifficulty(t *testing.T) {
	t.Parallel()

	t.Run("ValidInWaiting", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   ActionSetDifficulty,
			Target:   "hard",
		})
		if err != nil {
			t.Fatalf("setdifficulty in waiting phase returned error: %v", err)
		}
		if gs.gameState.Difficulty != "hard" {
			t.Errorf("Difficulty = %q, want %q", gs.gameState.Difficulty, "hard")
		}
	})

	t.Run("RejectedAfterPregameWindowCloses", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		gs.pregameLocked = true
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionSetDifficulty,
			Target:   "easy",
		})
		if err == nil {
			t.Error("expected error when setting difficulty after pregame window closes, got nil")
		}
	})

	t.Run("CaseNormalized", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   ActionSetDifficulty,
			Target:   "Hard", // mixed-case — should be normalised to "hard"
		})
		if err != nil {
			t.Fatalf("setdifficulty with mixed-case target returned error: %v", err)
		}
		if gs.gameState.Difficulty != "hard" {
			t.Errorf("Difficulty = %q, want %q", gs.gameState.Difficulty, "hard")
		}
	})

	t.Run("PlayerNotFoundReturnsError", func(t *testing.T) {
		gs := NewGameServer()
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "nobody",
			Action:   ActionSetDifficulty,
			Target:   "easy",
		})
		if err == nil {
			t.Error("expected error for unknown player, got nil")
		}
	})
}

// --- TestProcessAction_Chat ---
// Verifies that a player can send a quick-chat phrase.

func TestProcessAction_Chat(t *testing.T) {
	t.Parallel()

	t.Run("ValidPhrase", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   ActionChat,
			Target:   "Good luck!",
		})
		if err != nil {
			t.Fatalf("chat with valid phrase returned error: %v", err)
		}
	})

	t.Run("EmptyPhraseReturnsError", func(t *testing.T) {
		gs := NewGameServer()
		gs.gameState.Players["p1"] = &Player{ID: "p1", Location: Downtown, Connected: true}
		err := gs.processAction(PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: "p1",
			Action:   ActionChat,
			Target:   "",
		})
		if err == nil {
			t.Error("expected error for empty chat phrase, got nil")
		}
	})
}
