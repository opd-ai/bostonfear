# Implementation Plan: Phase 6 — AH3e Rules Compliance

> **Generated**: 2026-03-15  
> **Basis**: `go-stats-generator` metrics (18 files, 2,215 LOC) cross-referenced with
> `ROADMAP.md`, `GAPS.md`, `RULES.md`, and `CLIENT_SPEC.md`.

---

## Project Context

- **What it does**: Multiplayer Arkham Horror WebSocket game with a Go server and a
  Go/Ebitengine client (desktop, WASM, mobile alpha) supporting 1–6 concurrent
  investigators with real-time state synchronisation.
- **Current goal**: Fix the 7 documented implementation gaps that block alpha
  functionality, then advance to Phase 6 — full AH3e core-rulebook compliance (9 of
  10 rule systems unimplemented or partial).
- **Estimated scope**: **Large** — 9 Phase 6 rule systems to implement; 7 blocking
  bugs to fix; 13 functions above the complexity threshold; 0 test coverage on the
  rules engine.

---

## Goal-Achievement Status

| Stated Goal (ROADMAP.md) | Current Status | This Plan Addresses |
|---|---|---|
| Phase 1 — Ebitengine Client Foundation | ⚠️ Alpha; `gameState` fields invisible (GAP-01) | Yes — Step 1 |
| Phase 2 — Desktop Build | ⚠️ Alpha; broken by GAP-01 | Yes — Step 1 |
| Phase 3 — WASM Build | ⚠️ Alpha; broken by GAP-01 | Yes — Step 1 |
| Phase 4 — Mobile Build | ⚠️ Scaffolding only; not verified on device | No |
| Phase 5 — Enhanced Graphics & Presentation | ❌ Not started | No |
| Phase 6 — AH3e Rules Compliance | ❌ Not started (9/10 systems missing/partial) | Yes — Steps 4–13 |
| `net.Conn` interface contract honoured | ⚠️ Violated (GAP-02) | Yes — Step 2 |
| Mythos Phase randomness | ❌ Deterministic (GAP-03) | Yes — Step 3 |
| Reconnect-token round-trip (Ebitengine) | ❌ Token discarded (GAP-04) | Yes — Step 1 (included) |
| Prometheus metrics accuracy | ⚠️ `totalGamesPlayed` can double-count (GAP-07) | Yes — Step 3 |
| `component` action clearly unsupported | ⚠️ Accepted but silently errors (GAP-06) | Yes — Step 3 |
| README accuracy | ⚠️ Misstates session persistence (GAP-05) | Yes — Step 3 |

---

## Metrics Summary

| Metric | Value | Threshold | Assessment |
|---|---|---|---|
| Total LOC | 2,215 | — | Moderate codebase |
| Functions above overall complexity 9.0 | **13** | 5–15 = Medium | **Medium** |
| Duplication ratio | **0.42%** | <3% = Small | **Small** |
| Doc coverage | **96.3%** | gap <10% = Small | **Small** |
| Test coverage (rules engine) | **0%** | — | Critical gap |
| Anti-patterns flagged | 35 items | — | Medium cleanup needed |

**Complexity hotspots on goal-critical paths:**

| Function | File | Overall | CC | Relevance |
|---|---|---|---|---|
| `validateResources` | `game_server.go` | 17.4 | 13 | Blocks resource-type expansion (Phase 6) |
| `cleanupDisconnectedPlayers` | `game_server.go` | 14.7 | 9 | Reconnect / session-persistence path |
| `broadcastConnectionQuality` | `game_server.go` | 11.4 | 8 | Broadcast pipeline |
| `broadcastHandler` | `game_server.go` | 11.1 | 7 | Core broadcast loop |
| `advanceTurn` | `game_server.go` | 10.6 | 7 | Turn-order enforcement |
| `checkGameEndConditions` | `game_server.go` | 10.1 | 7 | Win/lose logic; double-count bug (GAP-07) |
| `drawPlayerPanel` | `game.go` | 10.1 | 7 | Ebitengine rendering |

**Duplication:** 1 clone pair — `game_server.go` lines 130–147 / 148–165 (18 lines,
renamed clone). Extract to a shared helper as part of Step 5.

**Anti-patterns (critical/high severity):**

