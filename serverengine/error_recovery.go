package serverengine

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

	errors = append(errors, validatePlayerCount(gs)...)
	errors = append(errors, validateDoomBounds(gs)...)
	errors = append(errors, validateCurrentPlayer(gs)...)
	errors = append(errors, validateTurnOrderConsistency(gs)...)

	// Validate individual player states
	for playerID, player := range gs.Players {
		if playerErrors := v.validatePlayer(playerID, player); len(playerErrors) > 0 {
			errors = append(errors, playerErrors...)
		}
	}

	return errors
}

// validatePlayerCount checks that the number of players is within game bounds.
func validatePlayerCount(gs *GameState) []ValidationError {
	var errors []ValidationError
	if len(gs.Players) < MinPlayers && gs.GamePhase == "playing" && gs.GameStarted {
		errors = append(errors, ValidationError{
			Type:        "INSUFFICIENT_PLAYERS",
			Description: fmt.Sprintf("Game in playing state with less than %d player(s)", MinPlayers),
			Severity:    "MEDIUM",
		})
	}
	if len(gs.Players) > MaxPlayers {
		errors = append(errors, ValidationError{
			Type:        "EXCESS_PLAYERS",
			Description: fmt.Sprintf("More than %d players in game", MaxPlayers),
			Severity:    "HIGH",
		})
	}
	return errors
}

// validateDoomBounds checks that the doom counter is within [0, 12].
func validateDoomBounds(gs *GameState) []ValidationError {
	if gs.Doom < 0 || gs.Doom > 12 {
		return []ValidationError{{
			Type:        "DOOM_OUT_OF_BOUNDS",
			Description: fmt.Sprintf("Doom counter %d outside valid range [0-12]", gs.Doom),
			Severity:    "CRITICAL",
		}}
	}
	return nil
}

// validateCurrentPlayer ensures a valid current player is set during the playing phase.
func validateCurrentPlayer(gs *GameState) []ValidationError {
	if gs.GamePhase != "playing" {
		return nil
	}
	if gs.CurrentPlayer == "" {
		return []ValidationError{{
			Type:        "MISSING_CURRENT_PLAYER",
			Description: "No current player set during playing phase",
			Severity:    "HIGH",
		}}
	}
	if _, exists := gs.Players[gs.CurrentPlayer]; !exists {
		return []ValidationError{{
			Type:        "INVALID_CURRENT_PLAYER",
			Description: fmt.Sprintf("Current player %s not found in players list", gs.CurrentPlayer),
			Severity:    "HIGH",
		}}
	}
	return nil
}

// validateTurnOrderConsistency checks that every player in TurnOrder also exists in Players.
func validateTurnOrderConsistency(gs *GameState) []ValidationError {
	var errors []ValidationError
	for _, playerID := range gs.TurnOrder {
		if _, exists := gs.Players[playerID]; !exists {
			errors = append(errors, ValidationError{
				Type:        "INVALID_TURN_ORDER",
				Description: fmt.Sprintf("Player %s in turn order but not in players list", playerID),
				Severity:    "MEDIUM",
			})
		}
	}
	return errors
}

// validatePlayer checks individual player state integrity
func (v *GameStateValidator) validatePlayer(playerID string, player *Player) []ValidationError {
	var errors []ValidationError

	if player.ID != playerID {
		errors = append(errors, ValidationError{
			Type:        "PLAYER_ID_MISMATCH",
			Description: fmt.Sprintf("Player ID mismatch: map key %s vs player.ID %s", playerID, player.ID),
			Severity:    "HIGH",
		})
	}

	errors = append(errors, validatePlayerResources(playerID, &player.Resources)...)
	errors = append(errors, validatePlayerLocation(playerID, player.Location)...)

	if player.ActionsRemaining < 0 || player.ActionsRemaining > 2 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_ACTIONS",
			Description: fmt.Sprintf("Player %s has invalid actions remaining: %d", playerID, player.ActionsRemaining),
			Severity:    "MEDIUM",
		})
	}

	return errors
}

// validatePlayerResources checks that a player's Health, Sanity, and Clues are within bounds.
// Health and Sanity may be 0 when an investigator has been defeated.
func validatePlayerResources(playerID string, r *Resources) []ValidationError {
	var errors []ValidationError
	if r.Health < 0 || r.Health > 10 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_HEALTH",
			Description: fmt.Sprintf("Player %s health %d outside valid range [0-10]", playerID, r.Health),
			Severity:    "MEDIUM",
		})
	}
	if r.Sanity < 0 || r.Sanity > 10 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_SANITY",
			Description: fmt.Sprintf("Player %s sanity %d outside valid range [0-10]", playerID, r.Sanity),
			Severity:    "MEDIUM",
		})
	}
	if r.Clues < 0 || r.Clues > 5 {
		errors = append(errors, ValidationError{
			Type:        "INVALID_CLUES",
			Description: fmt.Sprintf("Player %s clues %d outside valid range [0-5]", playerID, r.Clues),
			Severity:    "MEDIUM",
		})
	}
	return errors
}

