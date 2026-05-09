# Implementation Gaps - 2026-05-08

## Browser Client Script Does Not Parse

- **Evidence:** `client/game.js:397`, `client/index.html:316`
- **Intended Behavior:** The active browser client should load from `client/index.html`, connect to the Go server, and drive gameplay through `client/game.js` as described in `README.md`.
- **Current State:** `client/game.js` contains a duplicated `displayDiceResult` body after `displayGameUpdate`, leaving raw statements inside the class body. `node --check client/game.js` fails with `SyntaxError: Unexpected identifier 'resultDiv'`.
- **Blocked Goal:** The current HTML/JS client path in the README is not functional.
- **Implementation Path:** Remove the duplicated block beginning at `client/game.js:397`, keep a single `displayDiceResult()` implementation, and add a lightweight syntax check for browser assets to CI or a smoke-test script.
- **Dependencies:** None.
- **Validation:** `node --check client/game.js`, `go run ./cmd/server`, then load `/` and confirm the WebSocket client initializes.
- **Effort:** Small.

## Waiting-Phase Pregame Actions Cannot Run In Real Sessions

- **Evidence:** `serverengine/connection.go:110`, `serverengine/actions.go:388`, `serverengine/actions.go:405`, `serverengine/connection_test.go:66`, `docs/RULES.md:28`, `docs/RULES.md:32`
- **Intended Behavior:** Players should be able to select an investigator and set difficulty before live play begins. `docs/RULES.md` marks both systems complete.
- **Current State:** The first player connection starts the game immediately because `MinPlayers = 1`, which sets `GamePhase = "playing"`. Both `performSelectInvestigator()` and `performSetDifficulty()` reject calls unless the phase is still `waiting`. The unit tests cover only synthetic waiting-phase state, not the real connection path.
- **Blocked Goal:** Investigator selection and difficulty presets are effectively unreachable in normal multiplayer sessions.
- **Implementation Path:** Introduce a real pregame lobby boundary. Either keep `GamePhase = "waiting"` until an explicit ready/start condition is satisfied, or permit these two actions until the first turn begins. Add an end-to-end test that connects a player through the WebSocket server and successfully performs both pregame actions before the first normal turn action.
- **Dependencies:** None, but any client-side pregame UI work depends on this being fixed first.
- **Validation:** `go test ./serverengine -run 'TestHandleWebSocket|TestProcessAction_(SelectInvestigator|SetDifficulty)'` plus a new integration test for `connect -> select/difficulty`.
- **Effort:** Medium.

## Ebitengine Connect And Selection Scenes Are Below Published Spec

- **Evidence:** `docs/CLIENT_SPEC.md:62`, `docs/CLIENT_SPEC.md:68`, `docs/CLIENT_SPEC.md:79`, `docs/CLIENT_SPEC.md:93`, `client/ebiten/app/scenes.go:24`, `client/ebiten/app/scenes.go:94`, `client/ebiten/app/scenes.go:178`
- **Intended Behavior:** `SceneConnect` should expose server address, player display name, slot/wait status, and reconnect countdown behavior. `SceneCharacterSelect` should wait for all connected players to confirm before transitioning to `SceneGame`.
- **Current State:** `SceneConnect` is only an animated banner. There is no connect-form state, no slot display, and no reconnect countdown. `SceneCharacterSelect` only supports local key selection and `updateScene()` advances to `SceneGame` as soon as the local player has any investigator type.
- **Blocked Goal:** The alpha Ebitengine client does not yet satisfy the published four-scene UX contract.
- **Implementation Path:** Add connect-form state to `LocalState` and scene code, decide whether player display name belongs in the wire protocol, render slot/wait status, and change the select-scene exit condition to `all connected players have selected`. If the server continues to own lobby readiness, expose that ready-state in `gameState`.
- **Dependencies:** Waiting-phase pregame actions must work first.
- **Validation:** `go test -tags=requires_display ./client/ebiten/app/...` and a manual desktop/WASM session with at least two players.
- **Effort:** Medium to Large.

## Browser Connection-Quality Status Uses An Unreachable Branch

