# Implementation Gaps — 2026-04-24

> **Scope**: Gaps between the project's stated goals (README.md, RULES.md,
> CLIENT_SPEC.md, ROADMAP.md) and the current implementation.
> Ordered by severity (HIGH → MEDIUM → LOW) then by proximity to core game goals.
>
> **Previous cycle**: The GAPS.md from 2026-06-01 declared "No actionable gaps remain"
> after resolving GAP-11 through GAP-21. The gaps below are newly discovered in this
> 2026-04-24 audit pass.

---

## GAP-22 — Doom Bar Scale Silently Dropped in Compositor

- **Intended Behavior**: `CLIENT_SPEC.md §4.4` and the doom counter description
  in `README.md §4.4` specify a visual doom bar that fills proportionally as doom
  rises (0–12). `enqueueDoomEffect` in `client/ebiten/app/game.go:167` already
  computes the correct `ScaleX` fraction: `fraction * (200.0 / 64)` to stretch
  a 64-px tile across a 200-px bar width.
- **Current State**: `Compositor.Flush()` (`client/ebiten/render/layers.go:70-78`)
  calls `r.atlas.DrawSprite(screen, cmd.Sprite, cmd.X, cmd.Y, cmd.Tint)` but passes
  no scale factors. `DrawSprite` only calls `op.GeoM.Translate` — no
  `op.GeoM.Scale` call exists. The `DrawCmd.ScaleX`/`ScaleY` fields are normalized
  in `Enqueue()` but never consumed in `Flush()`. The doom bar always renders as
  a fixed 64×64 tile regardless of doom level.
- **Blocked Goal**: Visual doom feedback (CLIENT_SPEC.md §4.4 doom bar); the
  ROADMAP.md §"Replace Placeholder Sprites" visual polish work.
- **Implementation Path**:
  1. Update `Atlas.DrawSprite` signature to accept `scaleX, scaleY float64`:
     `func (a *Atlas) DrawSprite(dst *ebiten.Image, id SpriteID, dx, dy, scaleX, scaleY float64, tint color.RGBA)`
  2. Add `op.GeoM.Scale(scaleX, scaleY)` before `op.GeoM.Translate` in `DrawSprite`.
  3. Update all call sites: `Compositor.Flush()` passes `cmd.ScaleX, cmd.ScaleY`;
     any direct `DrawSprite` callers pass `1.0, 1.0`.
  4. Write a new unit test `TestEnqueueScale_AppliedOnFlush` in `atlas_test.go` that
     enqueues a `DrawCmd{Sprite: SpriteDoomMarker, ScaleX: 2.0, ScaleY: 0.5}` and
     verifies the drawn region on the target image.
- **Dependencies**: None.
- **Effort**: Small (< 2 hours).

---

## GAP-23 — Kage Shaders Compiled but Never Rendered

- **Intended Behavior**: `CLIENT_SPEC.md §4.4` specifies: "doom vignette" that
  darkens/desaturates as doom rises. `ROADMAP.md §Priority 6` lists "Add shader
  compilation test (verify Kage shaders compile without errors)" as an open item,
  implying shaders should be active. `client/ebiten/render/shaders.go` defines
  `NewShaderSet()` that compiles `fog.kage`, `glow.kage`, and `doom.kage`, and
  `DrawDoomVignette()` that composites the doom vignette over the screen.
- **Current State**: `NewShaderSet()` is never called from `NewCompositor()`,
  `NewGame()`, or any `Draw()` method. `Compositor` has no `shaders` field.
  `DrawDoomVignette()` is never called from `game.go:Draw()`. All three shaders
  are embedded, parseable, and correct Kage source, but no frame ever executes them.
  The fog-of-war, glow pulse, and doom vignette effects are completely absent at runtime.
- **Blocked Goal**: CLIENT_SPEC.md §4.4 "doom vignette"; ROADMAP.md §Priority 2
  "Implement Kage doom vignette shader per CLIENT_SPEC.md §4.4".
