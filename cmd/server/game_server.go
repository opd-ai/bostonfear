package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// GameServer manages the central game state for the Arkham Horror multiplayer game,
// handling WebSocket connections, player actions, turn management, and state broadcasting.
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

	// Performance monitoring fields
	startTime         time.Time
	totalConnections  int64
	peakConnections   int
	totalGamesPlayed  int64
	totalMessagesSent int64
	totalMessagesRecv int64
	errorCount        int64 // incremented atomically at every error site
	playerSessions    map[string]*PlayerSessionMetricsSimplified
	connectionEvents  []ConnectionEventSimplified
	performanceMutex  sync.RWMutex

	// Broadcast latency ring buffer — stores the last 100 write durations in nanoseconds
	latencySamples     [100]int64
	latencyHead        int
	latencySampleCount int
	latencyMu          sync.Mutex

	// Connection quality monitoring
	connectionQualities map[string]*ConnectionQuality
	pingTimers          map[string]*time.Timer
	qualityMutex        sync.RWMutex
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

		// Initialize performance monitoring
		startTime:         time.Now(),
		totalConnections:  0,
		peakConnections:   0,
		totalGamesPlayed:  0,
		totalMessagesSent: 0,
		totalMessagesRecv: 0,
		playerSessions:    make(map[string]*PlayerSessionMetricsSimplified),
		connectionEvents:  make([]ConnectionEventSimplified, 0),

		// Initialize connection quality monitoring
		connectionQualities: make(map[string]*ConnectionQuality),
		pingTimers:          make(map[string]*time.Timer),
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
	// Snapshot resources before action to compute delta for gameUpdate message
	prevResources := player.Resources
	actionResult := "success"

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
		} else {
			actionResult = "fail"
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
		} else {
			actionResult = "fail"
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

	// Track player action for performance monitoring
	gs.trackPlayerSession(action.PlayerID, "action")

	// Validate resources after action
	gs.validateResources(&player.Resources)

	// Check win/lose conditions
	gs.checkGameEndConditions()

	// Advance turn if player has no actions left
	if player.ActionsRemaining == 0 {
		gs.advanceTurn()
	}

	// Emit gameUpdate event message before the full state snapshot.
	// This satisfies the fifth required JSON protocol message type and lets
	// clients display transient action notifications without waiting for the
	// full gameState payload.
	gameUpdateMsg := &GameUpdateMessage{
		Type:      "gameUpdate",
		PlayerID:  action.PlayerID,
		Event:     string(action.Action),
		Result:    actionResult,
		DoomDelta: doomIncrease,
		ResourceDelta: ResourcesDelta{
			Health: player.Resources.Health - prevResources.Health,
			Sanity: player.Resources.Sanity - prevResources.Sanity,
			Clues:  player.Resources.Clues - prevResources.Clues,
		},
		Timestamp: time.Now(),
	}
	if updateData, err := json.Marshal(gameUpdateMsg); err == nil {
		select {
		case gs.broadcastCh <- updateData:
		default:
			log.Printf("Broadcast channel full, dropping gameUpdate")
		}
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
	// Expose the threshold in GameState so clients can render a win-progress bar
	gs.gameState.RequiredClues = requiredClues
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

	// Track connection for performance monitoring
	gs.trackConnection("connect", playerID, 0)
	gs.trackPlayerSession(playerID, "start")

	// Initialize connection quality monitoring
	gs.initializeConnectionQuality(playerID)

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
		// Reset read deadline for each message via the net.Conn interface (30-second timeout).
		// ConnectionWrapper.SetReadDeadline now delegates to the underlying WebSocket connection.
		if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
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

		// Track message received time for latency calculation
		messageReceiveTime := time.Now()

		var actionMsg PlayerActionMessage
		if err := json.Unmarshal(messageData, &actionMsg); err != nil {
			// Try to parse as ping response
			var pingMsg PingMessage
			if pingErr := json.Unmarshal(messageData, &pingMsg); pingErr == nil && pingMsg.Type == "pong" {
				gs.handlePongMessage(pingMsg, messageReceiveTime)
				continue
			}
			log.Printf("Message unmarshal error: %v", err)
			atomic.AddInt64(&gs.errorCount, 1)
			continue
		}

		// Validate action message
		if actionMsg.PlayerID != playerID {
			log.Printf("Invalid player ID in action: expected %s, got %s", playerID, actionMsg.PlayerID)
			atomic.AddInt64(&gs.errorCount, 1)
			continue
		}

		// Update connection quality based on message timing
		gs.updateConnectionQuality(playerID, messageReceiveTime)

		// Process action through channel
		gs.actionCh <- actionMsg
	}

	// Handle disconnection
	gs.mutex.Lock()
	if player, exists := gs.gameState.Players[playerID]; exists {
		player.Connected = false
	}
	gs.mutex.Unlock()

	// Track disconnection for performance monitoring
	gs.trackConnection("disconnect", playerID, 0)
	gs.trackPlayerSession(playerID, "end")

	// Cleanup connection quality monitoring
	gs.cleanupConnectionQuality(playerID)

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
	log.Printf("New WebSocket connection attempt from %s", r.RemoteAddr)

	wsConn, err := gs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		atomic.AddInt64(&gs.errorCount, 1)
		return
	}

	log.Printf("WebSocket connection established with %s", wsConn.RemoteAddr())

	// Create connection wrapper implementing net.Conn interface
	addr := wsConn.RemoteAddr()
	connWrapper := NewConnectionWrapper(wsConn, addr)

	// Store connections with proper interface usage
	gs.mutex.Lock()
	addrStr := addr.String()
	gs.connections[addrStr] = connWrapper
	gs.wsConns[addrStr] = wsConn
	log.Printf("Stored connection %s, total connections: %d", addrStr, len(gs.connections))
	gs.mutex.Unlock()

	// Handle connection in separate goroutine
	go func() {
		if err := gs.handleConnection(connWrapper); err != nil {
			log.Printf("Connection handling error: %v", err)
		}
	}()
}

