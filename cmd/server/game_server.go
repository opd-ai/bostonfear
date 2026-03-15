package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
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
			Players:     make(map[string]*Player),
			Doom:        scenario.StartingDoom,
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
		scenario:            scenario,
	}
	// Apply scenario setup (populates decks, sets doom, etc.).
	if scenario.SetupFn != nil {
		scenario.SetupFn(gs.gameState)
	}
	return gs
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

// validateResources ensures resources stay within bounds.
// Health and Sanity may reach 0 (investigator defeat); callers must call
// checkInvestigatorDefeat after this to handle that transition.
func (gs *GameServer) validateResources(resources *Resources) {
	if resources.Health < 0 {
		resources.Health = 0
	}
	if resources.Health > MaxHealth {
		resources.Health = MaxHealth
	}
	if resources.Sanity < 0 {
		resources.Sanity = 0
	}
	if resources.Sanity > MaxSanity {
		resources.Sanity = MaxSanity
	}
	if resources.Clues < 0 {
		resources.Clues = 0
	}
	if resources.Clues > MaxClues {
		resources.Clues = MaxClues
	}
	if resources.Money < 0 {
		resources.Money = 0
	}
	if resources.Money > MaxMoney {
		resources.Money = MaxMoney
	}
	if resources.Remnants < 0 {
		resources.Remnants = 0
	}
	if resources.Remnants > MaxRemnants {
		resources.Remnants = MaxRemnants
	}
	if resources.Focus < 0 {
		resources.Focus = 0
	}
	if resources.Focus > MaxFocus {
		resources.Focus = MaxFocus
	}
}

// checkInvestigatorDefeat transitions a player to the defeated state when their
// Health or Sanity reaches 0. Defeated players are skipped in turn rotation and
// have their ActionsRemaining zeroed. Caller must hold gs.mutex.
func (gs *GameServer) checkInvestigatorDefeat(playerID string) {
	player, exists := gs.gameState.Players[playerID]
	if !exists || player.Defeated {
		return
	}
	if player.Resources.Health == 0 || player.Resources.Sanity == 0 {
		player.Defeated = true
		player.ActionsRemaining = 0
		log.Printf("Investigator %s defeated (health=%d, sanity=%d)",
			playerID, player.Resources.Health, player.Resources.Sanity)
	}
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
		roll := mathrand.Intn(3) // 0, 1, 2
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
		ActionFocus, ActionResearch, ActionTrade, ActionComponent,
		ActionEncounter,
	} {
		if a == v {
			return true
		}
	}
	return false
}