| Type | Location | Severity |
|---|---|---|
| Resource leak (no defer close) | `client/ebiten/net.go:83` | Critical |
| Resource leak (no defer close) | `client/ebiten/game.go:62` | Critical |
| Goroutine without done channel | `client/ebiten/net.go:73` | High |
| Goroutine without done channel | `cmd/server/game_server.go:116,118,120,1067` | High |
| Bare error return (no wrapping) | `connection_wrapper.go:33`, `game_server.go:235,807` | High |
| String concatenation in loop | `game_server.go:1230,1261` | High |

**Dependency notes:**

- `gorilla/websocket v1.5.3` — no current CVEs; project in community-maintenance mode.
  No urgent action required, but consider migration to `nhooyr.io/websocket` or the
  Gorilla community fork when next doing network-layer work.
- `ebiten/v2 v2.7.0` — v2.8 is available; upgrading requires Go 1.22+ (project
  currently targets Go 1.24.1 ✅). Main breaking changes: Kage shader signature
  update (`custom vec4` 4th argument) and `text/v2` API. Upgrade is straightforward
  but should be deferred to Phase 5 (Enhanced Graphics) when shaders are introduced.

---

## Implementation Steps

Steps 1–3 fix the 7 documented gaps (blocking alpha stability).  
Steps 4–13 implement Phase 6 AH3e compliance.  
Each step is independently deployable and testable.

---

### Step 1: Fix GAP-01 and GAP-04 — Ebitengine Client Protocol + Reconnect Token

**Why first**: The Ebitengine client renders a permanently empty board regardless of
server state (GAP-01). Every other Phase 1–3 acceptance criterion is blocked until
the client can display live game state. GAP-04 (reconnect token) is in the same file
(`client/ebiten/net.go`) and state struct (`client/ebiten/state.go`), so fixing both
together avoids a second round-trip to those files.

- **Deliverable**:
  1. `client/ebiten/net.go` — `decodeGameState()`: unmarshal `msg.Data`
     (`json.RawMessage`) into a temporary `GameState` struct rather than reading
     top-level fields. Remove the now-redundant top-level field declarations
     (lines 8–18 of the current `serverMessage` struct).
  2. `client/ebiten/state.go` — Add `ReconnectToken string` to `LocalState`.
     Add `Token string \`json:"token"\`` to `ConnectionStatusData`.
  3. `client/ebiten/net.go` — `applyConnectionStatus()`: store received token in
     `LocalState.ReconnectToken`.
  4. `client/ebiten/net.go` — `reconnectLoop()`: append `?token=<token>` to the
     dial URL when `LocalState.ReconnectToken` is non-empty.
  5. `client/ebiten/net_test.go` (new file):
     - `TestDecodeGameState_FromDataWrapper`: encodes a known `GameState` under
       `"data"` key; asserts all fields round-trip correctly.
     - `TestReconnectLoop_IncludesToken`: asserts dial URL contains `?token=` on
       second connection attempt.

- **Dependencies**: None (isolated to `client/ebiten/`).
- **Goal Impact**: Unblocks Phase 1, 2, and 3 acceptance criteria. Fixes GAP-01 and
  GAP-04.
- **Acceptance**:
  - `go build ./client/ebiten/...` passes.
  - `go test ./client/ebiten/...` passes (both new tests green).
  - Desktop client connected to a running server displays non-zero doom counter,
    player locations, and resource levels.
- **Validation**:
  ```bash
  go test ./client/ebiten/... -v -run 'TestDecodeGameState|TestReconnect'
  go-stats-generator analyze ./client/ebiten/... --format json --sections functions \
    | jq '[.functions[] | select(.complexity.overall > 9)] | length'
  # Target: same or lower than baseline (writeLoop=9.3, readLoop=8.8)
  ```

---

### Step 2: Fix GAP-02 — `ConnectionWrapper.Read()` Byte Count

**Why second**: `ConnectionWrapper` is the project's stated `net.Conn` abstraction
for testing and middleware. Its incorrect return value (`len(data)` instead of
`n := copy(b, data); return n`) means any future middleware or test mock using
`Read` will silently corrupt data. Fixing it now prevents compounding the interface
violation during Phase 6 work on the rules engine.

- **Deliverable**:
  1. `cmd/server/connection_wrapper.go` line 42: replace `return len(data), nil`
     with:
     ```go
     n := copy(b, data)
     return n, nil
     ```
  2. `cmd/server/connection_wrapper_test.go` (add test):
     - `TestConnectionWrapper_ReadTruncation`: calls `Read` with a buffer smaller
       than the data; asserts returned `n == len(b)` and only `b[:n]` is populated.
     - `TestConnectionWrapper_ReadExact`: calls `Read` with a buffer exactly equal
       to data length; asserts `n == len(data)`.

