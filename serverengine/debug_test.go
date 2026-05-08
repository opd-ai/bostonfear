package serverengine

import (
	"fmt"
	"testing"
)

// TestGameStateValidator_Debug helps debug validation issues
func TestGameStateValidator_Debug(t *testing.T) {
	validator := NewGameStateValidator()

	// Test the supposedly valid state
	validState := &GameState{
		Players: map[string]*Player{
			"player1": {
				ID:       "player1",
				Location: Downtown,
				Resources: Resources{
					Health: 5,
					Sanity: 7,
					Clues:  2,
				},
				ActionsRemaining: 1,
				Connected:        true,
			},
		},
		CurrentPlayer: "player1",
		Doom:          3,
		GamePhase:     "playing",
		TurnOrder:     []string{"player1"},
		GameStarted:   true,
	}

	errors := validator.ValidateGameState(validState)
	fmt.Printf("Validation errors for 'valid' state: %d\n", len(errors))
	for _, err := range errors {
		fmt.Printf("  - Type: %s, Description: %s, Severity: %s\n", err.Type, err.Description, err.Severity)
	}

	// Test the corrupted state to see all errors
	corruptedState := &GameState{
		Players: map[string]*Player{
			"player1": {
				ID:       "player1",
				Location: Downtown,
				Resources: Resources{
					Health: 15, // Invalid: > 10
					Sanity: -2, // Invalid: < 1
					Clues:  8,  // Invalid: > 5
				},
				ActionsRemaining: -1, // Invalid: < 0
				Connected:        true,
			},
		},
		CurrentPlayer: "", // Invalid: missing current player
		Doom:          -5, // Invalid: < 0
		GamePhase:     "playing",
		TurnOrder:     []string{"player1"},
		GameStarted:   true,
	}

	errors = validator.ValidateGameState(corruptedState)
	fmt.Printf("\nValidation errors for corrupted state: %d\n", len(errors))
	for _, err := range errors {
		fmt.Printf("  - Type: %s, Description: %s, Severity: %s\n", err.Type, err.Description, err.Severity)
	}

	// Test recovery
	recoveredState, recoveryErr := validator.RecoverGameState(corruptedState, errors)
	if recoveryErr != nil {
		t.Fatalf("Recovery failed: %v", recoveryErr)
	}

	errors = validator.ValidateGameState(recoveredState)
	fmt.Printf("\nValidation errors after recovery: %d\n", len(errors))
	for _, err := range errors {
		fmt.Printf("  - Type: %s, Description: %s, Severity: %s\n", err.Type, err.Description, err.Severity)
	}
}
