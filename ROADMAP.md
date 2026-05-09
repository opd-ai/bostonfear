# Goal-Achievement Assessment

> Generated: 2026-05-08  
> Inputs: README claims, repository inspection, go-stats-generator metrics, `go test -race ./...`, `go vet ./...`, brief external research

## Project Context

- **What it claims to do**: Arkham Horror-themed multiplayer game with a Go WebSocket server and browser/Go clients, implementing core mechanics (Location, Resources, Actions, Doom, Dice), supporting 1-6 players with join-in-progress, reconnection, and observability endpoints.
- **Target audience**: Intermediate developers learning client-server WebSocket architecture and cooperative game mechanics.
- **Architecture (observed package layers)**:
	- `cmd/server`: composition root and runtime wiring
	- `transport/ws`: WebSocket HTTP upgrade and connection wrappers over `net.Conn`
	- `serverengine`: gameplay runtime, mechanics, turn handling, monitoring emitters
	- `serverengine/common/*`: cross-module contracts/runtime primitives
	- `serverengine/arkhamhorror`: module binding layer (compatibility-first)
	- `client/` HTML/JS: browser client in migration mode
	- `client/ebiten*` + `cmd/desktop|web|mobile`: Go/Ebitengine clients (desktop/WASM active, mobile alpha)
	- `protocol` and `monitoringdata`: shared DTO/wire schemas
	- `monitoring`: `/dashboard`, `/metrics`, `/health` handlers
- **Existing CI/quality gates**:
	- GitHub Actions: `go vet ./...`, `go test -race -tags=requires_display ./...` with Xvfb
	- Build checks for server, desktop, and WASM targets
	- Benchmark artifact generation and latency threshold gate for `BenchmarkBroadcastLatency`
	- Makefile parity for build/test/vet/display tests

## Goal-Achievement Summary

| Stated Goal | Status | Evidence | Gap Description |
|-------------|--------|----------|-----------------|
| 4-location adjacency movement restrictions | ✅ Achieved | `validateMovement` + adjacency map enforcement in server engine | No functional gap identified |
| Resource bounds for Health/Sanity/Clues with gain/loss | ✅ Achieved | `validateResources` clamps resources and tests cover edge behavior | No functional gap identified |
| 2-actions-per-turn action system | ✅ Achieved | `processActionCore` decrements `ActionsRemaining`, validates turn ownership, advances turn | No functional gap identified |
| Doom counter integration with tentacles and lose condition | ✅ Achieved | Doom updates integrated in action execution and checked against lose threshold | No functional gap identified |
| Dice resolution with configurable thresholds | ✅ Achieved | `rollDice` / `rollDicePool` with focus-spend behavior and tests | No functional gap identified |
| 1-6 players, join-in-progress | ✅ Achieved | `MinPlayers`/`MaxPlayers` enforcement + connection tests for join/rejoin paths | No functional gap identified |
| Reconnection/session persistence | ✅ Achieved | Reconnect token flow in server and clients (JS + Ebitengine state storage) | No functional gap identified |
| Interface-oriented networking (`net.Conn`, `net.Listener`, `net.Addr`) | ✅ Achieved | `transport/ws` wrappers and contracts rely on interface types, not concrete TCP structs | No functional gap identified |
| Real-time synchronization and observability endpoints | ✅ Achieved | Broadcast path + `/dashboard`, `/metrics`, `/health`; benchmark and monitoring tests present | No functional gap identified |
| Performance claim: stable with 6 players for 15+ minutes | ⚠️ Partial | Stress/benchmark coverage exists, latency threshold is CI-gated | No automated 15-minute soak gate in CI to prove stated duration continuously |
| Mobile support (iOS/Android) | ⚠️ Partial | Mobile binding entrypoint exists and build commands are documented | README itself states mobile is alpha/not verified on devices |
| Clear package separation between common runtime and Arkham rules | ⚠️ Partial | Module scaffolding exists and dependency direction is documented | Core runtime logic still concentrated in `serverengine` compatibility layer rather than fully moved under `serverengine/arkhamhorror/*` |
| Educational readability/documentation for intermediate developers | ⚠️ Partial | Rich README and rules/spec docs exist | `go-stats-generator` reports 67.6% overall doc coverage (types/methods at ~63%), below ideal for instructional codebase |

**Overall: 9/13 goals fully achieved, 4/13 partially achieved, 0 missing**

## Quantitative Snapshot (from `go-stats-generator analyze . --skip-tests`)

- Code size: 3,220 LoC across 68 files / 28 packages
- Behavioral complexity:
	- Avg function complexity: 3.92 (healthy baseline)
	- Highest function complexity: `processActionCore` at 16.6
	- Other hotspot functions: `cleanupDisconnectedPlayers` 14.7, `runMythosPhase` 13.2, `resolveEventEffect` 13.2
- Duplication: 0 clone pairs detected
- Documentation:
	- Overall: 67.58%
	- Functions: 96.30%
	- Types: 63.81%
	- Methods: 63.22%
- Organization signals:
	- Oversized files reported: 8
	- Oversized packages reported: 2 (`serverengine`, `app`)
	- Unreferenced functions: 4
- Baseline health checks:
	- `go test -race ./...` passed
	- `go vet ./...` passed

