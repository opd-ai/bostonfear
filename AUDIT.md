# IMPLEMENTATION GAP AUDIT — 2026-04-24

## Project Architecture Overview

**bostonfear** is an Arkham Horror–themed cooperative multiplayer web game with a
Go WebSocket server and two client tiers: a legacy HTML/JS canvas client (deprecated)
and an in-progress Ebitengine native/WASM client.

### Package Responsibilities

| Package | Role |
|---------|------|
| `cmd/server` | WebSocket server, game state, 12 AH3e actions, Mythos Phase, observability |
| `client/ebiten` | Ebitengine game core: LocalState mirror, NetClient, session persistence |
| `client/ebiten/app` | Ebitengine `ebiten.Game` implementation, input handling |
| `client/ebiten/render` | Layered sprite compositor, texture atlas, Kage shaders |
| `cmd/desktop` | Desktop (Linux/macOS/Windows) entrypoint |
| `cmd/web` | WASM entrypoint |
| `cmd/mobile` | Mobile (iOS/Android) binding scaffolding |

### Stated Goals (from README.md and ROADMAP.md)
1. 5 core game mechanics: Location, Resources, Actions, Doom, Dice — **all present in server**
2. 1-6 concurrent players with join-in-progress support — **implemented**
3. Real-time state sync < 500 ms — **implemented**
4. Interface-based design (`net.Conn`, `net.Listener`, `net.Addr`) — **implemented**
5. Ebitengine client: desktop + WASM builds — **compile; alpha placeholder sprites**
6. Performance monitoring: `/dashboard`, `/metrics`, `/health` — **implemented**
7. AH3e rules compliance — **server engine fully implemented**
8. Session persistence and reconnect token — **implemented**

### Baseline Metrics (go-stats-generator, 2026-04-24)

| Metric | Value |
|--------|-------|
| Total LoC | 2,578 |
| Functions / Methods | 53 / 145 |
| Packages | 5 |
| Avg function length | 14.7 lines |
| Avg cyclomatic complexity | 4.4 |
| High complexity (> 10) | 0 functions |
| Documentation coverage | 95.9% |
| Dead code (unreferenced) | 50 functions (includes test helpers) |
| Circular dependencies | 0 |

`go build ./cmd/server/...` → **clean**  
`go vet ./cmd/server/...` → **clean**  
`go test -race ./cmd/server/...` → **pass (7.6 s)**  
`go test -race ./client/ebiten/...` → **pass (1.0 s)**  
`go build ./...` → GLFW/X11 header error on this box when system GL/X11 development packages are not installed (expected in such environments; CI installs the required native deps, and uses Xvfb only for display-tagged tests)

---

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs / TODOs | 1 | 0 | 0 | 0 | 1 |
| Dead Code | 4 | 0 | 2 | 1 | 1 |
| Partially Wired | 4 | 0 | 2 | 2 | 0 |
| Interface Gaps | 0 | 0 | 0 | 0 | 0 |
| Dependency Gaps | 3 | 0 | 0 | 1 | 2 |
| **Totals** | **12** | **0** | **4** | **4** | **4** |

---

## Implementation Completeness by Package

| Package | Exported Functions | Implemented | Stubs | Dead | Notes |
|---------|--------------------|-------------|-------|------|-------|
| `cmd/server` | 185 exports | ✅ All core mechanics | 0 | 2 | `recoverInvestigator`, `applyDifficulty` never called from prod |
| `client/ebiten` | 29 | ✅ State + net | 0 | 0 | |
| `client/ebiten/app` | 16 | ✅ Game loop, input | 0 | 2 | `playerColourIndex`, `min8` test-only callers |
| `client/ebiten/render` | 12 | ⚠️ Partial | 0 | 3 | `NewShaderSet`, `DrawDoomVignette`, scale fields unused |
| `cmd/desktop` / `cmd/web` | 1 each | ✅ | 0 | 0 | |
| `cmd/mobile` | 3 | ✅ (binding scaffold) | 0 | 0 | Not verified on device |

