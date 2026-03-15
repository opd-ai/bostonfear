// Package main contains game-logic methods for the Arkham Horror multiplayer
// game server. This file implements all five core game mechanics: resource
// tracking, action dispatch, dice resolution, doom counter management, and
// the Mythos Phase.
package main

import (
	"fmt"
	"log"
	mathrand "math/rand"
	"sync/atomic"
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

// rollDice performs dice resolution with configurable difficulty
// Returns: dice results, successes, tentacles
// Moved from: main.go
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
	case ActionComponent:
		actionErr = gs.performComponent(player, action.PlayerID)
	case ActionEncounter:
		actionErr = gs.performEncounter(player, action.PlayerID)
	}

	return diceResult, doomIncrease, actionResult, actionErr
}

// performMove executes the Move action: validates adjacency and updates player location.
// The caller (processAction) holds gs.mutex and is responsible for releasing it.
func (gs *GameServer) performMove(player *Player, target string) error {
	targetLocation := Location(target)
	if !gs.validateMovement(player.Location, targetLocation) {
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
	results, successes, tentacles := gs.rollDicePool(2, focusSpend, player)
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
	results, successes, tentacles := gs.rollDicePool(3, focusSpend, player)
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
	results, successes, tentacles := gs.rollDicePool(3, focusSpend, player)
	actionResult := "success"
	if successes >= requiredSuccesses {
		gs.gameState.Doom = max(gs.gameState.Doom-2, 0)
		// Seal any anomaly at the player's current location.
		gs.sealAnomalyAtLocation(string(player.Location))
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
	results, successes, tentacles := gs.rollDicePool(3, focusSpend, player)
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

// performComponent is a stub for investigator-specific component abilities
// (AH3e §Component Action). Full implementation requires per-investigator ability
// tables; returns ErrNotImplemented until those are added.
// Caller must hold gs.mutex.
func (gs *GameServer) performComponent(_ *Player, playerID string) error {
	// TODO: implement per-investigator component abilities (Phase 6 final polish)
	return fmt.Errorf("component action for player %s: not yet implemented", playerID)
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
		gs.validateResources(&player.Resources)
		gs.checkInvestigatorDefeat(playerID)
	case "sanity_loss":
		player.Resources.Sanity = max(player.Resources.Sanity-card.Magnitude, 0)
		gs.validateResources(&player.Resources)
		gs.checkInvestigatorDefeat(playerID)
	case "clue_gain":
		player.Resources.Clues = min(player.Resources.Clues+card.Magnitude, MaxClues)
	case "doom_inc":
		gs.gameState.Doom = min(gs.gameState.Doom+card.Magnitude, 12)
	}
	log.Printf("Encounter at %s for %s: %s (%s %+d)", loc, playerID, card.FlavorText, card.EffectType, card.Magnitude)
	return nil
}

// Disconnected and defeated players are skipped so the game never stalls.
// When all players complete a round, runMythosPhase is called before starting
// the next round (AH3e §Mythos Phase).
func (gs *GameServer) advanceTurn() {
	if len(gs.gameState.TurnOrder) == 0 {
		return
	}

	// Find current player index
	currentIndex := -1
	for i, playerID := range gs.gameState.TurnOrder {
		if playerID == gs.gameState.CurrentPlayer {
			currentIndex = i
			break
		}
	}

	// Walk forward through the turn order, skipping disconnected or defeated players.
	// A full rotation without finding an active player means all players are gone.
	total := len(gs.gameState.TurnOrder)
	for i := 1; i <= total; i++ {
		nextIndex := (currentIndex + i) % total
		candidateID := gs.gameState.TurnOrder[nextIndex]
		candidate, exists := gs.gameState.Players[candidateID]
		if exists && candidate.Connected && !candidate.Defeated {
			// If we wrapped back to or past the first player, run Mythos Phase.
			if nextIndex <= currentIndex {
				gs.runMythosPhase()
			}
			gs.gameState.CurrentPlayer = candidateID
			candidate.ActionsRemaining = 2
			return
		}
	}
}

// runMythosPhase executes the AH3e Mythos Phase after all investigators complete
// a round. Steps:
//  1. Draw 2 events from MythosEventDeck (rebuild deck when empty).
//  2. Place each event on its target neighborhood; spread to an adjacent
//     neighborhood if a doom token is already present there.
//  3. Increment doom by 1 for each placed event.
//  4. Draw and resolve a Mythos cup token.
//  5. Restore GamePhase to "playing".
//
// Caller must hold gs.mutex.
func (gs *GameServer) runMythosPhase() {
	gs.gameState.GamePhase = "mythos"
	gs.gameState.MythosEvents = gs.gameState.MythosEvents[:0]

	// Rebuild event deck when exhausted.
	if len(gs.gameState.MythosEventDeck) == 0 {
		gs.gameState.MythosEventDeck = defaultMythosEventDeck()
	}

	// Draw up to 2 events.
	toDraw := 2
	if len(gs.gameState.MythosEventDeck) < toDraw {
		toDraw = len(gs.gameState.MythosEventDeck)
	}
	drawn := gs.gameState.MythosEventDeck[:toDraw]
	gs.gameState.MythosEventDeck = gs.gameState.MythosEventDeck[toDraw:]

	for _, evt := range drawn {
		target := evt.LocationID
		// Spread rule: if target already has a doom token, shift to first adjacent location.
		if gs.gameState.LocationDoomTokens[target] > 0 {
			if adjacent, ok := locationAdjacency[Location(target)]; ok && len(adjacent) > 0 {
				target = string(adjacent[0])
				evt.Spread = true
			}
		}
		gs.gameState.LocationDoomTokens[target]++
		gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
		gs.gameState.MythosEvents = append(gs.gameState.MythosEvents, evt)
		log.Printf("Mythos Phase: event placed at %s (spread=%v)", target, evt.Spread)
		// Spawn an anomaly if this event is an anomaly-type event.
		if evt.Effect == MythosEventAnomaly {
			gs.spawnAnomaly(target)
		}
	}

	// Draw and resolve mythos cup token.
	gs.gameState.MythosToken = gs.drawMythosToken()
	gs.resolveMythosToken(gs.gameState.MythosToken)

	log.Printf("Mythos Phase complete: doom=%d token=%s", gs.gameState.Doom, gs.gameState.MythosToken)
	gs.gameState.GamePhase = "playing"
	gs.checkGameEndConditions()
}

// drawMythosToken returns a random cup token for the Mythos Phase.
// Uses mathrand.Intn for uniform random selection, matching rollDice behaviour.
func (gs *GameServer) drawMythosToken() string {
	tokens := []string{MythosTokenDoom, MythosTokenBlessing, MythosTokenCurse, MythosTokenBlank}
	return string(tokens[mathrand.Intn(len(tokens))])
}

// resolveMythosToken applies the effect of the drawn Mythos cup token.
// Caller must hold gs.mutex.
func (gs *GameServer) resolveMythosToken(token string) {
	switch token {
	case MythosTokenDoom:
		gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
	case MythosTokenBlessing:
		if cur, ok := gs.gameState.Players[gs.gameState.CurrentPlayer]; ok {
			cur.Resources.Health = min(cur.Resources.Health+1, MaxHealth)
		}
	case MythosTokenCurse:
		if cur, ok := gs.gameState.Players[gs.gameState.CurrentPlayer]; ok {
			cur.Resources.Sanity = max(cur.Resources.Sanity-1, 0)
			gs.checkInvestigatorDefeat(gs.gameState.CurrentPlayer)
		}
	}
}

// rescaleActDeck adjusts the clue thresholds of the Act deck to match the
// documented AH3e win condition: 4 clues per investigator collectively
// (4 for 1P, 8 for 2P, 16 for 4P, 24 for 6P).  Thresholds are distributed
// evenly across three acts: base/3, 2*base/3, base (integer division).
// Call this once when the game starts, after all joining players are counted.
// Caller must hold gs.mutex.
func (gs *GameServer) rescaleActDeck(playerCount int) {
	n := max(playerCount, 1)
	base := 4 * n
	for i := range gs.gameState.ActDeck {
		switch i {
		case 0:
			gs.gameState.ActDeck[i].ClueThreshold = base / 3
		case 1:
			gs.gameState.ActDeck[i].ClueThreshold = (2 * base) / 3
		default:
			gs.gameState.ActDeck[i].ClueThreshold = base
		}
	}
	log.Printf("Act deck rescaled for %d player(s): thresholds %d / %d / %d",
		n,
		gs.gameState.ActDeck[0].ClueThreshold,
		gs.gameState.ActDeck[1].ClueThreshold,
		gs.gameState.ActDeck[2].ClueThreshold,
	)
}

// checkActAdvance evaluates whether the investigators have accumulated enough clues
// to advance the Act deck (AH3e §Act/Agenda).  On act completion the win condition
// is set when the final act card is advanced.
// Caller must hold gs.mutex.
func (gs *GameServer) checkActAdvance() {
	if len(gs.gameState.ActDeck) == 0 {
		return
	}
	totalClues := 0
	for _, p := range gs.gameState.Players {
		totalClues += p.Resources.Clues
	}
	act := gs.gameState.ActDeck[0]
	// Expose threshold for client rendering.
	gs.gameState.RequiredClues = act.ClueThreshold
	if totalClues >= act.ClueThreshold {
		log.Printf("Act advanced: %q (clues=%d)", act.Title, totalClues)
		gs.gameState.ActDeck = gs.gameState.ActDeck[1:]
		if len(gs.gameState.ActDeck) == 0 {
			gs.gameState.WinCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: Victory! Final act completed")
		}
	}
}

// checkAgendaAdvance evaluates whether doom has reached the threshold for the
// current Agenda card (AH3e §Act/Agenda).  Advances to the next agenda, and
// triggers the lose condition when the deck is exhausted.
// Caller must hold gs.mutex.
func (gs *GameServer) checkAgendaAdvance() {
	if len(gs.gameState.AgendaDeck) == 0 {
		return
	}
	agenda := gs.gameState.AgendaDeck[0]
	if gs.gameState.Doom >= agenda.DoomThreshold {
		log.Printf("Agenda advanced: %q (doom=%d)", agenda.Title, gs.gameState.Doom)
		gs.gameState.AgendaDeck = gs.gameState.AgendaDeck[1:]
		if len(gs.gameState.AgendaDeck) == 0 {
			gs.gameState.LoseCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: Final agenda reached — Ancient One awakens")
		}
	}
}

// checkGameEndConditions evaluates win/lose states.
// If the scenario provides custom WinFn/LoseFn, those take precedence;
// otherwise the default deck-driven act/agenda checks are used.
// Increments totalGamesPlayed when the game transitions to "ended".
func (gs *GameServer) checkGameEndConditions() {
	// Guard: do not re-evaluate (or double-count) an already-ended game.
	if gs.gameState.GamePhase == "ended" {
		return
	}

	// Hard doom cap — lose immediately if doom reaches 12.
	if gs.gameState.Doom >= 12 {
		gs.gameState.LoseCondition = true
		gs.gameState.GamePhase = "ended"
		atomic.AddInt64(&gs.totalGamesPlayed, 1)
		log.Printf("Game ended: Doom counter reached 12")
		return
	}

	// Scenario-provided lose check (overrides deck logic when non-nil).
	if gs.scenario.LoseFn != nil {
		if gs.scenario.LoseFn(gs.gameState) {
			gs.gameState.LoseCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: scenario lose condition triggered")
			return
		}
	} else {
		gs.checkAgendaAdvance()
		if gs.gameState.LoseCondition {
			return
		}
	}

	// Scenario-provided win check (overrides deck logic when non-nil).
	if gs.scenario.WinFn != nil {
		if gs.scenario.WinFn(gs.gameState) {
			gs.gameState.WinCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: scenario win condition triggered")
		}
	} else {
		gs.checkActAdvance()
	}
}

// spawnAnomaly places an anomaly at the given neighbourhood during the Mythos Phase.
// Each anomaly contributes 1 doom token to its location. Caller must hold gs.mutex.
func (gs *GameServer) spawnAnomaly(neighbourhood string) {
	gs.gameState.Anomalies = append(gs.gameState.Anomalies, Anomaly{
		NeighbourhoodID: neighbourhood,
		DoomTokens:      1,
	})
	gs.gameState.LocationDoomTokens[neighbourhood]++
	gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
	log.Printf("Anomaly spawned at %s (doom=%d)", neighbourhood, gs.gameState.Doom)
}

// sealAnomalyAtLocation removes the first anomaly found at neighbourhood and
// reduces doom by 2. This is the sealing effect applied on a successful Ward.
// Caller must hold gs.mutex.
func (gs *GameServer) sealAnomalyAtLocation(neighbourhood string) {
	for i, a := range gs.gameState.Anomalies {
		if a.NeighbourhoodID == neighbourhood {
			gs.gameState.Anomalies = append(gs.gameState.Anomalies[:i], gs.gameState.Anomalies[i+1:]...)
			gs.gameState.Doom = max(gs.gameState.Doom-2, 0)
			log.Printf("Anomaly sealed at %s (doom=%d)", neighbourhood, gs.gameState.Doom)
			return
		}
	}
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
