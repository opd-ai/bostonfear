package arkhamhorror

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// Module provides the Arkham Horror runtime binding.
type Module struct{}

// NewModule returns an Arkham Horror game module implementation.
func NewModule() contracts.GameModule {
	return Module{}
}

func (Module) Key() string {
	return "arkhamhorror"
}

func (Module) Description() string {
	return "Arkham Horror multiplayer rules engine"
}

func (Module) NewEngine() (contracts.Engine, error) {
	return serverengine.NewGameServer(), nil
}
