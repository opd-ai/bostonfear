# Goal-Achievement Assessment and Roadmap

**Generated**: 2026-05-17  
**Codebase baseline**: 9,237 LOC · 35 packages · 131 files  
**Tool**: `go-stats-generator v1.0.0`, `go test -race`, `go vet`

---

## Project Context

- **What it claims to do**: A rules-only multiplayer engine for the Arkham Horror series of cooperative board games, featuring live WebSocket gameplay, cross-platform clients (desktop, WASM, mobile), and a pluggable game-family module system supporting Arkham Horror 3rd Edition today and three more game families in the future.
- **Target audience**: 1–6 concurrent players connecting to a shared Go server; also intermediate developers learning WebSocket/goroutine architecture via a real game codebase.
- **Architecture**:
  - `serverengine/` — core game rules, turn engine, dice, doom, win/lose conditions
  - `serverengine/arkhamhorror/` — AH3e-specific actions, phases, rules, content, scenarios
  - `serverengine/eldersign/`, `eldritchhorror/`, `finalhour/` — scaffolded future game modules
  - `serverengine/common/` — shared contracts, session, state, validation, observability
  - `transport/ws/` — WebSocket upgrade handler wrapping `net.Conn` / `net.Listener`
  - `client/ebiten/` — Go/Ebitengine game client (desktop + WASM; mobile via ebitenmobile)
  - `protocol/` — JSON wire schema shared by server and client
  - `monitoring/` — Prometheus `/metrics` and JSON `/health` HTTP handlers
- **Existing CI / quality gates**:
  - `ci.yml`: `go vet`, doc-coverage threshold (`scripts/check-doc-coverage.sh`), common-dep direction (`scripts/check-common-deps.sh`), `go test -race` (all packages, with Xvfb), benchmark run with hard 200 ms broadcast-latency gate
  - `mobile.yml`: Android AAR bind + emulator test flow; iOS xcframework bind
  - `soak.yml`: nightly `TestStressTest_6Players` for 15 minutes (cron `0 3 * * *`), dispatachable short (30 s) profile
  - `dependency-sweep.yml`: weekly `go list -m -u all` report
  - `security.yml`: present (contents not detailed here)

---

## Goal-Achievement Summary

| # | Stated Goal | Status | Evidence | Gap |
|---|-------------|--------|----------|-----|
| 1 | 5 core game mechanics (Location, Resources, Actions, Doom Counter, Dice) | ✅ Achieved | All implemented in `serverengine/` and `serverengine/arkhamhorror/`; tests pass | — |
| 2 | 1–6 concurrent players | ✅ Achieved | `MaxPlayers = 6` in `serverengine/game_constants.go:8`; enforced in connection handler | — |
| 3 | Late-join (join a game already in progress) | ✅ Achieved | Players enter turn rotation automatically; tested in integration suite | — |
| 4 | 2 actions per turn with validation | ✅ Achieved | `common/validation` ActionChecker + TurnChecker; tests in `serverengine/game_mechanics_test.go` | — |
| 5 | Sub-500 ms state broadcast to all clients | ✅ Achieved (exceeded) | CI enforces ≤200 ms via `BenchmarkBroadcastLatency`; threshold stricter than the 500 ms doc claim | — |
| 6 | 30-second inactivity timeout (server-side) | ✅ Achieved | Documented and implemented; doom increments + connection closes on idle | — |
| 7 | WebSocket client with exponential backoff reconnect (5 s → 30 s cap) | ✅ Achieved | `client/ebiten/net.go:86–143` implements 5 s initial, doubling, 30 s max | — |
| 8 | Token-based session reclaim on reconnect | ✅ Achieved | Server issues `reconnectToken`; client appends `?token=` on redial (`net.go:103`) | — |
| 9 | Win condition: 4 clues per investigator before doom reaches 12 | ✅ Achieved | Scenario-driven Act deck with clue thresholds; hard doom cap at 12 in `mythos.go:196` | — |
| 10 | Lose condition: doom reaches 12 / agenda exhausted | ✅ Achieved | `checkGameEndConditions` + `checkAgendaAdvance` in `serverengine/mythos.go` | — |
| 11 | Prometheus `/metrics` endpoint | ✅ Achieved | `monitoring/handlers.go` MetricsHandler; format verified in tests | — |
| 12 | JSON `/health` endpoint with corruption history | ✅ Achieved | HealthHandler with rolling state-corruption log | — |
| 13 | Desktop build (Linux, macOS, Windows) | ✅ Achieved | `cmd/desktop/main.go`; CI builds and runs under Xvfb | — |
| 14 | WASM build | ✅ Achieved | `cmd/web/main.go`; CI `GOOS=js GOARCH=wasm go build` succeeds | — |
| 15 | Mobile build (Android AAR / iOS xcframework) | ⚠️ Partial | CI builds both artifacts; device runtime **not tested in automated environment** (per README and `mobile.yml`) | Device-level functional validation missing |
| 16 | 1280×720 logical resolution | ⚠️ Partial | README line 200 claims "Logical 1280×720"; code at `client/ebiten/app/game.go:30–33` and `cmd/desktop.go:39` use 800×600 | Documentation-vs-implementation mismatch |
| 17 | Game art (real sprites for locations, investigators, tokens) | ⚠️ Partial | README explicitly flags "alpha — placeholder sprites" on Desktop and WASM; procedural color primitives used | Acknowledged; no IP-safe art present |
| 18 | Multi-game-family support (Elder Sign, Eldritch Horror, Final Hour) | ⚠️ Partial | Three modules register and run; all three delegate entirely to shared `serverengine.GameServer` with no game-specific rules | Scaffolded; Arkham Horror rules execute regardless of selected module |
| 19 | `ROADMAP.md` file | ❌ Missing | README lines 13 and 288 link to `ROADMAP.md`; file is absent from the repository | Broken reference |