// handleHealthCheck provides a health monitoring endpoint
func (gs *GameServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	// Perform comprehensive health check
	isHealthy := gs.validator.IsGameStateHealthy(gs.gameState)
	playerCount := len(gs.gameState.Players)
	connectionCount := len(gs.connections)
	corruptionHistory := gs.validator.GetCorruptionHistory()
	recentCorruptions := 0

	// Count corruptions in last 5 minutes
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
		"gamePhase":          gs.gameState.GamePhase,
		"playerCount":        playerCount,
		"connectionCount":    connectionCount,
		"doomLevel":          gs.gameState.Doom,
		"gameStarted":        gs.gameState.GameStarted,
		"recentCorruptions":  recentCorruptions,
		"isGameStateHealthy": isHealthy,
		"timestamp":          time.Now().Unix(),

		// Enhanced performance metrics
		"performanceMetrics":  gs.collectPerformanceMetrics(),
		"connectionAnalytics": gs.collectConnectionAnalytics(),
		"gameStatistics":      gs.getGameStatistics(),
	}

	w.Header().Set("Content-Type", "application/json")
	if status != "healthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(healthData)
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

// handleMetrics provides Prometheus-compatible metrics export
func (gs *GameServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	// Set content type for Prometheus metrics
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Collect comprehensive metrics
	performanceMetrics := gs.collectPerformanceMetrics()
	connectionAnalytics := gs.collectConnectionAnalytics()
	memoryMetrics := gs.collectMemoryMetrics()
	gcMetrics := gs.collectGCMetrics()

	// Build Prometheus-compatible metrics output
	metrics := []string{
		"# HELP arkham_horror_uptime_seconds Total uptime of the server in seconds",
		"# TYPE arkham_horror_uptime_seconds counter",
		fmt.Sprintf("arkham_horror_uptime_seconds %.2f", performanceMetrics.Uptime.Seconds()),
		"",
		"# HELP arkham_horror_active_connections Current number of active WebSocket connections",
		"# TYPE arkham_horror_active_connections gauge",
		fmt.Sprintf("arkham_horror_active_connections %d", performanceMetrics.ActiveConnections),
		"",
		"# HELP arkham_horror_peak_connections Peak number of concurrent connections",
		"# TYPE arkham_horror_peak_connections gauge",
		fmt.Sprintf("arkham_horror_peak_connections %d", performanceMetrics.PeakConnections),
		"",
		"# HELP arkham_horror_total_connections_total Total connections established since server start",
		"# TYPE arkham_horror_total_connections_total counter",
		fmt.Sprintf("arkham_horror_total_connections_total %d", performanceMetrics.TotalConnections),
		"",
		"# HELP arkham_horror_connections_per_second Rate of new connections per second",
		"# TYPE arkham_horror_connections_per_second gauge",
		fmt.Sprintf("arkham_horror_connections_per_second %.2f", performanceMetrics.ConnectionsPerSecond),
		"",
		"# HELP arkham_horror_active_players Current number of active players",
		"# TYPE arkham_horror_active_players gauge",
		fmt.Sprintf("arkham_horror_active_players %d", connectionAnalytics.ActivePlayers),
		"",
		"# HELP arkham_horror_messages_per_second Rate of messages processed per second",
		"# TYPE arkham_horror_messages_per_second gauge",
		fmt.Sprintf("arkham_horror_messages_per_second %.2f", performanceMetrics.MessagesPerSecond),
		"",
		"# HELP arkham_horror_memory_allocated_bytes Currently allocated memory in bytes",
		"# TYPE arkham_horror_memory_allocated_bytes gauge",
		fmt.Sprintf("arkham_horror_memory_allocated_bytes %d", memoryMetrics.AllocatedBytes),
		"",
		"# HELP arkham_horror_memory_usage_percent Memory usage as percentage of allocated/system",
		"# TYPE arkham_horror_memory_usage_percent gauge",
		fmt.Sprintf("arkham_horror_memory_usage_percent %.2f", memoryMetrics.MemoryUsagePercent),
		"",
		"# HELP arkham_horror_goroutines Current number of goroutines",
		"# TYPE arkham_horror_goroutines gauge",
		fmt.Sprintf("arkham_horror_goroutines %d", memoryMetrics.GoroutineCount),
		"",
		"# HELP arkham_horror_gc_collections_total Total number of garbage collections",
		"# TYPE arkham_horror_gc_collections_total counter",
		fmt.Sprintf("arkham_horror_gc_collections_total %d", gcMetrics.NumGC),
		"",
		"# HELP arkham_horror_gc_pause_seconds_total Total time spent in garbage collection pauses",
		"# TYPE arkham_horror_gc_pause_seconds_total counter",
		fmt.Sprintf("arkham_horror_gc_pause_seconds_total %.6f", gcMetrics.PauseTotal.Seconds()),
		"",
		"# HELP arkham_horror_response_time_ms Current health check response time in milliseconds",
		"# TYPE arkham_horror_response_time_ms gauge",
		fmt.Sprintf("arkham_horror_response_time_ms %.2f", performanceMetrics.ResponseTimeMs),
		"",
		"# HELP arkham_horror_error_rate_percent Current error rate as percentage",
		"# TYPE arkham_horror_error_rate_percent gauge",
		fmt.Sprintf("arkham_horror_error_rate_percent %.2f", performanceMetrics.ErrorRate),
		"",
		"# HELP arkham_horror_game_doom_level Current doom counter level",
		"# TYPE arkham_horror_game_doom_level gauge",
		fmt.Sprintf("arkham_horror_game_doom_level %d", gs.gameState.Doom),
		"",
		"# HELP arkham_horror_games_played_total Total number of games played",
		"# TYPE arkham_horror_games_played_total counter",
		fmt.Sprintf("arkham_horror_games_played_total %d", performanceMetrics.TotalGamesPlayed),
		"",
		"# HELP arkham_horror_reconnection_rate_percent Player reconnection rate percentage",
		"# TYPE arkham_horror_reconnection_rate_percent gauge",
		fmt.Sprintf("arkham_horror_reconnection_rate_percent %.2f", connectionAnalytics.ReconnectionRate),
	}

	// Write metrics to response
	for _, line := range metrics {
		fmt.Fprintln(w, line)
	}
}

