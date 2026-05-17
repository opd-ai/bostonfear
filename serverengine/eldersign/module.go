// Package eldersign provides a scaffolded Elder Sign game-family module.
//
// The module is intentionally not playable yet. It registers cleanly in the
// runtime registry, but returns a placeholder engine until Elder Sign-specific
// rules are implemented in follow-up roadmap phases.
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
	commonruntime "github.com/opd-ai/bostonfear/serverengine/common/runtime"
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

// NewEngine creates a placeholder Elder Sign engine.
// Start always returns a not-implemented error until Elder Sign rules ship.
func (Module) NewEngine() (contracts.Engine, error) {
	return commonruntime.NewUnimplementedEngine("eldersign"), nil
}
