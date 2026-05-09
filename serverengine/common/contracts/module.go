package contracts

// GameModule describes a game-family implementation that can provide a concrete
// engine instance (for example Arkham Horror, Elder Sign, or Eldritch Horror).
type GameModule interface {
	Key() string
	Description() string
	NewEngine() (Engine, error)
}
