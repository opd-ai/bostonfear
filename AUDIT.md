# UI/UX Clarity Audit — May 9, 2026

## Scope

This report audits the current player-facing web client path for BostonFear.
The repository no longer ships a standalone JavaScript gameplay client; the browser path is a thin WASM host over the Go/Ebitengine client. The audit therefore covers the actual browser-visible UI, input, and state-feedback surfaces.

## Coverage

- Audited: `client/wasm/index.html`
- Audited: `client/ebiten/net.go`
- Audited: `client/ebiten/state.go`
- Audited: `client/ebiten/app/game.go`
- Audited: `client/ebiten/app/input.go`
- Audited: `client/ebiten/app/scenes.go`
- Audited: `client/ebiten/app/text_ui.go`
- Audited: `client/ebiten/render/atlas.go`
- Audited: `client/ebiten/render/layers.go`
- Audited: `client/ebiten/ui/onboarding.go`
- Audited: `client/ebiten/ui/results.go`
- Audited: `client/ebiten/ui/state.go`
- Skipped: transport adapters, package docs, and client tests
- Skipped: UI helper files not currently surfaced by the active scene renderer

## Executive Summary

The UI is not yet obvious for a first-time player. It has a usable connection scene, a persistent turn HUD, and a tutorial overlay, but the live game view still expects the player to infer too much from placeholder visuals and static key hints.

Top confusion drivers:

1. The board does not clearly label locations or show movement adjacency.
2. The HUD exposes a static, incomplete control legend rather than current legal actions.
3. Action feedback omits the detailed dice information needed to explain success, failure, and doom changes.

Primary improvement area: state visibility.

## Concise Remediation Checklist

- [x] Label all four locations directly on the board.
- [x] Visualize legal adjacency from the current player position.
- [x] Replace the static controls legend with a state-driven available-actions panel.
- [x] Show action costs and disable reasons in the HUD.
- [x] Render the actual dice faces and required success threshold after each roll.
- [x] Surface human-readable invalid-action errors instead of only retry counts.
- [x] Add touch parity for the full supported action set.
- [x] Prevent touch action taps from also rotating or toggling the camera.
- [x] Promote player display names into the shared wire-visible UI identity.
- [x] Correct the WASM host status so it distinguishes client boot from server connectivity.
- [x] Upgrade text rendering and wrap long tutorial, event, and result text.

## Findings

### [HIGH] Board does not teach locations or legal movement
- File: `client/ebiten/app/game.go`, `client/ebiten/render/atlas.go`
- Category: Discoverability
- Player Goal At Risk: Understand where they are and where they can move.
- Player Impact: A new player sees colored tiles but not a clearly readable neighborhood map.
- Problem: The live board relies on sprites and placeholder tiles without explicit location labels or adjacency cues.
- Evidence: The board only enqueues sprites, while movement names appear mainly in the controls legend and placeholder atlas output.
- Fix: Draw location labels on-board and visualize legal adjacent destinations for the active player.
- Validation: A first-time player should be able to name all four locations and identify legal moves without external documentation.
- Remediation Checklist:
  - [x] Add board labels for Downtown, University, Rivertown, and Northside.
  - [x] Highlight the active player location.
  - [x] Draw adjacency connectors or current legal move highlights.

### [HIGH] HUD shows static controls instead of current legal actions
- File: `client/ebiten/app/game.go`, `client/ebiten/app/input.go`
- Category: Discoverability
- Player Goal At Risk: Know what actions are available right now.
- Player Impact: The player sees a fixed legend that does not match the full supported input surface or current legality.
- Problem: The controls panel is static and incomplete even though the input layer supports more actions.
- Evidence: The HUD lists only move, gather, investigate, ward, camera, and tutorial hints, while input bindings also include focus, research, trade, component, attack, evade, and close gate.
- Fix: Replace the static legend with a dynamic action panel derived from current game state.
- Validation: The visible action list should change with turn ownership, resources, and location constraints.
- Remediation Checklist:
  - [x] Show all supported actions in one UI surface.
  - [x] Disable unavailable actions visibly.
  - [x] Explain why an action is unavailable.
  - [x] Show remaining actions for the turn beside the action list.

### [HIGH] Dice resolution feedback omits the actual dice outcomes
- File: `client/ebiten/app/game.go`, `client/ebiten/ui/results.go`
- Category: Feedback
- Player Goal At Risk: Understand why an action succeeded or failed.
- Player Impact: Players can see pass/fail text but not the dice faces or threshold that produced the outcome.
- Problem: The results panel does not render the detailed dice data already present in the model.
- Evidence: The renderer shows outcome text and deltas, but not the dice-specific formatting path.
- Fix: Add a dice row or icon strip showing each die face, successes achieved, and successes required.
- Validation: After an investigate or ward action, the player should be able to explain the result from the UI alone.
- Remediation Checklist:
  - [x] Render success, blank, and tentacle faces explicitly.
  - [x] Show `achieved / required` successes.
  - [x] Note doom increases caused by tentacles in the same result block.

### [HIGH] Touch controls do not expose the full action set
- File: `client/ebiten/app/input.go`
- Category: Input
- Player Goal At Risk: Execute all supported actions on touch devices.
- Player Impact: Mobile and touch players cannot discover or trigger the full supported action surface.
- Problem: Touch input only maps a subset of the keyboard-supported actions.
- Evidence: The touch action bar handles six actions while keyboard input supports a larger set.
- Fix: Expand touch affordances to reach full action parity.
- Validation: Every supported action should be reachable without a keyboard.
- Remediation Checklist:
  - [x] Add touch affordances for trade, component, attack, evade, and other supported actions.
  - [x] Keep touch targets at or above the existing 44 px minimum.
  - [x] Indicate which touch actions are context-sensitive.

### [HIGH] Multiplayer identity is reduced to opaque player IDs
- File: `client/ebiten/state.go`, `client/ebiten/app/game.go`
- Category: State Visibility
- Player Goal At Risk: Track whose turn it is and coordinate with teammates.
- Player Impact: Players enter a display name, but the live game still identifies everyone by server IDs.
- Problem: Display names are local-only and do not reach the shared gameplay UI.
- Evidence: The connect scene collects a display name, but local state documents that it is not sent over the wire, and the player panel renders raw player IDs.
- Fix: Add display names to the shared session/game protocol and prefer them in turn order, event log, and outcome text.
- Validation: In a 3-player session, all visible turn and event labels should use human-readable names.
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
