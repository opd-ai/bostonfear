// Package arkhamhorror implements the core Arkham Horror 3rd Edition (AH3e) game rules.
//
// This module provides a fully playable implementation of Arkham Horror 3rd Edition,
// including:
//   - Location system: Downtown, University, Rivertown, Northside with adjacency rules
//   - Investigator mechanics: 6 investigator types with distinct abilities
//   - Resource economy: Health, Sanity, Clues, Focus tokens with action costs
//   - Action system: 12 action types (Move, Gather, Investigate, CastWard, etc.)
//   - Mythos phase: Agenda/Act deck progression, enemy spawning, gate locations
//   - Doom counter: 0-12 with loss condition at 12
//   - Dice resolution: 3-sided dice with focus token spending for rerolls
//   - Encounter system: Location encounters with typed effects (sanity loss, clue gain, etc.)
//
// Features:
//   - Scenario-based setup with custom win/lose conditions
//   - Support for 1-6 concurrent players with automatic difficulty scaling
//   - Dungeon encounters and anomaly/gate mechanics
//   - Component abilities (investigator-specific powers triggered via actions)
//   - Session persistence: players can reconnect and resume sessions
//
// Usage: Create an engine instance with NewModule().NewEngine() to start a game.
// Configure origins with SetAllowedOrigins([]string{...}) before handling connections.
//
// Example:
//
//	ark := arkhamhorror.NewModule()
//	engine, err := ark.NewEngine()
//	if err != nil {
//		log.Fatalf("failed to create engine: %v", err)
//	}
//	engine.SetAllowedOrigins([]string{"localhost:3000"})
//	if err := engine.Start(); err != nil {
//		log.Fatalf("engine start failed: %v", err)
//	}
package arkhamhorror

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror/adapters"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// Module provides the Arkham Horror runtime binding.
type Module struct{}

// NewModule returns an Arkham Horror game module implementation.
func NewModule() contracts.GameModule {
	return Module{}
}

// Key returns the unique module identifier for this game implementation.
// Identifiers are lowercase and used in game selection routes.
func (Module) Key() string {
	return "arkhamhorror"
}

// Description returns the human-readable display name of this game module.
func (Module) Description() string {
	return "Arkham Horror multiplayer rules engine"
}

// NewEngine creates a new Arkham Horror game server instance.
// The returned engine manages game state, player connections, action processing,
// and broadcasting for one active game session.
// Call engine.Start() to begin accepting player connections.
// Configure engine.SetAllowedOrigins() before Start() to enable CORS filtering.
func (Module) NewEngine() (contracts.Engine, error) {
	gs := serverengine.NewGameServer()
	// Inject arkhamhorror's broadcast adapter to own wire protocol message shaping.
	gs.SetBroadcastAdapter(adapters.NewBroadcastAdapter())
	return &Engine{GameServer: gs}, nil
}
