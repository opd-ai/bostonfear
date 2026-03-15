# AUDIT — 2026-03-15

## Project Goals

BostonFear is a cooperative multiplayer Arkham Horror implementation for 1–6 concurrent players. Its stated goals are:

1. **5 core game mechanics**: Location System, Resource Tracking, Action System, Doom Counter, Dice Resolution — all integrated with cross-mechanic dependencies.
2. **Multiplayer WebSocket server**: Go server using `net.Conn` / `net.Listener` / `net.Addr` interfaces, goroutine-based concurrency, channel-based broadcast, join-in-progress support.
3. **JavaScript legacy client**: HTML5 Canvas (800×600), auto-reconnection with exponential backoff, turn/action UI.
4. **Ebitengine client** (called "planned" in README): Desktop, WASM, and Mobile build targets with layered renderer and Kage shaders.
5. **Observability**: `/health`, `/metrics` (Prometheus), and `/dashboard` endpoints with real-time performance data.
6. **Performance targets**: sub-500 ms state sync, 30-second connection-drop timeout, stable operation with 6 players over 15 minutes.
7. **AH3e compliance**: Full Arkham Horror Third Edition rule compliance is a stated roadmap goal (ROADMAP Phase 6); current compliance is documented in RULES.md.

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Location System — 4 neighborhoods, adjacency enforcement | ✅ Achieved | `cmd/server/game_constants.go:35-42`, `game_server.go:143-155` |
| Resource Tracking — Health/Sanity/Clues with bounds validation | ✅ Achieved | `game_server.go:106-128`, `game_types.go:17-22` |
| Action System — 2 actions/turn, 4 action types | ✅ Achieved | `game_server.go:177-213`, `game_constants.go:21-30` |
| Doom Counter — 0-12, increments on tentacles | ✅ Achieved | `game_server.go:328-401`, `game_server.go:192` |
| Dice Resolution — 3-sided, configurable threshold | ✅ Achieved | `game_server.go:157-175` |
| Cross-mechanic integration (dice→doom, action→resources, adjacency check) | ✅ Achieved | `game_server.go:247-315` |
| Turn order / 2-action-per-turn progression | ✅ Achieved | `game_server.go:419-441` |
| Win/Lose conditions (doom≥12 / clues≥4×players) | ✅ Achieved | `game_server.go:443-472` |
| 1–6 players, join-in-progress | ✅ Achieved | `game_server.go:508-538` |
| `net.Conn` / `net.Listener` / `net.Addr` interface usage | ✅ Achieved | `connection_wrapper.go`, `game_utils.go:18-36` |
| Goroutine + channel concurrency model | ✅ Achieved | `game_server.go:1163-1240` |
| All 5 JSON message types (gameState, playerAction, gameUpdate, connectionStatus, diceResult) | ✅ Achieved | `game_types.go`, `game_server.go` |
| Auto-reconnect with exponential backoff (JS client) | ✅ Achieved | `client/game.js:101-116` |
| Canvas 800×600 minimum | ✅ Achieved | `client/index.html:295` |
| `/health` endpoint with JSON response | ✅ Achieved | `game_server.go:669-737` |
| `/metrics` Prometheus endpoint | ✅ Achieved | `game_server.go:739-836` |
| `/dashboard` performance dashboard | ✅ Achieved | `game_server.go:738-741` |
| Error recovery / game state validation | ✅ Achieved | `error_recovery.go` |
| Doom increment on turn timeout (30 s read deadline) | ✅ Achieved | `game_server.go:561-576` |
| Desktop Ebitengine client (README says "Planned Phase 2") | ⚠️ Partial | Code + build succeed; atlas uses placeholder tiles; README mislabels as unimplemented |
| WASM Ebitengine client (README says "Planned Phase 3") | ⚠️ Partial | WASM build succeeds; `client/wasm/index.html` exists; atlas placeholder tiles |
| Mobile Ebitengine client (README says "Planned Phase 4") | ⚠️ Partial | Binding scaffolding exists (`cmd/mobile/mobile.go`); not verified on device |
| Kage shaders (fog-of-war, doom vignette, glow) | ⚠️ Partial | Source files exist and compile; not exercised by any test |
| `ConnectionWrapper.LocalAddr()` returns correct local address | ❌ Missing | `connection_wrapper.go:53-55` — returns remote addr |
| AH3e full compliance (8 action types, Mythos Phase, Encounters, etc.) | ❌ Missing | RULES.md compliance table; documented in GAPS.md |
| Session persistence / reconnection token | ❌ Missing | README §Connection Behaviour documents limitation |
| Investigator defeat / "lost in time and space" | ❌ Missing | RULES.md compliance table |