---

## Findings

### HIGH

- [ ] **Doom bar does not scale — `DrawCmd.ScaleX`/`ScaleY` silently dropped** —
  `client/ebiten/render/layers.go:70-78` —
  `Compositor.Flush()` calls `r.atlas.DrawSprite(screen, cmd.Sprite, cmd.X, cmd.Y, cmd.Tint)`
  but never passes `cmd.ScaleX`/`cmd.ScaleY`. `DrawSprite` only calls `op.GeoM.Translate`.
  `enqueueDoomEffect` in `game.go:167` sets `ScaleX: fraction * (200.0/64)` to stretch
  the doom marker proportionally to the current doom level, but the stretch is silently
  discarded. The doom bar renders as a fixed-size 64×64 tile regardless of doom level. —
  **Blocked goal**: visual doom feedback described in CLIENT_SPEC.md §4.4. —
  **Remediation**: In `Compositor.Flush()` pass scale to `DrawSprite`; update
  `DrawSprite` signature to accept `scaleX, scaleY float64` and call
  `op.GeoM.Scale(scaleX, scaleY)` before the `Translate`. Validate with:
  `DISPLAY=:99 go test -race -tags=requires_display ./client/ebiten/render/...`

- [ ] **Kage shaders compiled but never invoked — `ShaderSet`/`NewShaderSet`/`DrawDoomVignette` are dead code** —
  `client/ebiten/render/shaders.go:29,63` —
  `NewShaderSet()` is never called from `NewCompositor()`, `NewGame()`, or any
  other production path. `Compositor` has no `shaders` field. `DrawDoomVignette()` is
  never called from `game.go:Draw()`. All three Kage shaders (`fog.kage`, `glow.kage`,
  `doom.kage`) are embedded, parse correctly, and test-compilable — but no frame ever
  executes them. The fog-of-war, glow pulse, and doom vignette effects specified in
  CLIENT_SPEC.md §4.4 are absent from runtime. —
  **Blocked goal**: CLIENT_SPEC.md §4.4 "Doom Counter — doom vignette" and
  ROADMAP.md §Priority 2 "Kage doom vignette shader per CLIENT_SPEC §4.4". —
  **Remediation**: Add `shaders *render.ShaderSet` field to `Compositor` (or `Game`);
  call `render.NewShaderSet()` in `Game.Draw()` (once, lazily, on first draw); call
  `render.DrawDoomVignette(screen, shaders, float32(gs.Doom)/12)` at the end of
  `Draw()`. Handle compile errors gracefully (log + skip). Validate with
  `DISPLAY=:99 go test -race -tags=requires_display ./client/ebiten/render/...`.

- [ ] **`InvestigatorType` never set in production — all players default to Survivor ability** —
  `cmd/server/connection.go:70` (registerPlayer), `cmd/server/actions.go:215` —
  `registerPlayer` constructs every `Player` without setting `InvestigatorType`; the
  zero value `""` is not in `DefaultInvestigatorAbilities`, so `performComponent`
  silently falls back to `InvestigatorSurvivor` for every player. The six investigator
  archetypes (Researcher, Detective, Occultist, Soldier, Mystic, Survivor) are fully
  implemented with costs and effects but are unreachable from gameplay.
  The `selectInvestigator` action described in CLIENT_SPEC.md §3 is not handled by the
  server at all (`isValidActionType` does not include it). This is also a protocol-
  consistency issue: all existing server `ActionType` wire values are lowercase
  (`"move"`, `"ward"`, `"closegate"`, etc.), while CLIENT_SPEC.md §3 currently spells
  this action in camelCase (`"selectInvestigator"`). —
  **Blocked goal**: CLIENT_SPEC.md §3 "Character Selection", RULES.md "variable player
  powers through unique investigator abilities". —
  **Remediation**: (a) Add `ActionSelectInvestigator ActionType = "selectinvestigator"`
  to `game_constants.go`, following the server's established lowercase `ActionType`
  convention; (b) add it to `isValidActionType`; (c) implement
  `performSelectInvestigator(player *Player, target string) error` that sets
  `player.InvestigatorType` from `target` if valid and the game is in "waiting" phase;
  (d) route it through `dispatchAction`. Because CLIENT_SPEC.md §3 currently uses the
  camelCase spelling, the implementation must either update the spec/client to emit
  `"selectinvestigator"` or add server-side normalization (`strings.ToLower`) before
  validation/dispatch so both spellings are accepted without breaking existing action
  validation. Validate with new test `TestProcessAction_SelectInvestigator`.

