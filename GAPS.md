# Implementation Gaps — 2026-03-15

> This file supersedes all previous versions. It covers gaps between the project's
> stated goals (README, RULES.md, CLIENT_SPEC.md, ROADMAP.md) and the current
> implementation. Items already tracked in RULES.md are cross-referenced.

---

## GAP-01: `ConnectionWrapper.LocalAddr()` Returns Remote Address

- **Stated Goal**: Implement the `net.Conn` interface correctly so the
  `ConnectionWrapper` type can serve as a drop-in network abstraction.
  README §Technical Implementation: "Interface-based Design: Uses `net.Conn`,
  `net.Listener`, and `net.Addr` interfaces."
- **Current State**: `connection_wrapper.go:53-60` — the struct stores a single
  `addr net.Addr` field (set from `wsConn.RemoteAddr()`). Both `LocalAddr()` and
  `RemoteAddr()` return this same field. The `net.Conn` contract requires
  `LocalAddr()` to return the *local* endpoint (the server's listening address) and
  `RemoteAddr()` to return the *remote* endpoint (the client's address).
- **Impact**: Any consumer of the `net.Conn` interface that calls `LocalAddr()` —
  including future middleware, tests, or logging — will silently receive the remote
  client address. This is a latent correctness bug in the transport abstraction layer.
  The live game is unaffected today because the server never calls `LocalAddr()` on
  its own wrappers.
- **Closing the Gap**:
  1. Add a `localAddr net.Addr` field alongside `remoteAddr net.Addr` in
     `ConnectionWrapper`.
  2. Pass the listener's `Addr()` as `localAddr` when constructing the wrapper in
     `handleWebSocket`.
  3. Return the correct field from each method.
  4. Add a `TestConnectionWrapper_LocalRemoteAddrDistinct` unit test.

---

## GAP-02: README Mislabels Implemented Ebitengine Clients as "Planned"

- **Stated Goal**: README §Build Targets marks Desktop (Phase 2), WASM (Phase 3),
  and Mobile (Phase 4) as "Planned". ROADMAP.md presents Phases 1–5 as future work.
- **Current State**: All three build targets exist and compile:
  - `go build ./cmd/desktop/...` → success
  - `GOOS=js GOARCH=wasm go build ./cmd/web` → success
  - `cmd/mobile/mobile.go` binding scaffolding compiles
  - `client/ebiten/` package is substantively implemented (game loop, renderer,
    WebSocket client, input handler, Kage shaders)
- **Impact**: Contributors and evaluators underestimate the project's maturity and
  may duplicate work already done. ROADMAP milestones appear incomplete when they
  are substantially delivered.
- **Closing the Gap**:
  1. Update `README.md` Build Targets table: mark Desktop and WASM as
     "Active (alpha — placeholder sprites)" and Mobile as "Active (untested on device)".
  2. Update `ROADMAP.md` Phases 1–3 to reflect completion status.
  3. Mark Phase 5 (real art assets) as the remaining visual-fidelity work.

---

## GAP-03: Ebitengine Sprite Atlas Uses Placeholder Solid-Colour Tiles

- **Stated Goal**: CLIENT_SPEC.md §Sprite/Layer Rendering and README §Ebitengine
  Client Features describe real board artwork, token sprites, and UI overlays
  rendered through Ebitengine draw layers.
- **Current State**: `client/ebiten/render/atlas.go:73-99` — `generateAtlas()`
  fills every named sprite slot (`SpriteBackground`, `SpriteLocationDowntown`, …,
  `SpritePlayerToken`, `SpriteDoomMarker`, `SpriteActionOverlay`) with a
  solid-colour rectangle. No bitmap artwork is embedded.
- **Impact**: The rendered output does not match the stated UI/UX requirements.
  Players see coloured rectangles instead of thematic board artwork. The game is
  functional but visually incomplete.
- **Closing the Gap**:
  1. Create `client/ebiten/render/assets/` and add a sprite-sheet PNG covering all
     eight named sprites.
  2. Replace `generateAtlas()` with a function that loads the embedded PNG via
     `//go:embed assets/sprites.png` and slices it using the declared `spriteRect`
     coordinates.
  3. Keep the colour-rectangle fallback behind a `//go:build !production` tag for
     CI environments without art assets.

---

## GAP-04: JavaScript Client Reconnection Has a Silent Hard Limit

- **Stated Goal**: README §Connection Behaviour: "The client attempts reconnection
  starting after 5 seconds, with exponential backoff (doubling each attempt,
  maximum 30 seconds)." No upper bound on the number of attempts is mentioned.
- **Current State**: `client/game.js:11` — `this.maxReconnectAttempts = 10`.
  After 10 attempts (approximately 5 minutes of backoff), reconnection stops
  permanently with a console error.
- **Impact**: A player whose browser tab is backgrounded or whose network briefly
  drops for more than ~5 minutes is permanently locked out without any visible
  notification beyond a console log.
- **Closing the Gap**:
  1. Either remove the attempt cap (`maxReconnectAttempts = Infinity`) or increase it
     to a high value (e.g., 100) to match the documented "unlimited retry" behaviour.
  2. Add a visible UI notification when retries are exhausted (e.g., "Connection
     lost — please refresh the page").
  3. Update README §Connection Behaviour to document the actual cap, or remove the
     cap and note that retries continue indefinitely.

---

## GAP-05: AH3e Action System — 4 of 8 Actions Implemented

- **Stated Goal**: RULES.md §Action System specifies 8 investigator actions per
  the AH3e rulebook: Move, Gather (Resources), Investigate, Ward, Focus, Research,
  Trade, and Component (special ability activation).
- **Current State**: `game_constants.go:21-30` — Only 4 actions are defined and
  handled: `move`, `gather`, `investigate`, `ward`. The `rules_test.go` SKIP
  messages confirm: "action 'focus' not yet implemented", "action 'research' not yet
  implemented", "action 'trade' not yet implemented", "action 'component' not yet
  implemented".
- **Impact**: The game covers the project's own 4-action spec (README §Turn Structure)
  but is not compliant with the AH3e rulebook it claims to implement (RULES.md).
  Cooperative mechanics requiring trading, focusing dice, or activating abilities
  are absent.
- **Closing the Gap** (per ROADMAP Phase 6):
  1. Add `ActionFocus`, `ActionResearch`, `ActionTrade`, `ActionComponent` constants.
  2. Implement `performFocus` (award a focus token), `performResearch` (extended
     investigate with higher clue reward), `performTrade` (transfer resources
     between co-located players), `performComponent` (investigator-specific ability).
  3. Add focus token field to `Resources` struct.
  4. Unskip the `TestRulesFullActionSet/focus_not_implemented` (and sibling) tests.

---

## GAP-06: AH3e Resource System — Money, Remnants, and Focus Tokens Missing

- **Stated Goal**: RULES.md §Resources and README §Technical Implementation list
  AH3e resources as: Health, Sanity, Money, Clues, Remnants, and Focus Tokens.
- **Current State**: `game_types.go:17-22` — `Resources` struct contains only
  `Health`, `Sanity`, and `Clues`. The `rules_test.go` SKIP messages confirm:
  "Money resource not yet implemented", "Remnants resource not yet implemented",
  "Focus token resource not yet implemented".
- **Impact**: Economic gameplay (item purchase), supernatural currency (remnants),
  and dice-improvement (focus tokens) are absent. The action economy cannot be
  balanced per AH3e rules.
- **Closing the Gap**:
  1. Add `Money int`, `Remnants int`, `Focus int` to the `Resources` struct with
     appropriate bounds (Money 0–99, Remnants 0–5, Focus 0–3 per AH3e defaults).
  2. Update `validateResources()` to clamp the new fields.
  3. Wire `Focus` into dice-roll skill bonuses and `Money` into item-purchase flows.
  4. Unskip the three resource tests in `rules_test.go`.

---

## GAP-07: Mythos Phase Not Implemented

- **Stated Goal**: RULES.md §Mythos Phase specifies: draw 2 event cards, place doom
  tokens on locations, spread existing events, and resolve the mythos cup token.
  AH3e's primary game driver is the Mythos Phase alternating with the Investigator
  Phase.
- **Current State**: The game has only one phase (`"playing"`) with no Mythos Phase
  transition. Doom advances only via tentacle dice results and read-deadline
  timeouts. There is no event card system, mythos cup, or per-location doom tokens.
- **Impact**: The game's doom escalation is much slower and less thematic than AH3e.
  The core tension of the Mythos Phase—forced doom growth, event placement,
  spreading threats—is absent.
- **Closing the Gap** (per ROADMAP Phase 6):
  1. Add a `MythosPhase` game phase and a `mythosHandler()` goroutine.
  2. Implement a minimal event card deck (draw 2, place on locations, spread if
     doom token already present).
  3. Add per-location doom token tracking to `GameState`.
  4. Advance the Mythos Phase automatically after all players complete their turns.

---

## GAP-08: Encounter Resolution Not Implemented

- **Stated Goal**: RULES.md §Encounter Resolution and AH3e rules describe
  neighborhood-specific encounter decks that trigger when investigators engage
  with encounter tokens.
- **Current State**: No encounter tokens, no encounter decks, no encounter resolution
  logic exists in `game_server.go` or anywhere in the codebase.
- **Impact**: A major AH3e gameplay loop — encountering strange events and gaining
  narrative rewards or suffering thematic penalties — is absent. Investigators
  explore locations but never encounter anything.
- **Closing the Gap** (ROADMAP Phase 6):
  1. Define `EncounterCard` struct with effect type, flavor text, and resolution
     function.
  2. Add per-location encounter decks to `GameState`.
  3. Add `ActionEncounter` action type; dispatch to a `performEncounter()` handler.

---

## GAP-09: Act/Agenda Deck Progression Not Implemented

- **Stated Goal**: RULES.md §Act/Agenda Deck Progression describes AH3e's
  narrative progression engine: act cards advance on clue thresholds; agenda cards
  advance on doom thresholds.
- **Current State**: The win condition is a flat clue threshold (`playerCount × 4`).
  There are no act or agenda cards, no card draws, no narrative events. The
  `rules_test.go` SKIP: "Full act/agenda deck progression not yet implemented".
- **Impact**: The game has a functional win/lose condition but lacks the scenario
  narrative, branching objectives, and escalating agenda tension that define AH3e.
- **Closing the Gap** (ROADMAP Phase 6):
  1. Define `ActCard` and `AgendaCard` types with thresholds and effects.
  2. Add `ActDeck` and `AgendaDeck` slices to `GameState`.
  3. Call `checkActAdvance()` and `checkAgendaAdvance()` after each action and
     Mythos Phase respectively.

---

## GAP-10: Investigator Defeat / "Lost in Time and Space" Not Implemented

- **Stated Goal**: RULES.md §Investigator Defeat states that investigators are
  defeated if Health or Sanity reaches 0, entering a "lost in time and space"
  state with resource loss and relocation.
- **Current State**: `validateResources()` clamps Health and Sanity to a minimum
  of 1, preventing them from reaching 0. An investigator cannot be defeated.
  There is no "lost in time and space" state.
- **Impact**: Investigators are effectively immortal. The risk dimension of resource
  management is reduced — players cannot lose their investigator, removing a core
  source of AH3e tension.
- **Closing the Gap**:
  1. Change the lower bound in `validateResources()` for Health and Sanity from 1
     to 0.
  2. After calling `validateResources()` in `processAction()`, check if Health or
     Sanity reached 0 and transition the player to a `"defeated"` state.
  3. Implement relocation to the starting location, resource loss, and an optional
     "lost in time and space" penalty.
  4. Skip defeated players in `advanceTurn()`.

---

## GAP-11: Scenario System and Modular Difficulty Not Implemented

- **Stated Goal**: RULES.md §Scenario System and §Modular Difficulty describe
  scenario-based setup (board layout, starting doom, codex), victory conditions
  that vary per scenario, and adjustable difficulty via mythos cup composition.
- **Current State**: The game has a single hardcoded scenario: 4 fixed neighborhoods,
  doom starts at 0, win is 4 × player-count clues. There is no scenario selection,
  no modular board, no codex, no difficulty dial.
- **Impact**: Replay value is limited to a single fixed scenario. The AH3e "modular
  neighborhood" selling point and scenario variety are absent.
- **Closing the Gap** (ROADMAP Phase 6):
  1. Define a `Scenario` struct with setup parameters, victory condition function,
     and starting state.
  2. Add scenario selection to the connection flow or server startup flags.
  3. Parameterize `NewGameServer()` to accept a `Scenario`.

---

## GAP-12: Session Persistence / Reconnection Token Not Implemented

- **Stated Goal**: README §Connection Behaviour notes this as a known limitation:
  "Full session-persistence with reconnection tokens is planned for a future
  release." The Ebitengine client's `reconnectLoop()` re-dials but cannot reclaim
  the original player slot.
- **Current State**: Reconnecting creates a new `player_<UnixNano>` ID, a new
  empty player state, and appends to the turn order. The original disconnected
  player's slot remains in `gs.gameState.Players` with `Connected: false` and is
  never cleaned up.
- **Impact**: After a reconnect, a player controls a new investigator and their
  previous investigator becomes a permanent zombie in game state, skipped by
  `advanceTurn()` but never removed. Over multiple disconnects, the `Players` map
  grows unboundedly.
- **Closing the Gap**:
  1. Issue a unique token at connection time and send it in the `connectionStatus`
     message.
  2. Accept an optional `reconnect_token` query parameter on `/ws`.
  3. If the token matches a disconnected player, restore their slot instead of
     creating a new one.
  4. Add a reaper goroutine that removes zombie (disconnected, token-expired) player
     entries after a configurable TTL.

---

## Summary Table

| Gap ID | Area | AH3e Compliance | README Promise | Severity |
|--------|------|-----------------|----------------|----------|
| GAP-01 | `net.Conn` correctness | n/a | Interface-based design | CRITICAL |
| GAP-02 | Ebitengine client status | n/a | Build Targets table | HIGH |
| GAP-03 | Sprite atlas artwork | n/a | Sprite/Layer Rendering | HIGH |
| GAP-04 | JS reconnect cap | n/a | Unlimited backoff | HIGH |
| GAP-05 | Action system (4/8) | Partial | 4-action spec met | HIGH |
| GAP-06 | Resource system (3/6) | Partial | 3-resource spec met | HIGH |
| GAP-07 | Mythos Phase | Missing | Not promised by README | MEDIUM |
| GAP-08 | Encounter Resolution | Missing | Not promised by README | MEDIUM |
| GAP-09 | Act/Agenda Deck | Missing | Not promised by README | MEDIUM |
| GAP-10 | Investigator Defeat | Partial | Not promised by README | MEDIUM |
| GAP-11 | Scenario / Difficulty | Missing | Not promised by README | LOW |
| GAP-12 | Session Persistence | n/a | Future release planned | MEDIUM |