- **Dependencies**: None.
- **Goal Impact**: Restores `net.Conn` interface contract; prerequisite for any
  future middleware layer or test double using the `Read` path.
- **Acceptance**:
  - `go test -race ./cmd/server/... -run TestConnectionWrapper` passes.
  - `go vet ./cmd/server/...` reports no new issues.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestConnectionWrapper -v
  ```

---

### Step 3: Fix GAP-03, GAP-05, GAP-06, GAP-07 — Server Correctness Batch

**Why together**: These four gaps are all single-function or documentation fixes in
`cmd/server/` with no interdependencies. Batching them into one PR keeps the fix
history clean.

- **Deliverable**:

  **GAP-03** — `game_server.go` `drawMythosToken()` (line 672):  
  Replace `tokens[gs.gameState.Doom%len(tokens)]` with
  `tokens[mathrand.Intn(len(tokens))]`.  
  Add `TestDrawMythosToken_IsRandom` in `game_server_test.go`: calls the function
  ≥100 times and asserts all four token values appear at least once.

  **GAP-05** — `README.md` lines ~108–113:  
  Replace the "cannot reclaim their investigator" paragraph with accurate text:
  > "The JS legacy client reclaims its player slot automatically using a
  > server-issued reconnect token. The Ebitengine client does not yet implement
  > token reclaim (see GAPS.md GAP-04 — fixed in this release for Ebitengine); all
  > clients now support full session persistence."

  **GAP-06** — `cmd/server/game_server.go` `isValidActionType()`:  
  Remove `ActionComponent` from the accepted-action slice so the server returns a
  clear `"invalid action type: component"` error instead of a server-side panic.
  Add `TestProcessAction_ComponentActionRejected` asserting the correct error string.

  **GAP-07** — `cmd/server/game_server.go` `checkGameEndConditions()` (line 747):  
  Add an early-return guard:
  ```go
  if gs.gameState.GamePhase == "ended" {
      return
  }
  ```
  Add `TestCheckGameEndConditions_NoDoubleCount`: calls the function twice on an
  `"ended"` game; asserts `gs.totalGamesPlayed == 1`.

- **Dependencies**: Step 2 (establishes test pattern in `cmd/server/`).
- **Goal Impact**: Fixes Mythos Phase randomness; corrects Prometheus metrics;
  removes misleading `component` action; fixes README accuracy.
- **Acceptance**:
  - `go test -race ./cmd/server/...` passes with all four new tests green.
  - `grep -n "cannot reclaim" README.md` returns no matches.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestDrawMythos|TestProcessAction_Component|TestCheckGameEnd' -v
  go-stats-generator analyze ./cmd/server/... --format json --sections functions \
    | jq '[.functions[] | select(.name == "checkGameEndConditions")] | .[0].complexity'
  ```

---

### Step 4: Refactor `validateResources` — Prerequisite for Phase 6 Resource Types

**Why before Phase 6**: `validateResources` has the highest complexity in the entire
codebase (CC=13, overall=17.4). Phase 6 adds three new resource types (Money,
Remnants, Focus tokens) and will require modifying this function. Decomposing it
first makes the Phase 6 changes safe and independently reviewable.

- **Deliverable**:
  1. Extract a `validateSingleResource(name string, value, min, max int) error`
     helper in `cmd/server/game_server.go` (or `cmd/server/utils.go`).
  2. Rewrite `validateResources` to call `validateSingleResource` for each
     resource field, eliminating the repeated `if … < … || … > …` branches.
  3. Eliminate the 18-line clone (lines 130–147 / 148–165 flagged by
     `go-stats-generator`) by extracting a `buildNeighbourhoods() []Neighbourhood`
     function called from both sites.
  4. Update `cmd/server/utils_test.go` (or new `validate_test.go`) with a
     table-driven test covering all boundary cases for `validateSingleResource`.

- **Dependencies**: Step 3 (clean test baseline in `cmd/server/`).
- **Goal Impact**: Reduces `validateResources` complexity below threshold (target
  CC ≤ 6); eliminates the only duplication clone; makes resource-type expansion safe.
