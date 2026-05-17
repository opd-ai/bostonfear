// Package monitoring provides shared monitoring DTO and health payload helpers.
// This is distinct from the root package monitoring HTTP handlers.
package monitoring

import (
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

// CountRecentCorruptions returns how many corruption timestamps fall within the
// recent window ending at now.
func CountRecentCorruptions(corruptionHistory []time.Time, now time.Time, window time.Duration) int {
	cutoff := now.Add(-window)
	count := 0
	for _, eventTime := range corruptionHistory {
		if eventTime.After(cutoff) {
			count++
		}
	}
	return count
}

// DeriveHealthStatus maps health inputs into the canonical status string.
func DeriveHealthStatus(isGameStateHealthy bool, recentCorruptions int) string {
	if recentCorruptions > 10 {
		return "unhealthy"
	}
	if !isGameStateHealthy {
		return "degraded"
	}
	return "healthy"
}

// BuildHealthPayload assembles a stable health JSON payload map consumed by
// HTTP handlers and tests.
func BuildHealthPayload(
	status string,
	snapshot monitoringdata.HealthSnapshot,
	perf monitoringdata.PerformanceMetrics,
	conn monitoringdata.ConnectionAnalyticsSimplified,
	gameStats map[string]interface{},
	alerts []map[string]interface{},
	timestampUnix int64,
	recentCorruptions int,
) map[string]interface{} {
	return map[string]interface{}{
		"status":              status,
		"gamePhase":           snapshot.GamePhase,
		"playerCount":         snapshot.PlayerCount,
		"connectionCount":     snapshot.ConnectionCount,
		"doomLevel":           snapshot.Doom,
		"gameStarted":         snapshot.GameStarted,
		"recentCorruptions":   recentCorruptions,
		"isGameStateHealthy":  snapshot.IsHealthy,
		"timestamp":           timestampUnix,
		"performanceMetrics":  perf,
		"connectionAnalytics": conn,
		"gameStatistics":      gameStats,
		"systemAlerts":        alerts,
	}
}
