# IMPLEMENTATION GAP AUDIT — 2026-05-16

## Project Architecture Overview

**BostonFear** is a multiplayer Arkham Horror–style rules engine with a Go WebSocket
server (`serverengine`) and a Go/Ebitengine client (`client/ebiten`) compiling to
desktop, WASM, and mobile targets.

### Package Responsibilities (stated)
| Package | Responsibility |
|---------|---------------|
| `serverengine` | Core Arkham game loop, player state, WebSocket handling (compat facade) |
| `serverengine/arkhamhorror` | Arkham-specific rules extraction (active migration) |
| `serverengine/arkhamhorror/rules` | Pure rule functions: dice, movement |
| `serverengine/arkhamhorror/actions` | Action dispatch table |
| `serverengine/arkhamhorror/phases` | Turn/mythos phase orchestration |
| `serverengine/arkhamhorror/model` | Resource bounds / investigator archetypes |
| `serverengine/arkhamhorror/content` | Embedded scenario content |
| `serverengine/arkhamhorror/adapters` | Protocol adapter layer (scaffold) |
| `serverengine/arkhamhorror/scenarios` | Scenario template system (scaffold) |
| `serverengine/common/contracts` | Cross-engine interface definitions |
| `serverengine/common/runtime` | Unimplemented engine placeholder |
| `serverengine/common/{messaging,session,state,validation,observability}` | Future cross-engine primitives (all scaffold) |
| `serverengine/{eldersign,eldritchhorror,finalhour}` | Placeholder game modules (not implemented) |
| `transport/ws` | HTTP + WebSocket upgrade over `net.Listener` |
| `monitoring` | Health + Prometheus metrics HTTP handlers |
| `monitoringdata` | Shared DTO types |
| `protocol` | Wire schema (JSON structs, action enums) |
| `client/ebiten` | Ebitengine game client |
| `client/ebiten/app` | Scene management, update/draw loop |
| `client/ebiten/render` | Atlas, sprite resolution, Kage shaders |
| `client/ebiten/ui` | HUD, tokens, onboarding, camera |
| `cmd` | Cobra CLI commands |

### Stated Goals (from README / copilot-instructions)
1. All 5 core mechanics fully functional: Location, Resources, Actions, Doom, Dice
2. 1-6 concurrent players, turn-based, with in-progress join
3. Multiplayer via WebSocket with `net.Conn`/`net.Listener`/`net.Addr` interfaces
4. Ebitengine client for desktop, WASM, and mobile from single codebase
5. Real-time state sync ≤ 500 ms, sub-100 ms health checks
6. Multiple Fantasy Flight–style game families selectable via `BOSTONFEAR_GAME`

---

## Baseline Metrics (go-stats-generator)

| Metric | Value |
|--------|-------|
| Total files | 128 |
| Total LOC | 6 782 |
| Total functions | 229 |
| Total methods | 419 |
| Total structs | 161 |
| Total interfaces | 19 |
| Total packages | 35 |
| Doc coverage (functions) | 91.3 % |
| Doc coverage (overall) | 81.4 % |
| `go build ./...` | ✅ clean |
| `go vet ./...` | ✅ clean |

---

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs / Placeholders | 11 | 0 | 3 | 5 | 3 |
| Dead / Unreachable Code | 3 | 0 | 1 | 2 | 0 |
| Partially Wired Components | 5 | 0 | 2 | 2 | 1 |
| Interface / Contract Gaps | 3 | 0 | 1 | 1 | 1 |
| Dependency Gaps | 2 | 0 | 0 | 1 | 1 |

---

## Implementation Completeness by Package