---

## Findings

### CRITICAL

- [x] **`ConnectionWrapper.LocalAddr()` returns the remote address** — `cmd/server/connection_wrapper.go:53-55` — Both `LocalAddr()` and `RemoteAddr()` return the single `c.addr` field, which is set from `wsConn.RemoteAddr()` at construction time. Any caller relying on `LocalAddr()` to obtain the server's own listening address will silently receive the client's remote address instead. The net.Conn contract is violated. Currently the game server never calls `LocalAddr()` on its own wrappers, so the bug does not corrupt gameplay, but it is a spec violation of the `net.Conn` interface and a latent hazard for downstream code or tests that build on this abstraction.  
  **Remediation:** Capture the local address at construction and return it from `LocalAddr()`:  
  ```go
  // In NewConnectionWrapper, add a localAddr parameter:
  func NewConnectionWrapper(ws *websocket.Conn, localAddr, remoteAddr net.Addr) *ConnectionWrapper {
      return &ConnectionWrapper{ws: ws, localAddr: localAddr, remoteAddr: remoteAddr}
  }
  func (c *ConnectionWrapper) LocalAddr() net.Addr  { return c.localAddr }
  func (c *ConnectionWrapper) RemoteAddr() net.Addr { return c.remoteAddr }
  ```
  Then in `handleWebSocket`, pass `ws.NetConn().LocalAddr()` as the first address argument.  
  **Validation:** `go test -race ./cmd/server/...` + add a `TestConnectionWrapper_LocalRemoteAddrDistinct` test that asserts `LocalAddr() != RemoteAddr()`.

---

### HIGH

- [x] **README mislabels substantially-implemented Ebitengine clients as "Planned"** — `README.md` (Build Targets table) — The desktop (`cmd/desktop/`), WASM (`cmd/web/`), and mobile (`cmd/mobile/`) entrypoints are fully compilable and wired to a functional `client/ebiten` package containing board rendering, WebSocket client with exponential backoff, layered renderer, and Kage shaders. `go build ./cmd/desktop/...` and `GOOS=js GOARCH=wasm go build ./cmd/web` both succeed. Marking them "Planned (ROADMAP Phase 2/3/4)" creates misleading expectations for contributors and users evaluating the project.  
  **Remediation:** Update the Build Targets table in `README.md` to reflect actual status: `Active (alpha — placeholder sprites)` for Desktop/WASM, and update ROADMAP.md to close the completed phases.  
  **Validation:** `go build ./cmd/desktop/... && GOOS=js GOARCH=wasm go build ./cmd/web` must both succeed after any README edits.

- [ ] **Ebitengine sprite atlas uses placeholder solid-colour tiles** — `client/ebiten/render/atlas.go:37,73-99` — `generateAtlas()` fills every sprite slot with a solid-colour rectangle at compile time. The README/CLIENT_SPEC.md promises real board artwork and token sprites. The `Atlas` API is complete and replacing `generateAtlas()` is the only remaining step, but until real assets are integrated the visual output does not match the stated UI/UX requirements in `CLIENT_SPEC.md`.  
  **Remediation:** Embed real sprite sheet PNG(s) under `client/ebiten/render/assets/` via `//go:embed`, replace `generateAtlas()` with a function that slices the sheet according to the declared `spriteRect` layout in each `SpriteID`, and add a build tag or fallback path that restores the colour-rectangle atlas when no assets are present (for CI).  
  **Validation:** `go build ./client/ebiten/...` must succeed and visual inspection of a running `./cmd/desktop` must show board artwork.

- [x] **JS reconnection gives up permanently after 10 attempts (~5 min)** — `client/game.js:11,103-115` — `maxReconnectAttempts` is hard-coded to 10. After that the client logs "Max reconnection attempts reached" and never retries again. The README promises "automatic reconnection with exponential backoff (5 s initial, doubles, 30 s cap)" with no mention of a finite limit. A client that drops due to a brief server restart and reconnects after >5 minutes is stranded.  
  **Remediation:** Either remove the upper bound (set `maxReconnectAttempts = Infinity`) or increase it substantially (e.g. 50), and document the limit in README §Connection Behaviour.  
  **Validation:** Load `client/index.html`, stop the server for 6 minutes, restart it, and confirm the client reconnects.