- **Acceptance**:
  - `go test -race ./cmd/server/...` passes.
  - `go-stats-generator` reports `validateResources` overall complexity < 9.0.
  - Duplication ratio drops to 0.0%.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -v
  go-stats-generator analyze ./cmd/server/... --skip-tests --format json \
    --sections functions,duplication \
    | jq '{complexity: [.functions[] | select(.name=="validateResources")] | .[0].complexity,
           duplication: .duplication.duplication_ratio}'
  # Target: complexity.overall < 9.0, duplication_ratio == 0.0
  ```

---

### Step 5: Phase 6 — Extended Resource Types (Money, Remnants, Focus)

**Why here**: All subsequent Phase 6 work depends on the correct resource model.
`validateResources` is already decomposed (Step 4), so adding new fields is
straightforward.

- **Deliverable**:
  1. `cmd/server/types.go` — Add to `Player` struct:
     ```go
     Money   int `json:"money"`
     Remnants int `json:"remnants"`
     Focus   int `json:"focus"`
     ```
  2. `cmd/server/constants.go` — Add `MaxMoney`, `MaxRemnants`, `MaxFocus`
     constants (AH3e values: Money 0–99, Remnants 0–5, Focus 0–3).
  3. `cmd/server/game_server.go` — Extend `validateResources` to validate the
     three new fields using `validateSingleResource` (Step 4 helper).
  4. `cmd/server/game_server.go` — Initialise all three fields to 0 in player
     creation / reset paths.
  5. Broadcast the updated `Player` struct in `gameState` messages (struct
     serialisation handles this automatically).
  6. `client/ebiten/state.go` — Mirror new fields in `PlayerState`.
  7. `cmd/server/types_test.go` — Table-driven tests asserting new resource bounds
     are enforced.

- **Dependencies**: Step 4 (refactored `validateResources`).
- **Goal Impact**: Satisfies Phase 6 resource-management requirement; unblocks
  Focus-token dice-pool modifiers (Step 7).
- **Acceptance**:
  - `go test -race ./cmd/server/...` passes.
  - A `gameState` message sent to a connected client includes `money`, `remnants`,
    and `focus` fields with correct initial values.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestPlayerResource -v
  go-stats-generator analyze ./cmd/server/... --format json --sections documentation \
    | jq '.documentation.coverage'
  # Target: coverage.overall >= 96% (maintain existing standard)
  ```

---

### Step 6: Phase 6 — Investigator Defeat and Recovery

**Why before actions**: The defeat condition (health OR sanity reaches 0) must be
enforced before implementing actions that can reduce those stats to 0. Without it,
actions can produce an illegal game state.

- **Deliverable**:
  1. `cmd/server/types.go` — Add `IsDefeated bool` and `LostInTimeAndSpace bool`
     to `Player`.
  2. `cmd/server/game_server.go` — Add `checkInvestigatorDefeat(player *Player)`
     called after every resource change:
     - If `Health == 0 || Sanity == 0`: set `IsDefeated = true`,
       `LostInTimeAndSpace = true`, reset Health/Sanity to starting values,
       move player to starting location.
  3. `cmd/server/game_server.go` — `validateActionRequest()`: reject actions from
     defeated investigators (return `"investigator is defeated"` error).
  4. `cmd/server/game_server.go` — `advanceTurn()`: skip defeated players in the
     turn rotation.
  5. `cmd/server/game_server_test.go` — Tests:
     - `TestInvestigatorDefeat_HealthZero`
     - `TestInvestigatorDefeat_SanityZero`
     - `TestDefeatedInvestigatorSkippedInTurn`
     - `TestDefeatedInvestigatorCannotAct`

- **Dependencies**: Step 5 (extended resource model).
- **Goal Impact**: Implements AH3e investigator-defeat rule; reduces `advanceTurn`
  complexity (current CC=7) by pulling defeat-skip logic into the new helper.
