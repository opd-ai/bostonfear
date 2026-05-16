package serverengine

import "github.com/opd-ai/bostonfear/protocol"

// Shared JSON protocol aliases. The server engine owns gameplay behaviour, while
// the protocol package owns the Go wire schema shared with Go clients.
type (
	Location            = protocol.Location
	ActionType          = protocol.ActionType
	DiceResult          = protocol.DiceResult
	InvestigatorType    = protocol.InvestigatorType
	Resources           = protocol.Resources
	Message             = protocol.Message
	PlayerActionMessage = protocol.PlayerActionMessage
	DiceResultMessage   = protocol.DiceResultMessage
	Player              = protocol.Player
	EncounterCard       = protocol.EncounterCard
	MythosEvent         = protocol.MythosEvent
	ActCard             = protocol.ActCard
	AgendaCard          = protocol.AgendaCard
	Enemy               = protocol.Enemy
	Gate                = protocol.Gate
	Anomaly             = protocol.Anomaly
	GameState           = protocol.GameState
	GameUpdateMessage   = protocol.GameUpdateMessage
	ResourcesDelta      = protocol.ResourcesDelta
)

// Scenario defines the setup and win/lose conditions for a game session.
// Use DefaultScenario for standard Arkham Horror 3e play. Custom scenarios
// override the setup function and condition checks to create different experiences.
type Scenario struct {
	Name                 string
	StartingDoom         int
	MythosEventsPerRound int // Number of Mythos events drawn per round (default 2)
	SetupFn              func(*GameState)
	WinFn                func(*GameState) bool
	LoseFn               func(*GameState) bool
}
