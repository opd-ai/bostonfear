// Package serverengine provides action performer methods for the Arkham Horror multiplayer game server.
// This file implements the concrete handler for each of the 12 available investigator
// actions: Move, Gather, Investigate, CastWard, Focus, Research, Trade, Encounter,
// Component, Attack, Evade, and CloseGate (AH3e §Actions).
package serverengine

import (
	"fmt"
	"log"
	"strings"
)

// performMove executes the Move action: validates adjacency and updates player location.
// The caller (processAction) holds gs.mutex and is responsible for releasing it.
func (gs *GameServer) performMove(player *Player, target string) error {
	targetLocation := Location(target)
	if !gs.ValidateMovement(player.Location, targetLocation) {
		return fmt.Errorf("invalid movement from %s to %s", player.Location, targetLocation)
	}
	player.Location = targetLocation
	return nil
}

// performGather executes the Gather action: rolls dice and awards resources on success.
// On success: awards +1 Health, +1 Sanity, and +$1 Money (per AH3e §Gather Resources).
// Each Tentacle result increments the doom counter unconditionally.
// focusSpend tokens are deducted from the player and add dice plus rerolls.
// Returns the dice result message and the doom increase from any tentacle results.
func (gs *GameServer) performGather(player *Player, playerID string, focusSpend int) (*DiceResultMessage, int) {
	results, successes, tentacles := gs.RollDicePool(2, focusSpend, player)
	if successes >= 1 {
		player.Resources.Health = min(player.Resources.Health+1, MaxHealth)
		player.Resources.Sanity = min(player.Resources.Sanity+1, MaxSanity)
		player.Resources.Money = min(player.Resources.Money+1, MaxMoney)
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionGather,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= 1,
		DoomIncrease: doomIncrease,
	}, doomIncrease
}

// performInvestigate executes the Investigate action: rolls 3 dice requiring 2 successes.
// focusSpend tokens are deducted from the player and add dice plus rerolls.
// Returns the dice result, doom increase, and "success"/"fail" result string.
func (gs *GameServer) performInvestigate(player *Player, playerID string, focusSpend int) (*DiceResultMessage, int, string) {
	const requiredSuccesses = 2
	results, successes, tentacles := gs.RollDicePool(3, focusSpend, player)
	actionResult := "success"
	if successes >= requiredSuccesses {
		player.Resources.Clues = min(player.Resources.Clues+1, 5)
	} else {
		actionResult = "fail"
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionInvestigate,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= requiredSuccesses,
		DoomIncrease: doomIncrease,
	}, doomIncrease, actionResult
}

// performCastWard executes the Cast Ward action: costs 1 Sanity and rolls 3 dice requiring 3 successes.
// On success, reduces the doom counter by 2 and seals any anomaly in the player's current location.
// Returns dice result, doom increase, result string, and any error.
// focusSpend tokens are deducted from the player and add dice plus rerolls.
// The caller (processAction) holds gs.mutex and is responsible for releasing it.
func (gs *GameServer) performCastWard(player *Player, playerID string, focusSpend int) (*DiceResultMessage, int, string, error) {
	if player.Resources.Sanity <= 0 {
		return nil, 0, "", fmt.Errorf("insufficient sanity to cast ward")
	}
	player.Resources.Sanity--
	const requiredSuccesses = 3
	results, successes, tentacles := gs.RollDicePool(3, focusSpend, player)
	actionResult := "success"
	if successes >= requiredSuccesses {
		gs.GameState().Doom = max(gs.GameState().Doom-2, 0)
		// Seal any anomaly at the player's current location.
		gs.SealAnomalyAtLocation(string(player.Location))
	} else {
		actionResult = "fail"
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionCastWard,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= requiredSuccesses,
		DoomIncrease: doomIncrease,
	}, doomIncrease, actionResult, nil
}

// performFocus awards 1 Focus token to the investigator (AH3e §Focus Action).
// No dice roll is required; Focus tokens may be spent on future dice pools.
// Caller must hold gs.mutex.
func (gs *GameServer) performFocus(player *Player) {
	player.Resources.Focus = min(player.Resources.Focus+1, MaxFocus)
}

