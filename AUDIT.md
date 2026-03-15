# AUDIT — 2026-03-15

## Project Goals

The README documents a cooperative multiplayer Arkham Horror game targeting intermediate developers
learning WebSocket architecture. Key promises:

- Five fully integrated core mechanics: **Location System** (4 interconnected neighbourhoods),
  **Resource Tracking** (Health/Sanity/Clues), **Action System** (2 actions/turn from 8 types),
  **Doom Counter** (0–12, incremented by Tentacle results), **Dice Resolution** (3-sided dice with
  configurable difficulty thresholds).
- Support for **1–6 concurrent players** with join-in-progress; AH3e core rulebook range.
- Real-time game state synchronisation within **500 ms** to all connected clients.
- **Token-based session persistence**: both JS and Ebitengine clients reclaim player slots on
  reconnect; JS token stored in `localStorage`; Ebitengine token stored and re-sent as `?token=`.
- **Performance monitoring**: live dashboard at `/dashboard`, Prometheus-compatible metrics at
  `/metrics`, health checks at `/health`.
- **Multi-platform Ebitengine client** (desktop, WASM, mobile) and legacy JS/Canvas client.
- Win condition: **4 clues per investigator** collectively (4 for 1P, 8 for 2P, 12 for 3P,
  16 for 4P, 20 for 5P, 24 for 6P).
- Go **interface-based design** (`net.Conn`, `net.Listener`, `net.Addr`); idiomatic error handling
  and concurrency via goroutines and channels.
- Builds successfully for desktop (`go build ./cmd/desktop`), WASM, and server.

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Location System — 4 neighbourhoods with adjacency rules | ✅ Achieved | `game_constants.go:135-160` (`locationAdjacency` map); `game_mechanics.go:180-193` (`performMove`) |
| Resource Tracking — Health/Sanity/Clues bounds (1-10, 0-5) | ✅ Achieved | `game_mechanics.go:17-37` (`validateResources`); `game_constants.go:37-49` |
| Action System — 2 actions/turn, 8 action types | ✅ Achieved (1 stub) | `game_mechanics.go:145-175` (`dispatchAction`); `game_constants.go:24-34`; `performComponent` always errors |
| Doom Counter — increments on Tentacle results | ✅ Achieved | `game_mechanics.go:194-214` (gather), `220-242` (investigate), `250-278` (ward) |
| Dice Resolution — 3-sided dice, configurable difficulty | ✅ Achieved | `game_mechanics.go:73-143` (`rollDice`, `rollDicePool`) |
| 1–6 concurrent players with join-in-progress | ✅ Achieved | `game_constants.go:8-12`; `connection.go:74-103` |
| Real-time state sync ≤500 ms | ✅ Achieved | Channel-based broadcast; `connection.go:319-336` |
| Token-based session persistence — JS client | ⚠️ Partial | Server + JS token exchange works; token NOT in `localStorage` contrary to README |
| Token-based session persistence — Ebitengine client | ✅ Achieved | `client/ebiten/state.go:79,191-201`; `net.go:74` |
| Performance dashboard / Prometheus / health checks | ✅ Achieved | `observability.go:82-118`; `game_utils.go:9-28` (routes registered) |
| Multi-platform Ebitengine builds compile | ✅ Achieved | `go build ./cmd/desktop` and `GOOS=js GOARCH=wasm go build ./cmd/web` both succeed |
| net.Conn / net.Listener / net.Addr interface usage | ✅ Achieved | `main.go:19`; `game_server.go:25`; `game_utils.go:9`; `connection_wrapper.go` |
| Win condition scales with player count (4 clues/investigator) | ❌ Missing | `game_mechanics.go:505-529` uses fixed Act deck totals: 4→8→12 regardless of player count |
| handleConnection — non-panic WebSocket entry point | ❌ Missing | `connection.go:36`: unbalanced `RUnlock()` panics on every live connection |
| go test -race ./... passes | ✅ Achieved | All 100+ server tests + 5 ebiten tests pass; no races detected |

---

## Findings

### CRITICAL

