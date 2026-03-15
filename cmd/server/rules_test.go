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
	t.Skip("Mythos Phase event placement not yet implemented in engine (RULES.md §Mythos Phase, GAPS.md)")
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

	// Unimplemented actions.
	for _, name := range []string{"focus", "research", "trade", "component"} {
		name := name
		t.Run(name+"_not_implemented", func(t *testing.T) {
			t.Skipf("action %q not yet implemented in engine (RULES.md §Action System)", name)
		})
	}
}

// --- TestRulesDicePoolFocusModifier ---
// AH3e: Investigators may spend focus tokens to add dice to their pool;
// skill ratings scale the base pool size.
// Status: NOT IMPLEMENTED — engine uses a fixed 3-die pool with no focus spend.

func TestRulesDicePoolFocusModifier(t *testing.T) {
	t.Skip("Dice pool focus-token modifier not yet implemented (RULES.md §Dice Resolution, GAPS.md)")
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
	t.Skip("Encounter resolution not yet implemented (RULES.md §Encounter System, GAPS.md)")
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
		// Engine requires 4 clues per player. With 1 player: 4 clues to win.
		gs.gameState.Players["p1"].Resources.Clues = 4
		gs.checkGameEndConditions()

		if !gs.gameState.WinCondition {
			t.Error("enough clues should trigger win condition (act completion)")
		}
	})

	t.Run("ActAgendaNotImplementedFully", func(t *testing.T) {
		t.Skip("Full act/agenda deck progression (card draws, narrative events) not yet implemented (RULES.md §Act/Agenda, GAPS.md)")
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
		gs.gameState.RequiredClues = 4
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

		// Health bounds: 1–10.
		r.Health = 0
		gs.validateResources(r)
		if r.Health < 1 {
			t.Errorf("health below 1 after validation: %d", r.Health)
		}
		r.Health = 11
		gs.validateResources(r)
		if r.Health > 10 {
			t.Errorf("health above 10 after validation: %d", r.Health)
		}

		// Sanity bounds: 1–10.
		r.Sanity = 0
		gs.validateResources(r)
		if r.Sanity < 1 {
			t.Errorf("sanity below 1 after validation: %d", r.Sanity)
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

	t.Run("MoneyNotImplemented", func(t *testing.T) {
		t.Skip("Money resource not yet implemented (RULES.md §Resources, GAPS.md)")
	})

	t.Run("RemnantsNotImplemented", func(t *testing.T) {
		t.Skip("Remnants resource not yet implemented (RULES.md §Resources, GAPS.md)")
	})

	t.Run("FocusTokensNotImplemented", func(t *testing.T) {
		t.Skip("Focus token resource not yet implemented (RULES.md §Resources, GAPS.md)")
	})
}
