// Package finalhour provides a runnable Final Hour game-family module.
//
// The module currently reuses the shared serverengine gameplay loop so it can be
// selected and hosted in production while game-family-specific rules are expanded
// in follow-up roadmap phases.
//
// Planned features (when implemented):
//   - Real-time turn mechanics with time pressure
//   - Countdown tokens and escalating threat levels
//   - Objective track system with branching paths
//   - Simultaneous action programming (players plan actions in secret)
//   - Unique difficulty ramping based on player count and decisions
//
// Status: Scaffolding only. Use arkhamhorror module for a fully playable experience.
// See ROADMAP.md for implementation timeline.
package finalhour

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// Module is the Final Hour game-family registration point.
type Module struct{}

// NewModule returns a Final Hour game module implementation.
func NewModule() contracts.GameModule {
	return Module{}
}

// Key returns the unique module identifier for this game implementation.
// Identifiers are lowercase and used in game selection routes.
func (Module) Key() string {
	return "finalhour"
}

// Description returns the human-readable display name of this game module.
func (Module) Description() string {
	return "Final Hour multiplayer rules engine"
}

// NewEngine creates a new Final Hour game server instance.
// The returned engine is fully runnable and can host multiplayer sessions.
func (Module) NewEngine() (contracts.Engine, error) {
	return &Engine{GameServer: serverengine.NewGameServer()}, nil
}
