package monitoringdata

import "time"

// HealthSnapshot is the lock-bounded subset of state needed to assemble the
// health response outside the core engine package.
type HealthSnapshot struct {
	IsHealthy         bool
	PlayerCount       int
	ConnectionCount   int
	CorruptionHistory []time.Time
	GamePhase         string
	Doom              int
	GameStarted       bool
}

// PerformanceMetrics represents comprehensive server performance data.
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

// MemoryStats represents memory usage statistics for performance monitoring.
type MemoryStats struct {
	AllocMB      float64 `json:"allocMB"`
	TotalAllocMB float64 `json:"totalAllocMB"`
	SysMB        float64 `json:"sysMB"`
	NumGC        uint32  `json:"numGC"`
	GCPauseMs    float64 `json:"gcPauseMs"`
}

// MemoryMetrics represents detailed memory usage statistics.
type MemoryMetrics struct {
	AllocatedBytes      uint64  `json:"allocatedBytes"`
	TotalAllocatedBytes uint64  `json:"totalAllocatedBytes"`
	SystemBytes         uint64  `json:"systemBytes"`
	HeapInUse           uint64  `json:"heapInUse"`
	HeapReleased        uint64  `json:"heapReleased"`
	GoroutineCount      int     `json:"goroutineCount"`
	MemoryUsagePercent  float64 `json:"memoryUsagePercent"`
}

// GCMetrics represents garbage collection performance data.
type GCMetrics struct {
	NumGC       uint32        `json:"numGC"`
	PauseTotal  time.Duration `json:"pauseTotal"`
	PauseAvg    time.Duration `json:"pauseAvg"`
	LastPause   time.Duration `json:"lastPause"`
	CPUFraction float64       `json:"cpuFraction"`
}

// MessageThroughputMetrics represents message processing performance.
type MessageThroughputMetrics struct {
	MessagesPerSecond     float64 `json:"messagesPerSecond"`
	TotalMessagesSent     int64   `json:"totalMessagesSent"`
	TotalMessagesReceived int64   `json:"totalMessagesReceived"`
	AverageLatency        float64 `json:"averageLatency"`
	BroadcastLatency      float64 `json:"broadcastLatency"`
}

// ConnectionAnalyticsSimplified represents simplified connection analytics.
type ConnectionAnalyticsSimplified struct {
	TotalPlayers      int                              `json:"totalPlayers"`
	ActivePlayers     int                              `json:"activePlayers"`
	PlayerSessions    []PlayerSessionMetricsSimplified `json:"playerSessions"`
	AverageLatency    float64                          `json:"averageLatency"`
	ConnectionsIn5Min int                              `json:"connectionsIn5Min"`
	DisconnectsIn5Min int                              `json:"disconnectsIn5Min"`
	ReconnectionRate  float64                          `json:"reconnectionRate"`
}

// PlayerSessionMetricsSimplified tracks individual player session data.
type PlayerSessionMetricsSimplified struct {
	PlayerID         string        `json:"playerId"`
	SessionStart     time.Time     `json:"sessionStart"`
	SessionLength    time.Duration `json:"sessionLength"`
	ActionsPerformed int           `json:"actionsPerformed"`
	Reconnections    int           `json:"reconnections"`
	LastSeen         time.Time     `json:"lastSeen"`
	IsActive         bool          `json:"isActive"`
}

// ConnectionEventSimplified represents connection events for analytics tracking.
type ConnectionEventSimplified struct {
	EventType string    `json:"eventType"`
	PlayerID  string    `json:"playerId"`
	Timestamp time.Time `json:"timestamp"`
	Latency   float64   `json:"latency"`
}
