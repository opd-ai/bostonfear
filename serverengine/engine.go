package serverengine

import (
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// GameEngine is retained as a compatibility alias for existing callers.
// New code should consume contracts.Engine directly.
type GameEngine = contracts.Engine

// Compile-time guarantee that GameServer satisfies the exported engine contract.
var _ contracts.Engine = (*GameServer)(nil)
