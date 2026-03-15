# Implementation Plan: Phase 0 Baseline Fixes → Phase 6 AH3e Rules Compliance

> **Generated**: 2026-03-15  
> **Metrics source**: `go-stats-generator analyze . --skip-tests --format json`  
> **Baseline test status**: `go test -race ./cmd/server/...` → PASS (1.246 s)

---

## Project Context

- **What it does**: Cooperative multiplayer Arkham Horror game for 1–6 concurrent
  players with a Go WebSocket server, a legacy HTML/JS client, and a substantially
  implemented Go/Ebitengine client targeting desktop, WASM, and mobile.
- **Current goal**: Achieve full Arkham Horror 3rd Edition core-rulebook compliance
  (ROADMAP Phase 6), preceded by Phase 0 baseline correctness fixes that are
  prerequisites for that work.
- **Estimated Scope**: **Large** — 12 gaps identified across correctness, AH3e
  compliance, architecture, and code quality, spanning 8 complex functions, 12 gap
  items, and 3 new subsystems (Mythos Phase, Encounters, Act/Agenda Deck).

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|---|---|---|
| 5 core mechanics (Location, Resources, Actions, Doom, Dice) | ✅ Achieved | No — already done |
| `net.Conn`/`net.Listener`/`net.Addr` interface-based design | ⚠️ Bug: `LocalAddr()` returns remote addr | Yes — Step 1 |
| Goroutine + channel concurrency model | ✅ Achieved | No |
| JS client exponential backoff (unlimited retries) | ⚠️ Hard cap at 10 attempts | Yes — Step 2 |
| Investigator defeat on Health/Sanity = 0 | ❌ Not implemented (clamped to 1) | Yes — Step 3 |
| Application-layer interfaces for testability | ❌ 0 exported interfaces | Yes — Step 4 |
| AH3e full resource set (Health, Sanity, Money, Remnants, Focus, Clues) | ⚠️ Partial (3 of 6) | Yes — Step 5 |
| AH3e full action set (8 of 8 action types) | ⚠️ Partial (4 of 8) | Yes — Step 6 |
| Mythos Phase (event draw, placement, spread, cup token) | ❌ Not implemented | Yes — Step 7 |
| Act/Agenda Deck progression | ❌ Not implemented | Yes — Step 8 |
| Encounter Resolution | ❌ Not implemented | Yes — Step 9 |
| Scenario System (modular setup, difficulty) | ❌ Not implemented | Yes — Step 10 |
| Session persistence / reconnection token | ❌ Not implemented (zombie slots) | Yes — Step 11 |
| Observability function length (≤50 lines) | ⚠️ 4 functions exceed threshold | Yes — Step 12 |
| README Build Targets table accuracy | ⚠️ Mislabels live clients as "Planned" | Yes — Step 13 |

---

## Metrics Summary

| Metric | Value | Scope per Thresholds |
|---|---|---|
| Complexity hotspots on goal-critical paths (overall > 9.0) | 8 functions | **Medium** (5–15) |
| Duplication ratio | **0%** | Small (<3%) |
| Doc coverage gap | **4%** (type-level only; exported functions 100%) | Small (<10%) |
| Long functions (> 50 lines) | 4 in `cmd/server/game_server.go` | Small (<5) |
| Exported interfaces | **0** | Architecture debt |
| Server-side `main` package size | 9 files, 98 functions, 33 structs | Coupling concern |

### Complexity Hotspots on Goal-Critical Paths

| Function | File | Overall | Cyclomatic | Goal-Critical Path |
|---|---|---|---|---|
| `broadcastConnectionQuality` | `cmd/server/game_server.go` | 11.4 | 8 | Broadcast / testability (Step 4) |
| `broadcastHandler` | `cmd/server/game_server.go` | 11.1 | 7 | Broadcast / testability (Step 4) |
| `drawPlayerPanel` | `client/ebiten/game.go` | 10.1 | 7 | Client render (Phase 5, out of scope) |
| `assessConnectionQuality` | `cmd/server/game_server.go` | 9.8 | 6 | Monitoring (Step 12) |
| `validateResources` | `cmd/server/game_server.go` | 9.6 | 7 | AH3e resources (Step 5) |
| `writeLoop` | `client/ebiten/net.go` | 9.3 | 6 | Client concurrency (Phase 5, out of scope) |
| `runMessageLoop` | `cmd/server/game_server.go` | 9.3 | 6 | Action processing (Step 3/6) |
| `actionHandler` | `cmd/server/game_server.go` | 9.3 | 6 | Action processing (Step 6) |

