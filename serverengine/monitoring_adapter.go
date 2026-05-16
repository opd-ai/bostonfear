package serverengine

import "time"

// monitoring_adapter.go implements the monitoring.Provider interface for GameServer.

// CollectPerformanceMetrics exposes performance aggregation to the monitoring package.
func (gs *GameServer) CollectPerformanceMetrics() PerformanceMetrics {
	return gs.collectPerformanceMetrics()
}

// CollectConnectionAnalytics exposes connection analytics to the monitoring package.
func (gs *GameServer) CollectConnectionAnalytics() ConnectionAnalyticsSimplified {
	return gs.collectConnectionAnalytics()
}

// CollectMemoryMetrics exposes runtime memory metrics to the monitoring package.
func (gs *GameServer) CollectMemoryMetrics() MemoryMetrics {
	return gs.collectMemoryMetrics()
}

// CollectGCMetrics exposes garbage-collection metrics to the monitoring package.
func (gs *GameServer) CollectGCMetrics() GCMetrics {
	return gs.collectGCMetrics()
}

// CollectMessageThroughput exposes throughput metrics to the monitoring package.
func (gs *GameServer) CollectMessageThroughput(runtime time.Duration) MessageThroughputMetrics {
	return gs.collectMessageThroughput(runtime)
}

// GameStatistics exposes derived game-state analytics to the monitoring package.
func (gs *GameServer) GameStatistics() map[string]interface{} {
	return gs.getGameStatistics()
}

// GetActionTypeCounters exposes per-action type histogram to the monitoring package.
// Converts protocol.ActionType keys to string for JSON serialization.
func (gs *GameServer) GetActionTypeCounters() map[string]int64 {
	counters := gs.getActionTypeCounters()
	result := make(map[string]int64, len(counters))
	for k, v := range counters {
		result[string(k)] = v
	}
	return result
}

// GetDoomHistogram exposes doom level distribution to the monitoring package.
func (gs *GameServer) GetDoomHistogram() map[int]int64 {
	return gs.getDoomHistogram()
}

// GetLatencyPercentiles exposes broadcast latency percentiles to the monitoring package.
func (gs *GameServer) GetLatencyPercentiles() map[string]float64 {
	return gs.BroadcastLatencyPercentiles()
}
