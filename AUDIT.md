# IMPLEMENTATION GAP AUDIT - 2026-05-08

## Project Architecture Overview

BostonFear is a rules-only Arkham Horror engine with three active runtime layers and a larger migration scaffold around them.

- `serverengine` owns the active Arkham runtime: turn processing, action resolution, mythos progression, connection lifecycle, metrics, and health.
- `transport/ws` owns HTTP route registration, WebSocket upgrade, and the `net.Listener`-based server edge.
- `monitoring`, `monitoringdata`, and `protocol` provide HTTP monitoring endpoints, DTOs, and the shared JSON wire contract.
- `client/index.html` + `client/game.js` are the active browser client.
- `client/ebiten`, `client/ebiten/app`, and `client/ebiten/render` are the alpha Go/Ebitengine client for desktop, WASM, and mobile binding.
- `serverengine/arkhamhorror` is the intended long-term Arkham-specific module boundary, but most Arkham logic still lives in the root `serverengine` package.
- `serverengine/common/*`, `serverengine/eldersign/*`, `serverengine/eldritchhorror/*`, and `serverengine/finalhour/*` are mostly migration scaffolds and placeholder module families.

### Stated Goals Evaluated

- The repository README positions the project as an Arkham Horror multiplayer engine with an active browser client and an active alpha Ebitengine client.
- `docs/RULES.md` claims 13/13 core rule systems are fully implemented.
- `docs/CLIENT_SPEC.md` defines a richer four-scene Ebitengine client contract than the current code actually delivers.
- `serverengine/arkhamhorror/README.md` states that Arkham-specific rules should move out of shared packages and into the Arkham module tree.

### Phase 1 Online Research

Brief GitHub review found no open issues, no open projects, no milestones, and no open pull requests. The only public planning signal is in-repository documentation (`README.md`, `ROADMAP.md`, `docs/CLIENT_SPEC.md`, `docs/RULES.md`). There is no external tracker to explain away the gaps below as already tracked public backlog items.

## Baseline Evidence

- Prerequisite satisfied: `go-stats-generator` installed and used.
- `go build ./...`: clean.
- `go vet ./...`: clean.
- `go-stats-generator analyze . --skip-tests`:
  - 2,978 lines of Go code
  - 67 files processed
  - 69 functions, 195 methods, 62 structs, 7 interfaces
  - 67.0% overall documentation coverage
  - 0 TODO/FIXME/HACK/XXX annotations detected
  - 4 unreferenced-function candidates reported by the tool; only high-confidence items survived false-positive review
- Additional validation performed for non-Go surface:
  - `node --check client/game.js` fails with `SyntaxError: Unexpected identifier 'resultDiv'`

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 2 | 0 | 0 | 0 | 2 |
| Dead Code | 1 | 0 | 0 | 1 | 0 |
| Partially Wired | 3 | 1 | 1 | 1 | 0 |
| Interface Gaps | 1 | 0 | 0 | 0 | 1 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package

Coverage below is estimated implementation completeness for the package's intended role, not test coverage.

| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|--------------------|-------------|-------|------|----------|
| `client/ebiten` | 17 | 17 | 0 | 0 | 90% |
| `client/ebiten/app` | 15 | 15 | 0 | 0 | 75% |
| `client/ebiten/render` | 10 | 10 | 0 | 0 | 100% |
| `monitoring` | 4 | 4 | 0 | 0 | 100% |
| `serverengine` | 18 | 18 | 0 | 0 | 85% |
| `serverengine/arkhamhorror` | 4 | 4 | 0 | 0 | 100% |
| `serverengine/common/runtime` | 17 | 17 | 0 | 0 | 100% |
| `serverengine/eldersign` | 4 | 3 | 1 | 0 | 75% |
| `serverengine/eldritchhorror` | 4 | 3 | 1 | 0 | 75% |
| `serverengine/finalhour` | 4 | 3 | 1 | 0 | 75% |
| `transport/ws` | 11 | 11 | 0 | 0 | 100% |
| `serverengine` doc-only scaffold packages (25 total) | 0 | 0 | 25 | 0 | 0% |

## Findings

### CRITICAL

