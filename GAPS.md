# Implementation Gaps — 2026-05-21

## GAPS-2026-01: ReconnectToken broadcast to all clients

- **Stated Goal**: "Handle connection drops with 30-second reconnection timeout" and "session recovery" (README / GameServer GoDoc). The token is intended to let a _specific_ player reclaim their slot.
- **Current State**: `Player.ReconnectToken` is tagged `json:"reconnectToken"` and is embedded in `GameState.Players`, which is serialised and broadcast to **every** connected WebSocket client on every state change (`serverengine/connection.go:111`, `serverengine/game_server.go:564–578`). Each client receives the reconnect tokens of all other players.
- **Impact**: Any player can impersonate another by passing a stolen token as the `?token=` query parameter on reconnect. The security guarantee of the reconnect-token session system is entirely negated. In a 6-player game every player can take over every other player's slot.
- **Closing the Gap**: Tag `ReconnectToken` in the `Player` struct with `json:"-"` so it is excluded from the shared broadcast payload. Verify that `sendConnectionStatus` continues to deliver the token directly to its owner. Add an integration test that marshals `GameState` and asserts no `reconnectToken` field appears in the output.

---

## GAPS-2026-02: Doom is double-incremented by Attack and Evade actions

- **Stated Goal**: "Doom Counter: Maintain global doom tracker (0–12) that increments on failed dice rolls or turn timeouts" (task specification). "Each Tentacle result increments the doom counter" (actions.go GoDoc).
- **Current State**: `performAttack` and `performEvade` both modify `gs.gameState.Doom` directly when tentacles appear (actions.go:307, 350) **and** return `doomIncrease = tentacles`. `processActionCore` (game_server.go:451–453) applies `doomIncrease` a second time. Every tentacle from Attack or Evade advances doom by 2 instead of 1. All other dice-rolling actions (Gather, Investigate, CastWard, Research) only return `doomIncrease` and correctly rely on `processActionCore` for the single application.
- **Impact**: Game difficulty is substantially inflated for combat-heavy play. Doom races to 12 twice as fast when players fight enemies, making the game nearly unwinnable as soon as enemies spawn. The bug violates the core doom mechanic.
- **Closing the Gap**: Remove the direct `gs.gameState.Doom` mutations from `performAttack` (line 307) and `performEvade` (line 350). Let `processActionCore` remain the single write point, as it is for all other actions. Add `go test -run 'Test(Attack|Evade)$' ./serverengine/...`.

---

## GAPS-2026-03: Successful CastWard with an anomaly reduces doom by 4 instead of 2

- **Stated Goal**: "Cast Ward: On success, reduces the doom counter by 2 and seals any anomaly in the player's current location" (actions.go GoDoc).
- **Current State**: `performCastWard` (actions.go:97) decrements doom by 2, then calls `SealAnomalyAtLocation` (line 99). `SealAnomalyAtLocation` (mythos.go:290) decrements doom by another 2 when an anomaly is found. When no anomaly exists, only the first -2 applies. When an anomaly is present, the combined reduction is 4.
- **Impact**: Warding away an anomaly accelerates doom reduction unpredictably. Players who understand this can trivially defuse the doom counter by casting wards in anomaly-dense locations, undermining the intended game tension.
- **Closing the Gap**: The intended total is -2. Remove the explicit `-2` from `performCastWard` (actions.go:97) and let `SealAnomalyAtLocation` own the full -2 as the "sealing" reward. For wards with no anomaly, `SealAnomalyAtLocation` would need to return a bool indicating success so `performCastWard` can apply -2 on the no-anomaly path. Add `go test -run 'TestCastWard(WithAnomaly|NoAnomaly)$' ./serverengine/...`.

---

## GAPS-2026-04: Connection read-timeout increments doom regardless of turn ownership

