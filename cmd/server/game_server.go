package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GameServer manages the central game state
// Moved from: main.go
type GameServer struct {
	gameState   *GameState
	connections map[string]net.Conn        // Using net.Conn interface
	wsConns     map[string]*websocket.Conn // Internal WebSocket connections
	playerConns map[string]net.Conn        // Player ID to connection mapping
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
	broadcastCh chan []byte
	actionCh    chan PlayerActionMessage
	shutdownCh  chan struct{}
	validator   *GameStateValidator // Enhanced error recovery system
}

// NewGameServer creates a new game server instance
// Moved from: main.go
func NewGameServer() *GameServer {
	return &GameServer{
		gameState: &GameState{
			Players:     make(map[string]*Player),
			Doom:        0,
			GamePhase:   "waiting",
			TurnOrder:   []string{},
			GameStarted: false,
		},
		connections: make(map[string]net.Conn),
		wsConns:     make(map[string]*websocket.Conn),
		playerConns: make(map[string]net.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		broadcastCh: make(chan []byte, 100),
		actionCh:    make(chan PlayerActionMessage, 100),
		shutdownCh:  make(chan struct{}),
		validator:   NewGameStateValidator(), // Initialize error recovery system
	}
}

// Start initializes the game server with goroutines for concurrent handling
// Moved from: main.go
func (gs *GameServer) Start() error {
	// Start broadcast goroutine
	go gs.broadcastHandler()
	// Start action processing goroutine
	go gs.actionHandler()

	log.Printf("Game server started with broadcast and action handlers")
	return nil
}

// validateResources ensures resources stay within bounds
// Moved from: main.go
func (gs *GameServer) validateResources(resources *Resources) {
	if resources.Health < 1 {
		resources.Health = 1
	}
	if resources.Health > 10 {
		resources.Health = 10
	}
	if resources.Sanity < 1 {
		resources.Sanity = 1
	}
	if resources.Sanity > 10 {
		resources.Sanity = 10
	}
	if resources.Clues < 0 {
		resources.Clues = 0
	}
	if resources.Clues > 5 {
		resources.Clues = 5
	}
}

// validateMovement checks if movement between locations is allowed
// Moved from: main.go
func (gs *GameServer) validateMovement(from, to Location) bool {
	adjacentLocations, exists := locationAdjacency[from]
	if !exists {
		return false
	}

	for _, location := range adjacentLocations {
		if location == to {
			return true
		}
	}
	return false
}

// rollDice performs dice resolution with configurable difficulty
// Returns: dice results, successes, tentacles
// Moved from: main.go
func (gs *GameServer) rollDice(numDice int) ([]DiceResult, int, int) {
	if numDice <= 0 {
		return []DiceResult{}, 0, 0
	}

	results := make([]DiceResult, numDice)
	successes := 0
	tentacles := 0

	for i := 0; i < numDice; i++ {
		roll := rand.Intn(3) // 0, 1, 2
		switch roll {
		case 0:
			results[i] = DiceSuccess
			successes++
		case 1:
			results[i] = DiceBlank
		case 2:
			results[i] = DiceTentacle
			tentacles++
		}
	}

	return results, successes, tentacles
}

// processAction handles individual player actions with mechanic integration
// Moved from: main.go
func (gs *GameServer) processAction(action PlayerActionMessage) error {
	gs.mutex.Lock()
	defer gs.mutex.Unlock()

	// Validate game state
	if gs.gameState.GamePhase != "playing" {
		return fmt.Errorf("game is not in playing state")
	}

	player, exists := gs.gameState.Players[action.PlayerID]
	if !exists {
		return fmt.Errorf("player %s not found", action.PlayerID)
	}

	// Validate it's the player's turn and they have actions remaining
	if gs.gameState.CurrentPlayer != action.PlayerID {
		return fmt.Errorf("not player %s's turn (current: %s)", action.PlayerID, gs.gameState.CurrentPlayer)
	}

	if player.ActionsRemaining <= 0 {
		return fmt.Errorf("player %s has no actions remaining", action.PlayerID)
	}

	// Validate action type
	validActions := []ActionType{ActionMove, ActionGather, ActionInvestigate, ActionCastWard}
	isValidAction := false
	for _, validAction := range validActions {
		if action.Action == validAction {
			isValidAction = true
			break
		}
	}
	if !isValidAction {
		return fmt.Errorf("invalid action type: %s", action.Action)
	}

	var diceResult *DiceResultMessage
	doomIncrease := 0

	switch action.Action {
	case ActionMove:
		targetLocation := Location(action.Target)
		if !gs.validateMovement(player.Location, targetLocation) {
			return fmt.Errorf("invalid movement from %s to %s", player.Location, targetLocation)
		}
		player.Location = targetLocation

	case ActionGather:
		// Gather resources with dice roll
		results, successes, tentacles := gs.rollDice(2)
		if successes >= 1 {
			player.Resources.Health = min(player.Resources.Health+1, 10)
			player.Resources.Sanity = min(player.Resources.Sanity+1, 10)
		}
		if tentacles > 0 {
			doomIncrease = tentacles
		}

		diceResult = &DiceResultMessage{
			Type:         "diceResult",
			PlayerID:     action.PlayerID,
			Action:       action.Action,
			Results:      results,
			Successes:    successes,
			Tentacles:    tentacles,
			Success:      successes >= 1,
			DoomIncrease: doomIncrease,
		}

	case ActionInvestigate:
		// Investigate requires 1-2 successes
		requiredSuccesses := 2
		results, successes, tentacles := gs.rollDice(3)

		success := successes >= requiredSuccesses
		if success {
			player.Resources.Clues = min(player.Resources.Clues+1, 5)
		}
		if tentacles > 0 {
			doomIncrease = tentacles
		}

		diceResult = &DiceResultMessage{
			Type:         "diceResult",
			PlayerID:     action.PlayerID,
			Action:       action.Action,
			Results:      results,
			Successes:    successes,
			Tentacles:    tentacles,
			Success:      success,
			DoomIncrease: doomIncrease,
		}

	case ActionCastWard:
		// Cast Ward costs 1 Sanity and requires 2-3 successes
		if player.Resources.Sanity <= 1 {
			return fmt.Errorf("insufficient sanity to cast ward")
		}

		player.Resources.Sanity--
		requiredSuccesses := 3
		results, successes, tentacles := gs.rollDice(3)

		success := successes >= requiredSuccesses
		if success {
			// Ward success reduces doom counter
			gs.gameState.Doom = max(gs.gameState.Doom-2, 0)
		}
		if tentacles > 0 {
			doomIncrease = tentacles
		}

		diceResult = &DiceResultMessage{
			Type:         "diceResult",
			PlayerID:     action.PlayerID,
			Action:       action.Action,
			Results:      results,
			Successes:    successes,
			Tentacles:    tentacles,
			Success:      success,
			DoomIncrease: doomIncrease,
		}
	}

	// Apply doom increase from tentacle results
	if doomIncrease > 0 {
		gs.gameState.Doom = min(gs.gameState.Doom+doomIncrease, 12)
	}

	// Decrement actions remaining
	player.ActionsRemaining--

	// Validate resources after action
	gs.validateResources(&player.Resources)

	// Check win/lose conditions
	gs.checkGameEndConditions()

	// Advance turn if player has no actions left
	if player.ActionsRemaining == 0 {
		gs.advanceTurn()
	}

	// Broadcast dice result if applicable
	if diceResult != nil {
		diceData, _ := json.Marshal(diceResult)
		gs.broadcastCh <- diceData
	}

	// Broadcast updated game state
	gs.broadcastGameState()

	return nil
}

// advanceTurn progresses to the next player's turn
// Moved from: main.go
func (gs *GameServer) advanceTurn() {
	if len(gs.gameState.TurnOrder) == 0 {
		return
	}

	// Find current player index
	currentIndex := -1
	for i, playerID := range gs.gameState.TurnOrder {
		if playerID == gs.gameState.CurrentPlayer {
			currentIndex = i
			break
		}
	}

	// Move to next player
	nextIndex := (currentIndex + 1) % len(gs.gameState.TurnOrder)
	gs.gameState.CurrentPlayer = gs.gameState.TurnOrder[nextIndex]

	// Reset actions for new turn
	if player, exists := gs.gameState.Players[gs.gameState.CurrentPlayer]; exists {
		player.ActionsRemaining = 2
	}
}

// checkGameEndConditions evaluates win/lose states
// Moved from: main.go
func (gs *GameServer) checkGameEndConditions() {
	// Lose condition: Doom reaches 12
	if gs.gameState.Doom >= 12 {
		gs.gameState.LoseCondition = true
		gs.gameState.GamePhase = "ended"
		log.Printf("Game ended: Doom counter reached 12")
		return
	}

	// Win condition: Cooperative victory based on total clues collected
	// Scale based on number of players: 2 players = 8 clues, 3 players = 12 clues, 4 players = 16 clues
	totalClues := 0
	playerCount := len(gs.gameState.Players)
	for _, player := range gs.gameState.Players {
		totalClues += player.Resources.Clues
	}

	requiredClues := playerCount * 4 // 4 clues per player for victory
	if totalClues >= requiredClues {
		gs.gameState.WinCondition = true
		gs.gameState.GamePhase = "ended"
		log.Printf("Game ended: Victory! Players collected %d/%d clues", totalClues, requiredClues)
	}
}

// handleConnection manages WebSocket connections using net.Conn interface
// Moved from: main.go
func (gs *GameServer) handleConnection(conn net.Conn) error {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	// Set connection timeout for 30 seconds as specified in requirements
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}

	// For this implementation, we need to cast back to WebSocket for message handling
	// In a production environment, you'd implement a proper abstraction layer
	wsConn, ok := gs.wsConns[conn.RemoteAddr().String()]
	if !ok {
		return fmt.Errorf("websocket connection not found for %s", conn.RemoteAddr().String())
	}

	// Generate unique player ID with better uniqueness
	playerID := fmt.Sprintf("player_%d", time.Now().UnixNano())

	// Add player to game with proper validation
	gs.mutex.Lock()
	if len(gs.gameState.Players) >= 4 {
		gs.mutex.Unlock()
		return fmt.Errorf("game is full (max 4 players)")
	}

	// Initialize new player
	newPlayer := &Player{
		ID:       playerID,
		Location: Downtown, // Start at Downtown
		Resources: Resources{
			Health: 10,
			Sanity: 10,
			Clues:  0,
		},
		ActionsRemaining: 0,
		Connected:        true,
	}

	gs.gameState.Players[playerID] = newPlayer
	gs.gameState.TurnOrder = append(gs.gameState.TurnOrder, playerID)

	// Store player connection mapping for proper net.Conn interface usage
	gs.playerConns[playerID] = conn

	// Start game if we have at least 2 players
	if len(gs.gameState.Players) >= 2 && !gs.gameState.GameStarted {
		gs.gameState.GameStarted = true
		gs.gameState.GamePhase = "playing"
		gs.gameState.CurrentPlayer = gs.gameState.TurnOrder[0]
		gs.gameState.Players[gs.gameState.CurrentPlayer].ActionsRemaining = 2
	}

	gs.mutex.Unlock()

	// Send connection status
	connectionMsg := map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": playerID,
		"status":   "connected",
	}
	connData, _ := json.Marshal(connectionMsg)
	wsConn.WriteMessage(websocket.TextMessage, connData)

	// Broadcast updated game state
	gs.broadcastGameState()

	// Handle incoming messages with timeout
	for {
		// Reset read deadline for each message (30-second timeout)
		if err := wsConn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			log.Printf("Failed to set read deadline: %v", err)
		}

		_, messageData, err := wsConn.ReadMessage()
		if err != nil {
			// Check if it's a timeout error
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Connection timeout for player %s", playerID)
				// Increment doom counter on timeout as per requirements
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

		var actionMsg PlayerActionMessage
		if err := json.Unmarshal(messageData, &actionMsg); err != nil {
			log.Printf("Message unmarshal error: %v", err)
			continue
		}

		// Validate action message
		if actionMsg.PlayerID != playerID {
			log.Printf("Invalid player ID in action: expected %s, got %s", playerID, actionMsg.PlayerID)
			continue
		}

		// Process action through channel
		gs.actionCh <- actionMsg
	}

	// Handle disconnection
	gs.mutex.Lock()
	if player, exists := gs.gameState.Players[playerID]; exists {
		player.Connected = false
	}
	gs.mutex.Unlock()

	// Remove connections using proper cleanup
	addrStr := conn.RemoteAddr().String()
	delete(gs.connections, addrStr)
	delete(gs.wsConns, addrStr)
	delete(gs.playerConns, playerID)

	gs.broadcastGameState()

	return nil
}