- **Acceptance**:
  - All four tests pass under `go test -race`.
  - A player whose health drops to 0 is marked defeated, skipped in turn order, and
    cannot submit actions until recovered.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestInvestigator' -v
  go-stats-generator analyze ./cmd/server/... --format json --sections functions \
    | jq '[.functions[] | select(.name=="advanceTurn")] | .[0].complexity'
  # Target: advanceTurn complexity.overall <= 8.0
  ```

---

### Step 7: Phase 6 — Full Action Set (Focus, Research, Trade, Component)

**Why now**: Actions depend on: correct resource types (Step 5) and defeat
semantics (Step 6). The four missing AH3e actions can now be safely implemented.

- **Deliverable**:
  1. `cmd/server/constants.go` — Ensure `ActionFocus`, `ActionResearch`,
     `ActionTrade`, `ActionComponent` are defined.
  2. `cmd/server/game_server.go` — Implement:
     - `performFocus(gs, player)`: grants 1 Focus token (cap at MaxFocus).
     - `performResearch(gs, player)`: costs Money (1 per attempt per AH3e),
       awards Clues on success (reuse dice resolution with difficulty 1).
     - `performTrade(gs, player, targetPlayerID)`: transfers 1 resource of chosen
       type between players in the same location; validates co-location.
     - `performComponent(gs, player)`: stub returns
       `"component actions require card data (Phase 6 final polish)"` — retains
       the TODO comment from `game_server.go:545`.
  3. `cmd/server/game_server.go` — `isValidActionType()`: add all four to the
     accepted slice (remove current stub rejection of `component`).
  4. `cmd/server/game_server.go` — `dispatchAction()`: route new action types.
  5. `cmd/server/game_server_test.go` — Tests:
     - `TestFocusAction_GainsFocusToken`
     - `TestFocusAction_CapsAtMax`
     - `TestResearchAction_CostsMoneyAndRollsDice`
     - `TestTradeAction_TransfersResource`
     - `TestTradeAction_RequiresSameLocation`
     - `TestComponentAction_ReturnsExpectedStub`

- **Dependencies**: Step 5, Step 6.
- **Goal Impact**: Brings action-set implementation from 4/8 to 8/8 AH3e actions.
- **Acceptance**:
  - All six tests pass.
  - `go test -race ./cmd/server/...` passes.
  - `isValidActionType("focus")` returns `true`.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestFocusAction|TestResearchAction|TestTradeAction|TestComponentAction' -v
  go-stats-generator analyze ./cmd/server/... --format json --sections functions \
    | jq '[.functions[] | select(.complexity.overall > 9)] | length'
  # Target: no new functions above threshold introduced
  ```

---

### Step 8: Phase 6 — Dice Pool Modifiers (Focus Spend, Skill-Based Pool Size)

**Why here**: Dice pool modifiers are called by `performResearch` (Step 7) and
the `Cast Ward` action. Implementing them now completes the dice-resolution system.

- **Deliverable**:
  1. `cmd/server/types.go` — Extend `PlayerActionMessage` with
     `FocusSpend int \`json:"focusSpend"\`` (client-declared focus tokens to spend).
  2. `cmd/server/game_server.go` — Refactor `rollDice(numDice int)` into
     `rollDicePool(baseDice, focusSpend int, player *Player) DiceResult`:
     - Deduct `focusSpend` from `player.Focus` (validate ≥ 0).
     - Roll `baseDice + focusSpend` dice.
     - Allow one reroll of non-success results per focus token spent (AH3e rule).
  3. Update all `rollDice` call sites to use `rollDicePool`.
  4. `cmd/server/game_server_test.go` — Tests:
     - `TestDicePool_FocusSpendRerolls`
     - `TestDicePool_InvalidFocusSpend` (spend more than available)
     - `TestDicePool_ZeroFocusNoChange`

- **Dependencies**: Step 7 (Focus tokens in Player struct, all actions wired).
- **Goal Impact**: Closes dice-resolution gap (focus spend, skill-based pool);
  resolves the "Dice Resolution: partial" row in the GAPS engine status table.