// collectPerformanceMetrics gathers comprehensive server performance data
func (gs *GameServer) collectPerformanceMetrics() PerformanceMetrics {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	// Calculate runtime metrics
	uptime := time.Since(gs.startTime)
	activeConnections := len(gs.connections)

	// Calculate connections per second — guard against division by zero on startup
	connectionsPerSecond := 0.0
	if uptime.Seconds() > 0 {
		connectionsPerSecond = float64(gs.totalConnections) / uptime.Seconds()
	}

	// Calculate average session length and active sessions
	var totalSessionTime time.Duration
	activeSessions := 0
	for _, session := range gs.playerSessions {
		sessionDuration := time.Since(session.SessionStart)
		totalSessionTime += sessionDuration
		activeSessions++
	}

	var avgSessionLength time.Duration
	if len(gs.playerSessions) > 0 {
		avgSessionLength = totalSessionTime / time.Duration(len(gs.playerSessions))
	}

	// Calculate messages per second — guard against division by zero on startup
	messagesPerSecond := 0.0
	if uptime.Seconds() > 0 {
		messagesPerSecond = float64(gs.totalMessagesSent+gs.totalMessagesRecv) / uptime.Seconds()
	}

	// Collect memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryStats := MemoryStats{
		AllocMB:      float64(memStats.Alloc) / 1024 / 1024,
		TotalAllocMB: float64(memStats.TotalAlloc) / 1024 / 1024,
		SysMB:        float64(memStats.Sys) / 1024 / 1024,
		NumGC:        memStats.NumGC,
		GCPauseMs:    float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1000000,
	}

	// Calculate response time (simplified - using health check measurement)
	responseTimeMs := gs.measureHealthCheckResponseTime()

	// Calculate error rate (corruption events vs total operations)
	errorRate := gs.calculateErrorRate()

	return PerformanceMetrics{
		Uptime:               uptime,
		ActiveConnections:    activeConnections,
		PeakConnections:      gs.peakConnections,
		TotalConnections:     gs.totalConnections,
		ConnectionsPerSecond: connectionsPerSecond,
		AverageSessionLength: avgSessionLength,
		ActiveSessions:       activeSessions,
		TotalGamesPlayed:     gs.totalGamesPlayed,
		MessagesPerSecond:    messagesPerSecond,
		MemoryUsage:          memoryStats,
		ResponseTimeMs:       responseTimeMs,
		ErrorRate:            errorRate,
	}
}

