# AUDIT — 2026-03-15

## Project Goals

The project is an Arkham Horror-themed multiplayer web game targeting intermediate
developers learning client-server WebSocket architecture.  Stated goals (from
`README.md`, `PLAN.md`, `ROADMAP.md`, `.github/copilot-instructions.md`):

1. **5 core game mechanics**: Location System, Resource Tracking, Action System,
   Doom Counter, Dice Resolution — all integrated and cross-system dependent.
2. **Multiplayer**: 1–6 concurrent players; join a game already in progress.
3. **Go WebSocket server** using `net.Conn`/`net.Listener`/`net.Addr` interfaces,
   goroutines, channels, and idiomatic error handling.
4. **JavaScript/Canvas client** with automatic reconnection every 5 seconds.
5. **5-message JSON protocol**: `gameState`, `playerAction`, `gameUpdate`,
   `connectionStatus`, `diceResult`.
6. **Observability**: live `/dashboard`, Prometheus `/metrics`, `/health` endpoint
   with real error rate, real latency, and comprehensive metrics.
7. **Performance SLAs**: sub-500 ms broadcast, 30-second reconnection timeout,
   stable under 6 players for 15+ minutes.
8. **Planned Ebitengine client**: desktop, WASM, and mobile builds (ROADMAP).

---

## Goal-Achievement Summary

| Goal | Status | Evidence |
|------|--------|----------|
| Location System (4 neighborhoods, adjacency) | ✅ Achieved | `constants.go:35-40`, `game_server.go:131-143` |
| Resource Tracking (Health/Sanity/Clues, bounds) | ✅ Achieved | `game_server.go:108-127`, `types.go:17-21` |
| Action System (2 actions/turn, Move/Gather/Investigate/Ward) | ✅ Achieved | `game_server.go:176-363`, `constants.go:25-32` |
| Doom Counter (0-12, tentacle/timeout increments) | ✅ Achieved | `game_server.go:308-310`, `game_server.go:519-521` |
| Dice Resolution (3-sided, configurable difficulty) | ✅ Achieved | `game_server.go:148-172`, `game_server.go:248-304` |
| Mechanic cross-integration (dice→doom, actions→resources) | ✅ Achieved | `game_server.go:212-363` |
| 1-6 concurrent players | ✅ Achieved | `constants.go:12-13`, `game_server.go:454-456` |
| Join game in progress | ✅ Achieved | `game_server.go:484-489` |
| WebSocket server with net.Conn/net.Listener | ✅ Achieved | `connection_wrapper.go:16-83`, `utils.go:22-44` |
| Goroutine/channel concurrency | ✅ Achieved | `game_server.go:96-103`, `game_server.go:1092-1131` |
| 5 JSON protocol message types | ✅ Achieved | `game_server.go:333-361`, `game.js:149-176` |
| Canvas client (HTML5, player positions, resources) | ✅ Achieved | `client/index.html`, `client/game.js` |
| `/health` endpoint | ✅ Achieved | `game_server.go:619-669` |
| `/metrics` Prometheus endpoint | ✅ Achieved | `game_server.go:682-771` |
| `/dashboard` endpoint | ✅ Achieved | `game_server.go:672-680`, path uses `clientDir` constant |
| Real error-rate metric (non-zero) | ✅ Achieved | `game_server.go:1202-1213`, `game_server.go:39,542,549,592,1102,1125,1160` |
| Broadcast latency ring buffer | ✅ Achieved | `game_server.go:968-1013` |
| Win/lose condition with clue formula | ✅ Achieved | `game_server.go:392-419` |
| 3-step README setup | ✅ Achieved | `README.md:49-64` |
| Client reconnection | ⚠️ Partial | Starts at 5 s, doubles to 30 s cap; README says "every 5 seconds" — see Finding #8 |
| Session restore on reconnect | ⚠️ Partial | Reconnect creates new player slot; old slot and its turn slot persist — see Finding #5 |
| Sub-500 ms broadcast SLA — verifiable via metrics | ⚠️ Partial | Ring buffer computed; `collectMessageThroughput` never wired into `/metrics` output — see Finding #9 |
| `totalGamesPlayed` metric accuracy | ❌ Missing | Counter initialised to 0, never incremented — see Finding #10 |
| `getSystemAlerts` served to any endpoint | ❌ Missing | Function defined but never called — see Finding #11 |
| Deadlock-free ping/pong latency measurement | ❌ Bug | `handlePongMessage` deadlocks on `qualityMutex` — see Finding #1 |
| Race-free `wsConns` map access | ❌ Bug | Disconnect cleanup writes map outside mutex — see Finding #2 |
| Game continues after player disconnect | ❌ Bug | Disconnected player stays in turn order; game can get stuck — see Finding #3 |
| Safe `sendPingToPlayer` when player absent | ❌ Bug | Nil pointer dereference path — see Finding #4 |
| Ebitengine client (desktop/WASM/mobile) | ❌ Missing | Planned — ROADMAP Phases 1–5; no code exists |

