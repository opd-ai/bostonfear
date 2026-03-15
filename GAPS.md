# Implementation Gaps — supersedes prior gap analysis dated 2026-03-15

> This document annotates all original gaps with their current status and adds new
> gaps introduced by the Ebitengine migration goals defined in `ROADMAP.md`.

---

## Original Gaps

---

## Missing `gameUpdate` Protocol Message

**Status: OPEN — must close before Ebitengine migration begins (Phase 0)**

- **Stated Goal**: The JSON protocol must include five message types: `gameState`, `playerAction`, `gameUpdate`, `connectionStatus`, `diceResult` (README, `.github/copilot-instructions.md:61`)
- **Current State**: Only four message types are implemented. `gameUpdate` is never emitted by the server (`cmd/server/game_server.go`) and has no handler in the client (`client/game.js`). After every player action the server sends a full `gameState` snapshot and optionally a `diceResult`, but no lightweight event/delta message exists.
- **Impact**: Clients cannot distinguish a game-event notification from a full state sync. Downstream tooling (spectators, replays, analytics) that expects `gameUpdate` events cannot integrate. The protocol contract documented in the README is incomplete.
- **Closing the Gap**: Add a `gameUpdate` broadcast in `processAction` (after the action is applied, before the full `gameState` broadcast) that describes the event just occurred — e.g., `{"type":"gameUpdate","event":"investigate","playerId":"...","result":"fail","doomDelta":1}`. Add a `case 'gameUpdate':` handler in `game.js` to display event notifications. Update the README example to show a real `gameUpdate` payload.

---

## Broken Setup Instructions (Wrong Entry-Point Path)

**Status: CLOSED**

- **Stated Goal**: A 3-step setup — `go mod tidy`, `cd server && go run main.go`, open browser — lets anyone run the game on a clean environment (README:37–42, README:104)
- **Current State**: The actual server entry point is `cmd/server/main.go`. The `server/` directory does not contain Go source files; it contains only the compiled binary artifact. Running `cd server && go run main.go` produces: `stat main.go: no such file or directory`.
- **Impact**: Any developer following the README cannot start the server. This blocks the primary quick-start path for the project's target audience (intermediate developers learning WebSocket architecture).
- **Closing the Gap**: Update `README.md` Step 2 to `cd cmd/server && go run .`. Update the project structure table (README:104) to list `cmd/server/main.go`. Optionally add a `Makefile` target (`make run`) that wraps the correct command.
- **Resolution**: README now shows `cd cmd/server && go run .` as the correct setup command.

---

## Reconnection Does Not Restore Session State

**Status: OPEN — must close before Ebitengine migration begins (Phase 0)**

- **Stated Goal**: "Handle connection drops with 30-second reconnection timeout" / "Automatic reconnection handling" (README Performance Standards, Multiplayer Features sections)
- **Current State**: The server's 30-second read deadline (game_server.go:388–392, 460–462) triggers a doom increment and breaks the read loop when no message arrives for 30 seconds. The client (game.js:101–115) retries the WebSocket connection every 5 seconds. However, a reconnecting player receives a brand-new `player_<UnixNano>` ID (game_server.go:401) and is added as a fresh investigator — their former character's location, health, sanity, and clues are permanently lost. The old player record stays in `gameState.Players` with `Connected: false` and `ActionsRemaining: 0`, consuming one of the four player slots indefinitely.
- **Impact**: A brief network hiccup (> 30 s) permanently removes a player's investigator from cooperative gameplay. The orphaned disconnected-player slot prevents new players from joining until the game ends (once the 6-player limit is reached). This is the most significant functional gap between the README's UX claims and actual behavior.
- **Closing the Gap**:
  1. On first connect, generate and send a reconnection token to the client (stored in `localStorage`).
  2. On reconnect, the client sends the token; the server finds the matching player record and re-attaches the existing `Player` struct, setting `Connected: true` and restoring `ActionsRemaining`.
  3. Extend the orphaned-player grace period: instead of deleting the player immediately on disconnect, retain the record for 60 seconds while marking `Connected: false`, then remove it if no reconnect token arrives.
  4. The ROADMAP (Phase 1.1) lists Redis-based session storage as the production approach; a simpler in-memory token map suffices for the current single-server setup.

---

## Win Condition Threshold Undisclosed

**Status: CLOSED**

- **Stated Goal**: "Investigators must work cooperatively to gather clues and cast protective wards before the doom counter reaches 12. **Win**: Achieve sufficient collective clues (cooperative victory)" (README Game Rules)
- **Current State**: The code computes `requiredClues = playerCount * 4` (game_server.go:371). The win threshold is 8 clues for 2 players, 12 for 3, and 16 for 4. This formula is not documented anywhere for players.
- **Impact**: Players have no way to know the victory condition without reading the source code, making the game unplayable as a standalone experience.
- **Closing the Gap**: Add a concrete sentence to the README Win/Lose Conditions section: "**Win**: Collectively gather 4 clues per investigator (8 for 2 players, 12 for 3, 16 for 4)." No code changes required. Optionally expose `requiredClues` in the `gameState` JSON so the client can render a progress bar.
- **Resolution**: README now documents the 4-clues-per-investigator formula.

