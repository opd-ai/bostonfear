# Implementation Gaps — 2026-03-15

> This file covers gaps between the project's stated goals (README, RULES.md,
> CLIENT_SPEC.md, ROADMAP.md) and the current implementation, ordered by severity.
> Cross-reference with `AUDIT.md` for full evidence and remediation commands.
>
> **Previous-cycle gaps (GAP-01 through GAP-10) resolved since the last report:**
> GAP-01 (Ebitengine gameState decode), GAP-02 (ConnectionWrapper.Read byte count),
> GAP-03 (drawMythosToken determinism), GAP-04 (Ebitengine reconnect token),
> GAP-05 (README session-persistence description), GAP-07 (totalGamesPlayed over-count),
> GAP-09 (Focus token integration), GAP-10 (Anomaly/gate mechanics stubs) are all closed.
> GAP-06 (performComponent stub) and GAP-08 (Ebitengine test coverage) are partially addressed
> and carried forward as GAP-12 and GAP-14 below.

---

## GAP-11: Unbalanced `RUnlock()` in `handleConnection` Panics on Every Live Connection

- **Stated Goal**: README §Technical Implementation — "Concurrent Connection Handling:
  Goroutines with channel-based communication." The implication is that the server
  successfully accepts and manages WebSocket connections.
- **Current State**: `cmd/server/connection.go:35-36` —
  ```go
  wsConn, ok := gs.wsConns[conn.RemoteAddr().String()]
  gs.mutex.RUnlock()   // ← called without a prior RLock
  ```
  `handleConnection` calls `gs.mutex.RUnlock()` with no matching `RLock()`. Go's
  `sync.RWMutex` panics with `"sync: RUnlock of unlocked RWMutex"` on the first
  invocation. The map read on line 35 is also unprotected (no lock held). Every new
  non-token WebSocket connection is routed through `handleConnection` via a goroutine
  launched from `handleWebSocket:305-309`. No test exercises the live HTTP/WebSocket
  path, so the panic goes undetected by `go test -race ./...`.
- **Impact**: The server process crashes on the first player connection attempt.
  No gameplay is possible. The entire stated goal of supporting 1–6 concurrent
  players is unreachable until this is fixed.
- **Closing the Gap**:
  1. Wrap the map lookup in a matching read-lock pair (connection.go:35):
     ```go
     gs.mutex.RLock()
     wsConn, ok := gs.wsConns[conn.RemoteAddr().String()]
     gs.mutex.RUnlock()
     ```
  2. Add an integration test using `httptest.NewServer` and `gorilla/websocket.Dial`
     that upgrades a real connection and confirms the server reaches `runMessageLoop`
     without panicking.
  3. Run: `go test -race ./cmd/server/... -run TestHandleWebSocket_NewConnection`

---

## GAP-12: `performComponent` Registered as Valid Action but Always Errors

- **Stated Goal**: RULES.md §Full Action Set lists Component as a valid investigator
  action. `game_constants.go:30` defines `ActionComponent = "component"`.
  `isValidActionType()` accepts it. A client that sends `{"action":"component"}`
  expects either execution or a clear "not available" response.
- **Current State**: `cmd/server/game_mechanics.go:345-347` —
  `performComponent()` unconditionally returns
  `fmt.Errorf("component action for player %s: not yet implemented")`.
  `processAction` returns this error before decrementing `ActionsRemaining`, so
  the player's action budget is not consumed — but no client-visible error is
  surfaced and the server logs a spurious error.
- **Impact**: Players or developers testing the full action set receive an opaque
  server-side error without feedback. Server error logs are polluted by expected
  stub invocations, making real errors harder to spot.
- **Closing the Gap**:
  1. Short-term: Remove `ActionComponent` from `isValidActionType()` so the server
     returns the clear `"invalid action type: component"` error. This converts a
     silent opaque error into an explicit documented limitation.
  2. Long-term (ROADMAP Phase 6): Implement per-investigator ability tables and
     wire them into `performComponent()`.
  3. Run: `go test -race ./cmd/server/... -run TestProcessAction_InvalidActionType`

---

## GAP-13: Win Condition Implementation Contradicts Player-Count Scaling in README