// dispatchAction routes the action to its specific handler and returns the results.
// Caller must hold gs.mutex.
func (gs *GameServer) dispatchAction(action PlayerActionMessage, player *Player) (*DiceResultMessage, int, string, error) {
	actionResult := "success"
	var diceResult *DiceResultMessage
	var doomIncrease int
	var actionErr error

	switch action.Action {
	case ActionMove:
		actionErr = gs.performMove(player, action.Target)
	case ActionGather:
		diceResult, doomIncrease = gs.performGather(player, action.PlayerID)
		if diceResult != nil && !diceResult.Success {
			actionResult = "fail"
		}
	case ActionInvestigate:
		diceResult, doomIncrease, actionResult = gs.performInvestigate(player, action.PlayerID)
	case ActionCastWard:
		diceResult, doomIncrease, actionResult, actionErr = gs.performCastWard(player, action.PlayerID)
	case ActionFocus:
		gs.performFocus(player)
	case ActionResearch:
		diceResult, doomIncrease, actionResult = gs.performResearch(player, action.PlayerID)
	case ActionTrade:
		actionErr = gs.performTrade(action.PlayerID, action.Target)
	case ActionComponent:
		actionErr = gs.performComponent(player, action.PlayerID)
	case ActionEncounter:
		actionErr = gs.performEncounter(player, action.PlayerID)
	}

	return diceResult, doomIncrease, actionResult, actionErr
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

// performMove executes the Move action: validates adjacency and updates player location.
// The caller (processAction) holds gs.mutex and is responsible for releasing it.
func (gs *GameServer) performMove(player *Player, target string) error {
	targetLocation := Location(target)
	if !gs.validateMovement(player.Location, targetLocation) {
		return fmt.Errorf("invalid movement from %s to %s", player.Location, targetLocation)
	}
	player.Location = targetLocation
	return nil
}

// performGather executes the Gather action: rolls dice and awards resources on success.
// On success: awards +1 Health, +1 Sanity, and +$1 Money (per AH3e §Gather Resources).
// Each Tentacle result increments the doom counter unconditionally.
// Returns the dice result message and the doom increase from any tentacle results.
func (gs *GameServer) performGather(player *Player, playerID string) (*DiceResultMessage, int) {
	results, successes, tentacles := gs.rollDice(2)
	if successes >= 1 {
		player.Resources.Health = min(player.Resources.Health+1, MaxHealth)
		player.Resources.Sanity = min(player.Resources.Sanity+1, MaxSanity)
		player.Resources.Money = min(player.Resources.Money+1, MaxMoney)
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionGather,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= 1,
		DoomIncrease: doomIncrease,
	}, doomIncrease
}

// performInvestigate executes the Investigate action: rolls 3 dice requiring 2 successes.
// Returns the dice result, doom increase, and "success"/"fail" result string.
func (gs *GameServer) performInvestigate(player *Player, playerID string) (*DiceResultMessage, int, string) {
	const requiredSuccesses = 2
	results, successes, tentacles := gs.rollDice(3)
	actionResult := "success"
	if successes >= requiredSuccesses {
		player.Resources.Clues = min(player.Resources.Clues+1, 5)
	} else {
		actionResult = "fail"
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionInvestigate,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= requiredSuccesses,
		DoomIncrease: doomIncrease,
	}, doomIncrease, actionResult
}

// performCastWard executes the Cast Ward action: costs 1 Sanity and rolls 3 dice requiring 3 successes.
// On success, reduces the doom counter by 2. Returns dice result, doom increase, result string, and any error.
// The caller (processAction) holds gs.mutex and is responsible for releasing it.
func (gs *GameServer) performCastWard(player *Player, playerID string) (*DiceResultMessage, int, string, error) {
	if player.Resources.Sanity <= 0 {
		return nil, 0, "", fmt.Errorf("insufficient sanity to cast ward")
	}
	player.Resources.Sanity--
	const requiredSuccesses = 3
	results, successes, tentacles := gs.rollDice(3)
	actionResult := "success"
	if successes >= requiredSuccesses {
		gs.gameState.Doom = max(gs.gameState.Doom-2, 0)
	} else {
		actionResult = "fail"
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionCastWard,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= requiredSuccesses,
		DoomIncrease: doomIncrease,
	}, doomIncrease, actionResult, nil
}

// performFocus awards 1 Focus token to the investigator (AH3e §Focus Action).
// No dice roll is required; Focus tokens may be spent on future dice pools.
// Caller must hold gs.mutex.
func (gs *GameServer) performFocus(player *Player) {
	player.Resources.Focus = min(player.Resources.Focus+1, MaxFocus)
}

// performResearch executes the Research action: extended investigation requiring
// 2 successes from a 3-die roll, rewarding 2 Clues on success.
// Each Tentacle increments the doom counter unconditionally.
// Caller must hold gs.mutex.
func (gs *GameServer) performResearch(player *Player, playerID string) (*DiceResultMessage, int, string) {
	const requiredSuccesses = 2
	results, successes, tentacles := gs.rollDice(3)
	actionResult := "success"
	if successes >= requiredSuccesses {
		player.Resources.Clues = min(player.Resources.Clues+2, MaxClues)
	} else {
		actionResult = "fail"
	}
	doomIncrease := 0
	if tentacles > 0 {
		doomIncrease = tentacles
	}
	return &DiceResultMessage{
		Type:         "diceResult",
		PlayerID:     playerID,
		Action:       ActionResearch,
		Results:      results,
		Successes:    successes,
		Tentacles:    tentacles,
		Success:      successes >= requiredSuccesses,
		DoomIncrease: doomIncrease,
	}, doomIncrease, actionResult
}

// performTrade transfers 1 Clue from the acting player to a co-located target
// (AH3e §Trade Action: investigators at the same location may exchange resources).
// Caller must hold gs.mutex.
func (gs *GameServer) performTrade(fromID, toID string) error {
	from, ok := gs.gameState.Players[fromID]
	if !ok {
		return fmt.Errorf("trade: player %s not found", fromID)
	}
	to, ok := gs.gameState.Players[toID]
	if !ok {
		return fmt.Errorf("trade: target player %s not found", toID)
	}
	if from.Location != to.Location {
		return fmt.Errorf("trade: players must be in the same location")
	}
	if from.Resources.Clues < 1 {
		return fmt.Errorf("trade: no clues to transfer")
	}
	from.Resources.Clues--
	to.Resources.Clues = min(to.Resources.Clues+1, MaxClues)
	return nil
}

// performComponent is a stub for investigator-specific component abilities
// (AH3e §Component Action). Full implementation requires per-investigator ability
// tables; returns ErrNotImplemented until those are added.
// Caller must hold gs.mutex.
func (gs *GameServer) performComponent(_ *Player, playerID string) error {
	// TODO: implement per-investigator component abilities (Phase 6 final polish)
	return fmt.Errorf("component action for player %s: not yet implemented", playerID)
}

// performEncounter draws the top card from the player's current location encounter
// deck and applies its effect (AH3e §Encounter Action).
// Deck is rebuilt from defaults when exhausted.
// Caller must hold gs.mutex.
func (gs *GameServer) performEncounter(player *Player, playerID string) error {
	loc := string(player.Location)
	deck := gs.gameState.EncounterDecks[loc]
	if len(deck) == 0 {
		defaults := defaultEncounterDecks()
		deck = defaults[loc]
		if len(deck) == 0 {
			return fmt.Errorf("no encounter cards for location %s", loc)
		}
	}
	card := deck[0]
	gs.gameState.EncounterDecks[loc] = deck[1:]

	switch card.EffectType {
	case "health_loss":
		player.Resources.Health = max(player.Resources.Health-card.Magnitude, 0)
		gs.validateResources(&player.Resources)
		gs.checkInvestigatorDefeat(playerID)
	case "sanity_loss":
		player.Resources.Sanity = max(player.Resources.Sanity-card.Magnitude, 0)
		gs.validateResources(&player.Resources)
		gs.checkInvestigatorDefeat(playerID)
	case "clue_gain":
		player.Resources.Clues = min(player.Resources.Clues+card.Magnitude, MaxClues)
	case "doom_inc":
		gs.gameState.Doom = min(gs.gameState.Doom+card.Magnitude, 12)
	}
	log.Printf("Encounter at %s for %s: %s (%s %+d)", loc, playerID, card.FlavorText, card.EffectType, card.Magnitude)
	return nil
}

// Disconnected and defeated players are skipped so the game never stalls.
// When all players complete a round, runMythosPhase is called before starting
// the next round (AH3e §Mythos Phase).
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

	// Walk forward through the turn order, skipping disconnected or defeated players.
	// A full rotation without finding an active player means all players are gone.
	total := len(gs.gameState.TurnOrder)
	for i := 1; i <= total; i++ {
		nextIndex := (currentIndex + i) % total
		candidateID := gs.gameState.TurnOrder[nextIndex]
		candidate, exists := gs.gameState.Players[candidateID]
		if exists && candidate.Connected && !candidate.Defeated {
			// If we wrapped back to or past the first player, run Mythos Phase.
			if nextIndex <= currentIndex {
				gs.runMythosPhase()
			}
			gs.gameState.CurrentPlayer = candidateID
			candidate.ActionsRemaining = 2
			return
		}
	}
}

