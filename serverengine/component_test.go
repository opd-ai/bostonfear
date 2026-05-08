package serverengine

import (
	"testing"
)

// TestProcessAction_Component verifies component ability execution for all 6 archetypes
// and enforces resource-cost rejection when the player cannot afford the ability.
func TestProcessAction_Component(t *testing.T) {
	t.Run("researcher gains clue without dice", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorResearcher
		player.Resources.Clues = 0

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		if player.Resources.Clues != 1 {
			t.Errorf("want clues=1, got %d", player.Resources.Clues)
		}
	})

	t.Run("detective draws encounter card", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorDetective
		player.Location = Downtown
		// Ensure encounter deck is populated for the test.
		gs.gameState.EncounterDecks[string(Downtown)] = []EncounterCard{
			{FlavorText: "Test card", EffectType: "clue_gain", Magnitude: 1},
		}

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		// Encounter card was consumed.
		if len(gs.gameState.EncounterDecks[string(Downtown)]) != 0 {
			t.Error("encounter deck should be empty after drawing")
		}
	})

	t.Run("occultist reduces doom at sanity cost", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorOccultist
		player.Resources.Sanity = 5
		gs.gameState.Doom = 4

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		if player.Resources.Sanity != 3 {
			t.Errorf("want sanity=3 (cost 2), got %d", player.Resources.Sanity)
		}
		if gs.gameState.Doom != 3 {
			t.Errorf("want doom=3 (reduced by 1), got %d", gs.gameState.Doom)
		}
	})

	t.Run("occultist fails when sanity insufficient", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorOccultist
		player.Resources.Sanity = 1 // costs 2

		result, err := gs.performComponent(player, pid)
		if err == nil {
			t.Fatal("expected error for insufficient sanity, got nil")
		}
		if result != "fail" {
			t.Fatalf("want result=fail, got %q", result)
		}
	})

	t.Run("soldier gains health at sanity cost", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorSoldier
		player.Resources.Health = 5
		player.Resources.Sanity = 3

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		if player.Resources.Health != 7 {
			t.Errorf("want health=7 (+2), got %d", player.Resources.Health)
		}
		if player.Resources.Sanity != 2 {
			t.Errorf("want sanity=2 (cost 1), got %d", player.Resources.Sanity)
		}
	})

	t.Run("mystic gains focus", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorMystic
		player.Resources.Focus = 1

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		if player.Resources.Focus != 2 {
			t.Errorf("want focus=2, got %d", player.Resources.Focus)
		}
	})

	t.Run("survivor gains health and sanity", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorSurvivor
		player.Resources.Health = 5
		player.Resources.Sanity = 5

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		if player.Resources.Health != 6 {
			t.Errorf("want health=6, got %d", player.Resources.Health)
		}
		if player.Resources.Sanity != 6 {
			t.Errorf("want sanity=6, got %d", player.Resources.Sanity)
		}
	})

	t.Run("unknown type falls back to survivor", func(t *testing.T) {
		gs, pid := newTestServer(t)
		player := gs.gameState.Players[pid]
		player.InvestigatorType = InvestigatorType("unknown_archetype")
		player.Resources.Health = 5
		player.Resources.Sanity = 5

		result, err := gs.performComponent(player, pid)
		if err != nil {
			t.Fatalf("unexpected error for fallback: %v", err)
		}
		if result != "success" {
			t.Fatalf("want success, got %q", result)
		}
		// Survivor gives +1 health and +1 sanity.
		if player.Resources.Health != 6 || player.Resources.Sanity != 6 {
			t.Errorf("want health=6 sanity=6, got health=%d sanity=%d",
				player.Resources.Health, player.Resources.Sanity)
		}
	})
}
