package serverengine

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/opd-ai/bostonfear/serverengine/common/logging"
	"github.com/opd-ai/bostonfear/serverengine/common/messaging"
)

type connectionTarget struct {
	addr string
	conn net.Conn
}

// snapshotConnections returns a stable copy of current connection targets.
// Taking this snapshot lets broadcast writes happen without holding gs.mutex.
func (gs *GameServer) snapshotConnections() []connectionTarget {
	gs.mutex.RLock()
	targets := make([]connectionTarget, 0, len(gs.connections))
	for addr, conn := range gs.connections {
		targets = append(targets, connectionTarget{addr: addr, conn: conn})
	}
	gs.mutex.RUnlock()
	return targets
}

// broadcastHandler processes broadcast messages to all connected clients.
func (gs *GameServer) broadcastHandler() {
	for {
		select {
		case message := <-gs.broadcastCh:
			writeStart := time.Now()
			targets := gs.snapshotConnections()
			for _, target := range targets {
				if err := gs.writeToConn(target.conn, target.addr, message); err != nil {
					logging.Error("Broadcast error", "error", err, "address", target.addr)
					atomic.AddInt64(&gs.errorCount, 1)
				} else {
					gs.trackMessage("sent")
				}
			}
			// Record how long this broadcast round took for latency metrics.
			gs.recordBroadcastLatency(time.Since(writeStart))
		case <-gs.shutdownCh:
			logging.Info("Broadcast handler shutting down")
			return
		}
	}
}

// actionHandler processes queued player actions.
func (gs *GameServer) actionHandler() {
	for {
		select {
		case action := <-gs.actionCh:
			gs.trackMessage("received")
			if err := gs.processAction(action); err != nil {
				logging.Error("Action processing error", "error", err, "playerID", action.PlayerID, "action", action.Action)
				atomic.AddInt64(&gs.errorCount, 1)
			}
		case <-gs.shutdownCh:
			logging.Info("Action handler shutting down")
			return
		}
	}
}

// broadcastGameState sends current game state to all connected clients.
// Uses a full write lock because the recovery path may assign gs.gameState.
// json.Marshal is performed inside the lock so no concurrent write can race
// against serialization of game state fields.
func (gs *GameServer) broadcastGameState() {
	gs.mutex.Lock()
	gs.validateAndRecoverState()
	gameStateMsg := gs.buildGameStatePayloadLocked()
	data, err := messaging.EncodeJSON(gameStateMsg)
	gs.mutex.Unlock()

	if err != nil {
		logging.Error("Game state marshal error", "error", err)
		return
	}

	gs.trySendBroadcast(data, "gameState")
}

// buildGameStatePayloadLocked builds the gameState payload while gs.mutex is held.
func (gs *GameServer) buildGameStatePayloadLocked() interface{} {
	if adapter := gs.broadcastAdapter; adapter != nil {
		return adapter.ShapeGameStatePayload(gs.gameState)
	}
	return map[string]interface{}{
		"type": "gameState",
		"data": gs.gameState,
	}
}
