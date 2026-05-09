// Package eldersign is a placeholder for Elder Sign game module implementation.
//
// Elder Sign is a cooperative dice-placement game where investigators work together
// to seal gates and prevent the rise of Cthulhu. This package currently returns an
// unimplemented engine; full rules implementation is deferred to a future phase.
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
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	"github.com/opd-ai/bostonfear/serverengine/common/runtime"
)

// Module is the Elder Sign game-family registration point.
type Module struct{}

// NewModule returns an Elder Sign module placeholder.
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
	return "Elder Sign game-family placeholder module"
}

// NewEngine creates a new Elder Sign game server instance.
// Currently returns an unimplemented placeholder; full implementation is pending.
func (Module) NewEngine() (contracts.Engine, error) {
	return runtime.NewUnimplementedEngine("eldersign"), nil
}
