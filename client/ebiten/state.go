// Package ebiten implements the Arkham Horror game client using the Ebitengine
// game engine. It connects to the gorilla/websocket server, mirrors the server's
// game state locally, and renders the board with Ebitengine's 2-D drawing API.
package ebiten

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/bostonfear/protocol"
)

// sessionFile is the JSON structure for the persisted session file.
type sessionFile struct {
	Token string `json:"token"`
}

// tokenPath returns the path to the persisted session file:
// ~/.bostonfear/session.json
func tokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".bostonfear", "session.json"), nil
}

// LoadTokenFromFile reads the reconnect token from the session file.
// If the file does not exist, it returns nil without modifying the receiver.
func (s *LocalState) LoadTokenFromFile() error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return err
	}
	s.mu.Lock()
	s.ReconnectToken = sf.Token
	s.mu.Unlock()
	return nil
}

// SaveTokenToFile persists the current reconnect token to the session file.
// It creates the directory if it does not already exist.
func (s *LocalState) SaveTokenToFile() error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	s.mu.RLock()
	sf := sessionFile{Token: s.ReconnectToken}
	s.mu.RUnlock()
	data, err := json.Marshal(sf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Shared protocol aliases keep the Go client on the same wire schema as the server.
type (
	Location       = protocol.Location
	DiceResult     = protocol.DiceResult
	Resources      = protocol.Resources
	Player         = protocol.Player
	GameState      = protocol.GameState
	DiceResultData = protocol.DiceResultMessage
	GameUpdateData = protocol.GameUpdateMessage
)

// ConnectionQuality mirrors the server's ConnectionQuality type.
type ConnectionQuality struct {
	LatencyMs float64 `json:"latencyMs"`
	Quality   string  `json:"quality"`
}

// ConnectionStatusData mirrors the server's ConnectionStatusMessage.Data.
type ConnectionStatusData struct {
	PlayerID           string                       `json:"playerId"`
	DisplayName        string                       `json:"displayName"`
	Token              string                       `json:"token"`
	Quality            ConnectionQuality            `json:"quality"`
	AllPlayerQualities map[string]ConnectionQuality `json:"allPlayerQualities"`
}

// EventLogEntry records a recent game event for display in the event log panel.
type EventLogEntry struct {
	Timestamp time.Time
	Text      string
}

// UXMetricsSnapshot exposes client-side action usability counters for UI and tests.
type UXMetricsSnapshot struct {
	SessionStartedAt       time.Time
	FirstValidActionAt     time.Time
	TimeToFirstValidAction time.Duration
	HasFirstValidAction    bool
	ValidActionsSent       int
	InvalidActionRetries   int
	LastInvalidReason      string
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

	// ConnectAddress is the editable host:port field shown in SceneConnect.
	ConnectAddress string

	// DisplayName is the local player display name captured in SceneConnect.
	// It is sent to the server as part of the shared session join flow.
	DisplayName string

	// UX instrumentation for plan-level usability verification.
	sessionStartedAt     time.Time
	firstValidActionAt   time.Time
	validActionsSent     int
	invalidActionRetries int
	lastInvalidReason    string
	focusedActionHint    string
}

// NewLocalState creates an initialised LocalState ready for use.
// It attempts to restore a previously persisted reconnect token from
// ~/.bostonfear/session.json; missing or unreadable files are silently ignored.
func NewLocalState(serverURL string) *LocalState {
	ls := &LocalState{
		ServerURL:        serverURL,
		ConnectAddress:   hostPortFromURL(serverURL),
		sessionStartedAt: time.Now(),
		Game: GameState{
			Players:   make(map[string]*Player),
			TurnOrder: []string{},
		},
		EventLog: make([]EventLogEntry, 0, 20),
	}
	// Restore persisted token; ignore missing-file errors.
	_ = ls.LoadTokenFromFile()
	return ls
}

// RecordValidActionSent tracks a successfully queued player action.
func (s *LocalState) RecordValidActionSent() {
	s.recordValidActionSentAt(time.Now())
}

func (s *LocalState) recordValidActionSentAt(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.validActionsSent++
	if s.firstValidActionAt.IsZero() {
		s.firstValidActionAt = now
	}
}

// RecordInvalidActionRetry tracks a local action attempt that could not be sent.
func (s *LocalState) RecordInvalidActionRetry(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.invalidActionRetries++
	s.lastInvalidReason = strings.TrimSpace(reason)
}

// UXMetrics returns a point-in-time snapshot of local UX instrumentation values.
func (s *LocalState) UXMetrics() UXMetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := UXMetricsSnapshot{
		SessionStartedAt:     s.sessionStartedAt,
		FirstValidActionAt:   s.firstValidActionAt,
		HasFirstValidAction:  !s.firstValidActionAt.IsZero(),
		ValidActionsSent:     s.validActionsSent,
		InvalidActionRetries: s.invalidActionRetries,
		LastInvalidReason:    s.lastInvalidReason,
	}
	if !s.firstValidActionAt.IsZero() {
		out.TimeToFirstValidAction = s.firstValidActionAt.Sub(s.sessionStartedAt)
	}
	return out
}

// SetFocusedActionHint stores the currently focused action/location hint.
func (s *LocalState) SetFocusedActionHint(hint string) {
	s.mu.Lock()
	s.focusedActionHint = strings.TrimSpace(hint)
	s.mu.Unlock()
}