- **Implementation Path**:
  1. Add `shaders *ShaderSet` field to `Compositor` (or `Game`).
  2. In `NewCompositor()` (or lazily on first `Flush()`), call `NewShaderSet()`;
     log and continue gracefully if it returns an error (headless/unsupported GPU).
  3. At the end of `Game.Draw()`, after `g.renderer.Flush(screen)`, call:
     `render.DrawDoomVignette(screen, g.renderer.Shaders(), float32(gs.Doom)/12.0)`
  4. Expose `Compositor.Shaders() *ShaderSet` accessor.
  5. Add a `TestShaderSet_Compiles` test (Xvfb, `requires_display` tag) that calls
     `NewShaderSet()` and asserts no error.
  6. Add `Compositor.Deallocate()` that calls `shaders.Deallocate()` to release GPU
     resources on shutdown.
- **Dependencies**: GAP-22 (scale fix) should be completed first for a clean diff,
  but this gap is independent.
- **Effort**: Small (2–3 hours).

---

## GAP-24 — `InvestigatorType` Never Set in Production; Character Selection Missing

- **Intended Behavior**: `CLIENT_SPEC.md §3` specifies a `SceneCharacterSelect`
  where players send `{"type":"playerAction","action":"selectInvestigator","target":"<name>"}`.
  Six distinct investigator archetypes are defined in `game_constants.go`
  (`DefaultInvestigatorAbilities`), each with unique resource costs and gameplay effects.
  RULES.md §"variable player powers through unique investigator abilities" describes
  this as a core AH3e mechanic.
- **Current State**: `registerPlayer` (`cmd/server/connection.go:70`) constructs
  every `Player` with `InvestigatorType: ""` (zero value). `performComponent`
  (`cmd/server/actions.go:215`) looks up `player.InvestigatorType`; when not found in
  `DefaultInvestigatorAbilities` it silently falls back to `InvestigatorSurvivor`.
  There is no `ActionSelectInvestigator` constant, `isValidActionType` does not
  accept `"selectInvestigator"`, and no handler exists for it. All six investigator
  archetypes are dead from a gameplay perspective — tests set `InvestigatorType`
  directly, but no player action can.
- **Blocked Goal**: CLIENT_SPEC.md §3 "Character Selection"; RULES.md §"variable
  player powers"; `performComponent` gameplay differentiation.
- **Implementation Path**:
  1. Add `ActionSelectInvestigator ActionType = "selectinvestigator"` to `game_constants.go`.
  2. Add it to `isValidActionType` in `game_server.go`.
  3. Implement `performSelectInvestigator(player *Player, target string) error` in
     `actions.go`: validate `target` is a known `InvestigatorType` key, reject if game
     phase is not `"waiting"`, set `player.InvestigatorType = InvestigatorType(target)`.
  4. Route through `dispatchAction` switch.
  5. In the Ebitengine client (`client/ebiten/app/game.go`), add a keyboard binding
     (e.g., `R`/`D`/`O`/`S`/`M`/`V` for each archetype) or a pre-game selection screen.
  6. Add `TestProcessAction_SelectInvestigator_Valid` and
     `TestProcessAction_SelectInvestigator_InvalidPhase` tests.
- **Dependencies**: GAP-27 (scene state machine) desirable but not required — the
  server-side fix can land independently.
- **Effort**: Medium (half day).

---

## GAP-25 — Scene State Machine Not Implemented in Ebitengine Client

- **Intended Behavior**: `CLIENT_SPEC.md §1` specifies a four-scene finite state
  machine: `SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver`, with
  defined transition conditions, a 60-second reconnect countdown overlay, a
  "Game Full" state, and a dedicated game-over screen with stats.
