package phases

import (
	"log"
	mathrand "math/rand"

	"github.com/opd-ai/bostonfear/protocol"
)

const (
	gamePhaseWaiting = "waiting"
	gamePhasePlaying = "playing"
	gamePhaseMythos  = "mythos"

	mythosTokenDoom     = "doom"
	mythosTokenBlessing = "blessing"
	mythosTokenCurse    = "curse"
	mythosTokenBlank    = "blank"

	mythosEventAnomaly     = "anomaly"
	mythosEventFogMadness  = "fog_madness"
	mythosEventClueDrought = "clue_drought"
	mythosEventDoomSpread  = "doom_spread"
	mythosEventResurgence  = "resurgence"
)

type TurnCallbacks struct {
	RunMythosPhase func()
}

type MythosCallbacks struct {
	RecoverInvestigator func(string)
	DefaultEventDeck    func() []protocol.MythosEvent
	AdjacentLocations   func(protocol.Location) []protocol.Location
	ResolveEventEffect  func(protocol.MythosEvent)
	OpenGateAtLocation  func(protocol.Location)
	DrawMythosToken     func() string
	ResolveMythosToken  func(string)
	SpawnEnemiesForDoom func()
	CheckGameEnd        func()
}

type EventCallbacks struct {
	SpawnAnomaly            func(string)
	ValidateResources       func(*protocol.Resources)
	CheckInvestigatorDefeat func(string)
	MaxSanity               int
}

type TokenCallbacks struct {
	CheckInvestigatorDefeat func(string)
	MaxHealth               int
}

func AdvanceTurn(state *protocol.GameState, callbacks TurnCallbacks) {
	if len(state.TurnOrder) == 0 {
		return
	}

	currentIndex := -1
	for i, playerID := range state.TurnOrder {
		if playerID == state.CurrentPlayer {
			currentIndex = i
			break
		}
	}

	total := len(state.TurnOrder)
	for i := 1; i <= total; i++ {
		nextIndex := (currentIndex + i) % total
		candidateID := state.TurnOrder[nextIndex]
		candidate, exists := state.Players[candidateID]
		if exists && candidate.Connected && !candidate.Defeated {
			if nextIndex <= currentIndex && callbacks.RunMythosPhase != nil {
				callbacks.RunMythosPhase()
			}
			state.CurrentPlayer = candidateID
			candidate.ActionsRemaining = 2
			return
		}
	}

	hasConnected := false
	hasConnectedDefeated := false
	for _, playerID := range state.TurnOrder {
		player, exists := state.Players[playerID]
		if !exists || !player.Connected {
			continue
		}
		hasConnected = true
		if player.Defeated {
			hasConnectedDefeated = true
		}
	}

	if !hasConnected {
		return
	}
	if hasConnectedDefeated && callbacks.RunMythosPhase != nil {
		callbacks.RunMythosPhase()
	}

	for i := 1; i <= total; i++ {
		nextIndex := (currentIndex + i) % total
		candidateID := state.TurnOrder[nextIndex]
		candidate, exists := state.Players[candidateID]
		if exists && candidate.Connected && !candidate.Defeated {
			state.CurrentPlayer = candidateID
			candidate.ActionsRemaining = 2
			return
		}
	}

	state.GamePhase = gamePhaseWaiting
}

