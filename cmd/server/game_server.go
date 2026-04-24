// Package main is the entry point for the Arkham Horror multiplayer game server.
// This file defines the GameServer struct, its constructors, the Start method,
// and the core action-processing pipeline shared across all game mechanics.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GameServer manages the central game state for the Arkham Horror multiplayer game,
// handling WebSocket connections, player actions, turn management, and state broadcasting.
type GameServer struct {
	gameState   *GameState
	scenario    Scenario                   // scenario configuration for this session
	connections map[string]net.Conn        // Using net.Conn interface
	wsConns     map[string]*websocket.Conn // Internal WebSocket connections
	playerConns map[string]net.Conn        // Player ID to connection mapping
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
	broadcastCh chan []byte
	broadcaster Broadcaster // Interface for sending state updates to all clients
	actionCh    chan PlayerActionMessage
	shutdownCh  chan struct{}
	validator   StateValidator // Interface for game state invariant checking

	// Performance monitoring fields
	startTime         time.Time
	totalConnections  int64
	activeConnections int64 // managed atomically; mirrors len(gs.connections)
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

	// wsWriteMu serialises concurrent WebSocket writes per connection.
	// gorilla/websocket is not safe for concurrent writes; both broadcastHandler
	// and sendPingToPlayer write to the same *websocket.Conn, so each connection
	// needs its own write mutex.
	wsWriteMu map[string]*sync.Mutex
	wsMuMutex sync.Mutex // guards wsWriteMu map itself

	// allowedOrigins is the set of hostname:port values permitted by CheckOrigin.
	// When nil or empty the upgrader falls back to permissive mode (any origin
	// is accepted). Set via SetAllowedOrigins for production deployments.
	allowedOrigins []string
}

// NewGameServer creates a new game server instance using the provided Scenario
// for setup and win/lose conditions. Pass DefaultScenario for standard AH3e play.
func NewGameServer() *GameServer {
	return newGameServerWithScenario(DefaultScenario)
}

// newGameServerWithScenario is the underlying constructor; used in tests to inject
// custom scenarios without changing the public NewGameServer() signature.
func newGameServerWithScenario(scenario Scenario) *GameServer {
	ch := make(chan []byte, 100)
	gs := &GameServer{
		gameState: &GameState{
			Players:            make(map[string]*Player),
			Doom:               scenario.StartingDoom,
			GamePhase:          "waiting",
			TurnOrder:          []string{},
			GameStarted:        false,
			Enemies:            make(map[string]*Enemy),
			OpenGates:          []Gate{},
			LocationDoomTokens: make(map[string]int),
			EncounterDecks:     make(map[string][]EncounterCard),
		},
		connections: make(map[string]net.Conn),
		wsConns:     make(map[string]*websocket.Conn),
		playerConns: make(map[string]net.Conn),
		// CheckOrigin is wired to checkOrigin so that allowed origins can be
		// configured at runtime via SetAllowedOrigins. The default (empty list)
		// accepts any origin, matching the previous behaviour and keeping tests
		// green without requiring explicit origin configuration.
		upgrader: websocket.Upgrader{},
		broadcastCh: ch,
		broadcaster: &channelBroadcaster{ch: ch}, // Inject concrete Broadcaster
		actionCh:    make(chan PlayerActionMessage, 100),
		shutdownCh:  make(chan struct{}),
		validator:   NewGameStateValidator(), // Inject concrete StateValidator

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
		wsWriteMu:           make(map[string]*sync.Mutex),
		scenario:            scenario,
	}
	// Apply scenario setup (populates decks, sets doom, etc.).
	if scenario.SetupFn != nil {
		scenario.SetupFn(gs.gameState)
	}
	// Wire CheckOrigin now that gs is initialised; the closure captures gs.
	gs.upgrader.CheckOrigin = gs.checkOrigin
	return gs
}

