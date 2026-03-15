# AUDIT — 2026-03-15

## Project Goals

The project is a multiplayer Arkham Horror web game targeting intermediate
developers learning client-server WebSocket architecture. It promises:

1. **1–6 concurrent players** with real-time WebSocket communication (gorilla/websocket).
2. **Five core mechanics**: Location System (4 neighbourhoods, adjacency), Resource
   Tracking (Health/Sanity/Clues), Action System (2 actions/turn), Doom Counter
   (0–12), Dice Resolution (Success/Blank/Tentacle).
3. **Multiplayer features**: turn-order enforcement, join-in-progress, session
   persistence via reconnect tokens.
4. **Go idioms**: `net.Conn`/`net.Listener`/`net.Addr` interfaces, goroutine-based
   concurrency, explicit error handling, interface-based design.
5. **Ebitengine client** (alpha): desktop, WASM, mobile build targets.
6. **Legacy HTML5/JS client** (being replaced): 800×600 canvas, auto-reconnect.
7. **Performance monitoring**: `/health`, `/metrics` (Prometheus), `/dashboard`.
8. **Win/Lose conditions**: 4 clues per investigator collectively before doom
   reaches 12.
9. **Sub-500ms broadcast latency** and stable operation with 6 players over 15 min.

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Location System (4 neighbourhoods + adjacency) | ✅ Achieved | `game_constants.go:38-44`, adjacency map enforced in `performMove` |
| Resource Tracking (Health/Sanity/Clues) | ✅ Achieved | `game_mechanics.go:validateResources`, bounds enforced |
| Action System (2 actions/turn, 4 core actions) | ✅ Achieved | `game_server.go:processAction`, `dispatchAction` |
| Doom Counter (0–12, Tentacle/timeout increment) | ✅ Achieved | `game_mechanics.go:performGather/Investigate/CastWard`, `connection.go:runMessageLoop` |
| Dice Resolution (3-sided, configurable threshold) | ✅ Achieved | `game_mechanics.go:rollDice`, `rollDicePool` |
| 1–6 concurrent players | ✅ Achieved | `game_constants.go:MinPlayers/MaxPlayers`, `registerPlayer` |
| Join game in progress | ✅ Achieved | `connection.go:registerPlayer:103-107` |
| Turn-order enforcement | ✅ Achieved | `validateActionRequest`, `advanceTurn` |
| Session persistence (reconnect tokens) | ✅ Achieved | `connection.go:restorePlayerByToken`, `game.js:9,152` |
| net.Conn / net.Listener / net.Addr interfaces | ✅ Achieved | `connection_wrapper.go`, `game_server.go:connections map[string]net.Conn`, `main.go:net.Listen` |
| Goroutine-based concurrency | ✅ Achieved | `broadcastHandler`, `actionHandler`, `cleanupDisconnectedPlayers` goroutines |
| Ebitengine desktop build | ✅ Achieved | `go build ./cmd/desktop` passes (alpha, placeholder sprites) |
| Ebitengine WASM build | ✅ Achieved | `GOOS=js GOARCH=wasm go build ./cmd/web` passes |
| Legacy JS client (800×600 canvas, reconnect) | ✅ Achieved | `client/index.html:295`, `game.js:109-115` |
| /health endpoint | ✅ Achieved | `health.go:handleHealthCheck` |
| /metrics Prometheus endpoint | ✅ Achieved | `metrics.go:handleMetrics` |
| /dashboard endpoint | ✅ Achieved | `dashboard.go:handleDashboard` |
| Win condition: 4 clues per investigator | ✅ Achieved | `game_mechanics.go:rescaleActDeck`, `checkActAdvance` |
| 30-second inactivity timeout → doom increment | ✅ Achieved | `connection.go:runMessageLoop:148-155` |
| go vet passes | ✅ Achieved | `go vet ./...` → no warnings |
| All tests pass (race detector) | ✅ Achieved | `go test -race ./...` → all PASS |
| /health and /metrics deadlock-free under load | ❌ Missing | Nested `gs.mutex.RLock()` in same goroutine — see Finding H-01 |
| `gs.gameState.Doom` read without lock in alert | ❌ Missing | Bare field read after `RUnlock()` — see Finding H-02 |
| Win threshold rescales when player joins mid-game | ⚠️ Partial | `rescaleActDeck` only called at game start, not on late join — see Finding M-01 |
| `ActionComponent` dead code removed or documented | ⚠️ Partial | Case exists in `dispatchAction` but excluded from `isValidActionType` — see Finding L-01 |
| Ebitengine `app`/`render` tests run in headless CI | ⚠️ Partial | Tests guarded by `requires_display` build tag — see Finding M-02 |

---

## Findings

### HIGH

