# Arkham Horror - Multiplayer Game

A multiplayer implementation of Arkham Horror featuring investigators managing resources while exploring locations and facing supernatural threats. Built with a Go WebSocket server and a Go/Ebitengine game client supporting desktop, web (WASM), and mobile platforms with 1-6 concurrent players. Players can join a game already in progress.

> **Migration in progress:** The client is being migrated from HTML/JS canvas to
> Go/Ebitengine. See `ROADMAP.md` for the phased plan. The WebSocket server and its
> protocol remain unchanged throughout the migration.

## Features

### Core Game Mechanics
1. **Location System**: 4 interconnected neighborhoods (Downtown, University, Rivertown, Northside) with movement restrictions
2. **Resource Tracking**: Health (1-10), Sanity (1-10), and Clues (0-5) with gain/loss mechanics
3. **Action System**: 2 actions per turn from Move, Gather Resources, Investigate, Cast Ward
4. **Doom Counter**: Global doom tracker (0-12) that increments on failed dice rolls
5. **Dice Resolution**: 3-sided dice (Success/Blank/Tentacle) with configurable difficulty thresholds

### Multiplayer Features
- Support for 1-6 concurrent players (AH3e core rulebook range)
- Join a game already in progress — late joiners enter the turn rotation automatically
- Real-time game state synchronization
- Turn-based gameplay with action validation
- Automatic reconnection handling
- WebSocket-based communication

### Performance Monitoring
- **Real-time Dashboard**: Live performance metrics at `/dashboard`
- **Prometheus Metrics**: Export endpoint at `/metrics` for monitoring tools
- **Health Checks**: Comprehensive health status at `/health`
- **Connection Analytics**: Player session tracking and connection insights
- **Memory Monitoring**: Garbage collection and memory usage metrics
- **Error Recovery**: Automated game state validation and corruption detection

## Build Targets

| Platform | Entrypoint | Build Command | Status |
|---|---|---|---|
| **Desktop** (Linux, macOS, Windows) | `cmd/desktop/main.go` | `go build ./cmd/desktop` | Planned (ROADMAP Phase 2) |
| **Web (WASM)** | `cmd/web/main.go` | `GOOS=js GOARCH=wasm go build -o game.wasm ./cmd/web` | Planned (ROADMAP Phase 3) |
| **Mobile** (iOS 16+, Android 10+) | `cmd/mobile/mobile.go` | `ebitenmobile bind -target android ./cmd/mobile` | Planned (ROADMAP Phase 4) |
| **Legacy browser** (current) | `client/index.html` | N/A — served by Go server | Active (to be replaced) |

## Quick Setup

### Step 1: Install Dependencies
```bash
cd /workspaces/bostonfear
go mod tidy
```

### Step 2: Start Server
```bash
cd cmd/server
go run .
```

### Step 3: Access Client

**Legacy browser client** (current — to be replaced by Ebitengine client):
```
http://localhost:8080                # Game client
http://localhost:8080/dashboard      # Performance monitoring dashboard
http://localhost:8080/health         # Health check endpoint
http://localhost:8080/metrics        # Prometheus metrics
```

**Desktop client** (after ROADMAP Phase 2):
```bash
go run ./cmd/desktop -server ws://localhost:8080/ws
```

**Web WASM client** (after ROADMAP Phase 3):
```bash
GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web
# Serve via the Go server at /play, or use any static HTTP server:
# python3 -m http.server 8080 --directory client/wasm/
```

**Mobile client** (after ROADMAP Phase 4):
```bash
ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile
ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile
```

## Game Rules

### Objective
Investigators must work cooperatively to gather clues and cast protective wards before the doom counter reaches 12.

### Turn Structure
Each player gets 2 actions per turn:
- **Move**: Travel between adjacent locations only
- **Gather**: Roll 2 dice to potentially gain Health and Sanity
- **Investigate**: Roll 3 dice, need 2+ successes to gain a Clue
- **Cast Ward**: Costs 1 Sanity, roll 3 dice, need 3 successes to reduce Doom by 2

