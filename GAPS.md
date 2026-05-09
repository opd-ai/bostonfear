# Implementation Gaps — 2026-05-09

## Arkham Module Ownership Still Incomplete
- **Intended Behavior**: Arkham-specific rules, phases, models, scenarios, and protocol adapters should live under `serverengine/arkhamhorror/*`, with `serverengine` acting only as a temporary facade while migration slices close.
- **Current State**: `serverengine/arkhamhorror/module.go:66-67` still returns `serverengine.NewGameServer()`, while `actions`, `adapters`, `model`, `phases`, and `scenarios` remain doc-only scaffolds and `docs/MODULE_MIGRATION_MAP.md:27-32,45-46` shows every slice still `Planned` and none completed.
- **Blocked Goal**: The package-separation architecture described in `README.md` is not yet true in the executable path.
- **Implementation Path**: Start with slice `S1` from `docs/MODULE_MIGRATION_MAP.md`, move action dispatch and legality gates behind an internal Arkham interface, add parity tests in `serverengine`, then repeat for phases, dice/rules, model, scenarios/content, and broadcast shaping.
- **Dependencies**: Requires parity tests around current `serverengine` behavior before each slice move.
- **Effort**: large
- **Evidence**: `serverengine/arkhamhorror/module.go:66-67`; `serverengine/arkhamhorror/actions/doc.go:4-8`; `docs/MODULE_MIGRATION_MAP.md:27-32`; `docs/MODULE_MIGRATION_MAP.md:45-46`

## Placeholder Modules Exposed As Runtime Choices
- **Intended Behavior**: Module selection should offer runnable engines or fail earlier than server startup if an engine is intentionally unavailable.
- **Current State**: `cmd/server.go:57-60` registers `eldersign`, `eldritchhorror`, and `finalhour`, but each module returns `runtime.NewUnimplementedEngine(...)`, which only fails when `Start()` or session handling is attempted.
- **Blocked Goal**: The documented multi-engine startup path behaves like a supported feature even though three of the four options are placeholders.
- **Implementation Path**: Remove placeholder modules from the default registry, or add an `experimental`/`allow-placeholder-engines` gate and surface a pre-start validation error before listener startup. Add CLI tests covering selection behavior.
- **Dependencies**: None.
- **Effort**: small
- **Evidence**: `cmd/server.go:57-60`; `serverengine/eldersign/module.go:42-44`; `serverengine/eldritchhorror/module.go:44-46`; `serverengine/finalhour/module.go:43-45`; `serverengine/common/runtime/unimplemented_engine.go:19-57`

## `scenario.default_id` Never Reaches Runtime
- **Intended Behavior**: The server should honor `[scenario].default_id` from `config.toml`, fall back through the documented content index rules, and start the selected scenario.
- **Current State**: The key is documented in `README.md` and templated in `config.toml`, but startup flows directly from content installation to `serverengine.NewGameServer()` / `DefaultScenario`; a runtime-package search found no consumer for `scenario.default_id`.
- **Blocked Goal**: Scenario selection is advertised as configurable but is effectively fixed to the built-in default.
- **Implementation Path**: Read `viper.GetString("scenario.default_id")` during Arkham startup, resolve it against `serverengine/arkhamhorror/content/nightglass/scenarios/index.yaml`, pass the resolved scenario into `newGameServerWithScenario`, and add tests for valid, missing, and invalid configured IDs.
- **Dependencies**: Best done after or alongside the scenario/content migration slice so ownership stays consistent.
- **Effort**: medium
- **Evidence**: `README.md:145`; `README.md:159-162`; `config.toml:41-45`; `cmd/server.go:68-84`; `serverengine/game_server.go:112-117`

## `web.server` Config Is Documented But Ignored
- **Intended Behavior**: The WASM client should allow an explicit `web.server` override while preserving automatic same-origin defaults.
- **Current State**: `cmd/web_wasm.go:43-49` resolves the WebSocket URL only from `__serverURL` or `window.location`; there is no Viper binding or config read for the documented `web.server` key.
- **Blocked Goal**: Operators cannot configure the browser client endpoint through the advertised TOML key.
- **Implementation Path**: Bind `web.server` in the root/web command path, check it first in `resolveWebServerURL`, retain existing same-origin fallback behavior, and add a small resolver unit test.
- **Dependencies**: None.
- **Effort**: small
- **Evidence**: `README.md:147`; `config.toml:6`; `config.toml:25-27`; `cmd/web_wasm.go:28`; `cmd/web_wasm.go:43-49`

