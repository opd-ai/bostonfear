# Implementation Plan: Protocol Correctness, Reconnection Reliability & Architecture Hardening

## Project Context
- **What it does**: Multiplayer web implementation of Arkham Horror with Go WebSocket server and JavaScript client, supporting 2–4 players cooperating against a global doom counter.
- **Current goal**: Close all eight documented gaps in GAPS.md (functional and operational correctness), then advance ROADMAP Phase 1 production-excellence items.
- **Estimated Scope**: Large (25 functions above complexity threshold, 8 verified gaps, 0 interfaces despite interface-based design claim)

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|---|---|---|
| 3-step README setup works on clean environment | ❌ `cd server && go run main.go` fails — entry point is `cmd/server/` | Yes – Step 1 |
| Win condition disclosed to players | ❌ Formula `playerCount*4` exists only in source code | Yes – Step 2 |
| All 5 JSON protocol message types implemented | ❌ `gameUpdate` never emitted; 4 of 5 types active | Yes – Step 3 |
| `net.Conn` deadline interface contract honoured | ❌ `SetReadDeadline`, `SetWriteDeadline`, `SetDeadline` are no-ops | Yes – Step 4 |
| Reconnection restores player session state | ❌ Reconnect assigns new player ID; old slot orphaned forever | Yes – Step 5 |
| Error rate metric reflects real errors | ❌ `calculateErrorRate()` returns hardcoded `0.0` | Yes – Step 6 |
| Broadcast latency metric is non-zero under play | ❌ `AverageLatency` / `BroadcastLatency` hardcoded to `0` | Yes – Step 7 |
| `/dashboard` endpoint serves monitoring UI | ❌ `handleDashboard` resolves wrong relative path → HTTP 404 | Yes – Step 8 |
| Zero interfaces defined (architecture claim) | ❌ `total_interfaces: 0`; code calls itself interface-based | Yes – Step 9 |
| Goroutine leak risk eliminated | ⚠️ 3 high-severity goroutines without context/done channel | Yes – Step 10 |
| Complexity hotspots reduced | ⚠️ 25 functions above 9.0; `RecoverGameState` at 31.1 | Yes – Step 11 |
| Session persistence across reconnections (ROADMAP 1.1) | ❌ No reconnection token; no grace-period for disconnected players | Yes – Step 5 (in-memory token map) |
| Advanced Error Recovery (ROADMAP 1.2) | ⚠️ Validator exists; circuit breaker, rate limiting, structured logging absent | Partial – Step 6 |

---

## Metrics Summary (go-stats-generator baseline)

- **Complexity hotspots on goal-critical paths**: 25 functions above threshold 9.0 (Large)
  - Top 5: `RecoverGameState` 31.1, `processAction` 27.0, `handleConnection` 19.7, `ValidateGameState` 15.8, `collectConnectionAnalytics` 14.5, `broadcastGameState` 14.5
- **Duplication ratio**: 0% (clean — no action needed)
- **Doc coverage**: 93.6% overall (functions 100%, methods 100%, types 90.3%, packages 0%)
- **Interfaces defined**: 0 — single `main` package, all concrete types, no testable abstraction boundary
- **Package coupling**: Single monolithic `main` package with coupling score 0.5 (low external coupling but no internal separation)
- **Goroutine leak risk**: 3 high-severity anonymous goroutines (lines 90, 92, 560 of `game_server.go`) without context cancellation
- **TODO annotations**: 1 active — latency tracking at `game_server.go:919`
- **Anti-patterns**: 3 goroutine leak warnings, 3 bare error returns in `connection_wrapper.go`, 4 `append()` in loops without pre-allocation

---

## Implementation Steps

Steps are ordered: bug fixes that block all usage first → contract violations → functional gaps → monitoring accuracy → architecture quality.

---

### Step 1: Fix README Setup Instructions (Entry-Point Path)

- **Deliverable**: Update `README.md` Step 2 from `cd server && go run main.go` to `cd cmd/server && go run .`. Update the project structure table to show `cmd/server/` as the server entry point. Add a `Makefile` with a `run` target: `go run ./cmd/server/`.
- **Files**: `README.md`, new `Makefile`
- **Dependencies**: None — purely documentation/tooling
- **Goal Impact**: "Setup Verification: Can the project run following only the README instructions?" (currently fails)
- **Acceptance**: A developer on a clean checkout running only the README's 3 steps successfully starts the server and connects a browser client.
- **Validation**:
  ```bash
  cd cmd/server && go run . &
  sleep 2 && curl -sf http://localhost:8080/ | grep -q "Arkham" && echo PASS
  kill %1
  ```