- [ ] **Scene state machine from CLIENT_SPEC.md not implemented in Ebitengine client** —
  `client/ebiten/app/game.go` —
  CLIENT_SPEC.md §1 specifies four scenes: `SceneConnect → SceneCharacterSelect →
  SceneGame → SceneGameOver`. The client unconditionally renders the in-game view
  with no connection screen, no character selection, and no game-over screen. Players
  see the game board the moment `NewGame()` is called, even before WebSocket
  connection is established. The win/lose conditions are displayed only as text
  overlays at the bottom of the in-game screen, not as a dedicated `SceneGameOver`.
  The 60-second reconnect countdown overlay, the player-slot indicator, and the
  "Game Full" state from CLIENT_SPEC.md §2 are absent. —
  **Blocked goal**: CLIENT_SPEC.md §1–2, §6; ROADMAP.md Phase 3 "scene state machine". —
  **Remediation**: Introduce a `Scene` interface with `Update() error` and
  `Draw(*ebiten.Image)` methods; implement `SceneConnect`, `SceneGame`, and
  `SceneGameOver` as concrete types; manage transitions in `Game.Update()` based on
  `LocalState.Connected` and `gs.WinCondition`/`gs.LoseCondition`. `SceneCharacterSelect`
  can be deferred until `selectInvestigator` (HIGH gap above) is wired. Validate
  that `go test -race -tags=requires_display ./client/ebiten/app/...` covers
  scene transitions.

### MEDIUM

