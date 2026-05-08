// Package serverengine implements the Mythos Phase for the Arkham Horror multiplayer
// game server. This file implements the AH3e Mythos Phase (event draw, placement,
// spreading, token cup, enemy spawning) and the turn-order advancement logic that
// triggers it at the end of each investigator round.
package serverengine

import (
	"fmt"
	"log"
	mathrand "math/rand"
	"sync/atomic"
)

// advanceTurn moves the CurrentPlayer pointer to the next active investigator.
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
//  4. Resolve each event's typed mechanical effect (sanity loss, clue loss, etc.).
//  5. Draw and resolve a Mythos cup token.
//  6. Spawn enemies scaled to accumulated doom.
//  7. Restore GamePhase to "playing".
//
// Caller must hold gs.mutex.
func (gs *GameServer) runMythosPhase() {
	gs.gameState.GamePhase = "mythos"
	gs.gameState.MythosEvents = gs.gameState.MythosEvents[:0]
	gs.gameState.ActiveEvents = gs.gameState.ActiveEvents[:0]

	// Recover defeated investigators who are still connected (RULES.md §Defeat/Recovery).
	// Recovery happens at the start of each Mythos Phase so they re-enter the next round.
	for id, p := range gs.gameState.Players {
		if p.LostInTimeAndSpace && p.Connected {
			gs.recoverInvestigator(id)
		}
	}

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
		gs.gameState.ActiveEvents = append(gs.gameState.ActiveEvents, evt.Effect)
		log.Printf("Mythos Phase: event placed at %s (spread=%v type=%s)", target, evt.Spread, evt.MythosEventType)
		gs.resolveEventEffect(evt)
		// Open a gate when a neighbourhood accumulates ≥ 2 doom tokens.
		if gs.gameState.LocationDoomTokens[target] >= 2 {
			gs.openGateAtLocation(Location(target))
		}
	}

	// Draw and resolve mythos cup token.
	gs.gameState.MythosToken = gs.drawMythosToken()
	gs.resolveMythosToken(gs.gameState.MythosToken)

	// Spawn enemies: 1 per 3 accumulated doom (e.g., doom=6 → 2 spawns), capped at maxEnemiesOnBoard.
	gs.spawnEnemiesForDoom()

	log.Printf("Mythos Phase complete: doom=%d token=%s activeEvents=%d", gs.gameState.Doom, gs.gameState.MythosToken, len(gs.gameState.ActiveEvents))
	gs.gameState.GamePhase = "playing"
	gs.checkGameEndConditions()
}

