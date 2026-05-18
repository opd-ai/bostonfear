package rules

import (
	"fmt"
	"testing"
)

func TestNewMonsterPool(t *testing.T) {
	mp := NewMonsterPool()
	if mp == nil {
		t.Fatal("expected monster pool")
	}
	if len(mp.Monsters) != 0 {
		t.Error("expected no initial monsters")
	}
	if len(mp.Gates) != 0 {
		t.Error("expected no initial gates")
	}
}

func TestOpenGate(t *testing.T) {
	mp := NewMonsterPool()

	gate, err := mp.OpenGate(CityArkham)
	if err != nil {
		t.Errorf("unexpected error opening gate: %v", err)
	}
	if gate == nil {
		t.Fatal("expected gate")
	}
	if gate.Location != CityArkham {
		t.Errorf("expected gate at Arkham, got %s", gate.Location)
	}
	if !gate.IsOpen {
		t.Error("expected gate to be open")
	}

	// Test duplicate gate fails
	_, err = mp.OpenGate(CityArkham)
	if err == nil {
		t.Error("expected error when opening duplicate gate")
	}
}

func TestCloseGate(t *testing.T) {
	mp := NewMonsterPool()
	mp.OpenGate(CityArkham)

	err := mp.CloseGate(CityArkham)
	if err != nil {
		t.Errorf("unexpected error closing gate: %v", err)
	}

	// Verify gate is closed
	gate, err := mp.GetGateAtCity(CityArkham)
	if err == nil {
		t.Error("expected no open gate after closing")
	}
	if gate != nil {
		t.Error("expected nil gate after closing")
	}
}

func TestSealGate(t *testing.T) {
	mp := NewMonsterPool()
	mp.OpenGate(CityArkham)

	err := mp.SealGate(CityArkham)
	if err != nil {
		t.Errorf("unexpected error sealing gate: %v", err)
	}

	// Verify gate is sealed
	for _, gate := range mp.Gates {
		if gate.Location == CityArkham {
			if !gate.IsSealed {
				t.Error("expected gate to be sealed")
			}
			if gate.IsOpen {
				t.Error("expected sealed gate to be closed")
			}
			return
		}
	}
	t.Error("gate not found after sealing")
}

func TestSpawnMonsterAtGate(t *testing.T) {
	mp := NewMonsterPool()
	gate, _ := mp.OpenGate(CityArkham)

	initialMonsterCount := len(mp.Monsters)
	initialGateMonsterCount := gate.MonsterCount

	// Gate creation spawns one monster automatically, so this tests additional spawn
	err := mp.SpawnMonsterAtGate(gate)

	// Check if error based on rules
	if mp.Rules.MaxMonstersPerGate == 1 {
		// Should fail if max is 1 and we already have 1
		if err == nil {
			t.Error("expected error when spawning beyond max")
		}
	} else {
		if err != nil {
			t.Errorf("unexpected error spawning monster: %v", err)
		}
		if len(mp.Monsters) != initialMonsterCount+1 {
			t.Errorf("expected monster count to increase")
		}
		if gate.MonsterCount != initialGateMonsterCount+1 {
			t.Error("expected gate monster count to increase")
		}
	}
}

func TestGetOpenGateCount(t *testing.T) {
	mp := NewMonsterPool()

	if mp.GetOpenGateCount() != 0 {
		t.Error("expected 0 open gates initially")
	}

	mp.OpenGate(CityArkham)
	mp.OpenGate(CityLondon)

	if mp.GetOpenGateCount() != 2 {
		t.Errorf("expected 2 open gates, got %d", mp.GetOpenGateCount())
	}

	mp.CloseGate(CityArkham)

	if mp.GetOpenGateCount() != 1 {
		t.Errorf("expected 1 open gate after closing one, got %d", mp.GetOpenGateCount())
	}
}

func TestShouldTriggerMonsterSurge(t *testing.T) {
	mp := NewMonsterPool()

	if mp.ShouldTriggerMonsterSurge() {
		t.Error("should not trigger surge with 0 gates")
	}

	// Open gates up to threshold
	for i := 0; i < mp.Rules.SurgeThreshold; i++ {
		city := City(fmt.Sprintf("city%d", i))
		mp.OpenGate(city)
	}

	if !mp.ShouldTriggerMonsterSurge() {
		t.Errorf("should trigger surge at threshold %d", mp.Rules.SurgeThreshold)
	}
}