- **Acceptance**:
  - All three tests pass.
  - A `playerAction` with `"focusSpend": 1` correctly deducts 1 focus token and
    triggers a reroll.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestDicePool' -v
  ```

---

### Step 9: Phase 6 — Mythos Phase (Full Implementation)

**Why here**: The Mythos Phase is the game's primary doom-escalation and event
mechanism. The existing `runMythosPhase` stub (CC=6, line 9.3 overall) already
defines the entry point. GAP-03 (deterministic token draw) is already fixed
in Step 3.

- **Deliverable**:
  1. `cmd/server/types.go` — Add:
     - `MythosTokenType string` with constants (`TokenDoom`, `TokenBlessing`,
       `TokenCurse`, `TokenBlank`).
     - `MythosCup []MythosTokenType` on `GameState`.
     - `EventCard` struct with `NeighbourhoodTarget string` and `Effect string`.
     - `ActiveEvents []EventCard` on `GameState`.
  2. `cmd/server/game_server.go` — Implement `runMythosPhase(gs)`:
     - Draw 2 `EventCard`s from a fixed starter deck of 8 cards; place in
       `ActiveEvents`.
     - If an event targets a neighbourhood already containing a doom token,
       increment global doom by 1 (event spread rule).
     - Draw 1 `MythosToken` from `MythosCup` using `mathrand.Intn` (already fixed
       in Step 3); apply token effect (`doom` → increment doom, `blessing` →
       optional Health restore, `curse` → Sanity loss, `blank` → no effect).
     - Initialise `MythosCup` with the AH3e starter composition (4 doom, 2 blank,
       1 blessing, 1 curse) at game start.
  3. `cmd/server/game_server.go` — Wire `runMythosPhase` into the turn-advance
     logic: after all players have acted, execute the Mythos Phase before
     resetting action counts.
  4. `cmd/server/game_server_test.go` — Tests:
     - `TestMythosPhase_DrawsTwoEvents`
     - `TestMythosPhase_EventSpreadIncrementsDooom`
     - `TestMythosPhase_TokenDrawAffectsState`
     - `TestMythosPhase_StarterCupComposition`

- **Dependencies**: Step 8 (dice system complete), Step 6 (defeat semantics).
- **Goal Impact**: Closes the "Mythos Phase: not implemented" gap; introduces the
  full Investigator Phase → Mythos Phase → cycle that AH3e requires.
- **Acceptance**:
  - All four tests pass.
  - After all players act, the server broadcasts a `gameUpdate` event with
    `"event": "mythosPhase"` and the active events visible in the next `gameState`.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestMythosPhase' -v
  go-stats-generator analyze ./cmd/server/... --format json --sections functions \
    | jq '[.functions[] | select(.name=="runMythosPhase")] | .[0].complexity'
  # Target: complexity.overall <= 12.0 (acceptable for phase orchestrator)
  ```

---

### Step 10: Phase 6 — Encounter Resolution

**Why here**: Encounter cards are drawn after Mythos Phase placement. The Mythos
Phase (Step 9) provides the neighbourhood event system that encounter resolution
hooks into.

- **Deliverable**:
  1. `cmd/server/types.go` — Add `EncounterCard` struct:
     `{ Description string; SkillTest string; SuccessEffect string; FailureEffect string }`.
     Add `EncounterDecks map[string][]EncounterCard` (keyed by neighbourhood name).
  2. `cmd/server/game_server.go` — Add `resolveEncounter(gs, player, neighbourhood)`:
     - Draw top card from the neighbourhood's `EncounterCard` deck.
     - Run skill test using `rollDicePool` (difficulty from card).
     - Apply `SuccessEffect` or `FailureEffect` to player resources.
  3. `cmd/server/game_server.go` — Add `ActionEncounter` constant and
     `performEncounter` action handler calling `resolveEncounter`.
  4. Seed each of the four neighbourhoods with 3 encounter cards (minimal starter
     deck; not game content, just test fixtures for the engine).
  5. `cmd/server/game_server_test.go` — Tests:
     - `TestEncounterResolution_SuccessAppliesEffect`
     - `TestEncounterResolution_FailureAppliesEffect`
     - `TestEncounterResolution_DrawsFromCorrectNeighbourhood`

