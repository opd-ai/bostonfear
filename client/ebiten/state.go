// Package ebiten implements the Arkham Horror game client using the Ebitengine
// game engine. It connects to the gorilla/websocket server, mirrors the server's
// game state locally, and renders the board with Ebitengine's 2-D drawing API.
package ebiten

import (
	"sync"
	"time"
)

// Location mirrors the server's Location type for JSON decoding.
type Location string

// DiceResult mirrors the server's DiceResult type for JSON decoding.
type DiceResult string

// Resources mirrors the server's Resources type for JSON decoding.
// Health and Sanity range 1-10; Clues range 0-5; Money 0-99; Remnants 0-5; Focus 0-3.
type Resources struct {
	Health   int `json:"health"`
	Sanity   int `json:"sanity"`
	Clues    int `json:"clues"`
	Money    int `json:"money"`
	Remnants int `json:"remnants"`
	Focus    int `json:"focus"`
}

// Player mirrors the server's Player type for JSON decoding.
type Player struct {
	ID               string    `json:"id"`
	Location         Location  `json:"location"`
	Resources        Resources `json:"resources"`
	ActionsRemaining int       `json:"actionsRemaining"`
	Connected        bool      `json:"connected"`
}

// GameState mirrors the server's GameState type for JSON decoding.
type GameState struct {
	Players       map[string]*Player `json:"players"`
	CurrentPlayer string             `json:"currentPlayer"`
	Doom          int                `json:"doom"`
	GamePhase     string             `json:"gamePhase"`
	TurnOrder     []string           `json:"turnOrder"`
	GameStarted   bool               `json:"gameStarted"`
	WinCondition  bool               `json:"winCondition"`
	LoseCondition bool               `json:"loseCondition"`
	RequiredClues int                `json:"requiredClues"`
}

// DiceResultData mirrors the server's DiceResultMessage.Data for JSON decoding.
type DiceResultData struct {
	PlayerID     string       `json:"playerId"`
	Action       string       `json:"action"`
	Results      []DiceResult `json:"results"`
	Successes    int          `json:"successes"`
	Tentacles    int          `json:"tentacles"`
	Success      bool         `json:"success"`
	DoomIncrease int          `json:"doomIncrease"`
}

// GameUpdateData mirrors the server's GameUpdateMessage for JSON decoding.
type GameUpdateData struct {
	PlayerID  string    `json:"playerId"`
	Event     string    `json:"event"`
	Result    string    `json:"result"`
	DoomDelta int       `json:"doomDelta"`
	Timestamp time.Time `json:"timestamp"`
}

// ConnectionQuality mirrors the server's ConnectionQuality type.
type ConnectionQuality struct {
	Latency int    `json:"latency"`
	Rating  string `json:"rating"`
}

// ConnectionStatusData mirrors the server's ConnectionStatusMessage.Data.
type ConnectionStatusData struct {
	PlayerID           string                       `json:"playerId"`
	Token              string                       `json:"token"`
	Quality            ConnectionQuality            `json:"quality"`
	AllPlayerQualities map[string]ConnectionQuality `json:"allPlayerQualities"`
}

// EventLogEntry records a recent game event for display in the event log panel.
type EventLogEntry struct {
	Timestamp time.Time
	Text      string
}

// LocalState holds the client-side mirror of the server game state.
// All fields are protected by mu; callers must hold mu before reading or writing.
type LocalState struct {
	mu sync.RWMutex

	// PlayerID is this client's own player identifier, assigned on connection.
	PlayerID string

	// ReconnectToken is the server-issued token used to restore a player slot
	// on reconnection. It is updated on every connectionStatus message.
	ReconnectToken string

	// Game is the latest full game state received from the server.
	Game GameState

	// LastDiceResult is the most recent dice-roll result received.
	LastDiceResult *DiceResultData

	// LastGameUpdate is the most recent lightweight event notification.
	LastGameUpdate *GameUpdateData

	// ConnectionRating is this client's current connection quality rating.
	ConnectionRating string

	// EventLog holds the last 20 game events for the event-log panel.
	EventLog []EventLogEntry

	// Connected tracks whether the WebSocket is currently connected.
	Connected bool

	// ServerURL is the WebSocket server URL (e.g. "ws://localhost:8080/ws").
	ServerURL string
}

// NewLocalState creates an initialised LocalState ready for use.
func NewLocalState(serverURL string) *LocalState {
	return &LocalState{
		ServerURL: serverURL,
		Game: GameState{
			Players:   make(map[string]*Player),
			TurnOrder: []string{},
		},
		EventLog: make([]EventLogEntry, 0, 20),
	}
}

// UpdateGame replaces the full game state with the latest server snapshot.
func (s *LocalState) UpdateGame(gs GameState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Game = gs
}

// UpdateDiceResult stores the most recent dice result and appends an event log entry.
func (s *LocalState) UpdateDiceResult(dr DiceResultData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastDiceResult = &dr
	outcome := "failed"
	if dr.Success {
		outcome = "succeeded"
	}
	s.appendEventLocked(EventLogEntry{
		Timestamp: time.Now(),
		Text:      dr.PlayerID + " rolled " + dr.Action + ": " + outcome,
	})
}

// UpdateGameEvent stores the most recent game-update event and logs it.
func (s *LocalState) UpdateGameEvent(gu GameUpdateData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastGameUpdate = &gu
	s.appendEventLocked(EventLogEntry{
		Timestamp: gu.Timestamp,
		Text:      gu.PlayerID + " " + gu.Event + " → " + gu.Result,
	})
}

// UpdateConnectionStatus stores the latest connection quality rating.
func (s *LocalState) UpdateConnectionStatus(cs ConnectionStatusData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConnectionRating = cs.Quality.Rating
}

// SetConnected marks the WebSocket connection as up or down.
func (s *LocalState) SetConnected(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Connected = v
}

// SetPlayerID stores the ID assigned to this client by the server.
func (s *LocalState) SetPlayerID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PlayerID = id
}

// SetReconnectToken stores the server-issued reconnect token.
func (s *LocalState) SetReconnectToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReconnectToken = token
}

// GetReconnectToken returns the current reconnect token, safe for concurrent use.
func (s *LocalState) GetReconnectToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ReconnectToken
}

// Snapshot returns a copy of the current game state for rendering.
// The copy is shallow for the Players map, so callers must not mutate it.
func (s *LocalState) Snapshot() (gs GameState, playerID string, connected bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Game, s.PlayerID, s.Connected
}

// EventLogSnapshot returns a copy of the event log for rendering.
func (s *LocalState) EventLogSnapshot() []EventLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]EventLogEntry, len(s.EventLog))
	copy(out, s.EventLog)
	return out
}

// appendEventLocked appends an entry, keeping the log at most 20 entries.
// Must be called with mu held.
func (s *LocalState) appendEventLocked(e EventLogEntry) {
	s.EventLog = append(s.EventLog, e)
	if len(s.EventLog) > 20 {
		s.EventLog = s.EventLog[len(s.EventLog)-20:]
	}
}
