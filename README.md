# Arkham Horror - Multiplayer Game

> **⚠️ Intellectual Property Notice**
> BostonFear is a **rules-only game engine** designed to execute the mechanics of the
> Arkham Horror series of games. This repository contains **no copyrighted content**
> produced by Fantasy Flight Games. No card text, scenario narratives, investigator
> stories, artwork, encounter text, or any other proprietary material owned by
> Fantasy Flight Games (an Asmodee brand) is, or will ever be, reproduced here.
> *Arkham Horror* is a trademark of Fantasy Flight Games. This project is an
> independent, fan-made rules engine and is not affiliated with or endorsed by
> Fantasy Flight Games or Asmodee.

A multiplayer implementation of Arkham Horror featuring investigators managing resources while exploring locations and facing supernatural threats. Built with a Go WebSocket server and an HTML/JS canvas client with 1-6 concurrent players. Players can join a game already in progress. A Go/Ebitengine client supporting desktop, web (WASM), and mobile platforms is implemented and compilable — see `ROADMAP.md` for ongoing improvements.

> **Active migration:** The client is being migrated from HTML/JS canvas to
> Go/Ebitengine. Desktop and WASM builds compile successfully (alpha — placeholder
> sprites). See `ROADMAP.md` for the phased plan. The WebSocket server and its
> protocol remain stable.

## Features

### Core Game Mechanics
1. **Location System**: 4 interconnected neighborhoods (Downtown, University, Rivertown, Northside) with movement restrictions
2. **Resource Tracking**: Health (1-10), Sanity (1-10), and Clues (0-5) with gain/loss mechanics
3. **Action System**: 2 actions per turn from Move, Gather Resources, Investigate, Cast Ward
4. **Doom Counter**: Global doom tracker (0-12) that increments for each Tentacle result rolled (unconditional — not limited to failed rolls)
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
| **Desktop** (Linux, macOS, Windows) | `cmd/desktop/main.go` | `go build ./cmd/desktop` | Active (alpha — placeholder sprites) |
| **Web (WASM)** | `cmd/web/main.go` | `GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web` | Active (alpha — placeholder sprites) |
| **Mobile** (iOS 16+, Android 10+) | `cmd/mobile/binding.go` | `ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile` | Alpha (binding scaffolding; not verified on device) |
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

**Desktop client** (alpha — builds and runs; placeholder sprites):
```bash
go run ./cmd/desktop -server ws://localhost:8080/ws
```

**Web WASM client** (alpha — builds successfully; placeholder sprites):
```bash
GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web
# Serve via the Go server at /play, or use any static HTTP server:
# python3 -m http.server 8080 --directory client/wasm/
```

**Mobile client** (alpha — binding scaffolding; not verified on device):
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
- The client retries indefinitely using exponential backoff (5 s initial delay, doubling each attempt, 30 s cap). Example: first retry after 5 s, second after 10 s, third after 20 s, all subsequent retries after 30 s. There is no upper limit on attempts.
- The server applies a **30-second inactivity timeout**: if no message arrives from a connected player within 30 seconds, the doom counter is incremented and the connection is closed. This is an idle/inactivity deadline, not a reconnection window.
- **Session Persistence**: The JS legacy client reclaims its player slot automatically using a server-issued reconnect token (stored in `localStorage`). The Ebitengine desktop/WASM client also supports token-based slot reclaim — the token received in a `connectionStatus` message is stored and re-sent as a `?token=` query parameter on the next dial attempt. All clients now support full session persistence on reconnect.

## Technical Implementation

### Go Server Architecture
- **Interface-based Design**: Uses `net.Conn`, `net.Listener`, and `net.Addr` interfaces
- **Module-based Runtime Selection**: Server startup now resolves a game module registry (`arkhamhorror` default) via `BOSTONFEAR_GAME` to support multiple Fantasy Flight-style engines
- **Concurrent Connection Handling**: Goroutines with channel-based communication
- **State Management**: Centralized game state with mutex protection
- **Package Separation**: `serverengine` owns rules/state and transport-neutral session orchestration, `transport/ws` owns HTTP/WebSocket upgrade and route registration, and `monitoring` owns health/metrics/dashboard handlers
- **Error Handling**: Explicit Go-style error checking and propagation
- **WebSocket Origin Validation**: Configurable `allowedOrigins` list (empty = accept any origin for local dev; set to specific hosts for production)

#### Configuring Allowed Origins (Production)
By default the server accepts WebSocket upgrades from any origin, which is safe
for local development. For production deployments, restrict upgrades to your
specific domain(s):

```go
// In cmd/server/main.go, after module.NewEngine():
gameEngine.SetAllowedOrigins([]string{
    "mygame.example.com",   // production domain
    "localhost:8080",        // keep for local testing
})
```

Requests from origins not in the list receive HTTP 403 Forbidden.

#### Selecting Game Module at Startup
Use `BOSTONFEAR_GAME` to select which rules engine module the server loads:

```bash
# Default (if unset): arkhamhorror
BOSTONFEAR_GAME=arkhamhorror go run ./cmd/server

# Placeholder modules (registered for scaffolding)
BOSTONFEAR_GAME=eldersign go run ./cmd/server
BOSTONFEAR_GAME=eldritchhorror go run ./cmd/server
BOSTONFEAR_GAME=finalhour go run ./cmd/server
```