- **Evidence:** `client/game.js:462`, `serverengine/connection_quality.go:24`
- **Intended Behavior:** The browser client's top-right status badge should update when the server broadcasts a `connectionQuality` payload for the current player.
- **Current State:** The client checks `qualityMessage.playerID`, but the server emits `playerId`. The branch that calls `updateOwnConnectionStatus()` is never taken.
- **Blocked Goal:** The browser client's connection-quality status is only partially wired.
- **Implementation Path:** Change the comparison to `qualityMessage.playerId === this.playerId`. Add a small fixture-based test or smoke script that feeds a sample `connectionQuality` payload through `handleConnectionQuality()`.
- **Dependencies:** Independent, but the browser syntax error must be fixed before the path is reachable at runtime.
- **Validation:** Load the browser client against a live server, wait for ping/pong traffic, and confirm the badge text changes from generic `Connected` to a latency-qualified state.
- **Effort:** Small.

## Ebitengine Client Still Encodes An Old Connection-Quality Schema

- **Evidence:** `client/ebiten/state.go:87`, `client/ebiten/state.go:89`, `client/ebiten/state.go:194`, `client/ebiten/net_test.go:102`, `client/ebiten/net_test.go:331`, `serverengine/connection_quality.go:12`, `serverengine/connection_quality.go:13`
- **Intended Behavior:** The Go client should consume the same `connectionQuality` JSON shape the server emits.
- **Current State:** The Ebitengine client still expects `latency` and `rating`, while the server sends `latencyMs` and `quality`. The tests reinforce the stale payload shape.
- **Blocked Goal:** Future connection-quality HUD work in the Go client will silently bind to the wrong fields.
- **Implementation Path:** Update the Go client structs and fixtures to `latencyMs` / `quality`, then add a route-message test using an actual marshaled server payload instead of a hand-written stale fixture.
- **Dependencies:** None.
- **Validation:** `go test ./client/ebiten/...` and, if a quality HUD is later added, a live ping/pong smoke test.
- **Effort:** Small.

## 25 Scaffold Packages Have No Implementation Behind Them

- **Evidence:** `serverengine/arkhamhorror/actions/doc.go:1`, `serverengine/common/messaging/doc.go:1`, `serverengine/eldersign/adapters/doc.go:1`
- **Intended Behavior:** The package layout under `serverengine/arkhamhorror`, `serverengine/common`, and other game families is supposed to host the long-term rules decomposition described in `serverengine/arkhamhorror/README.md`.
- **Current State:** 25 packages are still `doc.go` only. They compile and document intent, but they do not yet own code, tests, or APIs.
- **Blocked Goal:** The advertised modular architecture is mostly structural, not implemented.
- **Implementation Path:** Migrate incrementally rather than filling all scaffolds at once. Start with `serverengine/arkhamhorror/model` and `serverengine/arkhamhorror/rules`, then move shared validation/state helpers into `serverengine/common/{validation,state}`. If migration is paused, remove or collapse the unused package directories to reduce noise.
- **Dependencies:** None.
- **Validation:** `go list ./...`, then package-specific tests as each slice is moved.
- **Effort:** Large overall, Small to Medium per migrated slice.

## Alternative Game Modules Are Registered But Not Playable

- **Evidence:** `cmd/server/main.go:29`, `cmd/server/main.go:30`, `cmd/server/main.go:31`, `cmd/server/main.go:32`, `serverengine/eldersign/module.go:22`
- **Intended Behavior:** The module registry suggests that `eldersign`, `eldritchhorror`, and `finalhour` are selectable runtime families.
- **Current State:** Each module's `NewEngine()` returns `runtime.NewUnimplementedEngine(...)`, so selecting any of them ends the server at startup with a not-implemented error.
- **Blocked Goal:** Multi-game-family runtime selection is not actually available yet.
- **Implementation Path:** Either keep only `arkhamhorror` in the default registry until a second engine exists, or add an explicit experimental gate and clearer startup messaging. When implementation begins, build one real engine family end-to-end before keeping it user-selectable.
- **Dependencies:** The shared/common and Arkham modularization work should settle first if code reuse is the goal.
- **Validation:** `BOSTONFEAR_GAME=arkhamhorror go run ./cmd/server` succeeds; each non-Arkham module should either be hidden or replaced with a real engine that also starts successfully.
- **Effort:** Large per additional game family.# Implementation Gaps — 2026-05-08

