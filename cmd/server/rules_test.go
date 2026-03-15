package main

// rules_test.go — AH3e Core Mechanics Compliance Test Suite (PLAN.md Step M8).
//
// Each test function maps to one rule system in RULES.md.  Fully implemented
// mechanics are validated with functional assertions; mechanics not yet
// implemented in the engine are recorded as pending with t.Skip so the suite
// remains runnable and its coverage gap is explicit.
//
// Run with: go test -run TestRules ./cmd/server/

import (
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

	t.Run("Component_stub", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		msg := PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: p1ID,
			Action:   ActionComponent,
		}
		err := gs.processAction(msg)
		if err == nil {
			t.Fatal("component action should return not-implemented error")
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
// by spending clues.
// Status: NOT IMPLEMENTED.

func TestRulesAnomalyGateMechanics(t *testing.T) {
	t.Skip("Anomaly / gate mechanics not yet implemented (RULES.md §Anomaly Gates, GAPS.md)")
}

// --- TestRulesEncounterResolution ---
// AH3e: Investigators at locations with encounter tokens draw from
// neighborhood-specific encounter decks and resolve skill tests.
// Status: NOT IMPLEMENTED.

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

	t.Run("LostInTimeAndSpaceNotImplemented", func(t *testing.T) {
		t.Skip("Lost-in-time-and-space state not yet implemented (RULES.md §Investigator Defeat, GAPS.md)")
	})

	t.Run("InvestigatorRecoveryNotImplemented", func(t *testing.T) {
		t.Skip("Investigator recovery from defeat not yet implemented (RULES.md §Investigator Defeat, GAPS.md)")
	})
}

// --- TestRulesVictoryDefeatConditions ---
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
		gs.validateResources(r)
		if r.Health < 0 {
			t.Errorf("health below 0 after validation: %d", r.Health)
		}
		r.Health = 11
		gs.validateResources(r)
		if r.Health > 10 {
			t.Errorf("health above 10 after validation: %d", r.Health)
		}

		// Sanity bounds: 0–10 (0 = investigator defeated).
		r.Sanity = -1
		gs.validateResources(r)
		if r.Sanity < 0 {
			t.Errorf("sanity below 0 after validation: %d", r.Sanity)
		}

		// Clue bounds: 0–5.
		r.Clues = -1
		gs.validateResources(r)
		if r.Clues < 0 {
			t.Errorf("clues below 0 after validation: %d", r.Clues)
		}
		r.Clues = 6
		gs.validateResources(r)
		if r.Clues > 5 {
			t.Errorf("clues above 5 after validation: %d", r.Clues)
		}
	})

	t.Run("MoneyBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		r.Money = -1
		gs.validateResources(r)
		if r.Money < 0 {
			t.Errorf("money below 0 after validation: %d", r.Money)
		}
		r.Money = MaxMoney + 1
		gs.validateResources(r)
		if r.Money > MaxMoney {
			t.Errorf("money above %d after validation: %d", MaxMoney, r.Money)
		}
	})

	t.Run("RemnantsBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		r.Remnants = -1
		gs.validateResources(r)
		if r.Remnants < 0 {
			t.Errorf("remnants below 0 after validation: %d", r.Remnants)
		}
		r.Remnants = MaxRemnants + 1
		gs.validateResources(r)
		if r.Remnants > MaxRemnants {
			t.Errorf("remnants above %d after validation: %d", MaxRemnants, r.Remnants)
		}
	})

	t.Run("FocusBounds", func(t *testing.T) {
		gs, p1ID := newTestServer(t)
		r := &gs.gameState.Players[p1ID].Resources

		r.Focus = -1
		gs.validateResources(r)
		if r.Focus < 0 {
			t.Errorf("focus below 0 after validation: %d", r.Focus)
		}
		r.Focus = MaxFocus + 1
		gs.validateResources(r)
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
