# Implementation Gaps — Detailed Roadmap — 2026-05-17

---

## Gap 1: Missing ROADMAP.md File

**Severity**: HIGH

**Intended Behavior**: Repository README.md explicitly references `ROADMAP.md` as the authoritative source for development phases, module implementation timelines, and planned features. ADR 003 describes phased Elder Sign (4-6 weeks), Eldritch Horror (8-10 weeks), and Final Hour implementation. No centralized roadmap document exists.

**Current State**: File is completely absent. References in README.md (lines 13, 288) point to a missing resource.

**Blocked Goal**: Project planning and contributor visibility into module implementation sequence and timeline.

**Implementation Path**:
1. Create `ROADMAP.md` at repository root.
2. Document phased implementation:
   - **Phase 1 (Complete)**: Arkham Horror 3rd Edition fully playable; 5 core mechanics implemented; 1-6 player support with late-join.
   - **Phase 2 (Planned)**: Elder Sign module (4-6 weeks). Dice tower mechanic, unique encounter system, scenario templates.
   - **Phase 3 (Planned)**: Eldritch Horror module (8-10 weeks). Global map, monster encounters, Ancient One system.
   - **Phase 4 (Planned)**: Final Hour module (6-8 weeks). Real-time mechanics, countdown tokens, objective progressions.
   - **Phase 5 (Future)**: Content expansion packs, alternate scenario sets, investigator balancing.
3. For each phase, list:
   - Blocking dependencies (e.g., Phase 2 depends on module architecture, which is complete)
   - Estimated team effort (weeks)
   - Key deliverables (rules/, adapters/, scenarios/, model/ per module)
   - Integration points with existing codebase
4. Add tracking for Arkham content roadmap (new investigator cards, scenarios, encounter types).

**Dependencies**: None. Module architecture is complete and supports immediate implementation.

**Effort**: 2-3 hours to draft phased timeline and expand with implementation checklists.

**Validation**: ROADMAP.md exists and is up-to-date; all hyperlinked references in README.md resolve.

---

## Gap 2: Client Rendering Resolution Mismatch (Design Document ↔ Implementation)

**Severity**: HIGH

**Intended Behavior**: README.md section "Multi-Resolution Support" and [docs/CLIENT_SPEC.md](docs/CLIENT_SPEC.md) state "Logical 1280×720 resolution scaled to any display."

