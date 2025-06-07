package main

import (
	"fmt"
	"log"
	"time"
)

// GameStateValidator provides comprehensive game state validation and recovery
type GameStateValidator struct {
	lastValidState *GameState
	corruptionLog  []CorruptionEvent
}

// CorruptionEvent records game state corruption incidents
type CorruptionEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	ErrorType   string    `json:"errorType"`
	Description string    `json:"description"`
	PlayerID    string    `json:"playerId,omitempty"`
	Action      string    `json:"action,omitempty"`
}

// NewGameStateValidator creates a new validator instance
func NewGameStateValidator() *GameStateValidator {
	return &GameStateValidator{
		corruptionLog: make([]CorruptionEvent, 0),
	}
}

// ValidateGameState performs comprehensive validation of game state integrity
func (v *GameStateValidator) ValidateGameState(gs *GameState) []ValidationError {
	var errors []ValidationError

	// Validate player count and constraints
	if len(gs.Players) < 2 && gs.GamePhase == "playing" && gs.GameStarted {
		errors = append(errors, ValidationError{
			Type:        "INSUFFICIENT_PLAYERS",
			Description: "Game in playing state with less than 2 players",
			Severity:    "MEDIUM", // Changed from HIGH to MEDIUM as this might be temporary during disconnections
		})
	}

	if len(gs.Players) > 4 {
		errors = append(errors, ValidationError{
			Type:        "EXCESS_PLAYERS",
			Description: "More than 4 players in game",
			Severity:    "HIGH",
		})
	}

	// Validate doom counter bounds
	if gs.Doom < 0 || gs.Doom > 12 {
		errors = append(errors, ValidationError{
			Type:        "DOOM_OUT_OF_BOUNDS",
			Description: fmt.Sprintf("Doom counter %d outside valid range [0-12]", gs.Doom),
			Severity:    "CRITICAL",
		})
	}

	// Validate current player exists and is in turn order
	if gs.GamePhase == "playing" {
		if gs.CurrentPlayer == "" {
			errors = append(errors, ValidationError{
				Type:        "MISSING_CURRENT_PLAYER",
				Description: "No current player set during playing phase",
				Severity:    "HIGH",
			})
		} else if _, exists := gs.Players[gs.CurrentPlayer]; !exists {
			errors = append(errors, ValidationError{
				Type:        "INVALID_CURRENT_PLAYER",
				Description: fmt.Sprintf("Current player %s not found in players list", gs.CurrentPlayer),
				Severity:    "HIGH",
			})
		}
	}

	// Validate turn order consistency
	for _, playerID := range gs.TurnOrder {
		if _, exists := gs.Players[playerID]; !exists {
			errors = append(errors, ValidationError{
				Type:        "INVALID_TURN_ORDER",
				Description: fmt.Sprintf("Player %s in turn order but not in players list", playerID),
				Severity:    "MEDIUM",
			})
		}
	}

	// Validate individual player states
	for playerID, player := range gs.Players {
		if playerErrors := v.validatePlayer(playerID, player); len(playerErrors) > 0 {
			errors = append(errors, playerErrors...)
		}
	}

	return errors
}

// validatePlayer checks individual player state integrity
func (v *GameStateValidator) validatePlayer(playerID string, player *Player) []ValidationError {
	var errors []ValidationError

	// Validate player ID consistency
	if player.ID != playerID {
		errors = append(errors, ValidationError{
			Type:        "PLAYER_ID_MISMATCH",
			Description: fmt.Sprintf("Player ID mismatch: map key %s vs player.ID %s", playerID, player.ID),
			Severity:    "HIGH",
		})
	}

	// Validate resource bounds
	if player.Resources.Health < 1 || player.Resources.Health > 10 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_HEALTH",
			Description: fmt.Sprintf("Player %s health %d outside valid range [1-10]", playerID, player.Resources.Health),
			Severity:    "MEDIUM",
		})
	}

	if player.Resources.Sanity < 1 || player.Resources.Sanity > 10 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_SANITY",
			Description: fmt.Sprintf("Player %s sanity %d outside valid range [1-10]", playerID, player.Resources.Sanity),
			Severity:    "MEDIUM",
		})
	}

	if player.Resources.Clues < 0 || player.Resources.Clues > 5 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_CLUES",
			Description: fmt.Sprintf("Player %s clues %d outside valid range [0-5]", playerID, player.Resources.Clues),
			Severity:    "MEDIUM",
		})
	}

	// Validate location is valid
	validLocations := []Location{Downtown, University, Rivertown, Northside}
	isValidLocation := false
	for _, loc := range validLocations {
		if player.Location == loc {
			isValidLocation = true
			break
		}
	}
	if !isValidLocation {
		errors = append(errors, ValidationError{
			Type:        "INVALID_LOCATION",
			Description: fmt.Sprintf("Player %s at invalid location %s", playerID, player.Location),
			Severity:    "HIGH",
		})
	}

	// Validate actions remaining
	if player.ActionsRemaining < 0 || player.ActionsRemaining > 2 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_ACTIONS",
			Description: fmt.Sprintf("Player %s has invalid actions remaining: %d", playerID, player.ActionsRemaining),
			Severity:    "MEDIUM",
		})
	}

	return errors
}