### Long Functions (> 50 lines)

| Function | Lines | File |
|---|---|---|
| `handleMetrics` | 93 | `cmd/server/game_server.go` |
| `collectPerformanceMetrics` | 64 | `cmd/server/game_server.go` |
| `handleHealthCheck` | 52 | `cmd/server/game_server.go` |
| `getSystemAlerts` | 52 | `cmd/server/game_server.go` |

---

## Implementation Steps

Steps are ordered: prerequisites first, then by descending impact on stated goals.
**Phase 0 (Steps 1–4)** fixes correctness and testability blockers.
**Phase 6 (Steps 5–10)** achieves AH3e rules compliance.
**Housekeeping (Steps 11–13)** closes remaining gaps.

---

### Step 1: Fix `ConnectionWrapper.LocalAddr()` Net.Conn Contract Violation

- **Deliverable**:
  - `cmd/server/connection_wrapper.go` — add `localAddr net.Addr` field alongside
    `remoteAddr net.Addr`; update `NewConnectionWrapper` signature to accept both;
    return correct field from each method.
  - `cmd/server/game_server.go` line 654 — pass `wsConn.NetConn().LocalAddr()` as
    `localAddr` argument when constructing `ConnectionWrapper`.
  - New test `TestConnectionWrapper_LocalRemoteAddrDistinct` in
    `cmd/server/connection_wrapper_test.go` asserting
    `LocalAddr().String() != RemoteAddr().String()`.
- **Dependencies**: None — isolated to `connection_wrapper.go` and its call site.
- **Goal Impact**: Closes GAP-01 (CRITICAL). Repairs the `net.Conn` interface
  contract violation that is a latent hazard for any downstream code relying on
  `LocalAddr()`.
- **Acceptance**: `go test -race -run TestConnectionWrapper ./cmd/server/...` passes;
  `LocalAddr()` returns the server's listening address, not the client's remote
  address.
- **Validation**:
  ```bash
  go test -race -run TestConnectionWrapper ./cmd/server/...
  go vet ./cmd/server/...
  ```

---

### Step 2: Remove JS Reconnection Hard Cap

- **Deliverable**:
  - `client/game.js` line 11 — change `this.maxReconnectAttempts = 10` to
    `this.maxReconnectAttempts = Infinity`.
  - Add a visible UI notification in `client/index.html` when reconnection is
    temporarily paused (e.g., after 30 s of continuous failure).
  - Update `README.md` §Connection Behaviour to note that retries continue
    indefinitely (remove the outdated "10 attempts" implication).
- **Dependencies**: None.
- **Goal Impact**: Closes GAP-04 (HIGH). Aligns the legacy JS client with the
  documented unlimited exponential-backoff behaviour.
- **Acceptance**: Manually stop the server for 6+ minutes, restart it; the browser
  client reconnects without requiring a page refresh.
- **Validation**:
  ```bash
  grep "maxReconnectAttempts" client/game.js   # must show Infinity
  ```

---

### Step 3: Implement Investigator Defeat (Health/Sanity → 0)

- **Deliverable**:
  - `cmd/server/game_server.go` `validateResources()` — change Health/Sanity lower
    bound from `1` to `0`.
  - New `checkInvestigatorDefeat(playerID string)` helper called from
    `processAction()` after `validateResources()`; transitions player to a
    `"defeated"` state.
  - `cmd/server/game_types.go` — add `Defeated bool` field to the `Player` struct.
  - `advanceTurn()` — skip defeated players.
  - `broadcastState()` — include `defeated` status in `gameState` JSON for client
    display.
  - New test `TestInvestigatorDefeat` in `cmd/server/game_mechanics_test.go`.
- **Dependencies**: None — Step 1 must be complete before running the full test
  suite, but this change is structurally independent.
- **Goal Impact**: Closes GAP-10 (MEDIUM). Restores the risk dimension of resource
  management essential for balanced AH3e play. Also prerequisite for Step 5 (defeat
  must interact correctly with the extended resource set).
