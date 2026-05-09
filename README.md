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

A multiplayer implementation of Arkham Horror featuring investigators managing resources while exploring locations and facing supernatural threats. Built with a Go WebSocket server and Go/Ebitengine clients supporting desktop, web (WASM), and mobile platforms with 1-6 concurrent players. Players can join a game already in progress. See `ROADMAP.md` for development roadmap and upcoming improvements.

> **Client:** The game client is exclusively implemented in Go/Ebitengine. Desktop
> and WASM builds compile successfully (alpha — placeholder sprites). Mobile
> bindings are scaffolded. The WebSocket server and protocol remain stable.

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

## Quick Setup

### Step 1: Install Dependencies
```bash
cd /workspaces/bostonfear
go mod tidy
```

### Step 2: Start Server
```bash
# New Cobra root CLI
go run . server

# Backward-compatible entrypoint still supported
go run ./cmd/server
```

### Step 3: Access Client

**Desktop client** (alpha — builds and runs; placeholder sprites):
```bash
go run ./cmd/desktop -server ws://localhost:8080/ws
```

**Web WASM client** (alpha — builds successfully; placeholder sprites):
```bash
GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web
# Serve via any static HTTP server; Go server also supports WASM serving:
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
- **Session Persistence**: The Ebitengine desktop/WASM client supports token-based slot reclaim — the token received in a `connectionStatus` message is stored and re-sent as a `?token=` query parameter on the next dial attempt for full session persistence.

## Technical Implementation

### Go Server Architecture
- **Interface-based Design**: Uses `net.Conn`, `net.Listener`, and `net.Addr` interfaces
- **Module-based Runtime Selection**: Server startup now resolves a game module registry (`arkhamhorror` default) via `BOSTONFEAR_GAME` to support multiple Fantasy Flight-style engines
- **Concurrent Connection Handling**: Goroutines with channel-based communication
- **State Management**: Centralized game state with mutex protection
- **Package Separation**: `serverengine/common` owns reusable runtime contracts/primitives, `serverengine/arkhamhorror` owns Arkham rules binding, and game-family roots (`serverengine/eldersign`, `serverengine/eldritchhorror`, `serverengine/finalhour`) are scaffolded for future engines; `transport/ws` owns HTTP/WebSocket upgrade and route registration, and `monitoring` owns health/metrics handlers
- **Error Handling**: Explicit Go-style error checking and propagation
- **WebSocket Origin Validation**: Configurable `allowedOrigins` list (empty = accept any origin for local dev; set to specific hosts for production)

#### CLI + Config (Cobra + Viper)
- Root CLI command: `go run . --help`
- TOML config template: `config.toml` at repository root
- Global config flag: `--config /path/to/config.toml`
- Key mappings:
  - `server.game` -> module selection (fallback: `BOSTONFEAR_GAME`, default `arkhamhorror`)
  - `server.listen` -> TCP listen address (default `:8080`)
  - `network.allowed-origins` -> WebSocket origin allow-list
  - `scenario.default_id` -> default scenario content ID (default `scn.nightglass.harbor-signal`)
  - `desktop.server` -> desktop client WebSocket URL (default `ws://localhost:8080/ws`)
  - `web.server` -> optional WASM client WebSocket URL override

#### Default Scenario Content
The demo content pack defines `scn.nightglass.harbor-signal` as the default scenario.

Fallback behavior (content-loader contract):
1. Use `scenario.default_id` when valid and enabled.
2. Else use `content/scenarios/index.yaml` `defaultScenarioId`.
3. Else use the first enabled scenario sorted by ID.
4. Else fail startup with a content validation error.