// validatePlayerLocation verifies that the player's location is one of the known neighborhoods.
func validatePlayerLocation(playerID string, loc Location) []ValidationError {
	for _, valid := range []Location{Downtown, University, Rivertown, Northside} {
		if loc == valid {
			return nil
		}
	}
	return []ValidationError{{
		Type:        "INVALID_LOCATION",
		Description: fmt.Sprintf("Player %s at invalid location %s", playerID, loc),
		Severity:    "HIGH",
	}}
}

// RecoverGameState attempts to recover from identified corruption
func (v *GameStateValidator) RecoverGameState(gs *GameState, errors []ValidationError) (*GameState, error) {
	log.Printf("Attempting game state recovery for %d validation errors", len(errors))

	recoveredState := v.copyGameState(gs)

	for _, err := range errors {
		switch err.Type {
		case "DOOM_OUT_OF_BOUNDS":
			recoverDoomBounds(recoveredState)
		case "MISSING_CURRENT_PLAYER":
			recoverMissingCurrentPlayer(recoveredState)
		case "INVALID_CURRENT_PLAYER":
			recoverInvalidCurrentPlayer(recoveredState)
		case "INVALID_HEALTH", "INVALID_SANITY", "INVALID_CLUES":
			recoverPlayerResources(recoveredState)
		case "INVALID_LOCATION":
			recoverPlayerLocations(recoveredState)
		case "INVALID_ACTIONS":
			recoverPlayerActions(recoveredState)
		}

		v.logCorruption(CorruptionEvent{
			Timestamp:   time.Now(),
			ErrorType:   err.Type,
			Description: fmt.Sprintf("Recovered: %s", err.Description),
		})
	}

	v.lastValidState = v.copyGameState(recoveredState)
	return recoveredState, nil
}

// recoverDoomBounds clamps the doom counter to the valid range [0, 12].
func recoverDoomBounds(gs *GameState) {
	if gs.Doom < 0 {
		gs.Doom = 0
	} else if gs.Doom > 12 {
		gs.Doom = 12
	}
	log.Printf("Recovered: Clamped doom counter to %d", gs.Doom)
}

// recoverMissingCurrentPlayer sets the first player in turn order as the current player.
func recoverMissingCurrentPlayer(gs *GameState) {
	if len(gs.TurnOrder) > 0 {
		gs.CurrentPlayer = gs.TurnOrder[0]
		log.Printf("Recovered: Set current player to %s", gs.CurrentPlayer)
	}
}

// recoverInvalidCurrentPlayer resets CurrentPlayer to the first valid player in TurnOrder.
func recoverInvalidCurrentPlayer(gs *GameState) {
	for _, playerID := range gs.TurnOrder {
		if _, exists := gs.Players[playerID]; exists {
			gs.CurrentPlayer = playerID
			log.Printf("Recovered: Reset current player to %s", playerID)
			return
		}
	}
}

// recoverPlayerResources clamps Health, Sanity, and Clues for every player to valid bounds.
func recoverPlayerResources(gs *GameState) {
	for _, player := range gs.Players {
		clampResources(&player.Resources)
	}
	log.Printf("Recovered: Fixed player resource bounds")
}

// clampResources ensures Health, Sanity, and Clues stay within their valid bounds.
// Health and Sanity may reach 0 (investigator defeated); negative values are clamped to 0.
func clampResources(r *Resources) {
	r.Health = clampInt(r.Health, 0, MaxHealth)
	r.Sanity = clampInt(r.Sanity, 0, MaxSanity)
	r.Clues = clampInt(r.Clues, 0, MaxClues)
	r.Money = clampInt(r.Money, 0, MaxMoney)
	r.Remnants = clampInt(r.Remnants, 0, MaxRemnants)
	r.Focus = clampInt(r.Focus, 0, MaxFocus)
}

// clampInt returns v clamped to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// recoverPlayerLocations resets any player at an unknown location back to Downtown.
func recoverPlayerLocations(gs *GameState) {
	for _, player := range gs.Players {
		if len(validatePlayerLocation(player.ID, player.Location)) > 0 {
			player.Location = Downtown
			log.Printf("Recovered: Reset player %s location to Downtown", player.ID)
		}
	}
}

// recoverPlayerActions clamps each player's ActionsRemaining to [0, 2].
func recoverPlayerActions(gs *GameState) {
	for _, player := range gs.Players {
		if player.ActionsRemaining < 0 {
			player.ActionsRemaining = 0
		} else if player.ActionsRemaining > 2 {
			player.ActionsRemaining = 2
		}
	}
	log.Printf("Recovered: Fixed player actions remaining")
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
				Health:   player.Resources.Health,
				Sanity:   player.Resources.Sanity,
				Clues:    player.Resources.Clues,
				Money:    player.Resources.Money,
				Remnants: player.Resources.Remnants,
				Focus:    player.Resources.Focus,
			},
			ActionsRemaining:   player.ActionsRemaining,
			Connected:          player.Connected,
			InvestigatorType:   player.InvestigatorType,
			ReconnectToken:     player.ReconnectToken,
			Defeated:           player.Defeated,
			LostInTimeAndSpace: player.LostInTimeAndSpace,
			DisconnectedAt:     player.DisconnectedAt,
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