// FocusedActionHint returns the current focused action/location hint.
func (s *LocalState) FocusedActionHint() string {
	s.mu.RLock()
	hint := s.focusedActionHint
	s.mu.RUnlock()
	return hint
}

// ConnectFormSnapshot returns the address and display-name values for SceneConnect.
func (s *LocalState) ConnectFormSnapshot() (address, displayName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ConnectAddress, s.DisplayName
}

// SetConnectAddress updates the editable address and normalized WebSocket URL.
func (s *LocalState) SetConnectAddress(address string) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConnectAddress = trimmed
	s.ServerURL = websocketURLFromAddress(trimmed)
}

// SetDisplayName stores the display name used in SceneConnect.
func (s *LocalState) SetDisplayName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DisplayName = strings.TrimSpace(name)
}

// ClearConnectAddress blanks the editable address field without changing the
// last successfully dialed ServerURL.
func (s *LocalState) ClearConnectAddress() {
	s.mu.Lock()
	s.ConnectAddress = ""
	s.mu.Unlock()
}

// ClearDisplayName blanks the editable display-name field.
func (s *LocalState) ClearDisplayName() {
	s.mu.Lock()
	s.DisplayName = ""
	s.mu.Unlock()
}

// hostPortFromURL strips ws:// and path segments for the connect-form field.
func hostPortFromURL(serverURL string) string {
	trimmed := strings.TrimSpace(serverURL)
	trimmed = strings.TrimPrefix(trimmed, "ws://")
	trimmed = strings.TrimPrefix(trimmed, "wss://")
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if trimmed == "" {
		return "localhost:8080"
	}
	return trimmed
}

// websocketURLFromAddress normalizes host:port and ws:// input to a ws URL with /ws.
func websocketURLFromAddress(address string) string {
	trimmed := strings.TrimSpace(address)
	trimmed = strings.TrimSuffix(trimmed, "/")
	if strings.HasPrefix(trimmed, "ws://") || strings.HasPrefix(trimmed, "wss://") {
		if strings.HasSuffix(trimmed, "/ws") {
			return trimmed
		}
		return trimmed + "/ws"
	}
	return "ws://" + trimmed + "/ws"
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
		Text:      s.playerLabelLocked(dr.PlayerID) + " rolled " + string(dr.Action) + ": " + outcome,
	})
}

// UpdateGameEvent stores the most recent game-update event and logs it.
func (s *LocalState) UpdateGameEvent(gu GameUpdateData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastGameUpdate = &gu
	s.appendEventLocked(EventLogEntry{
		Timestamp: gu.Timestamp,
		Text:      s.playerLabelLocked(gu.PlayerID) + " " + gu.Event + " → " + gu.Result,
	})
}

// UpdateConnectionStatus stores the latest connection quality rating.
func (s *LocalState) UpdateConnectionStatus(cs ConnectionStatusData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(cs.DisplayName) != "" {
		s.DisplayName = strings.TrimSpace(cs.DisplayName)
	}
	s.ConnectionRating = cs.Quality.Quality
}

// SetConnected marks the WebSocket connection as up or down.
func (s *LocalState) SetConnected(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Connected = v
	if v {
		updateHostStatus("Connected to server")
	} else {
		updateHostStatus("Reconnecting...")
	}
}

// SetPlayerID stores the ID assigned to this client by the server.
func (s *LocalState) SetPlayerID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PlayerID = id
}

// SetReconnectToken stores the server-issued reconnect token and persists it to
// ~/.bostonfear/session.json for recovery across client restarts.
func (s *LocalState) SetReconnectToken(token string) {
	s.mu.Lock()
	s.ReconnectToken = token
	s.mu.Unlock()
	_ = s.SaveTokenToFile()
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

// LatestEventsSnapshot returns copies of the latest game update and dice result.
// Nil is returned for either value when no corresponding message has been received.
func (s *LocalState) LatestEventsSnapshot() (gameUpdate *GameUpdateData, diceResult *DiceResultData) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.LastGameUpdate != nil {
		copy := *s.LastGameUpdate
		gameUpdate = &copy
	}
	if s.LastDiceResult != nil {
		copy := *s.LastDiceResult
		diceResult = &copy
	}
	return gameUpdate, diceResult
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

func (s *LocalState) playerLabelLocked(playerID string) string {
	if playerID == "" {
		return "unknown player"
	}
	if p, ok := s.Game.Players[playerID]; ok && p != nil {
		if name := strings.TrimSpace(p.DisplayName); name != "" {
			return name
		}
	}
	return playerID
}

// Reset clears the game state for a fresh game, preserving the server URL
// and reconnect token for session persistence.
func (s *LocalState) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PlayerID = ""
	s.Game = GameState{
		Players:   make(map[string]*Player),
		TurnOrder: []string{},
	}
	s.LastDiceResult = nil
	s.LastGameUpdate = nil
	s.ConnectionRating = ""
	s.EventLog = make([]EventLogEntry, 0, 20)
	s.Connected = false
	s.sessionStartedAt = time.Now()
	s.firstValidActionAt = time.Time{}
	s.validActionsSent = 0
	s.invalidActionRetries = 0
	s.lastInvalidReason = ""
	// ReconnectToken is intentionally preserved for session recovery.
}