// runMythosPhase executes the AH3e Mythos Phase after all investigators complete
// a round. Steps:
//  1. Draw 2 events from MythosEventDeck (rebuild deck when empty).
//  2. Place each event on its target neighborhood; spread to an adjacent
//     neighborhood if a doom token is already present there.
//  3. Increment doom by 1 for each placed event.
//  4. Draw and resolve a Mythos cup token.
//  5. Restore GamePhase to "playing".
//
// Caller must hold gs.mutex.
func (gs *GameServer) runMythosPhase() {
	gs.gameState.GamePhase = "mythos"
	gs.gameState.MythosEvents = gs.gameState.MythosEvents[:0]

	// Rebuild event deck when exhausted.
	if len(gs.gameState.MythosEventDeck) == 0 {
		gs.gameState.MythosEventDeck = defaultMythosEventDeck()
	}

	// Draw up to 2 events.
	toDraw := 2
	if len(gs.gameState.MythosEventDeck) < toDraw {
		toDraw = len(gs.gameState.MythosEventDeck)
	}
	drawn := gs.gameState.MythosEventDeck[:toDraw]
	gs.gameState.MythosEventDeck = gs.gameState.MythosEventDeck[toDraw:]

	for _, evt := range drawn {
		target := evt.LocationID
		// Spread rule: if target already has a doom token, shift to first adjacent location.
		if gs.gameState.LocationDoomTokens[target] > 0 {
			if adjacent, ok := locationAdjacency[Location(target)]; ok && len(adjacent) > 0 {
				target = string(adjacent[0])
				evt.Spread = true
			}
		}
		gs.gameState.LocationDoomTokens[target]++
		gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
		gs.gameState.MythosEvents = append(gs.gameState.MythosEvents, evt)
		log.Printf("Mythos Phase: event placed at %s (spread=%v)", target, evt.Spread)
	}

	// Draw and resolve mythos cup token.
	gs.gameState.MythosToken = gs.drawMythosToken()
	gs.resolveMythosToken(gs.gameState.MythosToken)

	log.Printf("Mythos Phase complete: doom=%d token=%s", gs.gameState.Doom, gs.gameState.MythosToken)
	gs.gameState.GamePhase = "playing"
	gs.checkGameEndConditions()
}

