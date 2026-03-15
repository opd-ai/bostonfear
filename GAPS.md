# Implementation Gaps — 2026-06-01 (updated)

> This file covers gaps between the project's stated goals (README, RULES.md,
> CLIENT_SPEC.md, ROADMAP.md) and the current implementation, ordered by severity.
>
> **Previous-cycle gaps resolved since the last report:**
> GAP-11 (handleConnection unbalanced RUnlock), GAP-12 (performComponent always
> errors), GAP-13 (win condition player-count scaling), GAP-14 (JS localStorage
> token), GAP-15 (gs.connections wrong mutex), GAP-16 (app/render test files),
> GAP-17 (/health+/metrics nested RLock deadlock), GAP-18 (gs.gameState.Doom
> unlocked read), GAP-19 (win threshold not rescaled on late join),
> GAP-20 (ActionComponent dead code — promoted to live feature),
> GAP-21 (Ebitengine tests skipped — documented in README; display-independent
> tests blocked by GLFW init() panic, tracked as open below).
>
> **No actionable HIGH or MEDIUM gaps remain.** One LOW-severity documentation gap
> (GAP-21) persists. See summary table below.

---

## GAP-17: Latent Deadlock in `/health` and `/metrics` Under Concurrent Player Actions

- **Stated Goal**: README §Performance Standards — "Sub-100ms response times for
  health checks" and "Maintains stable operation with 6 concurrent players."
  README §Go Server Architecture — "Concurrent Connection Handling: Goroutines
  with channel-based communication."