// handleWebSocket handles WebSocket upgrade and connection setup
func (gs *GameServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	wsConn, err := gs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create connection wrapper implementing net.Conn interface
	addr := wsConn.RemoteAddr()
	connWrapper := NewConnectionWrapper(wsConn, addr)

	// Store connections with proper interface usage
	gs.mutex.Lock()
	addrStr := addr.String()
	gs.connections[addrStr] = connWrapper
	gs.wsConns[addrStr] = wsConn
	gs.mutex.Unlock()

	// Handle connection in separate goroutine
	go func() {
		if err := gs.handleConnection(connWrapper); err != nil {
			log.Printf("Connection handling error: %v", err)
		}
	}()
}

// broadcastHandler processes broadcast messages to all connected clients
// Moved from: main.go
func (gs *GameServer) broadcastHandler() {
	for {
		select {
		case message := <-gs.broadcastCh:
			gs.mutex.RLock()
			for _, wsConn := range gs.wsConns {
				if err := wsConn.WriteMessage(websocket.TextMessage, message); err != nil {
					log.Printf("Broadcast error: %v", err)
				}
			}
			gs.mutex.RUnlock()
		case <-gs.shutdownCh:
			log.Printf("Broadcast handler shutting down")
			return
		}
	}
}

// actionHandler processes player actions through channel
// Moved from: main.go
func (gs *GameServer) actionHandler() {
	for {
		select {
		case action := <-gs.actionCh:
			if err := gs.processAction(action); err != nil {
				log.Printf("Action processing error: %v", err)
			}
		case <-gs.shutdownCh:
			log.Printf("Action handler shutting down")
			return
		}
	}
}

// broadcastGameState sends current game state to all connected clients
// Moved from: main.go
func (gs *GameServer) broadcastGameState() {
	gs.mutex.RLock()
	gameStateMsg := map[string]interface{}{
		"type": "gameState",
		"data": gs.gameState,
	}
	gs.mutex.RUnlock()

	data, err := json.Marshal(gameStateMsg)
	if err != nil {
		log.Printf("Game state marshal error: %v", err)
		return
	}

	// Non-blocking send to broadcast channel
	select {
	case gs.broadcastCh <- data:
		// Successfully queued broadcast
	default:
		log.Printf("Broadcast channel full, dropping game state update")
	}
}
