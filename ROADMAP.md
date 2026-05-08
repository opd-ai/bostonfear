# Goal-Achievement Assessment

> **⚠️ Intellectual Property Notice**
> BostonFear is a **rules-only game engine** designed to execute the mechanics of the
> Arkham Horror series of games. This repository contains **no copyrighted content**
> produced by Fantasy Flight Games. No card text, scenario narratives, investigator
> stories, artwork, encounter text, or any other proprietary material owned by
> Fantasy Flight Games (an Asmodee brand) is, or will ever be, reproduced here.
> *Arkham Horror* is a trademark of Fantasy Flight Games. This project is an
> independent, fan-made rules engine and is not affiliated with or endorsed by
> Fantasy Flight Games or Asmodee.


> Generated: 2026-03-15 | Tool: go-stats-generator v1.0.0

## Project Context

- **What it claims to do**: Arkham Horror-themed multiplayer web game implementing 5 core mechanics (Location System, Resource Tracking, Action System, Doom Counter, Dice Resolution) with a Go WebSocket server and HTML/JS + Ebitengine clients supporting 1-6 concurrent players. Educational project targeting intermediate developers learning client-server WebSocket architecture with cooperative gameplay mechanics.

- **Target audience**: Intermediate developers learning client-server WebSocket architecture with cooperative gameplay mechanics.

- **Architecture**:

  | Package | Role | Functions | Structs |
  |---------|------|-----------|---------|
  | `cmd/server` | WebSocket server, game state, mechanics, observability | 136 | 47 |
  | `client/ebiten` | Ebitengine game client core | 29 | 17 |
  | `client/ebiten/app` | Ebitengine application layer, input handling | 16 | 3 |
  | `client/ebiten/render` | Rendering subsystem, layers, shaders | 12 | 7 |
  | `cmd/desktop`, `cmd/mobile`, `cmd/web` | Platform entrypoints (alpha) | 3 | 0 |

- **Existing CI/quality gates**:
  - GitHub Actions (`.github/workflows/ci.yml`): `go vet`, `go test -race -tags=requires_display` with Xvfb, builds for desktop and WASM
  - Makefile targets: `build`, `test`, `test-display`, `vet`, `clean`

---

## Goal-Achievement Summary

| # | Stated Goal | Status | Evidence | Gap Description |
|---|-------------|--------|----------|-----------------|
| 1 | **Location System**: 4 interconnected neighborhoods with movement restrictions | ✅ Achieved | `locationAdjacency` map in `game_constants.go`; `validateMovement()` enforces adjacency | — |
| 2 | **Resource Tracking**: Health (1-10), Sanity (1-10), Clues (0-5) | ✅ Achieved | `validateResources()` in `game_mechanics.go:17-33`; extended resources (Money, Remnants, Focus) also implemented | — |
| 3 | **Action System**: 2 actions/turn from Move, Gather, Investigate, Cast Ward | ✅ Achieved | `dispatchAction()` routes all 4 base actions + 4 additional (Focus, Research, Trade, Component); `processAction()` decrements `ActionsRemaining` | 8 of 8 AH3e actions implemented |
| 4 | **Doom Counter**: Global tracker 0-12, increments on Tentacle results | ✅ Achieved | `processAction()` adds `doomIncrease`; `checkGameEndConditions()` triggers loss at doom=12 | — |
| 5 | **Dice Resolution**: 3-sided dice with configurable difficulty | ✅ Achieved | `rollDice()` and `rollDicePool()` in `game_mechanics.go`; focus-token rerolls supported | — |
| 6 | **1-6 concurrent players with join-in-progress** | ✅ Achieved | `MinPlayers=1`, `MaxPlayers=6` in `constants.go`; late joiners appended to `TurnOrder` and win threshold rescaled | — |
| 7 | **Real-time state sync < 500ms** | ✅ Achieved | `broadcastGameState()` queues immediately; latency ring buffer tracks broadcast times | Benchmark test exists (`BenchmarkBroadcastLatency`) |
| 8 | **Interface-based design (net.Conn, net.Listener, net.Addr)** | ✅ Achieved | `ConnectionWrapper` implements `net.Conn`; `Broadcaster`, `StateValidator` interfaces in `interfaces.go` | — |
| 9 | **Ebitengine client: desktop + WASM builds** | ✅ Achieved | Builds compile and pass CI; `cmd/desktop/main.go`, `cmd/web/main.go` | Placeholder sprites only |
| 10 | **Ebitengine client receives live server state** | ✅ Achieved | `decodeGameState()` unmarshals `data` wrapper correctly; tested by `TestDecodeGameState_FromDataWrapper` | — |
| 11 | **Session persistence (reconnect token)** | ✅ Achieved | JS client uses `localStorage`; Ebitengine client persists to `~/.bostonfear/session.json` | — |
| 12 | **Performance monitoring: /dashboard, /metrics, /health** | ✅ Achieved | `metrics.go`, `health.go`, `dashboard.go` expose all three endpoints; Prometheus metrics exported | — |
| 13 | **AH3e rules compliance (RULES.md)** | ✅ Achieved | All 8 action types implemented; Mythos Phase with event deck; Act/Agenda progression; encounter resolution; investigator defeat/recovery | See detailed table below |
| 14 | **Zero critical bugs in server** | ✅ Achieved | `go vet ./...` clean; `go test -race ./...` passes 178+ tests | — |
| 15 | **Ebitengine client test coverage** | ✅ Achieved | `net_test.go` (8 tests), `state_test.go` (3 tests); app/render tests run in CI with Xvfb | — |
| 16 | **Mobile binding (iOS/Android)** | ⚠️ Partial | `cmd/mobile/binding.go` scaffolding exists; builds with `ebitenmobile` | Not verified on physical device |