- [ ] **Active browser client cannot parse** - `client/game.js:397` - a second copy of the `displayDiceResult` body is embedded directly in the class body after `displayGameUpdate`, and `node --check client/game.js` fails with `SyntaxError: Unexpected identifier 'resultDiv'`. This blocks the README's active HTML/JS browser client path (`client/index.html`) from loading at all. **Remediation:** remove the duplicated block at `client/game.js:397-433`, add a syntax smoke check for browser assets, and validate with `node --check client/game.js` plus a manual load of `/` from the Go server.

### HIGH

- [ ] **Pregame actions are unreachable in live sessions** - `serverengine/connection.go:110`, `serverengine/actions.go:388`, `serverengine/actions.go:405`, `serverengine/connection_test.go:66`, `docs/RULES.md:28`, `docs/RULES.md:32` - the first connected player starts the game immediately because `MinPlayers = 1`, which flips `GamePhase` from `waiting` to `playing`. At the same time, both `performSelectInvestigator` and `performSetDifficulty` reject any call unless the phase is still `waiting`. The rules docs mark both systems complete, but the real connection path makes them unreachable. **Remediation:** keep the game in `waiting` until explicit readiness/start criteria are met, or allow these actions before first-turn assignment in `playing`; add an integration test that performs `connect -> selectInvestigator` and `connect -> setDifficulty` through the real socket/server path; validate with `go test ./serverengine -run 'TestHandleWebSocket|TestRescaleActDeck|TestProcessAction_(SelectInvestigator|SetDifficulty)'` plus a new end-to-end test.

### MEDIUM

- [ ] **Ebitengine scene flow is only partially wired to the published client contract** - `docs/CLIENT_SPEC.md:62`, `docs/CLIENT_SPEC.md:68`, `docs/CLIENT_SPEC.md:79`, `docs/CLIENT_SPEC.md:93`, `client/ebiten/app/scenes.go:24`, `client/ebiten/app/scenes.go:94`, `client/ebiten/app/scenes.go:178` - `SceneConnect` is only an animated banner; it has no server-address input, no player-display-name input, no slot status, and no reconnect countdown/slot-wait state. `SceneCharacterSelect` sends a local selection, but `updateScene()` advances to `SceneGame` as soon as the local player has an investigator rather than when all connected players have confirmed. This leaves the alpha client short of its own published scene contract. **Remediation:** extend connect-state data in `LocalState` and the wire contract as needed, add connect-scene input/wait UI, and gate the `SceneCharacterSelect -> SceneGame` transition on all connected players having selected an investigator; validate with `go test -tags=requires_display ./client/ebiten/app/...` and a manual desktop run against a multi-player server.

### DEAD CODE

- [ ] **Browser self-quality update branch never fires** - `client/game.js:462`, `serverengine/connection_quality.go:24` - the browser client checks `qualityMessage.playerID`, but the server emits `playerId`. The per-player list still uses `allPlayerQualities`, but the top-right connection-status upgrade path is unreachable. **Remediation:** change the comparison to `qualityMessage.playerId === this.playerId`, add a fixture-driven client-side test for a `connectionQuality` payload, and validate by receiving a ping/pong cycle from a live server while watching the status badge change.

### LOW

- [ ] **Ebitengine connection-quality schema is stale relative to the server contract** - `client/ebiten/state.go:87`, `client/ebiten/state.go:89`, `client/ebiten/state.go:194`, `client/ebiten/net_test.go:102`, `client/ebiten/net_test.go:331`, `serverengine/connection_quality.go:12`, `serverengine/connection_quality.go:13`, `serverengine/connection_quality.go:24` - the Go client still expects `quality: { latency, rating }` and stores `cs.Quality.Rating`, while the server emits `quality: { latencyMs, quality }`. Current rendering does not consume the field, so this is not yet user-visible, but it codifies a stale protocol into tests and will break future quality-HUD work. **Remediation:** align `client/ebiten/state.go` and the net tests to `latencyMs`/`quality`, then add a round-trip test using an actual server-shaped payload; validate with `go test ./client/ebiten/...`.

