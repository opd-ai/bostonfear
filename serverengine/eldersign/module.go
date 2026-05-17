// Package eldersign provides a runnable Elder Sign game-family module.
//
// The module currently reuses the shared serverengine gameplay loop so it can be
// selected and hosted in production while game-family-specific rules are expanded
// in follow-up roadmap phases.
//
// Planned features (when implemented):
//   - Dice tower mechanic with strategic placement
//   - Unique Elder Sign thematic elements (Lovecraft flavor)
//   - Difficulty scaling for 1-N players
//   - Session persistence (reconnection support)
//
// Status: Scaffolding only. Use arkhamhorror module for a fully playable experience.
// See ROADMAP.md for implementation timeline.
package eldersign

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
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
// The returned engine is fully runnable and can host multiplayer sessions.
func (Module) NewEngine() (contracts.Engine, error) {
	return &Engine{GameServer: serverengine.NewGameServer()}, nil
}
