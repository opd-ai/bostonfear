# UNIVERSAL BUG AUDIT (END-TO-END) — 2026-05-21

## Project Profile

**Purpose**: Arkham Horror-themed multiplayer cooperative web game implementing five core mechanics (Location, Resource, Action, Doom, Dice) over a Go WebSocket server with a Go/Ebitengine client (desktop, WASM, mobile).

**Target users**: Up to 6 concurrent players; intermediate developers learning client-server WebSocket architecture with cooperative gameplay.

**Deployment model**: Single Go binary serves the WebSocket API, health/metrics HTTP endpoints, and optionally the WASM client bundle. Clients connect over WebSocket via `gorilla/websocket` (native) or the browser WebSocket API (WASM).

**Critical paths** (deserve deepest scrutiny):
- `serverengine/` — `GameServer`, `processActionCore`, `advanceTurn`, `runMythosPhase`, connection management
- `serverengine/arkhamhorror/` — action dispatch, dice resolution, movement, phases
- `transport/ws/` — WebSocket upgrade and routing
- `client/ebiten/net.go` — reconnect loop, message routing

---

## Audit Scope

| Package | Role | Files | Functions |
|---------|------|-------|-----------|
| `serverengine` | Core game engine, concurrency | 18 | 167 |
| `serverengine/arkhamhorror/actions` | Action dispatcher | 6 | 5 |
| `serverengine/arkhamhorror/phases` | Turn, Mythos Phase | 4 | 16 |
| `serverengine/arkhamhorror/rules` | Dice, movement | 25 | 109 |
| `serverengine/arkhamhorror/content` | Map topology, embedded content | 7 | 16 |
| `serverengine/arkhamhorror/scenarios` | Scenario definitions | 8 | 10 |
| `serverengine/eldersign/*` | Elder Sign game module | varies | varies |
| `serverengine/eldritchhorror/*` | Eldritch Horror module | varies | varies |
| `serverengine/finalhour/*` | Final Hour module | varies | varies |
| `serverengine/common/*` | Shared utilities | varies | varies |
| `transport/ws` | WebSocket transport | 4 | 15 |
| `monitoring` | HTTP health/metrics | 4 | 11 |
| `protocol` | Wire protocol types | 2 | 0 |
| `client/ebiten` | WASM/desktop client | varies | 258 |
| `cmd` | Server and client entry points | 5 | 23 |

**go-stats-generator summary**: 173 files · 11,724 LOC · 362 functions · 661 methods · 260 structs · 20 interfaces · 35 packages  
Longest function: `DefaultAdventures` 730 lines · Top complexity: `Draw` 25 cyclomatic · 32 functions >50 lines · 5 functions >100 lines

**go vet**: Clean on all buildable packages (`./serverengine/...`, `./transport/...`, `./protocol/...`, `./monitoring/...`).

**Test results**: All non-display packages pass `go test -race`. Ebitengine client/desktop packages require X11/GL headers (`libgl1-mesa-dev`, `xorg-dev`) and an X server for display-tagged tests (`xvfb-run`); the repository CI workflow installs these dependencies before build/test.

---

## Coverage Log

