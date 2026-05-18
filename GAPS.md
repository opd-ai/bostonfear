# Implementation Gaps — 2026-05-18

## Gap 1: Elder Sign Game Module — Scaffolded, Rules Not Implemented

- **Intended Behavior**: Server should support Elder Sign gameplay with museum-room-based movement, 6-sided dice with Terror/Peril/Lore icons, adventure card deck system, dice tower placement mechanic, and victory condition based on sealing museum gates before 12 doom. Configuration via `BOSTONFEAR_GAME=eldersign` should start an Elder Sign server accepting 1-6 players.

- **Current State**: Module scaffolding exists in `serverengine/eldersign/` with subdirectories `adapters/`, `rules/`, `scenarios/`, `model/`. The `module.go:NewEngine()` method returns `UnimplementedEngine` which always fails `Start()` with error "eldersign engine not implemented". Baseline types like `TurnBudget` and payload adapters are defined but not wired. No adventure card logic, no Elder Sign-specific dice resolution, no museum room graph, and no game loop implementation.

- **Blocked Goal**: ROADMAP.md Goal 25 (multi-game-family support) marked ⚠️ Partial. Phase 2 implementation of Elder Sign module is documented but not executed.

- **Implementation Path**:
  1. **Define Elder Sign Action Types** (`serverengine/eldersign/actions/`):
     - `PlaceInvestigator` (assign investigator to museum room)
     - `RollDice` (roll 6-sided Elder Sign dice)
     - `LockDie` (lock a die result for multi-turn tasks)
     - `DiscardItem` (spend item for reroll)
     - `ClaimAdventure` (complete adventure card after meeting requirements)
  
  2. **Implement 6-Sided Dice Mechanics** (`serverengine/eldersign/rules/dice.go`):
     - Dice results: `Terror`, `Peril`, `Lore`, `Investigation`, `Scroll`, `Tentacle`
     - Distinct from Arkham's 3-sided Success/Blank/Tentacle system
     - Adventure cards define required die result patterns (e.g., "3 Lore + 2 Investigation")
  
  3. **Build Adventure Card System** (`serverengine/eldersign/model/adventure.go`):
     - `AdventureCard` struct with multi-stage tasks, required dice results, and success/failure outcomes
     - Adventure deck management (draw, discard, reshuffle)
     - Museum room assignment per adventure
  
  4. **Museum Room Graph** (`serverengine/eldersign/rules/locations.go`):
     - Define 8-10 museum rooms (e.g., Library, Curator's Office, Exhibit Hall)
     - No adjacency restrictions (all rooms accessible from any starting point; distinct from Arkham's neighborhood adjacency)
  
  5. **Resource Economy** (`serverengine/eldersign/model/investigator.go`):
     - Stamina (1-8) and Sanity (1-8) — different bounds from Arkham's 1-10
     - Clue tokens (0-5) — shared with Arkham but acquired differently
     - Item/Spell cards specific to Elder Sign
  
  6. **Victory/Defeat Conditions** (`serverengine/eldersign/rules/win_conditions.go`):
     - **Win**: Seal N elder signs before doom reaches 12 (N = scenario-dependent, typically 12)
     - **Lose**: Doom reaches 12 OR all investigators defeated
     - Distinct from Arkham's clue-gathering objective
  
  7. **Wire Module Engine** (`serverengine/eldersign/module.go:NewEngine()`):
     - Replace `return commonruntime.NewUnimplementedEngine("eldersign"), nil` with functional engine initialization
     - Inject Elder Sign-specific adapters, validators, and content loader
     - Override game loop to handle dice tower mechanic and adventure resolution phases
  
  8. **Content Pack** (`serverengine/eldersign/content/`):
     - 3-5 Ancient One scenarios (e.g., Azathoth, Yig, Cthulhu with Elder Sign-specific mechanics)
     - 30+ adventure card templates per scenario
     - Investigator roster (overlaps with Arkham but with different starting resources)
     - Mythos event deck specific to museum setting
  
  9. **Testing**:
     - Add integration tests: `BOSTONFEAR_GAME=eldersign go test ./serverengine/eldersign/...`
     - Verify dice resolution produces Elder Sign-specific outcomes (not Arkham Tentacle results)
     - Verify win condition (seal elder signs) and lose condition (doom=12 or all investigators defeated)
     - Verify no Arkham-specific mechanics (neighborhood adjacency, 3-sided dice, clue-per-investigator win condition) leak into Elder Sign gameplay

- **Dependencies**: Arkham Horror implementation complete (✅); modular architecture in place (✅); `serverengine/common/contracts/` defines `Engine` interface (✅).

- **Effort**: 4-6 weeks (single developer)

- **Validation**:
  - `BOSTONFEAR_GAME=eldersign go run . server` starts Elder Sign game successfully
  - 3 players connect and complete a full Elder Sign game using museum rooms, 6-sided dice, and adventure cards
  - No Arkham-specific mechanics appear in gameplay
  - CI tests pass with >75% coverage in `serverengine/eldersign/`
  - `go test -race ./...` passes with Elder Sign module active

---

## Gap 2: Eldritch Horror Game Module — Scaffolded, Rules Not Implemented

- **Intended Behavior**: Server should support Eldritch Horror gameplay with global map (18+ cities), multi-step mystery deck progression, Ancient One active antagonist mechanics, monster surge system, and regional encounter decks. Configuration via `BOSTONFEAR_GAME=eldritchhorror` should start an Eldritch Horror server.

- **Current State**: Module scaffolding exists in `serverengine/eldritchhorror/` with subdirectories mirroring Elder Sign. `module.go:NewEngine()` returns `UnimplementedEngine` which always fails `Start()` with error "eldritchhorror engine not implemented". No global map implementation, no mystery deck logic, no Ancient One mechanics, and no game loop.

- **Blocked Goal**: ROADMAP.md Goal 25 (multi-game-family support) marked ⚠️ Partial. Phase 3 implementation documented but not executed.

- **Implementation Path**:
  1. **Define Eldritch Horror Action Types** (`serverengine/eldritchhorror/actions/`):
     - `Travel` (move between cities with train/ship routes)
     - `LocalAction` (city-specific actions)
     - `ComponentAction` (interact with scenario-specific components)
     - `RestAction` (recover health/sanity; distinct from Arkham's Gather)
     - `TradeAction` (exchange items with other investigators)
  
  2. **Global Map Graph** (`serverengine/eldritchhorror/rules/map.go`):
     - Define 18+ cities across 6 continents (Americas, Europe, Asia, Africa, Pacific, Antarctica)
     - Train/ship routes with travel cost in actions and ticket resources
     - Complex travel system distinct from Arkham's 4-neighborhood adjacency
  
  3. **Mystery Deck System** (`serverengine/eldritchhorror/model/mystery.go`):
     - Multi-step objectives requiring worldwide coordination (e.g., "Solve clues in 3 different continents")
     - 3 mysteries must be solved to win (distinct from Arkham's clue-per-investigator threshold)
     - Mystery progression tracked per scenario
  
  4. **Ancient One Mechanics** (`serverengine/eldritchhorror/model/ancient_one.go`):
     - Active antagonist with unique abilities, attack patterns, and awakening conditions
     - Ancient One combat phase when awakened (distinct from Arkham's doom-only loss)
     - Per-Ancient-One ruleset (e.g., Azathoth vs. Cthulhu have different mechanics)
  
  5. **Monster Surge System** (`serverengine/eldritchhorror/rules/monsters.go`):
     - Global monster spawning across cities (not localized neighborhoods)
     - Monster movement phase between player turns
     - Combat resolution system
  
  6. **Regional Encounter Decks** (`serverengine/eldritchhorror/content/encounters/`):
     - Unique encounter decks per region (Americas, Europe, Asia, Africa, Pacific, General)
     - 50-100 encounter cards per deck
     - Region-specific narrative and mechanical outcomes
  
  7. **Resource Economy**:
     - Same Health/Sanity bounds as Arkham (1-10) but different acquisition mechanics
     - Focus tokens (unique to Eldritch)
     - Improvement tokens (persistent character upgrades)
  
  8. **Wire Module Engine** (`serverengine/eldritchhorror/module.go:NewEngine()`):
     - Replace `UnimplementedEngine` with functional engine
     - Inject Eldritch-specific adapters, validators, content loader
     - Override game loop to include monster phase between player turns
  
  9. **Content Pack** (`serverengine/eldritchhorror/content/`):
     - 3-5 Ancient Ones (Azathoth, Cthulhu, Shub-Niggurath, Yog-Sothoth, Nyarlathotep) with unique mechanics
     - 9-12 mystery templates per Ancient One
     - Regional encounter decks (200+ cards total)
     - Mythos event deck (100+ cards)
     - Investigator roster with starting cities
  
  10. **Testing**:
      - Integration tests: `BOSTONFEAR_GAME=eldritchhorror go test ./serverengine/eldritchhorror/...`
      - Verify global travel, mystery progression, Ancient One awakening
      - Verify win condition (solve 3 mysteries) and lose conditions (Ancient One defeats all OR doom threshold OR investigator count drops below minimum)
      - Verify no Arkham-specific mechanics (4-location map, clue-per-investigator win) appear

- **Dependencies**: Arkham Horror implementation complete (✅); Elder Sign started or complete (establishes multi-module testing patterns).

- **Effort**: 8-10 weeks (single developer; longer than Elder Sign due to global map complexity)

- **Validation**:
  - `BOSTONFEAR_GAME=eldritchhorror go run . server` starts Eldritch Horror with global map
  - 4 players travel between cities, progress mysteries across continents, and defeat an Ancient One
  - No Arkham-specific mechanics appear
  - Tests pass with >75% coverage

---

## Gap 3: Final Hour Game Module — Scaffolded, Rules Not Implemented

- **Intended Behavior**: Server should support Final Hour gameplay with real-time simultaneous action programming, countdown token mechanics, priority bidding system, and time-sensitive objectives. Configuration via `BOSTONFEAR_GAME=finalhour` should start a Final Hour server.

- **Current State**: Module scaffolding exists in `serverengine/finalhour/` with same subdirectory structure. `module.go:NewEngine()` returns `UnimplementedEngine` which always fails `Start()` with error "finalhour engine not implemented". No real-time action handling, no countdown mechanics, no priority system.

- **Blocked Goal**: ROADMAP.md Goal 25 (multi-game-family support) marked ⚠️ Partial. Phase 4 implementation documented but not executed.

- **Implementation Path**:
  1. **Define Final Hour Action Types** (`serverengine/finalhour/actions/`):
     - `PlaceInvestigator` (move to room during planning phase)
     - `ResolveAction` (execute action after priority resolution)
     - `BidPriority` (simultaneous priority bidding)
     - `SpendFocus` (spend focus tokens for action efficiency)
  
  2. **Real-Time Action Programming** (`serverengine/finalhour/phases/planning.go`):
     - All players act simultaneously within time windows (not sequential turns)
     - 60-second action planning phase per round
     - Action buffer collects all player submissions before resolution
  
  3. **Countdown Token Mechanics** (`serverengine/finalhour/rules/countdown.go`):
     - Countdown represents time until Ancient One victory (not doom)
     - Decrements at end of each round (not on failed dice rolls)
     - Reaching 0 = automatic loss
  
  4. **Priority Bidding System** (`serverengine/finalhour/rules/priority.go`):
     - Players bid priority values (1-10) simultaneously
     - Highest priority acts first when multiple investigators target same space/card
     - Conflict resolution engine applies priority order
  
  5. **Objective Progression** (`serverengine/finalhour/model/objective.go`):
     - Time-sensitive objectives with hard deadlines (in countdown tokens)
     - Multiple concurrent objectives
     - Failure consequences (countdown acceleration, resource loss)
  
  6. **Resource Economy**:
     - Focus tokens (0-5) — primary resource for action efficiency
     - Health/Sanity (simplified from Arkham; 1-8 bounds)
     - No clues or items
  
  7. **Wire Module Engine** (`serverengine/finalhour/module.go:NewEngine()`):
     - Replace turn-based loop with phase-based simultaneous action collection
     - Implement time window enforcement (e.g., 60-second planning deadline)
     - Override state broadcast to include real-time countdown and planning state
  
  8. **Content Pack** (`serverengine/finalhour/content/`):
     - 3-5 Ancient One scenarios with Final Hour-specific mechanics
     - 15-20 objective card templates per scenario
     - Omen card deck (real-time event triggers)
     - Investigator roster (simplified abilities focused on priority and focus)
  
  9. **Testing**:
      - Integration tests: `BOSTONFEAR_GAME=finalhour go test ./serverengine/finalhour/...`
      - Verify simultaneous action submission and priority resolution
      - Verify win condition (complete objectives before countdown=0) and lose condition (countdown=0 OR all investigators defeated)
      - Test conflict resolution when 2+ players act on same target
      - Verify time window enforcement (actions submitted after deadline are rejected)

- **Dependencies**: Arkham Horror implementation complete (✅); Phases 2 and 3 started (establishes testing patterns for multi-module architecture).

- **Effort**: 6-8 weeks (single developer; complex due to real-time coordination requirements)

- **Validation**:
  - `BOSTONFEAR_GAME=finalhour go run . server` starts Final Hour with real-time mechanics
  - 4 players simultaneously submit actions within time window and observe priority-based resolution
  - Countdown decrements correctly; game ends at countdown=0 or objectives completed
  - Tests pass with >75% coverage

---

## Gap 4: Client Resolution Documentation Mismatch

- **Intended Behavior**: Documentation should accurately describe the logical resolution used by the Ebitengine client for coordinate calculations, UI layout, and rendering.

- **Current State**: `README.md:200` states "Logical 1280×720 resolution"; `docs/CLIENT_SPEC.md` repeats this claim. Actual implementation uses 800×600 in `client/ebiten/app/game.go:30-33`:
  ```go
  const (
      screenWidth  = 800
      screenHeight = 600
  )
  ```
  All location rectangles (lines 35-82), action grid positions (lines 84-160), HUD panels (lines 162-630), and mobile safe-area insets are calibrated for 800×600.

- **Blocked Goal**: ROADMAP.md Goal 23 (multi-resolution support) marked ⚠️ Partial due to documentation-vs-implementation mismatch.

- **Implementation Path (Option A — Recommended)**:
  1. Update `README.md:200` from "Logical 1280×720 resolution" to "Logical 800×600 resolution"
  2. Update `docs/CLIENT_SPEC.md` sections referencing 1280×720 to 800×600
  3. Update `client/ebiten/app/doc.go:20` package documentation comment to match
  4. Verify all UI coordinate calculations remain consistent with 800×600 baseline
  5. No code changes required

- **Implementation Path (Option B — Not Recommended)**:
  1. Change `screenWidth` → 1280, `screenHeight` → 720 in `client/ebiten/app/game.go:30-33`
  2. Recalculate all location rectangle coordinates (lines 35-82) for 1280×720 canvas
  3. Recalculate action grid positions (lines 84-160)
  4. Recalculate HUD panel layouts (lines 162-630)
  5. Re-test mobile safe-area insets and touch action hit-boxes
  6. Run display tests: `DISPLAY=:99 go test -race -tags=requires_display ./client/ebiten/app/... ./client/ebiten/render/...`
  7. Regression test all rendering paths

- **Dependencies**: None.

- **Effort**: 30 minutes (Option A) or 4-6 hours (Option B)

- **Validation**:
  - **Option A**: README.md and CLIENT_SPEC.md accurately describe 800×600; no code changes; no test failures
  - **Option B**: Code implements 1280×720; all location/action/HUD coordinates correct; all tests passing

- **Recommendation**: Option A (update documentation to match implementation). The 800×600 resolution is functional and well-tested. Changing to 1280×720 provides no functional benefit and risks introducing coordinate calculation bugs.

---

## Summary

**Total Gaps**: 4 (3 placeholder modules + 1 documentation mismatch)

**Severity Distribution**:
- Critical: 0
- High: 0
- Medium: 4
- Low: 0

**Implementation Priority**:
1. **Gap 4** (Documentation mismatch) — 30 minutes; unblocks specification accuracy
2. **Gap 1** (Elder Sign module) — 4-6 weeks; demonstrates modular architecture in production
3. **Gap 2** (Eldritch Horror module) — 8-10 weeks; adds global-scale gameplay
4. **Gap 3** (Final Hour module) — 6-8 weeks; introduces real-time mechanics

**Key Observations**:
- All gaps are **intentional architectural scaffolding** documented in ROADMAP.md
- No unexpected stubs, dead code, or missing functionality discovered
- Arkham Horror implementation is complete and production-ready
- Modular architecture is sound and ready for Phase 2-4 implementations

**Next Steps**: Fix documentation mismatch (Gap 4) immediately, then proceed with Elder Sign implementation (Gap 1) per ROADMAP.md Phase 2 roadmap.
