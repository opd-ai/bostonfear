// Package eldritchhorror provides a runnable Eldritch Horror game-family module.
//
// The module currently reuses the shared serverengine gameplay loop so it can be
// selected and hosted in production while game-family-specific rules are expanded
// in follow-up roadmap phases.
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
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
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
// The returned engine is fully runnable and can host multiplayer sessions.
func (Module) NewEngine() (contracts.Engine, error) {
	return &Engine{GameServer: serverengine.NewGameServer()}, nil
}
