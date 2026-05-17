package eldersign

import "github.com/opd-ai/bostonfear/serverengine"

// Engine is the Elder Sign module-owned runtime wrapper.
// It provides a functional server loop while game-family specific rules
// continue to evolve in later roadmap phases.
type Engine struct {
	*serverengine.GameServer
}
