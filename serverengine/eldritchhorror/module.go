// Package eldritchhorror implements the Eldritch Horror global map cooperative game.
//
// This module provides a playable implementation of Eldritch Horror featuring:
//   - Global map: 18+ cities across 6 continents with travel routes
//   - Mystery system: Multi-stage objectives requiring worldwide coordination
//   - Ancient One mechanics: Active antagonist with unique abilities
//   - Resource economy: Health, Sanity, Focus, Clues, Money, Tickets
//   - Action system: Travel, Local, Component, Rest, Trade, Prepare, Research
//   - Monster surge: Global spawning with combat and encounter resolution
//   - Win condition: Solve 3 mysteries before Ancient One awakens or doom threshold
//   - Lose conditions: Ancient One defeats all OR doom threshold OR insufficient investigators
//
// Usage: Create an engine instance with NewModule().NewEngine() to start a game.
// Configure origins with SetAllowedOrigins([]string{...}) before handling connections.
//
// Example:
//
//	eh := eldritchhorror.NewModule()
//	engine, err := eh.NewEngine()
//	if err != nil {
//		log.Fatalf("failed to create engine: %v", err)
//	}
//	engine.SetAllowedOrigins([]string{"localhost:3000"})
//	if err := engine.Start(); err != nil {
//		log.Fatalf("engine start failed: %v", err)
//	}
package eldritchhorror

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/adapters"
)

// Module is the Eldritch Horror game-family registration point.
type Module struct{}

// NewModule returns an Eldritch Horror game module implementation.
func NewModule() contracts.GameModule {
	return Module{}
}

// Key returns the unique module identifier for this game implementation.
// Identifiers are lowercase and used in game selection routes.
func (Module) Key() string {
	return "eldritchhorror"
}

// Description returns the human-readable display name of this game module.
func (Module) Description() string {
	return "Eldritch Horror multiplayer rules engine"
}

// NewEngine creates a new Eldritch Horror game server instance.
// The returned engine manages game state, player connections, action processing,
// and broadcasting for one active game session.
// Call engine.Start() to begin accepting player connections.
// Configure engine.SetAllowedOrigins() before Start() to enable CORS filtering.
func (Module) NewEngine() (contracts.Engine, error) {
	gs := serverengine.NewGameServer()
	// Inject eldritchhorror's broadcast adapter to own wire protocol message shaping.
	gs.SetBroadcastAdapter(adapters.NewBroadcastAdapter())

	// Create the Eldritch Horror engine wrapper
	engine := &Engine{GameServer: gs}

	// Initialize Eldritch Horror-specific game state
	engine.InitializeEldritchState()

	// Register the monster movement phase to run after each turn
	gs.SetPostTurnCallback(engine.runMonsterPhase)

	return engine, nil
}