// SetAllowedOrigins configures the list of permitted WebSocket upgrade origins.
// Each entry should be a host or host:port string (e.g. "localhost:8080",
// "example.com"). When the list is empty (the default), any origin is accepted
// which is appropriate for local development. For production deployments, set
// this to the specific domain(s) that serve the game client.
//
// Example (from main.go or flags):
//
//	gs.SetAllowedOrigins([]string{"localhost:8080", "mygame.example.com"})
func (gs *GameServer) SetAllowedOrigins(origins []string) {
	gs.allowedOrigins = origins
}

// checkOrigin is the websocket.Upgrader.CheckOrigin implementation.
// It accepts the upgrade when:
//   - allowedOrigins is empty (permissive default — safe for local dev), OR
//   - the request's Origin header host matches one of the allowedOrigins entries.
//
// Comparison is case-insensitive and ignores scheme (http/https).
func (gs *GameServer) checkOrigin(r *http.Request) bool {
	if len(gs.allowedOrigins) == 0 {
		// Permissive default: accept any origin.
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No Origin header (e.g. direct TCP connections, curl); allow.
		return true
	}
	// Strip the scheme to get the host portion for comparison.
	host := origin
	if idx := strings.Index(host, "://"); idx >= 0 {
		host = host[idx+3:]
	}
	// Strip trailing slashes.
	host = strings.TrimRight(host, "/")
	hostLower := strings.ToLower(host)
	for _, allowed := range gs.allowedOrigins {
		if strings.ToLower(allowed) == hostLower {
			return true
		}
	}
	log.Printf("WebSocket upgrade rejected: origin %q not in allowed list", origin)
	return false
}

// connWriteLock returns the per-connection write mutex for addr, creating it if needed.
// This ensures gorilla/websocket's non-concurrent-write constraint is honoured when
// broadcastHandler and sendPingToPlayer both write to the same *websocket.Conn.
func (gs *GameServer) connWriteLock(addr string) *sync.Mutex {
	gs.wsMuMutex.Lock()
	defer gs.wsMuMutex.Unlock()
	if mu, ok := gs.wsWriteMu[addr]; ok {
		return mu
	}
	mu := &sync.Mutex{}
	gs.wsWriteMu[addr] = mu
	return mu
}

// removeConnWriteLock removes the per-connection write mutex when a connection closes.
func (gs *GameServer) removeConnWriteLock(addr string) {
	gs.wsMuMutex.Lock()
	delete(gs.wsWriteMu, addr)
	gs.wsMuMutex.Unlock()
}

// writeToConn serialises a single WebSocket write for addr through its per-connection
// mutex so that concurrent callers (broadcastHandler, sendPingToPlayer) cannot race.
func (gs *GameServer) writeToConn(wsConn *websocket.Conn, addr string, data []byte) error {
	mu := gs.connWriteLock(addr)
	mu.Lock()
	err := wsConn.WriteMessage(websocket.TextMessage, data)
	mu.Unlock()
	return err
}

// Start initializes the game server with goroutines for concurrent handling
// Moved from: main.go
func (gs *GameServer) Start() error {
	// Start broadcast goroutine
	go gs.broadcastHandler()
	// Start action processing goroutine
	go gs.actionHandler()
	// Start zombie-player reaper
	go gs.cleanupDisconnectedPlayers()

	log.Printf("Game server started with broadcast and action handlers")
	return nil
}

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

