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

func (Module) Key() string {
	return "eldersign"
}

func (Module) Description() string {
	return "Elder Sign game-family placeholder module"
}

func (Module) NewEngine() (contracts.Engine, error) {
	return runtime.NewUnimplementedEngine("eldersign"), nil
}
