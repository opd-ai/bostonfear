// Package eldritchhorror provides a scaffolded Eldritch Horror game-family module.
//
// The module is intentionally not playable yet. It registers cleanly in the
// runtime registry, but returns a placeholder engine until Eldritch Horror-
// specific rules are implemented in follow-up roadmap phases.
//
// Planned features (when implemented):
//   - Global map with intercontinental travel
//   - Monster encounter system with escalating difficulty
//   - Unique resource economy (special items, clues, sanity)
//   - Ancient One selection with scenario-specific mechanics
//   - Support for 2-8 players with asymmetric investigator abilities
//
// Status: Scaffolding only. Use arkhamhorror module for a fully playable experience.
// See ROADMAP.md for implementation timeline.
package eldritchhorror

import (
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	commonruntime "github.com/opd-ai/bostonfear/serverengine/common/runtime"
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

// NewEngine creates a placeholder Eldritch Horror engine.
// Start always returns a not-implemented error until Eldritch Horror rules ship.
func (Module) NewEngine() (contracts.Engine, error) {
	return commonruntime.NewUnimplementedEngine("eldritchhorror"), nil
}