- [ ] **H-01: Latent deadlock in `/health` and `/metrics` under write-lock pressure** — `cmd/server/health.go:19,30` and `cmd/server/metrics.go:17,28` — Both `handleHealthCheck` and `handleMetrics` acquire `gs.mutex.RLock()` and, while holding it, call `collectPerformanceMetrics()` (`metrics.go:161`), which calls `measureHealthCheckResponseTime()` (`health.go:104`), which calls `gs.mutex.RLock()` again from the same goroutine. `getGameStatistics()` (`health.go:132`) and `getSystemAlerts()` (`health.go:232`) also acquire `gs.mutex.RLock()` while it is already held by the outer call. Go's `sync.RWMutex` is **not reentrant**: if any goroutine calls `Lock()` between the outer `RLock()` and the inner `RLock()`, the inner `RLock()` blocks, while the writer blocks on the outer `RLock()`, causing a deadlock. Under 6 concurrent players performing actions (each holding `gs.mutex.Lock()` in `processAction`), this deadlock is realistically triggered during a health check. No test exercises concurrent writes + health checks, so `go test -race` does not catch this. **Remediation:** Snapshot all state needed by the monitoring helpers under a single `gs.mutex.RLock()` at the top of each HTTP handler, then release the lock before calling the helper functions:
  ```go
  // In handleHealthCheck and handleMetrics:
  gs.mutex.RLock()
  snapDoom    := gs.gameState.Doom
  snapPhase   := gs.gameState.GamePhase
  // ... other fields
  gs.mutex.RUnlock()  // release BEFORE calling helpers
  perfMetrics := gs.collectPerformanceMetrics()  // safe: no mutex needed
  ```
  Remove `gs.mutex.RLock/RUnlock` from `measureHealthCheckResponseTime`, `getGameStatistics`, and the doom-alert section of `getSystemAlerts`, replacing them with pre-snapshotted values passed as arguments. Validate: `go test -race ./cmd/server/... -run TestHandleWebSocket_NewConnection` with concurrent action load.

- [ ] **H-02: Data race — `gs.gameState.Doom` read without lock in `getSystemAlerts`** — `cmd/server/health.go:239,245` — After `gs.mutex.RUnlock()` at line 234, `gs.gameState.Doom` is read bare inside two `fmt.Sprintf` calls (lines 239 and 245). Any concurrent `processAction` holding `gs.mutex.Lock()` and writing `gs.gameState.Doom` races with these reads. The race detector does not catch this because no integration test runs `/health` concurrently with game actions. **Remediation:** Capture `gs.gameState.Doom` inside the existing `gs.mutex.RLock()` block (line 232) and use the captured value in the `Sprintf` calls:
  ```go
  gs.mutex.RLock()
  doom := gs.gameState.Doom
  doomPercent := float64(doom) / 12.0 * 100
  gs.mutex.RUnlock()
  // Use `doom` (not gs.gameState.Doom) in the Sprintf calls below.
  ```
  Validate: `go test -race ./cmd/server/... -run TestHandleHealthCheck` with a concurrent action-generating goroutine.

### MEDIUM

- [ ] **M-01: Win threshold not rescaled when a player joins mid-game** — `cmd/server/connection.go:103-107` — `rescaleActDeck` is called only when `!gs.gameState.GameStarted`. With `MinPlayers=1`, the first player always starts the game and `rescaleActDeck(1)` is called (final threshold: 4 clues). If a second player joins while the game is in progress, the threshold stays at 4 instead of updating to 8 as the README win table requires. The three-player case (4 → 12 at game-start via rescale) is the only player count that accidentally matches the documented 3-player threshold because the old fixed deck matched it. **Remediation:** After the late-join branch (`gs.gameState.GameStarted && gs.gameState.GamePhase == "playing"`, line 104), add a rescale call:
  ```go
  } else if gs.gameState.GameStarted && gs.gameState.GamePhase == "playing" {
      log.Printf("Player %s joined game in progress (turn order position %d)", playerID, len(gs.gameState.TurnOrder))
      if len(gs.gameState.ActDeck) >= 3 {
          gs.rescaleActDeck(len(gs.gameState.Players))
      }
  }
  ```
  Validate: `go test -race ./cmd/server/... -run TestRulesActAgendaProgression_PlayerCountScaling`; add a test that joins a second player mid-game and asserts the Act-1 threshold equals 4 (not still 1, the 1P value).

- [ ] **M-02: Ebitengine `app` and `render` test files excluded from default `go test` run** — `client/ebiten/app/game_test.go:9` and `client/ebiten/render/atlas_test.go:9` — Both files carry `//go:build requires_display`, so `go test ./...` (and any CI without a virtual display and the `-tags=requires_display` flag) silently skips them. The README Quick Setup docs (`go test ./...`) never mention this tag, meaning contributors cannot verify these tests without out-of-band knowledge. **Remediation:** (a) Extract the display-independent assertions from those test files into tag-free functions (e.g., verify struct initialisation, nil-safety, and logic paths that do not call Ebitengine image/draw APIs), or (b) add a `Makefile` target `make test-display` and document it in `README.md §Testing` alongside `go test ./...` so the build tag is discoverable. Validate: `go test -tags=requires_display -race ./client/ebiten/app/... ./client/ebiten/render/...` in a display-capable environment.