Specification and inventory details are documented in `docs/content/BASE_SET_DEFAULT_SCENARIO_SPEC.md`.

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
├── monitoring/             # HTTP health and metrics handlers
│   └── handlers.go
├── monitoringdata/         # Shared monitoring DTOs used by serverengine and monitoring
│   └── types.go
├── protocol/               # Shared Go WebSocket wire schema used by serverengine and Go clients
│   └── protocol.go
├── transport/
│   └── ws/                 # HTTP/WebSocket upgrade + route registration over net.Listener
│       └── server.go
├── serverengine/
│   ├── common/             # Shared runtime contracts and cross-game primitives
│   │   ├── contracts/
│   │   └── runtime/
│   ├── arkhamhorror/       # Arkham game-family module + rules documentation
│   │   ├── module.go
│   │   ├── README.md
│   │   └── RULES.md
│   ├── eldersign/          # Future game-family scaffold (placeholder module)
│   │   └── module.go
│   ├── eldritchhorror/     # Future game-family scaffold (placeholder module)
│   │   └── module.go
│   ├── finalhour/          # Future game-family scaffold (placeholder module)
│   │   └── module.go
│   ├── game_server.go      # Current Arkham runtime implementation (compat phase)
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
│   └── wasm/               # WASM host and compiled output
│       └── index.html
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

### Shared Go Protocol
- The Go server engine and Go/Ebitengine client compile against the shared wire schema in `protocol/`, which owns the JSON payload structs and protocol enums used on the WebSocket boundary.

### Testing Multi-player
1. Build desktop client: `go build ./cmd/desktop`
2. Start the server: `go run . server` (or build it)
3. Launch multiple desktop client instances connecting to `ws://localhost:8080/ws`
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

### WASM Launcher
When running `go run . server`, open `http://localhost:8080/play` to load the WASM host page from `client/wasm/index.html`.

### Monitoring Endpoints
Use these server endpoints for operational visibility:
- Health JSON: `http://localhost:8080/health`
- Prometheus metrics: `http://localhost:8080/metrics`

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

## API Stability and Public Packages

**Version**: v0.0.0 (pre-release; no stability guarantees)

### Recommended Stable Packages (for integration)
Use these packages if building custom tools, alternative transports, or game variants:
- **`protocol`**: Message types, DTOs, Location/ActionType/DiceResult enums
  - Stable wire format for WebSocket interop with non-Go clients
  - Safe to depend on for JSON serialization contracts
- **`serverengine`**: Core GameServer, game mechanics, state validation
  - Safe for: Starting a game, handling connections, integrating new transports
  - Key types: `GameServer`, `Scenario`, interfaces `Broadcaster`, `StateValidator`
  - Avoid direct access to internal fields; use public methods
- **`monitoring`**: Health and metrics HTTP handlers
  - Safe for: Prometheus scraping, health probes, observability integration
- **`transport/ws`**: Gorilla WebSocket adapter, HTTP server setup
  - Safe for: Upgrading WebSocket connections, custom route registration
  - Key interface: `SessionEngine` (minimal 2-method surface)

### Experimental Packages (subject to change)
These packages may be refactored or reorganized before v1:
- **`serverengine/arkhamhorror`**: Game module implementation (rules may change; scaffolding expected as new cards/mechanics are added)
- **`serverengine/common`**: Shared contracts and utilities (organizing principles may shift as modules mature)
- **`client/ebiten`**: Ebitengine client implementation (UI/UX subject to change)
- **`cmd`**: CLI commands and startup logic (config file format and flags may change)

### Unimplemented/Scaffolding Packages
Not yet functional; expect breaking changes or removal:
- **`serverengine/eldersign`**, `eldritchhorror`, `finalhour`: Placeholder game modules
- **`serverengine/common/messaging`**, `session`, `state`: Reserved for future cross-module sharing

### Breaking Changes
As a pre-v1 project, breaking changes may occur without deprecation periods. The JSON wire protocol for clients (in `protocol`) is considered more stable than Go API surface. Subscribe to releases for migration guidance.

## Troubleshooting

### Connection Issues
- Ensure server is running on port 8080
- Check firewall settings
- Verify WebSocket support in browser or Ebitengine client connectivity

### Game State Sync Issues
- Reload the WASM page at `/play` to re-establish the browser session
- Restart desktop client to reconnect (Ebitengine client)
- Check browser console or client logs for WebSocket errors
- Verify all players are using same server instance

### Performance Issues
- Close unnecessary browser tabs
- Ensure stable internet connection
- Check server resources if hosting remotely
