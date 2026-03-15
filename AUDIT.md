# AUDIT — 2026-03-15

## Project Goals

The project is an Arkham Horror-themed cooperative multiplayer web game, targeting intermediate developers learning client-server WebSocket architecture. It promises:

- **5 core game mechanics**: Location System, Resource Tracking, Action System, Doom Counter, Dice Resolution
- **Multiplayer**: 2–4 concurrent players with real-time state sync (<500 ms)
- **Go architecture**: Interface-based design using `net.Conn`, `net.Listener`, `net.Addr`; goroutines/channels for concurrency; explicit Go-style error handling
- **JavaScript client**: HTML5 Canvas (800×600), automatic reconnection every 5 s
- **JSON protocol**: `gameState`, `playerAction`, `gameUpdate`, `connectionStatus`, `diceResult` message types
- **Performance monitoring**: Real-time dashboard (`/dashboard`), Prometheus metrics (`/metrics`), health endpoint (`/health`)
- **Setup in 3 steps**: `go mod tidy`, `cd server && go run main.go`, open browser
- **Connection recovery**: 30-second reconnection timeout with state restoration
- **Performance standards**: Sub-500 ms sync, sub-100 ms health response, stable under 4 players × 15 minutes

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Location System (4 neighborhoods, adjacency) | ✅ Achieved | `cmd/server/constants.go:17–28`, validated in `game_server.go:123–135` |
| Resource Tracking (Health/Sanity/Clues, bounds) | ✅ Achieved | `game_server.go:99–119`, `types.go:17–22` |
| Action System (Move, Gather, Investigate, Cast Ward, 2/turn) | ✅ Achieved | `game_server.go:207–323` |
| Doom Counter (0–12, tentacle increment) | ✅ Achieved | `game_server.go:292–295`, `354–361` |
| Dice Resolution (Success/Blank/Tentacle, thresholds) | ✅ Achieved | `game_server.go:140–164` |
| Mechanic cross-dependencies (dice→doom, resources→ward cost) | ✅ Achieved | `game_server.go:222–289` |
| 2–4 concurrent players | ✅ Achieved | `game_server.go:412–414` |
| Real-time state sync | ✅ Achieved | `game_server.go:1008–1091` |
| Turn order (2 actions/turn, advance) | ✅ Achieved | `game_server.go:297–313`, `326–350` |
| `net.Conn`/`net.Listener`/`net.Addr` interface usage | ✅ Achieved | `connection_wrapper.go`, `utils.go:28`, `main.go:20` |
| Goroutines + channels for concurrency | ✅ Achieved | `game_server.go:88–96`, `broadcastCh`, `actionCh` |
| HTML5 Canvas 800×600 | ✅ Achieved | `client/index.html:295` |
| WebSocket reconnection (client-side) | ✅ Achieved | `client/game.js:101–115` |
| `gameState`, `playerAction`, `connectionStatus`, `diceResult` messages | ✅ Achieved | `game_server.go:447–453`, `316–318`; `game.js:147–165` |
| `gameUpdate` message type | ❌ Missing | Never emitted by server or handled by client |
| Performance dashboard (`/dashboard`) | ✅ Achieved | `game_server.go:619–628` |
| Prometheus metrics (`/metrics`) | ✅ Achieved | `game_server.go:630–718` |
| Health endpoint (`/health`) | ✅ Achieved | `game_server.go:567–617` |
| Setup step 2 ("cd server && go run main.go") | ❌ Broken | Entry point is `cmd/server/main.go`; README says `cd server` |
| Project structure (`server/main.go`) | ❌ Broken | File is at `cmd/server/main.go`, not `server/main.go` |
| Reconnection with session state restoration | ⚠️ Partial | 30-s read timeout exists; reconnect creates new player ID — no state preserved |
| Win condition documented | ⚠️ Partial | README: "sufficient collective clues"; code: `playerCount×4` clues — threshold undisclosed |
| Sub-100 ms health response | ✅ Achieved | `measureHealthCheckResponseTime()` confirms sub-ms timing |
| `calculateErrorRate()` real implementation | ❌ Missing | `game_server.go:1111–1113` always returns `0.0` |
| `AverageLatency`/`BroadcastLatency` in throughput metrics | ❌ Missing | `game_server.go:920–926` — hardcoded `0`, TODO comment at line 919 |

---

## Findings

### CRITICAL

- [ ] **Race condition: write to `gs.gameState` under RLock in `broadcastGameState`** — `cmd/server/game_server.go:1063–1065` — `broadcastGameState` acquires `gs.mutex.RLock()` at line 1046 then assigns `gs.gameState = recoveredState` at line 1065 while still holding the read lock. A concurrent writer in `processAction` (which holds a full `Lock`) would produce a data race. The `go test -race` suite does not exercise this path concurrently, so it passes, but the race is real under load.
  **Remediation:** Upgrade to a write lock before the assignment: replace `gs.mutex.RLock()` / `gs.mutex.RUnlock()` with `gs.mutex.Lock()` / `gs.mutex.Unlock()` in `broadcastGameState`, or restructure recovery to apply the write outside the broadcast path. Validate with: `go test -race -count=5 ./cmd/server/...`.

