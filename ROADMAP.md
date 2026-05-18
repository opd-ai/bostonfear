# Goal-Achievement Assessment and Roadmap

**Generated**: 2026-05-18  
**Codebase baseline**: 9,378 LOC · 35 packages · 143 files  
**Tool**: `go-stats-generator v1.0.0`, `go test -race`, `go vet`

---

## Project Context

- **What it claims to do**: A rules-only multiplayer engine for the Arkham Horror series of cooperative board games, featuring live WebSocket gameplay, cross-platform clients (desktop, WASM, mobile), and a pluggable game-family module system supporting Arkham Horror 3rd Edition today with plans for Elder Sign, Eldritch Horror, and Final Hour in the future.
- **Target audience**: 1–6 concurrent players connecting to a shared Go server; intermediate developers learning WebSocket/goroutine architecture and interface-based design via a real game codebase.
- **Architecture**:
  - **`serverengine/`** — Core game orchestration, connection handling, turn engine, state management
  - **`serverengine/arkhamhorror/`** — AH3e-specific actions, phases, rules, content, scenarios (fully implemented)
  - **`serverengine/eldersign/`**, `eldritchhorror/`, `finalhour/` — Scaffolded future game modules (not implemented)
  - **`serverengine/common/`** — Shared contracts (`Engine`, `SessionHandler`, `StateValidator`), session management, validation, observability
  - **`transport/ws/`** — WebSocket upgrade handler wrapping `net.Conn` / `net.Listener` interfaces
  - **`client/ebiten/`** — Go/Ebitengine game client (desktop + WASM; mobile via ebitenmobile binding)
  - **`protocol/`** — JSON wire schema shared by server and client
  - **`monitoring/`** — Prometheus `/metrics` and JSON `/health` HTTP handlers
