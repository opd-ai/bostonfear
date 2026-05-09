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

func (Module) Key() string {
	return "finalhour"
}

func (Module) Description() string {
	return "Final Hour game-family placeholder module"
}

func (Module) NewEngine() (contracts.Engine, error) {
	return runtime.NewUnimplementedEngine("finalhour"), nil
}