- **Current State**: `client/ebiten/app/game.go` implements a single monolithic
  `Game` struct. `Draw()` always renders the in-game board view. Win/lose conditions
  are shown only as text banners at the bottom of the board. There is no connection
  screen (the board renders immediately, even before WebSocket handshake completes),
  no character selection screen (blocked by GAP-24), no game-over screen with stats,
  and no reconnect countdown. The "Play Again" and "Quit" buttons from CLIENT_SPEC.md
  §6 do not exist.
- **Blocked Goal**: CLIENT_SPEC.md §1–2, §6; ROADMAP.md Phase 3 "scene state machine".
- **Implementation Path**:
  1. Define a `Scene` interface: `type Scene interface { Update() error; Draw(*ebiten.Image) }`.
  2. Implement `SceneConnect` — displays server address, shows connection status, auto-reconnects.
  3. Implement `SceneGame` — the existing game board rendering, extracted from `Game.Draw()`.
  4. Implement `SceneGameOver` — win/lose banner, stats (clues, doom, elapsed time),
     "Play Again" (transition back to `SceneConnect`) and "Quit" buttons.
  5. Add `currentScene Scene` field to `Game`; `Update()` and `Draw()` delegate to it.
  6. Transition logic: `SceneConnect` → `SceneGame` when `LocalState.Connected && gs.GamePhase != "waiting"`;
     `SceneGame` → `SceneGameOver` when `gs.WinCondition || gs.LoseCondition`;
     `SceneGameOver` → `SceneConnect` on "Play Again" key.
  7. `SceneCharacterSelect` can be deferred until GAP-24 is resolved.
  8. Add scene-transition tests in `app/game_test.go`.
- **Dependencies**: GAP-24 (character selection) for `SceneCharacterSelect` only.
- **Effort**: Large (1–2 days).

---

## GAP-26 — `recoverInvestigator` Dead in Production; Investigators Permanently Defeated

- **Intended Behavior**: RULES.md §"Investigator Defeat (health OR sanity = 0)"
  states investigators should become "lost in time and space" and eventually recover.
  `checkInvestigatorDefeat` sets `player.LostInTimeAndSpace = true` and
  `player.Defeated = true`. `recoverInvestigator` exists to reverse these flags.
- **Current State**: `recoverInvestigator` (`cmd/server/game_mechanics.go:59`) is
  defined and tested (`rules_test.go:572`) but is **never called from any production
  code path**. Once an investigator is defeated, `player.Defeated = true` forever.
  `validateActionRequest` rejects all actions with "player has been defeated and cannot
  take actions". `advanceTurn` skips defeated players. The investigator is effectively
  locked out for the rest of the game, contradicting the AH3e recovery rule.
- **Blocked Goal**: RULES.md §"Investigator Defeat/Recovery" (recovery path).
- **Implementation Path**:
  1. In `runMythosPhase()` (before restoring `GamePhase = "playing"`), iterate over
     all players and call `gs.recoverInvestigator(id)` for any player where
     `player.LostInTimeAndSpace && player.Connected`.
  2. Alternatively, reset stats to half-max in `recoverInvestigator` if not already
     done (currently `checkInvestigatorDefeat` resets on defeat, not on recovery;
     verify the intended reset timing).
  3. Add `TestInvestigatorAutoRecovery_AfterMythosPhase` to `rules_test.go`:
     defeat a player, trigger `runMythosPhase`, assert `Defeated == false`.
- **Dependencies**: None (server-only change).
- **Effort**: Small (< 1 hour).

---

## GAP-27 — `applyDifficulty` Not Wired to Any Client API

- **Intended Behavior**: RULES.md §"Modular Difficulty Settings" and `DifficultyConfig`
  in `game_constants.go` define three difficulty presets (easy/standard/hard) with
  different `InitialDoom` and `ExtraDoomTokens`. The game should allow the first
  player (or host) to select difficulty before starting.
