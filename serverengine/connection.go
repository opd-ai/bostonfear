// Package serverengine contains transport-neutral session handling for the
// Arkham Horror multiplayer game server.
package serverengine

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

// HandleConnection manages a player session using only net.Conn so transports
// can adapt websocket, tcp, or in-process connections without engine changes.
func (gs *GameServer) HandleConnection(conn net.Conn, reconnectToken string) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()
	addrStr := conn.RemoteAddr().String()
	gs.mutex.Lock()
	gs.connections[addrStr] = conn
	atomic.AddInt64(&gs.activeConnections, 1)
	gs.mutex.Unlock()

	// Initial timeout before first client message arrives.
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}

	if reconnectToken != "" {
		if restoredID := gs.restorePlayerByToken(reconnectToken, conn); restoredID != "" {
			log.Printf("Player %s reconnected via token", restoredID)
			gs.sendConnectionStatus(conn, restoredID)
			gs.broadcastGameState()
			gs.runMessageLoop(conn, restoredID)
			gs.handlePlayerDisconnect(restoredID, addrStr)
			return nil
		}
	}

	playerID, err := gs.registerPlayer(conn)
	if err != nil {
		gs.removeConnection(addrStr)
		return err
	}

	gs.sendConnectionStatus(conn, playerID)
	gs.broadcastGameState()
	gs.runMessageLoop(conn, playerID)

	gs.handlePlayerDisconnect(playerID, addrStr)
	return nil
}

func (gs *GameServer) removeConnection(addrStr string) {
	gs.mutex.Lock()
	delete(gs.connections, addrStr)
	atomic.AddInt64(&gs.activeConnections, -1)
	gs.mutex.Unlock()
	gs.removeConnWriteLock(addrStr)
}

// generateReconnectToken returns a cryptographically random 16-byte hex token
// used to restore a disconnected player's slot on reconnection.
func generateReconnectToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp on crypto failure (should never happen in practice).
		return fmt.Sprintf("tok_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// registerPlayer adds a new player to the game state and starts their monitoring.
// Returns the new player's ID or an error if the game is full.
func (gs *GameServer) registerPlayer(conn net.Conn) (string, error) {
	playerID := fmt.Sprintf("player_%d", time.Now().UnixNano())

	gs.trackConnection("connect", playerID, 0)
	gs.trackPlayerSession(playerID, "start")
	gs.initializeConnectionQuality(playerID)

	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if len(gs.gameState.Players) >= MaxPlayers {
		return "", fmt.Errorf("game is full (max %d players)", MaxPlayers)
	}

	gs.gameState.Players[playerID] = &Player{
		ID:       playerID,
		Location: Downtown,
		Resources: Resources{
			Health: 10,
			Sanity: 10,
			Clues:  0,
		},
		ActionsRemaining: 0,
		Connected:        true,
		ReconnectToken:   generateReconnectToken(),
	}
	gs.gameState.TurnOrder = append(gs.gameState.TurnOrder, playerID)
	gs.playerConns[playerID] = conn

	if len(gs.gameState.Players) >= MinPlayers && !gs.gameState.GameStarted {
		gs.gameState.GameStarted = true
		gs.gameState.GamePhase = "playing"
		gs.gameState.CurrentPlayer = gs.gameState.TurnOrder[0]
		gs.gameState.Players[gs.gameState.CurrentPlayer].ActionsRemaining = 2
		// Rescale act deck thresholds to match player count (4 clues/investigator).
		if len(gs.gameState.ActDeck) >= 3 {
			gs.rescaleActDeck(len(gs.gameState.Players))
		}
	} else if gs.gameState.GameStarted && gs.gameState.GamePhase == "playing" {
		log.Printf("Player %s joined game in progress (turn order position %d)", playerID, len(gs.gameState.TurnOrder))
		// Rescale act-deck win thresholds to include the new investigator
		// (4 clues/investigator per README win table). Without this, a late join
		// leaves the threshold at the original player count's value.
		if len(gs.gameState.ActDeck) >= 3 {
			gs.rescaleActDeck(len(gs.gameState.Players))
		}
	}

	return playerID, nil
}

// sendConnectionStatus sends the connectionStatus message to the player,
// including the reconnection token so the client can reclaim its slot.
func (gs *GameServer) sendConnectionStatus(conn net.Conn, playerID string) {
	gs.mutex.RLock()
	token := ""
	if p, ok := gs.gameState.Players[playerID]; ok {
		token = p.ReconnectToken
	}
	gs.mutex.RUnlock()

	msg := map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": playerID,
		"token":    token,
		"status":   "connected",
	}
	data, _ := json.Marshal(msg)
	gs.writeToConn(conn, conn.RemoteAddr().String(), data) //nolint:errcheck
}

// runMessageLoop reads incoming messages until the connection closes or errors.
func (gs *GameServer) runMessageLoop(conn net.Conn, playerID string) {
	buf := make([]byte, 64*1024)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			log.Printf("Failed to set read deadline: %v", err)
		}

		n, err := conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Connection timeout for player %s", playerID)
				gs.mutex.Lock()
				gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
				gs.checkGameEndConditions()
				gs.mutex.Unlock()
				gs.broadcastGameState()
			} else {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
		if n == 0 {
			continue
		}
		messageData := make([]byte, n)
		copy(messageData, buf[:n])

		receiveTime := time.Now()
		if !gs.handleIncomingMessage(messageData, playerID, receiveTime) {
			break
		}
	}
}

