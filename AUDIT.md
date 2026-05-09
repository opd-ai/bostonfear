# IMPLEMENTATION GAP AUDIT — 2026-05-09

## Project Architecture Overview

BostonFear is a Go module (`github.com/opd-ai/bostonfear`, Go `1.24.1`) positioned as a rules-only multiplayer Arkham Horror engine with an interface-oriented server runtime, transport-neutral contracts, HTTP/WebSocket transport, monitoring endpoints, and Go/Ebitengine clients for desktop, WASM, and mobile. The intended ownership split is:

- `cmd`: Cobra/Viper CLI wiring for server, desktop, and WASM entrypoints.
- `transport/ws`: `net.Listener`/`net.Conn`-based HTTP + WebSocket route setup and upgrade handling.
- `monitoring` and `monitoringdata`: `/health` and `/metrics` payload shaping.
- `protocol`: shared JSON wire types used by server and clients.
- `serverengine`: current playable Arkham runtime and compatibility facade.
- `serverengine/common/*`: cross-game contracts and future shared primitives.
- `serverengine/arkhamhorror/*`: intended long-term home for Arkham-specific rules, phases, model, adapters, and scenarios.
- `serverengine/{eldersign,eldritchhorror,finalhour}`: registered future game-family roots.
- `client/ebiten/*`: current Go/Ebitengine client stack.

Primary evidence:

- `go-stats-generator analyze . --skip-tests --format json --sections functions,documentation,packages,patterns,interfaces,structs,duplication`
- `go build ./...`
- `go vet ./...`
- `go list ./...`

Baseline results:

- Build: clean.
- Vet: clean.
- Code statistics: 6,243 LOC, 181 functions, 393 methods, 146 structs, 15 interfaces, 30 packages, 116 files.
- Documentation coverage: 82.5% overall.
- Maintenance signal: `go-stats-generator` reported 14 unreferenced functions, but only one dead-structure finding survived false-positive filtering.

External research (brief): the public GitHub repository had no open issues, no open PRs, and no milestones/project boards. Upstream content confirms that the non-Arkham game modules are placeholders and that the architecture migration is still planned rather than complete.

Dependency scan: all direct `go.mod` dependencies appear in active code paths; no unused direct dependency removal candidate was identified.

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 1 | 0 | 1 | 0 | 0 |
| Dead Code | 1 | 0 | 0 | 0 | 1 |
| Partially Wired | 4 | 0 | 0 | 4 | 0 |
| Interface Gaps | 1 | 0 | 0 | 0 | 1 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package

Coverage below is an exported-surface completeness estimate derived from this audit, not test coverage. Doc-only scaffold packages with zero exported functions are omitted from the table and discussed in Findings.

| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|-------------------|-------------|-------|------|----------|
| `app` | 16 | 16 | 0 | 0 | 100% |
| `arkhamhorror` | 4 | 4 | 0 | 0 | 100% |
| `cmd` | 7 | 7 | 0 | 0 | 100% |
| `content` | 1 | 1 | 0 | 0 | 100% |
| `ebiten` | 30 | 30 | 0 | 0 | 100% |
| `eldersign` | 4 | 3 | 1 | 0 | 75% |
| `eldritchhorror` | 4 | 3 | 1 | 0 | 75% |
| `finalhour` | 4 | 3 | 1 | 0 | 75% |
| `mobile` | 2 | 2 | 0 | 0 | 100% |
| `monitoring` | 3 | 3 | 0 | 0 | 100% |
| `render` | 27 | 27 | 0 | 0 | 100% |
| `rules` | 1 | 1 | 0 | 0 | 100% |
| `runtime` | 17 | 16 | 1 | 0 | 94% |
| `serverengine` | 21 | 21 | 0 | 0 | 100% |
| `ui` | 129 | 129 | 0 | 0 | 100% |
| `ws` | 13 | 13 | 0 | 0 | 100% |

## Findings

### CRITICAL

- [ ] No retained critical gaps after Phase 3f false-positive filtering. Core gameplay, build, and vet expectations are currently satisfied.

### HIGH

- [ ] Module-owned Arkham runtime slices are still scaffolds while the executable path remains centralized in `serverengine` — `serverengine/arkhamhorror/module.go:66-67`, `serverengine/arkhamhorror/actions/doc.go:4-8`, `serverengine/arkhamhorror/adapters/doc.go:5-8`, `serverengine/arkhamhorror/model/doc.go:3-10`, `serverengine/arkhamhorror/phases/doc.go:4-7`, `serverengine/arkhamhorror/scenarios/doc.go:4-7`, `docs/MODULE_MIGRATION_MAP.md:27-32`, `docs/MODULE_MIGRATION_MAP.md:45-46` — `Module.NewEngine()` still returns `serverengine.NewGameServer()`, and every planned Arkham ownership slice is still marked `Planned` with no completed slices. This blocks the stated package-separation goal from `README.md` and leaves the Arkham module tree largely structural rather than authoritative. **Remediation:** implement slices `S1`-`S6` behind internal contracts, move action/rules/phase/state/broadcast logic into `serverengine/arkhamhorror/*`, keep parity tests in `serverengine`, then validate with `go test -race ./serverengine ./serverengine/arkhamhorror/...` and `go vet ./...`.

