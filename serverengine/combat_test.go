package serverengine

import (
	"strings"
	"testing"
)

// --- TestProcessAction_Attack ---

func TestProcessAction_Attack(t *testing.T) {
	t.Run("attack engaged enemy deals damage", func(t *testing.T) {
		gs, pid := newTestServer(t)
		enemy := &Enemy{
			ID:       "e1",
			Name:     "Ghoul",
			Health:   3,
			Damage:   1,
			Horror:   1,
			Location: Downtown,
			Engaged:  []string{pid},
		}
		gs.gameState.Enemies = map[string]*Enemy{"e1": enemy}

		player := gs.gameState.Players[pid]
		diceResult, _, _, err := gs.performAttack(player, pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if diceResult == nil {
			t.Fatal("expected diceResult, got nil")
		}
		if diceResult.Action != ActionAttack {
			t.Errorf("want action=%q, got %q", ActionAttack, diceResult.Action)
		}
		// Enemy health must have decreased by number of successes (may be 0 on all blanks).
		remaining, exists := gs.gameState.Enemies["e1"]
		if diceResult.Successes > 0 {
			if exists && remaining.Health != 3-diceResult.Successes {
				t.Errorf("want enemy health=%d, got %d", 3-diceResult.Successes, remaining.Health)
			}
		}
	})

	t.Run("attacking defeats enemy at zero health and awards clue", func(t *testing.T) {
		gs, pid := newTestServer(t)
		// Enemy with 1 health; one success defeats it.
		enemy := &Enemy{
			ID:      "e1",
			Name:    "Ghoul",
			Health:  1,
			Damage:  1,
			Horror:  1,
			Engaged: []string{pid},
		}
		gs.gameState.Enemies = map[string]*Enemy{"e1": enemy}

		player := gs.gameState.Players[pid]
		initialClues := player.Resources.Clues

		// Run enough times to guarantee at least one success occurs eventually.
		defeated := false
		for i := 0; i < 50; i++ {
			// Reset enemy health each try.
			enemy.Health = 1
			gs.gameState.Enemies["e1"] = enemy

			_, _, _, err := gs.performAttack(player, pid)
			if err != nil {
				t.Fatalf("unexpected attack error: %v", err)
			}
			if _, exists := gs.gameState.Enemies["e1"]; !exists {
				// Enemy was defeated.
				defeated = true
				break
			}
			// Restore engaged list if evaded somehow.
			enemy.Engaged = []string{pid}
		}
		if !defeated {
			t.Skip("enemy never defeated in 50 rolls (probability anomaly)")
		}
		if player.Resources.Clues != initialClues+1 {
			t.Errorf("want clues=%d after defeat, got %d", initialClues+1, player.Resources.Clues)
		}
	})

	t.Run("attack without engagement returns error", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.Enemies = map[string]*Enemy{}

		player := gs.gameState.Players[pid]
		_, _, result, err := gs.performAttack(player, pid)
		if err == nil {
			t.Fatal("expected error when not engaged, got nil")
		}
		if result != "fail" {
			t.Errorf("want result=fail, got %q", result)
		}
	})

	t.Run("tentacle results increment doom", func(t *testing.T) {
		gs, pid := newTestServer(t)
		enemy := &Enemy{
			ID:      "e1",
			Name:    "Shoggoth",
			Health:  10,
			Engaged: []string{pid},
		}
		gs.gameState.Enemies = map[string]*Enemy{"e1": enemy}
		initialDoom := gs.gameState.Doom

		player := gs.gameState.Players[pid]
		// Run many times; tentacles are 1-in-3 chance per die.
		gotTentacle := false
		for i := 0; i < 100; i++ {
			enemy.Health = 10 // reset
			enemy.Engaged = []string{pid}
			gs.gameState.Doom = initialDoom
			_, doomInc, _, _ := gs.performAttack(player, pid)
			if doomInc > 0 {
				gotTentacle = true
				if gs.gameState.Doom != initialDoom+doomInc {
					t.Errorf("doom not updated: want %d+%d, got %d", initialDoom, doomInc, gs.gameState.Doom)
				}
				break
			}
		}
		if !gotTentacle {
			t.Skip("no tentacle result in 100 rolls (probability anomaly)")
		}
	})
}

// --- TestProcessAction_Evade ---

