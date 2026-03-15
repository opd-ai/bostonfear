// Package main contains WebSocket connection handling for the Arkham Horror
// multiplayer game server. This file manages player connections, authentication
// via reconnect tokens, message routing, and the broadcast/action goroutines.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// handleConnection manages WebSocket connections using net.Conn interface
// Moved from: main.go
func (gs *GameServer) handleConnection(conn net.Conn) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	// Handshake timeout: give the client 30 seconds to complete the WebSocket
	// upgrade. This is distinct from the per-message inactivity timeout applied
	// in runMessageLoop, which resets after every successfully received message.
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}
	wsConn, ok := gs.wsConns[conn.RemoteAddr().String()]
	gs.mutex.RUnlock()
	if !ok {
		return fmt.Errorf("websocket connection not found for %s", conn.RemoteAddr().String())
	}

	playerID, err := gs.registerPlayer(conn)
	if err != nil {
		return err
	}

	gs.sendConnectionStatus(wsConn, playerID)
	gs.broadcastGameState()
	gs.runMessageLoop(conn, wsConn, playerID)

	gs.handlePlayerDisconnect(playerID, conn.RemoteAddr().String())
	return nil
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
	} else if gs.gameState.GameStarted && gs.gameState.GamePhase == "playing" {
		log.Printf("Player %s joined game in progress (turn order position %d)", playerID, len(gs.gameState.TurnOrder))
	}

	return playerID, nil
}

// sendConnectionStatus sends the connectionStatus message to the newly connected client,
// including the reconnection token so the client can reclaim its slot on reconnect.
func (gs *GameServer) sendConnectionStatus(wsConn *websocket.Conn, playerID string) {
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
	wsConn.WriteMessage(websocket.TextMessage, data)
}

// runMessageLoop reads incoming WebSocket messages until the connection closes or errors.
// The read deadline is renewed after every received message (inactivity timeout),
// so the 30-second window applies per-message, not as a single reconnection window.
func (gs *GameServer) runMessageLoop(conn net.Conn, wsConn *websocket.Conn, playerID string) {
	for {
		// Per-message inactivity timeout: if no message arrives within 30 seconds
		// the deadline fires. The deadline is renewed after each successful read.
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			log.Printf("Failed to set read deadline: %v", err)
		}

		_, messageData, err := wsConn.ReadMessage()
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

// handlePlayerDisconnect cleans up all state for a disconnecting player.
// If the player held the current turn the turn is advanced so the game never
// stalls (fixes GAP-03). DisconnectedAt is set so the reaper can reclaim the
// slot after the reconnection TTL expires.
func (gs *GameServer) handlePlayerDisconnect(playerID, addrStr string) {
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
	delete(gs.wsConns, addrStr)
	delete(gs.playerConns, playerID)
	gs.mutex.Unlock()

	gs.broadcastGameState()
}

// restorePlayerByToken looks up a disconnected player whose ReconnectToken matches
// token, marks them connected, and returns their playerID.  Returns "" if not found.
// Caller must hold gs.mutex.
func (gs *GameServer) restorePlayerByToken(token string, conn net.Conn) string {
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

// handleWebSocket handles WebSocket upgrade and connection setup.
// If the request includes a ?token= query param, the matching disconnected
// player slot is restored instead of creating a new player.
func (gs *GameServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("New WebSocket connection attempt from %s", r.RemoteAddr)

	wsConn, err := gs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		atomic.AddInt64(&gs.errorCount, 1)
		return
	}

	log.Printf("WebSocket connection established with %s", wsConn.RemoteAddr())

	// Create connection wrapper implementing net.Conn interface with distinct local/remote addresses
	remoteAddr := wsConn.RemoteAddr()
	localAddr := wsConn.NetConn().LocalAddr()
	connWrapper := NewConnectionWrapper(wsConn, localAddr, remoteAddr)

	// Store connections with proper interface usage
	gs.mutex.Lock()
	addrStr := remoteAddr.String()
	gs.connections[addrStr] = connWrapper
	gs.wsConns[addrStr] = wsConn
	log.Printf("Stored connection %s, total connections: %d", addrStr, len(gs.connections))

	// Token-based reconnection: restore disconnected player if token matches.
	reconnectToken := r.URL.Query().Get("token")
	if reconnectToken != "" {
		if restoredID := gs.restorePlayerByToken(reconnectToken, connWrapper); restoredID != "" {
			gs.mutex.Unlock()
			log.Printf("Player %s reconnected via token", restoredID)
			gs.sendConnectionStatus(wsConn, restoredID)
			gs.broadcastGameState()
			gs.runMessageLoop(connWrapper, wsConn, restoredID)
			gs.handlePlayerDisconnect(restoredID, addrStr)
			return
		}
	}
	gs.mutex.Unlock()

	// Handle connection in separate goroutine
	go func() {
		if err := gs.handleConnection(connWrapper); err != nil {
			log.Printf("Connection handling error: %v", err)
		}
	}()
}

// broadcastHandler processes broadcast messages to all connected clients
func (gs *GameServer) broadcastHandler() {
	for {
		select {
		case message := <-gs.broadcastCh:
			writeStart := time.Now()
			gs.mutex.RLock()
			for _, wsConn := range gs.wsConns {
				if err := wsConn.WriteMessage(websocket.TextMessage, message); err != nil {
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
func (gs *GameServer) broadcastGameState() {
	gs.mutex.Lock()
	gs.validateAndRecoverState()
	gameStateMsg := map[string]interface{}{
		"type": "gameState",
		"data": gs.gameState,
	}
	gs.mutex.Unlock()

	data, err := json.Marshal(gameStateMsg)
	if err != nil {
		log.Printf("Game state marshal error: %v", err)
		return
	}

	gs.trySendBroadcast(data, "gameState")
}
