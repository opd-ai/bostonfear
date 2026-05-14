# TASK DESCRIPTION:
Build a functional Arkham Horror-themed multiplayer web game implementing all 5 core mechanics with Go WebSocket server and Go/Ebitengine client supporting 1-6 concurrent players following idiomatic Go conventions.

## CONTEXT:
Arkham Horror board game features investigators managing resources while exploring locations and facing supernatural threats. Target intermediate developers learning client-server WebSocket architecture with cooperative gameplay mechanics. Implementation prioritizes functional completeness, proper Go conventions, and interface-based design for enhanced testability.

**Design Decision**: This project uses Go/Ebitengine for all client platforms (desktop, WASM, mobile) rather than JavaScript/Canvas. This provides type safety, code reuse across platforms, and consistent behavior. The WASM build compiles Go to WebAssembly—it is not a JavaScript reimplementation.

## INSTRUCTIONS:

### 1. Implement All 5 Core Game Mechanics
A. **Location System**: Create 4 interconnected neighborhoods with movement restrictions between adjacent areas only
B. **Resource Tracking**: Implement Health (1-10), Sanity (1-10), and Clues (0-5) with gain/loss actions per investigator  
C. **Action System**: Allow 2 actions per turn from defined set: Move Location, Gather Resources, Investigate, Cast Ward
D. **Doom Counter**: Maintain global doom tracker (0-12) that increments on failed dice rolls or turn timeouts
E. **Dice Resolution**: Implement 3-sided dice outcomes (Success/Blank/Tentacle) with configurable difficulty thresholds (1-3 successes required)

### 2. Mechanic Integration Requirements
A. **Cross-System Dependencies**:
   1. Dice Resolution system must support Investigate actions (requiring 1-2 successes) and Cast Ward actions (requiring 2-3 successes)
   2. Failed dice rolls containing Tentacle results must increment the global Doom Counter
   3. Resource Tracking must validate resource costs for actions (e.g., Cast Ward costs 1 Sanity)
   4. Location System must enforce movement restrictions based on current player position

B. **State Consistency Validation**:
   1. Ensure turn order progresses correctly after each player completes exactly 2 actions
   2. Validate resource bounds (Health/Sanity 1-10, Clues 0-5, Doom 0-12) with appropriate game-ending conditions
   3. Maintain location adjacency rules preventing invalid movement attempts
   4. Synchronize all connected clients when any mechanic state changes

### 3. Go Coding Standards and Architecture
A. **Idiomatic Go Requirements**:
   1. Follow Go naming conventions with exported/unexported identifiers appropriately
   2. Implement proper Go-style error handling with explicit error returns and nil checks
   3. Use goroutines and channels for concurrent WebSocket connection management
   4. Leverage Go's garbage collection instead of manual memory management

B. **Network Interface Requirements** (Critical - Must Follow):
   1. Use `net.Conn` interface instead of concrete `net.TCPConn` or WebSocket-specific types
   2. Use `net.Listener` interface instead of `net.TCPListener` for server setup
   3. Use `net.Addr` interface instead of concrete address types like `net.TCPAddr`
   4. Implement connection handling through interface abstractions to enable testing with mocks

### 4. Technical Implementation Requirements
A. **Go Server Implementation**:
   1. Create WebSocket handler using `net.Conn` interface accepting 1-6 concurrent player connections with unique player IDs
   2. Maintain centralized game state with turn order enforcement using goroutines and channels for concurrency safety
   3. Broadcast state updates to all connected clients within 500ms using channel-based communication
   4. Handle connection drops with 30-second reconnection timeout using proper Go error handling patterns

B. **Go/Ebitengine Client Implementation**:
   1. Establish WebSocket connection with automatic reconnection attempts every 5 seconds on failure
   2. Render game state using Ebitengine (1280×720 logical resolution) showing player positions and current resources
   3. Capture player input for available actions and transmit to server with player ID validation
   4. Display current player turn indicator and action availability status
   5. Support cross-platform deployment (desktop, WASM, mobile) from single Go codebase

### 5. Required Project Structure
A. **File Organization**: Create `/cmd/server/main.go`, `/cmd/desktop/main.go`, `/cmd/web/main.go`, `/client/ebiten/`
B. **Dependencies**: Go standard library + gorilla/websocket package, hajimehoshi/ebiten/v2 for client rendering
C. **Setup Documentation**: Include 3-step README with: dependency installation, server startup, client access commands

## FORMATTING REQUIREMENTS:
- Implement JSON message protocol with these required message types: `gameState`, `playerAction`, `gameUpdate`, `connectionStatus`, `diceResult`
- Include Go-style error handling with explicit error checking and propagation for: WebSocket connection failures, invalid player actions, game state corruption
- Add code comments explaining each game mechanic implementation, mechanic interactions, Go interface usage, and state validation logic
- Provide visual feedback for: current player turn, available actions, resource levels, doom counter, win/lose conditions
- Use proper Go package structure with clear separation of concerns between network handling, game mechanics, and state management

## QUALITY CHECKS:
1. **Complete Mechanic Implementation**: Are all 5 core mechanics (Location, Resources, Actions, Doom, Dice) fully functional with proper validation?
2. **Mechanic Integration**: Do dice rolls properly affect doom counter, do actions consume appropriate resources, do location restrictions work correctly?
3. **Multi-player Validation**: Can 3 players connect simultaneously, take sequential turns with 2 actions each, and observe real-time state updates across all clients? Can a player join a game already in progress?
4. **Go Convention Adherence**: Does the code follow idiomatic Go patterns with proper interface usage, error handling, and concurrency management?
5. **Network Interface Compliance**: Are all network operations implemented using `net.Conn`, `net.Listener`, and `net.Addr` interfaces instead of concrete types?
6. **Setup Verification**: Can the project run successfully on a clean development environment following only the README instructions?
7. **Performance Standards**: Does the server maintain stable operation with 6 concurrent players performing actions every 10 seconds for 15 minutes?

## EXAMPLES:
**Complete Game Flow Sequence**: Player 1 uses action 1 to move from "Downtown" to "University" (Location System validates adjacency), then uses action 2 to Investigate requiring 2 dice successes (Action System calls Dice Resolution). Dice results: Success, Blank, Tentacle. Investigation fails due to insufficient successes (Resource Tracking - no clue gained), Tentacle result increments global doom counter by 1 (Doom Counter system). Turn advances to Player 2. All connected clients receive updated game state showing new player location, unchanged resources, and increased doom level within 500ms.

**Go Interface Usage**:
```go
// Correct: Use interface types for network operations
func handleConnection(conn net.Conn) error {
    // Connection handling logic
    return nil
}

func setupServer(listener net.Listener) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            return // Proper Go error handling
        }
        go handleConnection(conn) // Goroutine for concurrency
    }
}
```

**Enhanced JSON Protocol Sample**:
```json
// Client to Server - Player Action
{"type": "playerAction", "playerId": "player1", "action": "investigate", "target": "University"}

// Server to All Clients - Complete Game State Update  
{"type": "gameState", "currentPlayer": "player2", "doom": 5, "players": {"player1": {"location": "University", "health": 8, "sanity": 6, "clues": 2, "actionsRemaining": 0}}, "availableActions": ["move", "gather", "investigate", "ward"], "locations": ["Downtown", "University", "Rivertown", "Northside"], "gamePhase": "playerTurn"}
```