### MEDIUM

- [ ] The default `server` command registers three non-playable game modules and exposes them as normal runtime options — `cmd/server.go:57-60`, `README.md:188-194`, `serverengine/eldersign/module.go:42-44`, `serverengine/eldritchhorror/module.go:44-46`, `serverengine/finalhour/module.go:43-45`, `serverengine/common/runtime/unimplemented_engine.go:19-57` — selecting `eldersign`, `eldritchhorror`, or `finalhour` succeeds through module lookup but only fails after `Start()`/`HandleConnection()` return a not-implemented error. This is a partially wired feature because the CLI advertises runtime selection for engines that cannot run. **Remediation:** either hide these modules from the default registry until playable, or gate them behind an explicit experimental flag and fail module selection before server startup; add CLI coverage for unsupported-module behavior and validate with `go test ./cmd ./serverengine/common/runtime`.

- [ ] `scenario.default_id` is documented and templated but never reaches the runtime selection path — `README.md:145`, `README.md:159-162`, `config.toml:41-45`, `cmd/server.go:68-84`, `serverengine/game_server.go:112-117` — the server installs Nightglass content and then constructs `DefaultScenario` directly, while a repository-wide `rg` over runtime packages found no consumer for `scenario.default_id` or `default_id`. This blocks the documented content-loader fallback contract and leaves scenario selection effectively hard-coded. **Remediation:** thread `viper.GetString("scenario.default_id")` into Arkham module startup, resolve it against the content index before calling `newGameServerWithScenario`, add tests for valid, missing, and invalid IDs, and validate with `go test ./cmd ./serverengine/...`.

- [ ] `web.server` is declared as a configurable WASM override but the WASM client never reads it — `README.md:147`, `config.toml:6`, `config.toml:25-27`, `cmd/web_wasm.go:28`, `cmd/web_wasm.go:43-49` — the client always derives the URL from `window.__serverURL` or `window.location`, and there is no Viper binding or config read in the `web` command. This makes the documented config key inert. **Remediation:** add a `web.server` binding/read path in the WASM command, prefer configured value when present, retain `__serverURL` and browser-origin fallback behavior, add a resolver-focused test, and validate with `GOOS=js GOARCH=wasm go build ./cmd/web` plus a small command-level unit test.

- [ ] Asset-pipeline telemetry is collected in the renderer but never surfaced through the project's monitoring endpoints — `client/ebiten/render/asset_telemetry.go:34-39`, `client/ebiten/render/asset_resolver.go:110-152`, `docs/ASSET_PIPELINE_ROLLOUT.md:100-102`, `monitoring/handlers.go:25-75`, `cmd/server.go:109-110` — the code increments manifest/component/fallback/atlas counters and documents `AssetMetrics().LogSummary()` / `Snapshot()`, but `/health` and `/metrics` only expose server, connection, doom, memory, and GC data. This leaves the asset rollout observability plan partially wired. **Remediation:** add an asset-telemetry provider or export path, include the counters in `/health` and Prometheus output, add handler tests asserting those fields, and validate with `go test ./monitoring ./client/ebiten/render`.

### LOW

- [ ] Several `serverengine/common/*` packages are present only as doc stubs and are currently unreferenced by executable code — `serverengine/common/messaging/doc.go:1-5`, `serverengine/common/session/doc.go:1-5`, `serverengine/common/state/doc.go:1-5`, `serverengine/common/observability/doc.go:1-5`, `serverengine/common/monitoring/doc.go:1-5`, `serverengine/common/validation/doc.go:1-5`, `README.md:401` — import-graph inspection found no runtime imports for these packages. They are dead structural surface today and add maintenance noise until the planned extractions actually happen. **Remediation:** either remove these packages until an active slice moves into them, or migrate one concrete cross-game abstraction per package before continuing to advertise them; validate with `go list ./...` and an import-graph diff.

- [ ] Documentation repeatedly points readers to a missing `ROADMAP.md` file — `README.md:13`, `README.md:275`, `docs/CLIENT_SPEC.md:46`, `docs/CLIENT_SPEC.md:48`, `docs/CLIENT_SPEC.md:53-58`, `docs/CLIENT_SPEC.md:242`, `docs/RULES.md:211`, `client/ebiten/doc.go:124` — there is no `ROADMAP.md` in the repository root, so architectural and UX references dead-end. This blocks the educational readability goal and makes multiple scaffold notes unverifiable. **Remediation:** restore the referenced roadmap file or rewrite all `ROADMAP.md` references to existing planning sources (`PLAN.md`, `docs/MODULE_MIGRATION_MAP.md`, and package-local specs), then validate with `rg -n 'ROADMAP\\.md' README.md docs client serverengine cmd`.

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| `client/ebiten/host_status.go:3-5` empty native `updateHostStatus` | Intentional portability shim; DOM status updates are WASM-only and the native build has no corresponding host page element. |
| `cmd/web_nowasm.go:1-14` stub `web` command on non-WASM targets | Supported-platform guard, not unfinished code; it correctly rejects an impossible runtime on native Go builds. |
| Placeholder sprite fallbacks in `client/ebiten/render/atlas.go` and manifest loader | Documented alpha art strategy; gameplay still functions and the fallback behavior is explicit rather than hidden. |
| `go-stats-generator` report of 14 unreferenced functions | Many candidates were public API surface, build-tagged shims, or placeholder packages already documented as scaffolding; only one dead-structure finding remained after checking docs, tests, and call paths. |
- Remediation Checklist:
  - [x] Add display name to the session/join flow.
  - [x] Show display name with ID fallback in player panel.
  - [x] Use display names in event log and action results.

