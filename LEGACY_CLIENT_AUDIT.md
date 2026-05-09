# Legacy Client Code Audit

**Date**: May 9, 2026  
**Status**: Complete mapping of all legacy client references  
**Scope**: HTML/JS browser client in `client/` with game.js, index.html, dashboard.html serving via Go server

---

## 1. Legacy Client Files

### Primary Legacy Client Files
Location: `client/` (root directory)

| File | Purpose | Lines | Notes |
|------|---------|-------|-------|
| `client/index.html` | Main legacy HTML game UI (800×600 Canvas) | 506+ | HTML5 Canvas based interface, loads game.js on line 506 |
| `client/game.js` | Legacy JS client logic (WebSocket, game state, rendering) | n/a | Handles connection, state updates, canvas rendering; currently active in production |
| `client/dashboard.html` | Performance monitoring dashboard | n/a | Served at `/dashboard` route for real-time metrics |

### Legacy Client Test/Smoke Files
| File | Purpose | Type |
|------|---------|------|
| `client/connection_quality_smoke_test.js` | Tests `handleConnectionQuality()` method in game.js | Node.js executable smoke test |
| `client/responsive_canvas_smoke_test.js` | Tests `resizeCanvas()` method in game.js | Node.js executable smoke test |

### New Ebiten/WASM Client Files (Not Legacy)
These should remain and are the replacement target:
| File/Dir | Purpose |
|----------|---------|
| `client/ebiten/` | Go Ebiten desktop/WASM client source code |
| `client/wasm/` | WASM build output directory; contains compiled game.wasm |
| `client/wasm/index.html` | WASM launcher HTML |

---

## 2. Go Server Code That Serves/References Legacy Client

### A. Command-Line Interface & Configuration

**File**: [cmd/server.go](cmd/server.go)

- **Flag `--client-dir`** (line 35): "Path to browser client assets"
  - Default: `"./client"`
  - Bound to viper config: `server.client-dir`
  - Used in `runServer()` function (line 90-102)

- **Config loading** (line 90-94):
  ```go
  clientDir := strings.TrimSpace(viper.GetString("server.client-dir"))
  if clientDir == "" {
    clientDir = "./client"
  }
  ```

- **Static file handler setup** (line 102):
  ```go
  Static: http.FileServer(http.Dir(clientDir + "/")),
  ```

### B. HTTP Route Registration

**File**: [transport/ws/server.go](transport/ws/server.go)

- **RouteHandlers struct** (lines 8-15): Defines handlers including `Static http.Handler`
- **Route setup** (lines 22-26):
  - Line 22-26: Mux handles `/ws`, `/health`, `/metrics`, `/dashboard`, and `/`
  - Line 26: `mux.Handle("/", handlers.Static)` — serves all file system requests

- **Log output** (line 30): 
  ```go
  log.Printf("Client available at: http://localhost%s", listener.Addr().String())
  ```

### C. Monitoring/Dashboard Handler

**File**: [monitoring/handlers.go](monitoring/handlers.go)

- **DashboardHandler function** (lines 92-99):
  ```go
  func DashboardHandler(clientDir string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      w.Header().Set("Access-Control-Allow-Origin", "*")
      w.Header().Set("Access-Control-Allow-Methods", "GET")
      w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
      http.ServeFile(w, r, clientDir+"/dashboard.html")
    })
  }
  ```
  - Serves `{clientDir}/dashboard.html` (typically `client/dashboard.html`)
  - Registered at route `/dashboard` in transport/ws/server.go line 25

- **Doc references** (monitoring/doc.go lines 8, 29):
  - Line 8: "GET /dashboard: Interactive HTML5 dashboard..."
  - Line 29: "DashboardHandler serves an HTML5 single-page app..."

### D. Server Initialization Path

**File**: [cmd/server.go](cmd/server.go) — `runServer()` function

Flow:
1. Line 55-65: Load game module (arkhamhorror, eldersign, eldritchhorror, finalhour)
2. Line 68: Create game engine
3. Line 90-102: Set up HTTP handlers with `RouteHandlers` struct:
   - `WebSocket`: WS handler
   - `Health`: health check
   - `Metrics`: Prometheus metrics
   - `Dashboard`: reads from `clientDir` parameter
   - `Static`: reads from `clientDir` parameter ← **LEGACY CLIENT SERVED HERE**
4. Line 104: Call `transportws.SetupServer(listener, handlers)`

---

## 3. Routes, Handlers & Config Options

### HTTP Routes