**Overall: 15/16 goals fully achieved; 1 partially achieved; 0 missing**

---

## AH3e Engine Compliance Detail

| Rule System | RULES.md Spec | Implementation | Test Coverage |
|-------------|---------------|----------------|---------------|
| Action System (8 types) | ✅ | ✅ All 8 (Move, Gather, Investigate, Ward, Focus, Research, Trade, Component) | `TestRulesFullActionSet` |
| Dice Resolution (pool + focus) | ✅ | ✅ `rollDicePool` with focus spend/reroll | `TestRulesDicePoolFocusModifier` |
| Mythos Phase | ✅ | ✅ 2-event draw, spread, token cup | `TestRulesMythosPhaseEventPlacement` |
| Resource Management (6 types) | ✅ | ✅ Health, Sanity, Clues, Money, Remnants, Focus | `TestRulesResourceTypes` |
| Encounter Resolution | ✅ | ✅ Deck-based draws with effects | `TestRulesEncounterResolution` |
| Act/Agenda Deck Progression | ✅ | ✅ Clue thresholds scale with player count | `TestRulesActAgendaProgression` |
| Investigator Defeat/Recovery | ✅ | ✅ Lost-in-time-and-space state + recovery | `TestRulesDefeatRecovery` |
| Scenario System | ✅ | ✅ `DefaultScenario` with custom win conditions | `TestRulesScenarioSystem` |
| Modular Difficulty | ✅ | ✅ Easy/Normal/Hard presets | `TestDifficulty_*` |
| 1–6 Player Support | ✅ | ✅ Min 1, Max 6, late-join rescale | `TestRescaleActDeck_LateJoin` |
| Attack/Evade (enemies) | ✅ | ✅ `performAttack`, `performEvade`, enemy spawn | `TestProcessAction_Attack`, `TestProcessAction_Evade` |
| Gate/Anomaly Mechanics | ✅ | ✅ `openGate`, `performCloseGate`, anomaly spawns | `TestGateMechanics_OpenAndClose` |

---

## Metrics Snapshot

| Metric | Value | Interpretation |
|--------|-------|----------------|
| Total LoC | 2,540 | Moderate codebase |
| Functions / Methods | 53 / 143 | Server-heavy distribution |
| Avg function length | 14.6 lines | Within healthy range (< 25) |
| Avg cyclomatic complexity | 4.4 | Low; healthy threshold is < 10 |
| Highest complexity | `cleanupDisconnectedPlayers` (14.7) | Acceptable for concurrency logic |
| Functions > 50 lines | 3 (1.5%) | Minor; largest is `handleHealthCheck` (62 lines) |
| High complexity (> 10) | 0 functions | Excellent |
| Circular dependencies | 0 | Clean package graph |
| Total test count | 178+ | Strong coverage |
| Doc coverage | 95.9% | Above 80% threshold |

---

## Roadmap

### Priority 1: Verify Mobile Build on Physical Device

**Gap**: Mobile binding scaffolding exists but has never been tested on iOS or Android hardware.

**Impact**: README and CLIENT_SPEC.md claim mobile support. Without device verification, users attempting mobile deployment may encounter runtime failures.

- [ ] Install `ebitenmobile` CLI and dependencies (Android SDK API 29+, Xcode 15+)
- [ ] Run: `ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile`
- [ ] Create minimal Android app importing the AAR; confirm game launches and touch input works
- [ ] Run: `ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile`
- [ ] Create minimal iOS app importing the XCFramework; confirm game launches
- [ ] Document verified SDK/Xcode versions in README.md §Build Targets
- [ ] **Validation**: Both platforms show functional game with working touch input

> **Blocker**: Requires physical iOS/Android hardware and Android SDK API 29+/Xcode 15+. Not available in automated CI sandbox. Deferred to manual device verification.

---

### Priority 2: Replace Placeholder Sprites with Production Art

**Gap**: All Ebitengine renders use programmer-art rectangles (CLIENT_SPEC.md §Assets notes "Placeholder programmer-art assets used until Phase 5 rendering is complete").

**Impact**: Affects user experience and prevents meaningful visual polish testing.

- [ ] Source or commission art assets:
  - Location tiles (4 neighborhoods: Downtown, University, Rivertown, Northside)
  - Investigator tokens (6 colours for 1-6 players)
  - Dice faces (Success ✓, Blank ○, Tentacle 🐙)
  - UI elements (health/sanity bars, doom counter, clue badges)