- **Current State**: `applyDifficulty` (`cmd/server/game_mechanics.go:113`) is
  implemented and unit-tested (`game_mechanics_test.go:903`), but no WebSocket message
  type routes to it. `newGameServerWithScenario` does not call it. `GameState.Difficulty`
  is always `""` at runtime. `DefaultScenario.StartingDoom` is hardcoded to `0`
  regardless of what `DifficultyConfig["standard"]` specifies.
- **Blocked Goal**: RULES.md "Modular Difficulty Settings" compliance;
  CLIENT_SPEC.md §7 non-functional requirements table lists this as required.
- **Implementation Path**:
  1. Add `ActionSetDifficulty ActionType = "setdifficulty"` to `game_constants.go`.
  2. Add to `isValidActionType`.
  3. In `dispatchAction`, add `case ActionSetDifficulty:` that calls
     `gs.applyDifficulty(action.Target)` only when `gs.gameState.GamePhase == "waiting"`;
     return error otherwise.
  4. Add `F` keyboard binding in `client/ebiten/app/input.go` (cycles
     easy/standard/hard) or a pre-game UI element.
  5. Tests: `TestProcessAction_SetDifficulty_Waiting` (expect success),
     `TestProcessAction_SetDifficulty_Playing` (expect error),
     `TestProcessAction_SetDifficulty_Invalid` (expect error on unknown difficulty).
- **Dependencies**: None.
- **Effort**: Small (2–3 hours).

---

## GAP-28 — Quick-Chat Panel Unimplemented; Server Rejects `chat` Actions

- **Intended Behavior**: `CLIENT_SPEC.md §5` specifies a "Quick-Chat Panel" with
  six predefined phrases. Phrases are sent as
  `{"type":"playerAction","action":"chat","target":"<phrase>"}` and displayed in
  the event log. README.md §JSON Message Protocol lists `gameUpdate` as the output
  channel for chat entries.
- **Current State**: `isValidActionType` (`cmd/server/game_server.go:334`) does not
  include `"chat"`. Any chat message from a conforming client is rejected with
  `"invalid action type: chat"` and the server increments `errorCount`. No
  `ActionChat` constant exists. The Ebitengine client `keyBindings` and the legacy
  JavaScript client `game.js` do not include chat bindings.
- **Blocked Goal**: CLIENT_SPEC.md §5 "Player Communication".
- **Implementation Path**:
  1. Add `ActionChat ActionType = "chat"` to `game_constants.go`.
  2. Add to `isValidActionType`.
  3. Implement `performChat(playerID, phrase string) (string, error)` in `actions.go`:
     validate phrase is non-empty (max 200 chars); return `"success"` on acceptance.
     No resource cost; does not consume an action (decrement optional by design).
  4. Route in `dispatchAction`; the `gameUpdate` broadcast carries
     `event: "chat", result: phrase`.
  5. In Ebitengine client, add a collapsible panel or function-key bindings `F1`–`F6`
     for the six phrases defined in CLIENT_SPEC.md §5.
  6. Tests: `TestProcessAction_Chat_Valid`, `TestProcessAction_Chat_EmptyPhrase`.
- **Dependencies**: None.
- **Effort**: Small (2–3 hours).

---

## GAP-29 — `PLAN.md` Referenced but Does Not Exist

- **Intended Behavior**: README.md §Project Structure lists `PLAN.md` as
  "Implementation plan for current gaps + migration". Three test files reference it:
  `cmd/server/benchmark_test.go:3` (`PLAN.md Step 6`),
  `cmd/server/rules_test.go:3` (`PLAN.md Step M8`),
  `cmd/server/origin_test.go:1` (`ROADMAP Priority 5`).
- **Current State**: `PLAN.md` does not exist in the repository root (verified with
  `ls PLAN.md`). Contributors following the README's project structure map encounter a
  broken reference. Test package-doc comments reference step numbers that have no
  backing document.