| Route | Handler | Source | Serves |
|-------|---------|--------|--------|
| `GET /` | `handlers.Static` (http.FileServer) | [transport/ws/server.go:26](transport/ws/server.go#L26) | Legacy client: `client/index.html`, `client/game.js`, etc. |
| `GET /dashboard` | `monitoring.DashboardHandler()` | [cmd/server.go:101](cmd/server.go#L101) | `client/dashboard.html` |
| `GET /ws` | WebSocket handler | [transport/ws/server.go:22](transport/ws/server.go#L22) | WebSocket upgrade |
| `GET /health` | Health check | [transport/ws/server.go:23](transport/ws/server.go#L23) | JSON health status |
| `GET /metrics` | Prometheus metrics | [transport/ws/server.go:24](transport/ws/server.go#L24) | Metrics data |
| `GET *` (root path) | `handlers.Static` | [transport/ws/server.go:26](transport/ws/server.go#L26) | **Serves legacy client files** |

### Configuration Options

**TOML Config File** ([config.toml](config.toml)):
```toml
[server]
# No explicit legacy client settings; only generic client-dir option
```

**CLI Flags** ([cmd/server.go](cmd/server.go)):
- `--client-dir` (line 35): Path to browser client assets (default: `./client`)

**Environment Variables**:
- `BOSTONFEAR_GAME`: Game module selection (not legacy client specific)
- No dedicated env vars for legacy client paths

**Viper Config Keys**:
- `server.client-dir` (bound to `--client-dir` flag): Default `./client`

---

## 4. cmd/ Files with Legacy Client Serving Logic

### [cmd/server.go](cmd/server.go) — Primary Legacy Serving Entry Point

Lines 35, 90-102:
```go
// Line 35: CLI flag definition
cmd.Flags().String("client-dir", "./client", "Path to browser client assets")

// Lines 90-102: Handler setup with static file serving
clientDir := strings.TrimSpace(viper.GetString("server.client-dir"))
if clientDir == "" {
  clientDir = "./client"
}

handlers := transportws.RouteHandlers{
  WebSocket: transportws.NewWebSocketHandler(gameEngine),
  Health:    monitoring.HealthHandler(gameEngine),
  Metrics:   monitoring.MetricsHandler(gameEngine),
  Dashboard: monitoring.DashboardHandler(clientDir),      // ← dashboard.html
  Static:    http.FileServer(http.Dir(clientDir + "/")), // ← index.html, game.js
}
```

### [cmd/root.go](cmd/root.go) — Root Command

Registers the server command and other subcommands. No direct legacy client logic.

### [cmd/desktop.go](cmd/desktop.go) — Desktop Client (NOT Legacy)

- Uses Ebiten client (`client/ebiten/app`)
- Not serving legacy files; does not reference `game.js`, `index.html`, or `dashboard.html`

### [cmd/web_wasm.go](cmd/web_wasm.go) — WASM Client (NOT Legacy)

- Uses Ebiten WASM client (`client/ebiten/app`)
- Not serving legacy files

### [cmd/web_nowasm.go](cmd/web_nowasm.go) — WASM Stub (NOT Legacy)

- Placeholder for non-WASM builds
- Not serving legacy files

### [cmd/server/main.go](cmd/server/main.go) — Legacy Alternate Entry Point

- **Note**: Modern entry point is `cmd/server.go` + `main.go`
- `cmd/server/main.go` exists but appears to be unused (cmd/server.go is integrated into Cobra CLI)

### [cmd/mobile/binding.go](cmd/mobile/binding.go) — Mobile Client (NOT Legacy)

- Ebitengine mobile binding for Android/iOS
- Not serving legacy files

---

## 5. Tests Mentioning/Using Legacy Client

### JavaScript Smoke Tests (require Node.js)

**[client/connection_quality_smoke_test.js](client/connection_quality_smoke_test.js)**
- Extracts `handleConnectionQuality()` method from `game.js`
- Tests the extracted method with mock client object
- Used in CI to validate game.js syntax
- **Makefile target**: Could be added as `test-quality` (not currently in Makefile)

**[client/responsive_canvas_smoke_test.js](client/responsive_canvas_smoke_test.js)**
- Extracts `resizeCanvas()` method from `game.js`
- Tests canvas scaling across multiple viewport profiles
- Used in CI to validate game.js syntax
- **Makefile target**: Could be added as `test-responsive` (not currently in Makefile)

### Go Tests

**[Makefile](Makefile) — test-browser target** (line 18):
```make
test-browser:
	node --check client/game.js
```
- Validates JavaScript syntax in `game.js`
- Currently runs simple syntax check; smoke tests referenced above could be integrated

**No Go tests directly reference legacy client files**:
- `cmd/desktop/main_test.go` — tests Ebiten app creation (not legacy)
- `cmd/server.go` — has no test file; routing tests would need to be added
- `transport/ws/` — has connection tests but no legacy client-specific tests

### Coverage Gaps

**Missing tests for**:
- Static file serving via `/` route
- `/dashboard` handler response
- `--client-dir` flag parsing
- Config loading for `server.client-dir`

---

## 6. Documentation Mentioning Legacy Client

### Primary Documentation

**[README.md](README.md)** — Multiple sections document legacy client

| Section | Line Range | Content |
|---------|-----------|---------|
| "Build Targets" table | 51-52 | `client/index.html` marked as "current" with status "Active (migration in progress)" |
| Quick Setup section | 76+ | Lists URLs: `http://localhost:8080` (legacy), `/dashboard`, `/health`, `/metrics` |
| Game Rules section | 100-130 | Rules implementation applies to all clients |
| Technical Implementation | 187+ | Documents HTML/JS client: "in migration toward the Ebitengine client" |
| ASCII Project Structure | 259-262 | Shows `client/index.html`, `client/game.js`, `client/dashboard.html` |
| Performance Dashboard section | 326-330+ | Documents `/dashboard` route and real-time metrics |

### PLAN.md

| Reference | Line(s) | Context |
|-----------|---------|---------|
| Client architecture | 6 | "3. HTML Canvas client in [client/index.html](client/index.html) and [client/game.js](client/game.js)." |
| Canvas resize | 37 | Dependencies: "Canvas resize behavior in [client/game.js](client/game.js)." |
| Event stream | 56 | "Event stream from `gameUpdate`, `diceResult`, and `gameState` in [client/game.js](client/game.js)" |
| Session semantics | 139 | "Reconnect semantics in [client/ebiten/state.go](client/ebiten/state.go) and [client/game.js](client/game.js)." |

### docs/PERFORMANCE_MONITORING_COMPLETION.md

- Line 40: References `handleDashboard()` method (legacy handler implementation)

### .github/copilot-instructions.md

- Line 56: Original task spec calls for creating `/client/index.html` and `/client/game.js`

---

## 7. Environment Variables, Config Options & Feature Flags

### Environment Variables

**Legacy Client Specific**: None found

**Related Server Config**:
- `BOSTONFEAR_GAME` — Selects game engine module (arkhamhorror, eldersign, etc.); not client-specific
- No environment variable for legacy client path or feature toggle

### Config File Options

**[config.toml](config.toml)** — No legacy client-specific settings

```toml
[server]
# Generic client directory setting, used by all clients served via /
# and /dashboard handler
listen = ""
host = ""
port = 8080

[desktop]
# WebSocket server URL for Ebiten desktop client (not legacy)
server = "ws://localhost:8080/ws"

[web]
# WebSocket server URL for WASM client (not legacy)
server = ""

[network]
allowed_origins = []
```

### CLI Flags

**[cmd/server.go](cmd/server.go)** — Line 35

```go
cmd.Flags().String("client-dir", "./client", "Path to browser client assets")
```

- Bound to: `server.client-dir` in viper
- Affects: `DashboardHandler()` and static file serving at `/`

### Feature Flags or Toggles

**Legacy Client**: No feature flags to disable or enable legacy client serving

**Implications**:
- Legacy client is always served if files exist in `client/` directory
- No graceful degradation path (e.g., "disable legacy client at server startup")
- Removal requires:
  1. Deleting legacy files
  2. Removing static file serving handler
  3. Removing dashboard serving logic
  4. Updating documentation

---

## 8. Complete Reference Map

### Files to Remove (Legacy)
```
client/index.html
client/game.js
client/dashboard.html
client/connection_quality_smoke_test.js (optional; test artifact)
client/responsive_canvas_smoke_test.js  (optional; test artifact)
```

### Go Code to Modify

| File | Lines | Change |
|------|-------|--------|
| [cmd/server.go](cmd/server.go) | 35 | Remove `--client-dir` flag |
| [cmd/server.go](cmd/server.go) | 40 | Remove viper binding for `server.client-dir` |
| [cmd/server.go](cmd/server.go) | 90-102 | Remove clientDir resolution; remove `Dashboard` and `Static` from `RouteHandlers` |
| [transport/ws/server.go](transport/ws/server.go) | 8-15 | Remove `Dashboard http.Handler` and `Static http.Handler` from `RouteHandlers` struct |
| [transport/ws/server.go](transport/ws/server.go) | 25-26 | Remove `/dashboard` and `/` route handlers |
| [monitoring/handlers.go](monitoring/handlers.go) | 92-99 | Delete `DashboardHandler()` function |
| [monitoring/doc.go](monitoring/doc.go) | 8, 29 | Update documentation (remove references to `/dashboard`) |
| [Makefile](Makefile) | 18 | Remove `test-browser: node --check client/game.js` target |

### Documentation to Update

| File | Content |
|------|---------|
| [README.md](README.md) | Remove/update "Build Targets" table entry for HTML/JS client; update "Quick Setup" section to remove `/dashboard`, `/` URLs; remove "HTML/JS Browser Client" technical section |
| [PLAN.md](PLAN.md) | Update/remove references to `client/game.js` and `client/index.html` |
| [config.toml](config.toml) | Remove or mark obsolete: no changes if only serving Ebiten clients |

### Tests to Update

No Go tests directly test legacy client serving (coverage gap). After removal:
- Optional: Add tests for remaining routes (`/ws`, `/health`, `/metrics`)
- Remove: `test-browser` Makefile target
- Consider: Delete smoke test files or archive them

---

## 9. Impact Analysis

### What Will Break After Removal

1. **Clients**: HTTP client at `http://localhost:8080` will return 404
2. **Monitoring**: `/dashboard` route removed; Prometheus metrics at `/metrics` remain
3. **Build**: No impact (no legacy files compiled)
4. **Tests**: `test-browser` Makefile target will fail (intentional; remove it)
5. **Config**: No impact; `server.client-dir` unused after removal

### What Will Remain

- Ebiten desktop client (`cmd/desktop`) → runs via `go run ./cmd/desktop -server ws://localhost:8080/ws`
- WASM client (`cmd/web`) → compiled to `client/wasm/game.wasm`, served by external HTTP server or by adding WASM-specific route
- Server routes: `/ws` (WebSocket), `/health`, `/metrics`
- Game engine: All rules and mechanics unchanged

### Recommended Post-Removal Additions

1. **WASM serving**: Add explicit `/play` or `/game.html` route to serve WASM client from Go server
2. **Redirect**: Consider HTTP 301 redirect from `/` to `/play` until static site is removed
3. **Documentation**: Update README to clearly mark desktop/WASM as the official clients

---

## 10. Removal Checklist

- [ ] Delete `client/index.html`
- [ ] Delete `client/game.js`
- [ ] Delete `client/dashboard.html`
- [ ] Delete or archive `client/connection_quality_smoke_test.js`
- [ ] Delete or archive `client/responsive_canvas_smoke_test.js`
- [ ] Remove `--client-dir` flag from `cmd/server.go` (line 35)
- [ ] Remove viper binding in `cmd/server.go` (line 40)
- [ ] Update `runServer()` in `cmd/server.go` (lines 90-102)
- [ ] Remove `Dashboard` from `RouteHandlers` struct in `transport/ws/server.go`
- [ ] Remove `Static` from `RouteHandlers` struct in `transport/ws/server.go`
- [ ] Remove `/dashboard` route handler registration
- [ ] Remove `/` route handler registration
- [ ] Delete `DashboardHandler()` function from `monitoring/handlers.go`
- [ ] Update `monitoring/doc.go` documentation
- [ ] Remove `test-browser` Makefile target
- [ ] Update README.md (Build Targets table, Quick Setup, Technical Implementation)
- [ ] Update PLAN.md (remove game.js and index.html references)
- [ ] Create `/play` or WASM-serving route if continuing to serve from Go
- [ ] Run tests: `go test -race ./...`
- [ ] Verify: `go run . server` starts without errors
- [ ] Verify: `curl http://localhost:8080/` returns 404 (expected)
- [ ] Verify: Ebiten desktop/WASM clients still connect properly

---

## Summary Table

| Category | Count | Items |
|----------|-------|-------|
| **Legacy Client Files** | 3 | index.html, game.js, dashboard.html |
| **Legacy Test Files** | 2 | connection_quality_smoke_test.js, responsive_canvas_smoke_test.js |
| **Go Files to Modify** | 4 | cmd/server.go, transport/ws/server.go, monitoring/handlers.go, monitoring/doc.go |
| **Documentation Files** | 3 | README.md, PLAN.md, Makefile |
| **Routes to Remove** | 2 | GET `/`, GET `/dashboard` |
| **Config Keys to Remove** | 1 | server.client-dir (unused) |
| **CLI Flags to Remove** | 1 | --client-dir |
| **Total References** | ~40 | Across all categories |
