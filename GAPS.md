# Implementation Gaps — 2026-03-15

> This document supersedes all prior gap analyses.  Each entry records what the
> project documentation promises, what the code actually delivers, the impact on
> users and developers, and the concrete steps required to close the gap.

---

## GAP-01 — Deadlock in Connection-Quality Ping/Pong Path

- **Stated Goal**: Real-time latency measurement for each player with connection
  quality ratings (`excellent`/`good`/`fair`/`poor`) broadcast to all clients
  (README — Performance Monitoring; `game_server.go` comments at lines 1215–1354).
- **Current State**: `handlePongMessage` (game_server.go:1253) acquires
  `qualityMutex.Lock()` and then calls `broadcastConnectionQuality()` (line 1271),
  which immediately attempts `qualityMutex.RLock()` (line 1358).  Go's
  `sync.RWMutex` is not reentrant; the calling goroutine deadlocks.  Every player
  connection that successfully completes a ping round-trip will hang permanently,
  taking its read-loop goroutine with it and making the slot unresponsive.
- **Impact**: The ping timer fires after 5 seconds for every connected player.
  Once the first pong arrives, the affected goroutine deadlocks.  Under a 3-player
  game each player's connection goroutine will deadlock within the first 10 seconds,
  making the performance-monitoring feature the primary cause of server instability.
