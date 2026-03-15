# Implementation Plan: Functional Server Stability → AH3e Rules Completeness

> Generated: 2026-03-15 | Metrics: go-stats-generator v1.0.0 | Codebase: 2,278 LoC / 181 functions / 5 packages

## Project Context

- **What it does**: Cooperative Arkham Horror-themed multiplayer web game with a Go WebSocket server and Ebitengine/JS clients supporting 1-6 concurrent investigators.
- **Current goal**: Fix the critical server crash that makes gameplay impossible, then close the highest-impact correctness gaps before advancing AH3e rules completeness.
- **Estimated Scope**: **Medium** — 14 functions above complexity threshold 9.0; 6 open gaps in GAPS.md; 0% duplication; doc coverage at 96.5%.

---

## Goal-Achievement Status

| Stated Goal | Current Status | This Plan Addresses |
|---|---|---|
| Server accepts live WebSocket connections without panicking | ❌ CRITICAL (GAP-11: unbalanced `RUnlock`) | **Yes — Step 1** |
| Win condition scales with player count (4 clues/investigator) | ❌ HIGH (GAP-13: always fixed 12 clues) | **Yes — Step 2** |
| Thread-safe connection counting in observability | ❌ HIGH (GAP-15: `gs.connections` read under wrong mutex) | **Yes — Step 3** |
| JS reconnect token stored in `localStorage` (as documented) | ⚠️ MEDIUM (GAP-14: in-memory only) | **Yes — Step 4** |
| `performComponent` either works or returns a clear error | ⚠️ MEDIUM (GAP-12: silent opaque failure) | **Yes — Step 5** |
| Test coverage for `client/ebiten/app` and `render` packages | ⚠️ MEDIUM (GAP-16: 0 test files) | **Yes — Step 6** |
| File-based reconnect token persistence (Ebitengine client) | ⚠️ PARTIAL (ROADMAP P2: memory-only) | **Yes — Step 7** |
| CI with GitHub Actions (automated quality gates) | ❌ (ROADMAP P3: no workflows) | **Yes — Step 8** |
| Act/Agenda deck progression (narrative advancement) | ❌ (ROADMAP P4: stubs only) | **Yes — Step 9** |
| Encounter resolution (`performEncounter`) | ❌ (ROADMAP P5: stub) | **Yes — Step 10** |
| Ebitengine v2.7 → v2.8+ upgrade (deprecated API removal) | ⚠️ (ROADMAP P6: deprecated calls) | **Yes — Step 11** |
| `observability.go` split (cohesion, 714-line file) | ⚠️ (ROADMAP note) | **Yes — Step 12** |

---

## Metrics Summary

| Metric | Value | Threshold | Scope |
|---|---|---|---|
| Functions above complexity 9.0 | **14** | 5–15 = Medium | **Medium** |
| Top hotspot | `cleanupDisconnectedPlayers` (14.7) | — | `cmd/server/connection.go` |
| Duplication ratio | **0%** | <3% = Small | None |
| Doc coverage | **96.5%** | gap <10% = Small | 3 undocumented types |
| Circular dependencies | **0** | — | Clean |
| Anti-patterns (critical) | **2** resource leaks | — | `app/game.go:70`, `net.go:78` |
| Anti-patterns (high) | **9** goroutine leaks + string concat | — | `game_server.go`, `net.go`, `observability.go` |
| Unreferenced functions | **28** (mostly test helpers / stubs) | — | Low priority |

**Complexity hotspots on goal-critical paths:**

| Function | Complexity | File | Blocking Goal |
|---|---|---|---|
| `cleanupDisconnectedPlayers` | 14.7 | `cmd/server/connection.go` | GAP-11 (contains the unbalanced RUnlock) |
| `checkGameEndConditions` | 11.4 | `cmd/server/game_mechanics.go` | GAP-13 (win condition) |
| `broadcastConnectionQuality` | 11.4 | `cmd/server/observability.go` | GAP-15 (mutex mismatch) |
| `broadcastHandler` | 11.1 | `cmd/server/connection.go` | Goroutine lifecycle |
| `rollDicePool` | 10.6 | `cmd/server/game_mechanics.go` | Step 9 (Act deck wiring) |
| `advanceTurn` | 10.6 | `cmd/server/game_mechanics.go` | Step 9 |
| `drawPlayerPanel` | 10.1 | `client/ebiten/app/game.go` | Step 6 (test coverage) |
| `validateActionRequest` | 9.6 | `cmd/server/game_server.go` | Step 5 (component action) |

---

## Implementation Steps