- **Acceptance**: `go test -race -run TestInvestigatorDefeat ./cmd/server/...` passes;
  a player whose Health reaches 0 is marked `Defeated: true` and skipped in turn
  rotation.
- **Validation**:
  ```bash
  go test -race -run TestInvestigatorDefeat ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '[.functions[] | select(.complexity.cyclomatic > 10)] | length'
  # validateResources complexity should decrease after removing the Health≥1 clamp
  ```

---

### Step 4: Extract Application-Level Interfaces (Broadcaster, StateValidator)

- **Deliverable**:
  - `cmd/server/game_types.go` (or new `cmd/server/interfaces.go`) — define two
    exported interfaces:
    ```go
    // Broadcaster sends a JSON payload to all connected clients.
    type Broadcaster interface {
        Broadcast(payload []byte) error
    }

    // StateValidator checks the game state for invariant violations.
    type StateValidator interface {
        ValidateGameState(gs *GameState) []ValidationError
    }
    ```
  - `cmd/server/game_server.go` — update `GameServer` to hold a `Broadcaster` field
    and call it through the interface in `broadcastState()`.
  - `cmd/server/main.go` — inject the concrete `GameServer` broadcast implementation.
  - New test `TestBroadcasterInterface` with a no-op mock broadcaster, verifying
    that action processing calls `Broadcast` exactly once per state change.
- **Dependencies**: Steps 1–3 should be complete so the test suite is clean.
- **Goal Impact**: Closes the "0 exported interfaces" gap (HIGH, AUDIT.md). Enables
  unit tests for game mechanics without spinning up real goroutines, channels, or
  WebSocket connections — directly supporting the Phase 6 rules-compliance test
  suite (Steps 5–10).
- **Acceptance**: `go-stats-generator analyze ./cmd/server --skip-tests --format json
  2>/dev/null | jq '.overview.total_interfaces'` returns ≥ 2.
- **Validation**:
  ```bash
  go test -race ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '.overview.total_interfaces'
  # Expected: ≥ 2
  ```

---

### Step 5: Extend Resource System (Money, Remnants, Focus Tokens)

- **Deliverable**:
  - `cmd/server/game_types.go` `Resources` struct — add `Money int`, `Remnants int`,
    `Focus int` fields.
  - `cmd/server/game_constants.go` — add bound constants:
    `MaxMoney = 99`, `MaxRemnants = 5`, `MaxFocus = 3`.
  - `cmd/server/game_server.go` `validateResources()` — clamp the three new fields.
  - `performGather()` — award $1 Money instead of generic resources on a
    successful gather roll (per AH3e §Gather Resources).
  - `performInvestigate()` / `performWard()` — deduct Focus tokens when player
    chooses to spend them to reroll dice; wire dice-pool bonus logic.
  - `cmd/server/rules_test.go` — unskip the three resource tests:
    `TestRulesResourceMoney`, `TestRulesResourceRemnants`, `TestRulesResourceFocus`.
- **Dependencies**: Step 3 (defeat handling must be in place before resource bounds
  include 0 as a valid lower bound for Health/Sanity).
- **Goal Impact**: Closes GAP-06 (HIGH). Provides the full AH3e resource vocabulary
  required by Step 6 (actions) and Step 7 (Mythos Phase).
- **Acceptance**: `go test -race -run TestRulesResource ./cmd/server/...` — all three
  previously-skipped resource tests pass with no skips.
- **Validation**:
  ```bash
  go test -race -run TestRulesResource ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '[.functions[] | select(.name == "validateResources")] | .[0].complexity.cyclomatic'
  # Cyclomatic complexity should be ≤ 10 after adding 3 bounded clamps
  ```

---

### Step 6: Implement Missing AH3e Actions (Focus, Research, Trade, Component)

- **Deliverable**:
  - `cmd/server/game_constants.go` — add constants:
    `ActionFocus = "focus"`, `ActionResearch = "research"`,
    `ActionTrade = "trade"`, `ActionComponent = "component"`.
  - `cmd/server/game_server.go` — implement four new action handlers:
    - `performFocus(playerID)` — award 1 Focus token; no dice roll.
    - `performResearch(playerID, target)` — extended investigate requiring 2
      successes; reward 2 Clues on success.
    - `performTrade(fromID, toID string, delta Resources)` — transfer resources
      between co-located players; validate adjacency.
    - `performComponent(playerID)` — stub with a `TODO` comment and return a
      structured `ErrNotImplemented` error until investigator-specific abilities are
      added in Phase 6 final polish.
  - `actionHandler()` — add the four new cases to the action dispatch switch.
  - `cmd/server/rules_test.go` — unskip the four action tests:
    `TestRulesFullActionSet/focus_not_implemented` (and siblings).