**Current State**: 
- [client/ebiten/app/game.go:30-33](client/ebiten/app/game.go#L30-L33) defines `screenWidth = 800`, `screenHeight = 600`
- [Game.Layout()](client/ebiten/app/game.go#L242-243) returns (800, 600)
- [cmd/desktop.go:39](cmd/desktop.go#L39) and [cmd/web_wasm.go:35](cmd/web_wasm.go#L35) both call `ebiten.SetWindowSize(800, 600)`
- All UI coordinate calculations (location rects, action grid, panels) hard-coded to 800×600

**Blocked Goal**: Accurate specification of client display contract; developers and users cannot rely on documented resolution.

**Implementation Path** (Option A: Update documentation — PREFERRED):
1. Update [README.md](README.md#L200) line 200 from "1280×720" to "800×600 logical"
2. Update [docs/CLIENT_SPEC.md](docs/CLIENT_SPEC.md) to accurately reflect 800×600 as minimum logical resolution
3. Update [client/ebiten/app/doc.go](client/ebiten/app/doc.go) line 20 documentation comment from "1280x720 logical" to "800x600 logical"
4. Verify all scaling/inset math is correctly documented

**Implementation Path** (Option B: Update code to match documentation — NOT RECOMMENDED):
1. Change screenWidth → 1280, screenHeight → 720
2. Recalculate all location rectangles, panels, and action grid positions (line 35-630 in game.go)
3. Re-test UI scaling on mobile with safe-area insets
4. Regression test to ensure action hit-boxes remain accessible

**Dependencies**: None (Option A); Option B requires comprehensive UI layout review.

**Effort**: 30 minutes (Option A); 4-6 hours (Option B design verification).

**Validation**: README.md and CLIENT_SPEC.md accurately describe 800×600; or code upgraded to 1280×720 with all tests passing.

---

## Gap 3: Arkham Horror Scenarios Package Incorrectly Marked as Scaffolded

**Severity**: HIGH

**Intended Behavior**: Documentation should clarify ownership of scenario content and rules.

**Current State**: 
- [serverengine/arkhamhorror/scenarios/doc.go:7](serverengine/arkhamhorror/scenarios/doc.go#L7) declares "NOTE: This package is a scaffold. Implementation is deferred."
- However, scenario loading works correctly: server loads scenarios from `serverengine/arkhamhorror/content/nightglass/scenarios/`
- The scenarios/ package is doc-only with no actual code/types.

**Blocked Goal**: Clear ownership model; contributors are confused about where scenario definitions belong.

**Implementation Path**:
1. **Approach A (Recommended)**: Update [serverengine/arkhamhorror/scenarios/doc.go](serverengine/arkhamhorror/scenarios/doc.go) to clarify:
   ```
   // Package scenarios defines Arkham Horror scenario content traits and structure.
   // As of 2026-05-17, scenario content is defined in serverengine/arkhamhorror/content/nightglass/
   // and loaded at server startup. This package is reserved for future game-family-specific
   // scenario type systems (e.g., Elder Sign scenario types differ from AH3e Agenda/Act structure).
   // For current Arkham scenario definitions, see serverengine/arkhamhorror/content/.
   ```
2. **Approach B**: Move scenario-type definitions (if any exist in future) into serverengine/arkhamhorror/content/scenarios/ subdirectory for clarity.

**Dependencies**: None (documentation only).

**Effort**: 20 minutes.

**Validation**: [serverengine/arkhamhorror/scenarios/doc.go](serverengine/arkhamhorror/scenarios/doc.go) accurately describes current and future use; developers understand content ownership.

---

## Gap 4: Three Game-Family Modules Completely Scaffolded (No Game-Specific Rules)

**Severity**: MEDIUM

**Intended Behavior**: `eldersign`, `eldritchhorror`, and `finalhour` modules should implement game-family-specific rules (actions, dice mechanics, resources, scenarios) by extending the shared serverengine.GameServer.

**Current State**: 
- [serverengine/eldersign/module.go:43](serverengine/eldersign/module.go#L43), [eldritchhorror/module.go:44](serverengine/eldritchhorror/module.go#L44), [finalhour/module.go:44](serverengine/finalhour/module.go#L44)
- Each NewEngine() returns `&Engine{GameServer: serverengine.NewGameServer()}` with NO overrides
- Running `BOSTONFEAR_GAME=eldersign go run . server` launches Arkham Horror rules, not Elder Sign rules
- All three modules are feature-identical copies of each other

**Blocked Goal**: Playable Elder Sign, Eldritch Horror, and Final Hour implementations; true modular game architecture.

**Implementation Path** (per module):
1. **Define Game-Family Rules** in `serverengine/{module}/rules/`:
   - Action type definitions (Elder Sign has "spell tower placement", not "move/gather/investigate")
   - Dice mechanics (Elder Sign: special die with tokens, not 3-sided success/blank/tentacle)
   - Resource economy (different min/max for health/sanity, unique tokens if applicable)
   - Victory/defeat conditions (different from Arkham's doom-based loss)
   
2. **Implement Rules in Adapters** (`serverengine/{module}/adapters/`):
   - Extend BroadcastPayloadAdapter to shape game-specific state messages
   - Implement DispatchAction override if family uses different action resolution logic
   - Shape dice results for family-specific die types

3. **Define Scenario Content** (`serverengine/{module}/content/`):
   - Load scenario YAML/JSON with family-specific victory/defeat conditions
   - Register 3-5 playable scenarios for each family
   - Include investigator definitions (may differ per family)

4. **Implement Model Types** (`serverengine/{module}/model/`):
   - Create Game-specific game state struct extending base GameState
   - Define resource types, action types, location graph per family

5. **Wire Module Engine** to own variants:
   - Override NewEngine() to inject custom adapters, load family-specific content
   - Override Start() to initialize family-specific game rules if needed

6. **Testing**: Create module-specific integration tests verifying actions, dice, and win/lose conditions.

**Dependencies**: 
- Module architecture is complete (no blocking changes needed)
- ROADMAP.md should document phased timeline
- Arkham Horror implementation serves as reference pattern

**Effort**: 
- **Elder Sign**: 4-6 weeks (simpler than Arkham; fewer action types)
- **Eldritch Horror**: 8-10 weeks (global map adds complexity)
- **Final Hour**: 6-8 weeks (real-time mechanics require different turn structure)

**Validation**: 
- `BOSTONFEAR_GAME=eldersign go run . server` + 3 players → plays Elder Sign rules (not Arkham)
- Each family's dice, actions, scenarios, and win conditions match official rules
- Integration tests pass for each family
- No code duplication between families in rules/ or adapters/

---

## Gap 5: Six Scaffolded Subpackages Under Non-Arkham Modules (rules, adapters, scenarios, model)

**Severity**: MEDIUM

**Intended Behavior**: Each game-family module should define complete rules/, adapters/, scenarios/, and model/ subpackages with game-family-specific implementations.

**Current State**: 
- [serverengine/eldersign/rules/doc.go:4](serverengine/eldersign/rules/doc.go#L4) "NOTE: This package is a scaffold. Implementation is deferred."
- [serverengine/eldersign/adapters/doc.go:4-5](serverengine/eldersign/adapters/doc.go#L4-L5) "NOTE: This package is a scaffold."
- [serverengine/eldersign/scenarios/doc.go:3](serverengine/eldersign/scenarios/doc.go#L3) "NOTE: This package is a scaffold."
- Same pattern for eldritchhorror/ and finalhour/ (18 doc-only packages total)
- Zero implementation in any of these directories

**Blocked Goal**: Game-family-specific rule implementations; content loading; state shaping.

**Implementation Path** (performed in each module in parallel):
1. **rules/**: Implement action types, dice rules, location/zone system, resource constraints
2. **adapters/**: Implement BroadcastPayloadAdapter and game-specific message shaping
3. **scenarios/**: Implement scenario loader and scenario type definitions
4. **model/**: Define game-state extensions and action result types

See Gap 4 for detailed implementation path per module.

**Dependencies**: Gap 4 (module rules architectecture) must be started first.

**Effort**: Inherits effort from Gap 4 (4-6 weeks per module).

**Validation**: All 18 packages have substantive implementation (not doc-only); modules pass integration tests.

---

## Gap 6: Metrics Collection Instrumentation Missing

**Severity**: LOW

**Intended Behavior**: [serverengine/metrics.go](serverengine/metrics.go) defines GameMetrics with ActionTypeCounters (map[string]int64), DoomHistogram (map[int]int64), LatencyPercentiles (map[string]float64) that should be updated during action processing.

**Current State**: 
- Metrics are exported by HTTP /metrics endpoint (Prometheus format)
- Counters are initialized to zero and never incremented
- Latency percentiles not computed
- Doom histogram not updated

**Blocked Goal**: Operational observability into action distribution and doom progression; latency SLO tracking.

**Implementation Path**:
1. **Action Counter Instrumentation**:
   - In [serverengine/game_server.go:processActionCore()](serverengine/game_server.go#L538), after action type validation, increment `gs.metrics.ActionTypeCounters[actionType]++`
   - Thread-safe update (use mutex or atomic if counter is uint64)

2. **Doom Histogram**:
   - In [serverengine/mythos.go:RunMythosPhase()](serverengine/arkhamhorror/phases/mythos.go) or game_server doom increment, track `gs.metrics.DoomHistogram[newDoomValue]++`

3. **Latency Percentiles** (optional, more complex):
   - Wrap action dispatch with time.Now() start/stop
   - Maintain rolling percentile histogram (e.g., using golang.org/x/exp/slices or custom quantile tracker)
   - Compute p50, p95, p99 on-demand or via background goroutine

**Dependencies**: None.

**Effort**: 3-4 hours for action/doom counters + percentile infrastructure.

**Validation**: Run game with logging, verify /metrics endpoint includes non-zero ActionTypeCounters and DoomHistogram; latency percentiles present.

---

## Gap 7: Duplicate BroadcastPayloadAdapter Interface Definition

**Severity**: LOW

**Intended Behavior**: BroadcastPayloadAdapter contract should be defined once in serverengine and implemented by game modules.

**Current State**: 
- [serverengine/interfaces.go:24-27](serverengine/interfaces.go#L24-L27) defines BroadcastPayloadAdapter interface (exported)
- [serverengine/arkhamhorror/adapters/broadcast.go:7](serverengine/arkhamhorror/adapters/broadcast.go#L7) redefines identical BroadcastPayloadAdapter interface (unexported)
- Arkham module creates arkhamhorror.adapters.BroadcastPayloadAdapter and casts to serverengine.BroadcastPayloadAdapter (works because signatures match)

**Blocked Goal**: Clarity of contract ownership; code duplication.

**Implementation Path**:
1. Delete [serverengine/arkhamhorror/adapters/broadcast.go:7](serverengine/arkhamhorror/adapters/broadcast.go#L7) interface definition
2. Update [serverengine/arkhamhorror/adapters/adapter.go](serverengine/arkhamhorror/adapters/adapter.go) to reference serverengine.BroadcastPayloadAdapter directly:
   ```go
   // NewBroadcastAdapter creates a broadcast adapter for Arkham Horror.
   func NewBroadcastAdapter() serverengine.BroadcastPayloadAdapter {
       return &arkhamBroadcastAdapter{}
   }
   ```
3. Run `go test ./...` to verify no regressions

**Dependencies**: None.

**Effort**: 20 minutes.

**Validation**: Tests pass; no duplicate interface definitions; adapter contract is uniform across all modules.

---

## Gap 8: UnimplementedEngine Silent Method Failures

**Severity**: LOW

**Intended Behavior**: UnimplementedEngine methods should consistently signal "not yet implemented" state via error returns or clear no-op behavior.

**Current State**: 
- [serverengine/common/runtime/unimplemented_engine.go:27-32](serverengine/common/runtime/unimplemented_engine.go#L27-L32) Start() and HandleConnection() return errors ("game not implemented")
- [serverengine/common/runtime/unimplemented_engine.go:39-65](serverengine/common/runtime/unimplemented_engine.go#L39-L65) SetAllowedOrigins(), health/metrics methods succeed silently (return empty maps/snapshots)
- Inconsistency: some methods fail loudly, others fail silently

**Blocked Goal**: Clarity on which methods are safe to call on placeholder engines.

**Implementation Path**:
1. Verify behavior is intentional in design (SetAllowedOrigins must succeed to satisfy Engine interface even though Start always fails)
2. Add explanatory comment in [serverengine/common/runtime/unimplemented_engine.go](serverengine/common/runtime/unimplemented_engine.go) doc.go or method doc:
   ```go
   // SetAllowedOrigins stores allowed origins but has no filtering semantics
   // since Start() always fails. This method exists to satisfy the Engine interface
   // but does not alter the fact that the engine is non-functional.
   func (e *UnimplementedEngine) SetAllowedOrigins(origins []string) {
   ```

**Dependencies**: None (documentation only).

**Effort**: 10 minutes.

**Validation**: Comment explains intent; code behavior is documented.

---

## Gap 9: Client Resolution Update — Summary

**Summary**: The 800×600 vs. 1280×720 discrepancy (Gap 2) should be resolved by updating documentation to match implementation (Option A), as the current implementation is stable and tested. Only pursue Option B (code upgrade) if a design decision mandates higher resolution for improved usability.

---

## Implementation Priority & Sequencing

**Week 1** (documentation catching-up):
1. Create ROADMAP.md (2-3 hours) → unblocks contributor visibility
2. Update client resolution docs (30 min) → fixes specification accuracy
3. Clarify Arkham scenarios ownership (20 min) → removes confusion

**Week 2-3** (observability):
1. Instrument metrics collection (3-4 hours) → improves operational visibility

**Week 4+** (module development, in parallel):
1. Implement Elder Sign (4-6 weeks)
2. Implement Eldritch Horror (8-10 weeks, can start after Elder Sign rules pattern is established)
3. Implement Final Hour (6-8 weeks, parallel with Eldritch Horror)

**Ongoing** (low-priority cleanup):
1. Remove duplicate BroadcastPayloadAdapter (20 min, low risk)
2. Document UnimplementedEngine behavior (10 min)

---

## Estimation Summary

| Gap | Effort | Risk | ROI |
|---|---|---|---|
| ROADMAP.md | 2-3 hours | Low | High (unblocks planning) |
| Client resolution docs | 30 min | Low | High (fixes spec accuracy) |
| Arkham scenarios docs | 20 min | Low | Medium (clarifies ownership) |
| Metrics instrumentation | 3-4 hours | Low | Medium (observability) |
| **Elder Sign module** | **4-6 weeks** | **Medium** | **High** |
| **Eldritch Horror module** | **8-10 weeks** | **Medium** | **High** |
| **Final Hour module** | **6-8 weeks** | **Medium** | **High** |
| Duplicate adapter cleanup | 20 min | Low | Low (code quality) |
| UnimplementedEngine docs | 10 min | Low | Low (clarity) |

---

## Success Criteria

✅ All HIGH-severity gaps remediated (ROADMAP.md, client resolution, scenarios clarity)  
✅ Documentation accuracy verified against implementation  
✅ ROADMAP.md exists and is referenced successfully  
✅ Elder Sign, Eldritch Horror, Final Hour modules each have playable implementations  
✅ Metrics collection working for action counters and doom histogram  
✅ Tests pass for all modules with no regressions  
✅ New contrib can follow ROADMAP.md to implement a game family end-to-end
