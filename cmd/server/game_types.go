package main

import "time"

// Location represents a game location identifier for one of the four interconnected
// neighborhoods in the Arkham Horror game map.
type Location string

// ActionType represents available player actions during a turn including
// movement, resource gathering, investigation, and ward casting.
type ActionType string

// DiceResult represents the outcome of a single die roll with three possible
// results: success, blank, or tentacle.
type DiceResult string

// Resources represents player resource tracking with bounds validation.
// Health and Sanity range from 0-10 (0 means the investigator is defeated),
// Clues range from 0-5, Money 0-99, Remnants 0-5, Focus 0-3.
type Resources struct {
	Health   int `json:"health"`   // 0-10 (0 = defeated)
	Sanity   int `json:"sanity"`   // 0-10 (0 = defeated)
	Clues    int `json:"clues"`    // 0-5
	Money    int `json:"money"`    // 0-99
	Remnants int `json:"remnants"` // 0-5
	Focus    int `json:"focus"`    // 0-3
}

// Message represents the base JSON protocol message structure for WebSocket
// communication between server and clients.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// PlayerActionMessage represents player action requests in the JSON protocol,
// containing the player ID, action type, and optional target.
type PlayerActionMessage struct {
	Type       string     `json:"type"`
	PlayerID   string     `json:"playerId"`
	Action     ActionType `json:"action"`
	Target     string     `json:"target,omitempty"`
	FocusSpend int        `json:"focusSpend,omitempty"` // focus tokens to spend for extra dice
}

// DiceResultMessage represents dice roll results in the JSON protocol,
// including individual die results, success/tentacle counts, and doom impact.
type DiceResultMessage struct {
	Type         string       `json:"type"`
	PlayerID     string       `json:"playerId"`
	Action       ActionType   `json:"action"`
	Results      []DiceResult `json:"results"`
	Successes    int          `json:"successes"`
	Tentacles    int          `json:"tentacles"`
	Success      bool         `json:"success"`
	DoomIncrease int          `json:"doomIncrease"`
}

// Player represents an investigator with location, resources, and turn state.
// Each player has a unique ID and tracks their connection status.
type Player struct {
	ID                 string           `json:"id"`
	Location           Location         `json:"location"`
	Resources          Resources        `json:"resources"`
	ActionsRemaining   int              `json:"actionsRemaining"`
	Connected          bool             `json:"connected"`
	Defeated           bool             `json:"defeated"`           // true when Health or Sanity reaches 0
	LostInTimeAndSpace bool             `json:"lostInTimeAndSpace"` // true when investigator is defeated and awaiting recovery
	ReconnectToken     string           `json:"reconnectToken"`     // opaque token for session restoration
	DisconnectedAt     time.Time        `json:"disconnectedAt"`     // zero value means currently connected
	InvestigatorType   InvestigatorType `json:"investigatorType"`   // determines component ability
}

// Scenario defines the setup and win/lose conditions for a game session.
// Use DefaultScenario for standard Arkham Horror 3e play. Custom scenarios
// override the setup function and condition checks to create different experiences.
type Scenario struct {
	Name         string
	StartingDoom int
	SetupFn      func(*GameState)
	WinFn        func(*GameState) bool
	LoseFn       func(*GameState) bool
}

// EncounterCard represents a single card in a location-specific encounter deck.
// When an investigator performs the Encounter action, one card is drawn from
// the deck for their current location and its effect applied.
type EncounterCard struct {
	FlavorText string `json:"flavorText"`
	EffectType string `json:"effectType"` // "health_loss", "sanity_loss", "clue_gain", "doom_inc"
	Magnitude  int    `json:"magnitude"`  // amount to apply (positive = gain, negative = loss)
}

// MythosEvent represents a single drawn Mythos event card.
// During the Mythos Phase, 2 events are drawn, placed on target neighborhoods,
// and spread to adjacent neighborhoods if a doom token is already present.
type MythosEvent struct {
	LocationID      string `json:"locationId"`      // target neighborhood
	Effect          string `json:"effect"`          // narrative effect description
	Spread          bool   `json:"spread"`          // true when placed via spread rule
	MythosEventType string `json:"mythosEventType"` // event category, e.g. "anomaly"
}

// ActCard represents a single card in the Act deck (AH3e §Act/Agenda).
// Investigators advance the act by collectively accumulating ClueThreshold clues.
type ActCard struct {
	Title         string `json:"title"`
	ClueThreshold int    `json:"clueThreshold"` // total clues required to advance
	Effect        string `json:"effect"`        // narrative outcome when advanced
}