| Package | Exported Fns | Implemented | Stubs | Dead | Notes |
|---------|-------------|-------------|-------|------|-------|
| `serverengine` | ~90 | ~85 | 1 | 2 | Core mechanics complete |
| `serverengine/arkhamhorror` | 3 | 3 | 0 | 0 | Engine wrapper only |
| `serverengine/arkhamhorror/actions` | 1 | 1 | 0 | 0 | Dispatch complete |
| `serverengine/arkhamhorror/phases` | 3 | 3 | 0 | 0 | Turn/mythos wired |
| `serverengine/arkhamhorror/rules` | 5 | 5 | 0 | 0 | Dice + movement complete |
| `serverengine/arkhamhorror/model` | 5 | 5 | 0 | 0 | Resource bounds |
| `serverengine/arkhamhorror/content` | 4 | 4 | 0 | 0 | Nightglass embed |
| `serverengine/arkhamhorror/adapters` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/arkhamhorror/scenarios` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/common/contracts` | 4 | 4 | 0 | 0 | Interfaces defined |
| `serverengine/common/runtime` | 1 | 1 | 0 | 0 | UnimplementedEngine |
| `serverengine/common/messaging` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/common/session` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/common/state` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/common/validation` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/common/observability` | 0 | 0 | 0 | 0 | **Doc-only scaffold** |
| `serverengine/eldersign` | 1 | 0 | 1 | 0 | Placeholder module |
| `serverengine/eldritchhorror` | 1 | 0 | 1 | 0 | Placeholder module |
| `serverengine/finalhour` | 1 | 0 | 1 | 0 | Placeholder module |
| `transport/ws` | 2 | 2 | 0 | 0 | Complete |
| `monitoring` | 3 | 3 | 0 | 0 | Complete |
| `protocol` | ~10 | ~10 | 0 | 0 | Wire schema complete |
| `client/ebiten` | ~20 | ~20 | 0 | 0 | Client core complete |
| `client/ebiten/app` | ~15 | ~14 | 0 | 1 | Missing encounter action |
| `client/ebiten/render` | ~12 | ~12 | 0 | 0 | Placeholder sprites noted |
| `client/ebiten/ui` | ~20 | ~20 | 0 | 0 | Complete |

---

## Findings

### HIGH

- [x] **Encounter action missing from client action menu and keyboard bindings** —
  `client/ebiten/app/game.go:750` (action menu builder), `client/ebiten/app/input.go:44`
  (key bindings) — The `encounter` action type is defined in `protocol/protocol.go:36`
  (`ActionEncounter = "encounter"`), fully implemented in `serverengine/actions.go:181`
  (`performEncounter`), wired in `serverengine/game_server.go:472`, and accessible as a
  `ui.ActionEncounter` component constant (`client/ebiten/ui/components.go:130`), but it
  is absent from the client-side `buildActionMenu()` in `game.go` and has no keyboard
  binding in `keyBindings` (`input.go:44`). Players cannot trigger encounter resolution
  from the Ebitengine client. Blocked goal: stated action system completeness; also breaks
  `performComponent` fallback that calls `performEncounter` internally.
  **Remediation:** Add `add("Encounter", "draw location card", turnActive && remaining > 0)`
  to `buildActionMenu()` in `client/ebiten/app/game.go` and add
  `{ebiten.KeyN, "encounter", ""}` to `keyBindings` in `input.go`. Include
  `"encounter": protocol.ActionEncounter` in `touchActionMap`.
  Validate with `go build ./cmd/desktop`.

- [x] **`serverengine/arkhamhorror/adapters` is a doc-only scaffold — BroadcastPayloadAdapter
  migration is incomplete** — `serverengine/arkhamhorror/adapters/doc.go:9` and
  `serverengine/arkhamhorror/adapters/adapter.go:7` — The doc says this package "translates
  between Arkham Horror-specific game events and the shared runtime contracts" and is needed
  to decouple game rules from the network protocol. An `arkhamBroadcastAdapter` struct is
  declared in `adapter.go` but `serverengine/game_server.go` directly references the
  embedded `BroadcastPayloadAdapter` interface (line 219) using the `serverengine`-scoped
  copy, not the arkhamhorror one. The adapter package exports no functions and is unused
  by any non-test import. Blocked goal: package separation / migration plan for
  `serverengine/arkhamhorror`.
  **Remediation:** Implement `BroadcastPayloadAdapter` methods in
  `serverengine/arkhamhorror/adapters/adapter.go` and update `serverengine/game_server.go`
  to call `adapters.NewArkhamBroadcastAdapter()` instead of the embedded inline adapter.
  Verify with `go build ./... && go vet ./...`.

