package actions

// GameEngine defines the interface that arkhamhorror/actions requires from the game server.
// This allows the actions package to implement dispatching without importing serverengine,
// avoiding circular dependencies.
type GameEngine interface {
	FindEngagedEnemy(playerID string) interface{}                                      // *Enemy
	GameState() interface{}                                                            // *GameState
	RollDicePool(baseDice, focusSpend int, player interface{}) (interface{}, int, int) // ([]DiceResult, int, int)
	ValidateMovement(from, to interface{}) bool                                        // (Location, Location)
	ValidateResources(resources interface{})
	CheckInvestigatorDefeat(playerID string)
	SealAnomalyAtLocation(neighbourhood string)
}
