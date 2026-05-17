package runtime

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// UnimplementedEngine is a placeholder implementation used for game families
// that have package scaffolding but no playable runtime yet.
type UnimplementedEngine struct {
	gameName string
	mu       sync.Mutex
	origins  []string
}

// NewUnimplementedEngine returns a placeholder engine for a game family.
func NewUnimplementedEngine(gameName string) contracts.Engine {
	return &UnimplementedEngine{gameName: gameName}
}

func (e *UnimplementedEngine) Start() error {
	return fmt.Errorf("%s engine not implemented", e.gameName)
}

// HandleConnection satisfies the SessionHandler interface. Like Start, it always
// returns a "not implemented" error — no player session will ever be established
// for an unimplemented engine.
func (e *UnimplementedEngine) HandleConnection(_ net.Conn, _ string) error {
	return fmt.Errorf("%s engine not implemented", e.gameName)
}

// SetAllowedOrigins stores the allowed origins for contract compliance,
// though they are not used in practice since Start() always fails with
// "game not implemented". This method is provided to satisfy the SessionHandler
// interface but does not offer the filtering semantics that production engines provide.
func (e *UnimplementedEngine) SetAllowedOrigins(origins []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		if trimmed := strings.TrimSpace(strings.ToLower(origin)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	e.origins = normalized
}

// AllowedOrigins returns the stored list of allowed origins, or nil if none
// were configured. Note that as a placeholder engine, this does not provide
// actual origin filtering semantics - Start() always fails before origins are validated.
// This method exists to satisfy the SessionHandler interface contract.
func (e *UnimplementedEngine) AllowedOrigins() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.origins) == 0 {
		return nil
	}

	return append([]string(nil), e.origins...)
}

// SnapshotHealth returns an unhealthy snapshot. The engine is intentionally
// unhealthy since it is not implemented; callers can use IsHealthy==false as
// a signal that the selected game module was not yet implemented.
func (e *UnimplementedEngine) SnapshotHealth() monitoringdata.HealthSnapshot {
	return monitoringdata.HealthSnapshot{IsHealthy: false}
}

// CollectPerformanceMetrics returns zero-value metrics. No gameplay occurs on
// an unimplemented engine, so no performance data is ever collected.
func (e *UnimplementedEngine) CollectPerformanceMetrics() monitoringdata.PerformanceMetrics {
	return monitoringdata.PerformanceMetrics{}
}

// CollectConnectionAnalytics returns zero-value analytics for the same reason as
// CollectPerformanceMetrics — no connections are ever accepted.
func (e *UnimplementedEngine) CollectConnectionAnalytics() monitoringdata.ConnectionAnalyticsSimplified {
	return monitoringdata.ConnectionAnalyticsSimplified{}
}

// CollectMemoryMetrics returns zero-value metrics. Memory usage is not tracked for
// engines that never start.
func (e *UnimplementedEngine) CollectMemoryMetrics() monitoringdata.MemoryMetrics {
	return monitoringdata.MemoryMetrics{}
}

// CollectGCMetrics returns zero-value GC metrics.
func (e *UnimplementedEngine) CollectGCMetrics() monitoringdata.GCMetrics {
	return monitoringdata.GCMetrics{}
}

// CollectMessageThroughput returns zero-value throughput metrics.
func (e *UnimplementedEngine) CollectMessageThroughput(_ time.Duration) monitoringdata.MessageThroughputMetrics {
	return monitoringdata.MessageThroughputMetrics{}
}

// GameStatistics returns a map indicating the game is not implemented along with
// the registered game name, useful for operator dashboards to detect misconfiguration.
func (e *UnimplementedEngine) GameStatistics() map[string]interface{} {
	return map[string]interface{}{
		"status": "not_implemented",
		"game":   e.gameName,
	}
}

// GetActionTypeCounters returns an empty map. No actions are ever dispatched.
func (e *UnimplementedEngine) GetActionTypeCounters() map[string]int64 {
	return map[string]int64{}
}

// GetDoomHistogram returns an empty map. No doom increments occur.
func (e *UnimplementedEngine) GetDoomHistogram() map[int]int64 {
	return map[int]int64{}
}

// GetLatencyPercentiles returns zero percentiles. No action processing occurs.
func (e *UnimplementedEngine) GetLatencyPercentiles() map[string]float64 {
	return map[string]float64{"p50": 0, "p95": 0, "p99": 0}
}