- [x] **`serverengine/arkhamhorror/scenarios` is a doc-only scaffold — scenario selection
  does not use a scenario registry** — `serverengine/arkhamhorror/scenarios/doc.go:7` and
  `serverengine/actions.go:452-480` — `performSelectScenario` constructs a scenario by
  mutating `DefaultScenario` (a monolith constant) rather than looking up a registered
  scenario from a content index. The `scenarios` package is empty; no `ScenarioTemplate`
  types or index loader exist there. The README documents a multi-step content-loader
  fallback contract (`docs/content/BASE_SET_DEFAULT_SCENARIO_SPEC.md`) that the code does
  not implement. Blocked goal: scenario selection feature described in README; Nightglass
  content pack multi-scenario fallback.
  **Remediation:** Define a `ScenarioTemplate` struct and `Index` type in
  `serverengine/arkhamhorror/scenarios/`, populate from the embedded YAML in
  `serverengine/arkhamhorror/content/nightglass/scenarios/`, and update
  `performSelectScenario` to resolve against the index.
  Validate with `go test ./serverengine/arkhamhorror/scenarios/...`.

### MEDIUM

- [x] **Six `common` sub-packages are doc-only scaffolds with no exported symbols** —
  `serverengine/common/messaging/doc.go:5`, `serverengine/common/session/doc.go:5`,
  `serverengine/common/state/doc.go:5`, `serverengine/common/validation/doc.go:5`,
  `serverengine/common/observability/doc.go:5` — Each package contains only a `doc.go`
  with a `NOTE: scaffold. Implementation is deferred.` comment. None are imported by any
  production code. They add structural noise and inflate the package count from 35 to 40
  without providing any functionality. Blocked goal: cross-engine modularization described
  in README §Package Separation.
  **Remediation:** Either implement the minimum viable type set for each package
  (e.g., `session.Token`, `state.ResourceBounds`) and wire them into `serverengine`,
  or remove the empty packages and document the intent in a single
  `serverengine/common/ROADMAP.md` file until implementation is ready.
  Validate with `go build ./...` and confirm no import errors.

- [x] **Placeholder game modules (eldersign, eldritchhorror, finalhour) are scaffolded
  with sub-packages (model, rules, adapters, scenarios) that contain only `doc.go` files
  and are never registered in the server startup** — `serverengine/eldersign/module.go:25`,
  `serverengine/eldritchhorror/module.go:27`, `serverengine/finalhour/module.go:26`,
  `cmd/server.go:57-70` — Each module returns `UnimplementedEngine` from `NewEngine()`,
  making the `BOSTONFEAR_GAME=eldersign` selection fail at startup. The 12 scaffold
  sub-packages (3 modules × 4 sub-packages) each contain only a `doc.go`. The
  `cmd/server.go` comments them out explicitly. Blocked goal: multi-game-family selection
  via `BOSTONFEAR_GAME`.
  **Remediation (tracked/low-urgency):** No immediate remediation needed — this is an
  acknowledged scaffold per README §Unimplemented/Scaffolding Packages. Flag for tracking
  only. When any module is ready, uncomment the `registry.MustRegister` call in
  `cmd/server.go:68-70`.