---

### Step 2: Document Win Condition Threshold in README and Game State

- **Deliverable**: Add a concrete sentence to `README.md` Win/Lose Conditions: "**Win**: Collectively gather 4 clues per investigator (8 clues for 2 players, 12 for 3, 16 for 4)." Expose `requiredClues` in the `GameState` struct and JSON output so the client can render a win-progress bar.
- **Files**: `README.md`, `cmd/server/types.go` (add `RequiredClues int` to `GameState`), `cmd/server/game_server.go` (set `RequiredClues` in `checkGameEndConditions`)
- **Dependencies**: None
- **Goal Impact**: "Win: Achieve sufficient collective clues" is currently unplayable without source access; closes GAPS.md gap 4.
- **Acceptance**: `gameState` JSON includes `"requiredClues": 8` for a 2-player game; README states the formula explicitly.
- **Validation**:
  ```bash
  cd cmd/server && go build . && echo BUILD_PASS
  # Manual: connect 2 clients, observe requiredClues in gameState broadcast
  ```

---

### Step 3: Implement Missing `gameUpdate` Protocol Message

- **Deliverable**: After every action is processed in `processAction` (`cmd/server/game_server.go`), emit a lightweight `gameUpdate` event message **before** the full `gameState` broadcast. The message describes only the event that occurred. Add a `case 'gameUpdate':` handler in `client/game.js` to display a transient event notification. Update the README JSON protocol example with a real `gameUpdate` payload.
- **Files**: `cmd/server/game_server.go`, `cmd/server/types.go` (add `GameUpdateMessage` struct), `client/game.js`, `README.md`
- **Dependencies**: Step 1 (server must start correctly before testing)
- **Goal Impact**: Completes the 5-message JSON protocol contract; enables spectators, replays, and analytics integrations described in ROADMAP Phase 3.2. Closes GAPS.md gap 1.

**`gameUpdate` payload schema**:
```json
{
  "type": "gameUpdate",
  "playerId": "player_123",
  "event": "investigate",
  "result": "fail",
  "doomDelta": 1,
  "resourceDelta": {"clues": 0},
  "timestamp": "2026-03-15T00:00:00Z"
}
```

- **Acceptance**: After an Investigate action, connected clients receive `gameUpdate` (event details) followed by `gameState` (full snapshot). No client receives `gameState` without a preceding `gameUpdate` for player-triggered actions.
- **Validation**:
  ```bash
  cd cmd/server && go vet ./... && go build . && echo BUILD_PASS
  go-stats-generator analyze . --skip-tests --format json --sections documentation \
    | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['documentation']['coverage']['overall'])"
  # Expect coverage >= 93%
  ```

---

### Step 4: Fix `ConnectionWrapper` Deadline Methods (No-Op Violation)

- **Deliverable**: Implement `SetDeadline`, `SetReadDeadline`, and `SetWriteDeadline` in `cmd/server/connection_wrapper.go` by delegating to the underlying `*websocket.Conn`. Remove the duplicate direct `wsConn.SetReadDeadline` calls in `handleConnection` (`cmd/server/game_server.go` lines ~389, ~461) so all deadline management flows through the `net.Conn` interface.
- **Files**: `cmd/server/connection_wrapper.go`, `cmd/server/game_server.go`
- **Dependencies**: Step 1 (server entry point must be correct)
- **Goal Impact**: Restores the `net.Conn` interface contract; ensures the 30-second reconnection timeout works through the abstraction layer. Closes GAPS.md gap 7.

**Implementation**:
```go
func (c *ConnectionWrapper) SetDeadline(t time.Time) error {
    if err := c.ws.SetReadDeadline(t); err != nil {
        return fmt.Errorf("set read deadline: %w", err)
    }
    return c.ws.SetWriteDeadline(t)
}
func (c *ConnectionWrapper) SetReadDeadline(t time.Time) error {
    return c.ws.SetReadDeadline(t)
}
func (c *ConnectionWrapper) SetWriteDeadline(t time.Time) error {
    return c.ws.SetWriteDeadline(t)
}
```

- **Acceptance**: `go vet ./...` passes. Doom increments after 30 s of client silence. Removing the direct `wsConn.SetReadDeadline` call does not break timeout behaviour.
- **Validation**:
  ```bash
  cd cmd/server && go vet ./... && go test ./... && echo TEST_PASS
  ```