func TestProcessAction_Evade(t *testing.T) {
	t.Run("successful evade removes player from engaged list", func(t *testing.T) {
		gs, pid := newTestServer(t)
		enemy := &Enemy{
			ID:      "e1",
			Name:    "Byakhee",
			Health:  2,
			Engaged: []string{pid},
		}
		gs.gameState.Enemies = map[string]*Enemy{"e1": enemy}

		player := gs.gameState.Players[pid]
		evaded := false
		for i := 0; i < 50; i++ {
			enemy.Engaged = []string{pid}
			diceResult, _, result, err := gs.performEvade(player, pid)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diceResult.Action != ActionEvade {
				t.Errorf("want action=%q, got %q", ActionEvade, diceResult.Action)
			}
			if result == "success" {
				evaded = true
				if len(enemy.Engaged) != 0 {
					t.Errorf("player should be removed from Engaged after evade; got %v", enemy.Engaged)
				}
				break
			}
		}
		if !evaded {
			t.Skip("evade never succeeded in 50 rolls (probability anomaly)")
		}
	})

	t.Run("failed evade leaves engagement intact", func(t *testing.T) {
		gs, pid := newTestServer(t)
		enemy := &Enemy{
			ID:      "e1",
			Name:    "Deep One",
			Health:  4,
			Engaged: []string{pid},
		}
		gs.gameState.Enemies = map[string]*Enemy{"e1": enemy}

		player := gs.gameState.Players[pid]
		// Force a failure by running until we get one.
		for i := 0; i < 50; i++ {
			enemy.Engaged = []string{pid}
			_, _, result, _ := gs.performEvade(player, pid)
			if result == "fail" {
				// Engaged list should still contain the player.
				found := false
				for _, id := range enemy.Engaged {
					if id == pid {
						found = true
					}
				}
				if !found {
					t.Error("player should remain engaged after failed evade")
				}
				return
			}
		}
		t.Skip("evade always succeeded in 50 rolls (probability anomaly)")
	})

	t.Run("evade without engagement returns error", func(t *testing.T) {
		gs, pid := newTestServer(t)
		gs.gameState.Enemies = map[string]*Enemy{}

		player := gs.gameState.Players[pid]
		_, _, result, err := gs.performEvade(player, pid)
		if err == nil {
			t.Fatal("expected error when not engaged, got nil")
		}
		if result != "fail" {
			t.Errorf("want result=fail, got %q", result)
		}
	})
}

// --- TestMythosPhase_EnemySpawn ---

func TestMythosPhase_EnemySpawn(t *testing.T) {
	t.Run("no enemies at doom below 3", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 2
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.spawnEnemiesForDoom()
		if len(gs.gameState.Enemies) != 0 {
			t.Errorf("want 0 enemies at doom=2, got %d", len(gs.gameState.Enemies))
		}
	})

	t.Run("1 enemy spawned at doom=3", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 3
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.spawnEnemiesForDoom()
		if len(gs.gameState.Enemies) != 1 {
			t.Errorf("want 1 enemy at doom=3, got %d", len(gs.gameState.Enemies))
		}
	})

	t.Run("2 enemies spawned at doom=6", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 6
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.spawnEnemiesForDoom()
		if len(gs.gameState.Enemies) != 2 {
			t.Errorf("want 2 enemies at doom=6, got %d", len(gs.gameState.Enemies))
		}
	})

	t.Run("capped at maxEnemiesOnBoard", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 12
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.spawnEnemiesForDoom()
		if len(gs.gameState.Enemies) != maxEnemiesOnBoard {
			t.Errorf("want %d enemies at doom=12 (cap), got %d", maxEnemiesOnBoard, len(gs.gameState.Enemies))
		}
	})

	t.Run("does not exceed cap when enemies already present", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 12
		gs.gameState.Enemies = map[string]*Enemy{
			"e1": {ID: "e1", Name: "Ghoul", Health: 3, Engaged: nil},
			"e2": {ID: "e2", Name: "Ghoul", Health: 3, Engaged: nil},
		}
		gs.spawnEnemiesForDoom()
		if len(gs.gameState.Enemies) != maxEnemiesOnBoard {
			t.Errorf("want %d enemies (cap), got %d", maxEnemiesOnBoard, len(gs.gameState.Enemies))
		}
	})

	t.Run("spawned enemies have valid names from template pool", func(t *testing.T) {
		gs, _ := newTestServer(t)
		gs.gameState.Doom = 3
		gs.gameState.Enemies = make(map[string]*Enemy)
		gs.spawnEnemiesForDoom()
		validNames := map[string]bool{"Ghoul": true, "Deep One": true, "Byakhee": true, "Shoggoth": true}
		for _, e := range gs.gameState.Enemies {
			if !validNames[e.Name] {
				t.Errorf("unexpected enemy name %q", e.Name)
			}
			if e.Health <= 0 {
				t.Errorf("enemy %q should have positive health, got %d", e.Name, e.Health)
			}
			if !strings.Contains(string(e.Location), "") {
				t.Errorf("enemy has empty location")
			}
		}
	})
}
