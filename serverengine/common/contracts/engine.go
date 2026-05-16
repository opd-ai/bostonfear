package contracts

import (
	"net"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

// GameRunner defines the core gameplay startup and shutdown interface.
// Methods must be safe for concurrent calls from multiple goroutines.
// Intended for server initialization and lifecycle management.
type GameRunner interface {
	// Start initializes the game engine, begins the main game loop (if applicable),
	// and starts all internal handlers (action processor, broadcaster, mythos phase).
	// Calling Start multiple times on the same engine is undefined behavior.
	Start() error
}

// SessionHandler defines the connection and player session interface.
// Methods must be safe for concurrent calls from multiple goroutines.
// Intended for transport layer integration (WebSocket, TCP, in-process protocols).
type SessionHandler interface {
	// HandleConnection manages a player session via the provided net.Conn.
	// conn must be non-nil and readable/writable. reconnectToken may be empty (new player)
	// or non-empty (attempt to restore disconnected player). The method blocks until
	// the connection closes or an error occurs, then returns. Callers should run this
	// in a goroutine per connection.
	HandleConnection(conn net.Conn, reconnectToken string) error

	// SetAllowedOrigins configures the list of permitted WebSocket upgrade origins.
	// After SetAllowedOrigins, AllowedOrigins must return the configured list.
	// This method is safe to call concurrently with AllowedOrigins and HandleConnection.
	SetAllowedOrigins(origins []string)

	// AllowedOrigins returns the current list of permitted WebSocket upgrade origins.
	// An empty or nil list indicates permissive mode (any origin accepted).
	AllowedOrigins() []string
}

// HealthChecker defines the health monitoring interface.
// Implementations must be safe for concurrent calls and return consistent snapshots.
// Intended for monitoring handlers and health probes.
type HealthChecker interface {
	// SnapshotHealth returns a point-in-time health snapshot of the engine,
	// including goroutine count, connection count, and any detected anomalies.
	// The snapshot's IsHealthy field is true if no critical issues are detected.
	SnapshotHealth() monitoringdata.HealthSnapshot
}

// MetricsCollector defines the performance and analytics interface.
// All methods must be safe for concurrent calls and should return in < 5ms.
// Intended for HTTP metrics endpoints and periodic monitoring polls.
type MetricsCollector interface {
	// CollectPerformanceMetrics returns aggregate server performance since startup:
	// uptime, active/peak connections, message rates, memory usage, etc.
	CollectPerformanceMetrics() monitoringdata.PerformanceMetrics

	// CollectConnectionAnalytics returns connection-level insights: total players,
	// active players, recent reconnection rates, and average latency over 5 minutes.
	CollectConnectionAnalytics() monitoringdata.ConnectionAnalyticsSimplified

	// CollectMemoryMetrics returns Go runtime memory statistics: heap usage,
	// goroutine count, and memory pressure percentage.
	CollectMemoryMetrics() monitoringdata.MemoryMetrics

	// CollectGCMetrics returns garbage collection statistics: GC count, pause times,
	// and CPU fraction spent in GC.
	CollectGCMetrics() monitoringdata.GCMetrics

	// CollectMessageThroughput returns message throughput and latency statistics
	// over the provided runtime duration (e.g., since server start or last 5 minutes).
	CollectMessageThroughput(runtime time.Duration) monitoringdata.MessageThroughputMetrics

	// GameStatistics returns a map of game-family-specific statistics (e.g., scenario progress,
	// investigator details, turn count). The map is game-specific and may be empty for
	// unimplemented or placeholder engines. Keys and values are not version-stable.
	GameStatistics() map[string]interface{}

	// GetActionTypeCounters returns per-action type histogram: total count of each action
	// performed since server start. Action types are returned as strings (e.g., "move", "investigate").
	// Used by Prometheus metrics to expose per-action counters.
	GetActionTypeCounters() map[string]int64

	// GetDoomHistogram returns doom level distribution: total count of games ending at each
	// doom level (0-12). Used by Prometheus metrics to track doom progression patterns.
	GetDoomHistogram() map[int]int64

	// GetLatencyPercentiles returns broadcast latency percentiles (P50, P90, P95, P99) in milliseconds.
	// Keys are "p50", "p90", "p95", "p99". Used by Prometheus metrics for latency monitoring.
	GetLatencyPercentiles() map[string]float64
}

// Engine defines the transport-neutral runtime surface required by server startup,
// transport adapters, and monitoring handlers. An Engine implementation must satisfy
// all of the following role-based interfaces:
//
//   - GameRunner: Core gameplay startup
//   - SessionHandler: Player connection and origin validation
//   - HealthChecker: Health snapshots
//   - MetricsCollector: Performance analytics and statistics
//
// This union interface is provided for convenience; implementers may also implement
// each role interface independently for dependency injection with only needed methods.
//
// Implementations must be safe for concurrent use by multiple goroutines.
// Methods must not block indefinitely (max 5s for I/O, max 5ms for metric collection).
//
// Example implementation note: A minimal mock for testing needs only to implement
// methods the test exercises; use a partial mock library or define a separate
// interface with only required methods rather than implementing all 11 methods.
type Engine interface {
	GameRunner
	SessionHandler
	HealthChecker
	MetricsCollector
}