### Step 1: Fix Critical Server Crash — Unbalanced `RUnlock` in `handleConnection` (GAP-11)

- **Deliverable**:
  - `cmd/server/connection.go`: Add matching `gs.mutex.RLock()` before the `gs.wsConns` map read on line 35, before the existing (unmatched) `gs.mutex.RUnlock()` on line 36.
  - `cmd/server/connection_test.go` (new test): `TestHandleWebSocket_NewConnection` — use `httptest.NewServer` + `gorilla/websocket.Dial` to upgrade a live WebSocket connection and assert the server reaches `runMessageLoop` without panicking.
- **Dependencies**: None — this is the highest-priority prerequisite for everything else.
- **Goal Impact**: Makes the server functional; without this fix no player can connect.
- **Acceptance**: `go test -race ./cmd/server/... -run TestHandleWebSocket_NewConnection` passes without panic; manual `wscat -c ws://localhost:8080/ws` connects successfully.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestHandleWebSocket_NewConnection
  go vet ./cmd/server/...
  ```

---

### Step 2: Fix Win Condition Player-Count Scaling (GAP-13)

- **Deliverable**:
  - `cmd/server/game_mechanics.go` (`checkActAdvance` / `checkGameEndConditions`): Replace the fixed Act deck thresholds with a `rescaleActDeck(playerCount int)` helper that sets thresholds to `4n/3`, `8n/3`, and `4n` (rounded) where `n = max(playerCount, 1)`.
  - `cmd/server/game_server.go` (`registerPlayer`): Call `gs.rescaleActDeck(len(gs.gameState.Players))` when `!gs.gameState.GameStarted`.
  - `cmd/server/game_mechanics_test.go`: Unskip `TestRulesActAgendaProgression`; add sub-cases for 1P (target 4 clues), 2P (8), 4P (16), 6P (24).
- **Dependencies**: Step 1 (server must accept connections to test multiplayer counts).
- **Goal Impact**: Aligns implementation with README §Win/Lose Conditions; fixes solo play being 3× harder than documented.
- **Acceptance**: `TestRulesActAgendaProgression` sub-cases pass for all player counts; a 1-player game ends in victory when the investigator accumulates exactly 4 clues.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestRulesActAgendaProgression
  go-stats-generator analyze ./cmd/server --skip-tests --format json --sections functions | \
    python3 -c "import json,sys; d=json.load(sys.stdin); \
    [print(f['name'], f['complexity']['overall']) for f in d['functions'] \
    if 'checkGameEnd' in f['name'] or 'rescaleAct' in f['name']]"
  ```

---

### Step 3: Fix Data Race in `trackConnection` / `collectPerformanceMetrics` (GAP-15)

- **Deliverable**:
  - `cmd/server/game_server.go` (GameServer struct): Add field `activeConnections int64` (atomic counter).
  - `cmd/server/connection.go` (add player): `atomic.AddInt64(&gs.activeConnections, 1)` after writing to `gs.connections`.
  - `cmd/server/connection.go` (remove player): `atomic.AddInt64(&gs.activeConnections, -1)` before deleting from `gs.connections`.
  - `cmd/server/observability.go` (`trackConnection`, `collectPerformanceMetrics`, `handleHealthCheck`): Replace all `len(gs.connections)` reads (without `gs.mutex` held) with `int(atomic.LoadInt64(&gs.activeConnections))`.
  - Add integration test `TestConcurrentConnections_NoRace` spawning 3 simultaneous WebSocket connections and verifying the peak-connections metric.
- **Dependencies**: Step 1.
- **Goal Impact**: Eliminates an undetected data race under the server's primary use case (concurrent connections); satisfies the race-detector quality gate.
- **Acceptance**: `go test -race ./cmd/server/...` passes with no DATA RACE output; `curl http://localhost:8080/health` returns correct `activeConnections`.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestConcurrentConnections_NoRace
  go vet ./cmd/server/...
  ```

---

### Step 4: Store JS Reconnect Token in `localStorage` (GAP-14)

- **Deliverable**:
  - `client/game.js` (constructor, line ~9): Add `this.reconnectToken = localStorage.getItem('arkham_reconnect_token') || null;` to restore persisted token on page load.
  - `client/game.js` (connectionStatus handler, line ~150–151): Add `localStorage.setItem('arkham_reconnect_token', message.token);` immediately after `this.reconnectToken = message.token;`.
- **Dependencies**: None (client-only change).
- **Goal Impact**: Fulfills README §Connection Behaviour "stored in `localStorage`"; prevents zombie player slots after page refresh.
- **Acceptance**: Open game → receive token → hard refresh (`Ctrl+Shift+R`) → verify DevTools Network shows `ws://localhost:8080/ws?token=<token>` in the reconnect upgrade request; player slot is reclaimed.
- **Validation** (manual):
  ```
  1. Open http://localhost:8080 → note player ID
  2. DevTools → Application → Local Storage → confirm 'arkham_reconnect_token' is set
  3. Ctrl+Shift+R → confirm same player ID is shown in-game
  ```

