# IMPLEMENTATION GAP AUDIT - 2026-05-08

## Project Architecture Overview

BostonFear is a rules-only Arkham Horror engine with three active runtime layers and a larger migration scaffold around them.

- `serverengine` owns the active Arkham runtime: turn processing, action resolution, mythos progression, connection lifecycle, metrics, and health.
- `transport/ws` owns HTTP route registration, WebSocket upgrade, and the `net.Listener`-based server edge.
- `monitoring`, `monitoringdata`, and `protocol` provide HTTP monitoring endpoints, DTOs, and the shared JSON wire contract.
- `client/index.html` + `client/game.js` are the active browser client.
- `client/ebiten`, `client/ebiten/app`, and `client/ebiten/render` are the alpha Go/Ebitengine client for desktop, WASM, and mobile binding.
- `serverengine/arkhamhorror` is the intended long-term Arkham-specific module boundary, but most Arkham logic still lives in the root `serverengine` package.
- `serverengine/common/*`, `serverengine/eldersign/*`, `serverengine/eldritchhorror/*`, and `serverengine/finalhour/*` are mostly migration scaffolds and placeholder module families.

### Stated Goals Evaluated

- The repository README positions the project as an Arkham Horror multiplayer engine with an active browser client and an active alpha Ebitengine client.
- `docs/RULES.md` claims 13/13 core rule systems are fully implemented.
- `docs/CLIENT_SPEC.md` defines a richer four-scene Ebitengine client contract than the current code actually delivers.
- `serverengine/arkhamhorror/README.md` states that Arkham-specific rules should move out of shared packages and into the Arkham module tree.

### Phase 1 Online Research

Brief GitHub review found no open issues, no open projects, no milestones, and no open pull requests. The only public planning signal is in-repository documentation (`README.md`, `ROADMAP.md`, `docs/CLIENT_SPEC.md`, `docs/RULES.md`). There is no external tracker to explain away the gaps below as already tracked public backlog items.

## Baseline Evidence

- Prerequisite satisfied: `go-stats-generator` installed and used.
- `go build ./...`: clean.
- `go vet ./...`: clean.
- `go-stats-generator analyze . --skip-tests`:
  - 2,978 lines of Go code
  - 67 files processed
  - 69 functions, 195 methods, 62 structs, 7 interfaces
  - 67.0% overall documentation coverage
  - 0 TODO/FIXME/HACK/XXX annotations detected
  - 4 unreferenced-function candidates reported by the tool; only high-confidence items survived false-positive review
- Additional validation performed for non-Go surface:
  - `node --check client/game.js` fails with `SyntaxError: Unexpected identifier 'resultDiv'`

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 2 | 0 | 0 | 0 | 2 |
| Dead Code | 1 | 0 | 0 | 1 | 0 |
| Partially Wired | 3 | 1 | 1 | 1 | 0 |
| Interface Gaps | 1 | 0 | 0 | 0 | 1 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package

Coverage below is estimated implementation completeness for the package's intended role, not test coverage.

| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|--------------------|-------------|-------|------|----------|
| `client/ebiten` | 17 | 17 | 0 | 0 | 90% |
| `client/ebiten/app` | 15 | 15 | 0 | 0 | 75% |
| `client/ebiten/render` | 10 | 10 | 0 | 0 | 100% |
| `monitoring` | 4 | 4 | 0 | 0 | 100% |
| `serverengine` | 18 | 18 | 0 | 0 | 85% |
| `serverengine/arkhamhorror` | 4 | 4 | 0 | 0 | 100% |
| `serverengine/common/runtime` | 17 | 17 | 0 | 0 | 100% |
| `serverengine/eldersign` | 4 | 3 | 1 | 0 | 75% |
| `serverengine/eldritchhorror` | 4 | 3 | 1 | 0 | 75% |
| `serverengine/finalhour` | 4 | 3 | 1 | 0 | 75% |
| `transport/ws` | 11 | 11 | 0 | 0 | 100% |
| `serverengine` doc-only scaffold packages (25 total) | 0 | 0 | 25 | 0 | 0% |

## Findings

### CRITICAL

- [x] **Active browser client cannot parse** - `client/game.js:397` - a second copy of the `displayDiceResult` body is embedded directly in the class body after `displayGameUpdate`, and `node --check client/game.js` fails with `SyntaxError: Unexpected identifier 'resultDiv'`. This blocks the README's active HTML/JS browser client path (`client/index.html`) from loading at all. **Remediation:** remove the duplicated block at `client/game.js:397-433`, add a syntax smoke check for browser assets, and validate with `node --check client/game.js` plus a manual load of `/` from the Go server.

### HIGH

- [x] **Pregame actions are unreachable in live sessions** - `serverengine/connection.go:110`, `serverengine/actions.go:388`, `serverengine/actions.go:405`, `serverengine/connection_test.go:66`, `docs/RULES.md:28`, `docs/RULES.md:32` - the first connected player starts the game immediately because `MinPlayers = 1`, which flips `GamePhase` from `waiting` to `playing`. At the same time, both `performSelectInvestigator` and `performSetDifficulty` reject any call unless the phase is still `waiting`. The rules docs mark both systems complete, but the real connection path makes them unreachable. **Remediation:** keep the game in `waiting` until explicit readiness/start criteria are met, or allow these actions before first-turn assignment in `playing`; add an integration test that performs `connect -> selectInvestigator` and `connect -> setDifficulty` through the real socket/server path; validate with `go test ./serverengine -run 'TestHandleWebSocket|TestRescaleActDeck|TestProcessAction_(SelectInvestigator|SetDifficulty)'` plus a new end-to-end test.