- [x] **`monitoring/handlers.go` exports a `p90` Prometheus metric label that the
  production `GameServer.GetLatencyPercentiles()` never populates** —
  `monitoring/handlers.go:247-248` (NOTE comment acknowledges this), `serverengine/metrics.go:256`
  — `GetLatencyPercentiles()` returns keys `"p50"`, `"p95"`, `"p99"` only. The handler
  checks for `percentiles["p90"]` at line 245 and emits it, but this map key is never set
  by the production implementation, so the Prometheus series
  `arkham_horror_broadcast_latency_percentiles_ms{quantile="0.90"}` is silently absent
  from every scrape. The `contracts/engine.go:90` doc claims the keys are
  `"p50", "p90", "p95", "p99"`. Blocked goal: accurate Prometheus latency monitoring.
  **Remediation:** Either (a) replace `"p90"` with `"p95"` in the handler's map lookup
  and Prometheus label at `monitoring/handlers.go:244-246`, or (b) add `"p90"` computation
  to `GameServer.GetLatencyPercentiles()` in `serverengine/metrics.go:256`.
  Update `contracts/engine.go:90` doc to match. Validate with
  `curl http://localhost:8080/metrics | grep quantile`.

- [x] **`serverengine/arkhamhorror/adapters/adapter.go` declares `arkhamBroadcastAdapter`
  but the struct body is empty and no methods satisfy the interface** —
  `serverengine/arkhamhorror/adapters/adapter.go:7` — The file declares the struct
  comment promising it implements `BroadcastPayloadAdapter` for Arkham Horror, but the
  struct has no fields and no methods. No code instantiates it. This is a narrower
  sub-finding of the HIGH adapter-migration gap above, but captures the empty struct
  specifically.
  **Remediation:** Covered by the HIGH finding above.

- [x] **`cmd/web_nowasm.go` registers a `web` CLI command that always errors on
  non-WASM builds** — `cmd/web_nowasm.go:12` — On desktop and server builds, `go run .
  web` returns `"web command is only available for js/wasm builds"`. This is documented
  behavior but the command is registered un-hidden only on WASM targets; on non-WASM
  targets it is `Hidden: true`. The actual behavior is correct, but the `Hidden` field
  prevents its appearance in `--help` without documenting the cross-build constraint
  inline. Low user-facing impact.
  **Remediation:** No code change required; add a short `Long` description to the stub
  command explaining the js/wasm constraint. Low priority.

### LOW

- [x] **`serverengine/game_constants.go` defines `DifficultyConfig` with `easy`,
  `standard`, and `hard` presets, but the client exposes no difficulty-selection UI** —
  `serverengine/game_constants.go:210`, `client/ebiten/app/scenes.go:375`
  (SceneCharacterSelect) — `setDifficulty` is reachable via the `playerAction` protocol
  message (`ActionSetDifficulty = "setdifficulty"`) and is implemented server-side, but
  the onboarding / character-select scene has no UI element to send this message. Players
  can only change difficulty by sending a raw WebSocket message outside the client.
  Blocked goal: viable difficulty selection for non-developer users.
  **Remediation:** Add a difficulty selector widget to `SceneCharacterSelect.Draw()` in
  `client/ebiten/app/scenes.go` and wire it to send
  `protocol.PlayerActionMessage{Action: protocol.ActionSetDifficulty, Target: "standard"}`
  (or selected value) via `NetClient`. Validate with a manual client smoke test.

- [x] **`serverengine/arkhamhorror/rules/movement.go` is 20 lines and only re-exports
  the adjacency map from `serverengine/arkhamhorror/content`** —
  `serverengine/arkhamhorror/rules/movement.go:11` — The package doc says "owns all Arkham
  Horror rule logic including die mechanics, spell casting, and investigation system" but
  the file contains one function that delegates completely. Spell casting, investigation
  thresholds, and ward success rules remain in the `serverengine` monolith. Low urgency
  given active migration, but the package doc overstates current content.
  **Remediation:** Update the `rules` package doc to reflect the actual migration status,
  or migrate `performInvestigate` / `performCastWard` threshold constants into
  `rules/thresholds.go`.

- [x] **`UnimplementedEngine.GetLatencyPercentiles()` returns `"p90": 0` which is
  inconsistent with the production engine** — `serverengine/common/runtime/unimplemented_engine.go:107`
  — Inconsistency between the placeholder (p90) and production (no p90 key). If monitoring
  code branches on key presence this could mask a real p90 absence.
  **Remediation:** Align `UnimplementedEngine.GetLatencyPercentiles()` to match
  production keys `{"p50":0,"p95":0,"p99":0}` after resolving the HIGH p90/p95
  mismatch finding.

