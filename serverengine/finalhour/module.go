// Package finalhour provides a scaffolded Final Hour game-family module.
//
// The module is intentionally not playable yet. It registers cleanly in the
// runtime registry, but returns a placeholder engine until Final Hour-specific
// rules are implemented in follow-up roadmap phases.
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
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	commonruntime "github.com/opd-ai/bostonfear/serverengine/common/runtime"
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

// NewEngine creates a placeholder Final Hour engine.
// Start always returns a not-implemented error until Final Hour rules ship.
func (Module) NewEngine() (contracts.Engine, error) {
	return commonruntime.NewUnimplementedEngine("finalhour"), nil
}