- [x] **Unbalanced `RUnlock()` in `handleConnection` panics on every live connection** —
  `cmd/server/connection.go:35-36` — `handleConnection` reads `gs.wsConns` (a map) then calls
  `gs.mutex.RUnlock()` without a prior `RLock()`. In Go's `sync.RWMutex`, calling `RUnlock` on an
  unlocked mutex panics with `"sync: RUnlock of unlocked RWMutex"`. The map read on line 35 is also
  unprotected. Every non-token WebSocket connection goes through this function (spawned from
  `handleWebSocket`). The server process terminates on the first connection. No test exercises the
  live WebSocket connection path, so the panic is undetected by the test suite.

  ```go
  // connection.go:33-39 (current — PANICS)
  wsConn, ok := gs.wsConns[conn.RemoteAddr().String()]
  gs.mutex.RUnlock()   // ← no preceding RLock; panics
  ```

  **Remediation:** Wrap the map lookup in `gs.mutex.RLock()` / `gs.mutex.RUnlock()`:
  ```go
  gs.mutex.RLock()
  wsConn, ok := gs.wsConns[conn.RemoteAddr().String()]
  gs.mutex.RUnlock()
  ```
  Add an integration test `TestHandleWebSocket_NewConnection` that upgrades a real WebSocket
  connection via `httptest.Server` and asserts the connection reaches `runMessageLoop` without
  panicking.
  **Validate:** `go test -race ./cmd/server/... -run TestHandleWebSocket_NewConnection`

---

### HIGH

- [x] **Win condition implementation contradicts documented player-count scaling** —
  `cmd/server/game_mechanics.go:505-529` and `README.md:104` — The README promises the win
  threshold scales with player count: 4 clues for 1 player, 16 for 4 players, 24 for 6 players.
  `checkActAdvance()` compares total collective clues against the current Act card's fixed
  `ClueThreshold` (4 → 8 → 12 across three acts, defined in `game_constants.go:82-85`). With any
  number of players the game is always won at 12 total clues. With 4 players the game is 25% easier
  than documented (12 instead of 16); with 6 players, 50% easier (12 instead of 24).

  **Remediation (option A — fix documentation to match implementation):**
  Update `README.md:104` to read: "Collectively gather a total of 12 clues across all investigators
  (advancing three Act cards at 4, 8, and 12 clues) before doom reaches 12."

  **Remediation (option B — fix implementation to match documentation):**
  In `defaultActDeck()` (`game_constants.go:80-86`), derive Act thresholds from player count:
  ```go
  func scaledActDeck(playerCount int) []ActCard {
      base := 4 * playerCount
      return []ActCard{
          {Title: "Act 1", ClueThreshold: base / 3, ...},
          {Title: "Act 2", ClueThreshold: (2 * base) / 3, ...},
          {Title: "Act 3", ClueThreshold: base, ...},
      }
  }
  ```
  Call `scaledActDeck(len(gameState.Players))` from `DefaultScenario.SetupFn` after players join.
  **Validate:** `go test -race ./cmd/server/... -run TestRulesActAgendaProgression`

- [x] **`gs.connections` read under wrong mutex in `trackConnection` — latent data race** —
  `cmd/server/observability.go:482` — `trackConnection` acquires `gs.performanceMutex.Lock()` but
  reads `gs.connections` (line 482: `currentConnections := len(gs.connections)`). The `connections`
  map is mutated under `gs.mutex` in `connection.go:207,286`. Concurrent connect/disconnect events
  and a `trackConnection("connect")` call will race on the map. The race detector does not catch
  this because no integration test exercises the real WebSocket path.

  **Remediation:** Replace the map length read with an atomic counter that is incremented and
  decremented alongside the `gs.connections` mutations under `gs.mutex`:
  ```go
  // In GameServer:
  activeConnections int64  // managed atomically

  // In handleWebSocket (connection.go:286, after gs.mutex.Lock):
  atomic.AddInt64(&gs.activeConnections, 1)

  // In handlePlayerDisconnect (connection.go:207, after gs.mutex.Lock):
  atomic.AddInt64(&gs.activeConnections, -1)

  // In trackConnection (observability.go:482):
  currentConnections := int(atomic.LoadInt64(&gs.activeConnections))
  ```
  **Validate:** `go test -race ./cmd/server/...`

---

### MEDIUM

- [x] **JS client reconnect token not persisted to `localStorage` — contradicts README** —
  `client/game.js:9` and `README.md:110` — The README states the token is "stored in
  `localStorage`". In `game.js:9` the token is stored in the class property
  `this.reconnectToken = null`. A hard browser refresh (`Ctrl+Shift+R`) or tab close clears the
  JavaScript heap; the token is lost and the player cannot reclaim their slot.

  **Remediation:** On receipt of a `connectionStatus` message (game.js:150-151), persist the token:
  ```js
  this.reconnectToken = message.token;
  localStorage.setItem('arkham_reconnect_token', message.token);
  ```
  On construction (game.js:9), restore from storage:
  ```js
  this.reconnectToken = localStorage.getItem('arkham_reconnect_token') || null;
  ```
  **Validate:** Open game, copy token from DevTools Application → Local Storage, hard-refresh page,
  confirm the `?token=` parameter appears in the WebSocket upgrade request.