---

## Dead / Unreachable Code

- [x] **`serverengine/arkhamhorror/actions/interface.go` defines `GameEngine` interface
  with 7 methods — this interface is never used** — `serverengine/arkhamhorror/actions/interface.go:6`
  — `GameEngine` declares `FindEngagedEnemy`, `GameState`, `RollDicePool`,
  `ValidateMovement`, `ValidateResources`, `CheckInvestigatorDefeat`, `SealAnomalyAtLocation`
  with `interface{}` return types. No code in the project type-asserts to or accepts a
  `GameEngine` parameter. The actual dispatch uses `CallbackSet` (concrete function values),
  not this interface. The interface is dead code. Blocked goal: none (the callback approach
  works correctly).
  **Remediation:** Remove `serverengine/arkhamhorror/actions/interface.go` or document
  that it is a migration artifact reserved for a future refactor phase. Verify removal
  with `go build ./...`.

- [x] **`DifficultyConfig` is parsed and applied server-side but its `ExtraDoomTokens`
  field is never consumed** — `serverengine/game_constants.go:206`, `serverengine/game_mechanics.go:139`
  — `applyDifficulty` reads `cfg.InitialDoom` (sets `gameState.Doom`) but discards
  `cfg.ExtraDoomTokens`. No code adds extra doom tokens to the MythosCup based on this
  field. The field exists in the struct and is set in the config table
  (`"standard": {1, 1}`, `"hard": {3, 3}`) but is never used.
  **Remediation:** Either implement the mythos cup token-loading logic that reads
  `ExtraDoomTokens` during game setup (in `applyDifficulty` or `InitGameState`), or
  remove the field and update the `DifficultySetup` struct and config table.

- [x] **`serverengine/game_server.go` references `ActionSelectScenario` handling but the
  resolved scenario is always a copy of `DefaultScenario` with only the Name mutated** —
  `serverengine/actions.go:456-478` — `performSelectScenario` builds a scenario as
  `scenario := DefaultScenario; scenario.Name = scenarioID` which ignores any content
  differences between scenarios. For a single-scenario system this is harmless; it becomes
  dead logic once content-loader multi-scenario support is added. Covered as a sub-finding
  of the HIGH scenarios gap.

---

## Partially Wired Components

- [x] **`serverengine/arkhamhorror/engine.go` embeds `*serverengine.GameServer` but
  `contracts.Engine` is satisfied by `*GameServer` directly — the `arkhamhorror.Engine`
  wrapper is never instantiated by `module.go`** —
  `serverengine/arkhamhorror/engine.go:8`, `serverengine/arkhamhorror/module.go:35` —
  `module.go:NewEngine()` returns `serverengine.NewGameServer(...)` cast to
  `contracts.Engine`, not `&arkhamhorror.Engine{...}`. The `Engine` struct in `engine.go`
  is never constructed. It is a structural placeholder for future per-module method
  overrides.
  **Remediation:** Either use `arkhamhorror.Engine` in `module.go:NewEngine()` so the
  wrapper is instantiated, or remove `engine.go` and document the intent.

- [x] **`serverengine/common/contracts` defines a `GameRunner` interface with a single
  `Start() error` method, but `transport/ws` uses its own `SessionEngine` interface
  (`transport/ws/websocket_handler.go:26`) instead of `contracts.Engine`** —
  `serverengine/common/contracts/engine.go:112`, `transport/ws/websocket_handler.go:26`
  — `SessionEngine` duplicates a subset of `contracts.SessionHandler` (two methods:
  `HandleConnection`, `SetAllowedOrigins`) and is not connected to the full `contracts.Engine`
  contract. The transport layer is decoupled from the contracts package that was designed
  to unify them.
  **Remediation:** Change `SessionEngine` in `transport/ws/websocket_handler.go` to
  accept `contracts.SessionHandler` (or `contracts.Engine`) so that the transport layer
  uses the canonical interface. Validate with `go build ./transport/ws/...`.

