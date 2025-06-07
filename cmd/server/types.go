package main

// Location represents a game location identifier
// Moved from: main.go
type Location string

// ActionType represents available player actions
// Moved from: main.go
type ActionType string

// DiceResult represents the outcome of a dice roll
// Moved from: main.go
type DiceResult string

// Resources represents player resource tracking with bounds validation
// Moved from: main.go
type Resources struct {
	Health int `json:"health"` // 1-10
	Sanity int `json:"sanity"` // 1-10
	Clues  int `json:"clues"`  // 0-5
}

// Message represents the base JSON protocol message structure
// Moved from: main.go
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// PlayerActionMessage represents player action requests in JSON protocol
// Moved from: main.go
type PlayerActionMessage struct {
	Type     string     `json:"type"`
	PlayerID string     `json:"playerId"`
	Action   ActionType `json:"action"`
	Target   string     `json:"target,omitempty"`
}

// DiceResultMessage represents dice roll results in JSON protocol
// Moved from: main.go
type DiceResultMessage struct {
	Type        string       `json:"type"`
	PlayerID    string       `json:"playerId"`
	Action      ActionType   `json:"action"`
	Results     []DiceResult `json:"results"`
	Successes   int          `json:"successes"`
	Tentacles   int          `json:"tentacles"`
	Success     bool         `json:"success"`
	DoomIncrease int         `json:"doomIncrease"`
}

// Player represents an investigator with location and resources
// Moved from: server/main.go
type Player struct {
	ID               string    `json:"id"`
	Location         Location  `json:"location"`
	Resources        Resources `json:"resources"`
	ActionsRemaining int       `json:"actionsRemaining"`
	Connected        bool      `json:"connected"`
}

// GameState represents the complete game state
// Moved from: server/main.go
type GameState struct {
	Players       map[string]*Player `json:"players"`
	CurrentPlayer string             `json:"currentPlayer"`
	Doom          int                `json:"doom"`          // 0-12 doom counter
	GamePhase     string             `json:"gamePhase"`     // "waiting", "playing", "ended"
	TurnOrder     []string           `json:"turnOrder"`
	GameStarted   bool               `json:"gameStarted"`
	WinCondition  bool               `json:"winCondition"`
	LoseCondition bool               `json:"loseCondition"`
}
