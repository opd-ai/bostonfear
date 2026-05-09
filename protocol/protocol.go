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
// Resources represents investigator personal resources (health, sanity, clues, money, etc).
// Valid ranges are enforced by game mechanics:
//   - Health: [1, 10]; value ≤ 0 indicates investigator defeated
//   - Sanity: [1, 10]; value ≤ 0 indicates investigator defeated
//   - Clues: [0, 5]; used to advance Act deck
//   - Money: [0, unbounded]; used to purchase assets
//   - Remnants: [0, unbounded]; Elder Sign-specific resource (unused in Arkham Horror 3e)
//   - Focus: [0, unbounded]; Arkham Horror 3e focus tokens for ability use
//
// Consumers should validate these ranges before constructing Player objects or
// submitting actions that modify resources. Violations are detected during action
// processing and will trigger validation errors.
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
// Valid field values:
//   - ID: non-empty string; server-generated unique identifier
//   - Location: one of [Downtown, University, Rivertown, Northside]; default is Downtown
//   - Resources: subject to constraints in Resources type; read comment there
//   - ActionsRemaining: [0, 2]; number of unexecuted actions available this turn
//   - Connected: boolean; true if the WebSocket connection is active
//   - Defeated: boolean; true if Sanity ≤ 0 or Health ≤ 0
//   - LostInTimeAndSpace: boolean; true if investigator is in lost-and-separated state
//   - ReconnectToken: may be empty (transient) or non-empty (used to restore session)
//   - DisconnectedAt: zero-valued if connected; set to disconnection timestamp when not connected
//   - InvestigatorType: one of Researcher, Detective, Occultist, Soldier, Mystic, Survivor
//
// Consumers should not mutate Player fields directly; use action submissions instead.
// The server enforces field bounds during action processing and returns ValidationError
// with severity details if invariants are violated.
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
// GameState is the shared wire representation of the game's global state.
// Valid field values:
//   - Players: map of player ID to Player struct; all players referenced in TurnOrder must exist here
//   - CurrentPlayer: ID of the player whose turn it is; must be in Players or GameStarted is false
//   - Doom: [0, 12]; value > 12 indicates game loss
//   - GamePhase: one of [waiting, pregame, playing, mythos, mythos_resolution, game_over]
//   - TurnOrder: list of player IDs; length in [0, 6]; all IDs must exist in Players
//   - GameStarted: true after first player action; false during pregame setup
//   - WinCondition: true if investigators achieved victory goal (scenario-specific)
//   - LoseCondition: true if doom reached 12 or all players defeated
//   - RequiredClues: number of clues needed to advance Act deck (scales with player count)
//   - Difficulty: one of [easy, normal, hard]
//   - ActDeck: ordered list of unresolved Act cards; first card is current
//   - AgendaDeck: list of unresolved Agenda cards; consumed at mythos phase
//   - MythosEventDeck: deck of possible location/gate events shuffled at game start
//   - LocationDoomTokens: map of location ID to doom count at that location
//   - MythosToken: doom/blessing/curse indicator for this mythos phase
//   - MythosEvents: active location events currently in effect
//   - Anomalies: active anomalies and their locations
//   - OpenGates: list of open gates; limit typically 3 or 4
//   - Enemies: map of enemy ID to enemy state; creatures in play
//   - ActiveEvents: IDs of ongoing mythos events
//   - EncounterDecks: map of location ID to encounter card list for that location
//
// All integer bounds are validated during action processing. Consumers should not
// construct GameState manually; use server-provided state snapshots instead.
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
