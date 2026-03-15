# Implementation Plan: Correctness Hardening → AH3e Rules Completion

> Generated: 2026-03-15 | Metrics source: `go-stats-generator v1.0.0`

## Project Context

- **What it does**: Arkham Horror-themed cooperative multiplayer web game — Go WebSocket
  server + HTML/JS and Ebitengine clients — targeting intermediate developers learning
  client-server WebSocket architecture with 1-6 concurrent players.
- **Current milestone**: All 5 core mechanics ship and pass 178 tests; 2 confirmed concurrency
  bugs undermine the monitoring subsystem under real game load; 3 mid-priority gaps erode
  documented gameplay contracts; AH3e rules compliance is at 7/8 action types.
- **Most important unachieved goal**: Eliminate the two reachable concurrency defects in
  `/health` and `/metrics` that silently hang the monitoring system under the project's
  own primary use-case load (6 concurrent players, actions every few seconds).
- **Estimated Scope**: **Medium** — 4 correctness fixes (5–15 items above threshold) +
  3 AH3e extension milestones.

---

## Goal-Achievement Status

| # | Stated Goal | Current Status | This Plan Addresses |
|---|-------------|---------------|---------------------|
| 1 | Location System: 4 neighbourhoods + adjacency | ✅ Achieved | No |
| 2 | Resource Tracking: Health/Sanity/Clues with bounds | ✅ Achieved | No |
| 3 | Action System: 2 actions/turn, 4 core actions | ✅ Achieved | No |
| 4 | Doom Counter: 0-12, increments on Tentacle | ✅ Achieved | No |
| 5 | Dice Resolution: 3-sided, configurable threshold | ✅ Achieved | No |
| 6 | 1-6 concurrent players with join-in-progress | ✅ Achieved | No |
| 7 | Real-time state sync < 500ms | ✅ Achieved | No |
| 8 | Interface-based design (net.Conn/Listener/Addr) | ✅ Achieved | No |
| 9 | Ebitengine desktop + WASM builds compile | ✅ Achieved | No |
| 10 | Session persistence (reconnect token) | ✅ Achieved | No |
| 11 | Performance monitoring: /dashboard, /metrics, /health | ⚠️ Partial | **Yes** — GAP-17 deadlock |
| 12 | Sub-100ms health check response time | ⚠️ Partial | **Yes** — GAP-17 deadlock |
| 13 | Mutex-protection convention on all game state reads | ⚠️ Partial | **Yes** — GAP-18 data race |
| 14 | Win threshold: 4 clues × player count | ⚠️ Partial | **Yes** — GAP-19 |
| 15 | `go test ./...` exercises all production paths | ⚠️ Partial | **Yes** — GAP-21 |
| 16 | AH3e rules compliance (RULES.md §Full Action Set) | ⚠️ Partial | **Yes** — Phase 6 steps |
| 17 | Stable operation with 6 concurrent players (15 min) | ⚠️ Partial | **Yes** — GAP-17 + benchmark |
| 18 | Mobile build verified on device | ❌ Missing | No (out of scope this plan) |
| 19 | Production sprite assets | ❌ Missing | No (out of scope this plan) |

---

## Metrics Summary

> Source: `go-stats-generator analyze . --skip-tests --format json --sections functions,duplication,documentation,packages,patterns`

| Metric | Value | Threshold | Assessment |
|--------|-------|-----------|------------|
| Total LoC | 2,347 | — | Moderate; single-package server focus |
| Functions + methods | 187 | — | Server-heavy (cmd/server dominates) |
| **Complexity hotspots (cx > 9)** | **0** | < 5 = Small | **Clean** — no function above threshold |
| Functions cx > 5 (watch list) | 21 | — | All between 6–9; acceptable |
| Highest complexity | `cleanupDisconnectedPlayers` (cx=9.0) | < 10 | Within acceptable range |
| **Duplication ratio** | **0%** | < 3% = Small | **Excellent** |
| **Doc coverage** | **96.6%** | > 90% | **Excellent** (types gap: 93.75%) |
| TODO comments | 1 | — | `game_mechanics.go:346` (Phase 6) |
| Circular dependencies | 0 | 0 | Clean package graph |
| `go test -race ./...` failures | 0 | 0 | Baseline healthy |
| `go vet ./...` warnings | 0 | 0 | Baseline clean |

**Complexity on goal-critical paths:**