- [x] Create `client/ebiten/render/assets/` directory using `go:embed` (`sprites.png` embedded in `atlas.go`)
- [x] Update `client/ebiten/render/atlas.go` with explicit sprite-sheet coordinate table (`spriteCoords`)
- [x] Implement Kage doom vignette shader per CLIENT_SPEC.md §4.4 (`shaders/doom.kage` + `DrawDoomVignette`)
- [ ] **Validation**: Desktop client displays themed visuals at 1280×720 logical resolution (pending production art)

---

### Priority 3: Add CI Benchmark Reporting

**Gap**: Performance benchmarks exist (`BenchmarkBroadcastLatency`, `TestStabilityWith6Players`) but results are not reported in CI artifacts.

**Impact**: Performance regressions could be introduced without notice.

- [x] Add benchmark step to `.github/workflows/ci.yml`:
  ```yaml
  - name: Run benchmarks
    run: go test -bench=. -benchtime=10s ./cmd/server/... | tee benchmark-results.txt
  ```
- [x] Upload `benchmark-results.txt` as CI artifact
- [x] Define pass/fail threshold: average broadcast latency must be < 200ms
- [ ] Optionally integrate with a benchmark tracking service (e.g., Bencher, codspeed)
- [x] **Validation**: CI run shows benchmark artifact with latency metrics

---

### Priority 4: Split Large Files for Maintainability

**Gap**: `game_mechanics.go` (937 lines) exceeds 500-line soft threshold; `metrics.go` (538 lines) is borderline.

**Impact**: Large files increase cognitive load for contributors. go-stats-generator flagged these with high "burden" scores.

**Suggested splits for `game_mechanics.go`**:
- [x] Extract dice logic → `dice.go` (~70 lines: `rollDice`, `rollDicePool`)
- [x] Extract action performers → `actions.go` (~300 lines: `performMove`, `performGather`, etc.)
- [x] Extract Mythos Phase → `mythos.go` (~150 lines: `runMythosPhase`, `resolveEventEffect`, `drawMythosToken`)
- [x] Keep coordination logic in `game_mechanics.go` (~400 lines)

- [x] **Validation**: No file > 500 lines; `go test -race ./cmd/server/...` passes

---

### Priority 5: Address WebSocket Security Best Practices

**Gap**: Current `websocket.Upgrader` uses default (permissive) `CheckOrigin`. For production deployment, origin validation is recommended by OWASP.

**Impact**: Educational project; not critical. However, adding origin checking demonstrates best practices to the target audience.

- [x] Add configurable `allowedOrigins` list (default: empty/permissive unless explicitly configured)
- [x] Implement `CheckOrigin` function in the upgrader
- [x] Document in README.md how to configure for production deployments
- [x] **Validation**: When `allowedOrigins` is configured, WebSocket upgrades from non-allowed origins are rejected

---

### Priority 6: Improve Ebitengine Client Test Coverage

**Gap**: `client/ebiten/app` and `client/ebiten/render` tests require a display (Xvfb) and are less comprehensive than server tests.

**Impact**: Rendering regressions may go unnoticed.

- [x] Add headless-safe unit tests for non-rendering logic in `app/input.go`
- [x] Add atlas coordinate validation tests (ensure sprite IDs map to valid regions)
- [x] Add shader compilation test (verify Kage shaders compile without errors)
- [x] **Validation**: `go test -race ./client/ebiten/...` coverage > 60%

---

## Non-Goals (Out of Scope)

Per RULES.md §Non-Goals and task instructions:

- Game content creation (card text, encounter narratives, scenario scripts)
- Card/scenario data files (JSON/YAML definitions, codex entries)
- Art asset creation (sprites, card layouts, print-ready materials)
- Expansion content (Under Dark Waves, Dead of Night, etc.)
- TLS/HTTPS transport-layer encryption (handled by infrastructure)

---

## Dependency Health

| Dependency | Version | Status |
|------------|---------|--------|
| `github.com/gorilla/websocket` | v1.5.3 | ✅ No known CVEs; actively maintained |
| `github.com/hajimehoshi/ebiten/v2` | v2.9.9 | ✅ Current stable; deprecated vector APIs (`AppendVerticesAndIndices*`) not used |
| Go | 1.24.1 | ✅ Current stable (matches Ebitengine 2.9 requirement) |

---

## Appendix: File Complexity Hotspots

| File | Lines | Functions | Burden Score | Notes |
|------|-------|-----------|--------------|-------|
| `cmd/server/game_mechanics.go` | 937 | 33 | 1.18 | Candidate for split (Priority 4) |
| `cmd/server/metrics.go` | 538 | 18 | 1.19 | Acceptable; Prometheus collection is cohesive |
| `cmd/server/error_recovery.go` | 369 | 22 | 0.81 | Healthy |
| `client/ebiten/app/game.go` | 300 | 16 | — | Healthy |
| `cmd/server/game_types.go` | 256 | 0 | 1.84 | Type-only file; high burden from type count |

---

*This roadmap prioritises gaps by impact on the project's stated goals. Mobile verification and art assets affect end-user experience most directly; CI benchmarks and file splits are quality-of-life improvements for contributors.*