- [ ] **Deadlock bug in `getGameStatistics`** — `cmd/server/game_server.go:1303–1304` — Line 1304 reads `defer gs.mutex.RLock()` (acquires another read lock on return) instead of `defer gs.mutex.RUnlock()`. Any call to this function will deadlock the server. The function is currently unreferenced (dead code per `go-stats-generator`), so it does not trigger at runtime, but it is exported-adjacent and will deadlock if wired to a handler.
  **Remediation:** Change line 1304 from `defer gs.mutex.RLock()` to `defer gs.mutex.RUnlock()`. Validate with: `go vet ./cmd/server/...` and `go test -race ./cmd/server/...`.

### HIGH

- [ ] **README setup instructions reference non-existent path** — `README.md:39` — Step 2 says `cd server && go run main.go`. The actual entry point is `cmd/server/main.go`. A developer following the README will get a "no Go files" error. The project structure table at line 104 also lists `server/main.go`.
  **Remediation:** Update `README.md` Step 2 to `cd cmd/server && go run .` and correct the project structure table to show `cmd/server/main.go`. Validate by running the corrected command on a clean clone.

- [ ] **`gameUpdate` message type never implemented** — Required by the JSON protocol spec and referenced in `.github/copilot-instructions.md:61`, but no `gameUpdate` message is emitted by the server (`game_server.go`) or handled by the client (`client/game.js`). The spec requires five protocol message types; only four are present.
  **Remediation:** Add a `gameUpdate` broadcast after each action (distinct from the full `gameState` — a diff/event message) and add a `case 'gameUpdate':` handler in `game.js`. Validate by opening browser dev tools and confirming the message appears after a player action.

- [ ] **No session state restoration on reconnect** — `game_server.go:400–401` — Every connection generates a new `player_<UnixNano>` player ID. A disconnected player who reconnects is treated as a new player, receiving a new slot (up to the 4-player cap) or being rejected. The README promises "30-second reconnection timeout" and the ROADMAP acknowledges true session persistence is a future feature (Phase 1.1), but the README presents it as already working. Under this implementation, a disconnected player loses their investigator permanently.
  **Remediation:** Either document the limitation accurately in the README ("disconnected players cannot reclaim their investigator"), or implement a reconnection token: generate a session token on first connect, store it with the player record, and look it up on reconnect to re-attach the existing player state. Validate by disconnecting a client and reconnecting within 30 seconds — the same player ID should be restored.

- [ ] **`handleDashboard` serves from incorrect relative path** — `game_server.go:627` — The handler calls `http.ServeFile(w, r, "./client/dashboard.html")`. When the server is run from `cmd/server/` (the correct directory), this resolves to `cmd/server/client/dashboard.html`, which does not exist. The client files are at `../../client/` relative to `cmd/server/`. The static file handler at `/` correctly uses `../client/`, but the dashboard handler does not.
  **Remediation:** Change line 627 to `http.ServeFile(w, r, "../client/dashboard.html")` to match the relative path used by the static file server. Validate with: `curl -v http://localhost:8080/dashboard` after starting the server from `cmd/server/`.

- [ ] **`ConnectionWrapper` deadline methods are no-ops** — `connection_wrapper.go:56–68` — `SetDeadline`, `SetReadDeadline`, and `SetWriteDeadline` all return `nil` without delegating to the underlying `*websocket.Conn`. `handleConnection` calls `conn.SetReadDeadline(...)` at lines 389 and 461, believing it sets a 30-second timeout on the connection, but the timeout is silently ignored. The actual deadline is set separately on `wsConn` in the same function. This makes the `net.Conn` abstraction misleading.
  **Remediation:** Implement the deadline methods on `ConnectionWrapper` by delegating to `c.ws.SetReadDeadline`, `c.ws.SetWriteDeadline`, and `c.ws.SetDeadline`. Then remove the duplicate direct `wsConn.SetReadDeadline` calls in `handleConnection`. Validate that a connection that sends no messages for 30 seconds is dropped, and doom increments once.

### MEDIUM

- [ ] **`calculateErrorRate()` always returns 0.0** — `game_server.go:1110–1113` — The Prometheus `/metrics` endpoint exports `arkham_horror_error_rate_percent`, and the `/health` endpoint includes an alert threshold check against this rate (`> 5` triggers an alert). The function body is `return 0.0`. This means the error rate metric is permanently false and the alert can never trigger.
  **Remediation:** Track errors in a dedicated atomic counter (e.g., `errorCount int64`) incremented at every `log.Printf("... error ...")` call site. Divide by `totalMessagesRecv` to compute rate. Validate with: `curl http://localhost:8080/metrics | grep error_rate` after inducing deliberate invalid actions.