## Brief External Context (Phase 1)

- Project signal:
	- GitHub shows no open issues and no open PRs in `opd-ai/bostonfear`.
	- GitHub Security tab reports no published advisories and no security policy file.
- Dependency posture:
	- Direct dependencies (`github.com/gorilla/websocket`, `github.com/hajimehoshi/ebiten/v2`) are stable tagged releases.
	- `go list -m -u all` reports newer versions for some indirect/transitive modules (e.g., `golang.org/x/*`, `github.com/ebitengine/purego`).
- Comparable landscape:
	- `boardgame.io` provides a mature, high-level turn-based multiplayer framework with large community traction.
	- Implication for BostonFear: value is strongest in Go-first, mechanics-forward educational transparency and explicit engine architecture, so roadmap should prioritize proof of reliability and instructional clarity.

## Roadmap

### Priority 1: Prove the 15+ Minute Stability Claim in Automation

**Why this is first**: It is a top-level README promise that affects all multiplayer users and trust in core functionality.

- [x] Add a CI-selectable long soak test profile that drives 6 concurrent players for >=15 minutes with periodic actions and reconnect churn.
- [x] Gate on explicit pass/fail assertions: no deadlocks, no stuck turns, doom/resource bounds preserved, no goroutine leak growth trend, and bounded sync latency.
- [x] Emit machine-readable soak metrics as artifacts (latency distribution, reconnect success ratio, action throughput, final game-state invariants).
- [x] Keep default CI fast by running long soak nightly or on a dedicated workflow trigger.
- [x] Validation: a reproducible CI artifact demonstrates sustained correctness for the full claimed duration.

### Priority 2: Close the Mobile Gap from "Alpha Scaffolding" to "Verified"

**Why this is second**: README markets mobile targets but currently documents them as unverified; this is the largest user-visible platform gap.

- [ ] Execute Android and iOS binding outputs on real devices/emulators with a documented minimum support matrix.
- [ ] Verify touch input parity for move/gather/investigate/ward and reconnect-token reclaim behavior.
- [ ] Add a smoke checklist (connect, two turns, disconnect/reconnect, game over path) for each mobile platform.
- [ ] Update README build table with verified status, known constraints, and tested SDK/toolchain versions.
- [ ] Validation: reproducible mobile runbook + evidence of successful runtime behavior on both platforms.

### Priority 3: Complete Module Migration to Match Claimed Separation

**Why this is third**: The architecture claim is directionally true, but compatibility layering still centralizes too much logic in `serverengine`, increasing long-term change risk.

- [ ] Define and track a migration map from `serverengine/*` gameplay paths into `serverengine/arkhamhorror/{actions,phases,rules,model}`.
- [ ] Move one vertical slice at a time (e.g., action dispatch + validation + tests) to avoid broad regressions.
- [ ] Keep `serverengine/common/*` strictly game-family-agnostic and enforce dependency direction in CI.
- [ ] Add module-level package tests that assert behavior parity before/after each migration slice.
- [ ] Validation: primary Arkham mechanics execute from module-owned packages with no behavior drift and preserved test pass rate.

### Priority 4: Raise Documentation Quality to Match Educational Positioning

**Why this is fourth**: The project targets intermediate developers; current type/method docs are the clearest measurable mismatch with that mission.

- [ ] Raise overall doc coverage from 67.6% to >=80%, prioritizing exported types/methods in `serverengine`, `transport/ws`, `client/ebiten/app`, and `protocol`.
- [ ] Add mechanics-flow docs that map each user action to dice, doom, resource, and broadcast side effects.
- [ ] Resolve documentation drift where specs reference non-existent planning files.
- [ ] Introduce a lightweight docs lint gate for exported API comments.
- [ ] Validation: updated coverage metrics plus newcomer walkthroughs requiring no code spelunking to understand core flows.

### Priority 5: Dependency and Security Hygiene Hardening (Low Effort, Ongoing)

**Why this is fifth**: Not a functional blocker, but improves maintainability and contributor confidence.

- [ ] Add a minimal `SECURITY.md` with reporting/contact policy.
- [ ] Schedule periodic dependency update sweeps (especially indirect `golang.org/x/*` and ebitengine-adjacent modules).
- [ ] Keep changelog/release notes concise for compatibility-impacting updates.
- [ ] Validation: security policy visible in repository and regular dependency refresh cadence established.

## Execution Order Rationale

1. Reliability proof first because it validates a headline gameplay promise for all users.
2. Platform verification next because unverified mobile status is the most visible capability gap.
3. Architecture completion third because it reduces future regression risk while preserving current behavior.
4. Documentation uplift fourth because educational value scales with clarity once runtime confidence is established.
5. Hygiene last because it is important but less user-critical than gameplay correctness and platform readiness.

## Re-Assessment Checklist (After Roadmap Execution)

- [ ] `go test -race ./...` and `go vet ./...` remain clean
- [ ] 15+ minute 6-player soak evidence is automated and reproducible
- [ ] Mobile targets have verified runtime evidence, not just build artifacts
- [ ] Module ownership boundaries are reflected in implementation, not just scaffolding docs
- [ ] Documentation coverage and onboarding clarity metrics improve measurably
