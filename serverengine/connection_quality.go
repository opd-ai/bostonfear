package serverengine

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ConnectionQuality represents real-time connection quality metrics for a single player.
type ConnectionQuality struct {
	LatencyMs     float64   `json:"latencyMs"`
	Quality       string    `json:"quality"`
	PacketLoss    float64   `json:"packetLoss"`
	LastPingTime  time.Time `json:"lastPingTime"`
	MessageDelay  float64   `json:"messageDelay"`
	pingsSent     int
	pongsReceived int
}

// ConnectionStatusMessage represents connection quality updates broadcast to clients.
type ConnectionStatusMessage struct {
	Type               string                       `json:"type"`
	PlayerID           string                       `json:"playerId"`
	DisplayName        string                       `json:"displayName"`
	Token              string                       `json:"token,omitempty"`
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

// initializeConnectionQuality sets up initial connection quality for a player.
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

	gs.startPingTimer(playerID)
}

// updateConnectionQuality updates connection quality metrics based on message timing.
func (gs *GameServer) updateConnectionQuality(playerID string, messageTime time.Time) {
	gs.qualityMutex.Lock()
	defer gs.qualityMutex.Unlock()

	quality, exists := gs.connectionQualities[playerID]
	if !exists {
		return
	}

	now := time.Now()
	quality.MessageDelay = float64(now.Sub(messageTime).Nanoseconds()) / 1000000
	gs.assessConnectionQuality(playerID)
}

// handlePongMessage processes pong responses and calculates latency.
func (gs *GameServer) handlePongMessage(pingMsg PingMessage, receiveTime time.Time) {
	gs.qualityMutex.Lock()
	quality, exists := gs.connectionQualities[pingMsg.PlayerID]
	if !exists {
		gs.qualityMutex.Unlock()
		return
	}

	latency := float64(receiveTime.Sub(pingMsg.Timestamp).Nanoseconds()) / 1e6
	quality.LatencyMs = latency
	quality.LastPingTime = receiveTime
	quality.pongsReceived++
	gs.recalcPacketLoss(quality)
	gs.assessConnectionQuality(pingMsg.PlayerID)
	gs.qualityMutex.Unlock()

	gs.broadcastConnectionQuality()
}

// assessConnectionQuality determines connection quality rating based on metrics.
func (gs *GameServer) assessConnectionQuality(playerID string) {
	quality := gs.connectionQualities[playerID]

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

	if quality.PacketLoss > 0.05 {
		if quality.Quality == "excellent" {
			quality.Quality = "good"
		} else if quality.Quality == "good" {
			quality.Quality = "fair"
		} else if quality.Quality == "fair" {
			quality.Quality = "poor"
		}
	}
}

// recalcPacketLoss computes PacketLoss as the ratio of unanswered pings to pings sent.
func (gs *GameServer) recalcPacketLoss(q *ConnectionQuality) {
	if q.pingsSent == 0 {
		q.PacketLoss = 0
		return
	}
	missed := q.pingsSent - q.pongsReceived
	if missed < 0 {
		log.Printf("recalcPacketLoss: pongsReceived (%d) > pingsSent (%d); resetting",
			q.pongsReceived, q.pingsSent)
		q.pingsSent = q.pongsReceived
		missed = 0
	}
	q.PacketLoss = float64(missed) / float64(q.pingsSent)
}

// startPingTimer starts periodic ping for connection quality monitoring.
func (gs *GameServer) startPingTimer(playerID string) {
	timer := time.NewTimer(5 * time.Second)
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
func (gs *GameServer) sendPingToPlayer(playerID string) {
	gs.mutex.RLock()
	conn, connExists := gs.playerConns[playerID]
	gs.mutex.RUnlock()

	if !connExists || conn == nil {
		return
	}
	connAddr := conn.RemoteAddr().String()

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

	gs.qualityMutex.Lock()
	if q, exists := gs.connectionQualities[playerID]; exists {
		q.pingsSent++
		q.LastPingTime = time.Now()
		gs.recalcPacketLoss(q)
	}
	gs.qualityMutex.Unlock()

	if err := gs.writeToConn(conn, connAddr, pingData); err != nil {
		log.Printf("Error sending ping to player %s: %v", playerID, err)
		gs.qualityMutex.Lock()
		if quality, exists := gs.connectionQualities[playerID]; exists {
			quality.Quality = "poor"
			gs.recalcPacketLoss(quality)
		}
		gs.qualityMutex.Unlock()
	}
}

// broadcastConnectionQuality sends connection quality updates to all clients.
func (gs *GameServer) broadcastConnectionQuality() {
	gs.qualityMutex.RLock()
	allQualities := make(map[string]ConnectionQuality)
	for playerID, quality := range gs.connectionQualities {
		allQualities[playerID] = *quality
	}
	gs.qualityMutex.RUnlock()

	gs.mutex.RLock()
	playerIDs := make([]string, 0, len(gs.gameState.Players))
	for playerID := range gs.gameState.Players {
		playerIDs = append(playerIDs, playerID)
	}
	gs.mutex.RUnlock()

	for _, playerID := range playerIDs {
		displayName := playerID
		if player, ok := gs.gameState.Players[playerID]; ok && player != nil && strings.TrimSpace(player.DisplayName) != "" {
			displayName = player.DisplayName
		}
		statusMsg := ConnectionStatusMessage{
			Type:               "connectionQuality",
			PlayerID:           playerID,
			DisplayName:        displayName,
			Quality:            allQualities[playerID],
			AllPlayerQualities: allQualities,
		}

		statusData, err := json.Marshal(statusMsg)
		if err != nil {
			log.Printf("Error marshaling connection status: %v", err)
			continue
		}

		select {
		case gs.broadcastCh <- statusData:
		default:
			log.Printf("Broadcast channel full, dropping quality update for %s", playerID)
		}
	}
}

// cleanupConnectionQuality removes connection quality tracking for disconnected player.
func (gs *GameServer) cleanupConnectionQuality(playerID string) {
	gs.qualityMutex.Lock()
	defer gs.qualityMutex.Unlock()

	if timer, exists := gs.pingTimers[playerID]; exists {
		timer.Stop()
		delete(gs.pingTimers, playerID)
	}

	delete(gs.connectionQualities, playerID)
}
