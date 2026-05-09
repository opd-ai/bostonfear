package serverengine

import (
	"encoding/json"
	"log"
	"sync/atomic"
	"time"
)

// broadcastHandler processes broadcast messages to all connected clients.
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