// resolveEventEffect applies the typed mechanical effect of a Mythos event.
// Caller must hold gs.mutex.
func (gs *GameServer) resolveEventEffect(evt MythosEvent) {
	switch evt.MythosEventType {
	case MythosEventAnomaly:
		gs.spawnAnomaly(evt.LocationID)

	case MythosEventFogMadness:
		// All connected (and non-defeated) investigators lose 1 Sanity.
		for _, p := range gs.gameState.Players {
			if p.Connected && !p.Defeated {
				p.Resources.Sanity = max(p.Resources.Sanity-1, 0)
				gs.validateResources(&p.Resources)
			}
		}

	case MythosEventClueDrought:
		// All investigators lose 1 Clue (clues washed away).
		for _, p := range gs.gameState.Players {
			if !p.Defeated {
				p.Resources.Clues = max(p.Resources.Clues-1, 0)
			}
		}

	case MythosEventDoomSpread:
		// Doom +1 per open gate (minimum +1 when no gates are open).
		inc := len(gs.gameState.OpenGates)
		if inc < 1 {
			inc = 1
		}
		gs.gameState.Doom = min(gs.gameState.Doom+inc, 12)

	case MythosEventResurgence:
		// Each engaged enemy regains 1 Health, capped at its archetype MaxHealth.
		for _, e := range gs.gameState.Enemies {
			if len(e.Engaged) > 0 {
				e.Health = min(e.Health+1, e.MaxHealth)
			}
		}
	}
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
// to advance the Act deck (AH3e §Act/Agenda).  Cascades through all act cards
// whose thresholds are met in a single evaluation, setting the win condition
// when the final card is advanced.
// Caller must hold gs.mutex.
func (gs *GameServer) checkActAdvance() {
	if len(gs.gameState.ActDeck) == 0 {
		return
	}
	totalClues := 0
	for _, p := range gs.gameState.Players {
		totalClues += p.Resources.Clues
	}
	// Expose threshold of the current (front) card for client rendering.
	gs.gameState.RequiredClues = gs.gameState.ActDeck[0].ClueThreshold

	for len(gs.gameState.ActDeck) > 0 {
		act := gs.gameState.ActDeck[0]
		if totalClues < act.ClueThreshold {
			break
		}
		log.Printf("Act advanced: %q (clues=%d)", act.Title, totalClues)
		gs.gameState.ActDeck = gs.gameState.ActDeck[1:]
		if len(gs.gameState.ActDeck) == 0 {
			gs.gameState.WinCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: Victory! Final act completed")
			return
		}
		// Update required clues display for next card.
		gs.gameState.RequiredClues = gs.gameState.ActDeck[0].ClueThreshold
	}
}

// checkAgendaAdvance evaluates whether doom has reached the threshold for the
// current Agenda card (AH3e §Act/Agenda).  Advances through all agenda cards
// whose thresholds are met (cascade) and triggers the lose condition when the
// deck is exhausted.
// Caller must hold gs.mutex.
func (gs *GameServer) checkAgendaAdvance() {
	for len(gs.gameState.AgendaDeck) > 0 {
		agenda := gs.gameState.AgendaDeck[0]
		if gs.gameState.Doom < agenda.DoomThreshold {
			break
		}
		log.Printf("Agenda advanced: %q (doom=%d)", agenda.Title, gs.gameState.Doom)
		gs.gameState.AgendaDeck = gs.gameState.AgendaDeck[1:]
		if len(gs.gameState.AgendaDeck) == 0 {
			gs.gameState.LoseCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: Final agenda reached — Ancient One awakens")
			return
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

// spawnEnemiesForDoom spawns 1 enemy for every 3 doom on the board, up to maxEnemiesOnBoard.
// Each new enemy is placed at a random location from the canonical four neighbourhoods.
// Caller must hold gs.mutex.
func (gs *GameServer) spawnEnemiesForDoom() {
	targetCount := gs.gameState.Doom / 3
	if targetCount > maxEnemiesOnBoard {
		targetCount = maxEnemiesOnBoard
	}
	current := len(gs.gameState.Enemies)
	toSpawn := targetCount - current
	if toSpawn <= 0 {
		return
	}
	locations := []Location{Downtown, University, Rivertown, Northside}
	for i := 0; i < toSpawn; i++ {
		tmpl := enemyTemplates[mathrand.Intn(len(enemyTemplates))]
		loc := locations[mathrand.Intn(len(locations))]
		id := fmt.Sprintf("enemy_%d", mathrand.Int())
		e := &Enemy{
			ID:        id,
			Name:      tmpl.Name,
			Health:    tmpl.Health,
			MaxHealth: tmpl.Health, // archetype baseline for Resurgence capping
			Damage:    tmpl.Damage,
			Horror:    tmpl.Horror,
			Location:  loc,
			Engaged:   nil,
		}
		gs.gameState.Enemies[id] = e
		log.Printf("Enemy spawned: %s at %s (doom=%d)", e.Name, loc, gs.gameState.Doom)
	}
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

// openGateAtLocation opens a new Gate at the given neighbourhood if one is not
// already present there. Called from runMythosPhase when a location accumulates
// ≥ 2 doom tokens. Caller must hold gs.mutex.
func (gs *GameServer) openGateAtLocation(loc Location) {
	for _, g := range gs.gameState.OpenGates {
		if g.Location == loc {
			return // gate already open here
		}
	}
	id := fmt.Sprintf("gate_%s_%d", loc, mathrand.Int())
	gs.gameState.OpenGates = append(gs.gameState.OpenGates, Gate{ID: id, Location: loc})
	log.Printf("Gate opened at %s (doom=%d)", loc, gs.gameState.Doom)
}