### Dice Mechanics
- **Success** (✓): Counts toward action success
- **Blank** (○): No effect
- **Tentacle** (🐙): Increases Doom counter by 1

### Win/Lose Conditions
- **Win**: Collectively gather **4 clues per investigator** before doom reaches 12 (4 clues for 1 player, 8 for 2, 12 for 3, 16 for 4, 20 for 5, 24 for 6)
- **Lose**: Doom counter reaches 12

### Connection Behaviour
- The client reconnects automatically every 5 seconds on disconnection.
- **Note**: In the current version, a disconnected player cannot reclaim their investigator. Reconnecting after a drop creates a new player slot. Full session-persistence with reconnection tokens is planned for a future release.

## Technical Implementation

### Go Server Architecture
- **Interface-based Design**: Uses `net.Conn`, `net.Listener`, and `net.Addr` interfaces
- **Concurrent Connection Handling**: Goroutines with channel-based communication
- **State Management**: Centralized game state with mutex protection
- **Error Handling**: Explicit Go-style error checking and propagation

### Ebitengine Client Features (Planned — ROADMAP Phases 1–5)
- **Sprite/Layer Rendering**: Board, tokens, UI overlays, and animations via Ebitengine draw layers
- **Platform Input Handling**: Keyboard/mouse (desktop), touch (mobile), pointer events (WASM)
- **Multi-Resolution Support**: Logical 1280×720 resolution scaled to any display; safe-area insets on mobile
- **Shader Effects**: Kage shaders for fog-of-war, doom vignette, and interactive highlights
- **WASM Compatibility**: Same Go codebase compiled to WebAssembly for browser play
- **WebSocket Connection**: Automatic reconnection with 5-second retry (same protocol as legacy client)

### Legacy JavaScript Client (Current — to be replaced)
- **WebSocket Connection**: Automatic reconnection with exponential backoff
- **Canvas Rendering**: 800x600px game board with location visualization
- **Real-time Updates**: Live game state synchronization
- **Responsive UI**: Modern web interface with visual feedback

### JSON Message Protocol
```json
// Player Action
{"type": "playerAction", "playerId": "player1", "action": "investigate", "target": "University"}

// Game State Update (full snapshot)
{"type": "gameState", "data": {"currentPlayer": "player2", "doom": 5, "players": {...}}}

// Action Event (lightweight delta, emitted before gameState on every action)
{"type": "gameUpdate", "playerId": "player1", "event": "investigate", "result": "fail", "doomDelta": 1, "resourceDelta": {"health": 0, "sanity": 0, "clues": 0}, "timestamp": "..."}

// Dice Result
{"type": "diceResult", "playerId": "player1", "results": ["success", "blank", "tentacle"]}

// Connection Status
{"type": "connectionStatus", "playerId": "player1", "status": "connected"}
```

## Development

### Project Structure
```
bostonfear/
├── cmd/
│   ├── server/             # Go WebSocket server entry point + game logic
│   │   ├── main.go
│   │   ├── game_server.go
│   │   ├── types.go
│   │   ├── constants.go
│   │   ├── utils.go
│   │   ├── connection_wrapper.go
│   │   ├── error_recovery.go
│   │   └── *_test.go
│   ├── desktop/            # (Planned — Phase 2) Desktop entrypoint
│   │   └── main.go
│   ├── web/                # (Planned — Phase 3) WASM entrypoint
│   │   └── main.go
│   └── mobile/             # (Planned — Phase 4) Mobile entrypoint
│       └── mobile.go
├── client/
│   ├── ebiten/             # (Planned — Phase 1) Ebitengine client package
│   │   ├── game.go         #   ebiten.Game implementation
│   │   ├── net.go          #   WebSocket client
│   │   ├── state.go        #   Local state mirror
│   │   ├── input.go        #   Input handling
│   │   └── render/         # (Planned — Phase 5) Rendering subsystem
│   │       ├── atlas.go
│   │       ├── layers.go
│   │       └── shaders/
│   ├── wasm/               # (Planned — Phase 3) WASM host files
│   │   └── index.html
│   ├── index.html          # Legacy HTML game interface (to be replaced)
│   ├── game.js             # Legacy JavaScript game client (to be replaced)
│   └── dashboard.html      # Performance monitoring dashboard
├── go.mod                  # Go module dependencies
├── go.sum
├── ROADMAP.md              # Phased migration plan (Ebitengine + AH3e compliance)
├── PLAN.md                 # Implementation plan for current gaps + migration
├── GAPS.md                 # Known implementation gaps with status
├── RULES.md                # AH3e rules engine specification + compliance table
└── README.md               # This file
```

