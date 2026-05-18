// Package eldersign implements the Elder Sign museum-exploration game rules.
//
// This module provides a fully playable implementation of Elder Sign, including:
//   - Museum room locations with adventure card deck system
//   - 6-sided dice mechanics: Terror, Peril, Lore, Investigation, Scroll, Tentacle
//   - Dice tower with lock/unlock strategy across multiple rolls
//   - Investigator mechanics: Stamina (1-8), Sanity (1-8), Elder Sign tokens
//   - Action system: PlaceInvestigator, RollDice, LockDie, DiscardItem, ClaimAdventure
//   - Victory condition: Collect Elder Sign tokens before Ancient One awakens (doom=12)
//   - Support for 1-6 concurrent players with automatic difficulty scaling
//
// Usage: Create an engine instance with NewModule().NewEngine() to start a game.
// Configure origins with SetAllowedOrigins([]string{...}) before handling connections.
//
// Example:
//
//	es := eldersign.NewModule()
//	engine, err := es.NewEngine()
//	if err != nil {
//		log.Fatalf("failed to create engine: %v", err)
//	}
//	engine.SetAllowedOrigins([]string{"localhost:3000"})
//	if err := engine.Start(); err != nil {
//		log.Fatalf("engine start failed: %v", err)
//	}
package eldersign

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	"github.com/opd-ai/bostonfear/serverengine/eldersign/adapters"
)

// Module is the Elder Sign game-family registration point.
type Module struct{}

// NewModule returns an Elder Sign game module implementation.
func NewModule() contracts.GameModule {
	return Module{}
}

// Key returns the unique module identifier for this game implementation.
// Identifiers are lowercase and used in game selection routes.
func (Module) Key() string {
	return "eldersign"
}

// Description returns the human-readable display name of this game module.
func (Module) Description() string {
	return "Elder Sign multiplayer rules engine"
}

// NewEngine creates a new Elder Sign game server instance.
// The returned engine manages game state, player connections, action processing,
// and broadcasting for one active game session with Elder Sign-specific mechanics.
// Call engine.Start() to begin accepting player connections.
// Configure engine.SetAllowedOrigins() before Start() to enable CORS filtering.
func (Module) NewEngine() (contracts.Engine, error) {
	gs := serverengine.NewGameServer()
	// Inject eldersign's broadcast adapter to own wire protocol message shaping
	// for Elder Sign-specific mechanics: 6-sided dice, adventure cards, dice locking.
	gs.SetBroadcastAdapter(adapters.NewBroadcastAdapter())
	return &Engine{GameServer: gs}, nil
}