---

### Step 5: Implement Reconnection Token System for Session State Restoration

- **Deliverable**: Add an in-memory reconnection token map to `GameServer`. On first connect, generate a UUID token and send it to the client as a `connectionStatus` message with `{"type":"connectionStatus","playerId":"...","reconnectToken":"...","status":"connected"}`. On reconnect, the client sends the token in its first message; the server finds the matching `Player` record, reattaches it (`Connected: true`), restores `ActionsRemaining`, and skips creating a new player. Retain disconnected player records for a configurable grace period (60 s default, defined in `cmd/server/constants.go`) before permanently removing the slot.
- **Files**: `cmd/server/game_server.go`, `cmd/server/types.go` (add `ReconnectToken` to `Player`; extend `ConnectionStatusMessage`), `cmd/server/constants.go` (add `ReconnectGracePeriodSeconds`), `client/game.js` (store token in `localStorage`; send on reconnect)
- **Dependencies**: Step 4 (deadline methods must work correctly for reconnect timeout to function)
- **Goal Impact**: Closes the most significant functional gap (GAPS.md gap 3). Delivers ROADMAP Phase 1.1 session persistence goal without requiring Redis for the current single-server topology.

**Server-side token map addition to `GameServer`**:
```go
reconnectTokens map[string]string  // token -> playerID
tokenMutex      sync.Mutex
```

- **Acceptance**: A client that disconnects and reconnects within 60 s is reattached to its original `Player` record with unchanged location, health, sanity, and clues. A client reconnecting after 60 s is treated as a new player.
- **Validation**:
  ```bash
  cd cmd/server && go build . && echo BUILD_PASS
  go test ./... -run TestReconnect && echo TEST_PASS
  ```

---

### Step 6: Implement Real Error Rate Tracking

- **Deliverable**: Add an `atomic.Int64` error counter (`errorCount`) to `GameServer`. Increment it at every `log.Printf` error site: WebSocket upgrade failures, JSON unmarshal errors, action validation failures, and state corruption events. Replace the hardcoded `return 0.0` in `calculateErrorRate()` with `float64(gs.errorCount.Load()) / float64(gs.totalMessagesRecv.Load()) * 100`. Expose the counter as a `sync/atomic` field so no additional mutex is required.
- **Files**: `cmd/server/game_server.go`
- **Dependencies**: Step 1
- **Goal Impact**: Closes GAPS.md gap 5; makes ROADMAP Phase 1.2 error alerting (`errorRate > 5` threshold) actually trigger. Fixes the `arkham_horror_error_rate_percent` Prometheus metric.
- **Acceptance**: `curl http://localhost:8080/metrics | grep arkham_horror_error_rate_percent` returns a non-zero value after sending a malformed WebSocket message.
- **Validation**:
  ```bash
  cd cmd/server && go run . &
  sleep 1
  # Send malformed message via websocat or curl
  curl -sf http://localhost:8080/metrics | grep error_rate
  kill %1
  ```

---

### Step 7: Implement Real Broadcast Latency Metrics

- **Deliverable**: Record `time.Now()` immediately before `broadcastCh <- data` in `broadcastGameState`. In `broadcastHandler`, after consuming each item from `broadcastCh`, record the delta and store it in a fixed-size (last 100 samples) ring buffer protected by the existing `performanceMutex`. Expose the rolling average via `collectMessageThroughput()` and wire it into both `/metrics` (as `arkham_horror_broadcast_latency_ms`) and `/health` (in `performanceMetrics`). Remove the TODO comment at `game_server.go:919`.
- **Files**: `cmd/server/game_server.go`, `cmd/server/types.go` (add `LatencySamples []time.Duration` or ring-buffer fields)
- **Dependencies**: Step 3 (broadcast path must include `gameUpdate` messages before latency is measured end-to-end)
- **Goal Impact**: Closes GAPS.md gap 6; makes the sub-500 ms broadcast SLA verifiable through Prometheus.
- **Acceptance**: Under active play with 2+ clients, `arkham_horror_broadcast_latency_ms` is non-zero and below 500.
- **Validation**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections documentation \
    | python3 -c "