func RunMythosPhase(state *protocol.GameState, callbacks MythosCallbacks) {
	state.GamePhase = gamePhaseMythos
	state.MythosEvents = state.MythosEvents[:0]
	state.ActiveEvents = state.ActiveEvents[:0]

	for id, player := range state.Players {
		if player.LostInTimeAndSpace && player.Connected && callbacks.RecoverInvestigator != nil {
			callbacks.RecoverInvestigator(id)
		}
	}

	if len(state.MythosEventDeck) == 0 && callbacks.DefaultEventDeck != nil {
		state.MythosEventDeck = callbacks.DefaultEventDeck()
	}

	toDraw := 2
	if len(state.MythosEventDeck) < toDraw {
		toDraw = len(state.MythosEventDeck)
	}
	drawn := state.MythosEventDeck[:toDraw]
	state.MythosEventDeck = state.MythosEventDeck[toDraw:]

	for _, evt := range drawn {
		target := evt.LocationID
		if state.LocationDoomTokens[target] > 0 && callbacks.AdjacentLocations != nil {
			adjacent := callbacks.AdjacentLocations(protocol.Location(target))
			if len(adjacent) > 0 {
				target = string(adjacent[0])
				evt.Spread = true
			}
		}
		state.LocationDoomTokens[target]++
		state.Doom = min(state.Doom+1, 12)
		state.MythosEvents = append(state.MythosEvents, evt)
		state.ActiveEvents = append(state.ActiveEvents, evt.Effect)
		log.Printf("Mythos Phase: event placed at %s (spread=%v type=%s)", target, evt.Spread, evt.MythosEventType)
		if callbacks.ResolveEventEffect != nil {
			callbacks.ResolveEventEffect(evt)
		}
		if state.LocationDoomTokens[target] >= 2 && callbacks.OpenGateAtLocation != nil {
			callbacks.OpenGateAtLocation(protocol.Location(target))
		}
	}

	if callbacks.DrawMythosToken != nil {
		state.MythosToken = callbacks.DrawMythosToken()
	}
	if callbacks.ResolveMythosToken != nil {
		callbacks.ResolveMythosToken(state.MythosToken)
	}
	if callbacks.SpawnEnemiesForDoom != nil {
		callbacks.SpawnEnemiesForDoom()
	}

	log.Printf("Mythos Phase complete: doom=%d token=%s activeEvents=%d", state.Doom, state.MythosToken, len(state.ActiveEvents))
	state.GamePhase = gamePhasePlaying
	if callbacks.CheckGameEnd != nil {
		callbacks.CheckGameEnd()
	}
}

func ResolveEventEffect(state *protocol.GameState, evt protocol.MythosEvent, callbacks EventCallbacks) {
	switch evt.MythosEventType {
	case mythosEventAnomaly:
		if callbacks.SpawnAnomaly != nil {
			callbacks.SpawnAnomaly(evt.LocationID)
		}
	case mythosEventFogMadness:
		for _, player := range state.Players {
			if player.Connected && !player.Defeated {
				player.Resources.Sanity = max(player.Resources.Sanity-1, 0)
				if callbacks.ValidateResources != nil {
					callbacks.ValidateResources(&player.Resources)
				}
			}
		}
	case mythosEventClueDrought:
		for _, player := range state.Players {
			if !player.Defeated {
				player.Resources.Clues = max(player.Resources.Clues-1, 0)
			}
		}
	case mythosEventDoomSpread:
		inc := len(state.OpenGates)
		if inc < 1 {
			inc = 1
		}
		state.Doom = min(state.Doom+inc, 12)
	case mythosEventResurgence:
		for _, enemy := range state.Enemies {
			if len(enemy.Engaged) > 0 {
				enemy.Health = min(enemy.Health+1, enemy.MaxHealth)
			}
		}
	}
}

func DrawMythosToken() string {
	tokens := []string{mythosTokenDoom, mythosTokenBlessing, mythosTokenCurse, mythosTokenBlank}
	return tokens[mathrand.Intn(len(tokens))]
}

func ResolveMythosToken(state *protocol.GameState, token string, callbacks TokenCallbacks) {
	switch token {
	case mythosTokenDoom:
		state.Doom = min(state.Doom+1, 12)
	case mythosTokenBlessing:
		if current, ok := state.Players[state.CurrentPlayer]; ok {
			current.Resources.Health = min(current.Resources.Health+1, callbacks.MaxHealth)
		}
	case mythosTokenCurse:
		if current, ok := state.Players[state.CurrentPlayer]; ok {
			current.Resources.Sanity = max(current.Resources.Sanity-1, 0)
			if callbacks.CheckInvestigatorDefeat != nil {
				callbacks.CheckInvestigatorDefeat(state.CurrentPlayer)
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