### Dependencies
- **Server**: Go 1.24+ with `github.com/gorilla/websocket`
- **Ebitengine Client** (planned): `github.com/hajimehoshi/ebiten/v2` (v2.7+)
- **Mobile Build** (planned): `ebitenmobile` CLI, `gomobile`, Android SDK (API 29+), Xcode 15+
- **Legacy Client** (current): Modern web browser with HTML5 Canvas and WebSocket support

### Testing Multi-player
1. Start the server
2. Open multiple browser tabs/windows to `http://localhost:8080`
3. Each tab represents a different player
4. Game starts automatically when the first player connects; additional players may join at any time

## Game Flow Example

1. **Player 1** moves from Downtown to University (Location System validates adjacency)
2. **Player 1** investigates (Action System calls Dice Resolution)
3. **Dice Results**: Success, Blank, Tentacle (need 2 successes)
4. **Investigation fails** (Resource Tracking - no clue gained)
5. **Tentacle result** increments global doom counter (Doom Counter system)
6. **Turn advances** to Player 2
7. **All clients** receive updated game state within 500ms

## Performance Standards
- Maintains stable operation with 6 concurrent players
- Supports continuous gameplay for 15+ minutes
- Sub-500ms state synchronization across all clients
- Automatic handling of connection drops with 30-second timeout
- Sub-100ms response times for health checks
- Real-time performance monitoring with comprehensive metrics

## Monitoring and Observability

### Performance Dashboard
Access the real-time performance dashboard at `http://localhost:8080/dashboard` to monitor:
- Server uptime and connection analytics
- Memory usage and garbage collection metrics
- Player session tracking and reconnection rates
- Game state health and doom level progression
- Error rates and system alerts

### Prometheus Integration
Export metrics for monitoring tools at `http://localhost:8080/metrics`:
```bash
# Example metrics queries
curl http://localhost:8080/metrics | grep arkham_horror_active_connections
curl http://localhost:8080/metrics | grep arkham_horror_memory_usage_percent
curl http://localhost:8080/metrics | grep arkham_horror_game_doom_level
```

### Health Monitoring
Comprehensive health checks available at `http://localhost:8080/health`:
```json
{
  "status": "healthy",
  "timestamp": 1749441525,
  "performanceMetrics": {
    "uptime": 39368523587,
    "activeConnections": 0,
    "responseTimeMs": 0.00024,
    "errorRate": 0
  },
  "connectionAnalytics": {
    "totalPlayers": 0,
    "activePlayers": 0,
    "reconnectionRate": 0
  }
}
```

## Troubleshooting

### Connection Issues
- Ensure server is running on port 8080
- Check firewall settings
- Verify WebSocket support in browser or Ebitengine client connectivity

### Game State Sync Issues
- Refresh browser to re-establish connection (legacy client)
- Restart desktop client to reconnect (Ebitengine client)
- Check browser console or client logs for WebSocket errors
- Verify all players are using same server instance

### Performance Issues
- Close unnecessary browser tabs
- Ensure stable internet connection
- Check server resources if hosting remotely