### MEDIUM

- [ ] **Ebitengine scene flow is only partially wired to the published client contract** - `docs/CLIENT_SPEC.md:62`, `docs/CLIENT_SPEC.md:68`, `docs/CLIENT_SPEC.md:79`, `docs/CLIENT_SPEC.md:93`, `client/ebiten/app/scenes.go:24`, `client/ebiten/app/scenes.go:94`, `client/ebiten/app/scenes.go:178` - `SceneConnect` is only an animated banner; it has no server-address input, no player-display-name input, no slot status, and no reconnect countdown/slot-wait state. `SceneCharacterSelect` sends a local selection, but `updateScene()` advances to `SceneGame` as soon as the local player has an investigator rather than when all connected players have confirmed. This leaves the alpha client short of its own published scene contract. **Remediation:** extend connect-state data in `LocalState` and the wire contract as needed, add connect-scene input/wait UI, and gate the `SceneCharacterSelect -> SceneGame` transition on all connected players having selected an investigator; validate with `go test -tags=requires_display ./client/ebiten/app/...` and a manual desktop run against a multi-player server.

### DEAD CODE

- [ ] **Browser self-quality update branch never fires** - `client/game.js:462`, `serverengine/connection_quality.go:24` - the browser client checks `qualityMessage.playerID`, but the server emits `playerId`. The per-player list still uses `allPlayerQualities`, but the top-right connection-status upgrade path is unreachable. **Remediation:** change the comparison to `qualityMessage.playerId === this.playerId`, add a fixture-driven client-side test for a `connectionQuality` payload, and validate by receiving a ping/pong cycle from a live server while watching the status badge change.

### LOW

- [ ] **Ebitengine connection-quality schema is stale relative to the server contract** - `client/ebiten/state.go:87`, `client/ebiten/state.go:89`, `client/ebiten/state.go:194`, `client/ebiten/net_test.go:102`, `client/ebiten/net_test.go:331`, `serverengine/connection_quality.go:12`, `serverengine/connection_quality.go:13`, `serverengine/connection_quality.go:24` - the Go client still expects `quality: { latency, rating }` and stores `cs.Quality.Rating`, while the server emits `quality: { latencyMs, quality }`. Current rendering does not consume the field, so this is not yet user-visible, but it codifies a stale protocol into tests and will break future quality-HUD work. **Remediation:** align `client/ebiten/state.go` and the net tests to `latencyMs`/`quality`, then add a round-trip test using an actual server-shaped payload; validate with `go test ./client/ebiten/...`.

- [ ] **Modular package decomposition is still scaffolding-only in 25 packages** - representative refs: `serverengine/arkhamhorror/actions/doc.go:1`, `serverengine/common/messaging/doc.go:1`, `serverengine/eldersign/adapters/doc.go:1` - the directory structure for Arkham subpackages, shared cross-engine packages, and other game families exists, but 25 packages are still `doc.go` placeholders with no implementation. This is intentional scaffolding, but it means the architecture advertised in `serverengine/arkhamhorror/README.md` has barely started. **Remediation:** either collapse unused scaffolds until migration restarts, or migrate one slice at a time beginning with `serverengine/arkhamhorror/model` and `serverengine/arkhamhorror/rules`; validate with `go list ./...` and package-level tests as slices move.

- [ ] **Selectable non-Arkham modules are startup stubs** - `cmd/server/main.go:29`, `cmd/server/main.go:30`, `cmd/server/main.go:31`, `cmd/server/main.go:32`, `serverengine/eldersign/module.go:22` - `eldersign`, `eldritchhorror`, and `finalhour` are registered as selectable modules, but each `NewEngine()` returns `runtime.NewUnimplementedEngine(...)`. The behavior is documented, so this is low severity, but module selection currently advertises options that terminate with a not-implemented error. **Remediation:** either remove these modules from the default registry until they have a playable engine, or gate them behind an explicit experimental flag; validate with `BOSTONFEAR_GAME=<module> go run ./cmd/server` for each registered module.

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| Reconnect tokens / session persistence looked missing because `docs/CLIENT_SPEC.md` cites a nonexistent `PLAN.md` and a 60-second grace period | Rejected as an implementation gap. The actual code does persist reconnect tokens in `client/ebiten/state.go` and reclaims slots in `serverengine/connection.go`; the doc is stale, not the runtime. |
| `BuildSystemAlerts` looked unreferenced from `go-stats-generator`'s dead-code candidates | Rejected after direct usage check: `monitoring/handlers.go:31` calls it in the `/health` response path. |
| `cmd/mobile`'s `SetServerURL` and `Dummy` looked like no-op exports | Rejected. `Dummy` is required by `ebitenmobile` binding generation, and `SetServerURL` is the native-host override point documented in `cmd/mobile/binding.go`. |
| Touch input appeared absent from the Ebitengine client based on older reports | Rejected after code read: `client/ebiten/app/input.go` has a real `handleTouchInput` implementation wired from `InputHandler.Update()`. |
| Empty scaffold packages under `serverengine/*` | Downgraded to LOW, not MEDIUM/HIGH, because the scaffolding is explicitly documented as migration-first structure in `serverengine/arkhamhorror/README.md`. |
| Comment debt (TODO/FIXME/HACK/XXX) | Rejected. `go-stats-generator` and repo-wide search both returned zero actionable TODO/FIXME/HACK/XXX markers in production Go code. |
