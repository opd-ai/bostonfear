// Package eldritchhorror is a placeholder for Eldritch Horror game module implementation.
//
// Eldritch Horror is a globetrotting cooperative game featuring investigators
// traveling around the world to prevent the rise of ancient gods. This package
// currently returns an unimplemented engine; full rules implementation is deferred
// to a future phase.
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
	"github.com/opd-ai/bostonfear/serverengine/common/runtime"
)

// Module is the Eldritch Horror game-family registration point.
type Module struct{}

// NewModule returns an Eldritch Horror module placeholder.
func NewModule() contracts.GameModule {
	return Module{}
}

func (Module) Key() string {
	return "eldritchhorror"
}

func (Module) Description() string {
	return "Eldritch Horror game-family placeholder module"
}

func (Module) NewEngine() (contracts.Engine, error) {
	return runtime.NewUnimplementedEngine("eldritchhorror"), nil
}
