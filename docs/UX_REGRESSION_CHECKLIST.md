# UX Regression Checklist

This checklist validates first-time clarity and turn-flow trust for the Go/Ebitengine client.
Run it after UI, input, protocol, or state-sync changes.

## Scope

- Client packages: `client/ebiten/app`, `client/ebiten/ui`, `client/ebiten/render`
- Wire events: `gameState`, `gameUpdate`, `diceResult`, `connectionStatus`
- Session behavior: initial connect, reconnect, and post-reconnect action flow

## Pass Criteria

All items in sections A-F are marked pass for desktop and WASM builds.
If any item fails, capture a repro note and open a follow-up task.

## A. Onboarding Clarity

- [ ] New player can identify the current scene (connect/select/game/game-over) in under 5 seconds.
- [ ] Connect screen clearly shows editable server address and display name fields.
- [ ] Connect screen visibly indicates connection status and slot usage (players connected out of 6).
- [ ] Character select screen makes all six investigator choices visible and actionable.

## B. Turn and Action Clarity

- [ ] Current player indicator is visible without reading logs.
- [ ] Actions remaining for the local player are visible each turn.
- [ ] Available input hints for movement and actions are visible in-game.
- [ ] Out-of-turn action attempts are blocked and reflected in the invalid retry metric.

## C. Action Feedback and Outcome Trust

- [ ] Every submitted action generates a visible outcome path (`gameUpdate` and/or `diceResult`).
- [ ] Doom/resource changes are reflected in HUD and event log after state sync.
- [ ] First valid action metric transitions from pending to a concrete duration after first valid submission.
- [ ] Invalid retry counter increments only for blocked attempts, not for successful submissions.

## D. Synchronization and Reconnection

- [ ] On connection drop, status updates to disconnected/retrying state.
- [ ] Reconnect restores player slot when token reclaim is available.
- [ ] Post-reconnect game state is consistent with server snapshot (turn, doom, resources, location).
- [ ] Action submission works after reconnect without client restart.

## E. End State Visibility

- [ ] Win state banner is visible when win condition is reached.
- [ ] Lose state banner is visible when doom reaches loss condition.
- [ ] No stale "your turn" indicator persists after game over.

## F. Accessibility and Readability Baseline

- [ ] Core HUD labels remain legible at 800x600.
- [ ] Critical information (turn owner, actions left, doom) remains visible at all times.
- [ ] No overlapping HUD text in connect, selection, and active gameplay scenes.

## Execution Notes

- Desktop run: `go run ./cmd/desktop -server ws://localhost:8080/ws`
- WASM run: build with `GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web` and open `/play`
- Server run: `go run . server`
- Automated baseline checks:
  - `xvfb-run -a go test -race ./...`
  - `go vet ./...`
