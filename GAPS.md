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
- **Effort:** Large per additional game family.
