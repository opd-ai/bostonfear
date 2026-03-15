// Package main implements health check endpoint and state diagnostics for the
// Arkham Horror multiplayer game server. This file handles HTTP health reporting,
// game state validation and recovery, error rate tracking, and game statistics.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

// handleHealthCheck provides a health monitoring endpoint.
// Game state is snapshotted under a short RLock; serialization happens outside the lock.
func (gs *GameServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Snapshot required game-state fields under a single short read lock.
	// All helper calls that touch gs.mutex must happen AFTER this block.
	gs.mutex.RLock()
	isHealthy := gs.validator.IsGameStateHealthy(gs.gameState)
	playerCount := len(gs.gameState.Players)
	connectionCount := len(gs.connections)
	corruptionHistory := gs.validator.GetCorruptionHistory()
	gamePhase := gs.gameState.GamePhase
	doom := gs.gameState.Doom
	gameStarted := gs.gameState.GameStarted
	gs.mutex.RUnlock()

	// Helpers below may acquire their own locks (performanceMutex or gs.mutex).
	// Calling them outside gs.mutex prevents nested-RLock deadlock under write pressure.
	perfMetrics := gs.collectPerformanceMetrics()
	connAnalytics := gs.collectConnectionAnalytics()
	gameStats := gs.getGameStatistics()
	alerts := gs.getSystemAlerts()

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

	// Game state alerts: capture doom under a lock, then use the snapshot below.
	gs.mutex.RLock()
	doom := gs.gameState.Doom
	doomPercent := float64(doom) / 12.0 * 100
	gs.mutex.RUnlock()

	if doomPercent > 80 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "error",
			"message":  fmt.Sprintf("Critical doom level: %d/12 (%.0f%%)", doom, doomPercent),
			"severity": "critical",
		})
	} else if doomPercent > 60 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High doom level: %d/12 (%.0f%%)", doom, doomPercent),
			"severity": "medium",
		})
	}

	return alerts
}
