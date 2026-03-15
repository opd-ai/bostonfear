// Package main contains observability and monitoring methods for the Arkham Horror
// multiplayer game server. This file implements health checks, Prometheus-compatible
// metrics export, performance monitoring, connection quality tracking, and
// the game statistics dashboard.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// handleHealthCheck provides a health monitoring endpoint.
// Game state is snapshotted under a short RLock; serialization happens outside the lock.
func (gs *GameServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Snapshot required fields under a short read lock.
	gs.mutex.RLock()
	isHealthy := gs.validator.IsGameStateHealthy(gs.gameState)
	playerCount := len(gs.gameState.Players)
	connectionCount := len(gs.connections)
	corruptionHistory := gs.validator.GetCorruptionHistory()
	gamePhase := gs.gameState.GamePhase
	doom := gs.gameState.Doom
	gameStarted := gs.gameState.GameStarted
	perfMetrics := gs.collectPerformanceMetrics()
	connAnalytics := gs.collectConnectionAnalytics()
	gameStats := gs.getGameStatistics()
	alerts := gs.getSystemAlerts()
	gs.mutex.RUnlock()

	// Compute derived fields outside the lock.
	recentCorruptions := 0
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	for _, event := range corruptionHistory {
		if event.Timestamp.After(fiveMinutesAgo) {
			recentCorruptions++
		}
	}

	status := "healthy"
	if !isHealthy {
		status = "degraded"
	}
	if recentCorruptions > 10 {
		status = "unhealthy"
	}

	healthData := map[string]interface{}{
		"status":             status,
		"gamePhase":          gamePhase,
		"playerCount":        playerCount,
		"connectionCount":    connectionCount,
		"doomLevel":          doom,
		"gameStarted":        gameStarted,
		"recentCorruptions":  recentCorruptions,
		"isGameStateHealthy": isHealthy,
		"timestamp":          time.Now().Unix(),

		// Enhanced performance metrics
		"performanceMetrics":  perfMetrics,
		"connectionAnalytics": connAnalytics,
		"gameStatistics":      gameStats,

		// System alerts: high memory, slow response, high error rate, critical doom
		"systemAlerts": alerts,
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(healthData)
}

// handleDashboard serves the performance monitoring dashboard
func (gs *GameServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for dashboard access
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Serve the dashboard HTML file using the package-level clientDir constant
	http.ServeFile(w, r, clientDir+"/dashboard.html")
}

// handleMetrics provides Prometheus-compatible metrics export.
// Game state is snapshotted under a short RLock; serialization happens outside the lock.
func (gs *GameServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Snapshot required fields under a short read lock.
	gs.mutex.RLock()
	doom := gs.gameState.Doom
	uptime := time.Since(gs.startTime)
	perfMetrics := gs.collectPerformanceMetrics()
	connAnalytics := gs.collectConnectionAnalytics()
	memMetrics := gs.collectMemoryMetrics()
	gcMetrics := gs.collectGCMetrics()
	throughput := gs.collectMessageThroughput(uptime)
	gs.mutex.RUnlock()

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

// collectPerformanceMetrics gathers comprehensive server performance data
func (gs *GameServer) collectPerformanceMetrics() PerformanceMetrics {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	// Calculate runtime metrics
	uptime := time.Since(gs.startTime)
	activeConnections := int(atomic.LoadInt64(&gs.activeConnections))

	// Calculate connections per second — guard against division by zero on startup
	connectionsPerSecond := 0.0
	if uptime.Seconds() > 0 {
		connectionsPerSecond = float64(gs.totalConnections) / uptime.Seconds()
	}

	// Calculate average session length and active sessions
	var totalSessionTime time.Duration
	activeSessions := 0
	for _, session := range gs.playerSessions {
		sessionDuration := time.Since(session.SessionStart)
		totalSessionTime += sessionDuration
		activeSessions++
	}

	var avgSessionLength time.Duration
	if len(gs.playerSessions) > 0 {
		avgSessionLength = totalSessionTime / time.Duration(len(gs.playerSessions))
	}

	// Calculate messages per second — guard against division by zero on startup
	messagesPerSecond := 0.0
	if uptime.Seconds() > 0 {
		messagesPerSecond = float64(gs.totalMessagesSent+gs.totalMessagesRecv) / uptime.Seconds()
	}

	// Collect memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryStats := MemoryStats{
		AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(memStats.Sys) / 1024 / 1024,
		NumGC:        memStats.NumGC,
		GCPauseMs:    float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1000000,
	}

	// Calculate response time (simplified - using health check measurement)
	responseTimeMs := gs.measureHealthCheckResponseTime()

	// Calculate error rate (corruption events vs total operations)
	errorRate := gs.calculateErrorRate()

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
		MemoryUsage:          memoryStats,
		ResponseTimeMs:       responseTimeMs,
		ErrorRate:            errorRate,
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

// validateAndRecoverState validates the current game state and attempts recovery when
// critical or high-severity errors are found. Caller must hold gs.mutex.
func (gs *GameServer) validateAndRecoverState() {
	errors := gs.validator.ValidateGameState(gs.gameState)
	if len(errors) == 0 {
		return
	}
	log.Printf("Game state validation errors detected: %d errors", len(errors))

	for _, err := range errors {
		if err.Severity == "CRITICAL" || err.Severity == "HIGH" {
			log.Printf("Attempting game state recovery...")
			recovered, recoveryErr := gs.validator.RecoverGameState(gs.gameState, errors)
			if recoveryErr == nil {
				gs.gameState = recovered
				log.Printf("Game state successfully recovered")
			} else {
				log.Printf("Game state recovery failed: %v", recoveryErr)
				atomic.AddInt64(&gs.errorCount, 1)
			}
			return
		}
	}
}

// Helper methods for performance monitoring dashboard

// measureHealthCheckResponseTime measures the response time of health check operations
func (gs *GameServer) measureHealthCheckResponseTime() float64 {
	start := time.Now()

	// Simulate health check operations
	gs.mutex.RLock()
	_ = len(gs.gameState.Players)
	_ = len(gs.connections)
	gs.mutex.RUnlock()

	// Return response time in milliseconds
	return float64(time.Since(start).Nanoseconds()) / 1000000
}

// calculateErrorRate calculates the current error rate as a percentage of
// error events relative to total messages received. The errorCount field is
// incremented atomically at every error site (upgrade failures, unmarshal
// errors, invalid actions, and state recovery failures).
func (gs *GameServer) calculateErrorRate() float64 {
	errors := atomic.LoadInt64(&gs.errorCount)
	total := atomic.LoadInt64(&gs.totalMessagesRecv)
	if total == 0 {
		return 0.0
	}
	return float64(errors) / float64(total) * 100
}

// Connection Quality Management Methods

// initializeConnectionQuality sets up initial connection quality for a player
func (gs *GameServer) initializeConnectionQuality(playerID string) {
	gs.qualityMutex.Lock()
	defer gs.qualityMutex.Unlock()

	gs.connectionQualities[playerID] = &ConnectionQuality{
		LatencyMs:    0,
		Quality:      "unknown",
		PacketLoss:   0,
		LastPingTime: time.Now(),
		MessageDelay: 0,
	}

	// Start ping timer for this player
	gs.startPingTimer(playerID)
}

// updateConnectionQuality updates connection quality metrics based on message timing
func (gs *GameServer) updateConnectionQuality(playerID string, messageTime time.Time) {
	gs.qualityMutex.Lock()
	defer gs.qualityMutex.Unlock()

	quality, exists := gs.connectionQualities[playerID]
	if !exists {
		return
	}

	// Calculate message delay (simplified metric)
	now := time.Now()
	quality.MessageDelay = float64(now.Sub(messageTime).Nanoseconds()) / 1000000 // Convert to milliseconds

	// Update quality assessment based on current metrics
	gs.assessConnectionQuality(playerID)
}

// handlePongMessage processes pong responses and calculates latency.
// The write lock is released before calling broadcastConnectionQuality to
// prevent a deadlock: broadcastConnectionQuality acquires qualityMutex.RLock,
// and Go's sync.RWMutex is not reentrant.
func (gs *GameServer) handlePongMessage(pingMsg PingMessage, receiveTime time.Time) {
	gs.qualityMutex.Lock()
	quality, exists := gs.connectionQualities[pingMsg.PlayerID]
	if !exists {
		gs.qualityMutex.Unlock()
		return
	}

	// Calculate round-trip latency in milliseconds
	latency := float64(receiveTime.Sub(pingMsg.Timestamp).Nanoseconds()) / 1e6
	quality.LatencyMs = latency
	quality.LastPingTime = receiveTime

	// Update quality assessment while still holding the lock
	gs.assessConnectionQuality(pingMsg.PlayerID)
	gs.qualityMutex.Unlock() // release before broadcasting to avoid reentrant lock

	// Broadcast quality update to all clients
	gs.broadcastConnectionQuality()
}

// assessConnectionQuality determines connection quality rating based on metrics
func (gs *GameServer) assessConnectionQuality(playerID string) {
	quality := gs.connectionQualities[playerID]

	// Assess quality based on latency
	switch {
	case quality.LatencyMs < 50:
		quality.Quality = "excellent"
	case quality.LatencyMs < 100:
		quality.Quality = "good"
	case quality.LatencyMs < 200:
		quality.Quality = "fair"
	default:
		quality.Quality = "poor"
	}

	// Factor in packet loss (simplified - would need more sophisticated tracking)
	if quality.PacketLoss > 0.05 { // 5% packet loss threshold
		if quality.Quality == "excellent" {
			quality.Quality = "good"
		} else if quality.Quality == "good" {
			quality.Quality = "fair"
		} else if quality.Quality == "fair" {
			quality.Quality = "poor"
		}
	}
}

// startPingTimer starts periodic ping for connection quality monitoring
func (gs *GameServer) startPingTimer(playerID string) {
	timer := time.NewTimer(5 * time.Second) // Ping every 5 seconds
	gs.pingTimers[playerID] = timer

	go func() {
		for {
			select {
			case <-timer.C:
				gs.sendPingToPlayer(playerID)
				timer.Reset(5 * time.Second)
			case <-gs.shutdownCh:
				timer.Stop()
				return
			}
		}
	}()
}

// sendPingToPlayer sends a ping message to measure latency.
// Guards against nil connections that can appear when a concurrent disconnect
// cleanup removes playerConns[playerID] while this function is running.
func (gs *GameServer) sendPingToPlayer(playerID string) {
	gs.mutex.RLock()
	conn, connExists := gs.playerConns[playerID]
	var wsConn *websocket.Conn
	var wsExists bool
	if connExists && conn != nil {
		wsConn, wsExists = gs.wsConns[conn.RemoteAddr().String()]
	}
	gs.mutex.RUnlock()

	if !connExists || conn == nil || !wsExists {
		return
	}

	pingMsg := PingMessage{
		Type:      "ping",
		PlayerID:  playerID,
		Timestamp: time.Now(),
		PingID:    fmt.Sprintf("ping_%d", time.Now().UnixNano()),
	}

	pingData, err := json.Marshal(pingMsg)
	if err != nil {
		log.Printf("Error marshaling ping message: %v", err)
		return
	}

	if err := wsConn.WriteMessage(websocket.TextMessage, pingData); err != nil {
		log.Printf("Error sending ping to player %s: %v", playerID, err)
		// Mark connection quality as poor on send failure
		gs.qualityMutex.Lock()
		if quality, exists := gs.connectionQualities[playerID]; exists {
			quality.Quality = "poor"
			quality.PacketLoss += 0.1 // Increase packet loss indicator
		}
		gs.qualityMutex.Unlock()
	}
}

// broadcastConnectionQuality sends connection quality updates to all clients
func (gs *GameServer) broadcastConnectionQuality() {
	gs.qualityMutex.RLock()
	allQualities := make(map[string]ConnectionQuality)
	for playerID, quality := range gs.connectionQualities {
		allQualities[playerID] = *quality
	}
	gs.qualityMutex.RUnlock()

	// Hold a read lock on the game state while iterating players to prevent
	// a concurrent write (e.g., from handleConnection) from modifying the map.
	gs.mutex.RLock()
	playerIDs := make([]string, 0, len(gs.gameState.Players))
	for playerID := range gs.gameState.Players {
		playerIDs = append(playerIDs, playerID)
	}
	gs.mutex.RUnlock()

	for _, playerID := range playerIDs {
		statusMsg := ConnectionStatusMessage{
			Type:               "connectionQuality",
			PlayerID:           playerID,
			Quality:            allQualities[playerID],
			AllPlayerQualities: allQualities,
		}

		statusData, err := json.Marshal(statusMsg)
		if err != nil {
			log.Printf("Error marshaling connection status: %v", err)
			continue
		}

		// Non-blocking send mirrors the broadcastGameState pattern.
		// When the channel is full the quality update is dropped rather than
		// causing the ping goroutine to accumulate blocked sends under load.
		select {
		case gs.broadcastCh <- statusData:
		default:
			log.Printf("Broadcast channel full, dropping quality update for %s", playerID)
		}
	}
}

// cleanupConnectionQuality removes connection quality tracking for disconnected player
func (gs *GameServer) cleanupConnectionQuality(playerID string) {
	gs.qualityMutex.Lock()
	defer gs.qualityMutex.Unlock()

	// Stop ping timer
	if timer, exists := gs.pingTimers[playerID]; exists {
		timer.Stop()
		delete(gs.pingTimers, playerID)
	}

	// Remove quality tracking
	delete(gs.connectionQualities, playerID)
}

// Enhanced monitoring methods for comprehensive dashboard support

// getGameStatistics provides detailed game state analytics
func (gs *GameServer) getGameStatistics() map[string]interface{} {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	totalPlayers, connectedPlayers, totalClues, avgHealth, avgSanity := aggregatePlayerStats(gs.gameState.Players)
	gameProgress := computeGameProgress(totalPlayers, totalClues)
	doomPercent := float64(gs.gameState.Doom) / 12.0 * 100

	return map[string]interface{}{
		"totalPlayers":     totalPlayers,
		"connectedPlayers": connectedPlayers,
		"totalClues":       totalClues,
		"averageHealth":    avgHealth,
		"averageSanity":    avgSanity,
		"gameProgress":     gameProgress,
		"doomThreat":       classifyDoomThreat(doomPercent),
		"doomPercent":      doomPercent,
		"gamePhase":        gs.gameState.GamePhase,
		"gameStarted":      gs.gameState.GameStarted,
	}
}

// aggregatePlayerStats computes per-player totals and averages from the players map.
func aggregatePlayerStats(players map[string]*Player) (total, connected, totalClues int, avgHealth, avgSanity float64) {
	total = len(players)
	for _, p := range players {
		if p.Connected {
			connected++
		}
		totalClues += p.Resources.Clues
		avgHealth += float64(p.Resources.Health)
		avgSanity += float64(p.Resources.Sanity)
	}
	if total > 0 {
		avgHealth /= float64(total)
		avgSanity /= float64(total)
	}
	return total, connected, totalClues, avgHealth, avgSanity
}

// computeGameProgress returns the clue-collection progress (0–100) toward victory.
func computeGameProgress(totalPlayers, totalClues int) float64 {
	if totalPlayers == 0 {
		return 0
	}
	progress := float64(totalClues) / float64(totalPlayers*4) * 100
	if progress > 100 {
		return 100
	}
	return progress
}

// classifyDoomThreat maps a doom percentage to a human-readable threat level.
func classifyDoomThreat(doomPercent float64) string {
	switch {
	case doomPercent > 75:
		return "Critical"
	case doomPercent > 50:
		return "High"
	case doomPercent > 25:
		return "Medium"
	default:
		return "Low"
	}
}

// getSystemAlerts checks for system issues and returns alerts
func (gs *GameServer) getSystemAlerts() []map[string]interface{} {
	alerts := []map[string]interface{}{}

	// Performance alerts
	performanceMetrics := gs.collectPerformanceMetrics()

	// High memory usage alert
	if performanceMetrics.MemoryUsage.AllocMB > 100 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High memory usage: %.1f MB", performanceMetrics.MemoryUsage.AllocMB),
			"severity": "medium",
		})
	}

	// High response time alert
	if performanceMetrics.ResponseTimeMs > 100 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High response time: %.1f ms", performanceMetrics.ResponseTimeMs),
			"severity": "medium",
		})
	}

	// High error rate alert
	if performanceMetrics.ErrorRate > 5 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "error",
			"message":  fmt.Sprintf("High error rate: %.1f%%", performanceMetrics.ErrorRate),
			"severity": "high",
		})
	}

	// Game state alerts
	gs.mutex.RLock()
	doomPercent := float64(gs.gameState.Doom) / 12.0 * 100
	gs.mutex.RUnlock()

	if doomPercent > 80 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "error",
			"message":  fmt.Sprintf("Critical doom level: %d/12 (%.0f%%)", gs.gameState.Doom, doomPercent),
			"severity": "critical",
		})
	} else if doomPercent > 60 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High doom level: %d/12 (%.0f%%)", gs.gameState.Doom, doomPercent),
			"severity": "medium",
		})
	}

	return alerts
}
