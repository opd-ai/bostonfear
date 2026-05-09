// Package serverengine orchestrates the Arkham Horror multiplayer game server runtime.
//
// This is the core game logic package that manages game state, processes player actions,
// handles connection lifecycles, and broadcasts state updates to all connected clients.
//
// Architecture:
//
// GameServer is the central coordinator, holding:
//   - gameState (*GameState): current board position, players, resources, doom counter
//   - connections (map[string]net.Conn): active WebSocket connections by player ID
//   - actionCh (chan PlayerActionMessage): input channel for player actions
//   - broadcastCh (chan []byte): output channel for JSON state updates
//   - broadcaster (Broadcaster interface): decouples channel I/O from game logic
//   - validator (StateValidator interface): checks game state integrity and corruption
//
// Key Design Patterns:
//
//  1. Interface-Based Networking: GameServer accepts net.Conn (interface) instead of
//     concrete TCP/WebSocket types. This enables testing with mocks and swapping
//     transport protocols without touching game logic.
//
// 2. Concurrent Handlers: Three goroutines handle concurrent workloads:
//
//   - actionHandler: consumes player actions from actionCh, applies effects, advances turns
//
//   - broadcastHandler: consumes state updates from broadcastCh, writes to WebSocket
//
//   - runMythosPhase: spawned per-turn to manage global state transitions
//     All access gameState through a single RWMutex to prevent corruption.
//
//     3. Module System: The package registers game families (Arkham, Elder Sign, etc.)
//     via a Module registry in serverengine/common/contracts. Each module provides
//     an Engine that implements a well-defined interface, allowing future games to
//     plug in without modifying the transport or server startup code.
//
// File Organization:
//
//   - game_server.go: GameServer struct, Start(), HandleConnection(), and initialization
//   - game_types.go: Type aliases to protocol package (Location, ActionType, etc.)
//   - game_constants.go: Game setup variables, difficulty configs, investigator abilities
//   - game_mechanics.go: Resource validation, defeat/recovery, action dispatch
//   - actions.go: Twelve action performers (Move, Gather, Investigate, CastWard, etc.)
//   - dice.go: Dice rolling (rollDice, rollDicePool) with focus spending
//   - mythos.go: Mythos Phase execution, turn advancement, event/enemy spawning
//   - broadcast.go: State broadcasting and message serialization
//   - connection_quality.go: Ping/pong latency measurement and packet loss tracking
//   - health.go: Health snapshots, corruption detection, recovery logic
//   - error_recovery.go: GameStateValidator, state corruption detection and repair
//   - metrics.go: Performance metrics collection (latency, throughput, memory)
//   - interfaces.go: Broadcaster and StateValidator interfaces
//   - observability.go: Methods for metrics emission (performance, connection analytics)
//
// Core Mechanics Flow:
//
//  1. Player submits PlayerActionMessage via WebSocket
//  2. transport/ws.NewWebSocketHandler wraps the connection as net.Conn
//  3. GameServer.HandleConnection reads messages and routes to actionCh
//  4. actionHandler retrieves the message and calls processActionCore:
//     a. Validates player turn, resources, and action legality
//     b. Calls dispatchAction to execute the action (Move, Gather, etc.)
//     c. Applies effects to gameState (location, resources, doom)
//     d. Decrements player.ActionsRemaining
//     e. If ActionsRemaining == 0, calls advanceTurn()
//  5. advanceTurn cycles CurrentPlayer to next active player
//     If all players completed a round, calls runMythosPhase()
//  6. runMythosPhase draws events, places enemies, increments doom, checks win/lose
//  7. broadcastHandler sends updated GameState to all connected clients
//
// State Invariants (enforced by validateResources and checkInvestigatorDefeat):
//
//   - Health ∈ [0, 10]: Drop to 0 → Defeated
//   - Sanity ∈ [0, 10]: Drop to 0 → Defeated
//   - Clues ∈ [0, 5]: Capped at 5 per investigator
//   - Doom ∈ [0, 12]: Reach 12 → Game Lost
//   - ActionsRemaining ∈ [0, 2]: Enforced per turn
//   - TurnOrder ⊆ Players: All turn-order entries must exist in player map
//
// Dice Resolution (rollDicePool):
//
//   - Each die is 3-sided: Success, Blank, or Tentacle
//   - Players can spend Focus tokens to add dice or reroll Blanks
//   - Successes are counted per action threshold (1-3 required)
//   - Tentacles always increment Doom by 1 (unconditionally)
//   - Example: [Success, Blank, Tentacle] with threshold 2
//     → 1 success < 2 required → action fails → +1 doom from tentacle
//
// Testing & Mocking:
//
// All major interfaces (Broadcaster, StateValidator, SessionEngine in transport/ws)
// accept mockable implementations. Tests create GameServer instances with
// mock broadcasters to verify state transitions without WebSocket I/O.
// See serverengine/*_test.go files for examples.
//
// Performance Targets (from README.md):
//
//   - Broadcast Latency: < 500ms from action to all clients
//   - Maximum Players: 6 (AH3e core rulebook range)
//   - Sustained Operation: 6 players for 15+ minutes without deadlock or state corruption
//   - Reconnection Grace: 30-second inactivity timeout; clients use exponential backoff
//
// Educational Value:
//
// This package demonstrates:
//   - Concurrent state management with mutexes and channels
//   - Idiomatic Go error handling (explicit error returns, nil checks)
//   - Interface-based design for testability and extensibility
//   - Domain-driven design (game logic separate from transport)
//   - Metrics collection and observability (without external frameworks)
//   - Gameplay mechanics implementation: resource economy, turn structure, dice resolution
package serverengine