- **Stated Goal**: "Handle connection drops with 30-second reconnection timeout" (README). Turn timeouts should be a consequence of a player failing to act within their allotted time.
- **Current State**: `runMessageLoop` (connection.go:245–255) fires a `doom += 1` increment and broadcasts state on _any_ 30-second read timeout, for _any_ player, regardless of whether that player is the current active-turn holder. A player who is observing the game, connected but not their turn, will cause doom to advance if they are quiet for 30 seconds. After the timeout, the player is also disconnected.
- **Impact**: Doom advances spuriously during normal gameplay whenever a non-active player pauses input. In a 6-player game where 5 players are waiting, 5 simultaneous 30-second timeouts would advance doom by 5. This breaks the core doom mechanic documented as "increments on failed dice rolls or turn timeouts."
- **Closing the Gap**: Gate the doom increment on the timed-out player holding the active turn: `if gs.gameState.CurrentPlayer == playerID && gs.gameState.GamePhase == "playing"`. Consider extending the deadline on receiving any message so active observers are not disconnected. Separate read-deadline reset from turn enforcement. Add tests for timeout doom behaviour.

---

## GAPS-2026-05: Latency percentiles reported by `/metrics` endpoint are incorrect

- **Stated Goal**: "Performance Standards: Does the server maintain stable operation with 6 concurrent players" and the Prometheus `/metrics` endpoint advertises `p50`, `p95`, `p99` broadcast latency percentiles.
- **Current State**: `BroadcastLatencyPercentiles` (metrics.go:265–291) copies the ring buffer into a slice and indexes it at positions `N*p/100` **without sorting**. The returned "percentile" values are arbitrary ring-buffer positions, not statistical percentiles.
- **Impact**: Operators relying on the Prometheus metrics for SLO alerting or capacity planning receive meaningless latency data. The `p99` label may report a very low latency that is actually the ring-buffer position corresponding to the 99th slot, not the 99th worst write duration.
- **Closing the Gap**: Sort the sample slice with `sort.Slice` before computing index positions. Add a deterministic unit test that injects a known distribution and asserts p50/p95/p99 match expected values.

---

## GAPS-2026-06: No client-side shutdown / stop mechanism for `NetClient`

- **Stated Goal**: The client supports "cross-platform deployment (desktop, WASM, mobile) from single Go codebase" and the desktop binary is expected to exit cleanly.
- **Current State**: `NetClient.Connect()` spawns a `reconnectLoop` goroutine that runs forever with no context cancellation, no stop channel, and no lifecycle method. On desktop builds, closing the Ebitengine window triggers `os.Exit`, leaving the goroutine leaked (resources not released, deferred cleanup not run).
- **Impact**: Integration tests that instantiate a `NetClient` cannot stop the background goroutine, leading to goroutine leaks in the test binary. Future desktop shutdown flows (graceful save, network close handshake) cannot be implemented cleanly.
- **Closing the Gap**: Add a `ctx context.Context` parameter to `Connect` (or a `Stop()` method backed by a `context.CancelFunc`). Thread the context through `reconnectLoop`, `waitForReconnectSignal`, and `runConnection` so the goroutine exits when the context is cancelled. Add a test that cancels the context and asserts the goroutine exits.

---

## GAPS-2026-07: Focus reroll does not allow player choice of which dice to reroll

- **Stated Goal**: "Each focus token grants one reroll of a non-success die" (rules/dice.go GoDoc, mirroring AH3e rules).
- **Current State**: `RollDicePoolWithFocus` (rules/dice.go:82–99) iterates results left-to-right and rerolls the first non-success die it finds. Players cannot choose which die to reroll, contrary to AH3e rules where the investigator selects which non-success dice to reroll (typically preferring to reroll tentacles over blanks to avoid doom).
- **Impact**: The focus mechanic is slightly weaker than documented. A player who rolls `[Tentacle, Blank]` and spends 1 focus will always reroll the tentacle (index 0), which is actually the optimal play, but this is by coincidence of iteration order. If blank appears first (`[Blank, Tentacle]`) the player is forced to reroll the blank and keep the tentacle. This inconsistency affects game balance and violates the documented player-choice contract.
- **Closing the Gap**: When player choice is not modelled server-side (single-server determinism), use a priority rule: sort non-success indices so tentacles are rerolled before blanks. Alternatively, propagate a `rerollIndices []int` field in `PlayerActionMessage` to let the client specify which dice indices to reroll.
