// Package finalhour is a placeholder for Final Hour game module implementation.
//
// Final Hour is a high-tension cooperative game where investigators race against
// time to prevent an impending apocalypse. This package currently returns an
// unimplemented engine; full rules implementation is deferred to a future phase.
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
	"github.com/opd-ai/bostonfear/serverengine/common/runtime"
)

// Module is the Final Hour game-family registration point.
type Module struct{}

// NewModule returns a Final Hour module placeholder.
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
	return "Final Hour game-family placeholder module"
}

// NewEngine creates a new Final Hour game server instance.
// Currently returns an unimplemented placeholder; full implementation is pending.
func (Module) NewEngine() (contracts.Engine, error) {
	return runtime.NewUnimplementedEngine("finalhour"), nil
}