**Overall: 14 / 19 goals fully achieved; 4 partial; 1 missing**

---

## Complexity and Test Coverage Findings

These findings do not represent new requirements — they are risk signals for goals already claimed. Addressed in the roadmap only where they measurably threaten a stated goal.

| Item | Metric | Value | Risk |
|------|--------|-------|------|
| `SceneConnect.Draw` complexity | overall score | 33.5 (cyclomatic 25) | High — 157-line rendering function is the player's most-seen screen; bugs here break onboarding UX |
| `reconnectLoop` complexity | overall score | 22.8 (cyclomatic 16) | Medium — complex branch graph on the reconnect path; reconnect reliability is a stated goal |
| `RunMythosPhase` / `AdvanceTurn` / `DispatchAction` / `processActionCore` | overall > 19 | 4 functions | Medium — core turn-loop; test coverage at package level is 86.4% but sub-package (arkhamhorror/actions, arkhamhorror/phases) is 0% direct |
| `arkhamhorror/rules` line coverage | 6.7% | — | High — stated rules correctness goal depends on this being tested |
| `arkhamhorror/actions`, `arkhamhorror/phases`, `arkhamhorror/model`, `arkhamhorror/adapters` | 0% direct | — | Medium — exercised via integration tests in parent package; no isolated unit tests |
| `serverengine/common/runtime` | 25.5% | — | Low — module registry and `UnimplementedEngine`; mostly compile-time checked |
| Overall doc coverage | 82.3% | CI threshold enforced | Healthy; passing CI gate |
| Duplication ratio | 0.36% (66 lines) | 5 clone pairs | Negligible |

---

## Roadmap

Items are ordered by how much closing the gap advances the project's stated goals. Effort estimates are for a single developer familiar with the codebase.

---

### Priority 1 — Fix broken ROADMAP.md reference (Goal 19)
*Effort: 2–3 hours. Unblocks: documentation integrity; contributors & users who follow the link.*

The README references `ROADMAP.md` as the authoritative source for module timelines and phased development. The file does not exist. This is the item that converts from "❌ Missing" to "✅ Achieved" with the least effort of any item on this list.

- [x] Create `ROADMAP.md` at the repository root documenting the four-phase module plan, extracting dates and effort estimates already stated in ADR 003:
  - Phase 1 (Complete): Arkham Horror 3rd Edition — all five mechanics, 1–6 players, late-join, token reconnect
  - Phase 2 (Planned, ~4–6 weeks): Elder Sign — dice-tower placement, unique encounter system, scenario templates
  - Phase 3 (Planned, ~8–10 weeks): Eldritch Horror — global map, mysteries, ancient one system
  - Phase 4 (Planned, ~6–8 weeks): Final Hour — simultaneous action programming, countdown tokens, objectives
  - Phase 5 (Future): Arkham content expansions, additional investigator roster, scenario packs
- [x] For each planned phase, list the key files to create (`rules/`, `adapters/`, `scenarios/`, `model/` per module) so contributors know where to start.
- [x] Verify that both cross-references in `README.md` (lines 13 and 288) resolve correctly after the file exists.

