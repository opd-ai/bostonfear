package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

// Provider supplies the engine snapshots and derived metrics required by the
// monitoring HTTP endpoints.
type Provider interface {
	SnapshotHealth() monitoringdata.HealthSnapshot
	CollectPerformanceMetrics() monitoringdata.PerformanceMetrics
	CollectConnectionAnalytics() monitoringdata.ConnectionAnalyticsSimplified
	CollectMemoryMetrics() monitoringdata.MemoryMetrics
	CollectGCMetrics() monitoringdata.GCMetrics
	CollectMessageThroughput(time.Duration) monitoringdata.MessageThroughputMetrics
	GameStatistics() map[string]interface{}
}

// HealthHandler serves a JSON health payload assembled from engine snapshots.
func HealthHandler(provider Provider) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := provider.SnapshotHealth()
		perfMetrics := provider.CollectPerformanceMetrics()
		connAnalytics := provider.CollectConnectionAnalytics()
		gameStats := provider.GameStatistics()
		alerts := BuildSystemAlerts(perfMetrics, snapshot.Doom)

		recentCorruptions := 0
		fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
		for _, eventTime := range snapshot.CorruptionHistory {
			if eventTime.After(fiveMinutesAgo) {
				recentCorruptions++
			}
		}

		status := "healthy"
		if !snapshot.IsHealthy {
			status = "degraded"
		}
		if recentCorruptions > 10 {
			status = "unhealthy"
		}

		healthData := map[string]interface{}{
			"status":              status,
			"gamePhase":           snapshot.GamePhase,
			"playerCount":         snapshot.PlayerCount,
			"connectionCount":     snapshot.ConnectionCount,
			"doomLevel":           snapshot.Doom,
			"gameStarted":         snapshot.GameStarted,
			"recentCorruptions":   recentCorruptions,
			"isGameStateHealthy":  snapshot.IsHealthy,
			"timestamp":           time.Now().Unix(),
			"performanceMetrics":  perfMetrics,
			"connectionAnalytics": connAnalytics,
			"gameStatistics":      gameStats,
			"systemAlerts":        alerts,
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(healthData) //nolint:errcheck
	})
}

// MetricsHandler serves Prometheus-compatible monitoring metrics.
func MetricsHandler(provider Provider) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot := provider.SnapshotHealth()
		perfMetrics := provider.CollectPerformanceMetrics()
		connAnalytics := provider.CollectConnectionAnalytics()
		memMetrics := provider.CollectMemoryMetrics()
		gcMetrics := provider.CollectGCMetrics()
		throughput := provider.CollectMessageThroughput(perfMetrics.Uptime)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		metrics := buildGameMetrics(perfMetrics, connAnalytics, throughput, snapshot.Doom) +
			buildMemoryMetrics(memMetrics, gcMetrics)
		fmt.Fprint(w, metrics)
	})
}

// DashboardHandler serves the static monitoring dashboard HTML.
func DashboardHandler(clientDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		http.ServeFile(w, r, clientDir+"/dashboard.html")
	})
}