func TestTriggerMonsterSurge(t *testing.T) {
	mp := NewMonsterPool()
	mp.Rules.MaxMonstersPerGate = 2 // Allow multiple monsters per gate

	// Test surge fails below threshold
	err := mp.TriggerMonsterSurge()
	if err == nil {
		t.Error("expected error when triggering surge below threshold")
	}

	// Open gates to reach threshold
	for i := 0; i < mp.Rules.SurgeThreshold; i++ {
		city := City(fmt.Sprintf("city%d", i))
		mp.OpenGate(city)
	}

	initialMonsterCount := len(mp.Monsters)

	err = mp.TriggerMonsterSurge()
	if err != nil {
		t.Errorf("unexpected error triggering surge: %v", err)
	}

	// Verify monsters spawned
	if len(mp.Monsters) <= initialMonsterCount {
		t.Error("expected monsters to spawn during surge")
	}
}

func TestGetMonstersAtCity(t *testing.T) {
	mp := NewMonsterPool()
	mp.OpenGate(CityArkham) // Spawns 1 monster
	mp.OpenGate(CityLondon) // Spawns 1 monster

	arkhamMonsters := mp.GetMonstersAtCity(CityArkham)
	if len(arkhamMonsters) != 1 {
		t.Errorf("expected 1 monster at Arkham, got %d", len(arkhamMonsters))
	}

	londonMonsters := mp.GetMonstersAtCity(CityLondon)
	if len(londonMonsters) != 1 {
		t.Errorf("expected 1 monster at London, got %d", len(londonMonsters))
	}

	tokyoMonsters := mp.GetMonstersAtCity(CityTokyo)
	if len(tokyoMonsters) != 0 {
		t.Errorf("expected 0 monsters at Tokyo, got %d", len(tokyoMonsters))
	}
}

func TestDefeatMonster(t *testing.T) {
	mp := NewMonsterPool()
	gate, _ := mp.OpenGate(CityArkham)

	if len(mp.Monsters) == 0 {
		t.Fatal("expected at least one monster")
	}

	monsterID := mp.Monsters[0].ID
	initialMonsterCount := len(mp.Monsters)
	initialGateCount := gate.MonsterCount

	err := mp.DefeatMonster(monsterID)
	if err != nil {
		t.Errorf("unexpected error defeating monster: %v", err)
	}

	if len(mp.Monsters) != initialMonsterCount-1 {
		t.Error("expected monster count to decrease")
	}

	// Verify gate monster count decreased
	updatedGate, err := mp.GetGateAtCity(CityArkham)
	if err != nil {
		t.Errorf("unexpected error getting gate: %v", err)
	}
	if updatedGate == nil {
		t.Fatal("expected updated gate")
	}
	if updatedGate.MonsterCount != initialGateCount-1 {
		t.Errorf("expected gate monster count %d, got %d", initialGateCount-1, updatedGate.MonsterCount)
	}
}

func TestGetGateAtCity(t *testing.T) {
	mp := NewMonsterPool()

	_, err := mp.GetGateAtCity(CityArkham)
	if err == nil {
		t.Error("expected error when no gate exists")
	}

	mp.OpenGate(CityArkham)

	gate, err := mp.GetGateAtCity(CityArkham)
	if err != nil {
		t.Errorf("unexpected error getting gate: %v", err)
	}
	if gate == nil {
		t.Fatal("expected gate")
	}
	if gate.Location != CityArkham {
		t.Errorf("expected gate at Arkham, got %s", gate.Location)
	}
}

func TestResolveCombat(t *testing.T) {
	monster := Monster{
		ID:        "test-monster",
		Name:      "Test Monster",
		Horror:    2,
		Damage:    3,
		Toughness: 2,
	}

	// Test successful combat
	result := ResolveCombat(monster, 3)
	if !result.Success {
		t.Error("expected successful combat with 3 successes")
	}
	if !result.MonsterSlain {
		t.Error("expected monster to be slain")
	}
	if result.DamageTaken != 0 {
		t.Error("expected no damage on successful combat")
	}
	if result.HorrorLoss != 2 {
		t.Errorf("expected horror loss 2, got %d", result.HorrorLoss)
	}

	// Test failed combat
	result = ResolveCombat(monster, 1)
	if result.Success {
		t.Error("expected failed combat with 1 success")
	}
	if result.MonsterSlain {
		t.Error("expected monster to survive")
	}
	if result.DamageTaken != 3 {
		t.Errorf("expected damage 3, got %d", result.DamageTaken)
	}
}

func TestDefaultMonsterSpawningRules(t *testing.T) {
	rules := DefaultMonsterSpawningRules()
	if rules.MaxMonstersPerGate <= 0 {
		t.Error("expected positive max monsters per gate")
	}
	if rules.SurgeThreshold <= 0 {
		t.Error("expected positive surge threshold")
	}
}
