package protocol

import (
	"encoding/json"
	"testing"
)

// TestGameStateJSONRoundTrip verifies the shared game-state contract remains
// marshalable and decodable with nested protocol types intact.
func TestGameStateJSONRoundTrip(t *testing.T) {
	original := GameState{
		Players: map[string]*Player{
			"player1": {
				ID:               "player1",
				Location:         University,
				Resources:        Resources{Health: 8, Sanity: 6, Clues: 2, Money: 1},
				ActionsRemaining: 1,
				Connected:        true,
				InvestigatorType: InvestigatorResearcher,
			},
		},
		CurrentPlayer:      "player1",
		Doom:               3,
		GamePhase:          "playing",
		TurnOrder:          []string{"player1"},
		GameStarted:        true,
		RequiredClues:      4,
		Difficulty:         "standard",
		ActDeck:            []ActCard{{Title: "Act 1", ClueThreshold: 4, Effect: "Investigate"}},
		AgendaDeck:         []AgendaCard{{Title: "Agenda 1", DoomThreshold: 4, Effect: "Spread"}},
		MythosEventDeck:    []MythosEvent{{LocationID: string(Downtown), MythosEventType: "anomaly"}},
		LocationDoomTokens: map[string]int{string(Downtown): 1},
		Anomalies:          []Anomaly{{NeighbourhoodID: string(Northside), DoomTokens: 2}},
		OpenGates:          []Gate{{ID: "gate1", Location: Rivertown}},
		Enemies:            map[string]*Enemy{"enemy1": {ID: "enemy1", Name: "Ghoul", Location: Downtown}},
		ActiveEvents:       []string{"A whisper spreads."},
		EncounterDecks:     map[string][]EncounterCard{string(Downtown): {{FlavorText: "A clue", EffectType: "clue_gain", Magnitude: 1}}},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal game state: %v", err)
	}

	var decoded GameState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal game state: %v", err)
	}

	if decoded.CurrentPlayer != original.CurrentPlayer {
		t.Fatalf("current player = %q, want %q", decoded.CurrentPlayer, original.CurrentPlayer)
	}
	if decoded.Players["player1"].InvestigatorType != InvestigatorResearcher {
		t.Fatalf("investigator type = %q, want %q", decoded.Players["player1"].InvestigatorType, InvestigatorResearcher)
	}
	if decoded.OpenGates[0].Location != Rivertown {
		t.Fatalf("gate location = %q, want %q", decoded.OpenGates[0].Location, Rivertown)
	}
}

// TestActionAndDiceJSONShape verifies that core action and dice payloads preserve
// their shared enum values across JSON boundaries.
func TestActionAndDiceJSONShape(t *testing.T) {
	action := PlayerActionMessage{
		Type:       "playerAction",
		PlayerID:   "player1",
		Action:     ActionInvestigate,
		Target:     string(University),
		FocusSpend: 1,
	}
	data, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("marshal action: %v", err)
	}

	var decodedAction PlayerActionMessage
	if err := json.Unmarshal(data, &decodedAction); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	if decodedAction.Action != ActionInvestigate {
		t.Fatalf("action = %q, want %q", decodedAction.Action, ActionInvestigate)
	}

	dice := DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     "player1",
		Action:       ActionCastWard,
		Results:      []DiceResult{DiceSuccess, DiceTentacle},
		Successes:    1,
		Tentacles:    1,
		Success:      false,
		DoomIncrease: 1,
	}
	diceData, err := json.Marshal(dice)
	if err != nil {
		t.Fatalf("marshal dice: %v", err)
	}

	var decodedDice DiceResultMessage
	if err := json.Unmarshal(diceData, &decodedDice); err != nil {
		t.Fatalf("unmarshal dice: %v", err)
	}
	if decodedDice.Results[1] != DiceTentacle {
		t.Fatalf("second die = %q, want %q", decodedDice.Results[1], DiceTentacle)
	}
}