**Validation**: `ls ROADMAP.md` succeeds; both links in README render as valid anchors in a Markdown previewer; ROADMAP.md covers all four future module phases with effort estimates.

---

### Priority 2 — Reconcile documented vs. actual client resolution (Goal 16)
*Effort: 30 minutes (documentation fix) or 4–6 hours (code upgrade). Recommended: documentation fix.*

`README.md` line 200 and `docs/CLIENT_SPEC.md` document "Logical 1280×720 resolution scaled to any display." Every `SetWindowSize` call (`cmd/desktop.go:39`, `cmd/web_wasm.go:35`) and `Game.Layout()` return (`client/ebiten/app/game.go:30–33`) uses 800×600. All UI rectangles, action grids, and HUD insets are also hard-coded to 800×600. There is no code path that activates 1280×720.

**Option A (recommended): Update documentation to match code**
- [x] `README.md` line 200: replace "Logical 1280×720" with "800×600 logical"
- [x] `docs/CLIENT_SPEC.md`: update all resolution references to 800×600
- [x] `client/ebiten/app/doc.go` line ~20: if it says "1280×720 logical", correct it to "800×600 logical"
- [x] `Makefile` `test-display` target: replace `Xvfb :99 -screen 0 1280x720x24` with `1024x768x24` or `800x600x24` (purely cosmetic; Xvfb resolution does not bind the Ebitengine logical size)

**Option B (not recommended): Upgrade code to 1280×720**
- Requires recomputing every location rect, action cell, panel bounds, and touch inset in `game.go` lines 35–630 and `scenes.go`.
- Risk of regressions in mobile safe-area inset calculations.
- Only worth pursuing if 1280×720 is required for a specific feature (e.g., higher-resolution sprite atlas).

**Validation**: `grep -r "1280\|720" README.md docs/CLIENT_SPEC.md` returns zero hits (Option A); or `go test -race -tags=requires_display ./client/...` passes without layout failures (Option B).

---

### Priority 3 — Add direct unit tests for arkhamhorror sub-packages (Goal 1 quality risk)
*Effort: 1–2 days. Unblocks: rule correctness confidence for the five stated game mechanics.*

The core `serverengine` package achieves 86.4% line coverage via integration tests. However, the sub-packages that own the game rules have no direct coverage:

| Package | Direct Coverage |
|---------|----------------|
| `serverengine/arkhamhorror/rules` | 6.7% |
| `serverengine/arkhamhorror/actions` | 0% |
| `serverengine/arkhamhorror/phases` | 0% |
| `serverengine/arkhamhorror/model` | 0% |
| `serverengine/arkhamhorror/adapters` | 0% |

The risk is real: `DispatchAction` (complexity 19.2, 88 lines) and `RunMythosPhase` (complexity 21.0, 61 lines) are on the critical game loop path. Bugs in them require disassembling the integration test harness to diagnose.

- [x] `serverengine/arkhamhorror/actions/perform_test.go`: unit-test `DispatchAction` for each of the four action types (Move, Gather, Investigate, Cast Ward) in isolation, mocking the `GameServer` via the `contracts.Engine` interface.
- [x] `serverengine/arkhamhorror/phases/mythos_test.go`: table-driven tests for `RunMythosPhase` covering: doom increment from Tentacle result, clean pass with no Tentacles, and threshold behavior at the doom cap boundary.
- [x] `serverengine/arkhamhorror/phases/mythos_test.go` (continued): `AdvanceTurn` — verify correct player rotation for 1, 3, and 6 players; verify action counter resets.
- [x] `serverengine/arkhamhorror/rules/` — extend existing 6.7% coverage to ≥70%: focus on adjacency validation and resource cost enforcement for Cast Ward.

**Validation**: `go test -race -cover ./serverengine/arkhamhorror/...` reports ≥70% for `actions`, `phases`, and `rules`.

---

### Priority 4 — Reduce complexity in SceneConnect.Draw and reconnectLoop (Goals 7, 8 quality risk)
*Effort: 1 day. Unblocks: reconnect reliability; onboarding UX robustness.*

`SceneConnect.Draw` (complexity 33.5, 157 lines, `client/ebiten/app/scenes.go:105`) is the player's first screen. It has 25 cyclomatic branches — this is the most complex single function in the codebase by a wide margin. `reconnectLoop` (complexity 22.8, 55 lines) is the stated "automatic reconnection with exponential backoff" implementation; its 16 cyclomatic branches make it fragile to extend.

