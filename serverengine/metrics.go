// Package main implements Prometheus metrics export and performance data collection
// for the Arkham Horror multiplayer game server. This file handles all metric
// gathering, throughput analysis, broadcast latency tracking, and session analytics.
package serverengine

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

type PerformanceMetrics = monitoringdata.PerformanceMetrics
type MemoryStats = monitoringdata.MemoryStats
type MemoryMetrics = monitoringdata.MemoryMetrics
type GCMetrics = monitoringdata.GCMetrics
type MessageThroughputMetrics = monitoringdata.MessageThroughputMetrics
type ConnectionAnalyticsSimplified = monitoringdata.ConnectionAnalyticsSimplified
type PlayerSessionMetricsSimplified = monitoringdata.PlayerSessionMetricsSimplified
type ConnectionEventSimplified = monitoringdata.ConnectionEventSimplified

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
