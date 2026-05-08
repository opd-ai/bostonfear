package serverengine

import (
	"testing"
	"time"
)

// TestGameStateValidator_ValidateGameState tests comprehensive game state validation
func TestGameStateValidator_ValidateGameState(t *testing.T) {
	validator := NewGameStateValidator()

	// Test valid game state
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
			"player2": {
				ID:       "player2",
				Location: University,
				Resources: Resources{
					Health: 8,
					Sanity: 6,
					Clues:  1,
				},
				ActionsRemaining: 0,
				Connected:        true,
			},
		},
		CurrentPlayer: "player1",
		Doom:          3,
		GamePhase:     "playing",
		TurnOrder:     []string{"player1", "player2"},
		GameStarted:   true,
	}

	errors := validator.ValidateGameState(validState)
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid state, got %d errors", len(errors))
	}

	// Test invalid doom counter
	invalidDoomState := &GameState{
		Players:     make(map[string]*Player),
		Doom:        15, // Invalid: > 12
		GamePhase:   "waiting",
		TurnOrder:   []string{},
		GameStarted: false,
	}

	errors = validator.ValidateGameState(invalidDoomState)
	foundDoomError := false
	for _, err := range errors {
		if err.Type == "DOOM_OUT_OF_BOUNDS" {
			foundDoomError = true
			break
		}
	}
	if !foundDoomError {
		t.Error("Expected DOOM_OUT_OF_BOUNDS validation error")
	}

	// Test invalid player resources
	invalidPlayerState := &GameState{
		Players: map[string]*Player{
			"player1": {
				ID:       "player1",
				Location: Downtown,
				Resources: Resources{
					Health: 15, // Invalid: > 10
					Sanity: -5, // Invalid: < 1
					Clues:  10, // Invalid: > 5
				},
				ActionsRemaining: 5, // Invalid: > 2
				Connected:        true,
			},
		},
		CurrentPlayer: "player1",
		Doom:          5,
		GamePhase:     "playing",
		TurnOrder:     []string{"player1"},
		GameStarted:   true,
	}

	errors = validator.ValidateGameState(invalidPlayerState)
	expectedErrors := []string{"INVALID_HEALTH", "INVALID_SANITY", "INVALID_CLUES", "INVALID_ACTIONS"}
	for _, expectedError := range expectedErrors {
		found := false
		for _, err := range errors {
			if err.Type == expectedError {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected validation error %s not found", expectedError)
		}
	}
}

// TestGameStateValidator_RecoverGameState tests error recovery functionality
func TestGameStateValidator_RecoverGameState(t *testing.T) {
	validator := NewGameStateValidator()

	// Create corrupted game state
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

	// Validate and recover
	errors := validator.ValidateGameState(corruptedState)
	if len(errors) == 0 {
		t.Fatal("Expected validation errors for corrupted state")
	}

	recoveredState, err := validator.RecoverGameState(corruptedState, errors)
	if err != nil {
		t.Fatalf("Recovery failed: %v", err)
	}

	// Verify recovery
	if recoveredState.Doom != 0 {
		t.Errorf("Expected doom to be recovered to 0, got %d", recoveredState.Doom)
	}

	if recoveredState.CurrentPlayer != "player1" {
		t.Errorf("Expected current player to be recovered to 'player1', got %s", recoveredState.CurrentPlayer)
	}

	player := recoveredState.Players["player1"]
	if player.Resources.Health != 10 {
		t.Errorf("Expected health to be clamped to 10, got %d", player.Resources.Health)
	}
	if player.Resources.Sanity != 0 {
		t.Errorf("Expected sanity to be clamped to 0, got %d", player.Resources.Sanity)
	}
	if player.Resources.Clues != 5 {
		t.Errorf("Expected clues to be clamped to 5, got %d", player.Resources.Clues)
	}
	if player.ActionsRemaining != 0 {
		t.Errorf("Expected actions to be clamped to 0, got %d", player.ActionsRemaining)
	}

	// Verify recovered state is healthy (should only have MEDIUM severity error for insufficient players)
	recoveryErrors := validator.ValidateGameState(recoveredState)
	criticalOrHighErrors := 0
	for _, err := range recoveryErrors {
		if err.Severity == "CRITICAL" || err.Severity == "HIGH" {
			criticalOrHighErrors++
		}
	}
	if criticalOrHighErrors > 0 {
		t.Errorf("Recovered state still has %d critical/high validation errors", criticalOrHighErrors)
	}
}

// TestGameStateValidator_CorruptionLogging tests corruption event logging
func TestGameStateValidator_CorruptionLogging(t *testing.T) {
	validator := NewGameStateValidator()

	// Create and log corruption event
	event := CorruptionEvent{
		Timestamp:   time.Now(),
		ErrorType:   "TEST_CORRUPTION",
		Description: "Test corruption event",
		PlayerID:    "player1",
		Action:      "test",
	}

	validator.logCorruption(event)

	history := validator.GetCorruptionHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 corruption event in history, got %d", len(history))
	}

	if history[0].ErrorType != "TEST_CORRUPTION" {
		t.Errorf("Expected corruption type 'TEST_CORRUPTION', got %s", history[0].ErrorType)
	}
}

// TestGameStateValidator_IsGameStateHealthy tests health check functionality
func TestGameStateValidator_IsGameStateHealthy(t *testing.T) {
	validator := NewGameStateValidator()

	// Test healthy state
	healthyState := &GameState{
		Players: map[string]*Player{
			"player1": {
				ID:       "player1",
				Location: Downtown,
				Resources: Resources{
					Health: 8,
					Sanity: 6,
					Clues:  3,
				},
				ActionsRemaining: 2,
				Connected:        true,
			},
			"player2": {
				ID:       "player2",
				Location: University,
				Resources: Resources{
					Health: 7,
					Sanity: 5,
					Clues:  2,
				},
				ActionsRemaining: 1,
				Connected:        true,
			},
		},
		CurrentPlayer: "player1",
		Doom:          5,
		GamePhase:     "playing",
		TurnOrder:     []string{"player1", "player2"},
		GameStarted:   true,
	}

	if !validator.IsGameStateHealthy(healthyState) {
		t.Error("Expected healthy state to return true")
	}

	// Test unhealthy state with critical error
	unhealthyState := &GameState{
		Players:     make(map[string]*Player),
		Doom:        20, // Critical error: > 12
		GamePhase:   "playing",
		TurnOrder:   []string{},
		GameStarted: true,
	}

	if validator.IsGameStateHealthy(unhealthyState) {
		t.Error("Expected unhealthy state to return false")
	}
}

// BenchmarkGameStateValidation benchmarks validation performance
func BenchmarkGameStateValidation(b *testing.B) {
	validator := NewGameStateValidator()

	// Create a moderately complex game state
	gameState := &GameState{
		Players: map[string]*Player{
			"player1": {ID: "player1", Location: Downtown, Resources: Resources{Health: 8, Sanity: 6, Clues: 2}, ActionsRemaining: 1, Connected: true},
			"player2": {ID: "player2", Location: University, Resources: Resources{Health: 9, Sanity: 4, Clues: 3}, ActionsRemaining: 0, Connected: true},
			"player3": {ID: "player3", Location: Rivertown, Resources: Resources{Health: 5, Sanity: 8, Clues: 1}, ActionsRemaining: 2, Connected: false},
		},
		CurrentPlayer: "player1",
		Doom:          7,
		GamePhase:     "playing",
		TurnOrder:     []string{"player1", "player2", "player3"},
		GameStarted:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateGameState(gameState)
	}
}