- **Dependencies**: Steps 4–5 (interfaces for mock testing; Focus tokens must exist
  before `performFocus` can award them).
- **Goal Impact**: Closes GAP-05 (HIGH). Completes the 8-action AH3e action set.
  `actionHandler` cyclomatic complexity will increase from 6 to ~10; extract
  dispatch to a lookup table if it exceeds 12.
- **Acceptance**: `go test -race -run TestRulesFullActionSet ./cmd/server/...` — all
  previously-skipped sub-tests pass except `component` (which has a documented
  stub).
- **Validation**:
  ```bash
  go test -race -run TestRulesFullActionSet ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '[.functions[] | select(.name == "actionHandler")] | .[0].complexity.cyclomatic'
  # Must be ≤ 12; refactor to action dispatch table if exceeded
  ```

---

### Step 7: Implement Mythos Phase

- **Deliverable**:
  - `cmd/server/game_types.go` — add to `GameState`:
    - `GamePhase string` (`"investigator"` | `"mythos"`)
    - `MythosEventDeck []MythosEvent`
    - `LocationDoomTokens map[string]int`
    - `MythosToken string` (current cup token drawn)
  - New type `MythosEvent` with fields `LocationID string`, `Effect string`,
    `Spread bool`.
  - `cmd/server/game_server.go` — implement `runMythosPhase()`:
    1. Draw 2 events from `MythosEventDeck`.
    2. Place each event on its target neighborhood; if doom token already present,
       spread to adjacent neighborhood.
    3. Draw and resolve `MythosToken` (doom increment, resource penalty, etc.).
    4. Transition `GamePhase` back to `"investigator"`.
  - `advanceTurn()` — call `runMythosPhase()` after all players complete their turns.
  - `cmd/server/rules_test.go` — unskip
    `TestRulesMythosPhaseEventPlacement`.
- **Dependencies**: Steps 5–6 (Mythos Phase draws on the full resource and action
  vocabulary; game phase state requires the `GamePhase` field from Step 5's type
  work).
- **Goal Impact**: Closes GAP-07 (MEDIUM). Implements AH3e's primary doom-escalation
  driver; without it, doom grows only from dice tentacles and idle timeouts.
- **Acceptance**: `go test -race -run TestRulesMythosPhase ./cmd/server/...` passes
  with no skips. A 2-player game reaches the Mythos Phase automatically after both
  players exhaust their actions.
- **Validation**:
  ```bash
  go test -race -run TestRulesMythosPhase ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '[.functions[] | select(.name == "runMythosPhase")] | .[0].complexity.cyclomatic'
  # Must be ≤ 12
  ```

---

### Step 8: Implement Act/Agenda Deck Progression

- **Deliverable**:
  - `cmd/server/game_types.go` — add `ActDeck []ActCard`, `AgendaDeck []AgendaCard`
    to `GameState`. Define `ActCard{ClueThreshold int, Effect string}` and
    `AgendaCard{DoomThreshold int, Effect string}`.
  - `cmd/server/game_server.go` — implement:
    - `checkActAdvance()` — advance act if collective clues ≥ `ActCard.ClueThreshold`;
      replace flat `4 × playerCount` win check.
    - `checkAgendaAdvance()` — advance agenda when doom ≥ `AgendaDeck[0].DoomThreshold`;
      trigger loss on final agenda.
  - Wire `checkActAdvance()` into `processAction()` (after clue gain).
  - Wire `checkAgendaAdvance()` into `runMythosPhase()` (after doom increment).
  - Replace `checkGameEnd()` flat-clue logic with act/agenda deck checks.
  - `cmd/server/rules_test.go` — unskip `TestRulesActAgendaProgression`.
- **Dependencies**: Step 7 (Mythos Phase drives agenda advancement; `checkAgendaAdvance`
  is called from `runMythosPhase`).
