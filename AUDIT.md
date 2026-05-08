# IMPLEMENTATION GAP AUDIT — 2026-05-08

## Project Architecture Overview
- **Stated goal**: Arkham Horror multiplayer game with Go WebSocket server, legacy HTML/JS client, and active migration to Go/Ebitengine client (`README.md`).
- **Go packages** (`go list ./...`):
  - `github.com/opd-ai/bostonfear/cmd/server` — authoritative game engine, networking, turn/action rules, observability.
  - `github.com/opd-ai/bostonfear/client/ebiten` — client network/state mirror.
  - `github.com/opd-ai/bostonfear/client/ebiten/app` — scene/input/game loop.
  - `github.com/opd-ai/bostonfear/client/ebiten/render` — atlas/layers/shaders.
  - `github.com/opd-ai/bostonfear/cmd/desktop`, `cmd/mobile`, `cmd/web` — platform entrypoints.
- **Dependency graph highlights** (`go list -f '{{.ImportPath}}|{{join .Imports \",\"}}' ./...`):
  - `cmd/desktop` and `cmd/mobile` depend on `client/ebiten/app`.
  - `client/ebiten/app` depends on `client/ebiten` + `client/ebiten/render`.
  - `cmd/server` is independent from Ebitengine client packages.
- **Interfaces/API surface**:
  - `Scene` (3 implementations), `Broadcaster` (1 implementation), `StateValidator` (implemented by `GameStateValidator` in code usage).
- **External planning context (Phase 1)**:
  - No open GitHub issues found.
  - Recent merged PRs (#7–#10) show incremental closure of prior audit gaps and migration work; no separate external roadmap was discovered.
- **Baseline quality commands (Phase 2)**:
  - `go-stats-generator analyze` completed.
  - `go build ./...` and `go vet ./...` failed in this sandbox due missing system X11 headers required by Ebitengine (`X11/Xlib.h`), not due reported Go compile errors in project code paths.

## Gap Summary
| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 1 | 0 | 1 | 0 | 0 |
| Dead Code | 2 | 0 | 0 | 1 | 1 |
| Partially Wired | 3 | 1 | 2 | 0 | 0 |
| Interface Gaps | 0 | 0 | 0 | 0 | 0 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package
| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|-------------------:|------------:|------:|-----:|---------:|
| github.com/opd-ai/bostonfear/cmd/server | 18 | 18 | 0 | 0 | 100% |
| github.com/opd-ai/bostonfear/client/ebiten | 16 | 16 | 0 | 0 | 100% |
| github.com/opd-ai/bostonfear/client/ebiten/app | 12 | 11 | 1 | 1 | 92% |
| github.com/opd-ai/bostonfear/client/ebiten/render | 10 | 10 | 0 | 0 | 100% |
| github.com/opd-ai/bostonfear/cmd/mobile | 2 | 2 | 0 | 0 | 100% |
| github.com/opd-ai/bostonfear/cmd/desktop | 0 | 0 | 0 | 0 | 100% |
| github.com/opd-ai/bostonfear/cmd/web | 0 | 0 | 0 | 0 | 100% |

## Findings
### CRITICAL
- [ ] Legacy browser client JavaScript fails to parse due duplicated method body fragment — `client/game.js:397` — code begins with `const resultDiv = this.diceResult;` in class scope after method close, producing syntax error and preventing any legacy client execution (`node --check client/game.js` fails) — **Blocked goal:** active legacy browser client in README build targets — **Remediation:** remove duplicated stray block (`client/game.js:397-434`) so class contains only valid methods; validate with `node --check client/game.js` and manual browser load of `/`.

### HIGH
- [ ] Character-select scene is documented as deferred and not wired in Ebitengine scene flow — `client/ebiten/app/scenes.go:7-8`, `client/ebiten/app/scenes.go:108-112` — client transitions directly `SceneConnect -> SceneGame` without `SceneCharacterSelect` despite CLIENT_SPEC-required four-scene flow — **Blocked goal:** intended Ebitengine UX/state machine migration — **Remediation:** add `SceneCharacterSelect` implementation and route transition logic in `updateScene`; validate with focused scene transition tests and `go test ./client/ebiten/app/...`.
- [ ] Connection quality self-status update never triggers due key mismatch (`playerID` vs `playerId`) — `client/game.js:462`, `cmd/server/dashboard.go:31` — server emits `playerId` JSON, client checks `qualityMessage.playerID`, so own badge update path is unreachable — **Blocked goal:** real-time connection quality indicator behavior — **Remediation:** align client key read to `playerId`; validate by simulating `connectionQuality` message and observing status badge updates.
- [ ] Difficulty configuration includes `ExtraDoomTokens` but runtime ignores it — `cmd/server/game_constants.go:215-224`, `cmd/server/game_mechanics.go:113-120` — only `InitialDoom` is applied; mythos token/cup composition is unchanged across difficulties — **Blocked goal:** modular difficulty settings in RULES/ROADMAP intent — **Remediation:** extend game state/setup to apply `ExtraDoomTokens` to Mythos token pool resolution; validate with deterministic tests for easy/standard/hard token composition behavior.

### MEDIUM
- [ ] Helper functions present in production file but unused by runtime code paths — `client/ebiten/app/game.go:324`, `client/ebiten/app/game.go:334` — `playerColourIndex` and `min8` are only referenced by tests, adding maintenance noise on maintained path — **Blocked goal:** none directly; increases confusion and drift risk — **Remediation:** either remove helpers and adjust tests, or integrate them into rendering paths; validate with `go test ./client/ebiten/app/...`.

### LOW
- [ ] Unused protocol base type remains in server type surface — `cmd/server/game_types.go:31` — `Message` struct is defined but not used by server message handling pipeline, which uses typed message structs/map envelopes directly — **Blocked goal:** none directly; increases API surface ambiguity — **Remediation:** remove unused type or route envelope decoding through it consistently; validate with `go test ./cmd/server/...`.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| `cmd/mobile/binding.go:44` `Dummy()` is an empty function | Intentional export anchor required by `ebitenmobile bind`; documented in-file and covered by `cmd/mobile/binding_test.go`. |
| `StateValidator` flagged as having 0 implementations by static metrics | Concrete implementation exists (`GameStateValidator`) and is injected/used in `NewGameServer` (`cmd/server/game_server.go:107`) and health/recovery paths. |
| Multiple `return nil` one-liners in scene/server methods | Verified as valid control-flow returns, not placeholders; functions have surrounding logic matching documentation. |
