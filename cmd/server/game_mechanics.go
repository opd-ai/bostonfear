// Package main contains game-logic methods for the Arkham Horror multiplayer
// game server. This file coordinates the core mechanics: resource tracking,
// action dispatch, doom counter management, and investigator defeat/recovery.
// Dice resolution is in dice.go; action performers are in actions.go;
// the Mythos Phase and turn management are in mythos.go.
package main

import (
	"fmt"
	"log"
)

// validateResources ensures resources stay within bounds.
// Health and Sanity may reach 0 (investigator defeat); callers must call
// checkInvestigatorDefeat after this to handle that transition.
func (gs *GameServer) validateResources(resources *Resources) {
	type resourceField struct {
		ptr      *int
		min, max int
	}
	fields := []resourceField{
		{&resources.Health, 0, MaxHealth},
		{&resources.Sanity, 0, MaxSanity},
		{&resources.Clues, 0, MaxClues},
		{&resources.Money, 0, MaxMoney},
		{&resources.Remnants, 0, MaxRemnants},
		{&resources.Focus, 0, MaxFocus},
	}
	for _, f := range fields {
		*f.ptr = clampInt(*f.ptr, f.min, f.max)
	}
}

// checkInvestigatorDefeat transitions a player to the defeated state when their
// Health or Sanity reaches 0. Defeated players are placed in the LostInTimeAndSpace
// state: moved to Downtown, resources reset to half max, actions zeroed.
// Caller must hold gs.mutex.
func (gs *GameServer) checkInvestigatorDefeat(playerID string) {
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
		log.Printf("Investigator %s defeated — lost in time and space (reset to Downtown)",
			playerID)
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
	log.Printf("Investigator %s recovered", playerID)
}

// dispatchAction routes the action to its specific handler and returns the results.
// Caller must hold gs.mutex.
func (gs *GameServer) dispatchAction(action PlayerActionMessage, player *Player) (*DiceResultMessage, int, string, error) {
	actionResult := "success"
	var diceResult *DiceResultMessage
	var doomIncrease int
	var actionErr error

	switch action.Action {
	case ActionMove:
		actionErr = gs.performMove(player, action.Target)
	case ActionGather:
		diceResult, doomIncrease = gs.performGather(player, action.PlayerID, action.FocusSpend)
		if diceResult != nil && !diceResult.Success {
			actionResult = "fail"
		}
	case ActionInvestigate:
		diceResult, doomIncrease, actionResult = gs.performInvestigate(player, action.PlayerID, action.FocusSpend)
	case ActionCastWard:
		diceResult, doomIncrease, actionResult, actionErr = gs.performCastWard(player, action.PlayerID, action.FocusSpend)
	case ActionFocus:
		gs.performFocus(player)
	case ActionResearch:
		diceResult, doomIncrease, actionResult = gs.performResearch(player, action.PlayerID, action.FocusSpend)
	case ActionTrade:
		actionErr = gs.performTrade(action.PlayerID, action.Target)
	case ActionEncounter:
		actionErr = gs.performEncounter(player, action.PlayerID)
	case ActionComponent:
		actionResult, actionErr = gs.performComponent(player, action.PlayerID)
	case ActionAttack:
		diceResult, doomIncrease, actionResult, actionErr = gs.performAttack(player, action.PlayerID)
	case ActionEvade:
		diceResult, doomIncrease, actionResult, actionErr = gs.performEvade(player, action.PlayerID)
	case ActionCloseGate:
		actionResult, actionErr = gs.performCloseGate(player, action.PlayerID)
	}

	return diceResult, doomIncrease, actionResult, actionErr
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