// AgendaCard represents a single card in the Agenda deck (AH3e §Act/Agenda).
// The agenda advances each time doom reaches its DoomThreshold; the final
// agenda card triggers the lose condition.
type AgendaCard struct {
	Title         string `json:"title"`
	DoomThreshold int    `json:"doomThreshold"` // doom level at which this card advances
	Effect        string `json:"effect"`        // narrative outcome when advanced
}

// Enemy represents a monster on the board that investigators can attack or evade.
// Health tracks remaining hit points; the enemy is removed when Health reaches 0.
// MaxHealth is the archetype baseline set at spawn time; it caps Resurgence healing.
// Engaged lists the player IDs currently in combat with this enemy.
type Enemy struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Health    int      `json:"health"`
	MaxHealth int      `json:"maxHealth"` // archetype max; set at spawn, used to cap healing
	Damage    int      `json:"damage"`    // health damage dealt to the investigator per attack
	Horror    int      `json:"horror"`    // sanity damage dealt to the investigator per attack
	Location  Location `json:"location"`
	Engaged   []string `json:"engaged"` // player IDs currently engaged with this enemy
}

// Gate represents an interdimensional rift at a neighbourhood location.
// Gates open when a location accumulates ≥ 2 doom tokens and can be closed
// by an investigator spending 2 Clues (ActionCloseGate).
type Gate struct {
	ID       string   `json:"id"`
	Location Location `json:"location"`
}

// Anomaly represents a spatial tear spawned during the Mythos Phase.
// Investigators can seal anomalies by successfully casting a Ward (3 successes).
// Sealing an anomaly removes it and reduces global doom by 2.
type Anomaly struct {
	NeighbourhoodID string `json:"neighbourhoodId"` // location where anomaly is placed
	DoomTokens      int    `json:"doomTokens"`      // doom tokens this anomaly has accumulated
}

// GameState represents the complete game state including all players,
// turn order, doom counter, and win/lose conditions.
type GameState struct {
	Players       map[string]*Player `json:"players"`
	CurrentPlayer string             `json:"currentPlayer"`
	Doom          int                `json:"doom"`      // 0-12 doom counter
	GamePhase     string             `json:"gamePhase"` // "waiting", "playing", "mythos", "ended"
	TurnOrder     []string           `json:"turnOrder"`
	GameStarted   bool               `json:"gameStarted"`
	WinCondition  bool               `json:"winCondition"`
	LoseCondition bool               `json:"loseCondition"`
	RequiredClues int                `json:"requiredClues"` // kept for backward compatibility; derived from ActDeck
	Difficulty    string             `json:"difficulty"`    // "easy", "standard", "hard"
	// Act/Agenda deck state
	ActDeck    []ActCard    `json:"actDeck"`    // remaining act cards
	AgendaDeck []AgendaCard `json:"agendaDeck"` // remaining agenda cards
	// Mythos Phase state
	MythosEventDeck    []MythosEvent  `json:"mythosEventDeck"`    // event draw pile
	LocationDoomTokens map[string]int `json:"locationDoomTokens"` // doom tokens per neighborhood
	MythosToken        string         `json:"mythosToken"`        // current cup token drawn
	MythosEvents       []MythosEvent  `json:"mythosEvents"`       // events resolved this phase
	// Anomalies spawned during the Mythos Phase
	Anomalies []Anomaly `json:"anomalies"`
	// OpenGates lists currently open interdimensional rifts; keyed indirectly via the slice.
	OpenGates []Gate `json:"openGates"`
	// Enemies present on the board; keyed by enemy ID.
	Enemies map[string]*Enemy `json:"enemies"`
	// ActiveEvents holds the descriptions of Mythos events resolved in the last Mythos Phase.
	// Clients display this list as the current "active event" overlay.
	ActiveEvents []string `json:"activeEvents"`
	// Encounter decks keyed by location name
	EncounterDecks map[string][]EncounterCard `json:"encounterDecks"`
}

// GameUpdateMessage represents a lightweight event notification emitted after each
// player action. It is distinct from the full gameState broadcast and carries only
// the delta: which action occurred, its outcome, and any resource/doom changes.
// This satisfies the fifth required JSON protocol message type.
type GameUpdateMessage struct {
	Type          string         `json:"type"`
	PlayerID      string         `json:"playerId"`
	Event         string         `json:"event"`
	Result        string         `json:"result"`        // "success" or "fail"
	DoomDelta     int            `json:"doomDelta"`     // doom change from this action
	ResourceDelta ResourcesDelta `json:"resourceDelta"` // resource changes for the acting player
	Timestamp     time.Time      `json:"timestamp"`
}

// ResourcesDelta represents the net change in resources for a single action.
type ResourcesDelta struct {
	Health int `json:"health"`
	Sanity int `json:"sanity"`
	Clues  int `json:"clues"`
}
