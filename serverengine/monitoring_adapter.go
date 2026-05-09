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