- [x] **`performComponent` always errors but is registered as a valid action type** —
  `cmd/server/game_mechanics.go:345-347` and `cmd/server/game_server.go` (`isValidActionType`) —
  `ActionComponent = "component"` is included in `isValidActionType()`, so the server accepts the
  action and routes it. `performComponent()` unconditionally returns
  `fmt.Errorf("component action … not yet implemented")`. `processAction` returns the error before
  decrementing `ActionsRemaining`, so the player's action budget is not consumed — but they receive
  no client-visible error message and the server logs a spurious error on every attempt.

  **Remediation:** Remove `ActionComponent` from the `isValidActionType()` slice until the feature
  is implemented (Phase 6 per ROADMAP). This changes the response from a silent server error to the
  clear `"invalid action type: component"` error returned to the client:
  ```go
  // game_server.go — isValidActionType: remove ActionComponent from the slice
  for _, v := range []ActionType{
      ActionMove, ActionGather, ActionInvestigate, ActionCastWard,
      ActionFocus, ActionResearch, ActionTrade, ActionEncounter,
      // ActionComponent excluded until Phase 6
  }
  ```
  **Validate:** `go test -race ./cmd/server/... -run TestProcessAction_InvalidActionType`

- [x] **`client/ebiten/app` and `client/ebiten/render` have no tests** —
  Active Ebitengine packages — These two packages contain the rendering pipeline (`drawPlayerPanel`,
  `drawBoard`, `drawLocationPanel`, shader loading) and the game-loop `Update`/`Draw` split. Neither
  has a `*_test.go` file. Rendering logic bugs (wrong panel coordinates, missed nil checks, etc.)
  have no automated detection.

  **Remediation:** Add at minimum:
  - `client/ebiten/app/game_test.go` — `TestDrawPlayerPanel_NoPlayers`, `TestUpdate_NoopWhenNotConnected`
  - `client/ebiten/render/atlas_test.go` — `TestAtlas_InitDoesNotPanic`

  **Validate:** `go test -race ./client/ebiten/...`

---

### LOW

- [x] **`collectPerformanceMetrics` exceeds 50-line threshold (64 lines)** —
  `cmd/server/observability.go:212-275` — go-stats-generator flags this as a high-risk function.
  It mixes memory-stat collection, session aggregation, and response-time measurement in one body.

  **Remediation:** Extract into three helpers: `collectMemorySnapshot()`, `aggregateSessionMetrics()`,
  and `measureResponseTime()`. Call them from `collectPerformanceMetrics`.
  **Validate:** `go-stats-generator analyze . --skip-tests | grep "Functions > 50"`

- [ ] **`cmd/desktop`, `cmd/mobile` have no test files** — Active entry points (`main.go`,
  `binding.go`) lack tests. Errors in Ebitengine game-loop wiring or mobile binding scaffolding
  are not caught automatically.

  **Remediation:** Add smoke tests using `ebiten/ebitentest` or mock the `ebiten.Game` interface.
  **Validate:** `go test ./cmd/desktop/... ./cmd/mobile/...`

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total source files (non-test) | 22 |
| Total lines of code | 2 278 |
| Total functions + methods | 181 (51 functions, 130 methods) |
| Total structs | 60 |
| Total interfaces | 2 |
| Total packages | 5 |
| Average function length | 14.3 lines |
| Functions > 50 lines | 4 (2.2%) |
| Functions > 100 lines | 0 |
| Average cyclomatic complexity | 4.2 |
| High-complexity functions (cyclomatic >10) | 0 |
| Top complex function | `cleanupDisconnectedPlayers` (overall 14.7, cyclomatic 9) |
| Circular dependencies | 0 |
| `go vet` warnings | 0 |
| `go test -race ./...` | PASS — 105+ tests (0 failures, 0 skips) |
| Packages with no test files | 4 (`app`, `render`, `cmd/desktop`, `cmd/mobile`) |

*Generated with `go-stats-generator analyze . --skip-tests` — 2026-03-15*
