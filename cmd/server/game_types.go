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
	Type     string     `json:"type"`
	PlayerID string     `json:"playerId"`
	Action   ActionType `json:"action"`
	Target   string     `json:"target,omitempty"`
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
	ID               string    `json:"id"`
	Location         Location  `json:"location"`
	Resources        Resources `json:"resources"`
	ActionsRemaining int       `json:"actionsRemaining"`
	Connected        bool      `json:"connected"`
	Defeated         bool      `json:"defeated"`       // true when Health or Sanity reaches 0
	ReconnectToken   string    `json:"reconnectToken"` // opaque token for session restoration
	DisconnectedAt   time.Time `json:"disconnectedAt"` // zero value means currently connected
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
	LocationID string `json:"locationId"` // target neighborhood
	Effect     string `json:"effect"`     // narrative effect description
	Spread     bool   `json:"spread"`     // true when placed via spread rule
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
	// Act/Agenda deck state
	ActDeck    []ActCard    `json:"actDeck"`    // remaining act cards
	AgendaDeck []AgendaCard `json:"agendaDeck"` // remaining agenda cards
	// Mythos Phase state
	MythosEventDeck    []MythosEvent  `json:"mythosEventDeck"`    // event draw pile
	LocationDoomTokens map[string]int `json:"locationDoomTokens"` // doom tokens per neighborhood
	MythosToken        string         `json:"mythosToken"`        // current cup token drawn
	MythosEvents       []MythosEvent  `json:"mythosEvents"`       // events resolved this phase
	// Encounter decks keyed by location name
	EncounterDecks map[string][]EncounterCard `json:"encounterDecks"`
}

// PerformanceMetrics represents comprehensive server performance data for monitoring dashboard
type PerformanceMetrics struct {
	Uptime               time.Duration `json:"uptime"`
	ActiveConnections    int           `json:"activeConnections"`
	PeakConnections      int           `json:"peakConnections"`
	TotalConnections     int64         `json:"totalConnections"`
	ConnectionsPerSecond float64       `json:"connectionsPerSecond"`
	AverageSessionLength time.Duration `json:"averageSessionLength"`
	ActiveSessions       int           `json:"activeSessions"`
	TotalGamesPlayed     int64         `json:"totalGamesPlayed"`
	MessagesPerSecond    float64       `json:"messagesPerSecond"`
	MemoryUsage          MemoryStats   `json:"memoryUsage"`
	ResponseTimeMs       float64       `json:"responseTimeMs"`
	ErrorRate            float64       `json:"errorRate"`
}

// MemoryStats represents memory usage statistics for performance monitoring
type MemoryStats struct {
	AllocMB      float64 `json:"allocMB"`
	TotalAllocMB float64 `json:"totalAllocMB"`
	SysMB        float64 `json:"sysMB"`
	NumGC        uint32  `json:"numGC"`
	GCPauseMs    float64 `json:"gcPauseMs"`
}

// ConnectionAnalytics represents player connection patterns and engagement metrics
type ConnectionAnalytics struct {
	RecentConnections   []ConnectionEvent     `json:"recentConnections"`
	PlayerEngagement    map[string]float64    `json:"playerEngagement"`
	ConnectionQuality   ConnectionQualityData `json:"connectionQuality"`
	SessionDistribution SessionDistribution   `json:"sessionDistribution"`
	GeographicData      map[string]int        `json:"geographicData"`
}