- [ ] **Modular package decomposition is still scaffolding-only in 25 packages** - representative refs: `serverengine/arkhamhorror/actions/doc.go:1`, `serverengine/common/messaging/doc.go:1`, `serverengine/eldersign/adapters/doc.go:1` - the directory structure for Arkham subpackages, shared cross-engine packages, and other game families exists, but 25 packages are still `doc.go` placeholders with no implementation. This is intentional scaffolding, but it means the architecture advertised in `serverengine/arkhamhorror/README.md` has barely started. **Remediation:** either collapse unused scaffolds until migration restarts, or migrate one slice at a time beginning with `serverengine/arkhamhorror/model` and `serverengine/arkhamhorror/rules`; validate with `go list ./...` and package-level tests as slices move.

- [ ] **Selectable non-Arkham modules are startup stubs** - `cmd/server/main.go:29`, `cmd/server/main.go:30`, `cmd/server/main.go:31`, `cmd/server/main.go:32`, `serverengine/eldersign/module.go:22` - `eldersign`, `eldritchhorror`, and `finalhour` are registered as selectable modules, but each `NewEngine()` returns `runtime.NewUnimplementedEngine(...)`. The behavior is documented, so this is low severity, but module selection currently advertises options that terminate with a not-implemented error. **Remediation:** either remove these modules from the default registry until they have a playable engine, or gate them behind an explicit experimental flag; validate with `BOSTONFEAR_GAME=<module> go run ./cmd/server` for each registered module.

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| Reconnect tokens / session persistence looked missing because `docs/CLIENT_SPEC.md` cites a nonexistent `PLAN.md` and a 60-second grace period | Rejected as an implementation gap. The actual code does persist reconnect tokens in `client/ebiten/state.go` and reclaims slots in `serverengine/connection.go`; the doc is stale, not the runtime. |
| `BuildSystemAlerts` looked unreferenced from `go-stats-generator`'s dead-code candidates | Rejected after direct usage check: `monitoring/handlers.go:31` calls it in the `/health` response path. |
| `cmd/mobile`'s `SetServerURL` and `Dummy` looked like no-op exports | Rejected. `Dummy` is required by `ebitenmobile` binding generation, and `SetServerURL` is the native-host override point documented in `cmd/mobile/binding.go`. |
| Touch input appeared absent from the Ebitengine client based on older reports | Rejected after code read: `client/ebiten/app/input.go` has a real `handleTouchInput` implementation wired from `InputHandler.Update()`. |
| Empty scaffold packages under `serverengine/*` | Downgraded to LOW, not MEDIUM/HIGH, because the scaffolding is explicitly documented as migration-first structure in `serverengine/arkhamhorror/README.md`. |
| Comment debt (TODO/FIXME/HACK/XXX) | Rejected. `go-stats-generator` and repo-wide search both returned zero actionable TODO/FIXME/HACK/XXX markers in production Go code. |# IMPLEMENTATION GAP AUDIT — 2026-05-08

## Project Architecture Overview

BostonFear is a rules-only multiplayer Arkham Horror game engine.  The project
has three layers:

| Layer | Packages | Responsibility |
|-------|----------|----------------|
| **Server engine** | `serverengine`, `serverengine/arkhamhorror` | AH3e game rules, WebSocket connection lifecycle, state broadcast |
| **Transport** | `transport/ws` | HTTP upgrade, route registration over `net.Listener` |
| **Monitoring** | `monitoring`, `monitoringdata` | Health/metrics/dashboard HTTP handlers |
| **Protocol** | `protocol` | Shared JSON wire schema consumed by server and Go clients |
| **Common runtime** | `serverengine/common/contracts`, `serverengine/common/runtime` | Registy, `Engine` interface, `GameModule` interface, `UnimplementedEngine` |
| **Game-family scaffolds** | `serverengine/arkhamhorror/{actions,adapters,content,model,phases,rules,scenarios}`, `serverengine/common/{messaging,monitoring,observability,session,state,validation}`, `serverengine/eldersign/{adapters,model,rules,scenarios}`, `serverengine/eldritchhorror/…`, `serverengine/finalhour/…` | Intended future decomposition — currently empty `doc.go` placeholders |
| **Ebitengine client** | `client/ebiten`, `client/ebiten/app`, `client/ebiten/render` | Desktop/WASM/mobile game client (alpha) |
| **Entry points** | `cmd/server`, `cmd/desktop`, `cmd/web`, `cmd/mobile` | Platform boot |