- **Blocked Goal**: Contributor onboarding; test documentation traceability.
- **Implementation Path**:
  Option A (preferred): Create `PLAN.md` at repository root summarizing the current
  implementation phases (aligned with ROADMAP.md priorities), with steps that
  correspond to the numbers cited in test files. Minimal viable content: a numbered
  list with step descriptions.
  Option B: Remove `PLAN.md` from the README §Project Structure table and update
  the three test file comments to remove step-number references.
- **Dependencies**: None.
- **Effort**: Small (< 1 hour).

---

## GAP-30 — Four Unused Analytics Types Dead in `game_types.go`

- **Intended Behavior**: `game_types.go` defines `ConnectionAnalytics`,
  `SessionDistribution`, `PlayerSessionMetrics`, and `AlertThreshold`. Their field
  names and doc comments suggest they were intended to drive the dashboard and health
  endpoints.
- **Current State**: None of the four types are instantiated in any production code.
  The live analytics system uses `ConnectionAnalyticsSimplified` and
  `PlayerSessionMetricsSimplified` (defined in `metrics.go`). The four types in
  `game_types.go` appear to be unreferenced remnants of an earlier design that was
  refactored without cleaning up the type definitions.
- **Blocked Goal**: Code clarity; reduces misleading API surface.
- **Implementation Path**:
  Remove the four type definitions (`ConnectionAnalytics`, `SessionDistribution`,
  `PlayerSessionMetrics`, `AlertThreshold`) from `game_types.go`. Validate with
  `go build ./cmd/server/...` (should be clean) and confirm no test references them.
- **Dependencies**: None.
- **Effort**: Trivial (< 30 minutes).

---

## GAP-31 — `client/ebiten/input.go` Zombie File

- **Intended Behavior**: Per the file's own comment, it was superseded by
  `client/ebiten/app/input.go` and should no longer exist.
- **Current State**: `client/ebiten/input.go` exists with `//go:build ignore`, marking
  it excluded from compilation. It duplicates `InputHandler`, `NewInputHandler`,
  `keyBindings`, and `actionKey` — all of which exist identically in the active
  `client/ebiten/app/input.go`. It confuses directory listings, `grep` results,
  and contributor code searches.
- **Blocked Goal**: Code hygiene; contributor clarity.
- **Implementation Path**:
  Delete `client/ebiten/input.go`. Validate with
  `go build ./client/ebiten/...` (headless environment).
- **Dependencies**: None.
- **Effort**: Trivial (< 5 minutes).

---

## GAP-32 — RULES.md Compliance Table Severely Out of Date

- **Intended Behavior**: `RULES.md` contains an "Engine Implementation Status" table
  that is the canonical reference for AH3e compliance status. README.md and ROADMAP.md
  both refer contributors to this table. It should reflect the current implementation.
- **Current State**: The table marks the following as unimplemented or partial:
  - Action System: "⚠️ Partial (4 of 8 actions: Move, Gather, Investigate, Ward)" — **False**: all 12 action types implemented.
  - Dice Resolution: "⚠️ Partial (no focus token spend)" — **False**: `rollDicePool` fully implements focus spend and rerolls.
  - Mythos Phase: "❌ Not implemented" — **False**: `runMythosPhase`, `resolveEventEffect`, and `drawMythosToken` all exist.
  - Resource Management: "⚠️ Partial (health, sanity, clues only)" — **False**: all 6 resource types (Health, Sanity, Clues, Money, Remnants, Focus) implemented.
  - Encounter Resolution: "❌ Not implemented" — **False**: `performEncounter` with deck-based draw and 4 effect types.
  - Act/Agenda Deck: "❌ Not implemented" — **False**: `checkActAdvance`, `checkAgendaAdvance`, `rescaleActDeck` all present.
  - Investigator Defeat: "⚠️ Partial (no recovery)" — **True for recovery** (see GAP-26).
  - Scenario System: "❌ Not implemented" — **False**: `Scenario` type, `DefaultScenario`, `WinFn`/`LoseFn` hooks all present.
  - Modular Difficulty: "❌ Not implemented" — **Partial**: config present, no client API (see GAP-27).
  - Test coverage: all rows cite "❌ None" — **False**: 178+ passing tests cover all mechanics.
  The table's Gap Reference column points to "GAPS.md §4" entries that no longer exist.