// ConnectionEvent represents individual connection events for analytics
type ConnectionEvent struct {
	Type      string        `json:"type"` // "connect", "disconnect", "reconnect"
	PlayerID  string        `json:"playerId"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
	Reason    string        `json:"reason,omitempty"`
}

// ConnectionQualityData represents connection stability metrics
type ConnectionQualityData struct {
	AverageLatency    time.Duration `json:"averageLatency"`
	PacketLossRate    float64       `json:"packetLossRate"`
	ReconnectionRate  float64       `json:"reconnectionRate"`
	StableConnections int           `json:"stableConnections"`
}

// SessionDistribution represents session length distribution for analytics
type SessionDistribution struct {
	Short  int `json:"short"`  // < 5 minutes
	Medium int `json:"medium"` // 5-30 minutes
	Long   int `json:"long"`   // > 30 minutes
}

// PlayerSessionMetrics tracks individual player session data
type PlayerSessionMetrics struct {
	PlayerID        string        `json:"playerId"`
	SessionStart    time.Time     `json:"sessionStart"`
	LastActivity    time.Time     `json:"lastActivity"`
	TotalActions    int           `json:"totalActions"`
	GamesCompleted  int           `json:"gamesCompleted"`
	AvgResponseTime time.Duration `json:"avgResponseTime"`
	Disconnections  int           `json:"disconnections"`
}

// AlertThreshold represents configurable monitoring thresholds
type AlertThreshold struct {
	MaxResponseTime time.Duration `json:"maxResponseTime"`
	MaxErrorRate    float64       `json:"maxErrorRate"`
	MinUptime       float64       `json:"minUptime"`
	MaxMemoryUsage  float64       `json:"maxMemoryUsage"`
	MaxConnFailRate float64       `json:"maxConnFailRate"`
}

// Additional types for enhanced performance monitoring dashboard

// MemoryMetrics represents detailed memory usage statistics
type MemoryMetrics struct {
	AllocatedBytes      uint64  `json:"allocatedBytes"`
	TotalAllocatedBytes uint64  `json:"totalAllocatedBytes"`
	SystemBytes         uint64  `json:"systemBytes"`
	HeapInUse           uint64  `json:"heapInUse"`
	HeapReleased        uint64  `json:"heapReleased"`
	GoroutineCount      int     `json:"goroutineCount"`
	MemoryUsagePercent  float64 `json:"memoryUsagePercent"`
}

// GCMetrics represents garbage collection performance data
type GCMetrics struct {
	NumGC       uint32        `json:"numGC"`
	PauseTotal  time.Duration `json:"pauseTotal"`
	PauseAvg    time.Duration `json:"pauseAvg"`
	LastPause   time.Duration `json:"lastPause"`
	CPUFraction float64       `json:"cpuFraction"`
}

// MessageThroughputMetrics represents message processing performance
type MessageThroughputMetrics struct {
	MessagesPerSecond     float64 `json:"messagesPerSecond"`
	TotalMessagesSent     int64   `json:"totalMessagesSent"`
	TotalMessagesReceived int64   `json:"totalMessagesReceived"`
	AverageLatency        float64 `json:"averageLatency"`
	BroadcastLatency      float64 `json:"broadcastLatency"`
}

// ConnectionAnalyticsSimplified represents simplified connection analytics matching game_server.go usage
type ConnectionAnalyticsSimplified struct {
	TotalPlayers      int                              `json:"totalPlayers"`
	ActivePlayers     int                              `json:"activePlayers"`
	PlayerSessions    []PlayerSessionMetricsSimplified `json:"playerSessions"`
	AverageLatency    float64                          `json:"averageLatency"`
	ConnectionsIn5Min int                              `json:"connectionsIn5Min"`
	DisconnectsIn5Min int                              `json:"disconnectsIn5Min"`
	ReconnectionRate  float64                          `json:"reconnectionRate"`
}

// PlayerSessionMetricsSimplified tracks individual player session data with
// a simplified structure optimized for real-time dashboard display.
type PlayerSessionMetricsSimplified struct {
	PlayerID         string        `json:"playerId"`
	SessionStart     time.Time     `json:"sessionStart"`
	SessionLength    time.Duration `json:"sessionLength"`
	ActionsPerformed int           `json:"actionsPerformed"`
	Reconnections    int           `json:"reconnections"`
	LastSeen         time.Time     `json:"lastSeen"`
	IsActive         bool          `json:"isActive"`
}

// ConnectionEventSimplified represents connection events with a simplified
// structure optimized for analytics tracking in the game server.
type ConnectionEventSimplified struct {
	EventType string    `json:"eventType"`
	PlayerID  string    `json:"playerId"`
	Timestamp time.Time `json:"timestamp"`
	Latency   float64   `json:"latency"`
}

// ConnectionQuality represents real-time connection quality metrics
type ConnectionQuality struct {
	LatencyMs    float64   `json:"latencyMs"`
	Quality      string    `json:"quality"` // "excellent", "good", "fair", "poor"
	PacketLoss   float64   `json:"packetLoss"`
	LastPingTime time.Time `json:"lastPingTime"`
	MessageDelay float64   `json:"messageDelay"`
}

// ConnectionStatusMessage represents connection quality updates sent to clients
type ConnectionStatusMessage struct {
	Type               string                       `json:"type"`
	PlayerID           string                       `json:"playerId"`
	Quality            ConnectionQuality            `json:"quality"`
	AllPlayerQualities map[string]ConnectionQuality `json:"allPlayerQualities"`
}

// PingMessage represents ping/pong messages for latency measurement
type PingMessage struct {
	Type      string    `json:"type"`
	PlayerID  string    `json:"playerId"`
	Timestamp time.Time `json:"timestamp"`
	PingID    string    `json:"pingId"`
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