## Asset Telemetry Stops At In-Process Counters
- **Intended Behavior**: Asset-pipeline rollout telemetry should be visible through the project’s monitoring surface so manifest and fallback failures can be observed without local logs.
- **Current State**: The renderer increments `AssetMetrics()` counters, and docs describe reading them, but `/health` and `/metrics` handlers do not expose any asset-related values.
- **Blocked Goal**: The asset rollout plan cannot be operationally verified from the published monitoring endpoints.
- **Implementation Path**: Add an asset-telemetry snapshot to the monitoring provider contract or a dedicated client diagnostics export, include counters in `/health` JSON and Prometheus metrics, and add tests to lock the schema.
- **Dependencies**: Requires deciding whether client-side asset telemetry is exported through server monitoring, a client diagnostics route, or both.
- **Effort**: medium
- **Evidence**: `client/ebiten/render/asset_telemetry.go:34-39`; `client/ebiten/render/asset_resolver.go:110-152`; `docs/ASSET_PIPELINE_ROLLOUT.md:100-102`; `monitoring/handlers.go:25-75`

## Reserved Common Packages Are Still Structural Only
- **Intended Behavior**: `serverengine/common/*` packages should contain live cross-game primitives once the monolith is split.
- **Current State**: `messaging`, `session`, `state`, `observability`, `monitoring`, and `validation` contain only package comments and are currently unreferenced by executable code.
- **Blocked Goal**: The advertised shared-runtime modularization is broader than the actual codebase today, which increases navigation overhead for contributors.
- **Implementation Path**: Either remove these empty packages until an extraction is ready, or migrate one active abstraction per package and enforce the dependency direction in CI.
- **Dependencies**: Most naturally follows the Arkham migration slices.
- **Effort**: medium
- **Evidence**: `serverengine/common/messaging/doc.go:1-5`; `serverengine/common/session/doc.go:1-5`; `serverengine/common/state/doc.go:1-5`; `serverengine/common/observability/doc.go:1-5`; `serverengine/common/monitoring/doc.go:1-5`; `serverengine/common/validation/doc.go:1-5`

