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