> This file supersedes the previous organization-focused GAPS.md.
> Each gap is actionable, references specific files/lines, and is evaluated
> against the project's **own stated goals** — not aspirational features.

---

## Gap 1: SceneCharacterSelect Not Implemented

- **Intended Behavior**: `CLIENT_SPEC.md §1` and the scene-state-machine diagram specify the flow `SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver`. The `SceneCharacterSelect` scene should present the 6 investigator archetypes (Researcher, Detective, Occultist, Soldier, Mystic, Survivor), allow the player to pick one, and send `ActionSelectInvestigator` to the server before entering `SceneGame`.
- **Current State**: The type `SceneCharacterSelect` does not exist anywhere in the codebase. The package comment in [client/ebiten/app/scenes.go](client/ebiten/app/scenes.go#L5) explicitly acknowledges the deferral ("deferred until the selectInvestigator server action is fully wired on the client side"). `updateScene()` skips the character-select step and moves directly from `SceneConnect` to `SceneGame`. Players using the desktop or WASM client always start as the default investigator type.
- **Blocked Goal**: CLIENT_SPEC.md §2 (character selection); AH3e component-action parity for non-default archetypes.
- **Implementation Path**:
  1. Add `SceneCharacterSelect` struct in `client/ebiten/app/scenes.go` with `Update()` and `Draw()` methods implementing `Scene`.
  2. In `Draw()`, render the 6 investigator names with their key shortcut (e.g. `[1] Researcher  [2] Detective …`).
  3. In `Update()`, check key presses 1–6 and call `h.net.SendAction(PlayerActionMessage{Type:"playerAction", PlayerID: playerID, Action: ActionSelectInvestigator, Target: investigatorName})`.
  4. Transition to `SceneGame` once `gs.Players[playerID].InvestigatorType != ""`.
  5. Modify `updateScene()` in `client/ebiten/app/scenes.go` to insert `SceneCharacterSelect` between `SceneConnect` and `SceneGame` when `InvestigatorType` is unset.
- **Dependencies**: None — `ActionSelectInvestigator` is already handled server-side at `serverengine/game_server.go:231`.
- **Effort**: Small (≈ 60 lines across `scenes.go` and `input.go`)
- **Severity**: HIGH

---

## Gap 2: Ebitengine Client Missing 5 of 12 Server Actions

- **Intended Behavior**: The server engine accepts 12 action types: Move, Gather, Investigate, CastWard, Focus, Research, Trade, Component, Encounter, Attack, Evade, CloseGate. The Ebitengine client is the only first-party native client. `CLIENT_SPEC.md §3` requires action parity with the HTML/JS browser client.
- **Current State**: [client/ebiten/app/input.go](client/ebiten/app/input.go#L18) `keyBindings` contains only 7 entries: Move×4, Gather, Investigate, CastWard. Focus, Research, Trade, Component, Attack, Evade, and CloseGate are unreachable from the desktop/WASM/mobile client via keyboard input.
- **Blocked Goal**: AH3e rules compliance for native-client players; ROADMAP.md goal 13 ("AH3e rules compliance — all 8 action types").
- **Implementation Path**:
  1. Add key bindings to `keyBindings` in `client/ebiten/app/input.go`:
     - `{ebiten.KeyF, "focus", ""}` — Focus
     - `{ebiten.KeyR, "research", ""}` — Research
     - `{ebiten.KeyC, "component", ""}` — Component ability
     - `{ebiten.KeyA, "attack", ""}` — Attack (first engaged enemy)
     - `{ebiten.KeyE, "evade", ""}` — Evade (first engaged enemy)
     - `{ebiten.KeyX, "closeGate", ""}` — Close Gate
  2. Trade requires a target player ID. Add a simple multi-step input: `KeyT` enters a "select trade target" mode that lists co-located players; a second key press selects the target and sends `ActionTrade` with `Target: selectedPlayerID`.
  3. Update the HUD key-legend rendering in `game.go` to show all bindings.
- **Dependencies**: Gap 1 (SceneCharacterSelect) should be resolved first to ensure the Component action uses the correct archetype ability.
- **Effort**: Small–Medium (≈ 30 lines for simple bindings; ≈ 80 lines including trade target flow)
- **Severity**: HIGH

---

## Gap 3: SceneGameOver "Play Again" Is a No-Op

- **Intended Behavior**: `CLIENT_SPEC.md §1` state machine shows `SceneGameOver → SceneConnect` as the "Play Again" transition. The game-over screen should allow the player to restart without closing the application.
- **Current State**: `SceneGameOver.Update()` at [client/ebiten/app/scenes.go](client/ebiten/app/scenes.go#L73) returns `nil` with no input handling. The `Draw()` method renders "Close the window to exit." with no restart option. Restarting requires killing and relaunching the process.
- **Blocked Goal**: CLIENT_SPEC.md §1 state machine completeness; user experience for repeated play.
- **Implementation Path**:
  1. In `SceneGameOver.Update()`, add `if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace)` block.
  2. Inside the block, reset the `LocalState` via `g.state.Reset()` (or create a new one), call `g.net.Connect()` to re-establish the connection, and set `g.activeScene = &SceneConnect{game: g}`.
  3. Update `SceneGameOver.Draw()` to render "Press ENTER to play again • Close window to exit".
  4. If `LocalState` does not expose a `Reset()` method, add one that clears the cached `GameState` and `playerID`.
- **Dependencies**: None.
- **Effort**: Small (≈ 20 lines)
- **Severity**: MEDIUM

---

## Gap 4: Touch Input Not Implemented for Mobile Client

- **Intended Behavior**: `CLIENT_SPEC.md §1` specifies "Touch input on mobile platforms via `ebiten.TouchID` mapping". `ROADMAP.md Phase 4` lists touch support as a milestone. The mobile binding (`cmd/mobile`) compiles and registers the same `Game` used by desktop/WASM; on a touchscreen device, players cannot issue any action.
- **Current State**: `InputHandler.Update()` at [client/ebiten/app/input.go](client/ebiten/app/input.go#L46) polls only `inpututil.IsKeyJustPressed`. There is no call to `ebiten.TouchIDs` or `inpututil.JustPressedTouchIDs`. Touch events are silently ignored.
- **Blocked Goal**: ROADMAP.md Phase 4 — mobile touch input; mobile client usability.
- **Implementation Path**:
  1. Add a `handleTouchInput()` method to `InputHandler`.
  2. Call `ebiten.AppendJustPressedTouchIDs(nil)` to detect new touch events each frame.
  3. For each touch ID, call `ebiten.TouchPosition(id)` to get coordinates.
  4. Map tap coordinates against `locationRects` (imported from `game.go` or an exported constant) to resolve Move actions.
  5. Add fixed tap regions in the HUD (bottom strip) for Gather, Investigate, CastWard, Focus, etc.
  6. Call `h.net.SendAction(...)` with the resolved action, mirroring the keyboard path.
  7. Call `handleTouchInput()` from `Update()` alongside the existing keyboard loop.
- **Dependencies**: None; can be implemented independently of Gap 1 and Gap 2, though the HUD tap regions should include any actions added in Gap 2.
- **Effort**: Medium (≈ 80–120 lines)
- **Severity**: MEDIUM

---

## Gap 5: 22 Empty Scaffold Packages

- **Intended Behavior**: `ROADMAP.md` and `serverengine/arkhamhorror/README.md` describe a planned decomposition of the `serverengine` monolith into typed sub-packages: `actions/`, `adapters/`, `model/`, `phases/`, `rules/`, `scenarios/` for Arkham Horror, and matching stubs for Elder Sign, Eldritch Horror, and Final Hour. The `serverengine/common/` sub-packages (`messaging`, `monitoring`, `observability`, `session`, `state`, `validation`) are intended to hold cross-engine primitives.
- **Current State**: All 22 packages contain only a `doc.go` with a two-line package declaration. They export nothing, are not imported by any other package, and have no tests. The `go test ./...` output lists all 22 as `no test files`.
- **Blocked Goal**: Modular rules decomposition; testability of individual rule subsystems.
- **Implementation Path**: Decomposition should proceed incrementally per the migration boundary defined in `serverengine/arkhamhorror/README.md`:
  1. Start with `serverengine/arkhamhorror/model` — move type aliases and constants from `protocol` into a game-family-specific model layer.
  2. Then `serverengine/arkhamhorror/rules` — extract `validateMovement`, `validateResources`, and action-success thresholds.
  3. Then `serverengine/arkhamhorror/actions` — move `perform*` functions from `serverengine/actions.go`.
  4. Then `serverengine/common/validation` and `serverengine/common/state` — move generic bound-checking utilities.
  5. Populate `elderSign/`, `eldritchhorror/`, `finalhour/` sub-packages only after the Arkham migration template is stable.
- **Dependencies**: None for starting; downstream packages must update their imports as each migration completes.
- **Effort**: Large overall; each individual package is Small–Medium.
- **Severity**: LOW (scaffolding is intentional; no runtime impact)

---

## Gap 6: Elder Sign / Eldritch Horror / Final Hour Engines Are Unimplemented

- **Intended Behavior**: The module registry supports `BOSTONFEAR_GAME=eldersign`, `eldritchhorror`, and `finalhour` selection. Users expect selecting these modules to launch a playable variant.
- **Current State**: All three modules return `runtime.NewUnimplementedEngine(...)` from `NewEngine()`. Selecting any of them via `BOSTONFEAR_GAME` causes the server to fail at startup with `"X engine not implemented"`. This is documented and intentional, but these modules are registered in `cmd/server/main.go` and advertised by `registry.Keys()`, creating an expectation mismatch.
- **Blocked Goal**: Multi-game-family vision described in README and ROADMAP.
- **Implementation Path**: For each game family:
  1. Implement a playable engine in the respective `module.go` (or a new `engine.go` under the package).
  2. Wire it to `contracts.Engine` (all 11 interface methods).
  3. Add a `Start()` implementation that launches a game loop adapted from `serverengine.GameServer`.
  4. Populate the `rules/`, `model/`, and `scenarios/` sub-packages with the game-specific constants.
  Elder Sign is the recommended first implementation as it shares the AH3e dice-pool mechanic.
- **Dependencies**: Gap 5 (Arkham rules decomposition) should be completed first so the common runtime primitives are available for re-use.
- **Effort**: Large per game family.
- **Severity**: LOW (documented placeholder behavior; does not affect the primary Arkham Horror use case)

---

## Gap 7: Mobile Client Not Verified on Device

- **Intended Behavior**: README documents Android and iOS builds via `ebitenmobile bind`. The mobile client should launch, connect, and be fully playable on Android API 29+ and iOS 16+.
- **Current State**: `cmd/mobile/binding.go` compiles successfully. Per README: "not verified on physical device". Gap 2 (missing action bindings) and Gap 4 (no touch input) mean the mobile client would not be fully functional even if it launched correctly.
- **Blocked Goal**: ROADMAP.md "Mobile binding (iOS/Android)" — partial achievement.
- **Implementation Path**:
  1. Resolve Gap 4 (touch input) first — without it, mobile is unusable.
  2. Run `ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile` on a machine with Android SDK API 29+.
  3. Create a minimal Android host Activity that loads the AAR and calls `SetServerURL()`.
  4. Test connection, state sync, and action submission on emulator then physical device.
  5. Repeat for iOS with Xcode 15+.
  6. Document test results in ROADMAP.md.
- **Dependencies**: Gap 4 (touch input) must be resolved first.
- **Effort**: Medium (build toolchain setup); Small (code changes, since the binding scaffold is complete).
- **Severity**: LOW