- [x] **`SceneConnect.Draw`**: Extract the five discrete rendering regions (server-URL input, display-name input, connection status panel, error text, connect button) into separate `drawXxx(screen *ebiten.Image)` methods on `*SceneConnect`. The composite `Draw` becomes a 10-line orchestrator. Complexity target: <10 overall score.
- [x] **`reconnectLoop`**: Extract the backoff timer logic into a `nextBackoff(current, max time.Duration) time.Duration` pure function (pure → easy to unit test). Extract the URL-parameter appending into a named helper already partially done as `appendQueryParam`. Complexity target: <12 overall score.

**Validation**: `go-stats-generator analyze . --skip-tests` reports `SceneConnect.Draw` and `reconnectLoop` complexity below 12; `go test -race ./client/ebiten/...` passes (no display required for unit tests).

---

### Priority 5 — Verify mobile device runtime, not just build artifact (Goal 15)
*Effort: 1–3 days (CI setup) or defer with explicit README disclaimer.*

Mobile is claimed as "Alpha (touch input parity verified; device runtime **not yet tested in automated environment**)". The CI (`mobile.yml`) builds the AAR and xcframework and runs Go unit tests that do not exercise the Ebitengine rendering pipeline. It does **not** confirm that the game loop, WebSocket connection, or touch input actuall work on a physical device or high-fidelity emulator.

This does not block core gameplay — the server, desktop and WASM clients are all fully functional. However, it means the "mobile support" claim is build-artifact-only.

**Option A: Extend CI to use Android emulator with game loop smoke test**
- [x] In `mobile.yml`, add a step that launches an Android emulator (API 29), installs the bound AAR as a minimal test app (no Xcode needed), and verifies a WebSocket handshake with a local game server by checking for `connectionStatus` message receipt within 10 seconds.
- [x] Add analogous iOS simulator test using `xcrun simctl` on a macOS runner.

**Option B: Downgrade the README claim (low-effort, accurate)**
- [x] `README.md` build-targets table: revise Mobile status from "Alpha (touch input parity verified)" to "Alpha (library build verified; device gameplay not yet CI-validated)" to prevent false expectations.
- [x] Add a note to `docs/MOBILE_VERIFICATION_RUNBOOK.md` linking to what additional validation is needed before calling the mobile client "device-tested".

**Validation** (Option A): Mobile CI job passes without skipping emulator steps. (Option B): The README no longer overstates the validated scope.

---

### Priority 6 — Implement IP-safe procedural or abstract art (Goal 17)
*Effort: 1–2 weeks. Unblocks: making the game visually distinguishable between game states.*

Every client platform runs with placeholder sprites — solid-color rectangles for locations, investigators, tokens, and UI chrome. The game is fully functional, but visual feedback for location identity, resource levels, and doom state depends on colored boxes rather than meaningful art. No Arkham Horror IP art can be included.

- [x] **Location art**: Generate distinct procedural tile backgrounds for each of the four locations (Downtown, University, Rivertown, Northside) using `ebiten.NewImage` + Kage shader fills or `golang.org/x/image/draw` patterns. Each location should have a unique hue and texture pattern that is legible at 800×600 without any copyrighted imagery.
- [x] **Resource meters**: Replace the current rectangle-based Health/Sanity/Clues readout with segmented bar sprites generated programmatically (e.g., stamped glyphs from the Ebitengine font package).
- [x] **Doom counter**: Render the doom track as a radial arc or segmented 12-step ring rather than a number label, making the approaching end-state viscerally readable.
- [x] **Investigator tokens**: Per-player colored circular tokens with unicode investigator initials (e.g., "R.C." for Roland Carter) rather than blank squares.
- [x] Update `client/ebiten/render/` asset pipeline and sprite atlas to include the new procedurally generated assets; update `COMPONENT_ASSET_INVENTORY.md`.

**Validation**: Screenshots generated by `game_test.go` (under `requires_display` build tag) show visually distinct location tiles; resource bars update proportionally as Health/Sanity values change; doom ring fills at documented rate during a scripted game session.

---

### Priority 7 — Implement Elder Sign module (Goal 18, Phase 2)
*Effort: 4–6 weeks. Unblocks: the multi-game-family architecture claim; true module pluggability.*