import json,sys,re
raw=sys.stdin.read(); idx=raw.find('{'); d=json.loads(raw[idx:])
todos=[t for t in d['documentation'].get('todo_comments',[]) if 'latency' in t['description'].lower()]
print('Latency TODOs remaining:', len(todos))
assert len(todos)==0, 'TODO not resolved'
print('PASS')
"
  ```

---

### Step 8: Fix `/dashboard` Relative Path (HTTP 404)

- **Deliverable**: Change `handleDashboard` in `cmd/server/game_server.go` from `http.ServeFile(w, r, "./client/dashboard.html")` to `http.ServeFile(w, r, "../client/dashboard.html")`. Define a package-level constant `clientDir = "../client"` in `cmd/server/constants.go` and use it in both `utils.go` (static file server) and `game_server.go` (dashboard handler) to eliminate the path discrepancy.
- **Files**: `cmd/server/game_server.go`, `cmd/server/constants.go`, `cmd/server/utils.go`
- **Dependencies**: Step 1 (server must run from `cmd/server/`)
- **Goal Impact**: Closes GAPS.md gap 7; makes `/dashboard` actually serve the monitoring UI.
- **Acceptance**: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/dashboard` returns `200`.
- **Validation**:
  ```bash
  cd cmd/server && go run . &
  sleep 1 && curl -sf -o /dev/null -w "%{http_code}" http://localhost:8080/dashboard
  kill %1
  ```

---

### Step 9: Define Go Interfaces for Core Abstractions

- **Deliverable**: Extract at minimum three interfaces from the monolithic `main` package to enable unit testing and satisfy the codebase's stated "interface-based design" claim:
  1. `GameEngine` — methods called by `handleConnection`: `ProcessAction`, `AdvanceTurn`, `CheckEndConditions`, `BroadcastState`
  2. `StateValidator` — the existing `GameStateValidator` methods: `ValidateGameState`, `RecoverGameState`, `IsGameStateHealthy`
  3. `ConnectionQualityMonitor` — `InitializeConnectionQuality`, `UpdateConnectionQuality`, `BroadcastConnectionQuality`, `CleanupConnectionQuality`

  Move interface definitions to `cmd/server/types.go`. `GameServer` and `GameStateValidator` remain concrete implementations of these interfaces. Add one table-driven unit test per interface to `cmd/server/debug_test.go` using a mock implementation.
- **Files**: `cmd/server/types.go`, `cmd/server/game_server.go`, `cmd/server/error_recovery.go`, `cmd/server/debug_test.go`
- **Dependencies**: Steps 4–8 (all runtime fixes should be in place before refactoring abstractions)
- **Goal Impact**: Directly resolves the `total_interfaces: 0` metric; enables mock-based testing for Steps 10–11; satisfies the README architectural claim.
- **Acceptance**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections overview \
    | python3 -c "
import json,sys; raw=sys.stdin.read(); idx=raw.find('{'); d=json.loads(raw[idx:])
count=d['overview']['total_interfaces']
assert count >= 3, f'Expected >=3 interfaces, got {count}'
print(f'Interfaces defined: {count} — PASS')
"
  ```
- **Validation**: `go test ./... && echo ALL_TESTS_PASS`

---

### Step 10: Add Context Cancellation to Goroutines (Leak Prevention)

- **Deliverable**: Pass a `context.Context` (derived from a root context held by `GameServer`) to `broadcastHandler` (line 90), `actionHandler` (line 92), and the per-player ping goroutine in `startPingTimer` (line 1208). Each goroutine's loop must select on `ctx.Done()` as an exit condition. Add a `ctx context.Context` and `cancel context.CancelFunc` to `GameServer`; call `gs.cancel()` in a `Stop()` method. The `startPingTimer` goroutine already uses a `*time.Timer` — convert it to also respect a player-scoped context passed in.
- **Files**: `cmd/server/game_server.go`
- **Dependencies**: Step 9 (interfaces allow mock-based leak testing)
- **Goal Impact**: Eliminates 3 high-severity goroutine leak warnings; ensures clean server shutdown under ROADMAP Phase 1.2 resilience requirements.
- **Acceptance**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections patterns \
    | python3 -c "
import json,sys; raw=sys.stdin.read(); idx=raw.find('{'); d=json.loads(raw[idx:])
leaks=[l for l in d['patterns']['concurrency_patterns']['goroutines'].get('potential_leaks',[]) if l['risk_level']=='high']
print(f'High-risk goroutine leaks: {len(leaks)}')
assert len(leaks)==0, 'Leaks remain'
print('PASS')
"
  ```
- **Validation**: `go vet ./... && go test ./... -race && echo RACE_PASS`

