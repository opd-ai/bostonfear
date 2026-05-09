# Implementation Gaps — 2026-05-08

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
