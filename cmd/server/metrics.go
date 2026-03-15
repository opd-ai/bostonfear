// Package main implements Prometheus metrics export and performance data collection
// for the Arkham Horror multiplayer game server. This file handles all metric
// gathering, throughput analysis, broadcast latency tracking, and session analytics.
package main

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// handleMetrics provides Prometheus-compatible metrics export.
// Game state is snapshotted under a short RLock; serialization happens outside the lock.
func (gs *GameServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Snapshot the only game-state fields needed here under a single short read lock.
	// All helper calls that may themselves acquire gs.mutex happen AFTER this block.
	gs.mutex.RLock()
	doom := gs.gameState.Doom
	gs.mutex.RUnlock()

	uptime := time.Since(gs.startTime)
	perfMetrics := gs.collectPerformanceMetrics()
	connAnalytics := gs.collectConnectionAnalytics()
	memMetrics := gs.collectMemoryMetrics()
	gcMetrics := gs.collectGCMetrics()
	throughput := gs.collectMessageThroughput(uptime)

	// Set content type for Prometheus metrics
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Build Prometheus-compatible metrics output outside the lock.
	metrics := buildGameMetrics(perfMetrics, connAnalytics, throughput, doom) +
		buildMemoryMetrics(memMetrics, gcMetrics)

	fmt.Fprint(w, metrics)
}

