// Package rules implements dice resolution and doom coupling for Arkham Horror.
// S3 Migration: Dice resolution and doom coupling - arkhamhorror module ownership.
// This file defines the dice rolling mechanics with focus token handling.
package rules

import (
	mathrand "math/rand"
)

// DieResult represents a single die outcome
type DieResult string

// Die face constants
const (
	DiceSuccess  DieResult = "success"
	DiceBlank    DieResult = "blank"
	DiceTentacle DieResult = "tentacle"
)

// RollDice performs core dice resolution with configurable pool size.
// Returns: dice results, successes, tentacles.
// Each die face is equally likely: Success (1/3), Blank (1/3), Tentacle (1/3).
//
// S3 Implementation: Module-owned dice mechanics, preserving exact same randomization
// and result distribution as serverengine/dice.go
func RollDice(numDice int) ([]DieResult, int, int) {
	if numDice <= 0 {
		return []DieResult{}, 0, 0
	}

	results := make([]DieResult, numDice)
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

// FocusTokenSpender defines an interface for objects that can spend focus tokens
type FocusTokenSpender interface {
	SpendFocus(amount int) int // Returns amount actually spent (clamped to available)
	GetFocus() int             // Returns current focus count
}

// RollDicePoolWithFocus rolls baseDice dice plus focusSpend additional dice.
// This function calls spender.SpendFocus to deduct the focused tokens from the player.
// Each focus token grants one reroll of a non-success die.
// Returns final results, successes, and tentacle count.
//
// S3 Implementation: Focus spend logic moved to arkhamhorror module while maintaining
// exact parity with serverengine implementation.
func RollDicePoolWithFocus(baseDice, focusSpend int, spender FocusTokenSpender) ([]DieResult, int, int) {
	if focusSpend < 0 {
		focusSpend = 0
	}
	// Clamp spend to available focus tokens
	availableFocus := spender.GetFocus()
	if focusSpend > availableFocus {
		focusSpend = availableFocus
	}
	// Deduct the focus tokens from the spender
	spender.SpendFocus(focusSpend)

	totalDice := baseDice + focusSpend
	results, successes, tentacles := RollDice(totalDice)

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

// DoomCouplingRule enforces doom counter invariants:
// - Each tentacle result increments doom by 1
// - Doom is bounded [0, 12]
//
// S3: These rules are arkhamhorror-owned, ensuring any module can implement
// the same doom mechanics consistently.
func ApplyTentacleToDoom(currentDoom, tentacles int) int {
	newDoom := currentDoom + tentacles
	if newDoom > 12 {
		newDoom = 12
	}
	if newDoom < 0 {
		newDoom = 0
	}
	return newDoom
}