- [x] **Zero exported Go interfaces in project (0 total)** — go-stats-generator: `total_interfaces: 0` — The game server's core logic (`GameServer`, `GameStateValidator`) is not gated behind any locally-defined interface. This means unit tests that need to substitute a fake server or fake validator must instantiate the concrete structs and all their transitive dependencies (including goroutines and channels). The stated goal of "interface-based design for enhanced testability" is partially unmet at the application layer: `net.Conn` is used for transport but there is no `GameLogic`, `StateStore`, or `Broadcaster` interface to isolate the game engine from the network layer in tests.  
  **Remediation:** Extract at minimum a `Broadcaster` interface (`Broadcast([]byte)`) and a `StateValidator` interface (`ValidateGameState(*GameState) []ValidationError`) in `cmd/server/`, update `GameServer` to depend on the interface, and inject the concrete types in `main.go`. This allows tests to pass a no-op broadcaster.  
  **Validation:** `go vet ./cmd/server/...` must pass; add one test that injects a mock broadcaster.

---

### MEDIUM

- [x] **"30-second reconnection timeout" in README describes a read deadline, not a reconnection window** — `README.md` §Performance Standards / `game_server.go:561-576` — The README states "Automatic handling of connection drops with 30-second timeout" implying the server holds the player's slot open for 30 seconds while awaiting reconnection. In reality the 30-second timer is a WebSocket read deadline: if no message arrives within 30 seconds the server increments the doom counter and immediately terminates the connection, removing the player from the game. A reconnecting client always creates a new player slot. These are materially different behaviours.  
  **Remediation:** Update README §Performance Standards and §Connection Behaviour to accurately describe the 30-second idle/inactivity timeout, and clarify that reconnection creates a new player slot (the existing note in §Connection Behaviour is correct but isolated — cross-reference it from §Performance Standards).  
  **Validation:** No code change required; review updated README for consistency.

- [x] **Doom increments on tentacles during Gather even when Gather succeeds** — `game_server.go:326-353` — The `performGather` function increments doom for each tentacle regardless of whether the roll produced successes. The README states "Doom Counter increments on **failed** dice rolls" but the spec also separately states "Tentacle (🐙): Increases Doom counter by 1" (which is unconditional). These two clauses contradict each other. The current implementation follows the unconditional per-tentacle rule, which is arguably the more correct reading, but the phrase "failed dice rolls" in the Doom Counter section creates confusion.  
  **Remediation:** Clarify README §Game Rules: either change "increments on failed dice rolls" to "increments for each Tentacle result" to match the Dice Mechanics section and the implementation, or make doom on Gather conditional on failure (`if !diceResult.Success { doom += tentacles }`).  
  **Validation:** `go test -race -run TestProcessAction ./cmd/server/...` must pass after any implementation change.

- [x] **`handleMetrics` and `handleHealthCheck` hold `gs.mutex.RLock()` for entire response serialization** — `game_server.go:669,739` — Both handlers acquire `gs.mutex.RLock()` and hold it across `json.Encode()` / string building. Under load this blocks all game-state writes (player connects, actions) for the duration of the HTTP response. While sub-critical for a 1–6 player game, it directly contradicts the sub-500 ms state-sync target if an external monitoring system polls `/metrics` or `/health` at high frequency.  
  **Remediation:** Snapshot the required fields under a short `RLock`, release the lock, then serialize the snapshot outside the lock.  
  **Validation:** `go test -race ./cmd/server/...` should detect no data races; benchmark with `go test -bench=BenchmarkHandleHealthCheck ./cmd/server/...` before and after.

- [ ] **Win condition maximum-clue cap is reachable for 5–6 players** — `game_server.go:443-472`, `game_server.go:106-128` — Each player's `Clues` field is capped at 5 (max). For 6 players the win condition requires 24 total clues (6 × 4), but the maximum achievable is 30 (6 × 5). For 5 players: 20 needed, 25 achievable. The win is possible, but if only some players are active, the total pool shrinks. For example, 6 players connected but 2 have been eliminated (health=0 but no defeat mechanic), their clues still count — however the game provides no investigator-defeat handling, so this scenario cannot currently occur. No immediate functional bug, but worth noting as a future risk when investigator defeat is implemented.  
  **Remediation:** When investigator defeat is implemented, re-evaluate whether a defeated investigator's clues should remain in the win-condition pool or be removed.  
  **Validation:** `go test -race -run TestCheckGameEndConditions ./cmd/server/...`.