// collectConnectionAnalytics provides player connection insights
func (gs *GameServer) collectConnectionAnalytics() ConnectionAnalyticsSimplified {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	totalPlayers := len(gs.playerSessions)
	activePlayers := 0

	// Convert session map to slice and update active status
	playerSessions := make([]PlayerSessionMetricsSimplified, 0, len(gs.playerSessions))
	for _, session := range gs.playerSessions {
		// Update session length if still active
		if session.IsActive {
			session.SessionLength = time.Since(session.SessionStart)
			activePlayers++
		}
		playerSessions = append(playerSessions, *session)
	}

	// Calculate recent connection activity (last 5 minutes)
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	connectionsIn5Min := 0
	disconnectsIn5Min := 0
	totalReconnections := 0

	for _, event := range gs.connectionEvents {
		if event.Timestamp.After(fiveMinutesAgo) {
			switch event.EventType {
			case "connect":
				connectionsIn5Min++
			case "disconnect":
				disconnectsIn5Min++
			case "reconnect":
				totalReconnections++
			}
		}
	}

	// Calculate reconnection rate
	var reconnectionRate float64
	if connectionsIn5Min > 0 {
		reconnectionRate = float64(totalReconnections) / float64(connectionsIn5Min) * 100
	}

	// Calculate average latency from recent events
	var totalLatency float64
	latencyCount := 0
	for _, event := range gs.connectionEvents {
		if event.Latency > 0 && event.Timestamp.After(fiveMinutesAgo) {
			totalLatency += event.Latency
			latencyCount++
		}
	}

	var avgLatency float64
	if latencyCount > 0 {
		avgLatency = totalLatency / float64(latencyCount)
	}

	return ConnectionAnalyticsSimplified{
		TotalPlayers:      totalPlayers,
		ActivePlayers:     activePlayers,
		PlayerSessions:    playerSessions,
		AverageLatency:    avgLatency,
		ConnectionsIn5Min: connectionsIn5Min,
		DisconnectsIn5Min: disconnectsIn5Min,
		ReconnectionRate:  reconnectionRate,
	}
}