- **Dependencies**: Step 9 (Mythos Phase, active events model).
- **Goal Impact**: Closes "Encounter Resolution: not implemented" gap.
- **Acceptance**:
  - All three tests pass under `go test -race`.
  - An investigator in "Downtown" who performs an encounter receives a resource
    effect derived from the Downtown encounter deck.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestEncounterResolution' -v
  ```

---

### Step 11: Phase 6 — Act / Agenda Deck Progression

**Why here**: Act/Agenda progression is the win/lose clock; it replaces the current
simple clue-threshold win condition and requires the Mythos Phase doom increments
(Step 9) as its input.

- **Deliverable**:
  1. `cmd/server/types.go` — Add:
     - `ActCard { RequiredClues int; Title string; NextActIndex int }`.
     - `AgendaCard { DoomThreshold int; Title string; IsFinal bool }`.
     - `ActDeck []ActCard`, `AgendaDeck []AgendaCard`, `ActIndex int`,
       `AgendaIndex int` on `GameState`.
  2. Populate both decks with a minimal 2-act / 2-agenda sequence at game start
     (engine fixtures, not story content).
  3. `cmd/server/game_server.go` — `checkGameEndConditions()`:
     - Replace the clue-threshold win check with: if collective clues ≥
       `ActDeck[ActIndex].RequiredClues`, advance `ActIndex`; if `ActIndex` reaches
       end of act deck, declare win.
     - Replace the simple doom-12 lose check with: if global doom ≥
       `AgendaDeck[AgendaIndex].DoomThreshold`, advance `AgendaIndex`; if the
       agenda card is `IsFinal`, declare loss.
  4. `cmd/server/game_server_test.go` — Tests:
     - `TestActAdvancement_OnClueThreshold`
     - `TestAgendaAdvancement_OnDoomThreshold`
     - `TestGameWin_ActDeckExhausted`
     - `TestGameLose_FinalAgendaReached`

- **Dependencies**: Step 9 (doom-increment path), Step 10 (encounter clue gain).
- **Goal Impact**: Closes "Act/Agenda Deck Progression: not implemented" gap and
  "Scenario System: not implemented" gap (partial — provides the engine skeleton
  that a full scenario system builds on).
- **Acceptance**:
  - All four tests pass.
  - `checkGameEndConditions` complexity decreases (GAP-07 guard already added in
    Step 3; act/agenda logic is extracted into helpers).
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestAct|TestAgenda|TestGameWin|TestGameLose' -v
  go-stats-generator analyze ./cmd/server/... --format json --sections functions \
    | jq '[.functions[] | select(.name=="checkGameEndConditions")] | .[0].complexity'
  # Target: complexity.overall <= 9.0 after helper extraction
  ```

---

### Step 12: Phase 6 — Anomaly / Gate Mechanics

**Why here**: Anomalies are spawned by specific Mythos Phase events (Step 9) and
sealed by the `Cast Ward` action. Both prerequisites exist.

- **Deliverable**:
  1. `cmd/server/types.go` — Add `Anomaly { NeighbourhoodID string; DoomTokens int }`
     and `Anomalies []Anomaly` on `GameState`.
  2. `cmd/server/game_server.go` — `runMythosPhase`: if a drawn event card has type
     `"anomaly"`, call `spawnAnomaly(gs, neighbourhood)` which appends to
     `gs.gameState.Anomalies` and adds 1 doom to the neighbourhood.
  3. `cmd/server/game_server.go` — `performWard`: existing success path (3 dice,
     3 successes) now also removes the `Anomaly` from the neighbourhood if one
     exists there, reducing doom by 2 (AH3e seal mechanics).
  4. `cmd/server/game_server_test.go` — Tests:
     - `TestAnomalySpawn_OnMythosEvent`
     - `TestWardAction_SealsAnomaly`
     - `TestWardAction_NoDoomReductionWithoutAnomaly`

- **Dependencies**: Step 9 (Mythos Phase spawns anomalies), Step 7 (Ward action wired).
- **Goal Impact**: Closes "Anomaly / gate mechanics" gap.
- **Acceptance**:
  - All three tests pass.
  - A Ward success in a neighbourhood with an anomaly removes it and reduces doom
    by 2.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestAnomaly|TestWardAction' -v
  ```

---

### Step 13: Phase 6 — Modular Difficulty Settings

**Why last in Phase 6**: Difficulty modifies initial doom placement and Mythos Cup
composition — both of which are fully implemented by Step 12. Implementing
difficulty last avoids coupling it to incomplete systems.

- **Deliverable**:
  1. `cmd/server/types.go` — Add `Difficulty string` (`"easy"`, `"standard"`,
     `"hard"`) to `GameState`.
  2. `cmd/server/constants.go` — Add difficulty configuration table:
     ```go
     var DifficultyConfig = map[string]struct{
         InitialDoom    int
         MythosCupExtra []MythosTokenType
     }{
         "easy":     {0, []MythosTokenType{}},
         "standard": {1, []MythosTokenType{TokenDoom}},
         "hard":     {3, []MythosTokenType{TokenDoom, TokenDoom, TokenCurse}},
     }
     ```
  3. `cmd/server/game_server.go` — `NewGameServer` / game-start path: accept
     `difficulty` parameter (from the first `playerAction` with `action: "startGame"`)
     and apply the config table to initial doom and `MythosCup` composition.
  4. `cmd/server/game_server_test.go` — Tests:
     - `TestDifficulty_EasyStartsDoomAtZero`
     - `TestDifficulty_HardAddsExtraDoomTokens`
     - `TestDifficulty_InvalidDifficultyReturnsError`

- **Dependencies**: Step 12 (Mythos Cup and doom systems complete).
- **Goal Impact**: Closes "Modular Difficulty Settings: not implemented" gap;
  completes all 10 AH3e core rule systems.
- **Acceptance**:
  - All three tests pass.
  - Starting a game with `"difficulty": "hard"` results in initial doom of 3 and
    2 extra doom tokens in the Mythos Cup.
  - `go test ./cmd/server/...` passes with zero failures.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run 'TestDifficulty' -v
  # Full suite to confirm no regressions across all steps:
  go test -race ./... 2>&1 | tail -20
  go-stats-generator analyze . --skip-tests --format json \
    --sections functions,duplication,documentation \
    | jq '{
        high_complexity: [.functions[] | select(.complexity.overall > 9)] | length,
        duplication_ratio: .duplication.duplication_ratio,
        doc_coverage: .documentation.coverage.overall
      }'
  # Targets: high_complexity <= 8, duplication_ratio == 0.0, doc_coverage >= 96%
  ```