- **Goal Impact**: Closes GAP-09 (MEDIUM). Provides the scenario narrative engine
  that gives win/lose conditions proper AH3e structure.
- **Acceptance**: `go test -race -run TestRulesActAgenda ./cmd/server/...` passes.
  Win/loss conditions are driven by act/agenda decks, not the flat clue threshold.
- **Validation**:
  ```bash
  go test -race -run TestRulesActAgenda ./cmd/server/...
  go test -race ./cmd/server/...  # full suite must still pass
  ```

---

### Step 9: Implement Encounter Resolution

- **Deliverable**:
  - `cmd/server/game_types.go` — define `EncounterCard{EffectType string,
    FlavorText string, Resolve func(*Player, *GameState) error}`.
  - Add `EncounterDecks map[string][]EncounterCard` to `GameState` (keyed by
    location name).
  - `cmd/server/game_constants.go` — populate minimal encounter decks for each of
    the 4 neighborhoods (2–3 cards each for MVP).
  - `cmd/server/game_server.go` — implement `performEncounter(playerID)`:
    draw from the player's current location deck and apply the effect.
  - `cmd/server/game_constants.go` — add `ActionEncounter = "encounter"` constant.
  - Wire `performEncounter` into `actionHandler`.
  - `cmd/server/rules_test.go` — add `TestRulesEncounterResolution` covering success
    and failure paths.
- **Dependencies**: Steps 5–6 (encounters modify the full resource set; encounter
  action must be in the action vocabulary).
- **Goal Impact**: Closes GAP-08 (MEDIUM). Completes the investigator-phase
  gameplay loop; players can now encounter strange events beyond the fixed 4 actions.
- **Acceptance**: `go test -race -run TestRulesEncounter ./cmd/server/...` passes.
  A player at `University` can submit `ActionEncounter` and receive a drawn card's
  effect.
- **Validation**:
  ```bash
  go test -race -run TestRulesEncounter ./cmd/server/...
  ```

---

### Step 10: Implement Scenario System (Modular Setup + Difficulty)

- **Deliverable**:
  - `cmd/server/game_types.go` — define `Scenario{Name string,
    StartingDoom int, WinConditionFn func(*GameState) bool,
    LoseFn func(*GameState) bool, SetupFn func(*GameState)}`.
  - `cmd/server/game_constants.go` — define `DefaultScenario` wrapping the current
    hardcoded behavior (4 neighborhoods, doom starts at 0, 4×players clues win).
  - `cmd/server/game_server.go` `NewGameServer()` — accept a `Scenario` parameter;
    call `scenario.SetupFn` during initialization.
  - Replace direct win/loss checks with calls to `scenario.WinConditionFn` /
    `scenario.LoseFn`.
  - `cmd/server/main.go` — pass `DefaultScenario` or parse a `-scenario` flag.
  - `cmd/server/rules_test.go` — add `TestRulesScenarioSystem`.
- **Dependencies**: Steps 7–9 (scenario setup must configure event decks, encounter
  decks, and act/agenda decks defined in those steps).
- **Goal Impact**: Closes GAP-11 (LOW-MEDIUM). Provides the extensibility layer
  for future scenario content and modular difficulty. Even with only `DefaultScenario`
  the code is now parameterized correctly.
- **Acceptance**: `go test -race -run TestRulesScenario ./cmd/server/...` passes.
  `NewGameServer(DefaultScenario)` produces identical behaviour to the current
  hardcoded initialization.
- **Validation**:
  ```bash
  go test -race ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '.overview | {functions, structs, interfaces}'
  ```

---

### Step 11: Implement Session Persistence / Reconnection Token

- **Deliverable**:
  - `cmd/server/game_server.go` `handleWebSocket()` — generate a UUID reconnection
    token and include it in the initial `connectionStatus` message:
    `{"type":"connectionStatus","playerId":"...","token":"<uuid>","status":"connected"}`.
  - Accept an optional `?token=<uuid>` query parameter on the `/ws` endpoint; if it
    matches a disconnected player, restore their slot instead of creating a new one.
  - `cmd/server/game_types.go` `Player` struct — add `ReconnectToken string` and
    `DisconnectedAt time.Time` fields.
  - Add a reaper goroutine (`cleanupDisconnectedPlayers`) that removes zombie player
    entries after a configurable TTL (default 5 minutes).
  - `client/ebiten/net.go` — store the received token and pass it as `?token=...` on
    reconnect attempts.
  - `client/game.js` — same token plumbing for the legacy JS client.
  - New test `TestSessionReconnection` verifying slot restoration.