| Function | File | cx | Relevance to Gaps |
|----------|------|----|-------------------|
| `handleHealthCheck` | `cmd/server/health.go` | 6 | GAP-17 (nested RLock) |
| `getSystemAlerts` | `cmd/server/health.go` | 6 | GAP-17 + GAP-18 (unlocked read) |
| `runMythosPhase` | `cmd/server/game_mechanics.go` | 7 | ROADMAP P6 (Mythos events) |
| `checkGameEndConditions` | `cmd/server/game_mechanics.go` | 8 | GAP-19 (win threshold) |
| `cleanupDisconnectedPlayers` | `cmd/server/connection.go` | 9 | Concurrency (watch list) |

---

## Implementation Steps

Steps are ordered: **correctness bugs first** (prerequisite for stable operation),
then **test coverage** (validates fixes), then **AH3e rules extensions** (highest
remaining stated-goal impact), then **code hygiene**.

---

### Step 1: Fix Deadlock in `/health` and `/metrics` Under Concurrent Player Actions (GAP-17)

- **Deliverable**: Refactor `cmd/server/health.go` (`handleHealthCheck`, `getGameStatistics`,
  `measureHealthCheckResponseTime`, `getSystemAlerts`) and `cmd/server/metrics.go`
  (`handleMetrics`, `collectPerformanceMetrics`) so each HTTP handler acquires
  `gs.mutex.RLock()` exactly once at the top, snapshots all needed `gs.gameState` fields
  into local variables, releases the lock, then calls all helper functions with the
  snapshot values (no helper may call `gs.mutex` again).
- **Files**: `cmd/server/health.go`, `cmd/server/metrics.go`
- **Dependencies**: None — standalone correctness fix.
- **Goal Impact**: Restores "sub-100ms health check response time" and "stable operation
  with 6 concurrent players" to unconditionally achievable (Goals 11, 12, 17).
- **Acceptance**:
  1. `handleHealthCheck` and `handleMetrics` each contain exactly **one**
     `gs.mutex.RLock()` / `gs.mutex.RUnlock()` pair.
  2. No called helper function (`getGameStatistics`, `measureHealthCheckResponseTime`,
     `getSystemAlerts`, `collectPerformanceMetrics`) acquires `gs.mutex` itself.
  3. All 178 existing tests still pass.
  4. New integration test `TestHandleHealthCheck_ConcurrentActions` — spin up 4 goroutines
     calling `processAction` in a tight loop; concurrently issue 50 HTTP GET `/health`
     requests; assert all return 200 within 1 second and no deadlock (test timeout: 10s).
- **Validation**:
  ```bash
  go test -race -timeout 30s ./cmd/server/... -run TestHandleHealthCheck
  go test -race -timeout 30s ./cmd/server/... -run TestHandleMetrics
  go test -race -timeout 60s ./cmd/server/... -run TestHandleHealthCheck_ConcurrentActions
  ```

---

### Step 2: Fix Data Race on `gs.gameState.Doom` in `getSystemAlerts` (GAP-18)

- **Deliverable**: In `cmd/server/health.go`, within `getSystemAlerts`, capture
  `gs.gameState.Doom` inside the existing `gs.mutex.RLock()` block and replace all
  subsequent bare `gs.gameState.Doom` references (currently at lines 239 and 245)
  with the captured `doom` local variable. This step is a prerequisite for Step 1
  because the snapshot pattern adopted there subsumes this fix naturally.
- **Files**: `cmd/server/health.go`
- **Dependencies**: Step 1 (the snapshot refactor makes this fix automatic; document
  separately so it can be reviewed independently).
- **Goal Impact**: Enforces the project's stated "centralized game state with mutex
  protection" convention (Goal 13). Eliminates a silent data race under `go test -race`
  when an integration test combines concurrent writes with health checks.
- **Acceptance**:
  1. `getSystemAlerts` contains **zero** bare reads of `gs.gameState.*` fields after
     any `gs.mutex.RUnlock()` call.
  2. `go test -race ./cmd/server/...` reports no data-race warnings after adding a
     test that writes `gs.gameState.Doom` concurrently while calling `getSystemAlerts`.
- **Validation**:
  ```bash
  go vet ./cmd/server/...
  go test -race -count=3 ./cmd/server/... -run TestGetSystemAlerts
  ```

---

### Step 3: Rescale Win Threshold When a Player Joins Mid-Game (GAP-19)

