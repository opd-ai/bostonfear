# Implementation Plan: Ebitengine Migration & AH3e Rules Compliance

## Project Context

Migrating the game client from HTML/JS canvas to a Go/Ebitengine engine supporting desktop, web (WASM), and mobile build targets, while achieving 100% AH3e core rules compliance. The WebSocket server (`cmd/server/`, `gorilla/websocket`) is unchanged.

- **Current stack**: Go 1.24.1, `gorilla/websocket` server, HTML/JS canvas client
- **Target stack**: Go 1.24+, `gorilla/websocket` server (unchanged), `github.com/hajimehoshi/ebiten/v2` client with desktop/WASM/mobile targets
- **Scope**: Engine and client infrastructure only — no game content (cards, scenarios, lore)

---

## Prior Gap-Closure Plan (Steps 1–11) — Status

> The following steps were defined in the original plan to address gaps in `GAPS.md`.
> They are retained for history. Each step is annotated with its current status
> relative to the new `ROADMAP.md`.

| Step | Description | Status |
|---|---|---|
| 1 | Fix README setup instructions (entry-point path) | `[COMPLETED — retain for history]` README now shows `cd cmd/server && go run .` |
| 2 | Document win condition threshold in README and game state | `[COMPLETED — retain for history]` README shows 4 clues per investigator formula |
| 3 | Implement missing `gameUpdate` protocol message | `[SUPERSEDED by new ROADMAP]` Protocol changes deferred; Ebitengine client will implement all 5 message types from scratch |
| 4 | Fix `ConnectionWrapper` deadline methods (no-op violation) | `[SUPERSEDED by new ROADMAP]` Will be addressed as part of Phase 0 baseline |
| 5 | Implement reconnection token system | `[SUPERSEDED by new ROADMAP]` Reconnection will be re-implemented in the Ebitengine client |
| 6 | Implement real error rate tracking | `[SUPERSEDED by new ROADMAP]` Server metrics improvements deferred to post-migration |
| 7 | Implement real broadcast latency metrics | `[SUPERSEDED by new ROADMAP]` Server metrics improvements deferred to post-migration |
| 8 | Fix `/dashboard` relative path (HTTP 404) | `[SUPERSEDED by new ROADMAP]` Will be addressed as part of Phase 0 baseline |
| 9 | Define Go interfaces for core abstractions | `[SUPERSEDED by new ROADMAP]` Architecture refactoring will occur during Phase 6 rules compliance |
| 10 | Add context cancellation to goroutines | `[SUPERSEDED by new ROADMAP]` Will be addressed as part of Phase 0 baseline |
| 11 | Reduce complexity in top three hotspots | `[SUPERSEDED by new ROADMAP]` Complexity reduction will occur during Phase 6 refactoring |

---

## Phase 0: Pre-Migration Baseline

Before any Ebitengine work begins, the following conditions must be true:

### Prerequisites

1. **All critical GAPS.md items resolved** — server-side gaps that would block client testing must be closed:
   - `ConnectionWrapper` deadline methods must delegate correctly (GAPS.md §7)
   - `/dashboard` must serve from the correct path (GAPS.md §6)
   - `gameUpdate` protocol message must be emitted (GAPS.md §1)
2. **`go vet ./...` passes** with zero warnings on `cmd/server/`
3. **`go test ./...` passes** — all existing tests green
4. **`go.sum` is clean** — `go mod tidy` produces no diff
5. **Legacy client functional** — `client/index.html` + `client/game.js` can complete a full 2-player game against the server (manual smoke test)

### Acceptance Criteria

```bash
cd /workspaces/bostonfear
go mod tidy && git diff --exit-code go.sum
go vet ./...
go test ./...
cd cmd/server && go run . &
sleep 2 && curl -sf http://localhost:8080/ | grep -q "Arkham" && echo "PHASE_0_PASS"
kill %1
```

---

## Migration Plan

### Step M1: Add Ebitengine Dependency

- **Deliverable**: Add `github.com/hajimehoshi/ebiten/v2` (v2.7+) to `go.mod`. Run `go mod tidy` to resolve transitive dependencies.
- **Files**: `go.mod`, `go.sum`
- **Dependencies**: Phase 0 complete
- **Acceptance**: `go mod tidy` succeeds; `go.mod` lists `github.com/hajimehoshi/ebiten/v2 v2.7.x`; existing `go test ./...` still passes (no breaking transitive conflicts).

---

### Step M2: Create Ebitengine Client Package

- **Deliverable**: Create `client/ebiten/` package with a minimal `ebiten.Game` interface implementation that connects to the existing WebSocket server and mirrors game state.
- **Files**:
  - `client/ebiten/game.go` — `ebiten.Game` implementation (`Update`, `Draw`, `Layout`)
  - `client/ebiten/net.go` — WebSocket client using `gorilla/websocket`; JSON encode/decode for all 5 message types (`gameState`, `playerAction`, `diceResult`, `connectionStatus`, `gameUpdate`)
  - `client/ebiten/state.go` — local state mirror updated on `gameState` messages
  - `client/ebiten/input.go` — keyboard/mouse input mapped to player actions
- **Dependencies**: Step M1 (Ebitengine in `go.mod`)
- **Acceptance**: `go build ./client/ebiten/...` compiles. The client opens an 800×600 window, connects to the server, and displays placeholder rectangles for locations with text labels for player state.

