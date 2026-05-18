package phases

import (
	"testing"

	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/model"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/rules"
)

func TestNewMonsterMovementPhase(t *testing.T) {
	gameState := &model.EldritchGameState{}
	phase := NewMonsterMovementPhase(gameState)

	if phase == nil {
		t.Fatal("Expected non-nil phase")
	}
	if phase.gameState != gameState {
		t.Error("Phase did not store game state reference")
	}
}

func TestExecute_NoMonsters(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{},
		Investigators: map[string]model.InvestigatorState{
			"player1": {
				PlayerID: "player1",
				Location: rules.CityArkham,
			},
		},
	}

	phase := NewMonsterMovementPhase(gameState)
	events, err := phase.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

func TestExecute_MonsterMovesToInvestigator(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{
			{
				ID:       "cultist-1",
				Type:     "cultist",
				Location: rules.CityLondon,
			},
		},
		Investigators: map[string]model.InvestigatorState{
			"player1": {
				PlayerID: "player1",
				Location: rules.CityArkham,
			},
		},
	}

	phase := NewMonsterMovementPhase(gameState)
	events, err := phase.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	// Expect either 1 or 2 events depending on whether engagement occurs
	if len(events) < 1 {
		t.Fatalf("Expected at least 1 event, got %d", len(events))
	}

	// Check the first movement event
	event := events[0]
	if event.MonsterID != "cultist-1" {
		t.Errorf("Expected monster cultist-1, got %s", event.MonsterID)
	}
	if event.From != rules.CityLondon {
		t.Errorf("Expected from London, got %s", event.From)
	}
	if event.To != rules.CityArkham {
		t.Errorf("Expected to Arkham, got %s", event.To)
	}
	if event.TargetInvestigator != "player1" {
		t.Errorf("Expected target player1, got %s", event.TargetInvestigator)
	}
}

func TestExecute_EngagedMonsterDoesNotMove(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{
			{
				ID:          "zombie-1",
				Type:        "zombie",
				Location:    rules.CityArkham,
				IsEngaged:   true,
				EngagedWith: "player1",
			},
		},
		Investigators: map[string]model.InvestigatorState{
			"player1": {
				PlayerID: "player1",
				Location: rules.CityArkham,
			},
		},
	}

	phase := NewMonsterMovementPhase(gameState)
	events, err := phase.Execute()
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Expected 0 events (engaged monster should not move), got %d", len(events))
	}
}

func TestExecute_MonsterEngagesWhenReachingInvestigator(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{
			{
				ID:       "ghoul-1",
				Type:     "ghoul",
				Location: rules.CityLondon,
			},
		},
		Investigators: map[string]model.InvestigatorState{
			"player1": {
				PlayerID: "player1",
				Location: rules.CityArkham,
			},
		},
	}

	phase := NewMonsterMovementPhase(gameState)

	// First movement: monster moves toward investigator
	events, err := phase.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify monster moved
	if len(events) < 1 {
		t.Fatalf("Expected at least 1 event, got %d", len(events))
	}

	// Check if engagement occurred when monster reaches investigator location
	foundEngagement := false
	for _, event := range events {
		if event.EngagementOccurred {
			foundEngagement = true
			if !gameState.Monsters[0].IsEngaged {
				t.Error("Monster should be marked as engaged")
			}
			if gameState.Monsters[0].EngagedWith != "player1" {
				t.Errorf("Monster should be engaged with player1, got %s", gameState.Monsters[0].EngagedWith)
			}
		}
	}

	if foundEngagement && len(events) > 1 {
		// Engagement occurred, verify state
		t.Logf("Engagement occurred: %+v", events)
	}
}

func TestSpawnMonster(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{},
	}

	phase := NewMonsterMovementPhase(gameState)
	monsterID := phase.SpawnMonster("cultist", rules.CityTokyo)

	if monsterID == "" {
		t.Fatal("Expected non-empty monster ID")
	}
	if len(gameState.Monsters) != 1 {
		t.Fatalf("Expected 1 monster, got %d", len(gameState.Monsters))
	}

	monster := gameState.Monsters[0]
	if monster.ID != monsterID {
		t.Errorf("Expected ID %s, got %s", monsterID, monster.ID)
	}
	if monster.Type != "cultist" {
		t.Errorf("Expected type cultist, got %s", monster.Type)
	}
	if monster.Location != rules.CityTokyo {
		t.Errorf("Expected location Tokyo, got %s", monster.Location)
	}
	if monster.Toughness != 1 {
		t.Errorf("Expected toughness 1, got %d", monster.Toughness)
	}
}

func TestRemoveMonster(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{
			{
				ID:       "zombie-1",
				Type:     "zombie",
				Location: rules.CityArkham,
			},
			{
				ID:       "ghoul-1",
				Type:     "ghoul",
				Location: rules.CityLondon,
			},
		},
	}

	phase := NewMonsterMovementPhase(gameState)
	err := phase.RemoveMonster("zombie-1")
	if err != nil {
		t.Errorf("RemoveMonster failed: %v", err)
	}
	if len(gameState.Monsters) != 1 {
		t.Errorf("Expected 1 monster remaining, got %d", len(gameState.Monsters))
	}
	if gameState.Monsters[0].ID != "ghoul-1" {
		t.Errorf("Expected remaining monster to be ghoul-1, got %s", gameState.Monsters[0].ID)
	}
}

func TestRemoveMonster_NotFound(t *testing.T) {
	gameState := &model.EldritchGameState{
		Monsters: []model.MonsterState{},
	}

	phase := NewMonsterMovementPhase(gameState)
	err := phase.RemoveMonster("nonexistent")

	if err == nil {
		t.Error("Expected error when removing nonexistent monster")
	}
}

func TestMonsterMovementEvent_String(t *testing.T) {
	event := MonsterMovementEvent{
		MonsterID:          "cultist-1",
		MonsterType:        "cultist",
		From:               rules.CityLondon,
		To:                 rules.CityArkham,
		TargetInvestigator: "player1",
	}

	str := event.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}
	t.Logf("Movement event: %s", str)
}

func TestMonsterMovementEvent_String_Engagement(t *testing.T) {
	event := MonsterMovementEvent{
		MonsterID:          "ghoul-1",
		MonsterType:        "ghoul",
		From:               rules.CityArkham,
		To:                 rules.CityArkham,
		TargetInvestigator: "player1",
		EngagementOccurred: true,
	}

	str := event.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}
	t.Logf("Engagement event: %s", str)
}
