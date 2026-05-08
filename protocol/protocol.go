// Package protocol owns the shared JSON wire contract used by the Go server and
// Go clients. Keeping these enums and payloads in one package reduces schema
// drift when the multiplayer protocol evolves.
package protocol

import "time"

// Location identifies one of the four interconnected neighbourhoods on the map.
type Location string

// ActionType identifies a player action sent over the wire.
type ActionType string

// DiceResult identifies the outcome of one die in the three-sided dice system.
type DiceResult string

// InvestigatorType identifies the selected investigator archetype.
type InvestigatorType string

const (
	Downtown   Location = "Downtown"
	University Location = "University"
	Rivertown  Location = "Rivertown"
	Northside  Location = "Northside"
)

const (
	ActionMove               ActionType = "move"
	ActionGather             ActionType = "gather"
	ActionInvestigate        ActionType = "investigate"
	ActionCastWard           ActionType = "ward"
	ActionFocus              ActionType = "focus"
	ActionResearch           ActionType = "research"
	ActionTrade              ActionType = "trade"
	ActionComponent          ActionType = "component"
	ActionEncounter          ActionType = "encounter"
	ActionAttack             ActionType = "attack"
	ActionEvade              ActionType = "evade"
	ActionCloseGate          ActionType = "closegate"
	ActionSelectInvestigator ActionType = "selectinvestigator"
	ActionSetDifficulty      ActionType = "setdifficulty"
	ActionChat               ActionType = "chat"
)

const (
	InvestigatorResearcher InvestigatorType = "researcher"
	InvestigatorDetective  InvestigatorType = "detective"
	InvestigatorOccultist  InvestigatorType = "occultist"
	InvestigatorSoldier    InvestigatorType = "soldier"
	InvestigatorMystic     InvestigatorType = "mystic"
	InvestigatorSurvivor   InvestigatorType = "survivor"
)

const (
	DiceSuccess  DiceResult = "success"
	DiceBlank    DiceResult = "blank"
	DiceTentacle DiceResult = "tentacle"
)

// Resources represents player resource tracking with bounds validation.
type Resources struct {
	Health   int `json:"health"`
	Sanity   int `json:"sanity"`
	Clues    int `json:"clues"`
	Money    int `json:"money"`
	Remnants int `json:"remnants"`
	Focus    int `json:"focus"`
}

// Message is the base JSON envelope used by full game-state broadcasts.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// PlayerActionMessage represents an action request from a player.
type PlayerActionMessage struct {
	Type       string     `json:"type"`
	PlayerID   string     `json:"playerId"`
	Action     ActionType `json:"action"`
	Target     string     `json:"target,omitempty"`
	FocusSpend int        `json:"focusSpend,omitempty"`
}

// DiceResultMessage represents the dice outcome for a player action.
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

// Player is the shared wire representation of an investigator.
type Player struct {
	ID                 string           `json:"id"`
	Location           Location         `json:"location"`
	Resources          Resources        `json:"resources"`
	ActionsRemaining   int              `json:"actionsRemaining"`
	Connected          bool             `json:"connected"`
	Defeated           bool             `json:"defeated"`
	LostInTimeAndSpace bool             `json:"lostInTimeAndSpace"`
	ReconnectToken     string           `json:"reconnectToken"`
	DisconnectedAt     time.Time        `json:"disconnectedAt"`
	InvestigatorType   InvestigatorType `json:"investigatorType"`
}

// EncounterCard is the shared wire shape for a location encounter card.
type EncounterCard struct {
	FlavorText string `json:"flavorText"`
	EffectType string `json:"effectType"`
	Magnitude  int    `json:"magnitude"`
}

// MythosEvent is the shared wire shape for a resolved or queued Mythos event.
type MythosEvent struct {
	LocationID      string `json:"locationId"`
	Effect          string `json:"effect"`
	Spread          bool   `json:"spread"`
	MythosEventType string `json:"mythosEventType"`
}

// ActCard is the shared wire shape for an act-deck card.
type ActCard struct {
	Title         string `json:"title"`
	ClueThreshold int    `json:"clueThreshold"`
	Effect        string `json:"effect"`
}

// AgendaCard is the shared wire shape for an agenda-deck card.
type AgendaCard struct {
	Title         string `json:"title"`
	DoomThreshold int    `json:"doomThreshold"`
	Effect        string `json:"effect"`
}

// Enemy is the shared wire shape for an enemy currently on the board.
type Enemy struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Health    int      `json:"health"`
	MaxHealth int      `json:"maxHealth"`
	Damage    int      `json:"damage"`
	Horror    int      `json:"horror"`
	Location  Location `json:"location"`
	Engaged   []string `json:"engaged"`
}

// Gate is the shared wire shape for an open gate.
type Gate struct {
	ID       string   `json:"id"`
	Location Location `json:"location"`
}

// Anomaly is the shared wire shape for a spawned anomaly.
type Anomaly struct {
	NeighbourhoodID string `json:"neighbourhoodId"`
	DoomTokens      int    `json:"doomTokens"`
}

// GameState is the shared full-snapshot payload sent to Go clients.
type GameState struct {
	Players            map[string]*Player         `json:"players"`
	CurrentPlayer      string                     `json:"currentPlayer"`
	Doom               int                        `json:"doom"`
	GamePhase          string                     `json:"gamePhase"`
	TurnOrder          []string                   `json:"turnOrder"`
	GameStarted        bool                       `json:"gameStarted"`
	WinCondition       bool                       `json:"winCondition"`
	LoseCondition      bool                       `json:"loseCondition"`
	RequiredClues      int                        `json:"requiredClues"`
	Difficulty         string                     `json:"difficulty"`
	ActDeck            []ActCard                  `json:"actDeck"`
	AgendaDeck         []AgendaCard               `json:"agendaDeck"`
	MythosEventDeck    []MythosEvent              `json:"mythosEventDeck"`
	LocationDoomTokens map[string]int             `json:"locationDoomTokens"`
	MythosToken        string                     `json:"mythosToken"`
	MythosEvents       []MythosEvent              `json:"mythosEvents"`
	Anomalies          []Anomaly                  `json:"anomalies"`
	OpenGates          []Gate                     `json:"openGates"`
	Enemies            map[string]*Enemy          `json:"enemies"`
	ActiveEvents       []string                   `json:"activeEvents"`
	EncounterDecks     map[string][]EncounterCard `json:"encounterDecks"`
}

// GameUpdateMessage is the lightweight post-action delta broadcast.
type GameUpdateMessage struct {
	Type          string         `json:"type"`
	PlayerID      string         `json:"playerId"`
	Event         string         `json:"event"`
	Result        string         `json:"result"`
	DoomDelta     int            `json:"doomDelta"`
	ResourceDelta ResourcesDelta `json:"resourceDelta"`
	Timestamp     time.Time      `json:"timestamp"`
}

// ResourcesDelta is the net resource change caused by a single action.
type ResourcesDelta struct {
	Health int `json:"health"`
	Sanity int `json:"sanity"`
	Clues  int `json:"clues"`
}
