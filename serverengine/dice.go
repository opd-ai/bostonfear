// Package serverengine implements dice resolution for the Arkham Horror multiplayer game server.
// This file implements the 3-sided dice mechanics (Success/Blank/Tentacle) with
// configurable dice pool sizes and focus-token rerolls (AH3e §Dice Resolution).
package serverengine

import (
	mathrand "math/rand"
)

// rollDice performs dice resolution with configurable difficulty.
// Returns: dice results, successes, tentacles.
// Each die face is equally likely: Success (1/3), Blank (1/3), Tentacle (1/3).
func (gs *GameServer) rollDice(numDice int) ([]DiceResult, int, int) {
	if numDice <= 0 {
		return []DiceResult{}, 0, 0
	}

	results := make([]DiceResult, numDice)
	successes := 0
	tentacles := 0

	for i := 0; i < numDice; i++ {
		roll := mathrand.Intn(3) // 0, 1, 2
		switch roll {
		case 0:
			results[i] = DiceSuccess
			successes++
		case 1:
			results[i] = DiceBlank
		case 2:
			results[i] = DiceTentacle
			tentacles++
		}
	}

	return results, successes, tentacles
}

// rollDicePool rolls baseDice dice plus focusSpend additional dice, deducting
// the spent focus tokens from the player. Each focus token also grants one reroll
// of a non-success die (AH3e §Dice Resolution — Focus Spend). Returns the final
// results, successes, and tentacle count.
// Caller must hold gs.mutex; player must not be nil.
func (gs *GameServer) rollDicePool(baseDice, focusSpend int, player *Player) ([]DiceResult, int, int) {
	if focusSpend < 0 {
		focusSpend = 0
	}
	// Clamp spend to available focus tokens.
	if focusSpend > player.Resources.Focus {
		focusSpend = player.Resources.Focus
	}
	player.Resources.Focus -= focusSpend

	totalDice := baseDice + focusSpend
	results, successes, tentacles := gs.rollDice(totalDice)

	// Each focus token spent grants one reroll of a non-success die.
	rerollsLeft := focusSpend
	for i := 0; i < len(results) && rerollsLeft > 0; i++ {
		if results[i] != DiceSuccess {
			// Reroll this die.
			if results[i] == DiceTentacle {
				tentacles--
			}
			roll := mathrand.Intn(3)
			switch roll {
			case 0:
				results[i] = DiceSuccess
				successes++
			case 1:
				results[i] = DiceBlank
			case 2:
				results[i] = DiceTentacle
				tentacles++
			}
			rerollsLeft--
		}
	}

	return results, successes, tentacles
}
