// Package serverengine implements dice resolution for the Arkham Horror multiplayer game server.
// S3 Migration: Delegates dice resolution logic to arkhamhorror/rules module.
// This file provides GameServer wrapper methods that call the module-owned implementations.
package serverengine

import (
"github.com/opd-ai/bostonfear/serverengine/arkhamhorror/rules"
)

// rollDice performs dice resolution with configurable difficulty.
// S3: Delegates to arkhamhorror/rules module for pure dice mechanics.
// Returns: dice results (converted to serverengine DiceResult type), successes, tentacles.
func (gs *GameServer) rollDice(numDice int) ([]DiceResult, int, int) {
// Call arkhamhorror module - owned dice logic
dieResults, successes, tentacles := rules.RollDice(numDice)

// Convert DieResult (rule's string type) to DiceResult (serverengine's string type)
// Both are string types with identical constant values, so this is a safe cast.
results := make([]DiceResult, len(dieResults))
for i, dr := range dieResults {
results[i] = DiceResult(dr)
}

return results, successes, tentacles
}

// rollDicePool rolls baseDice dice plus focusSpend additional dice, deducting
// the spent focus tokens from the player. Each focus token also grants one reroll
// of a non-success die (AH3e §Dice Resolution — Focus Spend). Returns the final
// results, successes, and tentacle count.
// S3: Delegates to arkhamhorror/rules module for focus token logic.
// Caller must hold gs.mutex; player must not be nil.
func (gs *GameServer) RollDicePool(baseDice, focusSpend int, player *Player) ([]DiceResult, int, int) {
// Create a wrapper that implements rules.FocusTokenSpender interface
spender := &playerFocusSpender{player: player}

// Call arkhamhorror module - owned dice logic with focus handling
// RollDicePoolWithFocus will call spender.SpendFocus to deduct tokens
dieResults, successes, tentacles := rules.RollDicePoolWithFocus(baseDice, focusSpend, spender)

// Convert DieResult to DiceResult
results := make([]DiceResult, len(dieResults))
for i, dr := range dieResults {
results[i] = DiceResult(dr)
}

return results, successes, tentacles
}

// playerFocusSpender adapts *Player to implement rules.FocusTokenSpender
type playerFocusSpender struct {
player *Player
}

func (pfs *playerFocusSpender) SpendFocus(amount int) int {
if amount < 0 {
amount = 0
}
if amount > pfs.player.Resources.Focus {
amount = pfs.player.Resources.Focus
}
pfs.player.Resources.Focus -= amount
return amount
}

func (pfs *playerFocusSpender) GetFocus() int {
return pfs.player.Resources.Focus
}
