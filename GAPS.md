# Implementation Gaps — 2026-03-15

> This file covers gaps between the project's stated goals (README, RULES.md,
> CLIENT_SPEC.md, ROADMAP.md) and the current implementation, ordered by severity.
> Cross-reference with `AUDIT.md` for full evidence and remediation commands.

---

## GAP-01: Ebitengine Client Protocol Mismatch — Empty Game State

- **Stated Goal**: README §Ebitengine Client Features — "WebSocket Connection:
  Automatic reconnection with exponential backoff"; desktop and WASM clients are
  listed as "Active (alpha — placeholder sprites)". The implication is that the
  clients receive and display live server game state.
- **Current State**: `cmd/server/game_server.go:1636-1643` broadcasts
  `{"type":"gameState","data":{…full GameState…}}`. The Ebitengine
  `serverMessage` struct (`client/ebiten/net.go:3-18`) reads `Players`, `Doom`,
  `CurrentPlayer`, etc. as **top-level JSON fields**, not from the `"data"` wrapper.
  In Go's `encoding/json`, nested fields are not promoted to the outer object; all
  game-state fields decode to their zero values (`Players = nil`, `Doom = 0`,
  `GamePhase = ""`). The board renders but shows no players, a doom counter of 0,
  and a phase of `""` — the live server state is invisible to the Ebitengine client.
  No tests cover this path; the bug has been undetected in the absence of a client
  integration test suite.
- **Impact**: The desktop and WASM clients are non-functional as game displays.
  Players connecting via `cmd/desktop` or the compiled WASM binary see a permanently
  empty board regardless of server state. This negates the stated value of the
  multi-platform Ebitengine client.
- **Closing the Gap**:
  1. In `client/ebiten/net.go:decodeGameState()`, replace direct reads of
     `msg.Players`, `msg.Doom`, etc. with an unmarshal of `msg.Data` into a
     temporary `GameState` struct: `var gs GameState; json.Unmarshal(msg.Data, &gs)`.
  2. Remove the redundant top-level field declarations from `serverMessage`
     (lines 8–18) to eliminate the false impression that they are populated.
  3. Add `client/ebiten/net_test.go` with `TestDecodeGameState_FromDataWrapper`
     that encodes a known `GameState` under the `"data"` key and asserts all fields
     round-trip correctly.
  4. Run: `go test ./client/ebiten/...`

---

## GAP-02: `ConnectionWrapper.Read()` Returns Incorrect Byte Count

- **Stated Goal**: README §Technical Implementation — "Interface-based Design: Uses
  `net.Conn`, `net.Listener`, and `net.Addr` interfaces." The `net.Conn` contract
  requires `Read` to return the number of bytes actually written into the caller's
  buffer.
- **Current State**: `cmd/server/connection_wrapper.go:37-42` —
  ```go
  copy(b, data)
  return len(data), nil
  ```
  `copy` silently truncates when `len(b) < len(data)`, but the function reports
  `len(data)` bytes as read. Any caller that uses the returned count to slice `b`
  will read beyond the actually-written region.
- **Impact**: Currently low because the server reads WebSocket messages directly
  through `*websocket.Conn` (not through the `Read` method). However,
  `ConnectionWrapper` is the project's declared `net.Conn` abstraction for testing
  and future middleware. Its incorrect `Read` makes it a broken abstraction that
  cannot safely substitute for a real `net.Conn` in any code that inspects the
  byte count.
- **Closing the Gap**:
  1. Replace `return len(data), nil` with `n := copy(b, data); return n, nil`.
  2. Optionally, if the truncation case should be an error, also return
     `io.ErrShortBuffer` when `len(b) < len(data)`.
  3. Add `TestConnectionWrapper_ReadTruncation` asserting `n == min(len(b), len(data))`.
  4. Run: `go test -race ./cmd/server/...`

---

## GAP-03: `drawMythosToken()` Is Deterministic, Not Random