---

## Findings

### CRITICAL

- [ ] **Deadlock in `handlePongMessage` → `broadcastConnectionQuality`** —
  `cmd/server/game_server.go:1254,1271,1358` —
  `handlePongMessage` acquires `gs.qualityMutex.Lock()` (line 1254) via
  `defer gs.qualityMutex.Unlock()`, then calls `broadcastConnectionQuality()` (line
  1271).  `broadcastConnectionQuality` immediately tries to acquire
  `gs.qualityMutex.RLock()` (line 1358).  Go's `sync.RWMutex` is not reentrant;
  a goroutine that holds a write lock cannot re-acquire the same mutex for reading.
  This deadlocks every goroutine that processes a pong message.  The latency-
  measurement feature introduced to satisfy the performance-monitoring goal renders
  any ping-enabled connection permanently hung.
  **Remediation:** Extract the quality snapshot before locking, or release the write
  lock before broadcasting:
  ```go
  func (gs *GameServer) handlePongMessage(pingMsg PingMessage, receiveTime time.Time) {
      gs.qualityMutex.Lock()
      quality, exists := gs.connectionQualities[pingMsg.PlayerID]
      if !exists {
          gs.qualityMutex.Unlock()
          return
      }
      latency := float64(receiveTime.Sub(pingMsg.Timestamp).Nanoseconds()) / 1e6
      quality.LatencyMs = latency
      quality.LastPingTime = receiveTime
      gs.assessConnectionQuality(pingMsg.PlayerID)
      gs.qualityMutex.Unlock() // release before broadcasting
      gs.broadcastConnectionQuality()
  }
  ```
  Validate: `go test -race ./...` after the fix; start server, connect a client,
  wait for first ping round-trip — server must not hang.

- [ ] **Data race on `gs.wsConns` / `gs.connections` / `gs.playerConns` maps** —
  `cmd/server/game_server.go:437,576-578,1098-1106` —
  `handleConnection` reads `gs.wsConns` at line 437 with no lock held.  The
  disconnect-cleanup block (lines 576-578) deletes from all three maps without
  holding `gs.mutex`, while `broadcastHandler` iterates `gs.wsConns` under
  `gs.mutex.RLock()`.  Concurrent deletes and reads on the same map without
  synchronisation is an undefined data race; the existing `go test -race` suite
  does not exercise WebSocket paths, so it does not detect this.
  **Remediation:**
  1. Wrap the line-437 read in a mutex: acquire `gs.mutex.RLock()`, read
     `gs.wsConns`, release before proceeding.
  2. Wrap lines 576-578 in `gs.mutex.Lock() … gs.mutex.Unlock()`.
  Validate: add a concurrent-connection integration test and run with
  `go test -race -count=5 ./cmd/server/...`.

### HIGH

- [ ] **Game permanently stuck when disconnected player is in turn rotation** —
  `cmd/server/game_server.go:368-390,561-565` —
  On disconnect, a player is marked `Connected: false` (line 563) but is **not**
  removed from `gameState.TurnOrder` or `gameState.Players`.  When `advanceTurn`
  rotates to this player, it sets `ActionsRemaining = 2` (line 388), but the
  player's connection goroutine has already exited.  No goroutine reads messages for
  this player, so the 30-second I/O timeout will never fire and the turn will never
  advance.  With even one disconnected player, the game is permanently frozen.
  **Remediation:** In `advanceTurn`, skip players whose `Connected` field is `false`;
  or add a `disconnectedAt` timestamp and a ticker goroutine that auto-advances the
  turn after 30 s for disconnected players.  Also, remove disconnected players from
  `TurnOrder` in the disconnect cleanup under the mutex.
  Validate: `go test -race ./cmd/server/...` with a test that connects a player,
  disconnects it, and asserts the turn advances to the next connected player.