- **Closing the Gap**: Release `qualityMutex` before calling
  `broadcastConnectionQuality` (see `AUDIT.md` Finding #1 for the corrected code).
  Add a WebSocket integration test that connects, waits 10 seconds, sends a pong,
  and asserts the connection remains active.

---

## GAP-02 — Data Race on Shared Connection Maps

- **Stated Goal**: "Concurrent Connection Handling: Goroutines with channel-based
  communication" and "State Management: Centralized game state with mutex
  protection" (README — Technical Implementation).
- **Current State**: `handleConnection` reads `gs.wsConns` at line 437 with no
  mutex held.  The disconnect-cleanup block (lines 576-578) deletes from
  `gs.connections`, `gs.wsConns`, and `gs.playerConns` outside any mutex, while
  `broadcastHandler` iterates `gs.wsConns` under `gs.mutex.RLock()`.  This is an
  unsynchronised concurrent map access (data race) on two separate code paths.
- **Impact**: Under normal multi-player use the Go runtime can detect the race and
  panic with "concurrent map read and map write".  Even without a panic, the race
  can produce stale reads in `broadcastHandler` causing missed broadcasts.  The
  existing `go test -race` suite does not catch this because no WebSocket integration
  tests exist.
- **Closing the Gap**: (1) Wrap the line-437 read in `gs.mutex.RLock/RUnlock`.
  (2) Wrap lines 576-578 in `gs.mutex.Lock/Unlock`.  (3) Add an integration test
  with concurrent connect/disconnect cycles run under `-race`.

---

## GAP-03 — Game Permanently Stuck After Player Disconnect

- **Stated Goal**: "Handle connection drops with 30-second reconnection timeout" /
  "Automatic reconnection handling" (README — Multiplayer Features, Performance
  Standards).
- **Current State**: On disconnect a player is marked `Connected: false` but
  remains in `gameState.TurnOrder`.  When `advanceTurn` rotates to this player,
  `ActionsRemaining` is set to 2 (game_server.go:388) but no goroutine processes
  messages for this player; the 30-second I/O timeout can therefore never fire.
  The turn will never advance again.  With a single disconnect mid-game the entire
  cooperative session freezes.
- **Impact**: Any network drop — even a brief one — can permanently stall a
  multiplayer game.  This is the highest-impact functional failure for the stated
  audience of developers learning WebSocket-based multiplayer architecture; it
  contradicts the README example game flow that shows turns advancing correctly.
- **Closing the Gap**:
  1. During disconnect cleanup, remove the player from `TurnOrder` under
     `gs.mutex.Lock()`.
  2. If the removed player was `CurrentPlayer`, call `advanceTurn()` immediately.
  3. Alternatively, add a ticker goroutine that advances the turn after 30 s if the
     current player is `Connected: false`.
  See also GAP-04 (session restore) for the complementary reconnection story.

---

## GAP-04 — Reconnecting Player Cannot Restore Session

- **Stated Goal**: "Automatic reconnection handling" / reconnect after drops
  (README — Multiplayer Features). README Connection Behaviour note: "In the current
  version, a disconnected player cannot reclaim their investigator. Reconnecting
  after a drop creates a new player slot."
- **Current State**: The README accurately describes the limitation but presents it
  as a known shortcoming rather than a gap.  Each reconnect generates a new
  `player_<UnixNano>` ID (game_server.go:443).  The old disconnected player record
  stays in `Players` and `TurnOrder` occupying a slot.  If GAP-03 is closed by
  removing the player from `TurnOrder`, they still consume a `Players` slot toward
  the MaxPlayers=6 cap.  A 6-player game with two brief disconnects would be full
  but have only 4 active players.
- **Impact**: Late joiners may be blocked from joining a game that appears full but
  has vacant investigator slots.  Returning players lose all progress (location,
  health, sanity, clues).
- **Closing the Gap**:
  1. On first connect, generate a reconnection token and send it via
     `connectionStatus`; client stores it in `localStorage`.
  2. On reconnect, client sends token; server finds matching `Player` record, sets
     `Connected: true`, restores `ActionsRemaining`, re-attaches the connection.
  3. Retain disconnected player records for 60 seconds; remove after grace period
     if no reconnect token arrives.
  (This is Phase 0 work noted in ROADMAP.md §"Reconnection tokens".)

---

## GAP-05 — `totalGamesPlayed` Counter Always Zero

- **Stated Goal**: `arkham_horror_games_played_total` Prometheus counter tracks
  total completed games (README — Prometheus Integration; `handleMetrics`
  game_server.go:760-761).
- **Current State**: `totalGamesPlayed` is initialised to `0` at construction
  (game_server.go:82) and is never incremented anywhere in the codebase.  The
  metric permanently reports `0`.
- **Impact**: Operators cannot track how many games have completed.  The monitoring
  dashboard misleads operators into thinking no games have been played.
- **Closing the Gap**: In `checkGameEndConditions` (game_server.go:394), add
  `atomic.AddInt64(&gs.totalGamesPlayed, 1)` when `GamePhase` transitions to
  `"ended"`.

---

## GAP-06 — System Alerts Never Exposed

- **Stated Goal**: "Error rates and system alerts" on the performance dashboard
  (README — Performance Dashboard).
- **Current State**: `getSystemAlerts()` (game_server.go:1470-1524) computes
  alerts for high memory, slow response time, high error rate, and critical doom
  level, but is never called from `/health`, `/metrics`, or any other endpoint.
- **Impact**: The system-alerts feature advertised in the README is silently absent.
  Operators have no way to receive automated alerts even when the system is in an
  alerted state.
- **Closing the Gap**: Add `"systemAlerts": gs.getSystemAlerts()` to the
  `healthData` map inside `handleHealthCheck` (game_server.go:646).

---

## GAP-07 — Broadcast Latency Not Exported in `/metrics`

- **Stated Goal**: "Real-time performance monitoring with comprehensive metrics" /
  sub-500 ms broadcast SLA (README — Performance Standards, Monitoring).
- **Current State**: `collectMessageThroughput` (game_server.go:993-1013)
  correctly reads the ring-buffer rolling average (`averageBroadcastLatencyMs`)
  and populates `BroadcastLatency`, but its return value is never used in
  `handleMetrics` or `handleHealthCheck`.  No `arkham_horror_broadcast_latency_ms`
  metric exists in the Prometheus output.
- **Impact**: The sub-500 ms broadcast SLA cannot be verified through the
  monitoring endpoint the README documents.  Operators must instrument externally.
- **Closing the Gap**: Inside `handleMetrics`, call
  `gs.collectMessageThroughput(uptime)` and add two metrics lines:
  ```
  # HELP arkham_horror_broadcast_latency_ms Rolling avg broadcast write latency ms
  arkham_horror_broadcast_latency_ms <value>
  ```

---

## GAP-08 — Client Reconnection Delay Misrepresented

- **Stated Goal**: "The client reconnects automatically every 5 seconds on
  disconnection." (README — Connection Behaviour)
- **Current State**: `client/game.js` lines 101-113 implement exponential backoff:
  initial delay 5 s, doubled each attempt (10 s, 20 s), capped at 30 s.  After two
  failed reconnect attempts the delay is already 20 s — four times longer than the
  documented 5 s.
- **Impact**: Developers learning from this codebase will observe behaviour that
  contradicts the documentation, undermining the pedagogical goal of the project.
- **Closing the Gap**: Update the README Connection Behaviour section to accurately
  describe the exponential backoff: "starting after 5 seconds, doubling each
  attempt, maximum 30 seconds."

---

## GAP-09 — Test Coverage Too Low to Validate Core Mechanics

- **Stated Goal**: "Are all 5 core mechanics fully functional with proper
  validation?" (Quality Checks in project specification).
- **Current State**: `go test -cover ./...` reports **10.4%** statement coverage.
  Only `GameStateValidator` is tested.  All five game mechanics, `advanceTurn`,
  `checkGameEndConditions`, `broadcastGameState`, and the WebSocket handler have
  zero tests.  The three critical-severity bugs documented in this audit
  (GAP-01, GAP-02, GAP-03) exist in completely untested code.
- **Impact**: Regressions in core gameplay cannot be detected automatically.  The
  project cannot reliably be demonstrated to meet its own Quality Check criteria
  without manual testing.
- **Closing the Gap**: Add the following test types:
  1. Table-driven unit tests for `processAction` covering all four action types,
     resource boundary conditions, invalid actions, and out-of-turn attempts.
  2. Unit tests for `advanceTurn` including disconnected-player skipping.
  3. Unit tests for `checkGameEndConditions` covering win, lose, and in-progress
     states at every player-count level (1–6).
  4. Integration tests using `net/http/httptest` + `gorilla/websocket` for
     connection lifecycle, multi-player turn rotation, and reconnection scenarios.
  Target ≥70% coverage; run with `go test -race -cover ./...`.

---

## GAP-10 — Ebitengine Client, Platform Entrypoints, and WASM/Mobile Targets

- **Stated Goal**: Phased replacement of the HTML/JS client with a Go/Ebitengine
  client supporting desktop (Phase 2), WASM (Phase 3), and mobile (Phase 4)
  (README — Build Targets, ROADMAP).
- **Current State**: `go.mod` lists only `github.com/gorilla/websocket v1.5.3`.
  No `github.com/hajimehoshi/ebiten/v2` dependency exists.  No `cmd/desktop/`,
  `cmd/web/`, or `cmd/mobile/` directories exist.  No `client/ebiten/` package
  exists.
- **Impact**: The planned multi-platform distribution path is entirely absent.
  All four Build Target rows in the README table are in "Planned" status with no
  implementation started.
- **Closing the Gap**: This is intentional phased work; ROADMAP phases 1–5 describe
  the implementation sequence.  Prerequisite: `go get github.com/hajimehoshi/ebiten/v2@v2.7+`.
  Begin with Phase 1 (client package skeleton) before adding platform entrypoints.

---

## GAP-11 — `processAction` Monolith Impedes Mechanic Testing

- **Stated Goal**: "Implement proper Go-style error handling … Use goroutines and
  channels for concurrent WebSocket connection management" / clear separation of
  concerns (project specification — Go Coding Standards).
- **Current State**: `processAction` (game_server.go:176-363) is 187 lines with
  cyclomatic complexity 32.2.  It handles input validation, four action
  implementations, doom updates, resource validation, end-condition checking, turn
  advancement, and three broadcast calls under a single mutex lock.  No individual
  action mechanic can be unit-tested in isolation.
- **Impact**: Adding a new action type or fixing a dice calculation requires
  modifying this single function, risking regressions in all other actions.  The
  complexity is a primary reason test coverage remains at 10.4%.
- **Closing the Gap**: Extract each action implementation into a method
  (`gs.performMove`, `gs.performGather`, `gs.performInvestigate`,
  `gs.performCastWard`) each returning `(doomIncrease int, diceResult *DiceResultMessage, err error)`.
  `processAction` becomes a dispatch function of ≤40 lines.  Each extracted method
  can be unit-tested directly without WebSocket infrastructure.