// performResearch executes the Research action: extended investigation requiring
// 2 successes from a 3-die roll, rewarding 2 Clues on success.
// Each Tentacle increments the doom counter unconditionally.
// focusSpend tokens are deducted from the player and add dice plus rerolls.
// Caller must hold gs.mutex.
func (gs *GameServer) performResearch(player *Player, playerID string, focusSpend int) (*DiceResultMessage, int, string) {
	const requiredSuccesses = 2
	results, successes, tentacles := gs.RollDicePool(3, focusSpend, player)
	actionResult := "success"
	if successes >= requiredSuccesses {
		player.Resources.Clues = min(player.Resources.Clues+2, MaxClues)
	} else {
		actionResult = "fail"
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionResearch,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= requiredSuccesses,
		DoomIncrease: doomIncrease,
	}, doomIncrease, actionResult
}

// performTrade transfers 1 Clue from the acting player to a co-located target
// (AH3e §Trade Action: investigators at the same location may exchange resources).
// Caller must hold gs.mutex.
func (gs *GameServer) performTrade(fromID, toID string) error {
	from, ok := gs.gameState.Players[fromID]
	if !ok {
		return fmt.Errorf("trade: player %s not found", fromID)
	}
	to, ok := gs.gameState.Players[toID]
	if !ok {
		return fmt.Errorf("trade: target player %s not found", toID)
	}
	if from.Location != to.Location {
		return fmt.Errorf("trade: players must be in the same location")
	}
	if from.Resources.Clues < 1 {
		return fmt.Errorf("trade: no clues to transfer")
	}
	from.Resources.Clues--
	to.Resources.Clues = min(to.Resources.Clues+1, MaxClues)
	return nil
}

// performEncounter draws the top card from the player's current location encounter
// deck and applies its effect (AH3e §Encounter Action).
// Deck is rebuilt from defaults when exhausted.
// Caller must hold gs.mutex.
func (gs *GameServer) performEncounter(player *Player, playerID string) error {
	loc := string(player.Location)
	deck := gs.gameState.EncounterDecks[loc]
	if len(deck) == 0 {
		defaults := defaultEncounterDecks()
		deck = defaults[loc]
		if len(deck) == 0 {
			return fmt.Errorf("no encounter cards for location %s", loc)
		}
	}
	card := deck[0]
	gs.gameState.EncounterDecks[loc] = deck[1:]

	switch card.EffectType {
	case "health_loss":
		player.Resources.Health = max(player.Resources.Health-card.Magnitude, 0)
		gs.ValidateResources(&player.Resources)
		gs.CheckInvestigatorDefeat(playerID)
	case "sanity_loss":
		player.Resources.Sanity = max(player.Resources.Sanity-card.Magnitude, 0)
		gs.ValidateResources(&player.Resources)
		gs.CheckInvestigatorDefeat(playerID)
	case "clue_gain":
		player.Resources.Clues = min(player.Resources.Clues+card.Magnitude, MaxClues)
	case "doom_inc":
		gs.gameState.Doom = min(gs.gameState.Doom+card.Magnitude, 12)
	}
	log.Printf("Encounter at %s for %s: %s (%s %+d)", loc, playerID, card.FlavorText, card.EffectType, card.Magnitude)
	return nil
}

