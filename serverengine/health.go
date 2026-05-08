// Package serverengine implements health diagnostics and state validation for the
// Arkham Horror multiplayer game server. This file handles HTTP health reporting,
// game state validation and recovery, error rate tracking, and game statistics.
package serverengine

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/opd-ai/bostonfear/monitoring"
	"github.com/opd-ai/bostonfear/monitoringdata"
)

type HealthSnapshot = monitoringdata.HealthSnapshot

// SnapshotHealth captures the game-state fields needed by monitoring handlers.
func (gs *GameServer) SnapshotHealth() HealthSnapshot {
	gs.mutex.RLock()
	snapshot := HealthSnapshot{
		IsHealthy:         gs.validator.IsGameStateHealthy(gs.gameState),
		PlayerCount:       len(gs.gameState.Players),
		ConnectionCount:   len(gs.connections),
		CorruptionHistory: make([]time.Time, 0, len(gs.validator.GetCorruptionHistory())),
		GamePhase:         gs.gameState.GamePhase,
		Doom:              gs.gameState.Doom,
		GameStarted:       gs.gameState.GameStarted,
	}
	for _, event := range gs.validator.GetCorruptionHistory() {
		snapshot.CorruptionHistory = append(snapshot.CorruptionHistory, event.Timestamp)
	}
	gs.mutex.RUnlock()
	return snapshot
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

// measureHealthCheckResponseTime measures the response time of health check operations.
// Uses only lock-free atomic reads so it is safe to call while gs.mutex is held or not held.
func (gs *GameServer) measureHealthCheckResponseTime() float64 {
	start := time.Now()
	// Read an atomic counter — no mutex required, no deadlock risk.
	_ = atomic.LoadInt64(&gs.activeConnections)
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

// getGameStatistics provides detailed game state analytics via metricsCollector.
func (gs *GameServer) getGameStatistics() map[string]interface{} {
	if gs.metricsCollector == nil {
		return gs.getGameStatisticsCore()
	}
	return gs.metricsCollector.GameStatistics()
}

// getGameStatisticsCore computes detailed game state analytics.
func (gs *GameServer) getGameStatisticsCore() map[string]interface{} {
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

// getSystemAlerts delegates alert-threshold policy to the monitoring package.
func (gs *GameServer) getSystemAlerts() []map[string]interface{} {
	performanceMetrics := gs.collectPerformanceMetrics()
	gs.mutex.RLock()
	doom := gs.gameState.Doom
	gs.mutex.RUnlock()
	return monitoring.BuildSystemAlerts(performanceMetrics, doom)
}