---

## Summary Table

| Step | Scope | Files Changed | Blocks Step(s) | Closes Gap(s) |
|------|-------|---------------|----------------|---------------|
| 1 — Ebitengine protocol + reconnect token | Small | `client/ebiten/net.go`, `state.go`, `net_test.go` | — | GAP-01, GAP-04 |
| 2 — `ConnectionWrapper.Read` byte count | Small | `connection_wrapper.go`, `connection_wrapper_test.go` | 3 | GAP-02 |
| 3 — Server correctness batch | Small | `game_server.go`, `game_server_test.go`, `README.md` | 4 | GAP-03, GAP-05, GAP-06, GAP-07 |
| 4 — Refactor `validateResources` | Small | `game_server.go`, `utils.go` | 5–13 | Complexity + duplication |
| 5 — Extended resource types | Medium | `types.go`, `constants.go`, `game_server.go`, `state.go` | 6–13 | Phase 6 resources |
| 6 — Investigator defeat + recovery | Medium | `types.go`, `game_server.go`, tests | 7–13 | Phase 6 defeat |
| 7 — Full action set | Medium | `constants.go`, `game_server.go`, tests | 8–13 | Phase 6 actions |
| 8 — Dice pool modifiers | Medium | `types.go`, `game_server.go`, tests | 9–13 | Phase 6 dice |
| 9 — Mythos Phase | Medium | `types.go`, `game_server.go`, tests | 10–13 | Phase 6 Mythos |
| 10 — Encounter resolution | Medium | `types.go`, `game_server.go`, tests | 11–13 | Phase 6 encounters |
| 11 — Act / Agenda deck | Medium | `types.go`, `game_server.go`, tests | 12–13 | Phase 6 act/agenda |
| 12 — Anomaly / gate mechanics | Medium | `types.go`, `game_server.go`, tests | 13 | Phase 6 anomalies |
| 13 — Modular difficulty | Small | `types.go`, `constants.go`, `game_server.go`, tests | — | Phase 6 difficulty |

**Phase 6 completion check** (run after Step 13):
```bash
go test -race ./... && \
go-stats-generator analyze . --skip-tests --format json \
  --sections functions,duplication,documentation | \
  jq '{high_complexity: [.functions[] | select(.complexity.overall > 9)] | length,
       duplication_ratio: .duplication.duplication_ratio,
       doc_coverage: .documentation.coverage.overall}'
```
Expected: `{ "high_complexity": ≤8, "duplication_ratio": 0, "doc_coverage": ≥96 }`

---

## Out of Scope for This Plan

Per `ROADMAP.md` non-goals and the project's explicit phase structure:

- Phase 5 (Enhanced Graphics — Kage shaders, sprite atlas) — deferred until Phase 6
  is complete; upgrading Ebitengine to v2.8 should coincide with this phase.
- Phase 4 (Mobile — device verification) — requires physical device testing outside
  this plan's scope.
- Game content: no new cards, scenarios, investigators, encounter narratives, or
  flavor text.
- `gorilla/websocket` migration to an actively maintained alternative — no CVEs
  currently; revisit when Phase 5 network-layer work begins.
