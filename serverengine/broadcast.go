package serverengine

import (
	"encoding/json"
	"log"
	"net"
	"sync/atomic"
	"time"
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
					log.Printf("Broadcast error: %v", err)
					atomic.AddInt64(&gs.errorCount, 1)
				} else {
					gs.trackMessage("sent")
				}
			}
			// Record how long this broadcast round took for latency metrics.
			gs.recordBroadcastLatency(time.Since(writeStart))
		case <-gs.shutdownCh:
			log.Printf("Broadcast handler shutting down")
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
// against serialization of game state fields.
func (gs *GameServer) broadcastGameState() {
	gs.mutex.Lock()
	gs.validateAndRecoverState()
	gameStateMsg := gs.buildGameStatePayloadLocked()
	data, err := json.Marshal(gameStateMsg)
	gs.mutex.Unlock()

	if err != nil {
		log.Printf("Game state marshal error: %v", err)
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