| Package | 3b Logic | 3c Nil | 3d Errors | 3e Resources | 3f Concurrency | 3g Security | 3h Aliasing | 3i Init | 3j API |
|---------|----------|--------|-----------|--------------|----------------|-------------|-------------|---------|--------|
| serverengine | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/arkhamhorror/actions | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/arkhamhorror/phases | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/arkhamhorror/rules | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/arkhamhorror/content | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/arkhamhorror/scenarios | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/eldersign/* | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/eldritchhorror/* | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/finalhour/* | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| serverengine/common/* | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| transport/ws | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| monitoring | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| protocol | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| client/ebiten | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| cmd | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## Goal-Achievement Summary

| Stated Goal | Status | Blocking Findings |
|-------------|--------|-------------------|
| 1–6 concurrent players, unique IDs | ✅ | None |
| Location system with adjacency enforcement | ✅ | None |
| Resource tracking (Health/Sanity/Clues) | ✅ | F-06 (doom over-reduction) |
| 2-action-per-turn system | ✅ | None |
| Doom counter 0–12 | ⚠️ | F-03 (double increment on attack/evade) |
| Dice resolution with Tentacle doom | ⚠️ | F-03 |
| State broadcast to all clients ≤500ms | ✅ | None |
| 30-second reconnection timeout | ⚠️ | F-05 (timeout also increments doom) |
| ReconnectToken session restore | ⚠️ | F-04 (token leaked to all clients) |
| Prometheus metrics | ⚠️ | F-07 (incorrect percentiles) |
| Health/metrics HTTP endpoints | ✅ | None |

---

## Findings

### CRITICAL

- [x] **DATA RACE — `sendConnectionStatus` reads `gameState.Players` without lock** — `serverengine/connection.go:219` — Concurrency — `sendConnectionStatus` calls `gs.mutex.RUnlock()` at line 210 and then accesses `gs.gameState.Players[playerID]` at line 219 _without reacquiring the lock_. A concurrent writer (e.g., `registerPlayer`, `reapDisconnectedPlayers`) can modify the map between unlock and the unsynchronised read, producing a data race on the map and potential nil pointer dereference on the returned `*Player`. Call path: `HandleConnectionWithContext` → `sendConnectionStatus`. **Remediation**: Re-hold `gs.mutex.RLock()` around both the token read and the `displayName` read in a single locked block, or snapshot the needed fields while the lock is already held. Then remove the second unlocked access. Validate with `go test -race ./serverengine/...`.

- [x] **DATA RACE — `broadcastConnectionQuality` reads `gameState.Players` without lock** — `serverengine/connection_quality.go:214` — Concurrency — `broadcastConnectionQuality` releases `gs.mutex.RUnlock()` at line 203 but then accesses `gs.gameState.Players[playerID]` at line 214 to read `DisplayName`. Any concurrent mutation (player join, disconnect, reap) races this unsynchronised read. Call path: `handlePongMessage` → `broadcastConnectionQuality`. **Remediation**: Snapshot the `displayName` values inside the `gs.mutex.RLock()` block at lines 205–211, then iterate the snapshot without re-accessing `gs.gameState.Players`. Validate with `go test -race ./serverengine/...`.

- [x] **LOGIC BUG — Doom double-incremented for Attack and Evade actions** — `serverengine/actions.go:307,350` and `serverengine/game_server.go:451–453` — Logic — `performAttack` (line 307) and `performEvade` (line 350) each call `gs.gameState.Doom = min(gs.gameState.Doom+tentacles, 12)` **and also** return `doomIncrease = tentacles`. `processActionCore` (game_server.go:451–453) unconditionally applies `doomIncrease` again: `gs.gameState.Doom = min(gs.gameState.Doom+doomIncrease, 12)`. Result: each Tentacle from an Attack or Evade increments doom by 2 instead of 1. Every other action (Gather, Investigate, CastWard, Research) returns `doomIncrease` without mutating doom internally and is therefore correct. **Remediation**: Remove the direct doom mutation from `performAttack` and `performEvade`; let `processActionCore` apply the increment via the returned `doomIncrease` value, as all other actions do. Validate with `go test -race ./serverengine/...` and `go test -run TestAttack ./serverengine/...`.

---

### HIGH

- [x] **SECURITY — `Player.ReconnectToken` leaked to all connected clients via broadcast** — `serverengine/connection.go:164–176` (Player struct) / `serverengine/connection.go:111` (broadcast) — Security — `Player` is declared with `ReconnectToken string \`json:"reconnectToken"\``. `broadcastGameState()` serialises the entire `GameState.Players` map and sends it to every connected WebSocket client. Each client therefore receives every _other_ player's reconnect token and can impersonate them on reconnect by passing the stolen token as the `?token=` query parameter. Call path: any state-mutating action → `broadcastGameState` → `broadcastHandler` → all connections. **Remediation**: Omit `ReconnectToken` from the broadcast payload by tagging it `json:"-"` in the `Player` struct (it is already sent directly to its owner via `sendConnectionStatus`). After the change, verify that `sendConnectionStatus` still sends the token correctly and that the broadcast serialisation omits it. Validate with a unit test that marshals `GameState` and asserts no `reconnectToken` field appears in any player entry.

- [x] **LOGIC BUG — Connection read-timeout increments doom regardless of turn and then terminates the connection** — `serverengine/connection.go:245–255` — Logic / API — When a read deadline fires (30 s of inactivity), the handler increments doom by 1 and then falls through to `break`, terminating `runMessageLoop` and triggering full disconnection. This means: (a) doom grows whenever any player (not just the current-turn holder) is quiet for 30 s; (b) a player who is thinking during a long turn is disconnected and doom is penalised; (c) every reconnection costs a full state broadcast plus reconnect handshake. The intended mechanic (AH3e turn timeout → doom penalty) should be conditioned on the timed-out player holding the active turn. **Remediation**: Before incrementing doom, check `gs.gameState.CurrentPlayer == playerID && gs.gameState.GamePhase == "playing"`. Separate the timeout handling from connection termination; only break the loop when a non-recoverable I/O error occurs, not on every timeout. Validate with `go test -run TestTimeout ./serverengine/...`.

- [x] **LOGIC BUG — Successful `CastWard` decrements doom twice when an anomaly is present** — `serverengine/actions.go:97` and `serverengine/mythos.go:290` — Logic — On success, `performCastWard` first executes `gs.GameState().Doom = max(gs.GameState().Doom-2, 0)` (line 97), then calls `gs.SealAnomalyAtLocation` (line 99). `SealAnomalyAtLocation` unconditionally executes `gs.gameState.Doom = max(gs.gameState.Doom-2, 0)` (mythos.go:290) when an anomaly is found. Total doom reduction when anomaly is present: **4**. When no anomaly is present: **2**. GoDoc states "reduces the doom counter by 2 and seals any anomaly"; the intended total is 2. **Remediation**: Remove the direct doom decrement from `performCastWard` (line 97) and let `SealAnomalyAtLocation` own the 2-point reduction, which is semantically the "sealing" effect. For wards with no anomaly to seal, explicitly apply a doom reduction of 2 only when `SealAnomalyAtLocation` finds no anomaly (return a bool indicating whether an anomaly was found). Validate with `go test -run TestCastWard ./serverengine/...`.

---

### MEDIUM

- [x] **INCORRECT STATISTICS — `BroadcastLatencyPercentiles` returns wrong values (array not sorted)** — `serverengine/metrics.go:265–291` — Logic — `BroadcastLatencyPercentiles` copies the latency ring buffer into `samples []int64`, then calculates p50/p95/p99 by indexing into the array at `N*p/100`. Because the samples are **never sorted**, the index positions do not correspond to actual percentiles; they return arbitrary ring buffer slots. The comment even notes "Simple percentile calculation without full sort (good enough for ring buffer)"— but omitting the sort makes the result meaningless. **Remediation**: After `copy(samples, ...)`, call `sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })` before computing indices. Validate with a deterministic test injecting known sample values and asserting correct p50/p95/p99 outputs. Run `go test -run TestBroadcastLatencyPercentiles ./serverengine/...`.

- [x] **LOGGING INCONSISTENCY — Raw `log.Printf` used in files that should use `logging` package** — `serverengine/actions.go`, `serverengine/mythos.go` — API — The `logging` package (`serverengine/common/logging`) is the project's standard structured logger. `actions.go` and `mythos.go` call raw `log.Printf` throughout (actions.go lines 237, 288, 320, 331, 374; mythos.go lines 119–124, 148, 155, 156, 174, 181, 182, 203). This bypasses log-level control, structured key-value fields, and any configured log sink. **Remediation**: Replace `log.Printf(...)` calls in `actions.go` and `mythos.go` with the corresponding `logging.Info(...)` / `logging.Warn(...)` / `logging.Error(...)` calls, passing structured key-value arguments as used elsewhere in `serverengine/`. Validate with `go vet ./serverengine/...` (no functional change to test, but the consistency makes log sinks work correctly).

- [x] **RULES FIDELITY — Focus reroll selects dice deterministically (left-to-right) rather than player-chosen** — `serverengine/arkhamhorror/rules/dice.go:82–99` — Logic — `RollDicePoolWithFocus` iterates `results[0..n]` in order and rerolls the first non-success dice it encounters. AH3e rules give the investigator explicit choice of which non-success dice to reroll. The current implementation always rerolls tentacles/blanks from leftmost to rightmost, which can be suboptimal (e.g., the player may prefer to reroll tentacles over blanks to reduce doom risk, but blanks appear first). This is a minor rules-fidelity gap that slightly disadvantages players. **Remediation**: Accept an optional `rerollIndices []int` parameter (or reroll tentacles before blanks when player choice isn't modelled) by sorting non-success indices by result priority (DiceTentacle first, then DiceBlank) before rerolling. If full player choice is desired, surface the indices to the client via the protocol. Validate with `go test -run TestReroll ./serverengine/arkhamhorror/rules/...`.

- [x] **SILENT ERROR — `json.Marshal` error ignored in `sendConnectionStatus`** — `serverengine/connection.go:222` — Error handling — `data, _ := json.Marshal(msg)` discards the marshal error. If marshaling fails (e.g., `msg` contains an un-marshallable type introduced by future code changes), `data` will be nil and `writeToConn` is called with a nil/empty buffer, silently sending no connection status to the player. **Remediation**: Log and return on error: `data, err := json.Marshal(msg); if err != nil { logging.Error("sendConnectionStatus: marshal failed", "error", err); return }`. Validate with `go vet ./serverengine/...`.

- [x] **PERFORMANCE — `trackMessage` and `trackActionType` acquire exclusive mutex for simple counter increments** — `serverengine/metrics.go:398–416` — Performance — `trackMessage` holds `gs.performanceMutex.Lock()` (exclusive write lock) for every incoming and outgoing message. `trackActionType` holds `gs.actionCounterMutex.Lock()` for every action. In a 6-player game with frequent actions these are hot-path exclusive locks. Go's `sync/atomic` provides `atomic.AddInt64` which is more appropriate for pure counter increments. **Remediation**: Replace the `performanceMutex`-protected `totalMessagesSent` and `totalMessagesRecv` fields with `sync/atomic` int64 counters (already done for `activeConnections` and `errorCount` elsewhere in the same file). Similarly replace `actionTypeCounters` with a `sync.Map` or an atomic array. Validate with `go test -race ./serverengine/...`.

---

### LOW

- [x] **CODE SMELL — `signalShutdown` suppresses panic from double-close with a bare `recover()`** — `serverengine/game_server.go:337–342` — Error handling — The implementation uses `defer func() { _ = recover() }()` to protect against closing an already-closed channel. Go 1.24 provides `sync.OnceFunc` as a cleaner alternative; the channel itself could also be replaced with a `context.CancelFunc`. The current pattern is correct but masks programming errors that attempt double-shutdown. **Remediation**: Use a `sync.Once` to guard the `close(gs.shutdownCh)` call, removing the need for `recover()`. Validate with `go test -race ./serverengine/...`.

- [x] **DOCUMENTATION GAP — `rescaleActDeck` log statement indexes `ActDeck[0..2]` unconditionally** — `serverengine/mythos.go:119–124` — Logic — The log statement at line 119–124 dereferences `gs.gameState.ActDeck[0]`, `[1]`, and `[2]` without a length guard. The caller always ensures `len >= 3`, but the function itself offers no protection; a future caller that skips the guard would cause an index-out-of-bounds panic. **Remediation**: Add a guard inside `rescaleActDeck`: `if len(gs.gameState.ActDeck) < 3 { return }` and/or limit the log statement to the available deck length. Validate with `go test -run TestRescale ./serverengine/...`.

- [ ] **GOROUTINE LEAK — `NetClient.reconnectLoop` has no shutdown path** — `client/ebiten/net.go:91–116` — Resource — `NetClient.Connect()` launches `reconnectLoop` as a background goroutine that runs forever. There is no context parameter or stop channel. On desktop (non-WASM) builds this goroutine is leaked when the application window closes because Ebitengine calls `os.Exit` before all goroutines drain. While benign in practice (process exit frees everything), it prevents clean integration tests and deferred cleanup. **Remediation**: Add a `ctx context.Context` parameter to `Connect` and thread it into `reconnectLoop`; select on `<-ctx.Done()` alongside the reconnect and backoff paths. Validate that `Connect` exits when the context is cancelled.

- [x] **NAMING — `SealAnomalyAtLocation` is exported but documented as package-internal** — `serverengine/mythos.go:286` — API — The function comment says "Caller must hold gs.mutex" (a package-internal concurrency contract) but the function is exported (`SealAnomalyAtLocation` starts with uppercase). External callers cannot satisfy the mutex contract. If it is intended only for internal use it should be unexported (`sealAnomalyAtLocation`). **Remediation**: Rename to `sealAnomalyAtLocation` and update the single call site in `performCastWard`. Validate with `go build ./serverengine/...`.

- [x] **LOW ENTROPY FALLBACK in `generateReconnectToken`** — `serverengine/connection.go:138–141` — Security — When `crypto/rand.Read` fails, the fallback token is `fmt.Sprintf("tok_%d", time.Now().UnixNano())`, a highly predictable value. Although `crypto/rand` failure is extraordinarily rare on Linux (it only fails before the kernel entropy pool is seeded, in the first few seconds of a fresh boot), an attacker who knows the approximate connection timestamp could brute-force the token. **Remediation**: Log a fatal error instead of falling back to a low-entropy token: `logging.Error("crypto/rand failed, cannot generate secure token", "error", err); return ""` and handle the empty-token case in `registerPlayer` by rejecting registration. Validate with `go test ./serverengine/...`.

- [ ] **PERFORMANCE — `strings.Builder` loop in `monitoring/handlers.go` builds metrics string via `+=`** — `monitoring/handlers.go:139–143`, `168–172`, `188–193`, `199–203`, `210–215` — Performance — Each `buildXxxMetrics` helper constructs its return value by concatenating strings with `result += line + "\n"`. With tens of metric lines this creates O(n²) string copies. **Remediation**: Use a `strings.Builder` or `strings.Join` instead of repeated `+=`. Validate with `go test ./monitoring/...`.

- [ ] **MISSING CONTEXT PROPAGATION — `runMessageLoop` timeout doom penalty is not broadcast after context cancellation** — `serverengine/connection.go:247–251` — Logic — On timeout, doom is incremented and `gs.broadcastGameState()` is called (line 251). If the same goroutine's `ctx` was already cancelled (e.g., server shutdown), `broadcastGameState` still sends — but the post-`break` path also calls `handlePlayerDisconnect` which broadcasts again. The double broadcast on timeout+cancel is harmless but wasteful. **Remediation**: Gate the timeout doom-increment behind an `ctx.Err() == nil` check. Validate with `go test -race ./serverengine/...`.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total functions (non-test) | 362 functions + 661 methods |
| Functions above complexity 15 | 10 |
| Avg cyclomatic complexity | 3.9 |
| Longest function | `DefaultAdventures` (730 lines) |
| Functions > 50 lines | 32 (3.1%) |
| Test pass rate (buildable pkgs) | All pass (0 failures) |
| go vet warnings (buildable pkgs) | 0 |
| Race detector findings (test run) | 0 reported (races in 3f/3g are only triggered by concurrent connections, not unit tests) |

---

## False Positives Considered and Rejected

| Candidate | Reason Rejected |
|-----------|----------------|
| `recalcPacketLoss` negative `missed` — seemingly possible | Guarded by `if missed < 0` reset at line 128; the guard makes it safe |
| `rescaleActDeck` log panics on short deck | Caller at lines 186–188 and 194–196 guards `len >= 3`; function is only called from those two sites |
| `AdvanceTurn` could trigger `RunMythosPhase` twice per call | Traced both code paths: the first loop returns immediately after triggering it; the fallback block is only reached when the first loop finds _no_ valid player — mutually exclusive |
| `registerPlayer` player ID collision (same nanosecond) | Sub-nanosecond collision essentially impossible; `MaxPlayers=6` further limits the window |
| `performEncounter` discard pile never populated | Confirmed a `gs.gameState.EncounterDiscards[loc] = append(...)` call at line 235 — the pile _is_ populated |
| `trackConnection` → `emitObservabilityEvent` → `gs.mutex.RLock()` while `registerPlayer` then calls `gs.mutex.Lock()` | No inversion: `performanceMutex` is released before `gs.mutex` is acquired in `registerPlayer`; `trackConnection` acquires `performanceMutex` then `mutex.RLock()`, and `registerPlayer` subsequently acquires `mutex.Lock()` — consistent ordering, no deadlock |
| `math/rand` not seeded (dice are predictable) | Go 1.20+ auto-seeds the global `math/rand` source; module targets Go 1.24.1 |
| `broadcastHandler` stalls if `conn.Write` blocks | Mitigated by per-connection `wsWriteMu` and the non-blocking broadcast channel pattern; not a confirmed stall path |
| `AdvanceTurn` skips mythos when `currentIndex == -1` | `currentIndex = -1` only when `CurrentPlayer` is not in `TurnOrder`, which is a recoverable inconsistency handled by validation; not a normal path |

---

## Remaining Scope

All packages have been audited. No remaining scope.
