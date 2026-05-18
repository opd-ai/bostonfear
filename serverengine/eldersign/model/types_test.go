package model

import "testing"

func TestInvestigatorState_IsZero(t *testing.T) {
	// Zero value
	var zero InvestigatorState
	if !zero.IsZero() {
		t.Error("expected zero value to report IsZero() == true")
	}

	// Initialized value
	initialized := InvestigatorState{PlayerID: "player1"}
	if initialized.IsZero() {
		t.Error("expected initialized value to report IsZero() == false")
	}
}

func TestElderSignGameState_Initialization(t *testing.T) {
	state := ElderSignGameState{
		CurrentPlayer:          "player1",
		Doom:                   0,
		AdventureDeck:          make([]AdventureCard, 0),
		DicePool:               &DicePool{AvailableDice: 6, RollsRemaining: 3},
		ElderSignTokensAwarded: 0,
		GamePhase:              "setup",
		Investigators:          make(map[string]InvestigatorState),
	}

	if state.CurrentPlayer != "player1" {
		t.Errorf("expected CurrentPlayer=player1, got %s", state.CurrentPlayer)
	}
	if state.Doom != 0 {
		t.Errorf("expected Doom=0, got %d", state.Doom)
	}
	if state.GamePhase != "setup" {
		t.Errorf("expected GamePhase=setup, got %s", state.GamePhase)
	}
	if state.DicePool.AvailableDice != 6 {
		t.Errorf("expected 6 dice, got %d", state.DicePool.AvailableDice)
	}
}

func TestAdventureCard_Structure(t *testing.T) {
	card := AdventureCard{
		ID:   "adventure1",
		Name: "Entry Hall",
		Tasks: []AdventureTask{
			{
				RequiredResults: map[string]int{"red": 2, "lore": 1},
				Description:     "Investigate the dusty shelves",
			},
		},
		Rewards: []Reward{
			{Type: "elderSign", Value: 1},
		},
		Penalties: []Penalty{
			{Type: "doom", Value: 1},
		},
	}

	if card.ID != "adventure1" {
		t.Errorf("expected ID=adventure1, got %s", card.ID)
	}
	if len(card.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(card.Tasks))
	}
	if card.Tasks[0].RequiredResults["red"] != 2 {
		t.Error("expected task to require 2 red dice")
	}
	if len(card.Rewards) != 1 {
		t.Errorf("expected 1 reward, got %d", len(card.Rewards))
	}
	if card.Rewards[0].Type != "elderSign" {
		t.Errorf("expected reward type=elderSign, got %s", card.Rewards[0].Type)
	}
}

func TestDicePool_Reset(t *testing.T) {
	pool := &DicePool{
		LockedResults:  []string{"red", "green"},
		ActiveResults:  []string{"lore"},
		AvailableDice:  6,
		RollsRemaining: 1,
	}

	pool.Reset()

	if len(pool.LockedResults) != 0 {
		t.Errorf("expected no locked results after reset, got %d", len(pool.LockedResults))
	}
	if len(pool.ActiveResults) != 0 {
		t.Errorf("expected no active results after reset, got %d", len(pool.ActiveResults))
	}
	if pool.RollsRemaining != 3 {
		t.Errorf("expected 3 rolls after reset, got %d", pool.RollsRemaining)
	}
}

func TestDicePool_IsComplete(t *testing.T) {
	pool := &DicePool{
		LockedResults:  []string{"red", "red", "lore"},
		ActiveResults:  []string{},
		AvailableDice:  6,
		RollsRemaining: 2,
	}

	// Placeholder implementation always returns false
	if pool.IsComplete() {
		t.Error("expected placeholder IsComplete to return false")
	}
}

func TestReward_Structure(t *testing.T) {
	reward := Reward{
		Type:   "stamina",
		Value:  2,
		ItemID: "",
	}

	if reward.Type != "stamina" {
		t.Errorf("expected type=stamina, got %s", reward.Type)
	}
	if reward.Value != 2 {
		t.Errorf("expected value=2, got %d", reward.Value)
	}

	itemReward := Reward{
		Type:   "item",
		ItemID: "flashlight",
	}
	if itemReward.ItemID != "flashlight" {
		t.Errorf("expected itemID=flashlight, got %s", itemReward.ItemID)
	}
}

func TestPenalty_Structure(t *testing.T) {
	penalty := Penalty{
		Type:      "doom",
		Value:     1,
		MonsterID: "",
	}

	if penalty.Type != "doom" {
		t.Errorf("expected type=doom, got %s", penalty.Type)
	}
	if penalty.Value != 1 {
		t.Errorf("expected value=1, got %d", penalty.Value)
	}

	monsterPenalty := Penalty{
		Type:      "monster",
		MonsterID: "cultist",
	}
	if monsterPenalty.MonsterID != "cultist" {
		t.Errorf("expected monsterID=cultist, got %s", monsterPenalty.MonsterID)
	}
}