- **Dependencies**: Steps 1–4 (interface abstractions make the reconnection flow
  testable without live WebSocket connections).
- **Goal Impact**: Closes GAP-12 (MEDIUM). Eliminates the zombie-player memory leak
  and gives players a real chance to reclaim their investigator after a disconnect.
- **Acceptance**: `go test -race -run TestSessionReconnection ./cmd/server/...`
  passes. A client that disconnects and reconnects within 5 minutes with its token
  receives the same `playerId` and unchanged game state.
- **Validation**:
  ```bash
  go test -race -run TestSessionReconnection ./cmd/server/...
  go test -race ./cmd/server/...
  ```

---

### Step 12: Decompose Oversized Observability Functions

- **Deliverable**:
  - `cmd/server/game_server.go` — refactor the 4 long functions by extracting each
    metric group into a helper:
    - `handleMetrics` (93 lines) → extract `buildConnectionMetrics() string`,
      `buildGameMetrics() string`, `buildMemoryMetrics() string`; `handleMetrics`
      calls each and joins the output.
    - `collectPerformanceMetrics` (64 lines) → extract `collectMemoryStats()`,
      `collectConnectionStats()`.
    - `handleHealthCheck` (52 lines) → extract `buildHealthPayload() HealthResponse`.
    - `getSystemAlerts` (52 lines) → no structural change required; add an early
      return path to reduce nesting depth.
  - Also snapshot `gs.mutex.RLock()` data before releasing the lock in
    `handleMetrics` and `handleHealthCheck` to eliminate the lock-held-during-
    serialization hazard (AUDIT.md §MEDIUM finding).
- **Dependencies**: None — purely internal refactor.
- **Goal Impact**: Reduces the 4 long-function violations; brings `handleMetrics`
  under the 50-line threshold. Fixes the lock contention issue that can delay
  sub-500 ms state sync under monitoring load.
- **Acceptance**:
  ```bash
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '[.functions[] | select(.lines.total > 50 and (.file | contains("game_server")))] | length'
  # Expected: 0
  ```
- **Validation**:
  ```bash
  go test -race ./cmd/server/...
  go-stats-generator analyze ./cmd/server --skip-tests --format json 2>/dev/null \
    | jq '[.functions[] | select(.lines.total > 50)] | length'
  # Expected: 0 server-side; client/ebiten functions out of scope for this step
  ```

---

### Step 13: Correct README Build Targets Table and ROADMAP Phase Status

- **Deliverable**:
  - `README.md` Build Targets table — update status column:
    - Desktop: `Active (alpha — placeholder sprites)`
    - WASM: `Active (alpha — placeholder sprites)`
    - Mobile: `Active (untested on device)`
  - `README.md` §Ebitengine Client Features — remove "(Planned — ROADMAP Phases 1–5)"
    qualifiers for features that are already implemented.
  - `ROADMAP.md` — mark Phases 1–3 as complete; mark Phase 5 as "In Progress
    (placeholder sprites)" and Phase 6 as "In Progress" upon initiating Step 5.
  - Cross-reference the 30-second read deadline vs. reconnection timeout confusion
    (AUDIT.md §MEDIUM) in `README.md` §Performance Standards.
- **Dependencies**: None — documentation only.
- **Goal Impact**: Closes GAP-02 (HIGH). Prevents contributors from duplicating
  already-completed work, and gives evaluators an accurate view of project maturity.
- **Acceptance**: `go build ./cmd/desktop/... && GOOS=js GOARCH=wasm go build
  ./cmd/web` both succeed (unchanged); README and ROADMAP consistently describe the
  current state.
- **Validation**:
  ```bash
  go build ./cmd/desktop/...
  GOOS=js GOARCH=wasm go build ./cmd/web
  grep "Active" README.md | grep -E "Desktop|WASM|Mobile"
  # Each platform should appear with "Active"
  ```

---

## Dependency Graph