---

### Step M3: Wire Desktop Entrypoint

- **Deliverable**: Create `cmd/desktop/main.go` that parses a `-server` flag (default `ws://localhost:8080/ws`), instantiates the `client/ebiten` game, and calls `ebiten.RunGame(game)`.
- **Files**: `cmd/desktop/main.go`
- **Dependencies**: Step M2
- **Acceptance**: `go build ./cmd/desktop` produces a single binary. Running it connects to the server and renders the game board. Two instances can join the same game and observe state sync.

---

### Step M4: Implement WASM Build Target

- **Deliverable**: Create `cmd/web/main.go` (compiled with `GOOS=js GOARCH=wasm`), `client/wasm/index.html` host page, and integrate `wasm_exec.js`.
- **Files**: `cmd/web/main.go`, `client/wasm/index.html`
- **Dependencies**: Step M2
- **Acceptance**: `GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web` succeeds. Serving `client/wasm/` over HTTP loads the game in Chrome/Firefox/Safari. The WASM client behaves identically to the desktop client.

---

### Step M5: Implement Mobile Build Target

- **Deliverable**: Create `cmd/mobile/mobile.go` using `ebitenmobile` binding conventions with touch input handling.
- **Files**: `cmd/mobile/mobile.go`
- **Dependencies**: Step M2
- **Acceptance**: `ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile` produces an AAR. `ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile` produces an xcframework. Touch input allows completing a full turn on both platforms.

---

### Step M6: Implement Ebitengine Rendering Layers

- **Deliverable**: Replace placeholder rectangle rendering with a layered sprite system: board layer, token layer, effect layer, UI overlay layer, animation layer.
- **Files**: `client/ebiten/render/atlas.go`, `client/ebiten/render/layers.go`
- **Dependencies**: Step M3 (desktop target functional for visual testing)
- **Acceptance**: All five rendering layers draw correctly with proper z-ordering. Logical resolution (1280×720) scales to desktop, web, and mobile displays without layout breakage. ≥60 FPS on desktop, ≥30 FPS on mobile.

---

### Step M7: Add Shader-Based Visual Effects

- **Deliverable**: Implement Kage shaders for fog-of-war and doom vignette effects.
- **Files**: `client/ebiten/render/shaders/fog.kage`, `client/ebiten/render/shaders/glow.kage`, `client/ebiten/render/shaders/doom.kage`
- **Dependencies**: Step M6
- **Acceptance**: At least two shaders compile and render without errors on all three platforms.

---

### Step M8: Implement Rules-Compliance Test Suite

- **Deliverable**: Create a comprehensive test suite validating all AH3e core rulebook mechanics against the engine implementation, covering all rule systems listed in `RULES.md`.
- **Files**: `cmd/server/rules_test.go` (or `rules/rules_test.go`)
- **Dependencies**: None (tests exercise existing server code)
- **Test coverage targets** (one test function minimum per rule system):
  1. `TestTurnStructure` — Investigator Phase → Mythos Phase cycle
  2. `TestMythosPhaseEventPlacement` — event draw, placement, spread, mythos token
  3. `TestFullActionSet` — all 8 action types (Move, Gather, Focus, Ward, Research, Trade, Component, Attack/Evade)
  4. `TestDicePoolFocusModifier` — focus token spend, skill-based pool adjustment
  5. `TestAnomalyGateMechanics` — anomaly spawning, sealing
  6. `TestEncounterResolution` — neighborhood-specific encounters, skill tests
  7. `TestActAgendaProgression` — act advancement on clues, agenda advancement on doom
  8. `TestDefeatRecovery` — investigator defeat at 0 health/sanity, lost-in-time-and-space state
  9. `TestVictoryDefeatConditions` — scenario-driven win/lose
  10. `TestResourceTypes` — money, clues, remnants, focus tokens
- **Acceptance**: `go test ./... -run TestRules` passes; 100% of AH3e core mechanics covered.

---

## Dependency Graph

```
Phase 0 (baseline)
  └── Step M1 (add Ebitengine dep)
        └── Step M2 (client/ebiten package)
              ├── Step M3 (desktop entrypoint)
              │     └── Step M6 (rendering layers)
              │           └── Step M7 (shaders)
              ├── Step M4 (WASM target)
              └── Step M5 (mobile target)

Step M8 (rules tests) — independent, can start in parallel with M1–M7
```

---

## Quick Reference

| Migration Step | ROADMAP Phase | Estimated Effort |
|---|---|---|
| M1 – Add Ebitengine dep | Phase 1 | ~30 min |
| M2 – client/ebiten package | Phase 1 | ~2 weeks |
| M3 – Desktop entrypoint | Phase 2 | ~2 days |
| M4 – WASM target | Phase 3 | ~3 days |
| M5 – Mobile target | Phase 4 | ~1 week |
| M6 – Rendering layers | Phase 5 | ~2 weeks |
| M7 – Shaders | Phase 5 | ~1 week |
| M8 – Rules test suite | Phase 6 | ~3 weeks |

---

*Updated: March 15, 2026. Cross-referenced with: `ROADMAP.md`, `GAPS.md`, `RULES.md`, `README.md`.*