// collectMemoryMetrics gathers memory usage statistics
func (gs *GameServer) collectMemoryMetrics() MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate memory usage percentage (approximate)
	memUsagePercent := float64(m.Alloc) / float64(m.Sys) * 100
	if memUsagePercent > 100 {
		memUsagePercent = 100
	}

	return MemoryMetrics{
		AllocatedBytes:      m.Alloc,
		TotalAllocatedBytes: m.TotalAlloc,
		SystemBytes:         m.Sys,
		HeapInUse:           m.HeapInuse,
		HeapReleased:        m.HeapReleased,
		GoroutineCount:      runtime.NumGoroutine(),
		MemoryUsagePercent:  memUsagePercent,
	}
}

// collectGCMetrics gathers garbage collection performance data
func (gs *GameServer) collectGCMetrics() GCMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate average pause time
	var avgPause time.Duration
	if m.NumGC > 0 && len(m.PauseNs) > 0 {
		var totalPause uint64
		recentPauses := int(m.NumGC)
		if recentPauses > len(m.PauseNs) {
			recentPauses = len(m.PauseNs)
		}

		for i := 0; i < recentPauses; i++ {
			totalPause += m.PauseNs[i]
		}
		avgPause = time.Duration(totalPause / uint64(recentPauses))
	}

	// Get last pause time
	var lastPause time.Duration
	if m.NumGC > 0 {
		lastPause = time.Duration(m.PauseNs[(m.NumGC+255)%256])
	}

	return GCMetrics{
		NumGC:       m.NumGC,
		PauseTotal:  time.Duration(m.PauseTotalNs),
		PauseAvg:    avgPause,
		LastPause:   lastPause,
		CPUFraction: m.GCCPUFraction,
	}
}

// recordBroadcastLatency stores a single write-duration sample in the ring buffer.
func (gs *GameServer) recordBroadcastLatency(d time.Duration) {
	gs.latencyMu.Lock()
	gs.latencySamples[gs.latencyHead] = d.Nanoseconds()
	gs.latencyHead = (gs.latencyHead + 1) % len(gs.latencySamples)
	if gs.latencySampleCount < len(gs.latencySamples) {
		gs.latencySampleCount++
	}
	gs.latencyMu.Unlock()
}

// averageBroadcastLatencyMs returns the rolling average broadcast latency in milliseconds.
func (gs *GameServer) averageBroadcastLatencyMs() float64 {
	gs.latencyMu.Lock()
	defer gs.latencyMu.Unlock()
	if gs.latencySampleCount == 0 {
		return 0
	}
	var sum int64
	for i := 0; i < gs.latencySampleCount; i++ {
		sum += gs.latencySamples[i]
	}
	return float64(sum) / float64(gs.latencySampleCount) / 1e6
}