## Missing `ROADMAP.md` Breaks Documentation Contracts
- **Intended Behavior**: README, client specs, and rules docs should point to resolvable planning material that explains current priorities and future work.
- **Current State**: Multiple documents reference `ROADMAP.md`, but the file is not present in the repository root.
- **Blocked Goal**: Readers cannot verify scaffold notes, planning references, or educational guidance that depends on the missing roadmap.
- **Implementation Path**: Restore `ROADMAP.md` if it is a missing artifact, or rewrite each reference to the current planning documents (`PLAN.md`, `docs/MODULE_MIGRATION_MAP.md`, and package-local specs) and add a docs link check to CI.
- **Dependencies**: None.
- **Effort**: small
- **Evidence**: `README.md:13`; `README.md:275`; `docs/CLIENT_SPEC.md:46`; `docs/CLIENT_SPEC.md:242`; `docs/RULES.md:211`; `client/ebiten/doc.go:124`
     // The slice is copied; subsequent modifications to the input slice do not affect the server.
     // Example: SetAllowedOrigins([]string{"localhost:8080", "example.com"})
     func (gs *GameServer) SetAllowedOrigins(origins []string) {
     ```
  2. Document `HandleConnection()` parameters:
     ```go
     // HandleConnection manages a player session using conn.
     // reconnectToken: if non-empty, the server attempts to restore a disconnected player's session.
     // If the token is not found, a new player is registered.
     // conn: must be non-nil and readable/writable; caller is responsible for closing after return.
     func (gs *GameServer) HandleConnection(conn net.Conn, reconnectToken string) error {
     ```
  3. Document struct field constraints via GoDoc comments:
     ```go
     // Player represents an investigator in the game.
     // Resources: Health and Sanity must be in [1, 10]; values outside this range indicate defeat.
     // Clues must be in [0, 5]; attempting to assign Clues > MaxClues (5) is an error.
     // Location must be one of: Downtown, University, Rivertown, Northside.
     // ActionsRemaining is reset to 2 at the start of each turn; values < 0 indicate turn-over.
     type Player struct {
         ID                 string
         Location           Location
         Resources          Resources
         ActionsRemaining   int
         // ...
     }
     ```
  4. Consider adding validation methods if needed: `func (p *Player) Validate() error { ... }` (optional for v1+).

---

## Session Recovery & Late Joiner Scenarios Not Documented

- **API Element**: `GameServer.HandleConnection()`, `SnapshotHealth()`, `CollectConnectionAnalytics()`
- **Issue**: README states "late joiners enter the turn rotation automatically" and WebSocket protocol supports reconnection via tokens, but the API contract is not documented. Operators and integrators must infer behavior from code.
- **Consumer Impact**: Operators may misconfigure late-joiner support, fail to persist reconnect tokens, or overlook session persistence requirements. Support for this feature is implicit, not explicit in API.
- **Recommendation**:
  1. Add section to `GameServer` package comment:
     ```go
     // Session Recovery & Late Joiners:
     // When a player connects with HandleConnection(conn, reconnectToken), the server checks if the token was issued to a previous session.
     // If found, the player's session state (resources, location, actions) is restored and the player re-enters the turn rotation.
     // If not found, a new player is registered.
     // Reconnect tokens are single-use; the server issues a new token upon successful reconnection.
     // Late joiners (first-time connections) spawn at Downtown with default resources (Health 10, Sanity 10, Clues 0).
     ```
  2. Document `HandleConnection()` return value:
     ```go
     // Returns a reconnect token (if successful) that the client must persist and use for future reconnections.
     // The token is sent via ConnectionStatusMessage over the wire; client implementation stores it locally.
     ```
  3. Create example: `ExampleGameServer_SessionRecovery()` showing disconnect, reconnect with token, and state restoration.

---

## Game Module Package Documentation Missing

- **API Element**: `serverengine/arkhamhorror`, `eldersign`, `eldritchhorror`, `finalhour` packages
- **Issue**: Game module packages lack package-level comments explaining which game is implemented, which game-specific mechanics are supported, and how to instantiate the module (via `NewModule()` or `NewEngine()`).
- **Consumer Impact**: Operators cannot determine which module implements which game or switch between game variants via CLI without trial-and-error or code inspection.
- **Recommendation**:
  1. Add package comment to `serverengine/arkhamhorror/module.go`:
     ```go
     // Package arkhamhorror implements the core Arkham Horror 3rd Edition (AH3e) game rules.
     // 
     // Features:
     // - Four locations (Downtown, University, Rivertown, Northside) with interconnection restrictions.
     // - Six resources: Health, Sanity, Clues, Money, Remnants, Focus.
     // - Doom counter with Mythos phase events.
     // - Standard difficulty levels: Easy (Doom 3), Standard (Doom 5), Hard (Doom 7).
     // 
     // Usage:
     //   module := arkhamhorror.NewModule()
     //   engine, err := module.NewEngine()
     //   engine.SetAllowedOrigins([]string{"localhost:8080"})
     //   engine.Start()
     func ExampleNewModule() { ... }
     ```
  2. Similarly, add comments to eldersign, eldritchhorror, finalhour modules describing their game era and mechanics.
  3. Document what `NewModule()` returns: `// NewModule returns a registered GameModule for Arkham Horror 3e. Use module.Key() to get the game ID ("arkhamhorror").`

---

## Exposed Internal Synchronization Details

- **API Element**: `GameServer` struct fields: `wsWriteMu` (map of per-connection mutexes), `latencySamples` (ring buffer), `connectionQualities`, `playerSessions` (line 23–100 of game_server.go)
- **Issue**: Internal bookkeeping fields are public struct members. Consumers might inspect or depend on these fields, creating fragile coupling to implementation internals. Refactoring internals becomes a breaking change.
- **Consumer Impact**: Future refactoring of internal synchronization or metrics collection breaks consumer code that inspects these fields.
- **Recommendation**:
  1. Unexport internal-only fields: `wsWriteMu` → `wsWriteMu`, `latencySamples` → `latencySamples`, `latencyMu` → `latencyMu`, etc.
  2. Provide getters for metrics consumers need to access:
     ```go
     // LatencyPercentiles returns broadcast latency percentiles (p50, p95, p99) in milliseconds.
     // Computed from the last 100 broadcast operations.
     func (gs *GameServer) LatencyPercentiles() map[string]float64 {
         // Return computed result without exposing raw buffer
     }
     
     // ActivePlayerCount returns the current number of connected players.
     func (gs *GameServer) ActivePlayerCount() int {
         // Computed atomically
     }
     
     // ConnectionQualityFor returns connection quality metrics for a specific player.
     // Returns nil if player is not connected.
     func (gs *GameServer) ConnectionQualityFor(playerID string) *ConnectionQuality {
         // Read from internals under lock
     }
     ```
  3. This refactoring is backwards-compatible (existing code that reads fields will fail to compile; provide migration docs with getter function list).

---

## UI/UX Clarity Gaps — May 9, 2026

### Summary

The current browser-playable client has a working scene flow and baseline tutorial, but the live game screen still under-communicates core mechanics. The most important gaps are board readability, action discoverability, dice/result clarity, and touch-input parity.

### Priority Gaps

- **Board Readability Gap**
   - Issue: The board does not clearly label locations or visualize legal adjacency.
   - Impact: New players cannot reliably infer where they are or where they can move.
   - Concise Remediation Checklist:
      - [ ] Label every location directly on the board.
      - [ ] Highlight the active player location.
      - [ ] Show legal adjacent destinations for the active player.

- **Action Discoverability Gap**
   - Issue: The HUD shows a static, incomplete controls legend rather than current legal actions.
   - Impact: Players cannot tell what is available now, what costs resources, or why an action is blocked.
   - Concise Remediation Checklist:
      - [ ] Replace static hints with a dynamic available-actions panel.
      - [ ] Show disabled states and disable reasons.
      - [ ] Show resource costs and remaining actions per turn.

- **Dice and Outcome Feedback Gap**
   - Issue: Results omit explicit dice faces and success thresholds.
   - Impact: Investigate and ward outcomes are hard to trust or learn from.
   - Concise Remediation Checklist:
      - [ ] Render success, blank, and tentacle outcomes explicitly.
      - [ ] Show achieved versus required successes.
      - [ ] Attribute doom changes to tentacle results in the same feedback block.

- **Invalid Action Recovery Gap**
   - Issue: The client records invalid-action reasons but mostly exposes only a retry counter.
   - Impact: Players do not know how to recover from local action failures.
   - Concise Remediation Checklist:
      - [ ] Show the last invalid reason in player-readable text.
      - [ ] Pair each invalid message with a recovery hint.
      - [ ] Distinguish local validation failures from server-side rejection.

- **Touch Parity Gap**
   - Issue: Touch does not expose the full supported action set and may overlap with camera gestures.
   - Impact: Touch-first players have an inconsistent and potentially unstable control scheme.
   - Concise Remediation Checklist:
      - [ ] Add touch access to all supported actions.
      - [ ] Prevent action taps from triggering camera gestures.
      - [ ] Re-verify minimum tap target sizes after expanding the action surface.

- **Player Identity Gap**
   - Issue: Display names are captured locally but not promoted into shared multiplayer UI.
   - Impact: Turn order and event history use opaque player IDs.
   - Concise Remediation Checklist:
      - [ ] Send display names through the session/game protocol.
      - [ ] Prefer display names in player panel and event log.
      - [ ] Keep ID fallback only for debugging and recovery cases.

- **Connection-State Trust Gap**
   - Issue: The WASM host page reports `Connected` before gameplay connectivity exists.
   - Impact: Players can misread client boot as game-server readiness.
   - Concise Remediation Checklist:
      - [ ] Distinguish `client loaded` from `server connected`.
      - [ ] Mirror actual connection and reconnect state from the runtime client.
      - [ ] Keep host-level and in-game status text consistent.

- **Readability Gap**
   - Issue: Small bitmap text and truncation reduce clarity in onboarding, logs, and results.
   - Impact: Players can miss or misread important state changes.
   - Concise Remediation Checklist:
      - [ ] Replace the small bitmap face for primary HUD text.
      - [ ] Wrap long strings instead of trimming them.
      - [ ] Recheck readability at 800×600 and on small touch screens.

---

## SessionEngine Interface Size & Role Clarity

- **API Element**: `transport/ws.SessionEngine` interface
- **Issue**: Interface contract not documented in visible location. If large, new transport adapters must implement many methods. Role-based separation is unclear (game logic vs transport operations).
- **Consumer Impact**: Implementing alternative transports (TCP, in-process) is cumbersome. Mocking for testing requires implementing all interface methods.
- **Recommendation**:
  1. Locate SessionEngine interface definition and document it:
     ```go
     // SessionEngine represents the game server engine that the transport layer invokes.
     // Implementations must be safe for concurrent use by multiple WebSocket handlers.
     type SessionEngine interface {
         HandleConnection(conn net.Conn, reconnectToken string) error
         SetAllowedOrigins(origins []string)
         Start() error
         SnapshotHealth() HealthSnapshot
         CollectPerformanceMetrics() PerformanceMetrics
         // ...
     }
     ```
  2. Document each method's contract (parameters, error semantics).
  3. If interface is large (>5 methods), consider role-based interfaces:
     ```go
     type GameEngine interface {
         HandleConnection(conn net.Conn, reconnectToken string) error
         SetAllowedOrigins(origins []string)
     }
     
     type HealthChecker interface {
         SnapshotHealth() HealthSnapshot
     }
     
     type MetricsCollector interface {
         CollectPerformanceMetrics() PerformanceMetrics
         // ...
     }
     ```

---

## GameState & Player Struct Validation Not Explicit

- **API Element**: `protocol.GameState`, `protocol.Player`, `protocol.Resources` structs
- **Issue**: Struct fields are exported JSON members but lack validation constraints. Consumers constructing these structs manually (e.g., in tests or alternative transports) cannot enforce invariants (Health in [1, 10], Doom in [0, 12], Location one of four values).
- **Consumer Impact**: Manual state construction is error-prone. Testing setup may inadvertently create invalid states. Integration of alternative frontends requiring manual state construction is difficult.
- **Recommendation**:
  1. Add constraint documentation to struct comments:
     ```go
     // GameState represents the complete game snapshot sent to all clients.
     // 
     // Invariants:
     // - Doom must be in [0, 12]; game ends if Doom >= 12 (loss).
     // - CurrentPlayer must be a key in Players map or empty string (game not started).
     // - TurnOrder must be a permutation of Players keys (all players, no duplicates).
     // - Each Player's Location must be one of: Downtown, University, Rivertown, Northside.
     // - Each Player's Resources must satisfy: Health, Sanity in [1, 10] or [0, 10] if defeated; Clues in [0, 5]; Money, Remnants, Focus >= 0.
     type GameState struct { ... }
     ```
  2. For testing support, provide a `ValidateGameState()` function:
     ```go
     // ValidateGameState checks GameState invariants.
     // Returns nil if all invariants hold; otherwise returns a list of ValidationErrors.
     // Use this to validate GameState before sending to clients.
     func ValidateGameState(gs *GameState) []ValidationError { ... }
     ```
  3. Document in package comment that consumers building GameState should call ValidateGameState().

---

## No Documented API Stability Guarantee for v1

- **API Element**: Module declaration `github.com/opd-ai/bostonfear`, go.mod version `go 1.24.1`, no version tag, no CHANGELOG.md
- **Issue**: Project is v0 (pre-release) but no documented API stability expectations or breaking-change policy. No CHANGELOG or version history. Consumers don't know when/if the API will stabilize.
- **Consumer Impact**: Operators cannot plan for upgrades or predict breaking changes. Library consumers cannot commit to stable versions.
- **Recommendation**:
  1. Create `CHANGELOG.md` at repo root documenting:
     ```markdown
     # Changelog
     
     All notable changes to this project will be documented in this file.
     This project uses Semantic Versioning.
     
     ## [Unreleased]
     - Added: context.Context support to I/O-bound methods
     - Fixed: missing GoDoc comments on exported functions
     - Deprecated: GameServer.Start() (zero-arg); use GameServer.StartWithContext(ctx)
     
     ## [v0.1.0] - [Date]
     - Initial alpha release with stable WebSocket protocol.
     - Breaking changes expected; API not stable for v1.
     ```
  2. Add to README a section documenting API stability:
     ```
     ## API Stability
     
     This project is pre-v1 (alpha). The WebSocket protocol is stable,
     but the Go server API (GameServer, handlers, etc.) may change
     without notice before v1.0.0. Breaking changes will be documented
     in CHANGELOG.md.
     
     For stable upgrades, pin to specific commits or minor versions
     (e.g., go get github.com/opd-ai/bostonfear@v0.1.0).
     ```
  3. Tag releases: `git tag v0.1.0 && git push origin v0.1.0`

---

## Summary of Priority

| Priority | Count | Examples | Action |
|----------|-------|----------|--------|
| CRITICAL | 1 | Context.Context missing | Add params, update signatures, test |
| HIGH | 3 | Undocumented exports, no examples, error contract | Document all, create examples, add sentinels |
| MEDIUM | 5 | Concurrency docs, param constraints, late-joiner docs, module docs, exposed internals | Add comments, provide getters, refactor |
| LOW | 2 | No internal/ boundary, inconsistent receivers | Document decision, audit in next refactor |

**Estimated Effort to Address All Gaps:**
- CRITICAL: 4–6 hours (context.Context thread through codebase)
- HIGH: 8–12 hours (documentation and example functions)
- MEDIUM: 6–8 hours (comments, getters, refactor exposed fields)
- LOW: 2–4 hours (documentation and linting)
- **Total: 20–30 hours**

**Recommendation:** Address CRITICAL and HIGH gaps before v1.0.0 release announcement. Schedule MEDIUM and LOW improvements in next maintenance cycle.