// handleIncomingMessage parses and dispatches a single raw WebSocket message.
// Returns false only when the caller should stop the message loop.
func (gs *GameServer) handleIncomingMessage(data []byte, playerID string, receiveTime time.Time) bool {
	var actionMsg PlayerActionMessage
	if err := json.Unmarshal(data, &actionMsg); err != nil {
		var pingMsg PingMessage
		if pingErr := json.Unmarshal(data, &pingMsg); pingErr == nil && pingMsg.Type == "pong" {
			gs.handlePongMessage(pingMsg, receiveTime)
			return true
		}
		log.Printf("Message unmarshal error: %v", err)
		atomic.AddInt64(&gs.errorCount, 1)
		return true
	}

	if actionMsg.PlayerID != playerID {
		log.Printf("Invalid player ID in action: expected %s, got %s", playerID, actionMsg.PlayerID)
		atomic.AddInt64(&gs.errorCount, 1)
		return true
	}

	gs.updateConnectionQuality(playerID, receiveTime)
	gs.actionCh <- actionMsg
	return true
}

// handlePlayerDisconnect routes disconnect processing through the core handler.
func (gs *GameServer) handlePlayerDisconnect(playerID, addrStr string) {
	gs.handlePlayerDisconnectCore(playerID, addrStr)
}

// handlePlayerDisconnectCore cleans up all state for a disconnecting player.
// If the player held the current turn the turn is advanced so the game never
// stalls (fixes GAP-03). DisconnectedAt is set so the reaper can reclaim the
// slot after the reconnection TTL expires.
func (gs *GameServer) handlePlayerDisconnectCore(playerID, addrStr string) {
	gs.mutex.Lock()
	if player, exists := gs.gameState.Players[playerID]; exists {
		player.Connected = false
		player.DisconnectedAt = time.Now()
	}
	if gs.gameState.CurrentPlayer == playerID && gs.gameState.GamePhase == "playing" {
		gs.advanceTurn()
	}
	gs.mutex.Unlock()

	gs.trackConnection("disconnect", playerID, 0)
	gs.trackPlayerSession(playerID, "end")
	gs.cleanupConnectionQuality(playerID)

	gs.mutex.Lock()
	delete(gs.connections, addrStr)
	delete(gs.playerConns, playerID)
	atomic.AddInt64(&gs.activeConnections, -1)
	gs.mutex.Unlock()
	gs.removeConnWriteLock(addrStr)

	gs.broadcastGameState()
}

// restorePlayerByToken looks up a disconnected player whose ReconnectToken
// matches token, marks them connected, and returns their playerID.
func (gs *GameServer) restorePlayerByToken(token string, conn net.Conn) string {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	for id, p := range gs.gameState.Players {
		if p.ReconnectToken == token && !p.Connected {
			p.Connected = true
			p.DisconnectedAt = time.Time{}              // clear disconnect timestamp
			p.ReconnectToken = generateReconnectToken() // rotate token
			gs.playerConns[id] = conn
			return id
		}
	}
	return ""
}

// cleanupDisconnectedPlayers removes player entries that have been disconnected
// longer than the reconnection TTL (5 minutes).
// Must be called from a goroutine; polls every 30 seconds.
func (gs *GameServer) cleanupDisconnectedPlayers() {
	const ttl = 5 * time.Minute
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-gs.shutdownCh:
			return
		case <-ticker.C:
			now := time.Now()
			gs.mutex.Lock()
			for id, p := range gs.gameState.Players {
				if !p.Connected && !p.DisconnectedAt.IsZero() && now.Sub(p.DisconnectedAt) > ttl {
					log.Printf("Reaping zombie player %s (disconnected at %v)", id, p.DisconnectedAt)
					delete(gs.gameState.Players, id)
					// Remove from TurnOrder.
					for i, tid := range gs.gameState.TurnOrder {
						if tid == id {
							gs.gameState.TurnOrder = append(gs.gameState.TurnOrder[:i], gs.gameState.TurnOrder[i+1:]...)
							break
						}
					}
				}
			}
			gs.mutex.Unlock()
		}
	}
}

// broadcastHandler processes broadcast messages to all connected clients
func (gs *GameServer) broadcastHandler() {
	for {
		select {
		case message := <-gs.broadcastCh:
			writeStart := time.Now()
			gs.mutex.RLock()
			for addr, conn := range gs.connections {
				if err := gs.writeToConn(conn, addr, message); err != nil {
					log.Printf("Broadcast error: %v", err)
					atomic.AddInt64(&gs.errorCount, 1)
				} else {
					gs.trackMessage("sent")
				}
			}
			gs.mutex.RUnlock()
			// Record how long this broadcast round took for latency metrics
			gs.recordBroadcastLatency(time.Since(writeStart))
		case <-gs.shutdownCh:
			log.Printf("Broadcast handler shutting down")
			return
		}
	}
}

// actionHandler processes player actions through channel
func (gs *GameServer) actionHandler() {
	for {
		select {
		case action := <-gs.actionCh:
			gs.trackMessage("received")
			if err := gs.processAction(action); err != nil {
				log.Printf("Action processing error: %v", err)
				atomic.AddInt64(&gs.errorCount, 1)
			}
		case <-gs.shutdownCh:
			log.Printf("Action handler shutting down")
			return
		}
	}
}

// broadcastGameState sends current game state to all connected clients.
// Uses a full write lock because the recovery path may assign gs.gameState.
// json.Marshal is performed inside the lock so no concurrent write can race
// against the serialization of the gameState pointer's fields.
func (gs *GameServer) broadcastGameState() {
	gs.mutex.Lock()
	gs.validateAndRecoverState()
	gameStateMsg := map[string]interface{}{
		"type": "gameState",
		"data": gs.gameState,
	}
	data, err := json.Marshal(gameStateMsg)
	gs.mutex.Unlock()

	if err != nil {
		log.Printf("Game state marshal error: %v", err)
		return
	}

	gs.trySendBroadcast(data, "gameState")
}
