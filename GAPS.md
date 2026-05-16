# Implementation Gaps — 2026-05-16

This file provides a detailed gap specification and implementation roadmap for each
finding identified in `AUDIT.md`. Gaps are ordered by severity (HIGH → MEDIUM → LOW),
then by proximity to the project's core stated goals.

---

## GAP-01: Encounter Action Missing from Client (HIGH)

- **Intended Behavior**: The `encounter` action type is a first-class protocol action
  (`protocol.ActionEncounter = "encounter"`, `protocol/protocol.go:36`). The server
  implements `performEncounter` fully (`serverengine/actions.go:181`). The UI component
  enum references it (`client/ebiten/ui/components.go:130`). It is listed in the valid
  pregame-and-game action set (`serverengine/game_server.go:472`). Players should be able
  to trigger it from the desktop and WASM client.
- **Current State**: The client action menu builder (`client/ebiten/app/game.go`) does not
  call `add("Encounter", ...)`. The keyboard binding table (`client/ebiten/app/input.go:44`)
  has no entry for encounter. The `touchActionMap` (`input.go:17`) lacks `"encounter"`.
  Touch-based and keyboard-based input paths both silently omit the action.
- **Blocked Goal**: Complete action system (stated goal #1, 5 core mechanics); also,
  `performComponent` for some investigator archetypes internally calls `performEncounter`
  as a side-effect, so a player relying on the `component` action for an encounter draw
  gets the effect without realising it — the explicit `encounter` action is the intended
  user-facing path.
- **Implementation Path**:
  1. In `client/ebiten/app/game.go`, in `buildActionMenu()` (around line 750), add:
     ```go
     add("Encounter", "draw location encounter card", turnActive && remaining > 0)
     ```
  2. In `client/ebiten/app/input.go`, add to `keyBindings`:
     ```go
     {ebiten.KeyN, "encounter", ""},  // N — Encounter (draw location card)
     ```
  3. Add to `touchActionMap`:
     ```go
     "encounter": protocol.ActionEncounter,
     ```
  4. Add a detail function (or inline detail string) to the render pass if the
     `SceneGame.Draw()` annotates encounter-specific conditions (e.g., deck empty).
- **Dependencies**: None — server-side implementation is complete.
- **Effort**: Small (< 1 hour).

---

## GAP-02: `arkhamhorror/adapters` Package Migration Incomplete (HIGH)

- **Intended Behavior**: `serverengine/arkhamhorror/adapters` is documented as the layer
  that "translates between Arkham Horror-specific game events and the shared runtime
  contracts" (doc.go:1-8). It should decouple the game rules from network protocol
  serialization. `adapter.go` declares `arkhamBroadcastAdapter` as the intended concrete
  implementation of `BroadcastPayloadAdapter`. The README §Package Separation names this
  as an explicit responsibility separation goal.
- **Current State**: `adapter.go` contains only a struct declaration comment and an empty
  struct body. No methods are implemented. No production code imports the adapters package.
  The `BroadcastPayloadAdapter` used by `serverengine/game_server.go:219` is the
  `serverengine`-scoped copy of the interface, not the arkhamhorror-scoped one. The two
  interface definitions are structurally identical but the adapters package one goes unused.
- **Blocked Goal**: Package separation / modularization (README §Technical Implementation
  §Package Separation); enables testing of broadcast logic independent of the server facade.
- **Implementation Path**:
  1. In `serverengine/arkhamhorror/adapters/adapter.go`, implement the three
     `BroadcastPayloadAdapter` methods: `BuildGameStatePayload()`, `BuildPlayerActionPayload()`,
     `BuildDiceResultPayload()`, by adapting `protocol` types to JSON bytes.
  2. Add a constructor `NewArkhamBroadcastAdapter() *arkhamBroadcastAdapter`.
  3. In `serverengine/game_server.go`, replace the inline adapter struct (or nil adapter)
     with `gs.SetBroadcastAdapter(adapters.NewArkhamBroadcastAdapter())` during `NewGameServer()`.
  4. Ensure the `serverengine`-scoped `BroadcastPayloadAdapter` interface is also satisfied
     by the new adapter (or consolidate to the contracts package copy).
- **Dependencies**: GAP-03 (scenarios) is independent; no other gaps block this.
- **Effort**: Medium (~4-6 hours).

---

## GAP-03: `arkhamhorror/scenarios` Package Scaffold Blocks Multi-Scenario Support (HIGH)

- **Intended Behavior**: The README documents a content-loader fallback contract
  (docs §Default Scenario Content) with a 4-step precedence chain:
  `scenario.default_id` config → `index.yaml defaultScenarioId` → first enabled scenario
  → startup error. This implies a scenario registry that maps IDs to full content
  definitions. The `serverengine/arkhamhorror/scenarios` package is documented as owning
  "scenario templates: starting investigator count, initial location distribution, difficulty
  thresholds, act/agenda decks, and win conditions."
- **Current State**: The `scenarios` package contains only `doc.go`. The embedded
  Nightglass content directory (`serverengine/arkhamhorror/content/nightglass/`) exists
  with scenario YAML files, but no Go code in the `scenarios` package reads or parses
  them. `performSelectScenario` (serverengine/actions.go:456) creates a shallow copy of
  `DefaultScenario` and sets only its `.Name` field — no scenario-specific deck, location,
  or win-condition data is loaded. The 4-step fallback contract from the README is
  entirely absent from the code.
- **Blocked Goal**: Multi-scenario support; `selectscenario` action; content-loader
  contract documented in README; future Nightglass scenario playability.
- **Implementation Path**:
  1. Define `ScenarioTemplate` struct in `serverengine/arkhamhorror/scenarios/template.go`:
     fields for ID, Name, Description, InitialDoom, ActDeck, AgendaDeck,
     EncounterDecks, WinConditionClues, Enabled.
  2. Define `Index` struct and `LoadIndex(fs.FS, path string) (*Index, error)` function
     in `serverengine/arkhamhorror/scenarios/index.go` that parses the embedded YAML.
  3. Add a global registry or pass an `*Index` to `GameServer` at construction time.
  4. Update `performSelectScenario` to look up scenario by ID from the registry and call
     the real `SetupFn` or directly apply the scenario's deck data to `gs.gameState`.
  5. Implement the 4-step fallback chain in `cmd/server.go` during startup.
- **Dependencies**: `serverengine/arkhamhorror/content` (provides embedded FS — already
  complete); GAP-02 is independent.
- **Effort**: Large (~1-2 days).

---

## GAP-04: Six `common` Scaffold Packages Provide No Symbols (MEDIUM)

- **Intended Behavior**: `serverengine/common/{messaging,session,state,validation,observability,monitoring}`
  each have doc comments describing future cross-engine primitives: message encoding,
  session lifecycle, resource bounds, validation helpers, observability hooks.
  The README §Package Separation names these as the future home for shared logic
  extracted from the monolith.
- **Current State**: Every one of these packages contains only a `doc.go` with a
  `NOTE: scaffold. Implementation is deferred.` comment. None are imported anywhere.
  They contribute 6 empty packages to the `go list ./...` output without providing value.
- **Blocked Goal**: Not blocking any current functionality; blocking the cross-engine
  modularization roadmap.
- **Implementation Path** (committed):
  - **Implement now (roadmap-backed):** For each package, add the minimum viable exported type
    matching the doc description (e.g., `session.Token` string type with `Validate()`,
    `state.ResourceBounds` mirroring `model.ResourceBounds`, `validation.ActionChecker`
    interface). Wire each into `serverengine` to begin migration.
  - Track execution in `ROADMAP.md` Phase 1; do not resolve GAP-04 by deleting scaffold packages.
- **Dependencies**: None blocking this gap.
- **Effort**: Small per package if implementing stubs; trivial if removing.

---

## GAP-05: Three Game-Family Placeholder Modules Not Registered (MEDIUM)

- **Intended Behavior**: The README states `BOSTONFEAR_GAME=eldersign` (and eldritchhorror,
  finalhour) selects a different rules engine at startup. The `GameModule` contract
  `NewEngine()` should return a functional engine.
- **Current State**: All three modules return `runtime.NewUnimplementedEngine(name)` from
  `NewEngine()`. Their sub-packages (model, rules, adapters, scenarios) each contain only
  a `doc.go`. The modules are explicitly commented out in `cmd/server.go:68-70`. Selecting
  any of these via `BOSTONFEAR_GAME` causes `Start()` to return
  `"eldersign engine not implemented"` and crash the server.
- **Blocked Goal**: Multi-game-family selection (README §Selecting Game Module at Startup).
  Currently acknowledged as scaffold; becomes a gap when the feature is advertised.
- **Implementation Path**: For each module family, implement:
  1. A minimal `Engine` struct embedding the appropriate rule variations.
  2. `rules/` package: dice resolution for that game family (Elder Sign uses unique die
     faces, Eldritch Horror uses skill tests, etc.).
  3. Uncomment the `registry.MustRegister` calls in `cmd/server.go`.
- **Dependencies**: GAP-03 (scenario registry approach should be consistent across modules).
- **Effort**: Large per module (~1-3 days each).

---

## GAP-06: Prometheus `p90` Latency Metric Always Absent (MEDIUM)

- **Intended Behavior**: `contracts/engine.go:90` documents that `GetLatencyPercentiles()`
  returns keys `"p50", "p90", "p95", "p99"`. The Prometheus handler in
  `monitoring/handlers.go:244-246` checks for and emits the `p90` quantile series.
- **Current State**: `GameServer.GetLatencyPercentiles()` (`serverengine/metrics.go:256`)
  returns a map with only keys `"p50"`, `"p95"`, `"p99"`. The `"p90"` key is never set.
  The Prometheus scrape silently omits the `quantile="0.90"` series. A NOTE comment at
  `monitoring/handlers.go:247` acknowledges this mismatch but does not resolve it.
- **Blocked Goal**: Accurate latency monitoring (README §Performance Standards sub-500ms
  synchronization); Prometheus dashboard consumers expecting a p90 series will see gaps.
- **Implementation Path** (choose one):
  - **Option A:** Add p90 computation to `serverengine/metrics.go:GetLatencyPercentiles()`:
    compute the 90th percentile from `gs.broadcastLatencies` using the existing pattern
    (currently idx95 = `int(0.95 * n) - 1`), add `idx90 = int(0.90 * n) - 1`, and
    populate `result["p90"]`.
  - **Option B:** Remove the `p90` check from `monitoring/handlers.go:244-246` and update
    `contracts/engine.go:90` to document keys as `"p50", "p95", "p99"` only.
- **Dependencies**: None.
- **Effort**: Small (< 30 minutes).

---

## GAP-07: `arkhamhorror/engine.go` Struct Never Instantiated (MEDIUM)

- **Intended Behavior**: `serverengine/arkhamhorror/engine.go` defines `Engine` as a
  per-module runtime wrapper around `*serverengine.GameServer`. The intent is that the
  module boundary controls the executable ownership of the engine.
- **Current State**: `serverengine/arkhamhorror/module.go:NewEngine()` returns
  `serverengine.NewGameServer(...)` directly, bypassing the `Engine` wrapper struct.
  The `arkhamhorror.Engine` type is never instantiated anywhere in production code.
- **Blocked Goal**: Module-level method override (e.g., future Arkham-specific handling
  of `Start()` or `HandleConnection()`); architectural boundary enforcement.
- **Implementation Path**:
  1. In `module.go:NewEngine()`, change the return to:
     ```go
     return &Engine{GameServer: serverengine.NewGameServer(ctx, scenario, opt)}, nil
     ```
  2. Ensure `*Engine` satisfies `contracts.Engine` (it inherits all methods via embedding).
- **Dependencies**: None.
- **Effort**: Small (< 30 minutes).

---

## GAP-08: `transport/ws.SessionEngine` Duplicates `contracts.SessionHandler` (MEDIUM)

- **Intended Behavior**: `serverengine/common/contracts` defines the canonical engine
  interface hierarchy for all transports. `transport/ws` should consume the canonical
  interfaces, not define its own subset.
- **Current State**: `transport/ws/websocket_handler.go:26` defines a local `SessionEngine`
  interface with only `HandleConnection(net.Conn, string) error` and `SetAllowedOrigins([]string)`.
  This is a structural subset of `contracts.SessionHandler`. The transport layer is
  decoupled from the contracts package, creating interface duplication.
- **Blocked Goal**: Unified interface usage (README §Interface-based Design); a new
  transport implementation could implement `SessionEngine` without satisfying
  `contracts.SessionHandler`.
- **Implementation Path**:
  1. In `transport/ws/websocket_handler.go`, replace:
     ```go
     type SessionEngine interface { ... }
     ```
     with an import of `contracts` and use `contracts.SessionHandler` (or `contracts.Engine`)
     as the parameter type for `NewServer()` / `ServeWebSocket()`.
  2. Confirm that `*serverengine.GameServer` satisfies `contracts.SessionHandler` at the
     call site (it does, since it implements all three methods).
- **Dependencies**: None.
- **Effort**: Small (< 1 hour).

---

## GAP-09: `DifficultyConfig.ExtraDoomTokens` Field Defined but Never Consumed (LOW)

- **Intended Behavior**: AH3e difficulty scaling places extra doom tokens in the Mythos
  Cup at setup. `DifficultySetup.ExtraDoomTokens` (`serverengine/game_constants.go:206`)
  is defined with values 0 (easy), 1 (standard), 3 (hard).
- **Current State**: `applyDifficulty()` (`serverengine/game_mechanics.go:139`) reads
  only `cfg.InitialDoom`. `ExtraDoomTokens` is silently discarded. The Mythos Cup
  initialization does not vary by difficulty token count.
- **Blocked Goal**: Difficulty scaling fidelity (the standard/hard difficulty increases
  doom pressure from the Mythos Cup — this is absent).
- **Implementation Path**:
  1. In `serverengine/game_mechanics.go:applyDifficulty()`, after setting `InitialDoom`,
     add doom tokens to `gs.gameState.MythosCup` or an equivalent field:
     ```go
     for i := 0; i < cfg.ExtraDoomTokens; i++ {
         gs.gameState.MythosCup = append(gs.gameState.MythosCup, MythosTokenDoom)
     }
     ```
  2. Ensure `InitGameState` / `defaultMythosCup()` already populates the base token set
     so extras are additive.
- **Dependencies**: None.
- **Effort**: Small (~30 minutes).

---

## GAP-10: No Client UI for Difficulty Selection (LOW)

- **Intended Behavior**: Players should be able to select difficulty (easy/standard/hard)
  during pre-game setup. The server action `ActionSetDifficulty` is implemented and
  enforced. The `SceneCharacterSelect` scene is shown before game start.
- **Current State**: `client/ebiten/app/scenes.go:375` (`SceneCharacterSelect`) contains
  only an investigator selection form. No difficulty widget is present. Players cannot
  change difficulty via the client UI. Direct WebSocket message injection is the only path.
- **Blocked Goal**: Accessible difficulty selection for non-developer players.
- **Implementation Path**:
  1. Add a `difficulty` state field to `SceneCharacterSelect`.
  2. In `SceneCharacterSelect.Draw()`, render three buttons: Easy / Standard / Hard.
  3. In `SceneCharacterSelect.Update()`, on button press, send:
     ```go
     net.SendAction(protocol.ActionSetDifficulty, selected, 0)
     ```
  4. Highlight the active difficulty based on the reflected `gs.Difficulty` field.
- **Dependencies**: None.
- **Effort**: Small (~1 hour).

---

## GAP-11: Duplicate Location Adjacency Data (LOW)

- **Intended Behavior**: Location topology should be defined once, in the canonical
  `serverengine/arkhamhorror/content` package, and consumed by both movement validation
  and server initialization.
- **Current State**: `serverengine/game_constants.go:180` contains a `locationAdjacency`
  map literal that is also present in `serverengine/arkhamhorror/content/map.go`.
  `serverengine/game_server.go:ValidateMovement` uses the `game_constants.go` copy;
  `serverengine/arkhamhorror/rules/movement.go` uses the `content` copy. Both are
  currently identical but will diverge on a map expansion.
- **Blocked Goal**: Single source of truth for game topology (maintainability; blocked
  goal if new locations are added and only one copy is updated).
- **Implementation Path**:
  1. Remove `locationAdjacency` from `serverengine/game_constants.go`.
  2. Add an exported `AdjacencyMap()` function (or var) to `serverengine/arkhamhorror/content/`.
  3. Update `serverengine/game_server.go:ValidateMovement` to call
     `arkhamcontent.AdjacencyMap()`.
- **Dependencies**: None.
- **Effort**: Small (~30 minutes).

---

## GAP-12: `actions/interface.go` GameEngine Interface Is Dead Code (LOW)

- **Intended Behavior**: The `actions.GameEngine` interface (`serverengine/arkhamhorror/actions/interface.go`)
  was likely intended as a future injection target for the game-engine callbacks, enabling
  the `actions` package to be tested against a mock engine.
- **Current State**: No code accepts a `GameEngine` interface parameter. The actual dispatch
  pattern uses `CallbackSet` (concrete functions), which achieves the same testability goal
  without the interface. `GameEngine` is an unused type with no implementations and no
  callers.
- **Blocked Goal**: None — the `CallbackSet` approach fulfils the same goal.
- **Implementation Path**:
  1. Remove `serverengine/arkhamhorror/actions/interface.go`.
  2. Run `go build ./...` to confirm no dependents break.
  Alternatively, document that `GameEngine` is reserved for a future higher-level mock
  and add a `//nolint:unused` or suppress the unused-type linter warning.
- **Dependencies**: None.
- **Effort**: Trivial (< 5 minutes).