- **Stated Goal**: README §Dice Mechanics and the Mythos Phase description imply
  stochastic outcomes. `drawMythosToken()` is described in its own comment as
  returning a "pseudo-random cup token."
- **Current State**: `cmd/server/game_server.go:672-674` —
  ```go
  tokens := []string{"doom", "blessing", "curse", "blank"}
  return string(tokens[gs.gameState.Doom%len(tokens)])
  ```
  The token index is `Doom mod 4`. At doom=0 the token is always `"doom"`,
  doom=1 always `"blessing"`, etc. There is zero randomness; the full Mythos Phase
  sequence is entirely predictable from the current doom value.
- **Impact**: Players can anticipate every Mythos Phase outcome, trivially optimising
  turns to reach specific doom values and avoid the `"doom"` token. The intended
  randomness and uncertainty of the Mythos Phase are absent. This also violates the
  "pseudo-random cup token" claim in the code comment.
- **Closing the Gap**:
  1. Replace the modulo expression with `mathrand.Intn(len(tokens))` (the same
     `mathrand` alias already imported in `game_server.go:9`):
     ```go
     return tokens[mathrand.Intn(len(tokens))]
     ```
  2. Add a statistical test asserting that over ≥100 calls, all four token values
     appear (probability of false failure: negligibly small).
  3. Run: `go test -race ./cmd/server/...`

---

## GAP-04: Ebitengine Client Drops Server-Issued Reconnect Token

- **Stated Goal**: README §Connection Behaviour — "The client retries indefinitely
  using exponential backoff." The server generates a unique reconnect token at
  connection time and exposes `restorePlayerByToken()` specifically so that
  reconnecting clients can reclaim their original player slot.
- **Current State**: `client/ebiten/net.go:209-218` (`applyConnectionStatus`) —
  the `ConnectionStatusData` struct has no `Token` field; the server-provided token
  is discarded. `reconnectLoop` dials with no `?token=` URL parameter, so every
  reconnect creates a new player slot (new ID, fresh resources, appended to turn
  order). The disconnected slot persists as a zombie with `Connected: false` until
  the 5-minute reaper removes it.
- **Impact**: Ebitengine players (desktop and WASM) lose their investigator on every
  disconnect, doubling the player count in game state and polluting the turn order.
  This contradicts the server's session-restoration infrastructure, which correctly
  supports the feature for the JS legacy client.
- **Closing the Gap**:
  1. Add `Token string \`json:"token"\`` to `ConnectionStatusData`
     (`client/ebiten/state.go`).
  2. Store the received token in `LocalState` (add `ReconnectToken string` field).
  3. In `reconnectLoop()`, append `?token=<token>` to the dial URL when a token
     is stored.
  4. Add an integration test asserting the dial URL includes the token on the second
     connection attempt.
  5. Run: `go test ./client/ebiten/...`

---

## GAP-05: README Incorrectly Describes Session Persistence as Unimplemented

- **Stated Goal**: README:110 — "In the current version, a disconnected player
  cannot reclaim their investigator. Reconnecting after a drop creates a new player
  slot. Full session-persistence with reconnection tokens is planned for a future
  release."
- **Current State**: This statement is factually incorrect for the JS client.
  `cmd/server/game_server.go:976` (`restorePlayerByToken`), `game_server.go:1052-1054`
  (token query parameter on `/ws`), `game_server.go:874-879` (`sendConnectionStatus`
  with token), and `client/game.js:9,70-71,151` (token stored and sent on
  reconnect) together implement full session persistence for the JS legacy client.
  The statement is only true for the Ebitengine client (see GAP-04).
- **Impact**: Developers reading the README believe a large feature is missing when
  it is already partially shipped. This may lead to duplicate implementation effort.
- **Closing the Gap**:
  1. Replace README:110 with: "The JS legacy client reclaims its player slot
     automatically using a server-issued reconnect token. The Ebitengine client
     does not yet implement token reclaim (see GAPS.md GAP-04); reconnecting via
     desktop or WASM creates a new slot."
  2. Run: `grep -n "cannot reclaim" README.md` to confirm the old text is removed.