---

### Step 5: Remove `ActionComponent` from Valid Actions or Implement It (GAP-12)

Choose **Option A** (remove from valid actions) until full implementation is ready (ROADMAP P9):

- **Deliverable**:
  - `cmd/server/game_mechanics.go` (`isValidActionType`): Remove `ActionComponent` from the accepted set.
  - `cmd/server/game_server.go` (`validateActionRequest`): No change needed — `isValidActionType` already gates it.
  - `cmd/server/game_mechanics_test.go` (`TestProcessAction_InvalidActionType`): Add assertion that a `component` action returns the string `"invalid action type: component"` (not a stub error).
  - Update `GAPS.md` and `RULES.md` engine-status table to reflect `ActionComponent = unregistered` until Phase 9.
- **Dependencies**: None.
- **Goal Impact**: Converts silent opaque error into explicit documented limitation; cleans server error logs of false positives.
- **Acceptance**: `go test -race ./cmd/server/... -run TestProcessAction_InvalidActionType` passes; `go vet ./...` clean.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestProcessAction_InvalidActionType
  ```

---

### Step 6: Add Test Coverage for `client/ebiten/app` and `render` Packages (GAP-16)

- **Deliverable**:
  - `client/ebiten/app/game_test.go` (new file):
    - `TestUpdate_DoesNotPanicWhenNotConnected` — construct `Game` with nil `LocalState`, call `Update()`, assert no panic.
    - `TestDrawPlayerPanel_NilPlayers` — construct minimal `Game` with empty Players map, call `drawPlayerPanel` via exported harness (or export a `DrawPlayerPanel` test shim), assert no nil-pointer panic.
  - `client/ebiten/render/atlas_test.go` (new file):
    - `TestNewAtlas_ReturnsNonNil` — call `NewAtlas()`, assert result is non-nil.
    - `TestDrawLayer_DoesNotPanic` — call `DrawLayer` with a minimal `ebiten.Image`, assert no panic.
  - `client/ebiten/state_test.go` (extend existing or new):
    - `TestLocalState_UpdateGame_Concurrent` — launch 10 goroutines calling `UpdateGame` concurrently; run under `-race`.
  - Note: Ebitengine's `ebiten.Image` operations require a headless display; use `ebiten.RunGame` only in integration tests tagged `//go:build ebitentest`. Pure logic tests (nil-safety, state mutation) need no display.
- **Dependencies**: None.
- **Goal Impact**: Closes GAP-16; catches regression in `drawPlayerPanel` (complexity 10.1, highest in ebiten packages) and atlas initialisation.
- **Acceptance**: `go test -race ./client/ebiten/...` passes; coverage for `app` and `render` packages ≥ 40% (currently 0%).
- **Validation**:
  ```bash
  go test -race -coverprofile=cover.out ./client/ebiten/...
  go tool cover -func=cover.out | grep -E 'app|render'
  ```

---

### Step 7: Implement File-Based Reconnect Token Persistence for Ebitengine Client (ROADMAP P2)

- **Deliverable**:
  - `client/ebiten/state.go`:
    - Add `tokenPath() string` returning `filepath.Join(os.UserHomeDir(), ".bostonfear", "session.json")`.
    - Add `LoadTokenFromFile() error` — reads `{"token":"<value>"}` from `tokenPath()`; sets `ls.reconnectToken` if valid.
    - Add `SaveTokenToFile() error` — writes current token to `tokenPath()` (create dir if absent).
  - `NewLocalState()`: call `LoadTokenFromFile()` (ignore error if file absent).
  - `SetReconnectToken(token string)`: call `SaveTokenToFile()` after assignment.
  - `client/ebiten/state_test.go`: Add `TestTokenPersistence_RoundTrip` — save token, create new state, assert loaded token matches saved.
- **Dependencies**: Steps 1–3 (server must be stable to exercise reconnect in integration).
- **Goal Impact**: Fulfills CLIENT_SPEC.md §2 ("token stored in `~/.bostonfear/session.json`"); closing and reopening desktop client no longer loses the player slot.
- **Acceptance**: `TestTokenPersistence_RoundTrip` passes; close and reopen `cmd/desktop`, confirm same player ID is reassigned.
- **Validation**:
  ```bash
  go test -race ./client/ebiten/... -run TestTokenPersistence_RoundTrip
  cat ~/.bostonfear/session.json   # should show token after first connection
  ```