// performComponent executes the investigator's unique component ability.
// The ability is determined by the player's InvestigatorType; an unrecognised type
// falls back to the Survivor ability. Resource costs are validated before any effect
// is applied — if the player cannot pay, an error is returned and no state changes.
func (gs *GameServer) performComponent(player *Player, playerID string) (string, error) {
	invType := player.InvestigatorType
	ability, ok := DefaultInvestigatorAbilities[invType]
	if !ok {
		// Safe default: Survivor ability requires no cost and is always usable.
		invType = InvestigatorSurvivor
		ability = DefaultInvestigatorAbilities[invType]
	}

	// Validate resource costs before applying any effect.
	if player.Resources.Sanity < ability.SanityCost {
		return "fail", fmt.Errorf("component ability %q requires %d sanity (have %d)",
			ability.Name, ability.SanityCost, player.Resources.Sanity)
	}
	if player.Resources.Health < ability.HealthCost {
		return "fail", fmt.Errorf("component ability %q requires %d health (have %d)",
			ability.Name, ability.HealthCost, player.Resources.Health)
	}

	// Deduct costs.
	player.Resources.Sanity -= ability.SanityCost
	player.Resources.Health -= ability.HealthCost

	// Apply gains.
	player.Resources.Clues = min(player.Resources.Clues+ability.ClueGain, MaxClues)
	player.Resources.Health = min(player.Resources.Health+ability.HealthGain, MaxHealth)
	player.Resources.Sanity = min(player.Resources.Sanity+ability.SanityGain, MaxSanity)
	player.Resources.Focus = min(player.Resources.Focus+ability.FocusGain, MaxFocus)

	// Doom reduction (Occultist dark bargain).
	if ability.DoomReduct > 0 {
		gs.gameState.Doom = max(gs.gameState.Doom-ability.DoomReduct, 0)
	}

	// Free encounter draw (Detective street contacts).
	if ability.DrawEncounter {
		if err := gs.performEncounter(player, playerID); err != nil {
			log.Printf("Component encounter draw failed for %s: %v", playerID, err)
		}
	}

	gs.ValidateResources(&player.Resources)
	gs.CheckInvestigatorDefeat(playerID)
	log.Printf("Component action by %s (%s): %s", playerID, invType, ability.Name)
	return "success", nil
}

// performAttack executes the Attack action against the first enemy engaged with the player.
// Combat dice pool equals 2 (base) + focus spent. Each Success deals 1 damage; an enemy
// defeated at 0 health is removed and the investigator gains 1 Clue. Each Tentacle result
// increments the doom counter. Returns an error if the player is not engaged with any enemy.
// Caller must hold gs.mutex.
func (gs *GameServer) performAttack(player *Player, playerID string) (*DiceResultMessage, int, string, error) {
	engaged := gs.FindEngagedEnemy(playerID)
	if engaged == nil {
		return nil, 0, "fail", fmt.Errorf("player %s is not engaged with any enemy", playerID)
	}

	results, successes, tentacles := gs.RollDicePool(2, 0, player)
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
		gs.gameState.Doom = min(gs.gameState.Doom+tentacles, 12)
	}

	engaged.Health -= successes
	actionResult := "success"
	if successes == 0 {
		actionResult = "fail"
	}

	if engaged.Health <= 0 {
		// Enemy defeated — remove it and award a clue.
		delete(gs.gameState.Enemies, engaged.ID)
		player.Resources.Clues = min(player.Resources.Clues+1, MaxClues)
		log.Printf("Enemy %s defeated by %s; clue awarded", engaged.Name, playerID)
	}

	diceResult := &DiceResultMessage{
		Type:      "diceResult",
		PlayerID:  playerID,
		Results:   results,
		Successes: successes,
		Success:   successes > 0,
		Action:    ActionAttack,
	}
	log.Printf("Attack by %s on %s: %d successes, doom +%d", playerID, engaged.Name, successes, doomIncrease)
	return diceResult, doomIncrease, actionResult, nil
}

// performEvade executes the Evade action against the first engaged enemy.
// Agility dice pool equals 2 (base) + focus spent. On ≥1 Success the player is
// removed from the enemy's Engaged list. Tentacle results still increment doom.
// Returns an error when the player is not engaged with any enemy.
// Caller must hold gs.mutex.
func (gs *GameServer) performEvade(player *Player, playerID string) (*DiceResultMessage, int, string, error) {
	engaged := gs.FindEngagedEnemy(playerID)
	if engaged == nil {
		return nil, 0, "fail", fmt.Errorf("player %s is not engaged with any enemy", playerID)
	}

	results, successes, tentacles := gs.RollDicePool(2, 0, player)
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
		gs.gameState.Doom = min(gs.gameState.Doom+tentacles, 12)
	}

	actionResult := "fail"
	if successes >= 1 {
		actionResult = "success"
		// Remove the player from this enemy's Engaged list.
		updated := make([]string, 0, len(engaged.Engaged))
		for _, id := range engaged.Engaged {
			if id != playerID {
				updated = append(updated, id)
			}
		}
		engaged.Engaged = updated
	}

	diceResult := &DiceResultMessage{
		Type:      "diceResult",
		PlayerID:  playerID,
		Results:   results,
		Successes: successes,
		Success:   successes >= 1,
		Action:    ActionEvade,
	}
	log.Printf("Evade by %s from %s: %d successes, doom +%d", playerID, engaged.Name, successes, doomIncrease)
	return diceResult, doomIncrease, actionResult, nil
}