// processAction handles individual player actions with mechanic integration.
// The mutex is acquired at the start for all state validation and mutation,
// then released before broadcasting so broadcastGameState can re-acquire it.
func (gs *GameServer) processAction(action PlayerActionMessage) error {
	gs.mutex.Lock()

	player, err := gs.validateActionRequest(action)
	if err != nil {
		gs.mutex.Unlock()
		return err
	}

	prevResources := player.Resources
	diceResult, doomIncrease, actionResult, actionErr := gs.dispatchAction(action, player)
	if actionErr != nil {
		gs.mutex.Unlock()
		return actionErr
	}

	if doomIncrease > 0 {
		gs.gameState.Doom = min(gs.gameState.Doom+doomIncrease, 12)
	}

	player.ActionsRemaining--
	gs.trackPlayerSession(action.PlayerID, "action")
	gs.validateResources(&player.Resources)
	gs.checkInvestigatorDefeat(action.PlayerID)
	gs.checkGameEndConditions()
	if player.ActionsRemaining == 0 || player.Defeated {
		gs.advanceTurn()
	}

	gameUpdateMsg := gs.buildGameUpdateMessage(action, actionResult, doomIncrease, prevResources, player.Resources)
	gs.mutex.Unlock()

	gs.broadcastActionResults(gameUpdateMsg, diceResult)
	gs.broadcastGameState()
	return nil
}

// validateActionRequest checks game phase, player existence, turn ownership, and action type.
// Returns an error if the player has been defeated.
// Caller must hold gs.mutex.
func (gs *GameServer) validateActionRequest(action PlayerActionMessage) (*Player, error) {
	if gs.gameState.GamePhase != "playing" {
		return nil, fmt.Errorf("game is not in playing state")
	}
	player, exists := gs.gameState.Players[action.PlayerID]
	if !exists {
		return nil, fmt.Errorf("player %s not found", action.PlayerID)
	}
	if player.Defeated {
		return nil, fmt.Errorf("player %s has been defeated and cannot take actions", action.PlayerID)
	}
	if gs.gameState.CurrentPlayer != action.PlayerID {
		return nil, fmt.Errorf("not player %s's turn (current: %s)", action.PlayerID, gs.gameState.CurrentPlayer)
	}
	if player.ActionsRemaining <= 0 {
		return nil, fmt.Errorf("player %s has no actions remaining", action.PlayerID)
	}
	if !isValidActionType(action.Action) {
		return nil, fmt.Errorf("invalid action type: %s", action.Action)
	}
	return player, nil
}

// isValidActionType returns true when the given action type is one of the nine known actions.
func isValidActionType(a ActionType) bool {
	for _, v := range []ActionType{
		ActionMove, ActionGather, ActionInvestigate, ActionCastWard,
		ActionFocus, ActionResearch, ActionTrade,
		ActionEncounter, ActionComponent, ActionAttack, ActionEvade, ActionCloseGate,
	} {
		if a == v {
			return true
		}
	}
	return false
}

// buildGameUpdateMessage constructs the gameUpdate broadcast message from action results.
// Caller must hold gs.mutex.
func (gs *GameServer) buildGameUpdateMessage(
	action PlayerActionMessage,
	actionResult string,
	doomDelta int,
	prev, curr Resources,
) *GameUpdateMessage {
	return &GameUpdateMessage{
		Type:      "gameUpdate",
		PlayerID:  action.PlayerID,
		Event:     string(action.Action),
		Result:    actionResult,
		DoomDelta: doomDelta,
		ResourceDelta: ResourcesDelta{
			Health: curr.Health - prev.Health,
			Sanity: curr.Sanity - prev.Sanity,
			Clues:  curr.Clues - prev.Clues,
		},
		Timestamp: time.Now(),
	}
}

// broadcastActionResults sends the gameUpdate and optional diceResult to all clients.
func (gs *GameServer) broadcastActionResults(update *GameUpdateMessage, diceResult *DiceResultMessage) {
	if updateData, err := json.Marshal(update); err == nil {
		gs.trySendBroadcast(updateData, "gameUpdate")
	}
	if diceResult != nil {
		diceData, _ := json.Marshal(diceResult)
		gs.trySendBroadcast(diceData, "diceResult")
	}
}

// trySendBroadcast enqueues data via the Broadcaster interface, logging a warning on drop.
func (gs *GameServer) trySendBroadcast(data []byte, msgType string) {
	if err := gs.broadcaster.Broadcast(data); err != nil {
		log.Printf("Broadcast channel full, dropping %s", msgType)
	}
}