### Stated Goals (from README/ROADMAP)
1. 5 core game mechanics (Location, Resources, Actions, Doom, Dice) — **achieved**
2. 1-6 concurrent players with join-in-progress — **achieved**
3. Real-time sync < 500 ms — **achieved**
4. Interface-based design (`net.Conn`, `net.Listener`) — **achieved**
5. Ebitengine desktop + WASM builds — **achieved (alpha, placeholder sprites)**
6. Session persistence / reconnect tokens — **achieved**
7. Performance monitoring endpoints — **achieved**
8. Mobile binding (iOS/Android) — **partial (scaffold only; not device-verified)**
9. Module-based game-family registry — **achieved (server selection wired)**
10. Full Arkham rules decomposition into `serverengine/arkhamhorror/` sub-packages — **not started**

---

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs / Empty scaffolds | 22 | 0 | 0 | 4 | 18 |
| Dead Code | 0 | 0 | 0 | 0 | 0 |
| Partially Wired | 3 | 0 | 2 | 1 | 0 |
| Interface Gaps | 1 | 0 | 1 | 0 | 0 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

---

## Implementation Completeness by Package

| Package | Exported Functions | Implemented | Stubs | Dead | Notes |
|---------|--------------------|-------------|-------|------|-------|
| `serverengine` | ~132 total functions | 132 | 0 | 0 | Fully functional |
| `serverengine/arkhamhorror` | 4 | 4 | 0 | 0 | Module binding complete |
| `serverengine/arkhamhorror/actions` | 0 | 0 | 0 | 0 | Empty scaffold (`doc.go` only) |
| `serverengine/arkhamhorror/adapters` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/arkhamhorror/content` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/arkhamhorror/model` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/arkhamhorror/phases` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/arkhamhorror/rules` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/arkhamhorror/scenarios` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/common/contracts` | 2 interfaces | 2 | 0 | 0 | Fully defined |
| `serverengine/common/runtime` | 8 | 8 | 0 | 0 | Registry + UnimplementedEngine |
| `serverengine/common/messaging` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/common/monitoring` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/common/observability` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/common/session` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/common/state` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/common/validation` | 0 | 0 | 0 | 0 | Empty scaffold |
| `serverengine/eldersign` | 4 | 0 (stub engine) | 4 | 0 | `UnimplementedEngine` placeholder |
| `serverengine/eldersign/{adapters,model,rules,scenarios}` | 0 | 0 | 0 | 0 | Empty scaffolds |
| `serverengine/eldritchhorror` | 4 | 0 (stub engine) | 4 | 0 | `UnimplementedEngine` placeholder |
| `serverengine/eldritchhorror/{adapters,model,rules,scenarios}` | 0 | 0 | 0 | 0 | Empty scaffolds |
| `serverengine/finalhour` | 4 | 0 (stub engine) | 4 | 0 | `UnimplementedEngine` placeholder |
| `serverengine/finalhour/{adapters,model,rules,scenarios}` | 0 | 0 | 0 | 0 | Empty scaffolds |
| `transport/ws` | 13 | 13 | 0 | 0 | Fully functional |
| `monitoring` | 6 | 6 | 0 | 0 | Fully functional |
| `protocol` | ~40 types | 40 | 0 | 0 | Fully functional |
| `client/ebiten` | 27 | 27 | 0 | 0 | Fully functional |
| `client/ebiten/app` | 24 | 21 | 3 | 0 | SceneCharacterSelect + touch input + Play Again missing |
| `client/ebiten/render` | 14 | 14 | 0 | 0 | Placeholder sprites; shaders compile |
| `cmd/mobile` | 3 | 3 | 0 | 0 | Scaffold compiles; not device-verified |

---

## Findings

### HIGH

- [ ] **SceneCharacterSelect missing** — [client/ebiten/app/scenes.go](client/ebiten/app/scenes.go#L5) — `CLIENT_SPEC.md §1` mandates the scene transition `SceneConnect → SceneCharacterSelect → SceneGame`. The `SceneCharacterSelect` type is explicitly called out as deferred in the package comment (`"SceneCharacterSelect is deferred until the selectInvestigator server action is fully wired on the client side"`). The `updateScene()` function ([client/ebiten/app/scenes.go](client/ebiten/app/scenes.go#L101)) transitions directly from `SceneConnect` to `SceneGame`, bypassing investigator selection entirely. The `ActionSelectInvestigator` action is implemented server-side (`serverengine/game_server.go:231`) but is never sent by the Ebitengine client. This means desktop/WASM players cannot choose their investigator archetype.
  — **Blocked goal**: CLIENT_SPEC.md §2 (character selection before play begins)
  — **Remediation**: Implement `SceneCharacterSelect` struct with `Update()`/`Draw()` methods in `client/ebiten/app/scenes.go`. Add a state transition in `updateScene()` that checks whether the player's `InvestigatorType` is set; if not, route to `SceneCharacterSelect`. Wire keyboard/click input to send `ActionSelectInvestigator` via `InputHandler`. Add key bindings for the 6 investigator types in `input.go`. Validated by: `go test -race -tags=requires_display ./client/ebiten/app/...`.

- [ ] **Ebitengine InputHandler exposes only 7 of 12 server actions** — [client/ebiten/app/input.go](client/ebiten/app/input.go#L18) — `keyBindings` maps only Move (×4), Gather, Investigate, and CastWard. The server engine implements and accepts Focus, Research, Trade, Component, Attack, Evade, and CloseGate actions. The Ebitengine client is the only non-HTML/JS client; players using it cannot trigger 5 AH3e action types.
  — **Blocked goal**: AH3e action parity between browser client and native client (CLIENT_SPEC.md §3)
  — **Remediation**: Add key bindings in `keyBindings` slice for Focus (`KeyF`), Research (`KeyR`), Component (`KeyC`), Attack (`KeyA`), Evade (`KeyE`), CloseGate (`KeyX`). Trade requires a target player ID — add a secondary input flow or a dedicated `KeyT` binding that targets the first co-located player. Validate by running the desktop client and confirming actions reach the server log.

### MEDIUM

- [ ] **SceneGameOver "Play Again" transition not implemented** — [client/ebiten/app/scenes.go](client/ebiten/app/scenes.go#L73) — `SceneGameOver.Update()` is a no-op (`return nil`). `CLIENT_SPEC.md §1` state machine shows `SceneGameOver → SceneConnect` as the "Play Again" arc. No keyboard/click input is handled; the only way to restart is to close and reopen the process.
  — **Blocked goal**: CLIENT_SPEC.md §1 state machine completeness
  — **Remediation**: In `SceneGameOver.Update()`, detect a key press (e.g. `inpututil.IsKeyJustPressed(ebiten.KeyEnter)`) and call `g.net.Reconnect()` + reset `g.activeScene` to `&SceneConnect{game: g}`. The `Draw()` method already renders "Close the window to exit" — update this text to include the restart shortcut. Validate with `go test -race -tags=requires_display ./client/ebiten/app/...`.

- [ ] **Touch/mobile input not implemented in InputHandler** — [client/ebiten/app/input.go](client/ebiten/app/input.go) — `InputHandler.Update()` only polls keyboard events. `CLIENT_SPEC.md §1` ("Touch input on mobile platforms via `ebiten.TouchID` mapping") and `ROADMAP.md Phase 4` require touch input for the mobile client (`cmd/mobile`). The mobile binding compiles but players cannot issue any game action via touch.
  — **Blocked goal**: ROADMAP.md Phase 4 — mobile touch input
  — **Remediation**: Add a `handleTouchInput()` method to `InputHandler` that maps tap regions (using `locationRects` from `game.go`) to move actions and dedicated HUD regions to non-move actions. Call it from `Update()` when `ebiten.TouchIDs(nil)` returns any IDs. Validate on Android emulator or with Ebitengine's touch simulation in tests.

### LOW

- [ ] **22 empty scaffold packages (`doc.go` only)** — 7 under `serverengine/arkhamhorror/`, 6 under `serverengine/common/`, and 3×4 under `eldersign/`, `eldritchhorror/`, `finalhour/` — these packages compile empty and export nothing. They represent intended future decomposition of the monolithic `serverengine` package but have no planned implementation timeline beyond "future migration" in ROADMAP.md.
  — **Blocked goal**: ROADMAP.md modular rules decomposition
  — **Remediation**: No immediate code change needed — this is intentional scaffolding. If the decomposition is not planned for the next release, consider consolidating to reduce `go list ./...` noise. When implementation begins, start with `serverengine/arkhamhorror/model` (domain types) then `serverengine/arkhamhorror/rules` (rule evaluation), which are the natural migration boundary per `RULES.md`.

- [ ] **`elderSign`, `eldritchHorror`, `finalHour` engines always return `not-implemented` error** — [serverengine/eldersign/module.go](serverengine/eldersign/module.go#L22), [serverengine/eldritchhorror/module.go](serverengine/eldritchhorror/module.go), [serverengine/finalhour/module.go](serverengine/finalhour/module.go) — Selecting these modules via `BOSTONFEAR_GAME` causes `run()` to return an error at startup (via `module.NewEngine()`). This is documented behavior ("placeholder modules … intentionally return a not-implemented runtime error"), not a defect. Tracked here for completeness.
  — **Remediation**: Implement rules for each game family in their respective `rules/` packages when scope permits. Start with `eldersign` as it shares the most mechanical overlap with Arkham Horror (dice pools, clue-based win condition).

- [ ] **`serverengine/common/{messaging,monitoring,observability,session,state,validation}` are unreferenced** — None of these packages are imported by any other package in the module. They were scaffolded to define future cross-engine primitives but remain doc-only. No current code depends on them.
  — **Remediation**: Populate when the `serverengine` monolith begins migration. `validation` and `state` are the highest-priority stubs since `serverengine` already has `StateValidator` and resource-bound logic that could move there.

- [ ] **Mobile client not verified on a physical device** — [cmd/mobile/binding.go](cmd/mobile/binding.go) — as noted in README and ROADMAP. The binding compiles; on-device launch behavior is unknown.
  — **Remediation**: Run `ebitenmobile bind -target android` and test on emulator API 29+, or a physical device. Document results in ROADMAP.md.

- [ ] **`monitoring/common` (sub-package) is an empty scaffold** — [serverengine/common/monitoring/doc.go](serverengine/common/monitoring/doc.go) — The `monitoring` sub-package under `common` is empty while the fully functional `monitoring` package at the root already covers all health/metrics needs. Risk of future confusion between `serverengine/common/monitoring` and `monitoring`.
  — **Remediation**: Add a package-level comment to `serverengine/common/monitoring/doc.go` clarifying that this package is reserved for cross-engine monitoring primitives distinct from the HTTP-handler package at `monitoring/`. Alternatively, remove it until it has content.

---

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| `UnimplementedEngine` returns errors | Intentional design — documented in README as "intentionally return a not-implemented runtime error." The error is returned at startup, preventing silent failure. |
| `registry.MustRegister` panics on duplicate | Panic-on-programmer-error is idiomatic Go for invariant violations at init time. Not a production risk. |
| `Broadcaster` and `StateValidator` interfaces have only one implementation each | Both interfaces are used for testing (see `interfaces_test.go` and `test_helpers_test.go`). Single-implementation interfaces are appropriate here for testability. |
| `channelBroadcaster` is unexported | It is instantiated in `NewGameServer` and accessed only via the `Broadcaster` interface. Correct design. |
| `monitoring.Provider` mirrors `contracts.Engine` methods | The duplication is intentional — `monitoring.Provider` is a narrower surface exposed to HTTP handlers and does not carry game-action methods. |
| Empty `doc.go` files in eldersign/eldritchhorror/finalhour sub-packages | Explicitly documented as placeholder scaffolding in README ("registered for scaffolding"). |
| `SceneConnect.Update()` has no form fields | `SceneConnect` connects using the URL passed to `NewGame`; the server URL is a CLI flag, not a runtime UI input in the current alpha. This is intentional for the alpha client. |
| `sprites.png` contains only a placeholder | Documented as "placeholder programmer-art assets" in both README and CLIENT_SPEC. Not a code gap. |