### LOW

- [ ] **L-01: `ActionComponent` dead code in `dispatchAction`** — `cmd/server/game_mechanics.go:169-170` — `isValidActionType` explicitly excludes `ActionComponent` (see comment at `game_server.go:207`), making `case ActionComponent:` in `dispatchAction` unreachable. The `performComponent` stub at `game_mechanics.go:345` is likewise dead. Dead cases in a switch add noise and can mislead future developers. **Remediation:** Remove `case ActionComponent:` and `performComponent` from `game_mechanics.go`. Keep `ActionComponent` constant in `game_constants.go` with a comment marking it as reserved for Phase 6, matching `ROADMAP.md`. Validate: `go test -race ./cmd/server/... -run TestProcessAction_ComponentActionRejected`.

- [ ] **L-02: 31 functions reported as unreferenced by `go-stats-generator`** — `go-stats-generator analyze . --skip-tests` maintenance section — 31 functions flagged as potentially unreferenced. While some may be false positives (interface implementations, init-like helpers), unreviewed dead code increases maintenance burden. **Remediation:** Run `go-stats-generator analyze . --skip-tests --format json` and cross-reference the list against exported symbols and interface implementations; delete confirmed dead functions. Validate: re-run `go-stats-generator analyze . --skip-tests` and confirm dead-code count drops.

- [ ] **L-03: Low file cohesion across 11 files** — `go-stats-generator` placement analysis — 11 files have a cohesion score below 0.20. The most affected are `cmd/server/game_types.go` (metrics types mixed with game types), `cmd/server/game_server.go` (constructor + connection + action pipeline), and `client/ebiten/net.go` (net client + state). No correctness impact but increases navigation and review cost. **Remediation:** Follow the top-20 placement suggestions from `go-stats-generator analyze . --skip-tests --sections patterns --format json`. Priority moves: `PerformanceMetrics`/`MemoryMetrics`/`GCMetrics` → `metrics.go`; `ConnectionStatusMessage` → `dashboard.go`. Validate: `go build ./...` and `go test -race ./...` after each move.

---

## Previous-Cycle Findings — Now Closed

The following findings from the prior audit cycle are **fully resolved** in the
current codebase and are recorded here for traceability:

| Former ID | Title | Resolution |
|-----------|-------|------------|
| GAP-11 | Unbalanced `RUnlock()` in `handleConnection` panics on every live connection | Fixed: `gs.mutex.RLock()` added before map read at `connection.go:35`. Integration test `TestHandleWebSocket_NewConnection` added. |
| GAP-12 | `performComponent` registered as valid action but always errors | Fixed: `ActionComponent` removed from `isValidActionType`; server now returns explicit "invalid action type" error. Test `TestProcessAction_ComponentActionRejected` added. |
| GAP-13 | Win condition implementation contradicts player-count scaling in README | Fixed: `rescaleActDeck` called from `registerPlayer` at game start; validated by `TestRulesActAgendaProgression_PlayerCountScaling` (all 6 player counts pass). |
| GAP-14 | JS reconnect token stored in-memory only — not in `localStorage` | Fixed: `localStorage.setItem` at `game.js:152`; `localStorage.getItem` at `game.js:9`. |
| GAP-15 | `gs.connections` read under wrong mutex in `trackConnection` | Fixed: `trackConnection` no longer reads `len(gs.connections)`; uses `atomic.LoadInt64(&gs.activeConnections)` instead. |
| GAP-16 | No tests for `app` and `render` packages | Partially fixed: `game_test.go` and `atlas_test.go` added with `requires_display` build tag (see M-02). |

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total lines of code | 2,347 |
| Total functions | 53 |
| Total methods | 134 |
| Total structs | 61 |
| Total interfaces | 2 (`Broadcaster`, `StateValidator`) |
| Total packages | 5 |
| Total files | 24 |
| Average function length | 14.1 lines |
| Average cyclomatic complexity | 4.2 |
| Max cyclomatic complexity | 9 (`cleanupDisconnectedPlayers`) |
| Functions > 50 lines | 3 (1.6%) |
| Documentation coverage | 96.6% |
| Dead code functions | 31 |
| Magic number / string literals | 1,013 |
| Circular dependencies | 0 |
| Low-cohesion files | 11 |
| `go vet ./...` warnings | 0 |
| `go test -race ./...` failures | 0 |
| Open known CVEs (gorilla/websocket v1.5.3) | 0 |

---

*Generated by functional audit. Primary tools: `go-stats-generator v1.0.0`, `go vet`, `go test -race ./...`, manual code inspection.*
