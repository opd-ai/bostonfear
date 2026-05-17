// Package serverengine contains transport-neutral session handling for the
// Arkham Horror multiplayer game server.
package serverengine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/opd-ai/bostonfear/serverengine/common/logging"
	commonsession "github.com/opd-ai/bostonfear/serverengine/common/session"
)

const (
	// reconnectGracePeriod is the maximum time a disconnected slot can be reclaimed.
	reconnectGracePeriod = 30 * time.Second
	// staleSessionPollInterval controls how often zombie sessions are reaped.
	staleSessionPollInterval = 5 * time.Second
)

type displayNameProvider interface {
	DisplayName() string
}

// HandleConnection manages a player session using only net.Conn so transports
// can adapt websocket, tcp, or in-process connections without engine changes.
// It returns ErrGameFull when the session cannot register another player,
// ctx-related errors when startup cancellation has already occurred, or network
// I/O errors originating from conn.
//
// Parameter constraints:
//   - conn must be a non-nil readable/writable connection.
//   - reconnectToken may be empty. When non-empty, the server first attempts a
//     slot restore; if no disconnected slot matches, a new player is registered.
func (gs *GameServer) HandleConnection(conn net.Conn, reconnectToken string) error {
	return gs.HandleConnectionWithContext(context.Background(), conn, reconnectToken)
}

// HandleConnectionWithContext manages a player session and exits early when ctx
// is canceled.
// It returns ErrGameFull when registration exceeds MaxPlayers, ctx.Err() when
// canceled before startup, or connection I/O errors during session processing.
//
// Parameter constraints:
//   - ctx must be non-nil.
//   - conn must be a non-nil readable/writable connection.
//   - reconnectToken semantics are identical to HandleConnection.
func (gs *GameServer) HandleConnectionWithContext(ctx context.Context, conn net.Conn, reconnectToken string) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	defer close(done)
	defer func() {
		if err := conn.Close(); err != nil {
			logging.Error("Error closing connection", "error", err)
		}
	}()
	addrStr := conn.RemoteAddr().String()
	gs.mutex.Lock()
	gs.connections[addrStr] = conn
	atomic.AddInt64(&gs.activeConnections, 1)
	gs.mutex.Unlock()

	// Initial timeout before first client message arrives.
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		logging.Error("Failed to set read deadline", "error", err)
	}

	if reconnectToken != "" {
		if restoredID := gs.restorePlayerByToken(reconnectToken, conn, connectionDisplayName(conn)); restoredID != "" {
			logging.Info("Player reconnected via token", "playerID", restoredID)
			gs.trackConnection("reconnect", restoredID, 0)
			gs.trackPlayerSession(restoredID, "reconnect")
			gs.initializeConnectionQuality(restoredID)
			gs.sendConnectionStatus(conn, restoredID)
			gs.broadcastGameState()
			gs.runMessageLoop(ctx, conn, restoredID)
			gs.handlePlayerDisconnect(restoredID, addrStr)
			return nil
		}
	}

	playerID, err := gs.registerPlayer(conn, connectionDisplayName(conn))
	if err != nil {
		gs.removeConnection(addrStr)
		return err
	}

	gs.sendConnectionStatus(conn, playerID)
	gs.broadcastGameState()
	gs.runMessageLoop(ctx, conn, playerID)

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

func connectionDisplayName(conn net.Conn) string {
	if namedConn, ok := conn.(displayNameProvider); ok {
		return strings.TrimSpace(namedConn.DisplayName())
	}
	return ""
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
func (gs *GameServer) registerPlayer(conn net.Conn, displayName string) (string, error) {
	playerID := fmt.Sprintf("player_%d", time.Now().UnixNano())
	if strings.TrimSpace(displayName) == "" {
		displayName = playerID
	}

	gs.trackConnection("connect", playerID, 0)
	gs.trackPlayerSession(playerID, "start")
	gs.initializeConnectionQuality(playerID)

	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	if len(gs.gameState.Players) >= MaxPlayers {
		return "", fmt.Errorf("%w (max %d players)", ErrGameFull, MaxPlayers)
	}

	gs.gameState.Players[playerID] = &Player{
		ID:          playerID,
		DisplayName: displayName,
		Location:    Downtown,
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
		logging.Info("Player joined game in progress", "playerID", playerID, "turnOrderPosition", len(gs.gameState.TurnOrder))
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
		"type":        "connectionStatus",
		"playerId":    playerID,
		"displayName": "",
		"token":       token,
		"status":      "connected",
	}
	if p, ok := gs.gameState.Players[playerID]; ok && p != nil {
		msg["displayName"] = p.DisplayName
	}
	data, _ := json.Marshal(msg)
	gs.writeToConn(conn, conn.RemoteAddr().String(), data) //nolint:errcheck
}

// runMessageLoop reads incoming messages until the connection closes or errors.
func (gs *GameServer) runMessageLoop(ctx context.Context, conn net.Conn, playerID string) {
	buf := make([]byte, 64*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			logging.Error("Failed to set read deadline", "error", err)
		}

		n, err := conn.Read(buf)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logging.Warn("Connection timeout for player", "playerID", playerID)
				gs.mutex.Lock()
				gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
				gs.checkGameEndConditions()
				gs.mutex.Unlock()
				gs.broadcastGameState()
			} else {
				logging.Error("WebSocket read error", "error", err, "playerID", playerID)
			}
			break
		}
		if n == 0 {
			continue
		}
		messageData := make([]byte, n)
		copy(messageData, buf[:n])

		receiveTime := time.Now()
		if !gs.handleIncomingMessage(ctx, messageData, playerID, receiveTime) {
			break
		}
	}
}