- **Deliverable**: In `cmd/server/connection.go`, `registerPlayer()`, add a
  `gs.rescaleActDeck(len(gs.gameState.Players))` call inside the late-join branch
  (`gs.gameState.GameStarted && gs.gameState.GamePhase == "playing"`) so the Act deck
  thresholds always equal `4 × player_count`:

  ```go
  } else if gs.gameState.GameStarted && gs.gameState.GamePhase == "playing" {
      log.Printf("Player %s joined game in progress (turn order position %d)",
          playerID, len(gs.gameState.TurnOrder))
      if len(gs.gameState.ActDeck) >= 3 {
          gs.rescaleActDeck(len(gs.gameState.Players))
      }
  }
  ```

- **Files**: `cmd/server/connection.go`
- **Dependencies**: None — single-line addition.
- **Goal Impact**: Closes the documented gap where a 2-player game where player 2 joins
  after game start requires only 4 clues (1P threshold) instead of 8 (Goal 14:
  "4 clues per investigator collectively").
- **Acceptance**:
  1. Existing test `TestRulesActAgendaProgression_PlayerCountScaling` continues to pass
     for all 6 player counts at game-start.
  2. New test `TestRescaleActDeck_LateJoin` — connect player 1 (starts game, threshold=4),
     connect player 2 (late join), assert `ActDeck[2].ClueThreshold == 8`.
  3. New test `TestRescaleActDeck_LateJoin_ThreePlayers` — assert threshold rescales to 12
     when player 3 joins.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestRulesActAgendaProgression
  go test -race ./cmd/server/... -run TestRescaleActDeck_LateJoin
  ```

---

### Step 4: Expose Ebitengine Client Tests in Standard `go test ./...` (GAP-21)

- **Deliverable**: Extract the display-independent test logic from
  `client/ebiten/app/game_test.go` and `client/ebiten/render/atlas_test.go` into new
  `_logic_test.go` companion files **without** the `//go:build requires_display` tag:
  - `client/ebiten/app/game_logic_test.go` — covers struct construction, nil-safety in
    `drawPlayerPanel`, `drawLocations`, `drawInputHints` (pass nil/zero-value args; assert
    no panic before the first Ebitengine draw call).
  - `client/ebiten/render/atlas_logic_test.go` — covers `AtlasSpec` initialisation, bounds
    validation, and any path that does not call `ebiten.NewImage`.

  Add a note to `README.md §Quick Setup` (under the `go test ./...` step):
  ```
  # Display-requiring Ebitengine tests (needs X11 or virtual display):
  go test -tags=requires_display -race ./client/ebiten/...
  ```

- **Files**: `client/ebiten/app/game_logic_test.go` (new),
  `client/ebiten/render/atlas_logic_test.go` (new), `README.md`
- **Dependencies**: None.
- **Goal Impact**: Ensures rendering-path regressions (nil player dereferences, atlas
  panics) are caught in every standard CI run and every contributor's local `go test ./...`
  (Goal 15).
- **Acceptance**:
  1. `go test ./client/ebiten/app/... ./client/ebiten/render/...` (no tags) prints at least
     one `--- PASS` line per package (not `[no test files]`).
  2. Existing `requires_display`-tagged tests remain intact and still pass when run with the
     tag.
- **Validation**:
  ```bash
  go test -race ./client/ebiten/app/... ./client/ebiten/render/...
  go test -race -tags=requires_display ./client/ebiten/...   # CI with Xvfb
  ```

---

### Step 5: Remove `ActionComponent` Dead Code (GAP-20)

- **Deliverable**: In `cmd/server/game_mechanics.go`:
  1. Remove `case ActionComponent:` and its `gs.performComponent(...)` call from
     `dispatchAction` (currently unreachable because `isValidActionType` excludes
     `ActionComponent`).
  2. Delete `performComponent` (lines 341–348). Replace with an inline comment at the
     `ActionComponent` constant in `game_constants.go`:
     ```go
     ActionComponent ActionType = "component" // Phase 6 — per-investigator ability; not yet active
     ```

- **Files**: `cmd/server/game_mechanics.go`, `cmd/server/game_constants.go`
- **Dependencies**: None.
- **Goal Impact**: Reduces maintenance noise; future developers reading `dispatchAction`
  will no longer see a seemingly-handled case that is actually blocked upstream.
- **Acceptance**:
  1. `go build ./cmd/server` succeeds.
  2. `go test -race ./cmd/server/... -run TestProcessAction_ComponentActionRejected` passes.
  3. `grep -n 'performComponent' cmd/server/game_mechanics.go` returns no function body
     (only the TODO comment if kept).