// collectMessageThroughput calculates message performance metrics
func (gs *GameServer) collectMessageThroughput(runtime time.Duration) MessageThroughputMetrics {
	gs.performanceMutex.RLock()
	defer gs.performanceMutex.RUnlock()

	// Calculate messages per second — guard against zero uptime on startup
	totalMessages := gs.totalMessagesSent + gs.totalMessagesRecv
	messagesPerSecond := 0.0
	if runtime.Seconds() > 0 {
		messagesPerSecond = float64(totalMessages) / runtime.Seconds()
	}

	broadcastLatency := gs.averageBroadcastLatencyMs()
	return MessageThroughputMetrics{
		MessagesPerSecond:     messagesPerSecond,
		TotalMessagesSent:     gs.totalMessagesSent,
		TotalMessagesReceived: gs.totalMessagesRecv,
		AverageLatency:        broadcastLatency,
		BroadcastLatency:      broadcastLatency,
	}
}

// trackConnection records connection events for analytics
func (gs *GameServer) trackConnection(eventType, playerID string, latency float64) {
	gs.performanceMutex.Lock()
	defer gs.performanceMutex.Unlock()

	event := ConnectionEventSimplified{
		EventType: eventType,
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Latency:   latency,
	}

	gs.connectionEvents = append(gs.connectionEvents, event)

	// Keep only last 1000 events to prevent memory growth
	if len(gs.connectionEvents) > 1000 {
		gs.connectionEvents = gs.connectionEvents[len(gs.connectionEvents)-1000:]
	}

	// Update connection counters
	if eventType == "connect" {
		gs.totalConnections++
		currentConnections := len(gs.connections)
		if currentConnections > gs.peakConnections {
			gs.peakConnections = currentConnections
		}
	}
}

// trackPlayerSession manages player session metrics
func (gs *GameServer) trackPlayerSession(playerID, eventType string) {
	gs.performanceMutex.Lock()
	defer gs.performanceMutex.Unlock()

	switch eventType {
	case "start":
		gs.playerSessions[playerID] = &PlayerSessionMetricsSimplified{
			PlayerID:         playerID,
			SessionStart:     time.Now(),
			SessionLength:    0,
			ActionsPerformed: 0,
			Reconnections:    0,
			LastSeen:         time.Now(),
			IsActive:         true,
		}
	case "end":
		if session, exists := gs.playerSessions[playerID]; exists {
			session.SessionLength = time.Since(session.SessionStart)
			session.IsActive = false
		}
	case "action":
		if session, exists := gs.playerSessions[playerID]; exists {
			session.ActionsPerformed++
			session.LastSeen = time.Now()
		}
	case "reconnect":
		if session, exists := gs.playerSessions[playerID]; exists {
			session.Reconnections++
			session.LastSeen = time.Now()
			session.IsActive = true
		}
	}
}

// trackMessage increments message counters for throughput analysis
func (gs *GameServer) trackMessage(messageType string) {
	gs.performanceMutex.Lock()
	defer gs.performanceMutex.Unlock()

	switch messageType {
	case "sent":
		gs.totalMessagesSent++
	case "received":
		gs.totalMessagesRecv++
	}
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

	// Validate game state before broadcasting with error recovery
	if errors := gs.validator.ValidateGameState(gs.gameState); len(errors) > 0 {
		log.Printf("Game state validation errors detected: %d errors", len(errors))

		// Attempt recovery for critical/high severity errors
		hasCriticalErrors := false
		for _, err := range errors {
			if err.Severity == "CRITICAL" || err.Severity == "HIGH" {
				hasCriticalErrors = true
				break
			}
		}

		if hasCriticalErrors {
			log.Printf("Attempting game state recovery...")
			recoveredState, recoveryErr := gs.validator.RecoverGameState(gs.gameState, errors)
			if recoveryErr == nil {
				gs.gameState = recoveredState
				log.Printf("Game state successfully recovered")
			} else {
				log.Printf("Game state recovery failed: %v", recoveryErr)
				atomic.AddInt64(&gs.errorCount, 1)
			}
		}
	}

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

	// Non-blocking send to broadcast channel
	select {
	case gs.broadcastCh <- data:
		// Successfully queued broadcast
	default:
		log.Printf("Broadcast channel full, dropping game state update")
	}
}

// Helper methods for performance monitoring dashboard