### [MEDIUM] Browser host claims "Connected" before gameplay connectivity exists
- File: `client/wasm/index.html`
- Category: State Visibility
- Player Goal At Risk: Trust connection status.
- Player Impact: The page can imply gameplay readiness even when the game server is offline.
- Problem: The host marks itself connected after WASM boot, not after the WebSocket session succeeds.
- Evidence: Status text changes to `Connected` immediately after WebAssembly instantiation.
- Fix: Distinguish client boot from server connection, or bind the label to actual client connection state.
- Validation: With the server offline, the host should never claim a connected game session.
- Remediation Checklist:
  - [x] Change initial success text to `Client loaded` or similar.
  - [x] Reflect actual game connection state from the Ebitengine client.
  - [x] Show retry/reconnect state in the host when applicable.

### [MEDIUM] Invalid actions produce little actionable recovery guidance
- File: `client/ebiten/state.go`, `client/ebiten/app/game.go`
- Category: Feedback
- Player Goal At Risk: Recover from an invalid action attempt.
- Player Impact: Failed local attempts feel ignored or inscrutable.
- Problem: The client tracks invalid reasons but only renders a retry counter.
- Evidence: Local state records invalid reasons, while the HUD shows only `Invalid retries`.
- Fix: Show human-readable error feedback and a next-step hint.
- Validation: Out-of-turn and invalid trade attempts should display distinct recovery guidance.
- Remediation Checklist:
  - [x] Surface last invalid reason in the HUD.
  - [x] Translate machine reasons into player-readable text.
  - [x] Pair each invalid message with a recovery hint.

### [MEDIUM] Touch action taps may also trigger camera movement
- File: `client/ebiten/app/scenes.go`, `client/ebiten/app/input.go`
- Category: Input
- Player Goal At Risk: Use touch actions without destabilizing the view.
- Player Impact: Tapping an action or board region may also orbit or toggle the camera.
- Problem: The same touch press can be interpreted by both gameplay input and camera gesture handling.
- Evidence: Scene update runs both action touch handling and camera touch handling from the same just-pressed touch stream.
- Fix: Consume touch input once or suppress camera gestures when the touch lands in an interactive gameplay region.
- Validation: Needs runtime validation on touch hardware; action taps should never change the camera.
- Remediation Checklist:
  - [x] Add touch-consumption or gesture-priority rules.
  - [x] Reserve camera gestures for non-interactive regions.
  - [x] Add a touch regression test or manual checklist.

### [MEDIUM] Character selection is mechanically opaque for first-time players
- File: `client/ebiten/app/scenes.go`
- Category: Onboarding
- Player Goal At Risk: Pick an investigator confidently.
- Player Impact: Role choice appears as six names with no explanation of playstyle or differences.
- Problem: The selection screen lacks role summaries or consequence framing.
- Evidence: The scene displays role names and counts only.
- Fix: Add short archetype summaries and a selected-state preview.
- Validation: A new player should be able to explain the difference between at least two archetypes using the UI alone.
- Remediation Checklist:
  - [x] Add one-line descriptions for each investigator archetype.
  - [x] Highlight the currently selected choice.
  - [x] Explain when the scene advances and what selection changes in play.

### [LOW] Text readability is constrained by small bitmap text and truncation
- File: `client/ebiten/app/text_ui.go`, `client/ebiten/app/game.go`
- Category: Layout
- Player Goal At Risk: Read onboarding, results, and logs comfortably.
- Player Impact: Important text can be small and cut off in dense HUD areas.
- Problem: The UI uses a tiny bitmap font and trims long content instead of wrapping it.
- Evidence: The text system uses `basicfont.Face7x13`, and multiple UI panels trim strings to width.
- Fix: Switch to a larger scalable face and wrap multi-sentence content.
- Validation: At 800×600, no critical instructional or result text should be truncated.
- Remediation Checklist:
  - [x] Replace the bitmap body font with a more readable scalable face.
  - [x] Wrap onboarding, event log, and result text.
  - [x] Recheck readability at the project minimum resolution.

## Player Journey Assessment

- Game loads: Partial
- First player connects: Partial
- All players ready: Fail
- First turn: Partial
- Action execution: Partial
- Action resolves: Fail
- Multiple turns: Partial
- Game end: Partial

## Category Status

- Discoverability: Findings present
- Onboarding: Findings present
- State Visibility: Findings present
- Feedback: Findings present
- Input: Findings present
- Layout: Findings present
- Performance: No concrete player-visible performance issue identified in this static pass