---

### Step 11: Reduce Complexity in Top Three Hotspots

- **Deliverable**: Refactor the three highest-complexity functions on game-critical paths to bring each below complexity 12.0:
  1. **`RecoverGameState`** (31.1 → target ≤12): Extract `repairPlayerState`, `repairDoomCounter`, and `repairTurnOrder` as separate methods on `GameStateValidator`. Each handles one recovery scenario.
  2. **`processAction`** (27.0 → target ≤12): Extract one method per action type — `processMoveAction`, `processGatherAction`, `processInvestigateAction`, `processCastWardAction` — each returning `(bool, error)`. `processAction` becomes a dispatcher.
  3. **`handleConnection`** (19.7 → target ≤12): Extract `parseClientMessage`, `applyReconnectToken`, and `handleTimeoutDoom` to remove nested conditionals from the read loop.
- **Files**: `cmd/server/game_server.go`, `cmd/server/error_recovery.go`
- **Dependencies**: Step 9 (interfaces), Step 10 (context plumbing should be in place before restructuring handlers)
- **Goal Impact**: Reduces the 25-function hotspot count (currently Large scope) toward Medium; improves maintainability for ROADMAP Phase 2 gameplay expansion work.
- **Acceptance**:
  ```bash
  go-stats-generator analyze . --skip-tests --format json --sections functions \
    | python3 -c "
import json,sys
raw=sys.stdin.read(); idx=raw.find('{'); d=json.loads(raw[idx:])
funcs=d.get('functions',[])
above=[(f['name'],f['complexity']['overall']) for f in funcs if f['complexity']['overall']>12.0]
print(f'Functions above 12.0: {len(above)}')
for n,c in sorted(above, key=lambda x:-x[1]): print(f'  {n}: {c:.1f}')
assert len(above)<15, f'Still {len(above)} above threshold (target <15)'
print('PASS')
"
  ```
- **Validation**: `go test ./... && echo ALL_PASS`

---

## Dependency Graph

```
Step 1 (README fix)
  └── Step 2 (win condition docs)
  └── Step 3 (gameUpdate message) ──► Step 7 (latency metrics)
  └── Step 4 (deadline fix) ──────► Step 5 (reconnection)
  └── Step 6 (error rate)
  └── Step 8 (dashboard path)
       └── Steps 4–8 done ──────► Step 9 (interfaces)
                                     └── Step 10 (goroutine context)
                                           └── Step 11 (complexity reduction)
```

---

## Quick-Win Summary (Steps 1–2–8 are < 1 hour each)

| Step | Effort | Risk | Gap Closed |
|---|---|---|---|
| 1 – Fix README path | ~15 min | None | GAPS.md gap 2 |
| 2 – Document win condition | ~20 min | None | GAPS.md gap 4 |
| 8 – Fix dashboard path | ~5 min | None | GAPS.md gap 7 |
| 4 – Fix deadline no-ops | ~30 min | Low | GAPS.md gap 7 (interface contract) |
| 6 – Real error rate | ~1 hr | Low | GAPS.md gap 5 |
| 3 – gameUpdate message | ~2 hrs | Medium | GAPS.md gap 1 |
| 7 – Broadcast latency | ~2 hrs | Low | GAPS.md gap 6 |
| 5 – Reconnection tokens | ~4 hrs | Medium | GAPS.md gap 3 + ROADMAP 1.1 |
| 9 – Interfaces | ~4 hrs | Medium | Architecture claim |
| 10 – Goroutine contexts | ~3 hrs | Medium | Leak risks + ROADMAP 1.2 |
| 11 – Complexity refactor | ~6 hrs | Medium | Maintainability + ROADMAP Phase 2 |

---

## Default Thresholds Used

| Metric | Threshold | Project Baseline | Scope |
|---|---|---|---|
| Functions above complexity 9.0 | >15 = Large | 25 functions | **Large** |
| Duplication ratio | >10% = Large | 0% | None |
| Doc coverage gap | >25% = Large | 6.4% gap | Small |
| Interfaces defined | ≥1 expected | 0 | Actionable |

---

*Generated: 2026-03-15 using `go-stats-generator v1.0.0` against commit HEAD of `github.com/opd-ai/bostonfear`. Metrics source: `/tmp/metrics.json` (deleted after plan generation). Cross-referenced with: `GAPS.md`, `ROADMAP.md`, `README.md`, `AUDIT.md`.*