// RecoverGameState attempts to recover from identified corruption
func (v *GameStateValidator) RecoverGameState(gs *GameState, errors []ValidationError) (*GameState, error) {
	log.Printf("Attempting game state recovery for %d validation errors", len(errors))

	// Create a copy of the game state for recovery
	recoveredState := v.copyGameState(gs)

	for _, err := range errors {
		switch err.Type {
		case "DOOM_OUT_OF_BOUNDS":
			// Clamp doom counter to valid range
			if recoveredState.Doom < 0 {
				recoveredState.Doom = 0
			} else if recoveredState.Doom > 12 {
				recoveredState.Doom = 12
			}
			log.Printf("Recovered: Clamped doom counter to %d", recoveredState.Doom)

		case "MISSING_CURRENT_PLAYER":
			// Set first player in turn order as current player
			if len(recoveredState.TurnOrder) > 0 {
				recoveredState.CurrentPlayer = recoveredState.TurnOrder[0]
				log.Printf("Recovered: Set current player to %s", recoveredState.CurrentPlayer)
			}

		case "INVALID_CURRENT_PLAYER":
			// Reset to first valid player in turn order
			for _, playerID := range recoveredState.TurnOrder {
				if _, exists := recoveredState.Players[playerID]; exists {
					recoveredState.CurrentPlayer = playerID
					log.Printf("Recovered: Reset current player to %s", playerID)
					break
				}
			}

		case "INVALID_HEALTH", "INVALID_SANITY", "INVALID_CLUES":
			// Fix individual player resource bounds
			for _, player := range recoveredState.Players {
				if player.Resources.Health < 1 {
					player.Resources.Health = 1
				} else if player.Resources.Health > 10 {
					player.Resources.Health = 10
				}

				if player.Resources.Sanity < 1 {
					player.Resources.Sanity = 1
				} else if player.Resources.Sanity > 10 {
					player.Resources.Sanity = 10
				}

				if player.Resources.Clues < 0 {
					player.Resources.Clues = 0
				} else if player.Resources.Clues > 5 {
					player.Resources.Clues = 5
				}
			}
			log.Printf("Recovered: Fixed player resource bounds")

		case "INVALID_LOCATION":
			// Reset invalid player locations to Downtown
			for _, player := range recoveredState.Players {
				validLocations := []Location{Downtown, University, Rivertown, Northside}
				isValid := false
				for _, loc := range validLocations {
					if player.Location == loc {
						isValid = true
						break
					}
				}
				if !isValid {
					player.Location = Downtown
					log.Printf("Recovered: Reset player %s location to Downtown", player.ID)
				}
			}

		case "INVALID_ACTIONS":
			// Fix actions remaining bounds
			for _, player := range recoveredState.Players {
				if player.ActionsRemaining < 0 {
					player.ActionsRemaining = 0
				} else if player.ActionsRemaining > 2 {
					player.ActionsRemaining = 2
				}
			}
			log.Printf("Recovered: Fixed player actions remaining")
		}

		// Log recovery event
		v.logCorruption(CorruptionEvent{
			Timestamp:   time.Now(),
			ErrorType:   err.Type,
			Description: fmt.Sprintf("Recovered: %s", err.Description),
		})
	}

	// Store the recovered state as last valid state
	v.lastValidState = v.copyGameState(recoveredState)

	return recoveredState, nil
}

// logCorruption records corruption events for analysis
func (v *GameStateValidator) logCorruption(event CorruptionEvent) {
	v.corruptionLog = append(v.corruptionLog, event)

	// Keep only last 100 corruption events to prevent memory bloat
	if len(v.corruptionLog) > 100 {
		v.corruptionLog = v.corruptionLog[len(v.corruptionLog)-100:]
	}
}

// copyGameState creates a deep copy of game state for recovery
func (v *GameStateValidator) copyGameState(original *GameState) *GameState {
	copied := &GameState{
		Players:       make(map[string]*Player),
		CurrentPlayer: original.CurrentPlayer,
		Doom:          original.Doom,
		GamePhase:     original.GamePhase,
		TurnOrder:     make([]string, len(original.TurnOrder)),
		GameStarted:   original.GameStarted,
		WinCondition:  original.WinCondition,
		LoseCondition: original.LoseCondition,
	}

	// Deep copy turn order
	copy(copied.TurnOrder, original.TurnOrder)

	// Deep copy players
	for id, player := range original.Players {
		copied.Players[id] = &Player{
			ID:       player.ID,
			Location: player.Location,
			Resources: Resources{
				Health: player.Resources.Health,
				Sanity: player.Resources.Sanity,
				Clues:  player.Resources.Clues,
			},
			ActionsRemaining: player.ActionsRemaining,
			Connected:        player.Connected,
		}
	}

	return copied
}

// ValidationError represents a game state validation error
type ValidationError struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
}

// GetCorruptionHistory returns recent corruption events for analysis
func (v *GameStateValidator) GetCorruptionHistory() []CorruptionEvent {
	return v.corruptionLog
}

// IsGameStateHealthy performs a quick health check
func (v *GameStateValidator) IsGameStateHealthy(gs *GameState) bool {
	errors := v.ValidateGameState(gs)

	// Check for critical or high severity errors
	for _, err := range errors {
		if err.Severity == "CRITICAL" || err.Severity == "HIGH" {
			return false
		}
	}

	return true
}