---

### Step 8: Add GitHub Actions CI (ROADMAP P3)

- **Deliverable**:
  - `.github/workflows/ci.yml` (new file):
    ```yaml
    name: CI
    on: [push, pull_request]
    jobs:
      test:
        runs-on: ubuntu-latest
        steps:
          - uses: actions/checkout@v4
          - uses: actions/setup-go@v5
            with:
              go-version: '1.24'
          - run: go vet ./...
          - run: go test -race ./...
      build:
        runs-on: ubuntu-latest
        steps:
          - uses: actions/checkout@v4
          - uses: actions/setup-go@v5
            with:
              go-version: '1.24'
          - run: go build ./cmd/server ./cmd/desktop
          - run: GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/web
    ```
- **Dependencies**: Steps 1–7 (CI must pass a green baseline before gating PRs).
- **Goal Impact**: Closes ROADMAP P3; prevents regressions from going undetected; enables confidence for future AH3e rules work.
- **Acceptance**: First workflow run passes all jobs green; subsequent PRs are blocked by CI on `go test -race` failures.
- **Validation**:
  ```bash
  # Local simulation:
  go vet ./... && go test -race ./... && go build ./cmd/server ./cmd/desktop
  GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/web
  ```

---

### Step 9: Implement Act/Agenda Deck Progression (ROADMAP P4)

- **Deliverable**:
  - `cmd/server/game_mechanics.go`:
    - `advanceActDeck()`: compare `sumClues(gs.gameState)` against `gs.gameState.ActDeck[0].ClueThreshold`; pop the front card; emit `gameUpdate` event; if deck exhausted, trigger win.
    - `advanceAgendaDeck()`: compare `gs.gameState.Doom` against `gs.gameState.AgendaDeck[0].DoomThreshold`; pop front card; if deck exhausted, trigger doom loss.
    - Wire `advanceActDeck()` into `addClueToPlayer` (called after successful investigate/ward).
    - Wire `advanceAgendaDeck()` into `incrementDoom`.
  - `cmd/server/game_mechanics_test.go`: Unskip `TestRulesActAgendaProgression`; add sub-tests for mid-game Act 2 advancement, Agenda defeat at doom 8.