- **Current State**: `cmd/server/health.go:19-31` (`handleHealthCheck`) acquires
  `gs.mutex.RLock()` and, while holding it, calls:
  - `collectPerformanceMetrics()` (`metrics.go:161`) → `measureHealthCheckResponseTime()` (`health.go:104`) → `gs.mutex.RLock()` **(double-RLock, same goroutine)**
  - `getGameStatistics()` (`health.go:132`) → `gs.mutex.RLock()` **(double-RLock, same goroutine)**
  - `getSystemAlerts()` (`health.go:198`) → `collectPerformanceMetrics()` → same nested chain, *plus* `gs.mutex.RLock()` directly at `health.go:232` **(double-RLock, same goroutine)**

  `handleMetrics` (`metrics.go:17-28`) has the identical pattern through its call to
  `collectPerformanceMetrics()`.

  Go's `sync.RWMutex` is **not reentrant**. From the standard library docs:
  > "If any goroutine calls Lock while the lock is already held by one or more
  > readers, concurrent calls to RLock will block until the writer has acquired
  > (and released) the lock."
  This applies to the same goroutine: if `processAction` holds `gs.mutex.Lock()`
  (a write lock) between the outer `RLock()` at line 19 and any of the inner
  `RLock()` calls, the inner `RLock()` blocks forever while the writer waits for
  the outer `RLock()` to be released — a true deadlock. Under 6 concurrent players
  (the server's primary use case) sending actions every few seconds, a health check
  or metrics scrape during action processing will trigger this deadlock, hanging
  the HTTP handler goroutine and making the endpoint permanently unresponsive until
  the server is restarted.

- **Impact**: The monitoring system — the very subsystem intended to detect and
  diagnose server problems — can deadlock under normal game load, making `/health`
  and `/metrics` unresponsive. This violates both the sub-100ms health check SLA
  and the stable-operation goal. The deadlock is silent (no panic, no log entry)
  and will not be caught by `go test -race ./...` without an integration test that
  combines concurrent write-lock holders with health check requests.

- **Closing the Gap**:
  1. In both `handleHealthCheck` and `handleMetrics`, snapshot all game state
     fields under a **single** `gs.mutex.RLock()` block at the top of the handler,
     then **release** the lock before calling any helper:
     ```go
     gs.mutex.RLock()
     doom        := gs.gameState.Doom
     gamePhase   := gs.gameState.GamePhase
     playerCount := len(gs.gameState.Players)
     connCount   := len(gs.connections)
     gameStarted := gs.gameState.GameStarted
     // ... any other gs.gameState fields needed
     gs.mutex.RUnlock()  // lock released before any helper call

     perfMetrics  := gs.collectPerformanceMetrics()
     connAnalytics := gs.collectConnectionAnalytics()
     // ...
     ```
  2. Remove the `gs.mutex.RLock/RUnlock` pair from `measureHealthCheckResponseTime`
     (`health.go:108-111`) — the fields it reads must be passed as pre-snapshotted
     values or the function must be replaced by an inline operation.
  3. Remove the `gs.mutex.RLock/RUnlock` pair from `getGameStatistics`
     (`health.go:132-133`) and pass the snapshotted `gameState.Players` map as an
     argument instead.
  4. Remove the `gs.mutex.RLock/RUnlock` pair from the doom-alert section in
     `getSystemAlerts` (`health.go:232-234`) and pass the pre-snapshotted `doom`
     value.
  5. Add an integration test that hammers `/health` and `/metrics` with concurrent
     player actions to give the race detector a chance to catch this class of bug.
  6. Validate: `go test -race ./cmd/server/... -run TestHandleHealthCheck_ConcurrentActions`

---

## GAP-18: `gs.gameState.Doom` Read Without Lock in Alert Messages

- **Stated Goal**: README §Go Server Architecture — "Centralized game state with
  mutex protection." All reads and writes of `gs.gameState` must be performed while
  holding the appropriate `gs.mutex` lock.
- **Current State**: In `getSystemAlerts` (`cmd/server/health.go:239,245`), after
  `gs.mutex.RUnlock()` at line 234, the bare field `gs.gameState.Doom` is read
  inside two `fmt.Sprintf` calls:
  ```go
  gs.mutex.RLock()
  doomPercent := float64(gs.gameState.Doom) / 12.0 * 100
  gs.mutex.RUnlock()  // lock released here

  if doomPercent > 80 {
      alerts = append(alerts, map[string]interface{}{
          "message": fmt.Sprintf("Critical doom level: %d/12 ...", gs.gameState.Doom, ...),
          //                                             ^^^ UNLOCKED READ
      })
  } else if doomPercent > 60 {
      alerts = append(alerts, map[string]interface{}{
          "message": fmt.Sprintf("High doom level: %d/12 ...", gs.gameState.Doom, ...),
          //                                        ^^^ UNLOCKED READ
      })
  }
  ```
  Any concurrent call to `processAction` holding `gs.mutex.Lock()` and writing
  `gs.gameState.Doom` races with these reads. The race detector does not catch this
  in the current test suite because no test calls `getSystemAlerts` concurrently
  with an action-processing goroutine.
- **Impact**: A data race on an `int` field is benign on most architectures (torn
  reads are unlikely), but it is undefined behaviour under the Go memory model and
  can cause incorrect alert messages ("Critical doom level: 0/12") to be sent to
  operators, masking real danger. It is also a correctness violation of the
  project's own stated mutex-protection convention.
- **Closing the Gap**:
  1. Capture `gs.gameState.Doom` in the `gs.mutex.RLock()` block (line 232) and
     use the captured copy in the `Sprintf` calls:
     ```go
     gs.mutex.RLock()
     doom        := gs.gameState.Doom
     doomPercent := float64(doom) / 12.0 * 100
     gs.mutex.RUnlock()

     if doomPercent > 80 {
         alerts = append(alerts, map[string]interface{}{
             "message": fmt.Sprintf("Critical doom level: %d/12 (%.0f%%)", doom, doomPercent),
         })
     } else if doomPercent > 60 {
         alerts = append(alerts, map[string]interface{}{
             "message": fmt.Sprintf("High doom level: %d/12 (%.0f%%)", doom, doomPercent),
         })
     }
     ```
  2. Validate: `go test -race ./cmd/server/... -run TestGetSystemAlerts` with a
     goroutine concurrently incrementing `gs.gameState.Doom`.

---

## GAP-19: Win Threshold Not Rescaled When Player Joins Mid-Game

- **Stated Goal**: README §Win/Lose Conditions:
  > "Win: Collectively gather **4 clues per investigator** before doom reaches 12
  > (4 clues for 1 player, 8 for 2, 12 for 3, 16 for 4, 20 for 5, 24 for 6)"
  README §Multiplayer Features:
  > "Join a game already in progress — late joiners enter the turn rotation
  > automatically."
- **Current State**: `cmd/server/connection.go:98-107` — `rescaleActDeck` is called
  only in the `!gs.gameState.GameStarted` branch (game-start path). Because
  `MinPlayers=1`, the first player always triggers game start. `rescaleActDeck(1)`
  is called, setting the final Act threshold to 4 clues. When a second player joins
  via the `gs.gameState.GameStarted && GamePhase == "playing"` branch (line 104),
  `rescaleActDeck` is not called. The threshold remains 4, not 8.

  | Scenario | README promise | Implementation |
  |----------|---------------|----------------|
  | 1P start | 4 clues | 4 clues ✅ |
  | 2P late join | 8 clues | 4 clues ❌ |
  | 3P late join | 12 clues | 4 clues ❌ |

- **Impact**: A 2-player game where player 2 joins after player 1 has started is
  75% easier than documented (4 vs 16 cumulative clues across both players for 4
  three-act thresholds). Players who follow the README's cooperative strategy
  (distribute 4 clue-finding attempts per investigator) will win almost instantly,
  breaking the cooperative game loop.
- **Closing the Gap**:
  Add a `rescaleActDeck` call in the late-join branch of `registerPlayer`:
  ```go
  } else if gs.gameState.GameStarted && gs.gameState.GamePhase == "playing" {
      log.Printf("Player %s joined game in progress (turn order position %d)",
          playerID, len(gs.gameState.TurnOrder))
      if len(gs.gameState.ActDeck) >= 3 {
          gs.rescaleActDeck(len(gs.gameState.Players))
      }
  }
  ```
  Validate: `go test -race ./cmd/server/... -run TestRulesActAgendaProgression_PlayerCountScaling`.
  Add a new test that connects player 1, then player 2, and asserts
  `gs.gameState.ActDeck[2].ClueThreshold == 8`.

---

## GAP-20: `ActionComponent` Dead Code in Dispatch Switch

- **Stated Goal**: RULES.md §Full Action Set lists Component as a valid investigator
  action. `game_constants.go:33` defines `ActionComponent = "component"`.
- **Current State**: `cmd/server/game_server.go:207-220` — `isValidActionType`
  explicitly excludes `ActionComponent`. `dispatchAction` (`game_mechanics.go:169`)
  still contains `case ActionComponent: actionErr = gs.performComponent(...)`, which
  is dead code — it is never reached because `validateActionRequest` will have
  already rejected the action. The `performComponent` stub at `game_mechanics.go:345`
  is similarly unreachable. The comment at `game_server.go:207` acknowledges this.
- **Impact**: No functional impact — the action is correctly rejected by
  `isValidActionType`. The dead case is a maintenance liability: future developers
  may conclude `ActionComponent` is handled, attempt to re-enable it without
  reading the exclusion, and introduce a regression. Server error logs from the stub
  error return (if the dead code were ever reached) would be misleading.
- **Closing the Gap**:
  1. Remove `case ActionComponent:` and the `performComponent` call from
     `dispatchAction` (`game_mechanics.go:169-170`).
  2. Remove or replace `performComponent` (`game_mechanics.go:341-348`) with a
     `// TODO Phase 6: implement per-investigator component abilities` comment at
     the call site in `isValidActionType`.
  3. Keep `ActionComponent` constant in `game_constants.go:33` (reserved for
     Phase 6) with a comment: `ActionComponent ActionType = "component" // Phase 6 — not yet active`.
  4. Validate: `go test -race ./cmd/server/... -run TestProcessAction_ComponentActionRejected`
     and `go build ./cmd/server`.

---

## GAP-21: Ebitengine `app` and `render` Tests Silently Skipped in Standard CI

- **Stated Goal**: README §Ebitengine Client Features — "Sprite/Layer Rendering:
  Board, tokens, UI overlays, and animations via Ebitengine draw layers." The
  Quick Setup guide instructs contributors to run `go test ./...` with no mention
  of build tags.
- **Current State**: `client/ebiten/app/game_test.go:9` and
  `client/ebiten/render/atlas_test.go:9` carry `//go:build requires_display`.
  Running `go test ./...` (the only command documented in the README) outputs
  `[no test files]` for both packages, silently skipping all rendering regression
  tests. Contributors unaware of the build tag will never run these tests locally.
- **Impact**: Regressions in the rendering path (nil player dereferences in
  `drawPlayerPanel`, atlas initialisation panics, shader compilation failures) are
  not caught in standard `go test ./...` runs. The `app` package has the highest
  coupling score (5 dependencies) in the project, making it the most likely site
  for interface contract violations.
- **Closing the Gap**:
  1. Extract display-independent assertions (struct construction, nil-safety,
     logic branches that do not call Ebitengine image/draw APIs) into tag-free
     `_test.go` files in the same packages, so they run in every `go test ./...`.
  2. Add to `README.md §Quick Setup` (or a `CONTRIBUTING.md`):
     ```
     # Run display-requiring Ebitengine tests (needs X11 or virtual display):
     go test -tags=requires_display -race ./client/ebiten/...
     ```
  3. Validate: `go test ./client/ebiten/app/... ./client/ebiten/render/...` shows
     at least one test case (not `[no test files]`) after extracting tag-free tests.

---

## Summary Table

| Gap ID | Area | Severity | Stated Goal vs Reality |
|--------|------|----------|------------------------|
| GAP-17 | `/health` and `/metrics` nested RLock deadlock | HIGH | Sub-100ms health SLA unreachable under player load; monitoring hangs |
| GAP-18 | `gs.gameState.Doom` unlocked read in alerts | HIGH | Mutex-protection convention violated; data race on doom field |
| GAP-19 | Win threshold not rescaled on mid-game join | MEDIUM | README: 4 clues/player; late-join games stay at 1P threshold |
| GAP-20 | `ActionComponent` dead code in dispatch switch | LOW | Unreachable case adds maintenance noise; performComponent stub misleads |
| GAP-21 | Ebitengine `app`/`render` tests skip in standard CI | MEDIUM | Rendering regressions not caught by documented `go test ./...` command |