---

## GAP-06: `performComponent()` Registered as Valid Action but Always Fails

- **Stated Goal**: README §Turn Structure lists the action set as "Move, Gather,
  Investigate, Cast Ward." The extended RULES.md adds Focus, Research, Trade, and
  Component. `game_constants.go:29` defines `ActionComponent = "component"`.
  `isValidActionType()` accepts it.
- **Current State**: `cmd/server/game_server.go:540-547` —
  `performComponent()` always returns `fmt.Errorf("component action … not yet
  implemented")`. A client that sends `{"type":"playerAction","action":"component"}`
  receives an opaque server-side error with no feedback, and the turn-action counter
  is still decremented (processAction returns early on error without decrementing
  but the validation gate already confirmed the player's action is accepted).
  
  Actually checking more carefully: `dispatchAction()` returns the error from
  `performComponent`; `processAction()` receives a non-nil `actionErr` and returns
  the error before decrementing `player.ActionsRemaining`. So the action slot is not
  consumed — but the player receives no feedback and the server logs an error.
- **Impact**: Players or developers who discover or document the `"component"` action
  type will find it silently rejected, generating misleading server errors.
- **Closing the Gap**:
  1. Short-term: Remove `ActionComponent` from the `isValidActionType()` slice so
     the server returns `"invalid action type: component"` — a clear, expected error.
  2. Long-term (ROADMAP Phase 6): Implement per-investigator ability tables and wire
     them into `performComponent()`.
  3. Run: `go test -race ./cmd/server/... -run TestProcessAction_InvalidActionType`

---

## GAP-07: `totalGamesPlayed` Counter Can Over-Count on Repeated End-Condition Checks

- **Stated Goal**: README §Monitoring — `arkham_horror_game_doom_level` and related
  counters should accurately reflect game lifecycle statistics.
- **Current State**: `cmd/server/game_server.go:747-790` —
  `checkGameEndConditions()` has no guard for `gs.gameState.GamePhase == "ended"`.
  If called multiple times after a game has ended (e.g., doom-cap hit by a tentacle
  roll, then immediately again by a Mythos Phase event before the phase guard
  propagates), `atomic.AddInt64(&gs.totalGamesPlayed, 1)` fires twice.
  `checkGameEndConditions()` is invoked from `processAction()`,
  `runMythosPhase()`, and the timeout handler — all of which can interleave.
- **Impact**: The Prometheus metric `arkham_horror_games_played` over-counts in
  edge cases, giving incorrect lifetime statistics.
- **Closing the Gap**:
  1. Add an early guard at the top of `checkGameEndConditions()`:
     ```go
     if gs.gameState.GamePhase == "ended" {
         return
     }
     ```
  2. Add a unit test calling `checkGameEndConditions()` twice on an `"ended"` game
     and asserting `gs.totalGamesPlayed == 1`.
  3. Run: `go test -race ./cmd/server/...`

---

## GAP-08: Zero Test Coverage for Ebitengine Client Packages

- **Stated Goal**: README §Technical Implementation — "Ebitengine Client Features
  (Active — alpha; placeholder sprites)". Active code warrants test coverage.
  The project has 103 passing server tests; the client has none.
- **Current State**: `client/ebiten/` and `client/ebiten/render/` have no `*_test.go`
  files. GAP-01 (the most critical current bug — empty game state) went undetected
  because there are no integration tests for the client's message-routing path.
- **Impact**: Protocol regressions, state decoding bugs, and connection behaviour
  changes in the client have no automated detection. The risk is proportional to
  the ongoing Ebitengine migration activity.
- **Closing the Gap**:
  1. Create `client/ebiten/net_test.go` with at minimum:
     - `TestDecodeGameState_FromDataWrapper` (covers GAP-01)
     - `TestApplyConnectionStatus_CapturesToken` (covers GAP-04)
     - `TestRouteMessage_UnknownType_Ignored`
  2. Create `client/ebiten/state_test.go` with:
     - `TestLocalState_Snapshot_ReturnsConsistentView`
     - `TestLocalState_UpdateGame_Concurrent` (race detector)
  3. Run: `go test -race ./client/ebiten/...`

---

## GAP-09: Focus Dice-Pool Modifier Not Integrated into Dice Resolution

- **Stated Goal**: RULES.md §Dice Pool and README §Resource Tracking acknowledge
  Focus tokens as a player resource. The `Resources` struct includes `Focus int`.
  `TestRulesDicePoolFocusModifier` (rules_test.go) is SKIP'd with the note that
  the feature is pending.
- **Current State**: `game_server.go:performFocus()` correctly awards Focus tokens.
  However, none of the `perform*` dice-rolling functions (`performInvestigate`,
  `performCastWard`, `performGather`, `performResearch`) consult `player.Resources.Focus`
  when determining dice pool size. Focus tokens are accumulate-only; they have no
  game mechanical effect.
- **Impact**: Investigators who use their action to Focus gain no benefit. The
  Focus action exists but is mechanically inert, wasting player actions.
- **Closing the Gap**:
  1. In each dice-rolling `perform*` function, add the player's `Focus` count to the
     number of dice rolled (and optionally consume 1 Focus per roll per AH3e rules).
  2. Update `Resources.Focus` clamping in `validateResources()` for post-consume values.
  3. Unskip `TestRulesDicePoolFocusModifier`.
  4. Run: `go test -race ./cmd/server/... -run TestRulesDicePoolFocusModifier`

---

## GAP-10: Gate/Anomaly Mechanics Not Implemented

- **Stated Goal**: RULES.md §Anomaly/Gate Mechanics describes gate opening and
  closing as a core AH3e gameplay loop. `TestRulesAnomalyGateMechanics`
  (rules_test.go) is SKIP'd.
- **Current State**: No `Gate` type, no gate-opening events, no gate-closing
  mechanic, and no anomaly tokens exist in any server-side file. The Mythos Phase
  places doom tokens on locations but does not open gates.
- **Impact**: A significant AH3e gameplay mechanic is absent. Location encounters
  do not expose players to gate-related risks, and investigators cannot close gates
  to reduce doom pressure.
- **Closing the Gap** (ROADMAP Phase 6):
  1. Define `Gate` and `AnomalyToken` structs; add `OpenGates []Gate` to `GameState`.
  2. Extend `runMythosPhase()` to open a gate at a random location when a doom token
     is placed there for the second time.
  3. Add a `CloseGate` action type that removes a gate and reduces doom.
  4. Unskip `TestRulesAnomalyGateMechanics`.
  5. Run: `go test -race ./cmd/server/...`

---

## Summary Table

| Gap ID | Area | Severity | README Promise vs Reality |
|--------|------|----------|---------------------------|
| GAP-01 | Ebitengine gameState decode | CRITICAL | Ebitengine clients display empty board |
| GAP-02 | `ConnectionWrapper.Read()` byte count | CRITICAL | `net.Conn` contract violated |
| GAP-03 | Mythos token randomness | HIGH | "Pseudo-random" token is deterministic |
| GAP-04 | Ebitengine reconnect token | HIGH | Desktop/WASM cannot reclaim player slot |
| GAP-05 | README reconnection description | HIGH | README describes unimplemented JS behaviour as absent |
| GAP-06 | `performComponent` always errors | HIGH | Valid action type produces opaque failure |
| GAP-07 | `totalGamesPlayed` over-count | MEDIUM | Prometheus metric inaccurate at game end |
| GAP-08 | Zero Ebitengine test coverage | MEDIUM | Active code has no regression safety net |
| GAP-09 | Focus tokens mechanically inert | MEDIUM | Focus action has no dice-pool effect |
| GAP-10 | Gate/Anomaly mechanics absent | LOW | RULES.md compliance gap (ROADMAP Phase 6) |