- **Blocked Goal**: Contributor trust in documentation; accurate project status for
  external readers.
- **Implementation Path**:
  Update every ⚠️/❌ row to ✅ where implementation is confirmed. Mark Investigator
  Defeat/Recovery as "⚠️ Partial (defeat ✅, recovery not triggered — GAP-26)" and
  Modular Difficulty as "⚠️ Partial (config ✅, no client API — GAP-27)". Update
  test coverage citations to reference actual test function names from `rules_test.go`.
  Remove "GAPS.md §4" references (that section no longer exists).
- **Dependencies**: None (documentation change only).
- **Effort**: Small (< 1 hour).

---

## GAP-33 — `PacketLoss` Metric Structurally Inaccurate

- **Intended Behavior**: `ConnectionQuality.PacketLoss` (`cmd/server/dashboard.go:21`)
  is meant to track network packet loss and is used in `assessConnectionQuality` to
  classify a connection as "poor" when `PacketLoss > 0.05` (5%).
- **Current State**: `PacketLoss` is initialized to 0 and incremented by `0.1` in
  `updateConnectionQuality` only when `messageDelay > 200ms`. However `messageDelay`
  is computed as `time.Since(messageTime).Seconds()` — measuring how long ago the
  message arrived, not round-trip network latency. Under normal server load this
  value is always < 1 ms, so `PacketLoss` never increments and the "poor" quality
  classification is never triggered. The metric is broadcast to all clients in every
  `connectionStatus` message.
- **Blocked Goal**: Accurate connection quality reporting in the dashboard.
- **Implementation Path**:
  Option A (preferred): Replace `PacketLoss` with missed-pong tracking. In
  `startPingTimer`, record an outstanding ping. In `handlePongMessage`, clear it.
  If a ping timer fires without a matching pong, increment a miss counter;
  `PacketLoss = missCount / totalPingsSent`.
  Option B: Remove `PacketLoss` from `ConnectionQuality` entirely and restrict
  quality classification to `LatencyMs` alone (already accurately measured via
  ping-pong RTT in `handlePongMessage`).
- **Dependencies**: None.
- **Effort**: Small (1–2 hours for Option A; trivial for Option B).

---

## Summary Table

| Gap ID | Area | Severity | Effort | Dependencies |
|--------|------|----------|--------|--------------|
| GAP-22 | Doom bar scale dropped in Flush | HIGH | Small | None |
| GAP-23 | Kage shaders never instantiated | HIGH | Small | None |
| GAP-24 | InvestigatorType not set; no character selection | HIGH | Medium | GAP-25 (optional) |
| GAP-25 | Scene state machine missing | HIGH | Large | GAP-24 |
| GAP-26 | `recoverInvestigator` dead in production | MEDIUM | Small | None |
| GAP-27 | `applyDifficulty` not wired to API | MEDIUM | Small | None |
| GAP-28 | Quick-chat action rejected by server | MEDIUM | Small | None |
| GAP-29 | `PLAN.md` missing | MEDIUM | Small | None |
| GAP-30 | Four unused analytics types | LOW | Trivial | None |
| GAP-31 | `client/ebiten/input.go` zombie file | LOW | Trivial | None |
| GAP-32 | RULES.md compliance table out of date | LOW | Small | GAP-26, GAP-27 |
| GAP-33 | `PacketLoss` metric inaccurate | LOW | Small | None |

**Recommended implementation order**: GAP-26 → GAP-27 → GAP-28 → GAP-22 → GAP-23 →
GAP-24 → GAP-25 → GAP-29 → GAP-32 → GAP-30 → GAP-31 → GAP-33