- **Validation**:
  ```bash
  go build ./cmd/server && go test -race ./cmd/server/... -run TestProcessAction
  go-stats-generator analyze ./cmd/server --skip-tests --format json --sections functions \
    | jq '[.functions[] | select(.name=="performComponent")] | length == 0'
  ```

---

### Step 6: Add Automated Performance Benchmark (ROADMAP Priority 5) ✅ COMPLETE

- **Deliverable**: Create `cmd/server/benchmark_test.go` with:
  1. `BenchmarkBroadcastLatency` — measures time from action receipt to the last connected
     client receiving the `gameState` message. Uses in-process WebSocket pairs via
     `net.Pipe()` (no external network). Target: p99 < 200ms under 6 concurrent players.
  2. `TestStabilityWith6Players` — connects 6 players via `net.Pipe()` WebSocket pairs,
     each player submits 1 action per second for 60 seconds (scaled-down proxy for 15 min
     stability), asserts zero panics, zero connection drops, doom counter advances correctly.

- **Files**: `cmd/server/benchmark_test.go` (new)
- **Dependencies**: Steps 1–2 (concurrent load test only meaningful after deadlock is fixed).
- **Goal Impact**: Creates a machine-verifiable acceptance criterion for the "stable operation
  with 6 concurrent players for 15 minutes" and "sub-500ms broadcast latency" promises
  (Goal 17), replacing the current "architecture supports sub-500ms" aspiration.
- **Acceptance**:
  1. `go test -bench=BenchmarkBroadcastLatency -benchtime=10s ./cmd/server/...` completes
     with reported p99 < 200ms.
  2. `TestStabilityWith6Players` passes under `go test -race -timeout 120s`.
- **Validation**:
  ```bash
  go test -race -timeout 120s ./cmd/server/... -run TestStabilityWith6Players
  go test -bench=BenchmarkBroadcastLatency -benchtime=10s -benchmem ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json --sections functions \
    | jq '[.functions[] | select(.name | startswith("Benchmark"))] | length >= 1'
  ```

---

### Step 7: Implement `performComponent` Action (ROADMAP Priority 1) ✅ COMPLETE

- **Deliverable**: Implement the per-investigator component ability system, completing
  AH3e `ActionComponent`:
  1. Add `InvestigatorType` field to `Player` struct in `cmd/server/game_types.go`
     (string enum: `"researcher"`, `"detective"`, `"occultist"`, `"soldier"`,
     `"mystic"`, `"survivor"`).
  2. Define `InvestigatorAbility` struct and a `DefaultInvestigatorAbilities` map
     (keyed by `InvestigatorType`) in `cmd/server/game_constants.go`.
  3. Implement `performComponent(gs *GameState, playerID string) (string, error)` in
     `cmd/server/game_mechanics.go`: look up player's `InvestigatorType`, retrieve the
     ability, execute its effect (e.g., Researcher: gain 1 Clue without dice; Detective:
     draw an encounter card; Occultist: reduce Doom by 1 at cost of 2 Sanity).
  4. Re-enable `ActionComponent` in `isValidActionType` (`cmd/server/game_server.go`).
  5. Restore `case ActionComponent:` in `dispatchAction`.

- **Files**: `cmd/server/game_types.go`, `cmd/server/game_constants.go`,
  `cmd/server/game_mechanics.go`, `cmd/server/game_server.go`
- **Dependencies**: Step 5 (dead code removed first to avoid confusion during re-implementation).
- **Goal Impact**: Closes RULES.md §Component Action gap; AH3e compliance reaches 8/8
  action types (Goal 16).
