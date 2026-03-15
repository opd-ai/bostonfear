package main

import (
	"testing"
)

// TestMythosPhase_EventPlacement covers the full range of Mythos event types
// and verifies that ActiveEvents is populated and each effect is resolved correctly.
func TestMythosPhase_EventPlacement(t *testing.T) {
	t.Run("active events populated after mythos phase", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		// Seed a small deck so the draw is deterministic.
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "test fog", MythosEventType: MythosEventFogMadness},
		}
		gs.gameState.Players[pid].Resources.Sanity = 5

		gs.runMythosPhase()

		if len(gs.gameState.ActiveEvents) == 0 {
			t.Error("ActiveEvents should be non-empty after mythos phase")
		}
	})

	t.Run("fog of madness reduces all player sanity", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "fog", MythosEventType: MythosEventFogMadness},
		}
		gs.gameState.Players[pid].Resources.Sanity = 5

		gs.runMythosPhase()

		// Sanity reduced by at least 1 from the event (may also reduce from doom token).
		if gs.gameState.Players[pid].Resources.Sanity >= 5 {
			t.Errorf("fog of madness should reduce sanity; got %d (was 5)", gs.gameState.Players[pid].Resources.Sanity)
		}
	})

	t.Run("clue drought reduces all player clues", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "drought", MythosEventType: MythosEventClueDrought},
		}
		gs.gameState.Players[pid].Resources.Clues = 3

		gs.runMythosPhase()

		if gs.gameState.Players[pid].Resources.Clues >= 3 {
			t.Errorf("clue drought should reduce clues; got %d (was 3)", gs.gameState.Players[pid].Resources.Clues)
		}
	})

	t.Run("doom spread increments doom", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		gs.gameState.Doom = 2
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "spread", MythosEventType: MythosEventDoomSpread},
		}

		gs.runMythosPhase()

		// Doom should increase: at least +1 from event placement, +1 from doom_spread effect.
		if gs.gameState.Doom <= 2 {
			t.Errorf("doom spread should increase doom from 2; got %d", gs.gameState.Doom)
		}
	})

	t.Run("anomaly event spawns anomaly", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		gs.gameState.Anomalies = nil
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(University), Effect: "anomaly", MythosEventType: MythosEventAnomaly},
		}

		gs.runMythosPhase()

		if len(gs.gameState.Anomalies) == 0 {
			t.Error("anomaly event should spawn at least one anomaly")
		}
	})

	t.Run("resurgence restores health of engaged enemies", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		enemy := &Enemy{ID: "e1", Name: "Ghoul", Health: 1, Engaged: []string{pid}}
		gs.gameState.Enemies = map[string]*Enemy{"e1": enemy}
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "resurgence", MythosEventType: MythosEventResurgence},
		}

		gs.runMythosPhase()

		if enemy.Health <= 1 {
			t.Errorf("resurgence should increase engaged enemy health above 1; got %d", enemy.Health)
		}
	})

	t.Run("active events cleared at start of each mythos phase", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.gameState.LocationDoomTokens = make(map[string]int)
		gs.gameState.ActiveEvents = []string{"old event from last phase"}
		gs.gameState.MythosEventDeck = []MythosEvent{
			{LocationID: string(Downtown), Effect: "new fog", MythosEventType: MythosEventFogMadness},
		}

		gs.runMythosPhase()

		for _, e := range gs.gameState.ActiveEvents {
			if e == "old event from last phase" {
				t.Error("stale event should have been cleared before new phase")
			}
		}
	})

	t.Run("default event deck has at least 6 cards", func(t *testing.T) {
		deck := defaultMythosEventDeck()
		if len(deck) < 6 {
			t.Errorf("default event deck should have ≥6 cards; got %d", len(deck))
		}
	})
}