```
Step 1 (LocalAddr fix)
    │
    ▼
Step 2 (JS reconnect)     ←── independent
    │
Step 3 (Investigator defeat)
    │
    ▼
Step 4 (Interface extraction)
    │
    ▼
Step 5 (Resource system)
    │
    ▼
Step 6 (Full action set)
    │
    ▼
Step 7 (Mythos Phase)
    │
    ▼
Step 8 (Act/Agenda Deck) ──── requires Step 7
Step 9 (Encounters)      ──── requires Steps 5–6
    │
    ▼
Step 10 (Scenario system) ── requires Steps 7–9
    │
    ▼
Step 11 (Session persistence) ── requires Steps 1–4
Step 12 (Observability refactor) ── independent
Step 13 (README/ROADMAP docs) ── independent
```

Steps 2, 12, and 13 have no dependencies and can be executed at any time.

---

## AH3e Rules Compliance Projection

After completing all steps, the compliance table from `RULES.md` reaches:

| Rule System | Pre-Plan | Post-Plan |
|---|---|---|
| Action System (8 types) | ⚠️ 4/8 | ✅ 8/8 |
| Dice Resolution (pool, focus spend) | ⚠️ Partial | ✅ Full |
| Mythos Phase | ❌ | ✅ |
| Resource Management (6 types) | ⚠️ 3/6 | ✅ 6/6 |
| Encounter Resolution | ❌ | ✅ |
| Act/Agenda Deck Progression | ❌ | ✅ |
| Investigator Defeat | ⚠️ Partial | ✅ |
| Scenario System | ❌ | ✅ (MVP) |
| Modular Difficulty | ❌ | ✅ (via Scenario) |
| 1–6 Player Support | ✅ | ✅ (maintained) |

---

## Dependency Update

### Ebiten v2.7.0 → v2.9.9

Online research confirms **v2.9.9 is the latest stable** release of
`github.com/hajimehoshi/ebiten/v2` (the project uses v2.7.0). The upgrade is
low-risk — semantic versioning guarantees backward compatibility within v2.

```bash
go get github.com/hajimehoshi/ebiten/v2@v2.9.9
go mod tidy
go build ./...
GOOS=js GOARCH=wasm go build ./cmd/web
```

This is safe to perform at any step. No API changes affecting the current codebase
were identified between v2.7.0 and v2.9.9.

**gorilla/websocket v1.5.3** is confirmed as the latest stable release; no upgrade
needed. The project's WebSocket concurrency model (single reader per connection,
serialized writes via channel) is correct and passes the race detector.

---

## Deferred / Out of Scope for This Plan

Per `ROADMAP.md` Non-Goals, the following are explicitly deferred:

- **Sprite artwork** (GAP-03): Real PNG assets for the Ebitengine client. Tracked in
  Phase 5. No code blocker; a contributor can add `client/ebiten/render/assets/` and
  swap `generateAtlas()` independently of all steps above.
- **`render.Renderer` type stutter** (AUDIT.md §LOW): Rename to `render.R` or
  `layers.Renderer`. Deferred to avoid churn while Phase 5 rendering is in flux.
- **`cmd/mobile/mobile.go` file name stutter** (AUDIT.md §LOW): Rename to
  `cmd/mobile/main.go`. Safe to do at any point; no functional impact.
- **Expansion content** (ROADMAP §Non-Goals): No new cards, investigators, scenarios,
  or narrative text is in scope.
- **Audio system**: Separate effort; not referenced in any gap or roadmap entry.

---

## Quick-Start Validation After All Steps

```bash
# 1. Build everything
go build ./...
GOOS=js GOARCH=wasm go build ./cmd/web

# 2. Run the full test suite
go test -race ./...

# 3. Check metrics against targets
go-stats-generator analyze . --skip-tests --format json 2>/dev/null | jq '{
  interfaces: .overview.total_interfaces,
  long_functions: ([.functions[] | select(.lines.total > 50)] | length),
  high_complexity: ([.functions[] | select(.complexity.overall > 9.0)] | length),
  duplication_ratio: .duplication.duplication_ratio,
  doc_coverage: .documentation.coverage.overall
}'
# Expected:
# interfaces ≥ 2
# long_functions == 0  (server-side)
# high_complexity ≤ 4  (client-side items deferred to Phase 5)
# duplication_ratio == 0
# doc_coverage ≥ 96
```