// buildGameMetrics formats game and connection metrics in Prometheus text format.
func buildGameMetrics(perf PerformanceMetrics, conn ConnectionAnalyticsSimplified, throughput MessageThroughputMetrics, doom int) string {
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
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

// buildMemoryMetrics formats memory and GC metrics in Prometheus text format.
func buildMemoryMetrics(mem MemoryMetrics, gc GCMetrics) string {
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
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

// collectMemorySnapshot reads current Go runtime memory statistics.
// It is goroutine-safe and requires no lock.
func collectMemorySnapshot() MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return MemoryStats{
		AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(memStats.Sys) / 1024 / 1024,
		NumGC:        memStats.NumGC,
		GCPauseMs:    float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1000000,
	}
}

// aggregateSessionMetrics summarises player session data held in gs.playerSessions.
// Caller must hold gs.performanceMutex (at least for reading).
func (gs *GameServer) aggregateSessionMetrics() (activeSessions int, avgSessionLength time.Duration) {
	var total time.Duration
	for _, session := range gs.playerSessions {
		total += time.Since(session.SessionStart)
		activeSessions++
	}
	if activeSessions > 0 {
		avgSessionLength = total / time.Duration(activeSessions)
	}
	return activeSessions, avgSessionLength
}

// collectPerformanceMetrics gathers comprehensive server performance data
func (gs *GameServer) collectPerformanceMetrics() PerformanceMetrics {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	uptime := time.Since(gs.startTime)
	activeConnections := int(atomic.LoadInt64(&gs.activeConnections))

	connectionsPerSecond := 0.0
	if uptime.Seconds() > 0 {
		connectionsPerSecond = float64(gs.totalConnections) / uptime.Seconds()
	}
	messagesPerSecond := 0.0
	if uptime.Seconds() > 0 {
		messagesPerSecond = float64(gs.totalMessagesSent+gs.totalMessagesRecv) / uptime.Seconds()
	}

	activeSessions, avgSessionLength := gs.aggregateSessionMetrics()

	return PerformanceMetrics{
		Uptime:               uptime,
		ActiveConnections:    activeConnections,
		PeakConnections:      gs.peakConnections,
		TotalConnections:     gs.totalConnections,
		ConnectionsPerSecond: connectionsPerSecond,
		AverageSessionLength: avgSessionLength,
		ActiveSessions:       activeSessions,
		TotalGamesPlayed:     gs.totalGamesPlayed,
		MessagesPerSecond:    messagesPerSecond,
		MemoryUsage:          collectMemorySnapshot(),
		ResponseTimeMs:       gs.measureHealthCheckResponseTime(),
		ErrorRate:            gs.calculateErrorRate(),
	}
}

// collectConnectionAnalytics provides player connection insights
func (gs *GameServer) collectConnectionAnalytics() ConnectionAnalyticsSimplified {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	totalPlayers, activePlayers, playerSessions := gs.aggregatePlayerSessions()
	window := time.Now().Add(-5 * time.Minute)
	connectionsIn5Min, disconnectsIn5Min, totalReconnections := gs.countRecentConnectionEvents(window)

	var reconnectionRate float64
	if connectionsIn5Min > 0 {
		reconnectionRate = float64(totalReconnections) / float64(connectionsIn5Min) * 100
	}

	return ConnectionAnalyticsSimplified{
		TotalPlayers:      totalPlayers,
		ActivePlayers:     activePlayers,
		PlayerSessions:    playerSessions,
		AverageLatency:    gs.computeAverageLatency(window),
		ConnectionsIn5Min: connectionsIn5Min,
		DisconnectsIn5Min: disconnectsIn5Min,
		ReconnectionRate:  reconnectionRate,
	}
}

// aggregatePlayerSessions converts the playerSessions map into a slice and counts active players.
// Caller must hold gs.performanceMutex at least for reading.
func (gs *GameServer) aggregatePlayerSessions() (int, int, []PlayerSessionMetricsSimplified) {
	total := len(gs.playerSessions)
	active := 0
	sessions := make([]PlayerSessionMetricsSimplified, 0, total)
	for _, session := range gs.playerSessions {
		if session.IsActive {
			session.SessionLength = time.Since(session.SessionStart)
			active++
		}
		sessions = append(sessions, *session)
	}
	return total, active, sessions
}

// countRecentConnectionEvents counts connect, disconnect, and reconnect events after cutoff.
// Caller must hold gs.performanceMutex at least for reading.
func (gs *GameServer) countRecentConnectionEvents(after time.Time) (connects, disconnects, reconnects int) {
	for _, event := range gs.connectionEvents {
		if !event.Timestamp.After(after) {
			continue
		}
		switch event.EventType {
		case "connect":
			connects++
		case "disconnect":
			disconnects++
		case "reconnect":
			reconnects++
		}
	}
	return connects, disconnects, reconnects
}

// computeAverageLatency returns the mean latency of events with Latency > 0 after cutoff.
// Caller must hold gs.performanceMutex at least for reading.
func (gs *GameServer) computeAverageLatency(after time.Time) float64 {
	var total float64
	count := 0
	for _, event := range gs.connectionEvents {
		if event.Latency > 0 && event.Timestamp.After(after) {
			total += event.Latency
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// collectMemoryMetrics gathers memory usage statistics
func (gs *GameServer) collectMemoryMetrics() MemoryMetrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// Calculate memory usage percentage (approximate)
	memUsagePercent := float64(ms.Alloc) / float64(ms.Sys) * 100
	if memUsagePercent > 100 {
		memUsagePercent = 100
	}

	return MemoryMetrics{
		AllocatedBytes:      ms.Alloc,
		TotalAllocatedBytes: ms.TotalAlloc,
		SystemBytes:         ms.Sys,
		HeapInUse:           ms.HeapInuse,
		HeapReleased:        ms.HeapReleased,
		GoroutineCount:      runtime.NumGoroutine(),
		MemoryUsagePercent:  memUsagePercent,
	}
}

// collectGCMetrics gathers garbage collection performance data
func (gs *GameServer) collectGCMetrics() GCMetrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// Calculate average pause time
	var avgPause time.Duration
	if ms.NumGC > 0 && len(ms.PauseNs) > 0 {
		var totalPause uint64
		recentPauses := int(ms.NumGC)
		if recentPauses > len(ms.PauseNs) {
			recentPauses = len(ms.PauseNs)
		}

		for i := 0; i < recentPauses; i++ {
			totalPause += ms.PauseNs[i]
		}
		avgPause = time.Duration(totalPause / uint64(recentPauses))
	}

	// Get last pause time
	var lastPause time.Duration
	if ms.NumGC > 0 {
		lastPause = time.Duration(ms.PauseNs[(ms.NumGC+255)%256])
	}

	return GCMetrics{
		NumGC:       ms.NumGC,
		PauseTotal:  time.Duration(ms.PauseTotalNs),
		PauseAvg:    avgPause,
		LastPause:   lastPause,
		CPUFraction: ms.GCCPUFraction,
	}
}

// recordBroadcastLatency stores a single write-duration sample in the ring buffer.
func (gs *GameServer) recordBroadcastLatency(d time.Duration) {
	gs.latencyMu.Lock()
	gs.latencySamples[gs.latencyHead] = d.Nanoseconds()
	gs.latencyHead = (gs.latencyHead + 1) % len(gs.latencySamples)
	if gs.latencySampleCount < len(gs.latencySamples) {
		gs.latencySampleCount++
	}
	gs.latencyMu.Unlock()
}

// averageBroadcastLatencyMs returns the rolling average broadcast latency in milliseconds.
func (gs *GameServer) averageBroadcastLatencyMs() float64 {
	gs.latencyMu.Lock()
	defer gs.latencyMu.Unlock()
	if gs.latencySampleCount == 0 {
		return 0
	}
	var sum int64
	for i := 0; i < gs.latencySampleCount; i++ {
		sum += gs.latencySamples[i]
	}
	return float64(sum) / float64(gs.latencySampleCount) / 1e6
}

// collectMessageThroughput calculates message performance metrics
func (gs *GameServer) collectMessageThroughput(runtime time.Duration) MessageThroughputMetrics {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	// Calculate messages per second — guard against zero uptime on startup
	totalMessages := gs.totalMessagesSent + gs.totalMessagesRecv
	messagesPerSecond := 0.0
	if runtime.Seconds() > 0 {
		messagesPerSecond = float64(totalMessages) / runtime.Seconds()
	}

	broadcastLatency := gs.averageBroadcastLatencyMs()
	return MessageThroughputMetrics{
		MessagesPerSecond:     messagesPerSecond,
		TotalMessagesSent:     gs.totalMessagesSent,
		TotalMessagesReceived: gs.totalMessagesRecv,
		AverageLatency:        broadcastLatency,
		BroadcastLatency:      broadcastLatency,
	}
}

// trackConnection records connection events for analytics
func (gs *GameServer) trackConnection(eventType, playerID string, latency float64) {
	gs.performanceMutex.Lock()
	defer gs.performanceMutex.Unlock()

	event := ConnectionEventSimplified{
		EventType: eventType,
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Latency:   latency,
	}

	gs.connectionEvents = append(gs.connectionEvents, event)

	// Keep only last 1000 events to prevent memory growth
	if len(gs.connectionEvents) > 1000 {
		gs.connectionEvents = gs.connectionEvents[len(gs.connectionEvents)-1000:]
	}

	// Update connection counters
	if eventType == "connect" {
		gs.totalConnections++
		currentConnections := int(atomic.LoadInt64(&gs.activeConnections))
		if currentConnections > gs.peakConnections {
			gs.peakConnections = currentConnections
		}
	}
}

// trackPlayerSession manages player session metrics
func (gs *GameServer) trackPlayerSession(playerID, eventType string) {
	gs.performanceMutex.Lock()
	defer gs.performanceMutex.Unlock()

	switch eventType {
	case "start":
		gs.playerSessions[playerID] = &PlayerSessionMetricsSimplified{
			PlayerID:         playerID,
			SessionStart:     time.Now(),
			SessionLength:    0,
			ActionsPerformed: 0,
			Reconnections:    0,
			LastSeen:         time.Now(),
			IsActive:         true,
		}
	case "end":
		if session, exists := gs.playerSessions[playerID]; exists {
			session.SessionLength = time.Since(session.SessionStart)
			session.IsActive = false
		}
	case "action":
		if session, exists := gs.playerSessions[playerID]; exists {
			session.ActionsPerformed++
			session.LastSeen = time.Now()
		}
	case "reconnect":
		if session, exists := gs.playerSessions[playerID]; exists {
			session.Reconnections++
			session.LastSeen = time.Now()
			session.IsActive = true
		}
	}
}

// trackMessage increments message counters for throughput analysis
func (gs *GameServer) trackMessage(messageType string) {
	gs.performanceMutex.Lock()
	defer gs.performanceMutex.Unlock()

	switch messageType {
	case "sent":
		gs.totalMessagesSent++
	case "received":
		gs.totalMessagesRecv++
	}
}