// handleIncomingMessage parses and dispatches a single raw WebSocket message.
// Returns false only when the caller should stop the message loop.
func (gs *GameServer) handleIncomingMessage(ctx context.Context, data []byte, playerID string, receiveTime time.Time) bool {
	var actionMsg PlayerActionMessage
	if err := json.Unmarshal(data, &actionMsg); err != nil {
		var pingMsg PingMessage
		if pingErr := json.Unmarshal(data, &pingMsg); pingErr == nil && pingMsg.Type == "pong" {
			gs.handlePongMessage(pingMsg, receiveTime)
			return true
		}
		logging.Error("Message unmarshal error", "error", err, "playerID", playerID)
		atomic.AddInt64(&gs.errorCount, 1)
		return true
	}

	if actionMsg.PlayerID != playerID {
		logging.Warn("Invalid player ID in action", "expected", playerID, "got", actionMsg.PlayerID)
		atomic.AddInt64(&gs.errorCount, 1)
		return true
	}

	gs.updateConnectionQuality(playerID, receiveTime)
	select {
	case gs.actionCh <- actionMsg:
		return true
	case <-ctx.Done():
		return false
	case <-gs.shutdownCh:
		return false
	}
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
func (gs *GameServer) restorePlayerByToken(token string, conn net.Conn, displayName string) string {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()
	now := time.Now()

	for id, p := range gs.gameState.Players {
		if !canRestorePlayer(token, p, now) {
			continue
		}
		p.Connected = true
		p.DisconnectedAt = time.Time{}              // clear disconnect timestamp
		p.ReconnectToken = generateReconnectToken() // rotate token
		if strings.TrimSpace(p.DisplayName) == "" && strings.TrimSpace(displayName) != "" {
			p.DisplayName = strings.TrimSpace(displayName)
		}
		gs.playerConns[id] = conn
		return id
	}
	return ""
}

func canRestorePlayer(token string, p *Player, now time.Time) bool {
	if p == nil {
		return false
	}
	candidate := commonsession.Token(token)
	if !candidate.Validate() {
		return false
	}
	record := commonsession.Record{
		PlayerID:       p.ID,
		Token:          commonsession.Token(p.ReconnectToken),
		Connected:      p.Connected,
		DisconnectedAt: p.DisconnectedAt,
	}
	return commonsession.Default.CanRestore(record, candidate, now, reconnectGracePeriod)
}

// cleanupDisconnectedPlayers removes player entries that have been disconnected
// longer than the reconnection grace period.
// Must be called from a goroutine; polls every staleSessionPollInterval.
func (gs *GameServer) cleanupDisconnectedPlayers() {
	ticker := time.NewTicker(staleSessionPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-gs.shutdownCh:
			return
		case <-ticker.C:
			reaped := gs.reapDisconnectedPlayers(time.Now())
			gs.postReapCleanup(reaped)
		}
	}
}

func (gs *GameServer) reapDisconnectedPlayers(now time.Time) []string {
	reaped := make([]string, 0)
	gs.mutex.Lock()
	for id, p := range gs.gameState.Players {
		if !isStaleDisconnectedPlayer(p, now) {
			continue
		}
		logging.Info("Reaping zombie player", "playerID", id, "disconnectedAt", p.DisconnectedAt)
		delete(gs.gameState.Players, id)
		reaped = append(reaped, id)
		removeFromTurnOrder(&gs.gameState.TurnOrder, id)
		if gs.gameState.CurrentPlayer == id {
			gs.gameState.CurrentPlayer = firstTurnPlayer(gs.gameState.TurnOrder)
		}
	}
	gs.mutex.Unlock()
	return reaped
}

func isStaleDisconnectedPlayer(p *Player, now time.Time) bool {
	if p == nil || p.Connected || p.DisconnectedAt.IsZero() {
		return false
	}
	return commonsession.Default.IsExpired(p.DisconnectedAt, now, reconnectGracePeriod)
}

func removeFromTurnOrder(turnOrder *[]string, playerID string) {
	for i, id := range *turnOrder {
		if id == playerID {
			*turnOrder = append((*turnOrder)[:i], (*turnOrder)[i+1:]...)
			return
		}
	}
}

func firstTurnPlayer(turnOrder []string) string {
	if len(turnOrder) == 0 {
		return ""
	}
	return turnOrder[0]
}

func (gs *GameServer) postReapCleanup(reaped []string) {
	for _, id := range reaped {
		gs.trackConnection("reap", id, 0)
		gs.cleanupConnectionQuality(id)
	}
	if len(reaped) > 0 {
		gs.broadcastGameState()
	}
}