- **Dependencies**: Step 2 (rescaled Act deck thresholds must be correct before testing deck progression).
- **Goal Impact**: Advances ROADMAP P4; Act/Agenda progression delivers narrative structure that is currently missing. This is the next highest-value AH3e mechanics gap after correctness fixes.
- **Acceptance**: `TestRulesActAgendaProgression` passes; `go test -race ./cmd/server/... -run TestRulesActAgenda` green.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestRulesActAgenda
  go vet ./cmd/server/...
  ```

---

### Step 10: Implement Encounter Resolution (ROADMAP P5)

- **Deliverable**:
  - `cmd/server/game_mechanics.go` (`performEncounter`): Replace stub with:
    1. Guard: return early if `EncounterDecks[player.Location]` is empty (shuffle discard if needed).
    2. Draw top card; apply `EffectType` (`sanity_loss`, `health_loss`, `clue_gain`, `doom_inc`) with appropriate resource mutation via existing helpers (`validateResources`, `incrementDoom`).
    3. Move drawn card to a discard pile; reshuffle when exhausted.
    4. Return `EncounterResult` struct (effect type, magnitude, card title) for broadcast.
  - `cmd/server/game_mechanics_test.go`: Unskip `TestRulesEncounterResolution`; cover all 4 effect types and deck-exhaustion reshuffle.
- **Dependencies**: Step 9 (encounter outcomes must respect Act state for correct clue routing).
- **Goal Impact**: Advances ROADMAP P5; encounter resolution is core AH3e gameplay that currently always no-ops.
- **Acceptance**: `TestRulesEncounterResolution` sub-tests pass for all effect types; deck reshuffles correctly when exhausted.
- **Validation**:
  ```bash
  go test -race ./cmd/server/... -run TestRulesEncounterResolution
  ```

---

### Step 11: Upgrade Ebitengine v2.7 → v2.8+ (ROADMAP P6)

- **Deliverable**:
  - `go.mod` / `go.sum`: `go get github.com/hajimehoshi/ebiten/v2@latest && go mod tidy`.
  - `client/ebiten/render/atlas.go`, `client/ebiten/app/game.go`, `client/ebiten/render/layers.go`: Replace deprecated calls:
    - `(*ebiten.Image).Dispose()` → `Deallocate()`
    - `ebiten.DeviceScaleFactor()` → monitor/window scale API per v2.8 release notes.
  - Verify no new deprecated symbols remain: `grep -rn 'Dispose\|DeviceScaleFactor' ./client/ebiten/`.
- **Dependencies**: Steps 6 (tests added); Step 8 (CI will catch regression).
- **Goal Impact**: ROADMAP P6; prevents future build breakage; aligns with upstream Ebitengine community standards.
- **Acceptance**: `go build ./cmd/desktop && GOOS=js GOARCH=wasm go build ./cmd/web` succeeds with zero deprecation warnings; `go test -race ./client/ebiten/...` still passes.
- **Validation**:
  ```bash
  go build ./cmd/desktop
  GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/web
  go vet ./client/ebiten/...
  grep -rn 'Dispose\|DeviceScaleFactor' ./client/ebiten/  # expect: no output
  ```

---

### Step 12: Split `observability.go` into Cohesive Sub-files (ROADMAP Appendix)

- **Deliverable**:
  - Split `cmd/server/observability.go` (714 lines, 34 functions, complexity hotspot `broadcastConnectionQuality` 11.4) into three files:
    - `cmd/server/metrics.go`: Prometheus metric registration, `collectPerformanceMetrics`, `handleMetrics` (~200 lines).
    - `cmd/server/health.go`: `handleHealthCheck`, health-state calculation (~150 lines).
    - `cmd/server/dashboard.go`: `handleDashboard`, WebSocket dashboard broadcast, `broadcastConnectionQuality` (~360 lines).
  - `cmd/server/observability.go`: Retain only the `ObservabilityConfig` struct and package-level init if needed; or delete entirely.
  - Confirm existing tests pass after the rename (no logic changes).
- **Dependencies**: Step 3 (atomic counter already applied; no mutex changes needed here); Step 8 (CI guards against breakage).
- **Goal Impact**: Improves package cohesion (ROADMAP Appendix note); reduces per-file burden score from 1.18; makes `broadcastConnectionQuality` (11.4 complexity) easier to refactor in isolation.
- **Acceptance**: `go build ./cmd/server`, `go test -race ./cmd/server/...` both pass; no file in `cmd/server` exceeds 350 lines; `go-stats-generator` `broadcastConnectionQuality` complexity unchanged (refactor is file-split only, not logic change).
- **Validation**:
  ```bash
  go build ./cmd/server && go test -race ./cmd/server/...
  wc -l cmd/server/metrics.go cmd/server/health.go cmd/server/dashboard.go
  go-stats-generator analyze ./cmd/server --skip-tests --format json --sections functions | \
    python3 -c "import json,sys; d=json.load(sys.stdin); \
    [print(f['name'], f['file'], f['complexity']['overall']) \
    for f in d['functions'] if f['complexity']['overall'] > 9.0]"
  ```

---

## Dependency Graph

```
Step 1 (server crash fix)
    └── Step 2 (win condition)
    └── Step 3 (mutex race)
        └── Step 7 (file token persistence)
            └── Step 8 (CI)
                └── Step 9 (Act deck)
                    └── Step 10 (encounters)
                └── Step 11 (Ebitengine upgrade)
                └── Step 12 (observability split)
Step 4 (JS localStorage) — independent
Step 5 (component action) — independent
Step 6 (ebiten tests) — independent
```

## Non-Goals (Out of Scope for This Plan)

Per RULES.md and ROADMAP.md §Non-Goals:
- Game content creation (card text, encounter narratives, scenario scripts)
- Card/scenario data files (JSON/YAML codex entries)
- Art assets or production sprites (ROADMAP P8)
- Mobile device verification (ROADMAP P7)
- Gate/Anomaly mechanics (ROADMAP P10 — depends on Steps 9–10 completing first)
- Expansion content

---

## Scope Assessment

| Category | Count | Scope |
|---|---|---|
| Functions above complexity 9.0 | 14 | **Medium** |
| Open GAPS.md items | 6 | **Medium** |
| Duplication ratio | 0% | Small |
| Doc coverage gap | 3.5% | Small |
| Total plan steps | 12 | **Medium** |

**Overall: Medium scope.** Steps 1–5 are correctness fixes achievable in 1–3 days each. Steps 6–12 are enhancement work suitable for weekly sprints. Steps 9–10 (AH3e rules completeness) are the highest-value long-term goals once the server is stable.
