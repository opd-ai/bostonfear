# IMPLEMENTATION GAP AUDIT — 2026-05-08

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