- **Stated Goal**: README §Win/Lose Conditions:
  > "Win: Collectively gather **4 clues per investigator** before doom reaches 12
  > (4 clues for 1 player, 8 for 2, 12 for 3, 16 for 4, 20 for 5, 24 for 6)"
- **Current State**: `cmd/server/game_mechanics.go:505-529` (`checkActAdvance`) sums
  all players' clues and compares against the current Act card's `ClueThreshold`
  (defined in `game_constants.go:82-85` as a fixed three-act deck: 4 → 8 → 12
  cumulative). The win threshold is always 12 total collective clues, regardless of
  how many players are in the game.

  | Players | README promise | Implementation |
  |---------|---------------|----------------|
  | 1 | 4 clues | 12 clues |
  | 2 | 8 clues | 12 clues |
  | 3 | 12 clues | 12 clues ✅ |
  | 4 | 16 clues | 12 clues |
  | 5 | 20 clues | 12 clues |
  | 6 | 24 clues | 12 clues |

  Solo players face a 3× harder win condition than documented; 4-player groups face
  a 25% easier one. Only 3-player games match the README.
- **Impact**: Players experience a substantially different difficulty curve from what
  is documented. Solo play (1 player) is effectively unwinnable relative to
  expectations. Multi-player games (4–6) are easier than documented.
- **Closing the Gap** — two options:
  **Option A (fix the documentation):** Update README:104 to reflect the fixed
  12-clue requirement: "Collectively gather 12 total clues across all investigators
  (advancing three Act cards at 4, 8, and 12 cumulative clues) before doom reaches 12."

  **Option B (fix the implementation to match the README):** Derive Act thresholds
  from player count in `DefaultScenario.SetupFn` (`game_constants.go`):
  ```go
  SetupFn: func(gs *GameState) {
      n := max(len(gs.Players), 1)
      base := 4 * n
      gs.ActDeck = []ActCard{
          {Title: "Act 1", ClueThreshold: base / 3},
          {Title: "Act 2", ClueThreshold: (2 * base) / 3},
          {Title: "Act 3", ClueThreshold: base},
      }
  },
  ```
  Because `SetupFn` is called at server init before players join, a further hook
  is needed to re-derive thresholds when the first player connects. The simplest
  approach is to call a `rescaleActDeck` helper from `registerPlayer` when
  `!gs.gameState.GameStarted`.

  Run: `go test -race ./cmd/server/... -run TestRulesActAgendaProgression`

---

## GAP-14: JS Client Reconnect Token Stored In-Memory Only — Not in `localStorage`

- **Stated Goal**: README §Connection Behaviour:
  > "The JS legacy client reclaims its player slot automatically using a
  > server-issued reconnect token (stored in `localStorage`)."
- **Current State**: `client/game.js:9` initialises `this.reconnectToken = null`
  as an in-memory class property. `game.js:151` stores the received token:
  `this.reconnectToken = message.token`. No call to `localStorage.setItem` exists
  anywhere in `game.js`. A hard browser refresh (`Ctrl+Shift+R`), tab close, or
  browser restart clears the JavaScript heap; the token is permanently lost.
  On reconnect the client dials `/ws` with no `?token=` parameter, creating a new
  player slot in the game state and leaving the original slot as a zombie.
- **Impact**: The session-persistence guarantee documented in the README works only
  within a single page session. Any browser navigation event destroys the player's
  investigator and clutters the game's turn order with stale zombie entries until
  the 5-minute reaper (`cleanupDisconnectedPlayers`) runs.
- **Closing the Gap**:
  1. On receipt of `connectionStatus` (game.js:150-151), also write to storage:
     ```js
     this.reconnectToken = message.token;
     localStorage.setItem('arkham_reconnect_token', message.token);
     ```
  2. In the `ArkhamGame` constructor (game.js:9), restore the token on load:
     ```js
     this.reconnectToken = localStorage.getItem('arkham_reconnect_token') || null;
     ```
  3. On a confirmed `"connected"` status after a successful token-based restore,
     consider clearing stale tokens from storage is unnecessary (the server rotates
     the token on restore).
  4. Validate manually: open game, copy token from DevTools → Application →
     Local Storage, hard-refresh, confirm `?token=` appears in the WS upgrade URL.

---

