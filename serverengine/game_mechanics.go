// Package serverengine contains game-logic methods for the Arkham Horror multiplayer
// game server. This file coordinates the core mechanics: resource tracking,
// action dispatch, doom counter management, and investigator defeat/recovery.
// Dice resolution is in dice.go; action performers are in actions.go;
// the Mythos Phase and turn management are in mythos.go.
package serverengine

import (
	"fmt"

	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror/actions"
	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror/model"
	"github.com/opd-ai/bostonfear/serverengine/common/logging"
)

// ValidateResources ensures resources stay within bounds.
// S4: Uses arkhamhorror/model clamping functions to enforce resource bounds.
// Health and Sanity may reach 0 (investigator defeat); callers must call
// checkInvestigatorDefeat after this to handle that transition.
func (gs *GameServer) ValidateResources(resources *Resources) {
	// S4 Migration: Delegate to arkhamhorror module resource bounds
	resources.Health = model.ClampHealth(resources.Health)
	resources.Sanity = model.ClampSanity(resources.Sanity)
	resources.Clues = model.ClampClues(resources.Clues)
	resources.Money = model.ClampMoney(resources.Money)
	resources.Focus = model.ClampFocus(resources.Focus)
	resources.Remnants = model.ClampRemnants(resources.Remnants)
}

// CheckInvestigatorDefeat transitions a player to the defeated state when their
// Health or Sanity reaches 0. Defeated players are placed in the LostInTimeAndSpace
// state: moved to Downtown, resources reset to half max, actions zeroed.
// Caller must hold gs.mutex.
func (gs *GameServer) CheckInvestigatorDefeat(playerID string) {
	player, exists := gs.gameState.Players[playerID]
	if !exists || player.Defeated {
		return
	}
	if player.Resources.Health == 0 || player.Resources.Sanity == 0 {
		player.Defeated = true
		player.LostInTimeAndSpace = true
		player.ActionsRemaining = 0
		player.Location = Downtown
		player.Resources.Health = MaxHealth / 2
		player.Resources.Sanity = MaxSanity / 2
		logging.Info("Investigator defeated — lost in time and space (reset to Downtown)", "playerID", playerID)
	}
}

// recoverInvestigator clears the defeated and LostInTimeAndSpace flags for
// the given player, allowing them to re-enter the turn rotation normally.
// ActionsRemaining is left at 0; advanceTurn grants actions when their turn arrives.
// Caller must hold gs.mutex.
func (gs *GameServer) recoverInvestigator(playerID string) {
	player, exists := gs.gameState.Players[playerID]
	if !exists {
		return
	}
	player.Defeated = false
	player.LostInTimeAndSpace = false
	logging.Info("Investigator recovered", "playerID", playerID)
}

// dispatchAction routes the action to its specific handler and returns the results.
// S1 Migration: Delegates to arkhamhorror/actions dispatcher.
// Caller must hold gs.mutex.
func (gs *GameServer) dispatchAction(action PlayerActionMessage, player *Player) (*DiceResultMessage, int, string, error) {
	// Build callback set from GameServer methods
	callbacks := actions.CallbackSet{
		Move: func(target string) error {
			return gs.performMove(player, target)
		},
		Gather: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			dr, doom := gs.performGather(player, playerID, focusSpend)
			result := "success"
			if dr != nil && !dr.Success {
				result = "fail"
			}
			return dr, doom, result, nil
		},
		Investigate: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			diceResult, doom, actionResult := gs.performInvestigate(player, playerID, focusSpend)
			return diceResult, doom, actionResult, nil
		},
		CastWard: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			return gs.performCastWard(player, playerID, focusSpend)
		},
		Focus: func() error {
			gs.performFocus(player)
			return nil
		},
		Research: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			diceResult, doom, actionResult := gs.performResearch(player, playerID, focusSpend)
			return diceResult, doom, actionResult, nil
		},
		Trade: func(fromID, toID string) error {
			return gs.performTrade(fromID, toID)
		},
		Encounter: func(playerID string) error {
			return gs.performEncounter(player, playerID)
		},
		Component: func(playerID string) (string, error) {
			return gs.performComponent(player, playerID)
		},
		Attack: func(playerID string) (interface{}, int, string, error) {
			return gs.performAttack(player, playerID)
		},
		Evade: func(playerID string) (interface{}, int, string, error) {
			return gs.performEvade(player, playerID)
		},
		CloseGate: func(playerID string) (string, error) {
			return gs.performCloseGate(player, playerID)
		},
	}

	// Dispatch through arkhamhorror module dispatcher
	diceResult, doomIncrease, actionResult, actionErr := actions.DispatchAction(
		string(action.Action),
		action.Target,
		action.FocusSpend,
		callbacks,
		action.PlayerID,
	)

	// Convert diceResult back to *DiceResultMessage if needed
	var diceMsg *DiceResultMessage
	if diceResult != nil {
		if dm, ok := diceResult.(*DiceResultMessage); ok {
			diceMsg = dm
		}
	}

	return diceMsg, doomIncrease, actionResult, actionErr
}

// applyDifficulty configures game setup parameters from the DifficultyConfig table.
// Must be called before the game starts (during the waiting phase).
// Returns an error if the difficulty name is unrecognised.
func (gs *GameServer) applyDifficulty(difficulty string) error {
	cfg, ok := DifficultyConfig[difficulty]
	if !ok {
		return fmt.Errorf("invalid difficulty %q: must be easy, standard, or hard", difficulty)
	}
	gs.gameState.Difficulty = difficulty
	gs.gameState.Doom = cfg.InitialDoom
	return nil
}