- [x] **`Scenario.SetupFn` callback is defined in `DefaultScenario` but is `nil` for
  the default scenario — `performSelectScenario` guards on `nil` without logging** —
  `serverengine/actions.go:475-478` — The `if scenario.SetupFn != nil` check silently
  skips re-applying scenario setup when a client requests the default scenario by name.
  If a YAML-loaded scenario has a non-nil `SetupFn` this will work; for the default
  hardcoded scenario, players selecting it via `selectscenario` action receive no deck
  reset, which may leave stale state from a previous selection.
  **Remediation:** Initialize `DefaultScenario.SetupFn` to call `InitGameState(gs.gameState)`
  so scenario re-selection always resets decks and doom to initial values.

- [x] **Metrics counter `DoomHistogram` is defined on `GameServer` but doom-level
  entries are only recorded at game-end, not during incremental doom advances** —
  `serverengine/metrics.go` (`GetDoomHistogram`), `serverengine/game_server.go:347`
  (broadcast on doom change) — The histogram is updated by recording the doom level at
  game end. Prometheus scrapes mid-game will always show the histogram populated only
  for completed games; a 15-minute game never updates the histogram until it ends.
  This partially fulfils the Prometheus analytics intent. Low severity.
  **Remediation:** No code change required for the current monitoring intent; document
  the sampling policy in `monitoring/handlers.go`.

- [x] **`serverengine/arkhamhorror/content/map.go` provides neighbourhood topology but
  `serverengine/game_server.go` also hard-codes `locationAdjacency` in
  `serverengine/game_constants.go`** — `serverengine/game_constants.go:180`,
  `serverengine/arkhamhorror/content/map.go` — Two copies of the same adjacency data
  exist. `serverengine/game_server.go` uses the `game_constants.go` version;
  `serverengine/arkhamhorror/rules/movement.go` delegates to the `content` package.
  They are consistent today but will diverge in a four-neighbourhood expansion.
  **Remediation:** Remove `locationAdjacency` from `serverengine/game_constants.go` and
  have `serverengine/game_server.go::ValidateMovement` call the `content` package directly.
  Validate with `go test ./serverengine/...`.

---

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| `serverengine/common/runtime/UnimplementedEngine` — all methods return zero values | By design: satisfies `contracts.Engine` for placeholder game families. Documented, used by `eldersign`/`eldritchhorror`/`finalhour` modules. |
| `cmd/web_nowasm.go::NewWebCommand` always errors | By design: web command is only meaningful for `GOOS=js GOARCH=wasm` builds. `Hidden:true` on non-WASM. |
| All `return nil` / `return` bare returns in client UI code | Guards and early returns inside `Update()`/`Draw()` — idiomatic Ebitengine game loop patterns, not stubs. |
| Empty `doc.go` files in eldersign/eldritchhorror/finalhour sub-packages | Explicitly documented scaffolds per README §Unimplemented/Scaffolding Packages. Not a gap in the current scope. |
| `serverengine/arkhamhorror/engine.go` — 10-line file | Contains the `Engine` struct declaration (structural wrapper). Its minimal size is correct for its current migration-phase role. Flagged as partially wired, not as a stub. |
| `client/ebiten/render` placeholder atlas path | Correctly falls back to procedural colours when PNG assets are absent. The placeholder is designed fallback, not a missing implementation. |
| `serverengine/game_constants.go::DifficultyConfig` — 3 presets defined | All three presets are reachable via the `setdifficulty` action. The gap is the missing UI, not the server implementation. |
| `actions.CallbackSet.Trade` — passes two IDs but server resolves target from `action.Target` | `performTrade` validates co-location server-side; the callback design is intentional. |
| `BroadcastPayloadAdapter` interface defined twice (in `serverengine` and `serverengine/arkhamhorror`) | The serverengine copy is the active one; the arkhamhorror copy is for the planned migration. Duplicate interface is a migration artifact, not a contract conflict. |