All three non-Arkham modules (`eldersign`, `eldritchhorror`, `finalhour`) are identical wrappers that run Arkham Horror rules when selected via `BOSTONFEAR_GAME=eldersign`. The module architecture is complete and ready — `contracts.Engine` interface, module registry, and per-module subpackage structure all exist. Elder Sign has the smallest delta from Arkham Horror and is the right starting point.

Key mechanics differences in Elder Sign (vs. Arkham Horror 3rd Edition):
- **Dice tower**: Roll up to 6 custom dice; lock successes between attempts (multiple roll phases per action, not single roll)
- **Museum locations**: Fixed grid of museum rooms rather than city neighborhoods; movement is room-based
- **Adventure/monster cards**: Draw and resolve encounter cards per room; no Act/Agenda deck
- **Elder Sign tokens**: Win by collecting Elder Sign results from encounters before the doom track completes

**Implementation path (per existing scaffold structure)**:
- [ ] `serverengine/eldersign/model/`: Define Elder Sign domain types — `Room`, `AdventureCard`, `MonsterCard`, `DicePool`, `LockResult`. Do not reuse Arkham `Location` or `Player` types directly; model the differences.
- [ ] `serverengine/eldersign/rules/`: Implement the dice-tower resolution loop (lock mechanic), room entry cost validation, encounter card draw and resolve.
- [ ] `serverengine/eldersign/scenarios/`: Implement at least one scenario (e.g., "Enter the Gate") with museum room layout and a starting adventure deck.
- [ ] `serverengine/eldersign/adapters/`: Implement the `contracts.Engine` broadcast adapter to serialize Elder Sign state as `gameState` messages (reusing `protocol/` wire types where possible; extending for ES-specific fields under an optional `gameSpecific` envelope).
- [ ] `serverengine/eldersign/module.go`: Wire the above into `NewEngine()` by replacing the pass-through `&Engine{GameServer: serverengine.NewGameServer()}` with an Elder Sign-specific engine that overrides action dispatch and phase progression.
- [ ] Tests: Mirror Arkham's `serverengine/game_mechanics_test.go` structure with Elder Sign scenarios; aim for ≥70% coverage from day one.

**Validation**: `BOSTONFEAR_GAME=eldersign go run . server` starts a server that responds to `playerAction` messages with Elder Sign dice-tower outcomes, not Arkham Horror outcomes; `go test ./serverengine/eldersign/...` reports ≥70% coverage.

---

## Non-Goals (Out of Scope for This Roadmap)
The following items were evaluated and explicitly excluded because they do not trace to any stated project goal:
- **Golint / staticcheck naming violations** (`FeedbackQueue`, `InputMapper`, etc.): These are package-stuttering patterns flagged by `go-stats-generator`; they are not bugs and the project does not claim a zero-lint standard beyond `go vet`.
- **Placement refactors** (163 suggestions from `go-stats-generator`): All are "move to package X for better cohesion" and none affect runtime behavior or stated goals.
- **Duplication elimination** (5 clone pairs, 66 lines, 0.36% ratio): Below any meaningful threshold.
- **Dependency upgrades**: Both `gorilla/websocket` (v1.5.3) and `ebiten/v2` (v2.9.9) are already on the latest published release as of 2026-05-17. No pending breaking changes or known vulnerabilities.
- **Transport encryption**: Explicitly out of scope per task instructions; handled by infrastructure.

---

## Summary Table

| Priority | Goal(s) Affected | Current Status | Effort | Impact |
|----------|-----------------|----------------|--------|--------|
| 1 | ROADMAP.md (Goal 19) | ❌ Missing | 2–3 h | Fixes broken link; contributor clarity |
| 2 | Resolution documentation (Goal 16) | ⚠️ Partial | 30 min | Accurate spec for clients/contributors |
| 3 | Rule correctness test coverage (Goal 1) | ✅ at integration level, 0–6.7% at unit level | 1–2 d | Reduces bug risk on stated core mechanics |
| 4 | Complexity / reconnect reliability (Goals 7, 8) | ✅ functional, high cyclomatic | 1 d | Reduces regression risk on reconnect path |
| 5 | Mobile device validation (Goal 15) | ⚠️ build-only | 1–3 d | Closes gap between build and runtime claim |
| 6 | Visual art — placeholder sprites (Goal 17) | ⚠️ acknowledged alpha | 1–2 w | Improves playability; no IP content allowed |
| 7 | Elder Sign module (Goal 18, Phase 2) | ⚠️ scaffolded | 4–6 w | First real multi-game-family implementation |