func buildGameMetrics(perf monitoringdata.PerformanceMetrics, conn monitoringdata.ConnectionAnalyticsSimplified, throughput monitoringdata.MessageThroughputMetrics, doom int) string {
	lines := []string{
		"# HELP arkham_horror_uptime_seconds Total uptime of the server in seconds",
		"# TYPE arkham_horror_uptime_seconds counter",
		fmt.Sprintf("arkham_horror_uptime_seconds %.2f", perf.Uptime.Seconds()),
		"",
		"# HELP arkham_horror_active_connections Current number of active WebSocket connections",
		"# TYPE arkham_horror_active_connections gauge",
		fmt.Sprintf("arkham_horror_active_connections %d", perf.ActiveConnections),
		"",
		"# HELP arkham_horror_peak_connections Peak number of concurrent connections",
		"# TYPE arkham_horror_peak_connections gauge",
		fmt.Sprintf("arkham_horror_peak_connections %d", perf.PeakConnections),
		"",
		"# HELP arkham_horror_total_connections_total Total connections established since server start",
		"# TYPE arkham_horror_total_connections_total counter",
		fmt.Sprintf("arkham_horror_total_connections_total %d", perf.TotalConnections),
		"",
		"# HELP arkham_horror_connections_per_second Rate of new connections per second",
		"# TYPE arkham_horror_connections_per_second gauge",
		fmt.Sprintf("arkham_horror_connections_per_second %.2f", perf.ConnectionsPerSecond),
		"",
		"# HELP arkham_horror_active_players Current number of active players",
		"# TYPE arkham_horror_active_players gauge",
		fmt.Sprintf("arkham_horror_active_players %d", conn.ActivePlayers),
		"",
		"# HELP arkham_horror_messages_per_second Rate of messages processed per second",
		"# TYPE arkham_horror_messages_per_second gauge",
		fmt.Sprintf("arkham_horror_messages_per_second %.2f", perf.MessagesPerSecond),
		"",
		"# HELP arkham_horror_broadcast_latency_ms Rolling average broadcast write latency in milliseconds",
		"# TYPE arkham_horror_broadcast_latency_ms gauge",
		fmt.Sprintf("arkham_horror_broadcast_latency_ms %.4f", throughput.BroadcastLatency),
		"",
		"# HELP arkham_horror_response_time_ms Current health check response time in milliseconds",
		"# TYPE arkham_horror_response_time_ms gauge",
		fmt.Sprintf("arkham_horror_response_time_ms %.2f", perf.ResponseTimeMs),
		"",
		"# HELP arkham_horror_error_rate_percent Current error rate as percentage",
		"# TYPE arkham_horror_error_rate_percent gauge",
		fmt.Sprintf("arkham_horror_error_rate_percent %.2f", perf.ErrorRate),
		"",
		"# HELP arkham_horror_game_doom_level Current doom counter level",
		"# TYPE arkham_horror_game_doom_level gauge",
		fmt.Sprintf("arkham_horror_game_doom_level %d", doom),
		"",
		"# HELP arkham_horror_games_played_total Total number of games played",
		"# TYPE arkham_horror_games_played_total counter",
		fmt.Sprintf("arkham_horror_games_played_total %d", perf.TotalGamesPlayed),
		"",
		"# HELP arkham_horror_reconnection_rate_percent Player reconnection rate percentage",
		"# TYPE arkham_horror_reconnection_rate_percent gauge",
		fmt.Sprintf("arkham_horror_reconnection_rate_percent %.2f", conn.ReconnectionRate),
		"",
	}
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

func buildMemoryMetrics(mem monitoringdata.MemoryMetrics, gc monitoringdata.GCMetrics) string {
	lines := []string{
		"# HELP arkham_horror_memory_allocated_bytes Currently allocated memory in bytes",
		"# TYPE arkham_horror_memory_allocated_bytes gauge",
		fmt.Sprintf("arkham_horror_memory_allocated_bytes %d", mem.AllocatedBytes),
		"",
		"# HELP arkham_horror_memory_usage_percent Memory usage as percentage of allocated/system",
		"# TYPE arkham_horror_memory_usage_percent gauge",
		fmt.Sprintf("arkham_horror_memory_usage_percent %.2f", mem.MemoryUsagePercent),
		"",
		"# HELP arkham_horror_goroutines Current number of goroutines",
		"# TYPE arkham_horror_goroutines gauge",
		fmt.Sprintf("arkham_horror_goroutines %d", mem.GoroutineCount),
		"",
		"# HELP arkham_horror_gc_collections_total Total number of garbage collections",
		"# TYPE arkham_horror_gc_collections_total counter",
		fmt.Sprintf("arkham_horror_gc_collections_total %d", gc.NumGC),
		"",
		"# HELP arkham_horror_gc_pause_seconds_total Total time spent in garbage collection pauses",
		"# TYPE arkham_horror_gc_pause_seconds_total counter",
		fmt.Sprintf("arkham_horror_gc_pause_seconds_total %.6f", gc.PauseTotal.Seconds()),
		"",
	}
	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