---

## Error Rate Metric Is Permanently Zero

**Status: OPEN — must close before Ebitengine migration begins (Phase 0)**

- **Stated Goal**: "Error Recovery: Automated game state validation and corruption detection" / Prometheus metric `arkham_horror_error_rate_percent` (README Monitoring section, game_server.go:700)
- **Current State**: `calculateErrorRate()` (game_server.go:1110–1113) returns a hardcoded `0.0`. The `/metrics` endpoint exports `arkham_horror_error_rate_percent 0.00` unconditionally. The `/health` alert check (`errorRate > 5`) can never trigger.
- **Impact**: Operators monitoring the Prometheus endpoint receive a false signal of zero errors even when the server is logging repeated action failures or state corruption events. The monitoring dashboard's "Error rates and system alerts" feature is non-functional for error rate.
- **Closing the Gap**: Track errors with an atomic counter incremented at each error log site (WebSocket upgrade failures, unmarshal errors, action validation failures). Divide by `totalMessagesRecv` to produce a percentage. Expose the real value through `calculateErrorRate()`. Validate with: `curl http://localhost:8080/metrics | grep error_rate` after sending several malformed WebSocket messages.

---

## Message Latency Metrics Are Placeholder Zeros

**Status: OPEN — must close before Ebitengine migration begins (Phase 0)**

- **Stated Goal**: "Real-time performance monitoring with comprehensive metrics" / Performance Dashboard shows "Server uptime and connection analytics" (README Monitoring section)
- **Current State**: `AverageLatency` and `BroadcastLatency` in `MessageThroughputMetrics` (game_server.go:920–926) are hardcoded to `0` with a TODO comment acknowledging the gap. These fields are computed in `collectMessageThroughput` but the function result is never used in the `/metrics` or `/health` handlers.
- **Impact**: The dashboard's latency numbers are permanently `0`, preventing operators from detecting slow broadcast paths. The sub-500 ms sync SLA cannot be verified through the provided metrics.
- **Closing the Gap**: Record `time.Now()` before each `broadcastCh <- data` send and after it is consumed in `broadcastHandler`. Store the delta in a ring buffer; expose the rolling average as `BroadcastLatency`. Wire `collectMessageThroughput` into the `/metrics` handler output. Validate that `arkham_horror_broadcast_latency_ms` is non-zero under active play.

---

## `handleDashboard` Serves from Wrong Relative Path

**Status: OPEN — must close before Ebitengine migration begins (Phase 0)**

- **Stated Goal**: "Real-time Dashboard: Live performance metrics at `/dashboard`" (README Features)
- **Current State**: `handleDashboard` (game_server.go:627) calls `http.ServeFile(w, r, "./client/dashboard.html")`. When the server is started from `cmd/server/` (the correct working directory per the corrected setup instructions), this path resolves to `cmd/server/client/dashboard.html`, which does not exist. The static file server at `/` correctly uses `"../client/"` (utils.go:35).
- **Impact**: `GET /dashboard` returns a 404 Not Found error. The performance monitoring dashboard — advertised as a key feature — is inaccessible via the standard setup.
- **Closing the Gap**: Change game_server.go:627 from `"./client/dashboard.html"` to `"../client/dashboard.html"`. Alternatively, define a single `clientDir` constant and use it everywhere. Validate with: `curl -v http://localhost:8080/dashboard` returning HTTP 200 with the dashboard HTML.

---

## `ConnectionWrapper` Deadline Methods Are No-Ops

**Status: OPEN — must close before Ebitengine migration begins (Phase 0)**

- **Stated Goal**: "Interface-based Design: Uses `net.Conn`, `net.Listener`, and `net.Addr` interfaces" / "Handle connection drops with 30-second reconnection timeout" (README Technical Implementation)
- **Current State**: `ConnectionWrapper.SetDeadline`, `SetReadDeadline`, and `SetWriteDeadline` (connection_wrapper.go:56–68) return `nil` without delegating to the underlying `*websocket.Conn`. `handleConnection` calls `conn.SetReadDeadline(...)` on the wrapper (lines 389, 461) believing it sets a 30-second I/O timeout, but the wrapper silently discards the call. The actual timeout is set by a duplicate direct call to `wsConn.SetReadDeadline` — which means the `net.Conn` abstraction layer is bypassed for a core requirement.
- **Impact**: The `net.Conn` interface contract is violated: callers cannot rely on `SetReadDeadline` through the abstraction. If the direct `wsConn` call is ever removed (e.g., during refactoring), the 30-second timeout silently disappears, changing game behavior without any compile-time warning.
- **Closing the Gap**: Implement the three deadline methods in `ConnectionWrapper` by delegating to the underlying `*websocket.Conn`:
  ```go
  func (c *ConnectionWrapper) SetDeadline(t time.Time) error {
      return c.ws.SetReadDeadline(t) // WebSocket has no combined deadline
  }
  func (c *ConnectionWrapper) SetReadDeadline(t time.Time) error {
      return c.ws.SetReadDeadline(t)
  }
  func (c *ConnectionWrapper) SetWriteDeadline(t time.Time) error {
      return c.ws.SetWriteDeadline(t)
  }
  ```
  Then remove the direct `wsConn.SetReadDeadline` calls in `handleConnection`. Validate by confirming doom increments after 30 s of client silence.

