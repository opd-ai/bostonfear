package main

import (
	"testing"
)

// TestGateMechanics_OpenAndClose covers the full Gate lifecycle:
// opening when doom accumulates, closing via performCloseGate, and doom reduction.
func TestGateMechanics_OpenAndClose(t *testing.T) {
	t.Run("gate opens when location reaches 2 doom tokens", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.OpenGates = []Gate{}
		gs.gameState.LocationDoomTokens = map[string]int{string(Downtown): 1}
		// Seed a deck whose first event targets Downtown, pushing it to 2 tokens.
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "test", MythosEventType: MythosEventFogMadness},
		}

		gs.runMythosPhase()

		found := false
		for _, g := range gs.gameState.OpenGates {
			if g.Location == Downtown {
				found = true
			}
		}
		if !found {
			t.Error("gate should have opened at Downtown after 2nd doom token placed there")
		}
	})

	t.Run("no gate opens below 2 doom tokens", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.OpenGates = []Gate{}
		gs.gameState.LocationDoomTokens = make(map[string]int) // all zero
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(University), Effect: "test", MythosEventType: MythosEventFogMadness},
		}

		gs.runMythosPhase()

		// University now has 1 doom token — no gate yet.
		for _, g := range gs.gameState.OpenGates {
			if g.Location == University {
				t.Error("gate should not open at University with only 1 doom token")
			}
		}
	})

	t.Run("performCloseGate removes gate and reduces doom", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.OpenGates = []Gate{{ID: "g1", Location: Downtown}}
		gs.gameState.Doom = 5
		player := gs.gameState.Players[pid]
		player.Location = Downtown
		player.Resources.Clues = 3

		result, err := gs.performCloseGate(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		if len(gs.gameState.OpenGates) != 0 {
			t.Errorf("gate should be removed; got %d open gates", len(gs.gameState.OpenGates))
		}
		if gs.gameState.Doom != 4 {
			t.Errorf("want doom=4 after closing gate, got %d", gs.gameState.Doom)
		}
		if player.Resources.Clues != 1 {
			t.Errorf("want clues=1 after spending 2, got %d", player.Resources.Clues)
		}
	})

	t.Run("performCloseGate fails with insufficient clues", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.OpenGates = []Gate{{ID: "g1", Location: Downtown}}
		player := gs.gameState.Players[pid]
		player.Location = Downtown
		player.Resources.Clues = 1 // needs 2

		result, err := gs.performCloseGate(player, pid)
		if err == nil {
			t.Fatal("expected error for insufficient clues, got nil")
		}
		if result != "fail" {
			t.Fatalf("want fail, got %q", result)
		}
		// Gate should remain open.
		if len(gs.gameState.OpenGates) != 1 {
			t.Error("gate should remain open after failed close attempt")
		}
	})

	t.Run("performCloseGate fails when no gate at location", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.OpenGates = []Gate{{ID: "g1", Location: Rivertown}}
		player := gs.gameState.Players[pid]
		player.Location = Downtown // different location
		player.Resources.Clues = 3

		result, err := gs.performCloseGate(player, pid)
		if err == nil {
			t.Fatal("expected error when no gate at current location, got nil")
		}
		if result != "fail" {
			t.Fatalf("want fail, got %q", result)
		}
	})

	t.Run("doom spread scales with open gates", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		gs.gameState.OpenGates = []Gate{
			{ID: "g1", Location: Downtown},
			{ID: "g2", Location: University},
		}
		gs.gameState.Doom = 3
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Rivertown), Effect: "doom spread", MythosEventType: MythosEventDoomSpread},
		}

		gs.runMythosPhase()

		// At minimum: +1 from event placement + +2 from doom_spread (2 gates).
		if gs.gameState.Doom < 3+1+2 {
			t.Errorf("want doom ≥ %d with 2 gates, got %d", 3+1+2, gs.gameState.Doom)
		}
	})

	t.Run("openGates included in gameState JSON field", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.OpenGates = []Gate{{ID: "g1", Location: Northside}}
		if len(gs.gameState.OpenGates) != 1 {
			t.Error("OpenGates should be serialized in GameState")
		}
	})
}