// drawMythosToken returns a pseudo-random cup token for the Mythos Phase.
func (gs *GameServer) drawMythosToken() string {
	tokens := []string{MythosTokenDoom, MythosTokenBlessing, MythosTokenCurse, MythosTokenBlank}
	return string(tokens[gs.gameState.Doom%len(tokens)])
}

// resolveMythosToken applies the effect of the drawn Mythos cup token.
// Caller must hold gs.mutex.
func (gs *GameServer) resolveMythosToken(token string) {
	switch token {
	case MythosTokenDoom:
		gs.gameState.Doom = min(gs.gameState.Doom+1, 12)
	case MythosTokenBlessing:
		if cur, ok := gs.gameState.Players[gs.gameState.CurrentPlayer]; ok {
			cur.Resources.Health = min(cur.Resources.Health+1, MaxHealth)
		}
	case MythosTokenCurse:
		if cur, ok := gs.gameState.Players[gs.gameState.CurrentPlayer]; ok {
			cur.Resources.Sanity = max(cur.Resources.Sanity-1, 0)
			gs.checkInvestigatorDefeat(gs.gameState.CurrentPlayer)
		}
	}
}

// checkActAdvance evaluates whether the investigators have accumulated enough clues
// to advance the Act deck (AH3e §Act/Agenda).  On act completion the win condition
// is set when the final act card is advanced.
// Caller must hold gs.mutex.
func (gs *GameServer) checkActAdvance() {
	if len(gs.gameState.ActDeck) == 0 {
		return
	}
	totalClues := 0
	for _, p := range gs.gameState.Players {
		totalClues += p.Resources.Clues
	}
	act := gs.gameState.ActDeck[0]
	// Expose threshold for client rendering.
	gs.gameState.RequiredClues = act.ClueThreshold
	if totalClues >= act.ClueThreshold {
		log.Printf("Act advanced: %q (clues=%d)", act.Title, totalClues)
		gs.gameState.ActDeck = gs.gameState.ActDeck[1:]
		if len(gs.gameState.ActDeck) == 0 {
			gs.gameState.WinCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: Victory! Final act completed")
		}
	}
}

// checkAgendaAdvance evaluates whether doom has reached the threshold for the
// current Agenda card (AH3e §Act/Agenda).  Advances to the next agenda, and
// triggers the lose condition when the deck is exhausted.
// Caller must hold gs.mutex.
func (gs *GameServer) checkAgendaAdvance() {
	if len(gs.gameState.AgendaDeck) == 0 {
		return
	}
	agenda := gs.gameState.AgendaDeck[0]
	if gs.gameState.Doom >= agenda.DoomThreshold {
		log.Printf("Agenda advanced: %q (doom=%d)", agenda.Title, gs.gameState.Doom)
		gs.gameState.AgendaDeck = gs.gameState.AgendaDeck[1:]
		if len(gs.gameState.AgendaDeck) == 0 {
			gs.gameState.LoseCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: Final agenda reached — Ancient One awakens")
		}
	}
}

