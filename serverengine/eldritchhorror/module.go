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