`eldersign`, `eldritchhorror`, and `finalhour` are currently scaffolding modules and intentionally return a not-implemented runtime error on startup.

### Ebitengine Client Features (Active — alpha; placeholder sprites)
- **Sprite/Layer Rendering**: Board, tokens, UI overlays, and animations via Ebitengine draw layers
- **Platform Input Handling**: Keyboard/mouse (desktop), touch (mobile), pointer events (WASM)
- **Multi-Resolution Support**: Logical 1280×720 resolution scaled to any display; safe-area insets on mobile
- **Shader Effects**: Kage shaders for fog-of-war, doom vignette, and interactive highlights
- **WASM Compatibility**: Same Go codebase compiled to WebAssembly for browser play
- **WebSocket Connection**: Automatic reconnection with exponential backoff (5 s initial, doubles per attempt, 30 s cap)

### Legacy JavaScript Client (Deprecated — being replaced by Ebitengine client)
> The HTML5 Canvas / JavaScript client (`client/index.html`, `client/game.js`) is
> being replaced by the Ebitengine client described in `CLIENT_SPEC.md`. See that
> document for the full UI/UX requirements of the new client.

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
│   ├── server/             # Go WebSocket server entry point and dependency wiring
│   │   ├── main.go
│   ├── desktop/            # Desktop entrypoint (Ebitengine, alpha)
│   │   └── main.go
│   ├── web/                # WASM entrypoint (Ebitengine, alpha)
│   │   └── main.go
│   └── mobile/             # Mobile entrypoint (Ebitengine, alpha binding scaffolding)
│       └── binding.go
├── monitoring/             # HTTP health, metrics, and dashboard handlers
│   └── handlers.go
├── monitoringdata/         # Shared monitoring DTOs used by serverengine and monitoring
│   └── types.go
├── protocol/               # Shared Go WebSocket wire schema used by serverengine and Go clients
│   └── protocol.go
├── transport/
│   └── ws/                 # HTTP/WebSocket upgrade + route registration over net.Listener
│       └── server.go
├── serverengine/           # Importable game engine: rules, state, transport-neutral session orchestration
│   ├── actions.go
│   ├── connection.go
│   ├── connection_quality.go
│   ├── game_server.go
│   ├── game_types.go
│   ├── metrics.go
│   ├── health.go
│   └── *_test.go
├── client/
│   ├── ebiten/             # Ebitengine client package (alpha)
│   │   ├── game.go         #   ebiten.Game implementation
│   │   ├── net.go          #   WebSocket client
│   │   ├── state.go        #   Local state mirror
│   │   ├── input.go        #   Input handling
│   │   └── render/         # Rendering subsystem (alpha — placeholder sprites)
│   │       ├── atlas.go
│   │       ├── layers.go
│   │       └── shaders/
│   ├── wasm/               # WASM host files
│   │   └── index.html
│   ├── index.html          # Legacy HTML game interface (to be replaced)
│   ├── game.js             # Legacy JavaScript game client (to be replaced)
│   └── dashboard.html      # Performance monitoring dashboard
├── go.mod                  # Go module dependencies
├── go.sum
├── ROADMAP.md              # Phased migration plan, AH3e compliance, and future priorities
├── RULES.md                # AH3e rules engine specification + compliance table
├── CLIENT_SPEC.md          # Ebitengine client UI/UX requirements
└── README.md               # This file
```

### Dependencies
- **Server**: Go 1.24+ with `github.com/gorilla/websocket`
- **Ebitengine Client** (alpha): `github.com/hajimehoshi/ebiten/v2` (v2.7+)
- **Mobile Build** (alpha): `ebitenmobile` CLI, `gomobile`, Android SDK (API 29+), Xcode 15+
- **Legacy Client** (current): Modern web browser with HTML5 Canvas and WebSocket support

### Shared Go Protocol
- The Go server engine and Go/Ebitengine client compile against the shared wire schema in `protocol/`, which owns the JSON payload structs and protocol enums used on the WebSocket boundary.

### Testing Multi-player
1. Start the server
2. Open multiple browser tabs/windows to `http://localhost:8080`
3. Each tab represents a different player
4. Game starts automatically when the first player connects; additional players may join at any time

### Running Tests

**Standard tests** (no display required — CI-safe):
```bash
go test -race ./...
```

**Ebitengine display tests** (require a local display / virtual framebuffer):
```bash
go test -race -tags=requires_display ./client/ebiten/app/... ./client/ebiten/render/...
```
These tests are guarded by the `requires_display` build tag and are skipped by the
standard `go test ./...` invocation. They verify Ebitengine `App` initialisation,
renderer atlas logic, and nil-safety paths. Run them locally with a real or virtual
display (`Xvfb` on Linux):
```bash
Xvfb :99 -screen 0 1280x720x24 &
DISPLAY=:99 go test -race -tags=requires_display ./client/ebiten/app/... ./client/ebiten/render/...
```

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
  - 30-second idle inactivity timeout per connection (see §Connection Behaviour)
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