// measureHealthCheckResponseTime measures the response time of health check operations
func (gs *GameServer) measureHealthCheckResponseTime() float64 {
	start := time.Now()

	// Simulate health check operations
	gs.mutex.RLock()
	_ = len(gs.gameState.Players)
	_ = len(gs.connections)
	gs.mutex.RUnlock()

	// Return response time in milliseconds
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

// Connection Quality Management Methods

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

// handlePongMessage processes pong responses and calculates latency
func (gs *GameServer) handlePongMessage(pingMsg PingMessage, receiveTime time.Time) {
	gs.qualityMutex.Lock()
	defer gs.qualityMutex.Unlock()

	quality, exists := gs.connectionQualities[pingMsg.PlayerID]
	if !exists {
		return
	}

	// Calculate round-trip latency
	latency := float64(receiveTime.Sub(pingMsg.Timestamp).Nanoseconds()) / 1000000 // Convert to milliseconds
	quality.LatencyMs = latency
	quality.LastPingTime = receiveTime

	// Update quality assessment
	gs.assessConnectionQuality(pingMsg.PlayerID)

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

// sendPingToPlayer sends a ping message to measure latency
func (gs *GameServer) sendPingToPlayer(playerID string) {
	gs.mutex.RLock()
	wsConn, exists := gs.wsConns[gs.playerConns[playerID].RemoteAddr().String()]
	gs.mutex.RUnlock()

	if !exists {
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

	if err := wsConn.WriteMessage(websocket.TextMessage, pingData); err != nil {
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

		gs.broadcastCh <- statusData
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

// Enhanced monitoring methods for comprehensive dashboard support

// getGameStatistics provides detailed game state analytics
func (gs *GameServer) getGameStatistics() map[string]interface{} {
	gs.mutex.RLock()
	defer gs.mutex.RUnlock()

	// Calculate game statistics
	totalPlayers := len(gs.gameState.Players)
	connectedPlayers := 0
	totalClues := 0
	avgHealth := 0.0
	avgSanity := 0.0

	for _, player := range gs.gameState.Players {
		if player.Connected {
			connectedPlayers++
		}
		totalClues += player.Resources.Clues
		avgHealth += float64(player.Resources.Health)
		avgSanity += float64(player.Resources.Sanity)
	}

	if totalPlayers > 0 {
		avgHealth /= float64(totalPlayers)
		avgSanity /= float64(totalPlayers)
	}

	// Calculate game progress
	gameProgress := 0.0
	if totalPlayers > 0 {
		requiredClues := totalPlayers * 4 // Victory condition
		gameProgress = float64(totalClues) / float64(requiredClues) * 100
		if gameProgress > 100 {
			gameProgress = 100
		}
	}

	// Calculate doom threat level
	doomThreat := "Low"
	doomPercent := float64(gs.gameState.Doom) / 12.0 * 100
	if doomPercent > 75 {
		doomThreat = "Critical"
	} else if doomPercent > 50 {
		doomThreat = "High"
	} else if doomPercent > 25 {
		doomThreat = "Medium"
	}

	return map[string]interface{}{
		"totalPlayers":     totalPlayers,
		"connectedPlayers": connectedPlayers,
		"totalClues":       totalClues,
		"averageHealth":    avgHealth,
		"averageSanity":    avgSanity,
		"gameProgress":     gameProgress,
		"doomThreat":       doomThreat,
		"doomPercent":      doomPercent,
		"gamePhase":        gs.gameState.GamePhase,
		"gameStarted":      gs.gameState.GameStarted,
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

	// Game state alerts
	gs.mutex.RLock()
	doomPercent := float64(gs.gameState.Doom) / 12.0 * 100
	gs.mutex.RUnlock()

	if doomPercent > 80 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "error",
			"message":  fmt.Sprintf("Critical doom level: %d/12 (%.0f%%)", gs.gameState.Doom, doomPercent),
			"severity": "critical",
		})
	} else if doomPercent > 60 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High doom level: %d/12 (%.0f%%)", gs.gameState.Doom, doomPercent),
			"severity": "medium",
		})
	}

	return alerts
}
