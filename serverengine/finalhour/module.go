// Package finalhour provides the Final Hour game-family module with real-time mechanics.
//
// The module implements Final Hour-specific gameplay features:
//   - Real-time action programming with simultaneous player submissions
//   - Countdown tokens and escalating threat levels
//   - Priority-based conflict resolution system
//   - Focus token resource management
//   - Time-sensitive objective completion
//
// Key differences from other modules:
//   - Simultaneous action planning (not sequential turns)
//   - Priority bidding for conflict resolution (not dice-based)
//   - Single crisis location with room-based movement (not multi-neighborhood map)
//   - Countdown token pressure (not doom counter accumulation)
//
// Status: Fully implemented. Use BOSTONFEAR_GAME=finalhour to start a Final Hour game.
package finalhour

import (
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
	"github.com/opd-ai/bostonfear/serverengine/finalhour/adapters"
)

// Module is the Final Hour game-family registration point.
type Module struct{}

// NewModule returns a Final Hour game module implementation.
func NewModule() contracts.GameModule {
	return Module{}
}

// Key returns the unique module identifier for this game implementation.
// Identifiers are lowercase and used in game selection routes.
func (Module) Key() string {
	return "finalhour"
}

// Description returns the human-readable display name of this game module.
func (Module) Description() string {
	return "Final Hour multiplayer rules engine"
}

// NewEngine creates a new Final Hour game server instance.
// The returned engine manages game state, player connections, action processing,
// and broadcasting for one active game session with Final Hour-specific mechanics.
// Call engine.Start() to begin accepting player connections.
// Configure engine.SetAllowedOrigins() before Start() to enable CORS filtering.
func (Module) NewEngine() (contracts.Engine, error) {
	gs := serverengine.NewGameServer()
	// Inject finalhour's broadcast adapter to own wire protocol message shaping
	// for Final Hour-specific mechanics: priority bidding, countdown tokens, objectives.
	gs.SetBroadcastAdapter(adapters.NewBroadcastAdapter())
	return &Engine{GameServer: gs}, nil
}
