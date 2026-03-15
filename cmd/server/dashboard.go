// Package main implements connection quality tracking and the performance monitoring
// dashboard for the Arkham Horror multiplayer game server. This file manages
// WebSocket ping/pong latency measurement, per-player quality classification,
// and the dashboard HTML endpoint.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionQuality represents real-time connection quality metrics for a single player.
type ConnectionQuality struct {
	LatencyMs    float64   `json:"latencyMs"`
	Quality      string    `json:"quality"` // "excellent", "good", "fair", "poor"
	PacketLoss   float64   `json:"packetLoss"`
	LastPingTime time.Time `json:"lastPingTime"`
	MessageDelay float64   `json:"messageDelay"`
}

// ConnectionStatusMessage represents connection quality updates broadcast to clients.
type ConnectionStatusMessage struct {
	Type               string                       `json:"type"`
	PlayerID           string                       `json:"playerId"`
	Quality            ConnectionQuality            `json:"quality"`
	AllPlayerQualities map[string]ConnectionQuality `json:"allPlayerQualities"`
}

// PingMessage represents ping/pong messages used for round-trip latency measurement.
type PingMessage struct {
	Type      string    `json:"type"`
	PlayerID  string    `json:"playerId"`
	Timestamp time.Time `json:"timestamp"`
	PingID    string    `json:"pingId"`
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
	var wsAddr string
	var wsExists bool
	if connExists && conn != nil {
		wsAddr = conn.RemoteAddr().String()
		wsConn, wsExists = gs.wsConns[wsAddr]
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

	// Use writeToConn so this write is serialised with broadcastHandler writes
	// on the same connection (gorilla/websocket is not concurrent-write safe).
	if err := gs.writeToConn(wsConn, wsAddr, pingData); err != nil {
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