---

### LOW

- [x] **`ConnectionWrapper.LocalAddr()` and `RemoteAddr()` share the same backing field name** — `cmd/server/connection_wrapper.go:13-15` — The struct has a single `addr net.Addr` field, making the distinction between local and remote invisible at the type level and easy to re-introduce after any future refactor. (Also tracked as a CRITICAL finding above for its net.Conn contract violation.)  
  **Remediation:** Rename field to `remoteAddr`, add `localAddr net.Addr`, update both methods.  
  **Validation:** `go vet ./cmd/server/...`.

- [x] **`Renderer` type in `client/ebiten/render` package stutters** — `client/ebiten/render/layers.go:42` — The exported type is `render.Renderer`; callers must write `render.Renderer`. go-stats-generator flags this as a package-name stutter. The idiomatic Go name would be `render.R` or the package itself could be renamed `layers` with the type being `layers.Renderer`.  
  **Remediation:** Rename `Renderer` to `R` or rename the package to `ui` or `layers`; update all references in `game.go`.  
  **Validation:** `go build ./client/ebiten/...`.

- [x] **`cmd/mobile/mobile.go` filename stutters with package name** — go-stats-generator naming analysis — The file `cmd/mobile/mobile.go` in package `mobile` is considered stuttering by the analyzer. Convention for command packages is to use `main.go` or a descriptive non-stutter name.  
  **Remediation:** Rename to `cmd/mobile/main.go` (or `binding.go`).  
  **Validation:** `go build ./cmd/mobile/...`.

- [ ] **46 misplaced functions flagged by go-stats-generator** — go-stats-generator placement analysis — Many functions in `game_utils.go`, `game_types.go`, and `connection_wrapper.go` are suggested to be co-located with `game_server.go`. This is a file-organization concern and does not affect correctness.  
  **Remediation:** Apply the top-priority moves from `go-stats-generator analyze . --format json` output, particularly moving `locationAdjacency` and game constants to `game_server.go` or a dedicated `game_map.go`.  
  **Validation:** `go build ./cmd/server/... && go test ./cmd/server/...`.

- [x] **`handleMetrics` function is 93 lines** — go-stats-generator: "Longest Function: handleMetrics (93 lines)" — Exceeds the 50-line threshold flagged by the analyzer. The function is primarily string concatenation, which is low-risk, but it could be split into `buildGameMetrics()`, `buildConnectionMetrics()`, and `buildMemoryMetrics()` helpers.  
  **Remediation:** Extract each metric group into a `formatXxxMetrics() []string` helper called from `handleMetrics`.  
  **Validation:** `go build ./cmd/server/...`.

---

## Metrics Snapshot

| Metric | Value | Source |
|--------|-------|--------|
| Total functions (excl. tests) | 44 | go-stats-generator |
| Total methods (excl. tests) | 108 | go-stats-generator |
| Total structs | 50 | go-stats-generator |
| Total exported interfaces | 0 | go-stats-generator |
| Total packages | 4 | go-stats-generator |
| Total LOC (excl. tests) | 1,871 | go-stats-generator |
| Average function length | 14.1 lines | go-stats-generator |
| Longest function | `handleMetrics` — 93 lines | go-stats-generator |
| Functions > 50 lines | 4 (2.6%) | go-stats-generator |
| Average cyclomatic complexity | 4.2 | go-stats-generator |
| High-complexity functions (>10 cyclomatic) | 0 | go-stats-generator |
| Top-complexity function | `broadcastConnectionQuality` — 11.4 overall | go-stats-generator |
| Documentation coverage (overall) | 96% | go-stats-generator |
| Duplication ratio | 0% | go-stats-generator |
| `go test -race ./cmd/server/...` | PASS (1.246 s) | verified |
| `go vet ./...` | Clean (no warnings) | verified |
| `go build ./cmd/desktop/...` | PASS | verified |
| `GOOS=js GOARCH=wasm go build ./cmd/web` | PASS | verified |
| Test results | All PASS; 10 tests SKIP (known AH3e gaps documented in RULES.md) | verified |