// FindEngagedEnemy returns the first enemy that has playerID in its Engaged list,
// or nil when the player is not engaged with any enemy.
// Caller must hold gs.mutex.
func (gs *GameServer) FindEngagedEnemy(playerID string) *Enemy {
	for _, e := range gs.gameState.Enemies {
		for _, id := range e.Engaged {
			if id == playerID {
				return e
			}
		}
	}
	return nil
}

// performCloseGate executes the CloseGate action: the investigator spends 2 Clues
// to seal a Gate at their current location. Doom decreases by 1 on success.
// Returns an error if the player lacks clues or there is no gate at their location.
// Caller must hold gs.mutex.
func (gs *GameServer) performCloseGate(player *Player, playerID string) (string, error) {
	const clueCost = 2
	if player.Resources.Clues < clueCost {
		return "fail", fmt.Errorf("closing a gate requires %d clues (have %d)", clueCost, player.Resources.Clues)
	}
	loc := player.Location
	for i, g := range gs.gameState.OpenGates {
		if g.Location == loc {
			gs.gameState.OpenGates = append(gs.gameState.OpenGates[:i], gs.gameState.OpenGates[i+1:]...)
			player.Resources.Clues -= clueCost
			gs.gameState.Doom = max(gs.gameState.Doom-1, 0)
			log.Printf("Gate closed at %s by %s (doom=%d)", loc, playerID, gs.gameState.Doom)
			return "success", nil
		}
	}
	return "fail", fmt.Errorf("no open gate at %s", loc)
}

// pregameActionsAllowedLocked reports whether lobby-time pregame actions
// (investigator selection and difficulty setting) are currently allowed.
// Caller must hold gs.mutex.
func (gs *GameServer) pregameActionsAllowedLocked() bool {
	if gs.gameState.GamePhase == "waiting" {
		return true
	}
	if gs.gameState.GamePhase != "playing" {
		return false
	}
	return !gs.pregameLocked
}

// performSelectInvestigator sets the player's InvestigatorType during the pregame window.
// target must be one of the six valid archetype strings (e.g. "researcher", "soldier").
// Returns an error if pregame setup has closed or the archetype is unrecognised.
// Caller must hold gs.mutex.
func (gs *GameServer) performSelectInvestigator(player *Player, playerID, target string) error {
	if !gs.pregameActionsAllowedLocked() {
		return fmt.Errorf("investigator selection is only allowed before the first turn action")
	}
	invType := InvestigatorType(strings.ToLower(target))
	if _, ok := DefaultInvestigatorAbilities[invType]; !ok {
		return fmt.Errorf("unknown investigator type %q", target)
	}
	player.InvestigatorType = invType
	log.Printf("Player %s selected investigator type: %s", playerID, invType)
	return nil
}

// performSetDifficulty applies a difficulty preset during the pregame window.
// target must be "easy", "standard", or "hard".
// Returns an error if pregame setup has closed or the name is unrecognised.
// Caller must hold gs.mutex.
func (gs *GameServer) performSetDifficulty(target string) error {
	if !gs.pregameActionsAllowedLocked() {
		return fmt.Errorf("difficulty can only be set before the first turn action")
	}
	return gs.applyDifficulty(strings.ToLower(strings.TrimSpace(target)))
}

// performChat broadcasts a quick-chat phrase from the player to all connected clients.
// phrase must be non-empty. The message is recorded in the game update event log.
// Caller must hold gs.mutex.
func (gs *GameServer) performChat(playerID, phrase string) error {
	if phrase == "" {
		return fmt.Errorf("chat phrase must not be empty")
	}
	log.Printf("Chat from %s: %s", playerID, phrase)
	return nil
}