- **Acceptance**:
  1. `go test -race ./cmd/server/... -run TestProcessAction_Component` passes with at
     least 6 sub-tests (one per investigator type), covering valid ability execution and
     resource cost enforcement.
  2. A client can send `{"type":"playerAction","action":"component"}` and receive a
     `gameUpdate` response (not an "invalid action type" error).
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestProcessAction_Component
  go-stats-generator analyze ./cmd/server --skip-tests --format json --sections functions \
    | jq '[.functions[] | select(.name=="performComponent" and .complexity.cyclomatic > 0)] | length == 1'
  ```

---

### Step 8: Implement Attack/Evade Actions and Enemy Spawning (ROADMAP Priority 2) ✅ COMPLETE

- **Deliverable**: Add combat mechanics required by RULES.md §Attack/Evade:
  1. Define `Enemy` struct in `cmd/server/game_types.go`:
     ```go
     type Enemy struct {
         ID       string
         Name     string
         Health   int
         Damage   int
         Horror   int
         Location Location
         Engaged  []string // player IDs currently engaged
     }
     ```
  2. Add `Enemies map[string]*Enemy` to `GameState`; include in JSON serialisation.
  3. Add `ActionAttack ActionType = "attack"` and `ActionEvade ActionType = "evade"` to
     `game_constants.go`; register both in `isValidActionType`.
  4. Implement `performAttack(playerID string)` in `game_mechanics.go`: roll combat dice
     pool (size = player's effective combat skill); each Success deals 1 damage to the
     engaged enemy; enemy defeated when health reaches 0 (removed from `Enemies` map,
     player gains 1 Clue). Each Tentacle increments Doom.
  5. Implement `performEvade(playerID string)` in `game_mechanics.go`: roll agility dice
     pool; on ≥1 Success, disengage player from enemy (remove from `Engaged` list).
  6. Add enemy spawn logic to `runMythosPhase`: spawn 1 enemy per 3 accumulated Doom at a
     random location (capped at 4 total enemies).
  7. Add `case ActionAttack:` and `case ActionEvade:` to `dispatchAction`.

- **Files**: `cmd/server/game_types.go`, `cmd/server/game_constants.go`,
  `cmd/server/game_mechanics.go`, `cmd/server/game_server.go`
- **Dependencies**: Step 7 (shares `GameState` struct extensions; order reduces merge conflicts).
- **Goal Impact**: Completes AH3e combat loop; together with Step 7 brings rules compliance
  to "full action set implemented" (Goal 16).
- **Acceptance**:
  1. `go test -race ./cmd/server/... -run TestProcessAction_Attack` passes.
  2. `go test -race ./cmd/server/... -run TestProcessAction_Evade` passes.
  3. `go test -race ./cmd/server/... -run TestMythosPhase_EnemySpawn` passes.
  4. Enemy state included in `gameState` broadcast; connected clients see enemy positions.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestProcessAction_Attack
  go test -race ./cmd/server/... -run TestProcessAction_Evade
  go test -race ./cmd/server/... -run TestMythosPhase_EnemySpawn
  go build ./cmd/server
  ```

---

### Step 9: Implement Mythos Phase Events (ROADMAP Priority 6) ✅ COMPLETE

- **Deliverable**: Extend `runMythosPhase` in `cmd/server/game_mechanics.go` to draw and
  resolve event cards (currently it only advances Doom):
  1. Define `MythosEvent` struct in `game_types.go`:
     ```go
     type MythosEvent struct {
         ID          string
         Description string
         Effect      func(gs *GameState) error
     }
     ```
  2. Create `DefaultMythosEventDeck` (6 cards minimum) in `game_constants.go`: examples —
     "Fog of Madness" (all players lose 1 Sanity), "Clue Drought" (Clue gain blocked next
     turn), "Doom Spread" (Doom +1 per open gate).
  3. In `runMythosPhase`: draw 1 event card, execute its effect, append the event summary
     to the `gameUpdate` broadcast.
  4. Include `ActiveEvents []string` in `GameState` JSON so clients can display current
     event cards.

- **Files**: `cmd/server/game_types.go`, `cmd/server/game_constants.go`,
  `cmd/server/game_mechanics.go`