- [ ] **Win condition threshold undocumented** — `game_server.go:371` — The README states the win condition is "Achieve sufficient collective clues" with no threshold given. The code computes `playerCount * 4` clues required. A 2-player game ends on 8 clues; 4-player on 16. This is not stated in the rules section, making the game unplayable without reading the source.
  **Remediation:** Add a sentence to the README Game Rules section: "**Win**: Collectively gather 4 clues per investigator (8 for 2 players, 12 for 3, 16 for 4) before doom reaches 12." No code change required.

- [ ] **Reconnect delay is fixed, not exponential backoff as claimed** — `client/game.js:10–12, 103–110` — The README says "automatic reconnection attempts every 5 seconds on failure." The code uses a fixed `reconnectDelay = 5000` ms. The `attemptReconnect` comment says "exponential backoff" but the delay never changes. This is a minor gap (fixed 5-s matches the documented behavior) but the comment creates a maintenance risk.
  **Remediation:** Either implement true exponential backoff (`this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000)`) to match the comment, or remove the misleading comment. Validate by watching the browser console during repeated reconnects.

- [ ] **`connectionsPerSecond` can produce `Inf` on startup** — `game_server.go:730–731` — If `uptime.Seconds()` is near zero (e.g., the first health check fires in the same millisecond as server start), the division `float64(gs.totalConnections) / uptime.Seconds()` produces `+Inf` or `NaN`, which breaks JSON serialization of the metrics response.
  **Remediation:** Guard the division: `if uptime.Seconds() > 0 { connectionsPerSecond = float64(gs.totalConnections) / uptime.Seconds() }`. Apply the same pattern to `messagesPerSecond` at line 748. Validate with: `curl http://localhost:8080/health` immediately after server start.

- [ ] **`broadcastConnectionQuality` iterates `gs.gameState.Players` without holding `gs.mutex`** — `game_server.go:1266–1281` — This method is called from `handlePongMessage`, which holds `gs.qualityMutex` but not `gs.mutex`. The iteration over `gs.gameState.Players` (a shared map) races with `handleConnection` and `processAction`, which mutate the map under `gs.mutex`.
  **Remediation:** Add `gs.mutex.RLock()` / `gs.mutex.RUnlock()` around the `for playerID := range gs.gameState.Players` loop at lines 1266–1281. Validate with `go test -race ./cmd/server/...`.

### LOW

- [ ] **`getGameStatistics` is dead code with a latent deadlock** — `game_server.go:1301–1360` — The function is never called (confirmed by `go-stats-generator`: 11 unreferenced functions). Beyond being unused, it contains the `defer gs.mutex.RLock()` deadlock bug (see CRITICAL finding). Leaving dead code with a critical bug in it is a maintenance hazard.
  **Remediation:** Either remove the function entirely, or fix the defer and wire it to the `/health` or `/metrics` endpoint if the game statistics data is wanted. Validate by grepping: `grep -n "getGameStatistics" ./cmd/server/*.go` — should show zero call sites after removal.

- [ ] **`AverageLatency` and `BroadcastLatency` in `MessageThroughputMetrics` are placeholder zeros** — `game_server.go:919–926` — A TODO comment at line 919 acknowledges these are unimplemented. The `/metrics` endpoint does not expose these fields, but they appear in the internal `MessageThroughputMetrics` struct, which may be exposed in future dashboard work.
  **Remediation:** Either track message send/receive timestamps to compute real latency, or remove the fields until they can be properly implemented. Validate with: `go build ./cmd/server/...` after changes.

- [ ] **Magic number `../client/` hardcoded without a named constant** — `utils.go:35` — The static file serving path is an inline string. Combined with the `dashboard.html` path inconsistency (see HIGH finding), this creates two divergent path references that must be kept in sync manually.
  **Remediation:** Define a package-level constant `clientDir = "../client/"` in `constants.go` and use it in both `setupServer` (line 35) and `handleDashboard` (line 627). Validate with: `go build ./cmd/server/...`.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total Lines of Code | 1,220 (non-test) |
| Total Functions + Methods | 57 |
| Average Function Length | 27.1 lines |
| Longest Function (`processAction`) | 155 lines |
| Functions > 50 lines | 10 (17.5%) |
| Average Cyclomatic Complexity | 6.2 |
| High Complexity (> 10) | 4 functions |
| Most Complex (`RecoverGameState`) | cyclomatic 22, overall 31.1 |
| Documentation Coverage | 93.6% |
| Dead Code (Unreferenced Functions) | 11 |
| Duplication / Copy-Paste Drift | Not detected |
| `go vet` warnings | 0 |
| `go test -race` | PASS (5 tests) |
| Packages | 1 (`main` in `cmd/server`) |
| Total Structs | 28 |
| Total Interfaces Defined | 0 (relies on stdlib `net.Conn` etc.) |

*Report generated by go-stats-generator v1.0.0 and manual code review.*
