package adapters

import (
	"time"

	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/model"
)

// BroadcastPayloadAdapter is the canonical interface from serverengine/common/contracts.
type BroadcastPayloadAdapter = contracts.BroadcastPayloadAdapter

// eldritchBroadcastAdapter implements BroadcastPayloadAdapter for Eldritch Horror.
// This adapter shapes broadcast messages to include global map state, active mysteries,
// and Ancient One status specific to the Eldritch Horror game family.
type eldritchBroadcastAdapter struct{}

// NewBroadcastAdapter creates a broadcast adapter for Eldritch Horror.
func NewBroadcastAdapter() BroadcastPayloadAdapter {
	return &eldritchBroadcastAdapter{}
}

// ShapeGameStatePayload transforms Eldritch Horror game state into wire format.
// The payload includes global map state, active mysteries, Ancient One status,
// and investigator positions across cities.
func (a *eldritchBroadcastAdapter) ShapeGameStatePayload(state interface{}) interface{} {
	// Eldritch Horror gameState shape: full state snapshot with global map
	return map[string]interface{}{
		"type": "gameState",
		"data": state,
	}
}

// ShapeActionResultPayload transforms action results into wire format for gameUpdate messages.
// Eldritch Horror actions may include travel costs, encounter results, and mystery progression.
func (a *eldritchBroadcastAdapter) ShapeActionResultPayload(
	action string,
	result string,
	resources interface{},
) interface{} {
	// If full payload provided by serverengine, use it
	if resources != nil {
		return resources
	}

	// Fallback shape for Eldritch Horror action events
	return map[string]interface{}{
		"type":      "gameUpdate",
		"event":     action,
		"result":    result,
		"timestamp": time.Now(),
	}
}

// ShapeDiceResultPayload transforms dice outcomes into wire format for diceResult messages.
// Eldritch Horror uses standard dice mechanics similar to Arkham Horror.
func (a *eldritchBroadcastAdapter) ShapeDiceResultPayload(diceResult interface{}) interface{} {
	// Return dice result directly - struct tags handle wire format
	return diceResult
}

// ConnectionStatusPayload creates a wire payload for Eldritch Horror connection status.
func ConnectionStatusPayload(playerID, status string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": playerID,
		"status":   status,
	}
}

// GameUpdatePayload creates an event payload for Eldritch Horror updates.
func GameUpdatePayload(playerID, event, result string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "gameUpdate",
		"playerId": playerID,
		"event":    event,
		"result":   result,
	}
}

// SerializeGameState converts EldritchGameState to a wire-safe map for broadcast.
// This includes global map state, mystery deck, Ancient One status, and all investigator positions.
func SerializeGameState(state *model.EldritchGameState) map[string]interface{} {
	return map[string]interface{}{
		"gameId":        state.GameID,
		"currentPlayer": state.CurrentPlayer,
		"turnNumber":    state.TurnNumber,
		"phaseNumber":   state.PhaseNumber,
		"gamePhase":     state.GamePhase,
		"investigators": serializeInvestigators(state.Investigators),
		"globalMap":     serializeGlobalMap(state.GlobalMap),
		"mysteryDeck":   serializeMysteryDeck(state.MysteryDeck),
		"ancientOne":    serializeAncientOne(state.AncientOne),
		"doom":          state.Doom,
		"gates":         serializeGates(state.Gates),
		"monsters":      serializeMonsters(state.Monsters),
		"gameOver":      state.GameOver,
		"gameResult":    state.GameResult,
	}
}

func serializeInvestigators(investigators map[string]model.InvestigatorState) map[string]interface{} {
	result := make(map[string]interface{})
	for id, inv := range investigators {
		result[id] = map[string]interface{}{
			"playerId":         inv.PlayerID,
			"location":         inv.Location,
			"health":           inv.Health,
			"sanity":           inv.Sanity,
			"focus":            inv.Focus,
			"clues":            inv.Clues,
			"money":            inv.Money,
			"actionsRemaining": inv.ActionsRemaining,
			"shipTickets":      inv.ShipTickets,
			"trainTickets":     inv.TrainTickets,
		}
	}
	return result
}

func serializeGlobalMap(gm model.GlobalMapState) map[string]interface{} {
	return map[string]interface{}{
		"investigatorLocations": gm.InvestigatorLocations,
		"gateLocations":         gm.GateLocations,
		"encounterLocations":    gm.EncounterLocations,
	}
}

func serializeMysteryDeck(md model.MysteryDeckState) map[string]interface{} {
	var activeMystery interface{}
	if md.ActiveMystery != nil {
		activeMystery = map[string]interface{}{
			"id":               md.ActiveMystery.ID,
			"name":             md.ActiveMystery.Name,
			"description":      md.ActiveMystery.Description,
			"currentStage":     md.ActiveMystery.CurrentStage,
			"totalStages":      md.ActiveMystery.TotalStages,
			"stageProgress":    md.ActiveMystery.StageProgress,
			"cluesContributed": md.ActiveMystery.CluesContributed,
		}
	}

	return map[string]interface{}{
		"activeMystery":      activeMystery,
		"completedMysteries": md.CompletedMysteries,
		"mysteriesToSolve":   md.MysteriesToSolve,
	}
}

func serializeAncientOne(ao model.AncientOneState) map[string]interface{} {
	return map[string]interface{}{
		"id":              ao.ID,
		"name":            ao.Name,
		"description":     ao.Description,
		"doomTrack":       ao.DoomTrack,
		"currentDoom":     ao.CurrentDoom,
		"isAwakened":      ao.IsAwakened,
		"combatRating":    ao.CombatRating,
		"horror":          ao.Horror,
		"damage":          ao.Damage,
		"activeAbilities": ao.ActiveAbilities,
	}
}

func serializeGates(gates []model.GateState) []interface{} {
	result := make([]interface{}, len(gates))
	for i, gate := range gates {
		result[i] = map[string]interface{}{
			"location":        gate.Location,
			"stability":       gate.Stability,
			"monstersSpawned": gate.MonstersSpawned,
		}
	}
	return result
}

func serializeMonsters(monsters []model.MonsterState) []interface{} {
	result := make([]interface{}, len(monsters))
	for i, monster := range monsters {
		result[i] = map[string]interface{}{
			"id":          monster.ID,
			"type":        monster.Type,
			"location":    monster.Location,
			"toughness":   monster.Toughness,
			"horror":      monster.Horror,
			"damage":      monster.Damage,
			"isEngaged":   monster.IsEngaged,
			"engagedWith": monster.EngagedWith,
		}
	}
	return result
}