---

## New Gaps: Ebitengine Migration

> The following gaps are introduced by the new project goals defined in `ROADMAP.md`.
> None of these exist in the current codebase because the migration has not yet begun.

---

### 1. No Ebitengine Dependency

- **Stated Goal**: Replace the HTML/JS canvas client with a Go/Ebitengine client (ROADMAP Phase 1).
- **Current State**: `go.mod` lists only `github.com/gorilla/websocket v1.5.3`. There is no `github.com/hajimehoshi/ebiten/v2` entry. WASM and mobile build tags are absent.
- **Impact**: No Ebitengine client code can be compiled until the dependency is added.
- **Closing the Gap**: Run `go get github.com/hajimehoshi/ebiten/v2@latest && go mod tidy`.

---

### 2. No Platform Entrypoints

- **Stated Goal**: Desktop (`cmd/desktop/`), WASM (`cmd/web/`), and mobile (`cmd/mobile/`) build targets (ROADMAP Phases 2–4).
- **Current State**: Only `cmd/server/` exists. No `cmd/desktop/`, `cmd/web/`, or `cmd/mobile/` directories.
- **Impact**: No desktop, WASM, or mobile client can be built until entrypoints are created.
- **Closing the Gap**: Create `cmd/desktop/main.go`, `cmd/web/main.go`, and `cmd/mobile/mobile.go` per ROADMAP specifications.

---

### 3. Client Is HTML/JS Only

- **Stated Goal**: Go/Ebitengine client package at `client/ebiten/` with sprite/layer rendering (ROADMAP Phases 1, 5).
- **Current State**: `client/game.js` uses browser Canvas API (`getContext('2d')`, `fillRect`, `fillText`). No Go rendering layer exists. No `client/ebiten/` directory.
- **Impact**: The game can only be played in a web browser via the legacy JS client. No native desktop, WASM, or mobile experience.
- **Closing the Gap**: Create `client/ebiten/` package implementing `ebiten.Game` interface with `game.go`, `net.go`, `state.go`, `input.go`, and the `render/` subsystem.

---

### 4. AH3e Rules Compliance Unverified

- **Stated Goal**: 100% AH3e core rulebook compliance with automated test suite (ROADMAP Phase 6).
- **Current State**: No test suite exists that validates engine behaviour against `RULES.md` specifications. The engine implements only 4 of 8 AH3e action types (Move, Gather, Investigate, Ward — missing Focus, Trade, Component Action, Attack/Evade). The Mythos Phase sequence, encounter resolution, act/agenda deck progression, and modular difficulty settings are not implemented.
- **Impact**: The game diverges from AH3e rules in multiple areas without automated detection. Rule deviations are undocumented.
- **Closing the Gap**: Implement the rules-compliance test suite (`cmd/server/rules_test.go`) covering all 10 rule systems listed in ROADMAP Phase 6. Implement missing engine mechanics to pass all tests.

---

### 5. Mobile Input Unimplemented

- **Stated Goal**: Touch input handling within the Ebitengine render loop for iOS and Android (ROADMAP Phase 4).
- **Current State**: No touch event handling exists in any current client code. `client/game.js` handles only mouse click events. No `ebiten.TouchID` mapping exists.
- **Impact**: The game cannot be played on mobile devices.
- **Closing the Gap**: Implement touch input mapping in `client/ebiten/input.go` translating `ebiten.TouchID` events to the same action vocabulary used by keyboard/mouse input.

---

### 6. Resolution/Aspect Ratio Policy Undefined

- **Stated Goal**: Logical resolution 1280×720 (16:9) scaling to all platforms; safe-area insets on mobile (ROADMAP Phase 5).
- **Current State**: The legacy JS client renders at a fixed 800×600 canvas. No constants or configuration specify target resolutions for desktop, web, or mobile viewports. No aspect ratio policy exists.
- **Impact**: Multi-platform visual consistency cannot be achieved without defined resolution targets.
- **Closing the Gap**: Define resolution constants in `client/ebiten/game.go` (`Layout` returns 1280×720). Implement `ebiten.DeviceScaleFactor` handling for mobile safe-area insets. Document target resolutions in the rendering subsystem.