// checkGameEndConditions evaluates win/lose states.
// If the scenario provides custom WinFn/LoseFn, those take precedence;
// otherwise the default deck-driven act/agenda checks are used.
// Increments totalGamesPlayed when the game transitions to "ended".
func (gs *GameServer) checkGameEndConditions() {
	// Hard doom cap — lose immediately if doom reaches 12.
	if gs.gameState.Doom >= 12 {
		gs.gameState.LoseCondition = true
		gs.gameState.GamePhase = "ended"
		atomic.AddInt64(&gs.totalGamesPlayed, 1)
		log.Printf("Game ended: Doom counter reached 12")
		return
	}

	// Scenario-provided lose check (overrides deck logic when non-nil).
	if gs.scenario.LoseFn != nil {
		if gs.scenario.LoseFn(gs.gameState) {
			gs.gameState.LoseCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: scenario lose condition triggered")
			return
		}
	} else {
		gs.checkAgendaAdvance()
		if gs.gameState.LoseCondition {
			return
		}
	}

	// Scenario-provided win check (overrides deck logic when non-nil).
	if gs.scenario.WinFn != nil {
		if gs.scenario.WinFn(gs.gameState) {
			gs.gameState.WinCondition = true
			gs.gameState.GamePhase = "ended"
			atomic.AddInt64(&gs.totalGamesPlayed, 1)
			log.Printf("Game ended: scenario win condition triggered")
		}
	} else {
		gs.checkActAdvance()
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

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}

	gs.mutex.RLock()
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
func (gs *GameServer) runMessageLoop(conn net.Conn, wsConn *websocket.Conn, playerID string) {
	for {
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

// handleHealthCheck provides a health monitoring endpoint.
// Game state is snapshotted under a short RLock; serialization happens outside the lock.
func (gs *GameServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Snapshot required fields under a short read lock.
	gs.mutex.RLock()
	isHealthy := gs.validator.IsGameStateHealthy(gs.gameState)
	playerCount := len(gs.gameState.Players)
	connectionCount := len(gs.connections)
	corruptionHistory := gs.validator.GetCorruptionHistory()
	gamePhase := gs.gameState.GamePhase
	doom := gs.gameState.Doom
	gameStarted := gs.gameState.GameStarted
	perfMetrics := gs.collectPerformanceMetrics()
	connAnalytics := gs.collectConnectionAnalytics()
	gameStats := gs.getGameStatistics()
	alerts := gs.getSystemAlerts()
	gs.mutex.RUnlock()

	// Compute derived fields outside the lock.
	recentCorruptions := 0
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
		"gamePhase":          gamePhase,
		"playerCount":        playerCount,
		"connectionCount":    connectionCount,
		"doomLevel":          doom,
		"gameStarted":        gameStarted,
		"recentCorruptions":  recentCorruptions,
		"isGameStateHealthy": isHealthy,
		"timestamp":          time.Now().Unix(),

		// Enhanced performance metrics
		"performanceMetrics":  perfMetrics,
		"connectionAnalytics": connAnalytics,
		"gameStatistics":      gameStats,

		// System alerts: high memory, slow response, high error rate, critical doom
		"systemAlerts": alerts,
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

// handleMetrics provides Prometheus-compatible metrics export.
// Game state is snapshotted under a short RLock; serialization happens outside the lock.
func (gs *GameServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Snapshot required fields under a short read lock.
	gs.mutex.RLock()
	doom := gs.gameState.Doom
	uptime := time.Since(gs.startTime)
	perfMetrics := gs.collectPerformanceMetrics()
	connAnalytics := gs.collectConnectionAnalytics()
	memMetrics := gs.collectMemoryMetrics()
	gcMetrics := gs.collectGCMetrics()
	throughput := gs.collectMessageThroughput(uptime)
	gs.mutex.RUnlock()

	// Set content type for Prometheus metrics
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Build Prometheus-compatible metrics output outside the lock.
	metrics := buildGameMetrics(perfMetrics, connAnalytics, throughput, doom) +
		buildMemoryMetrics(memMetrics, gcMetrics)

	fmt.Fprint(w, metrics)
}

// buildGameMetrics formats game and connection metrics in Prometheus text format.
func buildGameMetrics(perf PerformanceMetrics, conn ConnectionAnalyticsSimplified, throughput MessageThroughputMetrics, doom int) string {
	lines := []string{
		"# HELP arkham_horror_uptime_seconds Total uptime of the server in seconds",
		"# TYPE arkham_horror_uptime_seconds counter",
		fmt.Sprintf("arkham_horror_uptime_seconds %.2f", perf.Uptime.Seconds()),
		"",
		"# HELP arkham_horror_active_connections Current number of active WebSocket connections",
		"# TYPE arkham_horror_active_connections gauge",
		fmt.Sprintf("arkham_horror_active_connections %d", perf.ActiveConnections),
		"",
		"# HELP arkham_horror_peak_connections Peak number of concurrent connections",
		"# TYPE arkham_horror_peak_connections gauge",
		fmt.Sprintf("arkham_horror_peak_connections %d", perf.PeakConnections),
		"",
		"# HELP arkham_horror_total_connections_total Total connections established since server start",
		"# TYPE arkham_horror_total_connections_total counter",
		fmt.Sprintf("arkham_horror_total_connections_total %d", perf.TotalConnections),
		"",
		"# HELP arkham_horror_connections_per_second Rate of new connections per second",
		"# TYPE arkham_horror_connections_per_second gauge",
		fmt.Sprintf("arkham_horror_connections_per_second %.2f", perf.ConnectionsPerSecond),
		"",
		"# HELP arkham_horror_active_players Current number of active players",
		"# TYPE arkham_horror_active_players gauge",
		fmt.Sprintf("arkham_horror_active_players %d", conn.ActivePlayers),
		"",
		"# HELP arkham_horror_messages_per_second Rate of messages processed per second",
		"# TYPE arkham_horror_messages_per_second gauge",
		fmt.Sprintf("arkham_horror_messages_per_second %.2f", perf.MessagesPerSecond),
		"",
		"# HELP arkham_horror_broadcast_latency_ms Rolling average broadcast write latency in milliseconds",
		"# TYPE arkham_horror_broadcast_latency_ms gauge",
		fmt.Sprintf("arkham_horror_broadcast_latency_ms %.4f", throughput.BroadcastLatency),
		"",
		"# HELP arkham_horror_response_time_ms Current health check response time in milliseconds",
		"# TYPE arkham_horror_response_time_ms gauge",
		fmt.Sprintf("arkham_horror_response_time_ms %.2f", perf.ResponseTimeMs),
		"",
		"# HELP arkham_horror_error_rate_percent Current error rate as percentage",
		"# TYPE arkham_horror_error_rate_percent gauge",
		fmt.Sprintf("arkham_horror_error_rate_percent %.2f", perf.ErrorRate),
		"",
		"# HELP arkham_horror_game_doom_level Current doom counter level",
		"# TYPE arkham_horror_game_doom_level gauge",
		fmt.Sprintf("arkham_horror_game_doom_level %d", doom),
		"",
		"# HELP arkham_horror_games_played_total Total number of games played",
		"# TYPE arkham_horror_games_played_total counter",
		fmt.Sprintf("arkham_horror_games_played_total %d", perf.TotalGamesPlayed),
		"",
		"# HELP arkham_horror_reconnection_rate_percent Player reconnection rate percentage",
		"# TYPE arkham_horror_reconnection_rate_percent gauge",
		fmt.Sprintf("arkham_horror_reconnection_rate_percent %.2f", conn.ReconnectionRate),
		"",
	}
	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

// buildMemoryMetrics formats memory and GC metrics in Prometheus text format.
func buildMemoryMetrics(mem MemoryMetrics, gc GCMetrics) string {
	lines := []string{
		"# HELP arkham_horror_memory_allocated_bytes Currently allocated memory in bytes",
		"# TYPE arkham_horror_memory_allocated_bytes gauge",
		fmt.Sprintf("arkham_horror_memory_allocated_bytes %d", mem.AllocatedBytes),
		"",
		"# HELP arkham_horror_memory_usage_percent Memory usage as percentage of allocated/system",
		"# TYPE arkham_horror_memory_usage_percent gauge",
		fmt.Sprintf("arkham_horror_memory_usage_percent %.2f", mem.MemoryUsagePercent),
		"",
		"# HELP arkham_horror_goroutines Current number of goroutines",
		"# TYPE arkham_horror_goroutines gauge",
		fmt.Sprintf("arkham_horror_goroutines %d", mem.GoroutineCount),
		"",
		"# HELP arkham_horror_gc_collections_total Total number of garbage collections",
		"# TYPE arkham_horror_gc_collections_total counter",
		fmt.Sprintf("arkham_horror_gc_collections_total %d", gc.NumGC),
		"",
		"# HELP arkham_horror_gc_pause_seconds_total Total time spent in garbage collection pauses",
		"# TYPE arkham_horror_gc_pause_seconds_total counter",
		fmt.Sprintf("arkham_horror_gc_pause_seconds_total %.6f", gc.PauseTotal.Seconds()),
		"",
	}
	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
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

	totalPlayers, activePlayers, playerSessions := gs.aggregatePlayerSessions()
	window := time.Now().Add(-5 * time.Minute)
	connectionsIn5Min, disconnectsIn5Min, totalReconnections := gs.countRecentConnectionEvents(window)

	var reconnectionRate float64
	if connectionsIn5Min > 0 {
		reconnectionRate = float64(totalReconnections) / float64(connectionsIn5Min) * 100
	}

	return ConnectionAnalyticsSimplified{
		TotalPlayers:      totalPlayers,
		ActivePlayers:     activePlayers,
		PlayerSessions:    playerSessions,
		AverageLatency:    gs.computeAverageLatency(window),
		ConnectionsIn5Min: connectionsIn5Min,
		DisconnectsIn5Min: disconnectsIn5Min,
		ReconnectionRate:  reconnectionRate,
	}
}

// aggregatePlayerSessions converts the playerSessions map into a slice and counts active players.
// Caller must hold gs.performanceMutex at least for reading.
func (gs *GameServer) aggregatePlayerSessions() (int, int, []PlayerSessionMetricsSimplified) {
	total := len(gs.playerSessions)
	active := 0
	sessions := make([]PlayerSessionMetricsSimplified, 0, total)
	for _, session := range gs.playerSessions {
		if session.IsActive {
			session.SessionLength = time.Since(session.SessionStart)
			active++
		}
		sessions = append(sessions, *session)
	}
	return total, active, sessions
}

// countRecentConnectionEvents counts connect, disconnect, and reconnect events after cutoff.
// Caller must hold gs.performanceMutex at least for reading.
func (gs *GameServer) countRecentConnectionEvents(after time.Time) (connects, disconnects, reconnects int) {
	for _, event := range gs.connectionEvents {
		if !event.Timestamp.After(after) {
			continue
		}
		switch event.EventType {
		case "connect":
			connects++
		case "disconnect":
			disconnects++
		case "reconnect":
			reconnects++
		}
	}
	return connects, disconnects, reconnects
}

// computeAverageLatency returns the mean latency of events with Latency > 0 after cutoff.
// Caller must hold gs.performanceMutex at least for reading.
func (gs *GameServer) computeAverageLatency(after time.Time) float64 {
	var total float64
	count := 0
	for _, event := range gs.connectionEvents {
		if event.Latency > 0 && event.Timestamp.After(after) {
			total += event.Latency
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// collectMemoryMetrics gathers memory usage statistics
func (gs *GameServer) collectMemoryMetrics() MemoryMetrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// Calculate memory usage percentage (approximate)
	memUsagePercent := float64(ms.Alloc) / float64(ms.Sys) * 100
	if memUsagePercent > 100 {
		memUsagePercent = 100
	}

	return MemoryMetrics{
		AllocatedBytes:      ms.Alloc,
		TotalAllocatedBytes: ms.TotalAlloc,
		SystemBytes:         ms.Sys,
		HeapInUse:           ms.HeapInuse,
		HeapReleased:        ms.HeapReleased,
		GoroutineCount:      runtime.NumGoroutine(),
		MemoryUsagePercent:  memUsagePercent,
	}
}

// collectGCMetrics gathers garbage collection performance data
func (gs *GameServer) collectGCMetrics() GCMetrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// Calculate average pause time
	var avgPause time.Duration
	if ms.NumGC > 0 && len(ms.PauseNs) > 0 {
		var totalPause uint64
		recentPauses := int(ms.NumGC)
		if recentPauses > len(ms.PauseNs) {
			recentPauses = len(ms.PauseNs)
		}

		for i := 0; i < recentPauses; i++ {
			totalPause += ms.PauseNs[i]
		}
		avgPause = time.Duration(totalPause / uint64(recentPauses))
	}

	// Get last pause time
	var lastPause time.Duration
	if ms.NumGC > 0 {
		lastPause = time.Duration(ms.PauseNs[(ms.NumGC+255)%256])
	}

	return GCMetrics{
		NumGC:       ms.NumGC,
		PauseTotal:  time.Duration(ms.PauseTotalNs),
		PauseAvg:    avgPause,
		LastPause:   lastPause,
		CPUFraction: ms.GCCPUFraction,
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

// validateAndRecoverState validates the current game state and attempts recovery when
// critical or high-severity errors are found. Caller must hold gs.mutex.
func (gs *GameServer) validateAndRecoverState() {
	errors := gs.validator.ValidateGameState(gs.gameState)
	if len(errors) == 0 {
		return
	}
	log.Printf("Game state validation errors detected: %d errors", len(errors))

	for _, err := range errors {
		if err.Severity == "CRITICAL" || err.Severity == "HIGH" {
			log.Printf("Attempting game state recovery...")
			recovered, recoveryErr := gs.validator.RecoverGameState(gs.gameState, errors)
			if recoveryErr == nil {
				gs.gameState = recovered
				log.Printf("Game state successfully recovered")
			} else {
				log.Printf("Game state recovery failed: %v", recoveryErr)
				atomic.AddInt64(&gs.errorCount, 1)
			}
			return
		}
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

// handlePongMessage processes pong responses and calculates latency.
// The write lock is released before calling broadcastConnectionQuality to
// prevent a deadlock: broadcastConnectionQuality acquires qualityMutex.RLock,
// and Go's sync.RWMutex is not reentrant.
func (gs *GameServer) handlePongMessage(pingMsg PingMessage, receiveTime time.Time) {
	gs.qualityMutex.Lock()
	quality, exists := gs.connectionQualities[pingMsg.PlayerID]
	if !exists {
		gs.qualityMutex.Unlock()
		return
	}

	// Calculate round-trip latency in milliseconds
	latency := float64(receiveTime.Sub(pingMsg.Timestamp).Nanoseconds()) / 1e6
	quality.LatencyMs = latency
	quality.LastPingTime = receiveTime

	// Update quality assessment while still holding the lock
	gs.assessConnectionQuality(pingMsg.PlayerID)
	gs.qualityMutex.Unlock() // release before broadcasting to avoid reentrant lock

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

// sendPingToPlayer sends a ping message to measure latency.
// Guards against nil connections that can appear when a concurrent disconnect
// cleanup removes playerConns[playerID] while this function is running.
func (gs *GameServer) sendPingToPlayer(playerID string) {
	gs.mutex.RLock()
	conn, connExists := gs.playerConns[playerID]
	var wsConn *websocket.Conn
	var wsExists bool
	if connExists && conn != nil {
		wsConn, wsExists = gs.wsConns[conn.RemoteAddr().String()]
	}
	gs.mutex.RUnlock()

	if !connExists || conn == nil || !wsExists {
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

		// Non-blocking send mirrors the broadcastGameState pattern.
		// When the channel is full the quality update is dropped rather than
		// causing the ping goroutine to accumulate blocked sends under load.
		select {
		case gs.broadcastCh <- statusData:
		default:
			log.Printf("Broadcast channel full, dropping quality update for %s", playerID)
		}
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

	totalPlayers, connectedPlayers, totalClues, avgHealth, avgSanity := aggregatePlayerStats(gs.gameState.Players)
	gameProgress := computeGameProgress(totalPlayers, totalClues)
	doomPercent := float64(gs.gameState.Doom) / 12.0 * 100

	return map[string]interface{}{
		"totalPlayers":     totalPlayers,
		"connectedPlayers": connectedPlayers,
		"totalClues":       totalClues,
		"averageHealth":    avgHealth,
		"averageSanity":    avgSanity,
		"gameProgress":     gameProgress,
		"doomThreat":       classifyDoomThreat(doomPercent),
		"doomPercent":      doomPercent,
		"gamePhase":        gs.gameState.GamePhase,
		"gameStarted":      gs.gameState.GameStarted,
	}
}

// aggregatePlayerStats computes per-player totals and averages from the players map.
func aggregatePlayerStats(players map[string]*Player) (total, connected, totalClues int, avgHealth, avgSanity float64) {
	total = len(players)
	for _, p := range players {
		if p.Connected {
			connected++
		}
		totalClues += p.Resources.Clues
		avgHealth += float64(p.Resources.Health)
		avgSanity += float64(p.Resources.Sanity)
	}
	if total > 0 {
		avgHealth /= float64(total)
		avgSanity /= float64(total)
	}
	return total, connected, totalClues, avgHealth, avgSanity
}

// computeGameProgress returns the clue-collection progress (0–100) toward victory.
func computeGameProgress(totalPlayers, totalClues int) float64 {
	if totalPlayers == 0 {
		return 0
	}
	progress := float64(totalClues) / float64(totalPlayers*4) * 100
	if progress > 100 {
		return 100
	}
	return progress
}

// classifyDoomThreat maps a doom percentage to a human-readable threat level.
func classifyDoomThreat(doomPercent float64) string {
	switch {
	case doomPercent > 75:
		return "Critical"
	case doomPercent > 50:
		return "High"
	case doomPercent > 25:
		return "Medium"
	default:
		return "Low"
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