- [ ] **`recoverInvestigator` is dead production code** —
  `cmd/server/game_mechanics.go:59` —
  `recoverInvestigator` is defined and doc-commented but is only called from test code
  (`cmd/server/rules_test.go:572`). No production code path triggers investigator
  recovery. An investigator defeated via `checkInvestigatorDefeat` enters
  `LostInTimeAndSpace = true` permanently — the state is never cleared unless a test
  directly calls the recovery method. As-documented (RULES.md §"Investigator
  Defeat/Recovery"), investigators should recover and re-enter the turn rotation. —
  **Blocked goal**: RULES.md §"Investigator Defeat (health OR sanity = 0)" recovery path. —
  **Remediation**: Add logic in `advanceTurn()` or a dedicated Mythos Phase step to
  call `gs.recoverInvestigator(id)` for all players where
  `player.LostInTimeAndSpace && player.Connected`. Alternatively, auto-recover at the
  start of the Mythos Phase. Add `TestInvestigatorAutoRecovery` to `rules_test.go`.

- [ ] **`applyDifficulty` not wired to any WebSocket message handler** —
  `cmd/server/game_mechanics.go:113` —
  `applyDifficulty("easy"|"standard"|"hard")` is implemented and tested but is never
  called from any production code path. The `DifficultyConfig` map, `DifficultySetup`
  struct, and the `GameState.Difficulty` field are all in place. However, no
  `playerAction` message type dispatches to this function, and `newGameServerWithScenario`
  does not apply a difficulty preset. Players and tests are the only callers. As a
  result, `gs.gameState.Difficulty` is always `""` at runtime. —
  **Blocked goal**: CLIENT_SPEC.md §7 "Modular Difficulty Settings"; RULES.md
  "Modular Difficulty Settings" compliance row. —
  **Remediation**: (a) Add `ActionSetDifficulty ActionType = "setdifficulty"` constant;
  (b) add a `difficulty` action handler in `dispatchAction` that calls
  `gs.applyDifficulty(action.Target)` only when `gs.gameState.GamePhase == "waiting"`;
  (c) add it to `isValidActionType`. Validate with
  `TestProcessAction_SetDifficulty_Waiting` and `TestProcessAction_SetDifficulty_Playing`
  (should return error when game is already in progress).

- [ ] **Quick-chat panel from CLIENT_SPEC.md §5 not implemented** —
  `cmd/server/game_server.go:334` (`isValidActionType`) —
  CLIENT_SPEC.md §5 specifies a "Quick-Chat Panel" sending
  `{"type":"playerAction","action":"chat","target":"<phrase>"}`. The server does not
  define `ActionChat`, `isValidActionType` does not include `"chat"`, and
  `dispatchAction` has no case for it. Any `chat` message from a conforming client
  is silently rejected with "invalid action type" and the error is counted against
  `errorCount`. The six mandatory phrases, event-log display, and collapsible panel
  are absent from both the server and the Ebitengine client. —
  **Blocked goal**: CLIENT_SPEC.md §5 "Player Communication". —
  **Remediation**: (a) Add `ActionChat ActionType = "chat"` to `game_constants.go`;
  (b) add it to `isValidActionType`; (c) implement `performChat(playerID, phrase string)`
  that validates the phrase is non-empty, logs it, and returns a `gameUpdate` event
  with `event: "chat"` and `result: phrase`; (d) display the chat entry in the client
  event log. Validate with `TestProcessAction_Chat`.

- [ ] **`PLAN.md` referenced in source comments but does not exist** —
  `cmd/server/benchmark_test.go:3`, `cmd/server/rules_test.go:3`,
  `cmd/server/origin_test.go:1` —
  Several test files reference "PLAN.md Step N" or "PLAN.md Step M8" in their package
  doc comments, and README.md lists `PLAN.md` in the project structure. The file does
  not exist. This is a navigation/onboarding gap: new contributors following the
  README's structure map will find a broken reference. —
  **Blocked goal**: README.md §Project Structure ("PLAN.md — Implementation plan for
  current gaps + migration"). —
  **Remediation**: Either (a) create `PLAN.md` that documents the phases referenced
  in test files, or (b) update the README.md §Project Structure table to remove the
  `PLAN.md` entry and update the three test file comments to remove `PLAN.md`
  step references.

### LOW

- [ ] **Four unused analytics types dead in `game_types.go`** —
  `cmd/server/game_types.go:187–238` —
  `ConnectionAnalytics`, `SessionDistribution`, `PlayerSessionMetrics`, and
  `AlertThreshold` are defined but never instantiated by any production code.
  The analytics system uses the `*Simplified` variants defined in `metrics.go`.
  These four types are likely remnants of an earlier design iteration that was
  superseded. They add noise to the type surface and may mislead contributors
  into thinking they drive live behaviour. —
  **Remediation**: Remove the four unused type definitions from `game_types.go`.
  Verify with `go vet ./cmd/server/...` and `go build ./cmd/server/...`.

- [ ] **`client/ebiten/input.go` is a zombie file** —
  `client/ebiten/input.go:1` —
  The file opens with `//go:build ignore` and its own comment explains it is
  superseded by `client/ebiten/app/input.go`. It is never compiled, never tested,
  and duplicates type and function names (`InputHandler`, `NewInputHandler`,
  `keyBindings`) that exist in the active package. It adds confusion to directory
  listings and `grep` results without providing value. —
  **Remediation**: Delete `client/ebiten/input.go`. Validate with
  `go build ./client/ebiten/...` (headless) and confirm no tests reference it.

- [ ] **RULES.md compliance table severely out of date** —
  `RULES.md` (compliance table header) —
  The embedded compliance table marks Action System as "⚠️ Partial (4 of 8 actions)",
  Dice Resolution as "⚠️ Partial (no focus token spend)", Mythos Phase as
  "❌ Not implemented", Resource Management as "⚠️ Partial", Encounter Resolution
  as "❌ Not implemented", Act/Agenda Deck as "❌ Not implemented", Investigator
  Defeat as "⚠️ Partial", Scenario System as "❌ Not implemented", and Modular
  Difficulty as "❌ Not implemented" — all with "None" test coverage cited. In
  reality the server fully implements all of these with 178+ passing tests. The stale
  table misinforms contributors and user-facing documentation. —
  **Remediation**: Update RULES.md compliance table to reflect current implementation
  status (all ✅ except Modular Difficulty which is partially wired — see MEDIUM gap
  above). Verify by cross-referencing `TestRulesFullActionSet`, `TestRulesMythosPhase*`,
  etc. against the table rows.

- [ ] **`PacketLoss` always reports 0 in connection quality** —
  `cmd/server/dashboard.go:61` —
  `initializeConnectionQuality` sets `PacketLoss: 0` and `updateConnectionQuality`
  increments it by `0.1` only when `messageDelay > 200ms`, but `messageDelay` is
  computed as `time.Since(messageTime).Seconds()` relative to receipt time — a
  measure of server-side processing latency, not network packet loss. The field is
  broadcast to all clients in `connectionStatus` messages and read by
  `assessConnectionQuality` to classify connections as "poor", but the measure is
  structurally inaccurate as a packet-loss proxy. —
  **Remediation**: Either (a) remove `PacketLoss` from `ConnectionQuality` and
  simplify to ping-RTT-only quality classification, or (b) implement real packet-loss
  tracking via missed pong responses (increment on ping timeout, decrement on receipt).
  Document the chosen metric clearly in the struct comment.

---

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|-----------------|
| `playerColourIndex` and `min8` in `client/ebiten/app/game.go` are "dead code" | Both are called from `client/ebiten/app/game_test.go`; they are tested helper functions, not dead code. Rejected per Phase 3f check 3. |
| `applyDifficulty` tests prove it is "complete" | Tests exercise the function in isolation. The gap is that no production message path calls it — it is a partially wired feature, not a complete one. |
| `ConnectionAnalyticsSimplified` vs `ConnectionAnalytics` duplication | `ConnectionAnalytics` (game_types.go) is an architectural type; `ConnectionAnalyticsSimplified` (metrics.go) is a concrete runtime type. The former is unused, confirming the dead-type finding. |
| `recoverInvestigator` called from `rules_test.go` proves it is reachable | Test callers do not constitute a production execution path. The function is only tested, never triggered by gameplay. This is a legitimate gap. |
| `ConnectionWrapper.Write` is "unused" because the server sends via `wsConn` directly | `ConnectionWrapper` is stored in `gs.connections` and used as the `net.Conn` argument to `handleConnection`. It is also set as the deadline target. Its `Write` method is part of the `net.Conn` contract and may be called via the interface. Not a gap. |
| `ShaderSet.Deallocate` never called | The shaders are never instantiated, so cleanup is moot. The gap is the instantiation, not the cleanup path. Already captured in the main finding. |
| `DifficultyConfig` and `defaultActDeck`/`defaultAgendaDeck` are unused | These are called from `DefaultScenario.SetupFn` and from `rescaleActDeck`. They are fully wired for the standard game flow. Rejected. |
| `Dummy()` in `cmd/mobile/binding.go` is "dead code" | Required symbol for `ebitenmobile` binding generator toolchain. Intentional per inline comment. Rejected. |
| `ROADMAP.md` Priority 3 (CI benchmark reporting) has unchecked items | The CI benchmark step exists in `.github/workflows/ci.yml`; the unchecked items concern an optional third-party service (Bencher/codspeed). Not a code gap. |