- [ ] **Nil pointer dereference in `sendPingToPlayer`** —
  `cmd/server/game_server.go:1324` —
  `gs.wsConns[gs.playerConns[playerID].RemoteAddr().String()]` is evaluated with
  `gs.mutex.RLock()` held.  If `playerConns[playerID]` was deleted by a concurrent
  disconnect cleanup (which races with the mutex — see Finding #2), the map returns
  `nil` and calling `.RemoteAddr().String()` panics.  Under the racy cleanup path,
  this is reachable in production with multiple concurrent players.
  **Remediation:**
  ```go
  conn, ok := gs.playerConns[playerID]
  gs.mutex.RUnlock()
  if !ok || conn == nil {
      return
  }
  wsConn, exists := gs.wsConns[conn.RemoteAddr().String()]
  if !exists {
      return
  }
  ```
  Validate: `go test -race ./cmd/server/...` covering disconnect-during-ping scenario.

- [ ] **`totalGamesPlayed` metric always reports 0** —
  `cmd/server/game_server.go:36,82,833` —
  `totalGamesPlayed` is initialised to `0` and never incremented anywhere in the
  codebase.  The Prometheus metric `arkham_horror_games_played_total` and the health
  dashboard's `TotalGamesPlayed` field permanently report `0`, invalidating the
  game-completion analytics promised by the monitoring feature.
  **Remediation:** In `checkGameEndConditions`, when `GamePhase` transitions to
  `"ended"`, add `atomic.AddInt64(&gs.totalGamesPlayed, 1)`.  Validate with
  `curl http://localhost:8080/metrics | grep games_played` after completing a game.

- [ ] **`getSystemAlerts` defined but never served** —
  `cmd/server/game_server.go:1470-1524` —
  `getSystemAlerts` constructs alert payloads for high memory, slow response time,
  high error rate, and critical doom level — matching the README's "Error rates and
  system alerts" monitoring promise — but is never called from `/health`, `/metrics`,
  or any other handler.  Operators cannot receive these alerts.
  **Remediation:** Add `"systemAlerts": gs.getSystemAlerts()` to the `healthData`
  map in `handleHealthCheck` (line 646).  Validate with
  `curl http://localhost:8080/health | jq .systemAlerts`.

- [ ] **Test coverage at 10.4% — core game paths entirely untested** —
  `cmd/server/` —
  Only `GameStateValidator` is tested (`debug_test.go`, `error_recovery_test.go`).
  The five core mechanics, all four action types, `advanceTurn`, `broadcastGameState`,
  `handleConnection`, win/lose conditions, dice distribution, and the entire
  observability stack have zero test coverage.  High-severity bugs (Findings #1–#3
  above) exist in untested code and were not caught by `go test -race`.
  **Remediation:** Add table-driven tests for `processAction` covering every action
  type and edge case (sanity=1 ward, tentacle doom increment, clue cap, etc.),
  `advanceTurn` with disconnected players, `checkGameEndConditions`, and an
  integration test using `httptest` + `gorilla/websocket` for the WebSocket flow.
  Target: ≥70% coverage via `go test -cover ./...`.

### MEDIUM

- [ ] **`broadcastConnectionQuality` may block ping goroutine indefinitely** —
  `cmd/server/game_server.go:1388` —
  `gs.broadcastCh <- statusData` is a blocking channel send.  If the broadcast
  channel (capacity 100) is full, the goroutine running the ping timer loop will
  block here while holding no lock — but it can accumulate pending sends if the
  broadcast handler falls behind.  Under six concurrent players each pinging every
  5 seconds, the channel fills when the server is under load, causing ping goroutines
  to pile up behind a full channel.
  **Remediation:** Replace with a non-blocking send, mirroring the pattern used in
  `broadcastGameState` (line 1178):
  ```go
  select {
  case gs.broadcastCh <- statusData:
  default:
      log.Printf("Broadcast channel full, dropping quality update for %s", playerID)
  }
  ```
  Validate: `go test -race -count=10 ./cmd/server/...` with 6-player load test.

- [ ] **`collectMessageThroughput` result never wired into `/metrics`** —
  `cmd/server/game_server.go:993-1013` —
  `collectMessageThroughput` computes real `BroadcastLatency` from the ring buffer
  introduced to satisfy the sub-500 ms broadcast SLA requirement, but its return
  value is never used by `handleMetrics` or `handleHealthCheck`.  The Prometheus
  output therefore lacks `arkham_horror_broadcast_latency_ms`, making the SLA
  unverifiable through the advertised monitoring endpoint.
  **Remediation:** Call `gs.collectMessageThroughput(uptime)` inside `handleMetrics`
  and emit:
  ```
  # HELP arkham_horror_broadcast_latency_ms Rolling avg broadcast write latency
  arkham_horror_broadcast_latency_ms <value>
  ```
  Validate: `curl http://localhost:8080/metrics | grep broadcast_latency` returns a
  non-zero value during active play.

- [ ] **Client reconnection delay misrepresented in README** —
  `client/game.js:12,101-113` / `README.md:Connection Behaviour` —
  The README states "The client reconnects automatically every 5 seconds."  The
  actual implementation uses exponential backoff: initial delay 5 s, doubling each
  attempt, capped at 30 s (`Math.min(reconnectDelay * 2, 30000)`).  After two
  disconnections the delay is 10 s, then 20 s, then 30 s for all subsequent attempts.
  **Remediation:** Update `README.md` Connection Behaviour section to read:
  "The client attempts reconnection starting after 5 seconds, with exponential
  backoff (doubling each attempt, maximum 30 seconds)."
  Validate: browser console shows increasing delays between reconnect attempts.

- [ ] **Turn does not skip disconnected players in `advanceTurn`** —
  `cmd/server/game_server.go:368-390` —
  (Secondary aspect of Finding #3, isolated for tracking.)  Even if the stuck-game
  deadlock is resolved by timeout, `advanceTurn` does not filter by `Connected`,
  meaning a briefly-disconnected player who reconnects as a new identity still blocks
  one turn cycle per original disconnect.
  **Remediation:** Filter `TurnOrder` to only `Connected` players in `advanceTurn`,
  or skip `Connected: false` entries in the rotation loop.

### LOW

- [ ] **Single-letter variable names `m` in `game_server.go`** —
  `cmd/server/game_server.go:913,935` —
  `go-stats-generator` flags `m` (used as the `runtime.MemStats` receiver) as a
  single-letter identifier violation.  Go convention recommends short but meaningful
  names; `ms` or `memStats` would be clearer in a public-facing codebase example.
  **Remediation:** Rename `m` to `ms` at lines 913 and 935.
  Validate: `go vet ./...` remains clean.

- [ ] **`processAction` cyclomatic complexity 32.2 (187 lines)** —
  `cmd/server/game_server.go:176-363` —
  The longest and most complex function in the codebase handles validation, four
  action types, doom updates, session tracking, game-end checking, turn advancement,
  `gameUpdate` emission, `diceResult` emission, and `gameState` broadcast — all
  under a single lock.  Any new action type requires editing this function.  The
  go-stats-generator threshold for high complexity is >15.
  **Remediation:** Extract each action case into its own method
  (`gs.performMove`, `gs.performGather`, `gs.performInvestigate`,
  `gs.performCastWard`), leaving `processAction` as a dispatch function.  This also
  improves testability.  Target: complexity ≤15 per function.
  Validate: `go-stats-generator analyze . --skip-tests | grep processAction`.

- [ ] **File naming violations (`types.go`, `constants.go`, `utils.go`)** —
  `cmd/server/` —
  `go-stats-generator` flags these as generic names with low cohesion signals.
  While not a functional problem, they are inconsistent with Go project layout
  conventions that encourage domain-driven file names (e.g., `game_types.go`,
  `game_constants.go`).
  **Remediation:** Rename to `game_types.go`, `game_constants.go`, `game_utils.go`.
  Validate: `go build ./...` succeeds after rename.

---

## Metrics Snapshot

| Metric | Value | Source |
|--------|-------|--------|
| Total lines of code | 1,299 | go-stats-generator |
| Total functions + methods | 59 | go-stats-generator |
| Total structs | 30 | go-stats-generator |
| Total interfaces | 0 | go-stats-generator |
| Average function length | 27.8 lines | go-stats-generator |
| Average cyclomatic complexity | 6.3 | go-stats-generator |
| Functions with complexity >10 | 4 (6.8%) | go-stats-generator |
| Most complex function | `processAction` (32.2 overall, 24 cyclomatic) | go-stats-generator |
| Functions >50 lines | 10 (16.9%) | go-stats-generator |
| Documentation coverage | 93.9% | go-stats-generator |
| Test coverage | 10.4% | `go test -cover ./...` |
| `go vet` warnings | 0 | `go vet ./...` |
| Race detector failures | 0 in current tests | `go test -race ./...` |
| Oversized files | 2 (`game_server.go` 1,144 lines; `types.go` 190 lines) | go-stats-generator |
| Unreferenced functions | 2 (`getSystemAlerts`, `collectMessageThroughput`) | go-stats-generator, manual |
| Duplication | Not flagged by tool | go-stats-generator |
| Packages | 1 (`main`) | `go list ./...` |

> **Note on race detector**: the 0 race failures reflect the current test scope
> (validator unit tests only).  The data races documented in Findings #1 and #2
> are on WebSocket paths not exercised by any existing test.