- **Dependencies**: Step 8 (gate-dependent event "Doom Spread" requires Step 9 to come after
  Step 8's gate scaffolding, or be stubbed initially).
- **Goal Impact**: Closes RULES.md §Mythos Phase Sequence gap; increases replay variation
  and advances AH3e compliance beyond the action set.
- **Acceptance**:
  1. `go test -race ./cmd/server/... -run TestMythosPhase_EventPlacement` passes.
  2. `gameState` broadcasts include `"activeEvents"` field after a mythos phase resolves.
  3. `runMythosPhase` cyclomatic complexity remains ≤ 10 (currently cx=7; event dispatch
     adds ~2).
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestMythosPhase
  go-stats-generator analyze ./cmd/server --skip-tests --format json --sections functions \
    | jq '[.functions[] | select(.name=="runMythosPhase")] | .[0].complexity.cyclomatic <= 10'
  ```

---

### Step 10: Implement Gate/Anomaly Mechanics (ROADMAP Priority 7)

- **Deliverable**: Implement Anomaly/Gate mechanics as specified in RULES.md §Anomaly/Gate
  Mechanics:
  1. Define `Gate` struct in `game_types.go`; add `OpenGates []Gate` to `GameState`.
  2. In `runMythosPhase`: open a Gate when a Location accumulates ≥ 2 Doom tokens; track
     per-location doom in a new `LocationDoom map[Location]int` field on `GameState`.
  3. Add `ActionCloseGate ActionType = "closegate"` in `game_constants.go`; implement
     `performCloseGate` in `game_mechanics.go` (requires 2 Clues spent; removes gate,
     reduces global Doom by 1).
  4. Include `OpenGates` and `LocationDoom` in `gameState` JSON broadcasts.

- **Files**: `cmd/server/game_types.go`, `cmd/server/game_constants.go`,
  `cmd/server/game_mechanics.go`, `cmd/server/game_server.go`
- **Dependencies**: Step 9 (Mythos Phase event "Doom Spread" depends on `OpenGates`).
- **Goal Impact**: Completes the Anomaly/Gate subsystem required by RULES.md; gives the
  doom-management loop a player-agency counterplay beyond Cast Ward.
- **Acceptance**:
  1. `go test -race ./cmd/server/... -run TestGateMechanics_OpenAndClose` passes (covers:
     gate opens after 2 doom accumulate at a location; `performCloseGate` spends 2 clues;
     doom decrements; gate removed from `OpenGates`).
  2. `gameState` JSON includes `"openGates"` and `"locationDoom"` fields.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestGateMechanics
  go build ./cmd/server
  ```

---

## Non-Goals for This Plan

The following are explicitly deferred — they appear in the project's own roadmap but are
not addressed here:

| Deferred Item | Reason |
|---------------|--------|
| Production sprite assets (ROADMAP P4) | Requires art direction decisions outside engineering scope |
| Mobile build device verification (ROADMAP P3) | Requires physical devices / CI infrastructure |
| Ebitengine display-level rendering tests | Requires virtual display in CI; orthogonal to correctness bugs |
| Game content (encounter text, scenario scripts) | Out of scope per RULES.md §Non-Goals |
| TLS / HTTPS | Infrastructure-layer concern per project conventions |

---

## Dependency Graph

```
Step 1 (deadlock fix)
Step 2 (data race fix)        ← Steps 1 & 2 may land together; Step 2 is subsumed by Step 1's snapshot pattern
Step 3 (win threshold)
Step 4 (test visibility)
Step 5 (dead code cleanup)
  └── Step 7 (performComponent)    ← depends on Step 5
        └── Step 8 (Attack/Evade)  ← depends on Step 7 (shared GameState structs)
              └── Step 9 (Mythos events)  ← depends on Step 8 (gate scaffolding)
                    └── Step 10 (Gates)   ← depends on Step 9 (doom-spread event)
Step 6 (benchmark)            ← depends on Steps 1 & 2 (concurrent load only meaningful after deadlock fixed)
```

---

## Quick Reference: Validation Commands

```bash
# After Steps 1-2 (concurrency fixes):
go test -race -timeout 60s ./cmd/server/... -run "TestHandleHealthCheck|TestHandleMetrics|TestGetSystemAlerts"

# After Step 3 (win threshold):
go test -race ./cmd/server/... -run "TestRulesActAgendaProgression|TestRescaleActDeck"

# After Step 4 (test visibility):
go test -race ./client/ebiten/app/... ./client/ebiten/render/...

# After Step 5 (dead code):
go build ./cmd/server && go test -race ./cmd/server/... -run TestProcessAction_Component

# After Step 6 (benchmark):
go test -race -timeout 120s ./cmd/server/... -run TestStabilityWith6Players
go test -bench=BenchmarkBroadcastLatency -benchtime=10s ./cmd/server/...

# After Steps 7-10 (AH3e completion):
go test -race ./cmd/server/... -run "TestProcessAction_Component|TestProcessAction_Attack|TestProcessAction_Evade|TestMythosPhase|TestGateMechanics"

# Full suite (all steps complete):
go test -race ./... && go build ./cmd/server && go build ./cmd/desktop
go-stats-generator analyze . --skip-tests --format json --sections functions \
  | jq '.functions | map(select(.complexity.cyclomatic > 9)) | length'
# Expected: 0
```