## GAP-15: `gs.connections` Read Under Wrong Mutex in `trackConnection`

- **Stated Goal**: README §Go Server Architecture — "Concurrent Connection Handling:
  Goroutines with channel-based communication." Thread-safe access to shared maps
  is required for correct concurrent operation.
- **Current State**: `cmd/server/observability.go:461-488` (`trackConnection`) holds
  `gs.performanceMutex.Lock()` and reads `len(gs.connections)` (line 482). The
  `connections` map is written under `gs.mutex` in `connection.go:207` and
  `connection.go:286`. A goroutine running `handleWebSocket` (writing under
  `gs.mutex`) can race concurrently with `trackConnection` (reading without
  `gs.mutex`). The race detector does not catch this because no integration test
  exercises the live WebSocket connection path.
- **Impact**: Under concurrent connections (the server's primary use case), this is
  an undetected data race that can cause silent map-read corruption or incorrect
  `peakConnections` metrics. Go's map implementation has no internal synchronisation;
  a concurrent write while a goroutine reads will corrupt the map's internal state.
- **Closing the Gap**:
  1. Replace the `len(gs.connections)` map read in `trackConnection` with an atomic
     counter maintained alongside the map mutations:
     ```go
     // GameServer field:
     activeConnections int64  // incremented/decremented atomically

     // connection.go:286 (after gs.mutex.Lock):
     atomic.AddInt64(&gs.activeConnections, 1)

     // connection.go:207 (after gs.mutex.Lock):
     atomic.AddInt64(&gs.activeConnections, -1)

     // observability.go:482 — replace len(gs.connections):
     currentConnections := int(atomic.LoadInt64(&gs.activeConnections))
     ```
  2. Apply the same substitution to `collectPerformanceMetrics:218` and
     `handleHealthCheck:26` (both currently read `gs.connections` under `gs.mutex`
     which is correct, but the atomic counter simplifies all three sites).
  3. Run: `go test -race ./cmd/server/...` (add an integration test exercising
     concurrent connections to give the race detector a chance to catch this class
     of bug).

---

## GAP-16: No Tests for `client/ebiten/app`, `client/ebiten/render` Packages

- **Stated Goal**: README §Ebitengine Client Features — "Sprite/Layer Rendering:
  Board, tokens, UI overlays, and animations via Ebitengine draw layers." Active
  rendering code warrants test coverage.
- **Current State**: `client/ebiten/app/` (2 files: `game.go`, `input.go`) and
  `client/ebiten/render/` (3 files: `atlas.go`, `layers.go`, `shaders.go`) have no
  `*_test.go` files. Only `client/ebiten/net_test.go` (5 tests) exists for the
  entire Ebitengine client tree.
- **Impact**: Regressions in rendering logic (e.g., nil player dereferences in
  `drawPlayerPanel`, shader compilation failures, atlas initialisation panics) are
  not caught automatically. The `app` package has the highest coupling score (5
  dependencies) in the project, making it the most likely site for interface
  contract violations.
- **Closing the Gap**:
  1. `client/ebiten/app/game_test.go`:
     - `TestUpdate_DoesNotPanicWhenNotConnected` — call `Update()` with empty state
     - `TestDrawPlayerPanel_NilPlayers` — call with nil/empty Players map
  2. `client/ebiten/render/atlas_test.go`:
     - `TestNewAtlas_ReturnsNonNilAtlas`
     - `TestAtlas_DoesNotPanicOnDraw`
  3. Run: `go test -race ./client/ebiten/...`

---

## Summary Table

| Gap ID | Area | Severity | Stated Goal vs Reality |
|--------|------|----------|------------------------|
| GAP-11 | `handleConnection` unbalanced RUnlock | CRITICAL | Server panics on first real connection |
| GAP-13 | Win condition player-count scaling | HIGH | README: 4 clues/player; code: always 12 total |
| GAP-15 | `gs.connections` read under wrong mutex | HIGH | Data race under concurrent connections |
| GAP-12 | `performComponent` always errors | MEDIUM | Valid action type silently fails |
| GAP-14 | JS reconnect token not in localStorage | MEDIUM | README claims localStorage; code uses memory only |
| GAP-16 | No tests for `app` and `render` packages | MEDIUM | Active rendering code has no regression safety net |