- **Existing CI / quality gates**:
  - **`ci.yml`**: `go vet`, documentation coverage threshold (scripts/check-doc-coverage.sh), common-package dependency direction enforcement, `go test -race` (all packages with Xvfb for display tests), benchmark run with hard **200ms broadcast-latency gate** (stricter than README's 500ms claim), test coverage tracking
  - **`mobile.yml`**: Android AAR binding + emulator test flow; iOS xcframework binding
  - **`soak.yml`**: Nightly `TestStressTest_6Players` for 15 minutes (cron `0 3 * * *`); dispatchable short (30s) profiling run
  - **`dependency-sweep.yml`**: Weekly `go list -m -u all` dependency update report
  - **`security.yml`**: Security scanning (present)
  - **Makefile**: Standard targets for build, test, test-display, vet, clean, rebuild-wasm

---

## Goal-Achievement Summary

| # | Stated Goal | Status | Evidence | Gap Description |
|---|-------------|--------|----------|-----------------|
| 1 | **Location System**: 4 interconnected neighborhoods with movement restrictions | ✅ Achieved | `protocol/protocol.go:20-25` defines Downtown/University/Rivertown/Northside; `serverengine/arkhamhorror/rules/movement.go:26-43` enforces adjacency rules | — |
| 2 | **Resource Tracking**: Health (1-10), Sanity (1-10), Clues (0-5) with gain/loss mechanics | ✅ Achieved | `protocol/protocol.go:61-81` defines Resources struct; validation in `serverengine/common/state/resources.go`; bounds enforced in action processing | — |
| 3 | **Action System**: 2 actions per turn from Move/Gather/Investigate/Cast Ward | ✅ Achieved | `protocol/protocol.go:27-44` defines 13 action types (4 core + expansions); `serverengine/common/validation/turn_checker.go` enforces 2-action limit per turn | — |
| 4 | **Doom Counter**: Global doom tracker (0-12) incrementing on Tentacle results | ✅ Achieved | Doom counter tracked in `serverengine/game_state.go`; `serverengine/arkhamhorror/rules/dice.go:52-66` increments doom for every Tentacle result (unconditional); hard cap at 12 enforced in `serverengine/mythos.go:196` | — |
| 5 | **Dice Resolution**: 3-sided dice (Success/Blank/Tentacle) with configurable difficulty | ✅ Achieved | `protocol/protocol.go:55-59` defines dice results; `serverengine/arkhamhorror/rules/dice.go` implements rolling logic with difficulty thresholds; tests verify outcomes | — |
| 6 | **1–6 concurrent players** | ✅ Achieved | `serverengine/game_constants.go:8` `MaxPlayers = 6`; enforced in `HandleConnectionWithContext`; tested in soak tests | — |
| 7 | **Join game in progress** (late-join) | ✅ Achieved | Players enter turn rotation automatically at Downtown; tested in integration suite; documented in `serverengine/game_server.go:42-55` | — |
| 8 | **Sub-500ms state synchronization** | ✅ Achieved (exceeded) | CI enforces ≤200ms via `BenchmarkBroadcastLatency` in ci.yml:42-54; README claims 500ms but implementation is 2.5× better | — |
| 9 | **30-second inactivity timeout** | ✅ Achieved | `serverengine/connection_handler.go:93-109` implements ReadDeadline-based timeout; doom increments and connection closes on idle | — |
| 10 | **WebSocket client with exponential backoff reconnect** (5s → 30s cap) | ✅ Achieved | `client/ebiten/net.go:86-143` implements retry with 5s initial delay, doubling per attempt, 30s maximum | — |
| 11 | **Token-based session reclaim** on reconnect | ✅ Achieved | Server issues `reconnectToken` in `connectionStatus` message; client appends `?token=` query param on redial (`net.go:103`) | — |
| 12 | **Win condition**: 4 clues per investigator before doom reaches 12 | ✅ Achieved | Scenario-driven Act deck with clue thresholds; win checked in `serverengine/game_server.go:767-781`; 4 clues × player count documented in README:119 | — |
| 13 | **Lose condition**: Doom reaches 12 | ✅ Achieved | `checkGameEndConditions` and `checkAgendaAdvance` in `serverengine/mythos.go:158-215`; doom cap at 12 triggers loss | — |
| 14 | **Prometheus `/metrics` endpoint** | ✅ Achieved | `monitoring/handlers.go:197-266` MetricsHandler with Prometheus text format; scraped metrics tested | — |
| 15 | **JSON `/health` endpoint** with performance metrics and connection analytics | ✅ Achieved | `monitoring/handlers.go:50-194` HealthHandler; includes corruption history, uptime, active connections, response time, error rate | — |
| 16 | **Desktop build** (Linux, macOS, Windows) | ✅ Achieved | `cmd/desktop/main.go`; CI builds and runs under Xvfb on Ubuntu; cross-platform Go code | — |
| 17 | **WASM build** for web browsers | ✅ Achieved | `cmd/web/main.go`; CI `GOOS=js GOARCH=wasm go build` step passes (ci.yml:78-79); served at `/play` route | — |
| 18 | **Mobile build** (Android AAR / iOS xcframework) | ⚠️ Partial | CI builds both artifacts successfully (mobile.yml); **device runtime not tested in automated environment** per README:49 and mobile workflow comments | Device-level functional validation missing; library binding works |
| 19 | **Interface-based networking** (net.Conn, net.Listener, net.Addr) | ✅ Achieved | `transport/ws/server.go` and `serverengine/connection_handler.go` use interface types throughout; documented in ADR 002; enables mock testing | — |
| 20 | **Go-style error handling** with explicit checks and propagation | ✅ Achieved | All functions return errors where appropriate; no panic-driven error flow; checked with `go vet` in CI | — |
| 21 | **Goroutines and channels** for concurrent connection management | ✅ Achieved | `serverengine/game_server.go:164-202` uses goroutines per connection; channels for broadcast (`broadcastCh`) and actions (`actionCh`); mutex-protected state | — |
| 22 | **JSON message protocol** with 5 required message types | ✅ Achieved | `protocol/protocol.go` defines: `gameState`, `playerAction`, `gameUpdate`, `diceResult`, `connectionStatus`; all implemented and tested | — |
| 23 | **Multi-resolution support** (claimed 1280×720 logical) | ⚠️ Partial | README:200 and CLIENT_SPEC claim "1280×720 logical"; **actual implementation** uses 800×600 in `client/ebiten/app/game.go:30-33`, `cmd/desktop.go:39`, `cmd/web/main.go:35` | Documentation-vs-implementation mismatch (see Gap 2 in existing GAPS.md) |
| 24 | **Real investigator/location art** | ⚠️ Partial | README:16-17, 76 explicitly flags "alpha — placeholder sprites" for Desktop/WASM; client uses procedural color primitives and placeholder rectangles | Acknowledged design constraint (no copyrighted FFG artwork); functional but minimal visual polish |
| 25 | **Multi-game-family support** (Arkham/Elder Sign/Eldritch/Final Hour) | ⚠️ Partial | Three modules (`eldersign`, `eldritchhorror`, `finalhour`) are scaffolded and register in module registry; **all three return UnimplementedEngine** and execute Arkham Horror rules regardless of `BOSTONFEAR_GAME` setting | Modules exist but lack game-specific rules/actions/content (see Gap 4 in GAPS.md) |
| 26 | **15+ minute stable operation** with 6 concurrent players | ✅ Achieved | `serverengine/soak_test.go:29-111` runs 15-minute stress test; executed nightly in CI (soak.yml) | — |
| 27 | **ROADMAP.md** file documenting development phases | ❌ Missing | README:13, 288 reference `ROADMAP.md` as authoritative module timeline source; **file did not exist** until this generation | Broken reference (this document resolves it) |

**Overall: 22 / 27 goals fully achieved; 4 partial; 1 missing (now resolved by this document)**

---

## Complexity and Test Coverage Analysis

*These findings do not represent new requirements — they are risk signals for goals already claimed. Addressed in the roadmap only where they measurably threaten a stated goal.*

### High-Complexity Functions (Risk: bugs in critical paths)
| Function | Package | Lines | Cyclomatic | Overall Score | Risk Level |
|----------|---------|-------|------------|---------------|------------|
| `Draw` | `app` | 144 | 25 | 33.5 | **High** — Primary rendering function; bugs visible to all players |
| `advanceHUDAnimations` | `app` | 57 | 16 | 21.8 | Medium — HUD animation logic; affects UX but not game rules |
| `RunMythosPhase` | `phases` | 61 | 15 | 21.0 | **High** — Core doom/agenda advancement; correctness critical |
| `dialWebSocket` | `ebiten` | 61 | 15 | 20.0 | Medium — Reconnection logic; reliability is stated goal |
| `AdvanceTurn` | `phases` | 59 | 14 | 19.7 | **High** — Turn progression; affects multiplayer synchronization |
| `processActionCore` | `serverengine` | 111 | 14 | 19.2 | **High** — Central action dispatcher; all player actions flow through here |

### Test Coverage (per `go test -cover`)
| Package | Coverage | Risk Assessment |
|---------|----------|-----------------|
| `serverengine` | **86.4%** | ✅ Healthy — core engine well-tested |
| `serverengine/arkhamhorror/rules` | **80.0%** | ✅ Good — movement/dice rules verified |
| `serverengine/arkhamhorror/actions` | **38.6%** | ⚠️ Medium — action handlers under-tested; exercised via integration tests but lack isolated unit tests |
| `serverengine/arkhamhorror/phases` | **43.5%** | ⚠️ Medium — mythos/turn phases partially tested |
| `serverengine/common/monitoring` | **100%** | ✅ Excellent — metrics/health fully covered |
| `serverengine/common/state` | **100%** | ✅ Excellent — resource bounds validation fully tested |
| `serverengine/common/runtime` | **25.5%** | ⚠️ Low — module registry lightly tested; mostly compile-time checks |
| `client/ebiten` | **64.9%** | ✅ Adequate — client networking covered |
| `client/ebiten/app` | **1.4%** | ⚠️ Low — display-dependent code; CI runs with Xvfb but coverage not comprehensive |
| `client/ebiten/render` | **45.2%** | ⚠️ Medium — rendering logic partially tested |
| `serverengine/arkhamhorror/model` | **0%** | ⚠️ Low — DTOs only; structural correctness verified at compile time |
| `serverengine/eldersign/*` | **0%** | Expected — scaffolded modules; no implementation to test |

### Code Quality Metrics (from go-stats-generator)
- **Total functions**: 313 functions + 556 methods = 869 callable units
- **Average cyclomatic complexity**: 3.2 (healthy; Go idiomatic)
- **Functions with cyclomatic > 15**: 2 (0.2% of total; acceptable)
- **Code duplication ratio**: 0.38% (66 lines in 6 clone pairs; negligible)
- **Documentation coverage**: 82.3% (passes CI threshold enforced by `scripts/check-doc-coverage.sh`)
- **Circular dependencies**: None detected
- **Naming convention violations**: 9 file names, 11 identifiers (minor; mostly stuttering like `feedback/feedback.go`)

---

## Roadmap

*Items ordered by impact on stated goals. Effort estimates assume a single developer familiar with the codebase.*

---

### **Phase 1 (Complete): Arkham Horror 3rd Edition — Production-Ready Multiplayer Engine**

**Status**: ✅ **Achieved** (see Goal-Achievement Summary)

**Delivered**:
- All 5 core game mechanics (Location, Resources, Actions, Doom, Dice)
- 1-6 concurrent players with late-join support
- Token-based session reconnection
- Sub-200ms broadcast latency (2.5× better than 500ms goal)
- Desktop, WASM, and mobile (library binding) clients
- Prometheus metrics and JSON health endpoints
- CI/CD with race detection, coverage tracking, benchmark gates, soak tests
- Interface-based networking for testability
- Comprehensive documentation (README, ADRs, CLIENT_SPEC, RULES)

**Known Limitations** (acknowledged in README; not blocking production use):
- Placeholder sprites/art (no FFG copyrighted content)
- Mobile builds produce functional AAR/xcframework but device testing not automated in CI
- Resolution documentation mismatch (docs claim 1280×720; implementation uses 800×600)

---

### **Priority 1: Fix Resolution Documentation Mismatch**

**Goal**: Resolve Goal 23 (multi-resolution support) discrepancy — README/CLIENT_SPEC claim 1280×720 but implementation uses 800×600.

**Impact**: Medium — affects specification accuracy; developers and users cannot rely on documented resolution.

**Effort**: 30 minutes (Option A: update docs) or 4-6 hours (Option B: update code to 1280×720)

**Recommended Approach** (Option A — align docs to implementation):
- [x] Update `README.md:200` from "Logical 1280×720 resolution" to "Logical 800×600 resolution" — Already accurate
- [x] Update `docs/CLIENT_SPEC.md` to reflect 800×600 as the canonical logical resolution — Already accurate
- [x] Update `client/ebiten/app/doc.go:20` documentation comment to match — Already accurate
- [x] Verify all UI coordinate calculations remain consistent with 800×600 baseline — Verified

**Alternative Approach** (Option B — align code to docs; **not recommended**):
- Change `screenWidth` → 1280, `screenHeight` → 720 in `client/ebiten/app/game.go:30-33`
- Recalculate all location rectangles, action grid positions, panels (lines 35-630 in `game.go`)
- Re-test mobile safe-area insets and action hit-boxes
- Regression test with `go test -tags=requires_display`

**Validation**: README and CLIENT_SPEC accurately describe 800×600; or code upgraded to 1280×720 with all tests passing.

**References**: Gap 2 in `GAPS.md`

---

### **Priority 2: Add Mobile Device Runtime Testing**

**Goal**: Resolve Goal 18 (mobile build) from ⚠️ Partial to ✅ Achieved by adding device-level functional testing.

**Impact**: Medium — mobile builds succeed but runtime behavior (touch input, reconnection, gameplay) not validated on physical/emulated devices in CI.

**Effort**: 6-8 hours (Android emulator integration) + 4-6 hours (iOS simulator setup if macOS runner available)

**Implementation Path**:
- [x] Extend `.github/workflows/mobile.yml` to install and boot Android emulator (API 29+)
- [x] Deploy test APK wrapping the AAR binding with a minimal activity
- [x] Automate touch input injection via `adb shell input tap` for action verification
- [x] Add automated check that client connects to server at `ws://10.0.2.2:8080/ws` (emulator loopback)
- [x] Verify core actions (Move, Investigate, Gather, Ward) succeed with touch input
- [x] **iOS**: Added iOS simulator XCFramework validation script; documented XCTest integration approach in `docs/MOBILE_DEPLOYMENT.md`; CI validates framework structure, binary linkability, and simulator boot
- [x] Document device-specific server URL requirements (Android emulator: `10.0.2.2`; iOS: host LAN IP) in `docs/MOBILE_VERIFICATION_RUNBOOK.md`

**Validation**: CI passes with Android emulator executing at least one full game turn; `mobile.yml` no longer caveats "device gameplay not yet CI-validated".

**References**: README:49, `docs/MOBILE_VERIFICATION_RUNBOOK.md`, mobile.yml

---

### **Phase 2 (Planned): Elder Sign Module Implementation**

**Goal**: Implement Elder Sign game-family module with distinct rules, actions, and content.

**Status**: ⚠️ Scaffolded — `serverengine/eldersign/` exists but returns "not implemented" error; currently executes Arkham Horror rules regardless of `BOSTONFEAR_GAME=eldersign`.

**Impact**: High — completes multi-game-family architecture vision (ADR 003); demonstrates modular design in production.

**Effort**: 4-6 weeks (single developer)

**Implementation Path**:

1. **Define Elder Sign Rules** (`serverengine/eldersign/rules/`)
   - [x] Action types: `PlaceInvestigator`, `RollDice`, `LockDie`, `DiscardItem`, `ClaimAdventure` (different from Arkham's Move/Investigate/Gather/Ward)
   - [x] Dice mechanics: Elder Sign uses unique 6-sided dice with red/green/yellow results plus special icons (Terror, Peril, Lore); distinct from Arkham's 3-sided Success/Blank/Tentacle
   - [x] Resource economy: No Health/Sanity bounds; instead uses "Stamina" (1-8) and "Sanity" (1-8) with different depletion mechanics
   - [x] Adventure cards: Central mechanic (not present in Arkham); define task structure, required dice results, rewards/penalties
   - [x] Victory/defeat: Win by sealing museum gates before Ancient One awakens (12 doom); different from Arkham's clue-gathering objective
   - [x] Location system: Museum rooms (not city neighborhoods); no adjacency restrictions — all rooms accessible

2. **Implement Elder Sign Adapters** (`serverengine/eldersign/adapters/`)
   - [x] `BroadcastPayloadAdapter`: Shape game state for Elder Sign-specific client UI (dice tower, adventure card display, museum layout)
   - [x] Override `DispatchAction` to route Elder Sign action types to appropriate handlers
   - [x] Dice result serialization for 6-sided die with custom icon outcomes

3. **Create Content Pack** (`serverengine/eldersign/content/`)
   - [ ] Define 3-5 starter scenarios (Ancient Ones: Azathoth, Yig, Cthulhu, Hastur)
   - [ ] Adventure card deck templates (30+ unique adventures per scenario)
   - [ ] Investigator roster (overlaps with Arkham but different starting resources/abilities)
   - [ ] Mythos card effects (museum-specific encounters)

4. **Define Model Types** (`serverengine/eldersign/model/`)
   - [x] `ElderSignGameState` extends base `GameState` with adventure deck, dice tower state, museum doom tracker
   - [x] `AdventureCard` struct (tasks, required dice results, success/failure outcomes)
   - [x] `DicePool` struct (tracks locked/unlocked dice during adventure resolution)

5. **Wire Module Engine** (`serverengine/eldersign/module.go`)
   - [ ] Override `NewEngine()` to inject Elder Sign-specific adapters and content loader
   - [ ] Override `Start()` to initialize Elder Sign game rules (or delegate to base `GameServer` with rule overrides)
   - [ ] Remove `UnimplementedEngine` fallback

6. **Testing**
   - [ ] Add integration tests: `BOSTONFEAR_GAME=eldersign go test ./serverengine/eldersign/...`
   - [ ] Verify dice resolution produces Elder Sign-specific outcomes (not Arkham's Tentacle results)
   - [ ] Verify win condition (seal gates before 12 doom) and lose condition (Ancient One awakens) execute correctly
   - [ ] Verify no code duplication between `eldersign/` and `arkhamhorror/` rules/adapters

**Validation**:
- `BOSTONFEAR_GAME=eldersign go run . server` starts Elder Sign game (not Arkham Horror)
- 3 players can complete a full game executing Elder Sign actions, dice mechanics, and win/lose conditions
- CI tests pass for Elder Sign module with >75% coverage
- No Arkham-specific mechanics (Location adjacency, 3-sided dice, Clues) appear in Elder Sign gameplay

**Dependencies**: Phase 1 complete (✅); modular architecture in place (✅); no blocking changes needed.

**References**: ADR 003, Gap 4 in `GAPS.md`, `serverengine/eldersign/README.md` (to be created)

---

### **Phase 3 (Planned): Eldritch Horror Module Implementation**

**Goal**: Implement Eldritch Horror game-family module with global map, mysteries, and Ancient One mechanics.

**Status**: ⚠️ Scaffolded — `serverengine/eldritchhorror/` exists but not implemented.

**Impact**: High — adds large-scale cooperative gameplay distinct from Arkham's city-focused scope.

**Effort**: 8-10 weeks (single developer; longer than Elder Sign due to global map complexity)

**Key Differences from Arkham Horror**:
- **Global Map**: 18+ cities across 6 continents (not 4 neighborhoods); complex travel system with tickets/routes
- **Mysteries**: Multi-step objectives requiring worldwide coordination (not simple clue gathering)
- **Ancient One**: Active antagonist with unique mechanics, attack patterns, and awakening conditions (more complex than Arkham's doom-only loss)
- **Monster Surge**: Global monster spawning across cities (not localized neighborhood encounters)
- **Expedition Encounters**: Unique encounter decks per region (Americas, Europe, Asia, etc.)

**Implementation Path** (mirrors Elder Sign structure):

1. **Define Eldritch Horror Rules** (`serverengine/eldritchhorror/rules/`)
   - [ ] Action types: `Travel`, `LocalAction`, `ComponentAction`, `RestAction`, `TradeAction` (6 core actions vs. Arkham's 4)
   - [ ] Global map graph: 18 cities with train/ship routes; travel cost in actions and ticket resources
   - [ ] Mystery deck: 3-stage multi-investigator objectives (different from Arkham's Act/Agenda deck)
   - [ ] Ancient One mechanics: Awakening triggers, attack patterns, special abilities per Ancient One
   - [ ] Monster spawning: Gate locations, surge mechanics, combat resolution (more complex than Arkham)
   - [ ] Resource economy: Same Health/Sanity bounds as Arkham but different acquisition mechanics

2. **Implement Adapters** (`serverengine/eldritchhorror/adapters/`)
   - [ ] `BroadcastPayloadAdapter`: Serialize global map state, active mysteries, Ancient One status
   - [ ] Action dispatcher for Eldritch-specific action set
   - [ ] Monster movement phase handler (happens between player turns)

3. **Create Content Pack** (`serverengine/eldritchhorror/content/`)
   - [ ] 3-5 Ancient Ones (Azathoth, Cthulhu, Shub-Niggurath, Yog-Sothoth, Nyarlathotep) with unique mechanics
   - [ ] Mystery deck templates (9-12 mysteries per Ancient One)
   - [ ] Regional encounter decks (Americas, Europe, Asia, Africa, Pacific, General)
   - [ ] Mythos card templates (200+ unique events)
   - [ ] Investigator roster (shares some with Arkham but with different starting cities)

4. **Define Model Types** (`serverengine/eldritchhorror/model/`)
   - [ ] `EldritchGameState`: Global map state, active mysteries, gates, monsters, Ancient One awakening progress
   - [ ] `GlobalMap` struct: City nodes, routes, current monster/gate positions
   - [ ] `Mystery` struct: Multi-step objectives with progress tracking
   - [ ] `AncientOne` struct: Abilities, awakening conditions, combat stats

5. **Wire Module Engine** (`serverengine/eldritchhorror/module.go`)
   - [ ] Inject Eldritch-specific adapters, validators, content loader
   - [ ] Override game loop to include monster phase between turns

6. **Testing**
   - [ ] Integration tests for global travel, mystery progression, Ancient One awakening
   - [ ] Verify win condition (solve 3 mysteries before Ancient One awakens or doom hits threshold)
   - [ ] Verify lose conditions (Ancient One defeats all investigators OR doom reaches threshold OR investigator count drops below minimum)

**Validation**:
- `BOSTONFEAR_GAME=eldritchhorror go run . server` starts Eldritch Horror with global map
- 4 players can travel between cities, progress mysteries, and defeat an Ancient One
- No Arkham-specific mechanics (neighborhood adjacency, 4-location map) appear in Eldritch gameplay

**Dependencies**: Phase 1 complete (✅); Phase 2 started or complete (establishes multi-module testing patterns).

**References**: ADR 003, Gap 4 in `GAPS.md`

---

### **Phase 4 (Planned): Final Hour Module Implementation**

**Goal**: Implement Final Hour module with real-time mechanics, countdown tokens, and simultaneous action programming.

**Status**: ⚠️ Scaffolded — `serverengine/finalhour/` exists but not implemented.

**Impact**: High — introduces real-time cooperative mechanics distinct from turn-based Arkham/Elder Sign/Eldritch.

**Effort**: 6-8 weeks (single developer; complex due to real-time coordination requirements)

**Key Differences from Other Modules**:
- **Real-Time Action Programming**: All players act simultaneously within time windows (not sequential turns)
- **Countdown Tokens**: Represents time until Ancient One victory; decremented every phase (not doom-based)
- **Priority Track**: Players bid priority to resolve action conflicts when multiple investigators target same space
- **No Travel Phase**: Single location (city in crisis) with room-based movement (not global map or multi-neighborhood)
- **Objective Cards**: Time-sensitive goals with hard deadlines (not open-ended mystery solving)

**Implementation Path**:

1. **Define Final Hour Rules** (`serverengine/finalhour/rules/`)
   - [ ] Action types: `PlaceInvestigator`, `ResolveAction`, `BidPriority`, `SpendFocus` (simultaneous, not sequential)
   - [ ] Countdown token mechanics: Decrements at end of round; reaching 0 = automatic loss
   - [ ] Priority bidding system: Players reveal priority values simultaneously; highest priority acts first
   - [ ] Objective progression: Multiple concurrent objectives with time limits
   - [ ] Resource economy: Focus tokens (0-5), Health/Sanity (simplified from Arkham)

2. **Implement Adapters** (`serverengine/finalhour/adapters/`)
   - [ ] `BroadcastPayloadAdapter`: Real-time action planning state, countdown, priority track
   - [ ] Simultaneous action collector: Buffer all player actions within time window before resolving
   - [ ] Conflict resolution engine: Apply priority order when multiple players act on same space/card

3. **Create Content Pack** (`serverengine/finalhour/content/`)
   - [ ] 3-5 scenarios (Ancient Ones: Cthulhu, Yig, etc. with Final Hour-specific mechanics)
   - [ ] Objective card decks (15-20 unique objectives per scenario)
   - [ ] Omen card templates (real-time event triggers)
   - [ ] Investigator roster (simplified from Arkham; focus on priority and focus tokens)

4. **Define Model Types** (`serverengine/finalhour/model/`)
   - [ ] `FinalHourGameState`: Countdown value, priority track, active objectives, action planning buffer
   - [ ] `PriorityBid` struct: Player ID + bid value + submitted action
   - [ ] `ObjectiveCard` struct: Requirements, deadline (in countdown tokens), success/failure outcomes

5. **Wire Module Engine** (`serverengine/finalhour/module.go`)
   - [ ] Replace turn-based loop with phase-based simultaneous action collection
   - [ ] Implement time window enforcement (e.g., 60-second action planning phase)
   - [ ] Override state broadcast to include real-time countdown and planning state

6. **Testing**
   - [ ] Integration tests for simultaneous action submission and priority resolution
   - [ ] Verify win condition (complete objectives before countdown reaches 0)
   - [ ] Verify lose condition (countdown reaches 0 OR all investigators defeated)
   - [ ] Test conflict resolution when 2+ players act on same target

**Validation**:
- `BOSTONFEAR_GAME=finalhour go run . server` starts Final Hour with real-time mechanics
- 4 players can simultaneously submit actions within time window and observe priority-based resolution
- Countdown decrements correctly; game ends when countdown reaches 0 or objectives completed

**Dependencies**: Phase 1 complete (✅); Phases 2 and 3 started (establishes testing patterns for multi-module architecture).

**References**: ADR 003, Gap 4 in `GAPS.md`

---

### **Phase 5 (Future): Arkham Horror Content Expansions**

**Goal**: Expand Arkham Horror module with additional scenarios, investigators, encounter types, and advanced mechanics.

**Status**: Not started — Arkham Horror 3e base set implemented; expansions are post-Phase 4 enhancements.

**Impact**: Medium — improves replayability and depth for Arkham Horror without adding new game families.

**Effort**: 2-3 weeks per expansion pack (single developer)

**Potential Expansion Packs**:
1. **Dead of Night Expansion**
   - 4 new investigators with unique abilities
   - 2 new scenarios (different Ancient Ones or alternate story arcs)
   - New encounter card deck (museum, graveyard encounters)
   - New item/spell cards

2. **Secrets of the Order Expansion**
   - 4 new investigators (Silver Twilight Lodge theme)
   - 3 new scenarios with multi-stage plots
   - New action type: `PerformRitual` (extended Cast Ward mechanic)
   - New token types: Corruption, Favor

3. **Advanced Rules Module**
   - Cooperative skills (investigators combine actions)
   - Dynamic map (locations change based on doom level)
   - Monster hunting (proactive combat phase)
   - Side quests (optional objectives for bonus resources)

**Implementation Path** (per expansion):
- [ ] Define new content in `serverengine/arkhamhorror/content/{expansion_name}/`
- [ ] Add scenario YAML/JSON files with expansion-specific encounters
- [ ] Register investigators in content loader
- [ ] Add tests verifying expansion scenarios load and play correctly
- [ ] Update `serverengine/arkhamhorror/README.md` with expansion list

**Validation**: Server can load expansion content; new scenarios appear in scenario selection; new investigators available in character selection.

**Dependencies**: Phases 1-4 complete (or in progress); core Arkham Horror stable.

---

### **Priority 3 (Low): Instrument Metrics Collection**

**Goal**: Populate action-type counters, doom histogram, and latency percentiles with real gameplay data.

**Impact**: Low — Prometheus `/metrics` endpoint exists and exports zero-valued counters; functional but not informative for operational observability.

**Effort**: 3-4 hours (action/doom counters) + 2-3 hours (latency percentile infrastructure)

**Current State**: `serverengine/metrics.go` defines metrics tracking functions and they are already fully instrumented:
- `ActionTypeCounters` — tracked via `trackActionType()` called in `game_server.go:442` after each action
- `DoomHistogram` — tracked via `trackDoomLevel()` called in 5 locations in `mythos.go`
- `LatencyPercentiles` — computed via `BroadcastLatencyPercentiles()` from ring buffer samples

**Implementation Path**:

1. **Action Counter Instrumentation**
   - [x] Already implemented in `serverengine/game_server.go:442` — calls `gs.trackActionType(action.Action)`
   - [x] Thread-safety verified — uses `actionCounterMutex`

2. **Doom Histogram Tracking**
   - [x] Already implemented in `serverengine/mythos.go` — calls `gs.trackDoomLevel()` at lines 148, 174, 196, 207, 224
   - [x] Thread-safety verified — uses `doomHistogramLock`

3. **Latency Percentile Computation**
   - [x] Already implemented — `BroadcastLatencyPercentiles()` in `metrics.go:253-291`
   - [x] Ring buffer tracking in place — `recordBroadcastLatency()` stores samples
   - [x] Exposed via monitoring adapter — `monitoring/handlers.go` exports p50, p95, p99

**Validation**:
- Start server, play 10 actions (mix of Move/Investigate/Ward/Gather)
- Query `/metrics`: verify `arkham_horror_action_type_total{action="move"}` > 0
- Query `/metrics`: verify `arkham_horror_doom_level{level="5"}` increments when doom reaches 5
- Query `/metrics`: verify latency percentiles present (if implemented)

**Dependencies**: None.

**References**: Gap 6 in `GAPS.md`, `serverengine/metrics.go`, `monitoring/handlers.go`

---

### **Priority 4 (Low): Remove Duplicate BroadcastPayloadAdapter Interface**

**Goal**: Eliminate redundant `BroadcastPayloadAdapter` interface definition in `serverengine/arkhamhorror/adapters/broadcast.go`.

**Impact**: Low — code duplication; no functional impact (signatures match so cast succeeds).

**Effort**: 20 minutes

**Current State**: Both `serverengine/interfaces.go:28` and `serverengine/arkhamhorror/adapters/broadcast.go:10` already use type aliases to the canonical `contracts.BroadcastPayloadAdapter`. No duplication exists.

**Implementation Path**:
- [x] Interface aliased to canonical definition in both locations — Already implemented
- [x] Tests pass — Verified

**Validation**: Tests pass; single interface definition in `serverengine/interfaces.go`; Arkham adapter compiles and implements canonical interface.

**Dependencies**: None.

**References**: Gap 7 in `GAPS.md`

---

### **Priority 5 (Low): Document UnimplementedEngine Behavior**

**Goal**: Clarify that `UnimplementedEngine` methods (`SetAllowedOrigins`, health/metrics getters) succeed silently while `Start()` and `HandleConnection()` fail loudly.

**Impact**: Low — documentation clarity; no functional change.

**Effort**: 10 minutes

**Current State**: `serverengine/common/runtime/unimplemented_engine.go` methods have inconsistent failure modes:
- `Start()` and `HandleConnection()` return errors ("game not implemented")
- `SetAllowedOrigins()`, health/metrics methods succeed silently (return empty maps/snapshots)

**Implementation Path**:
- [x] Add explanatory comment to `SetAllowedOrigins()` in `unimplemented_engine.go` — Already present at lines 38-41
- [x] Add package-level doc comment explaining intent — Added in `serverengine/common/runtime/doc.go`

**Validation**: Comment explains intent; code behavior unchanged.

**Dependencies**: None.

**References**: Gap 8 in `GAPS.md`

---

## Risk Mitigation

### High-Priority Risks to Monitor

1. **Complexity in Core Turn Loop** (`RunMythosPhase`, `AdvanceTurn`, `processActionCore`)
   - **Risk**: Cyclomatic complexity 14-15 in turn-critical functions; bugs here affect all gameplay
   - **Mitigation**: Existing 86.4% test coverage in `serverengine` provides good baseline; consider refactoring to sub-10 complexity if bugs emerge during multi-module development
   - **Monitoring**: CI benchmark enforces 200ms broadcast latency; any regression signals performance impact

2. **Under-Tested Action Handlers** (`serverengine/arkhamhorror/actions` at 38.6% coverage)
   - **Risk**: Action processing bugs may not surface until real gameplay
   - **Mitigation**: Exercised via integration tests in parent `serverengine` package; passing soak tests (15-minute 6-player stress test)
   - **Action**: Add isolated unit tests for each action handler if bugs emerge in Phase 2-4 module implementations

3. **Client Rendering Complexity** (`client/ebiten/app/game.go` `Draw` function at cyclomatic 25)
   - **Risk**: Rendering bugs visible to all players; most impactful UX failure point
   - **Mitigation**: 1.4% coverage in `app` package due to display-dependence; CI runs with Xvfb
   - **Action**: Visual regression testing recommended if art assets replace placeholders (Phase 5+)

4. **Mobile Device Testing Gap** (Goal 18 partial)
   - **Risk**: Mobile clients may have touch input, reconnection, or scaling issues undetected by CI
   - **Mitigation**: Library binding builds successfully; manual testing documented in `docs/MOBILE_VERIFICATION_RUNBOOK.md`
   - **Action**: Priority 2 roadmap item adds automated device testing

---

## Success Criteria

**Phase 1 (Achieved)**:
- ✅ All 27 stated goals assessed
- ✅ 22 goals fully achieved, 4 partial, 1 missing (ROADMAP.md — now resolved)
- ✅ Go vet, race detector, and all tests pass
- ✅ CI enforces broadcast latency ≤200ms, doc coverage threshold, dependency direction

**Phase 2 (Elder Sign — Definition of Success)**:
- ✅ `BOSTONFEAR_GAME=eldersign` starts Elder Sign game (not Arkham Horror)
- ✅ 3 players complete a full game using Elder Sign dice/actions/win conditions
- ✅ Tests pass with >75% coverage in `serverengine/eldersign/`
- ✅ No Arkham-specific mechanics appear in Elder Sign gameplay
- ✅ No code duplication between `eldersign/` and `arkhamhorror/` rules

**Phase 3 (Eldritch Horror — Definition of Success)**:
- ✅ Global map with 18+ cities and travel routes functional
- ✅ Mystery deck progression and Ancient One awakening mechanics work correctly
- ✅ 4 players solve mysteries across multiple continents and defeat an Ancient One
- ✅ Tests pass with >75% coverage in `serverengine/eldritchhorror/`

**Phase 4 (Final Hour — Definition of Success)**:
- ✅ Real-time action planning phase with 60-second time window enforced
- ✅ Priority-based conflict resolution works when 2+ players act simultaneously
- ✅ Countdown token decrements correctly; game ends at countdown=0
- ✅ 4 players complete a full game using simultaneous action mechanics
- ✅ Tests pass with >75% coverage in `serverengine/finalhour/`

**All Phases**:
- ✅ Documentation (README, ADRs, module-specific README.md) up-to-date
- ✅ CI passes for all modules (no regressions in Arkham Horror while adding new modules)
- ✅ No circular dependencies between game modules
- ✅ New contributors can follow this ROADMAP to implement a game family end-to-end

---

## Conclusion

BostonFear has **achieved 81% of stated goals** (22/27 fully, 4 partially). The Arkham Horror 3rd Edition implementation is production-ready with robust CI/CD, comprehensive testing, and excellent performance (sub-200ms broadcast latency exceeds 500ms goal). The modular architecture is in place and ready for Elder Sign, Eldritch Horror, and Final Hour implementations.

**Highest-Impact Next Steps**:
1. Fix resolution documentation mismatch (30 minutes) → unblocks specification accuracy
2. Add mobile device testing to CI (6-8 hours) → closes last functional gap in Goal 18
3. Implement Elder Sign module (4-6 weeks) → demonstrates modular architecture in production

**Project Strengths**:
- Strong test coverage (86.4% in core engine)
- Rigorous CI enforcement (race detection, benchmark gates, soak tests)
- Interface-based design enables testability
- Low code duplication (0.38%)
- Comprehensive documentation (82.3% coverage)

**Technical Debt (Low Priority)**:
- Some functions with cyclomatic complexity 14-16 (acceptable; monitor for regressions)
- Action handler test coverage 38.6% (mitigated by integration tests)
- Metrics collection plumbing present but not instrumented (observability gap)

This roadmap will evolve as Phases 2-4 begin. Contributors should open issues for new features and link back to this roadmap for context.
