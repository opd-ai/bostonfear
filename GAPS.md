# Implementation Gaps — 2026-05-09

## 1) Experimental Game Families Are Selectable But Not Runnable
- Status (2026-05-09): [x] Completed decision: hidden-until-ready policy documented and enforced in server startup path.
- Intended Behavior: Module-based runtime selection should allow registered families to initialize engines through a consistent contracts.Engine path.
- Current State: Registration and selection are wired, but startup rejects non-Arkham modules and module NewEngine functions return unimplemented placeholders.
  - Evidence: cmd/server.go:60, cmd/server.go:135, serverengine/eldersign/module.go:44, serverengine/eldritchhorror/module.go:46, serverengine/finalhour/module.go:45, serverengine/common/runtime/unimplemented_engine.go:27.
- Blocked Goal: “Module-based runtime selection” and game-family extensibility are operationally incomplete.
- Implementation Path:
  1. Decide product contract: hidden-until-ready vs minimally runnable modules.
  2. If hidden-until-ready: remove registration and CLI discoverability for non-runnable modules.
  3. If runnable: implement Start/HandleConnection/game-state bootstrap in each family module and provide scenario/content defaults.
  4. Add integration tests for `--game` resolution across registered modules.
- Dependencies: Documentation alignment (Gap 6), roadmap ownership clarity (Gap 7).
- Effort: large.

## 2) Broadcast Adapter Slice Is Defined But Not Executed
- Status (2026-05-09): [x] Completed: adapter is now wired for gameState/gameUpdate/diceResult payload shaping.
- Intended Behavior: S6 migration indicates broadcast payload shaping should be module-owned via adapters, reducing facade coupling.
- Current State: Broadcast adapter interface/types exist, but no implementation is used in runtime broadcast code.
  - Evidence: serverengine/arkhamhorror/adapters/broadcast.go:7, docs/MODULE_MIGRATION_COMPLETION.md:40, docs/MODULE_MIGRATION_MAP.md:30, serverengine/broadcast.go:51.
- Blocked Goal: Clear responsibility boundary between module-owned payload shaping and facade transport execution.
- Implementation Path:
  1. Create concrete adapter (for gameState, gameUpdate, diceResult).
  2. Inject adapter into GameServer construction (constructor wiring).
  3. Replace direct map/object shaping in broadcast path with adapter calls.
  4. Add parity tests to verify wire schema compatibility.
- Dependencies: None.
- Effort: medium.

## 3) Unreferenced Duplicate Movement Function
- Status (2026-05-09): [x] Already resolved in current tree (no movement_fixed.go and no IsAdjacentFixed references found).
- Intended Behavior: Single authoritative adjacency resolver for movement legality.
- Current State: Duplicate function exists with no call sites.
  - Evidence: serverengine/arkhamhorror/rules/movement_fixed.go:12, and only self-reference for IsAdjacentFixed.
- Blocked Goal: Increases maintenance burden and ambiguity for movement ownership.
- Implementation Path:
  1. Remove IsAdjacentFixed or replace IsAdjacent with aliasing strategy if transitional compatibility is required.
  2. Keep one canonical API and one test suite.
- Dependencies: None.
- Effort: small.

## 4) Common Cross-Engine Packages Remain Scaffold-Only
- Intended Behavior: serverengine/common should hold reusable session/state/validation/messaging/observability/monitoring primitives used by modules.
- Current State: Packages are doc-only placeholders and not imported by runtime paths.
  - Evidence: serverengine/common/messaging/doc.go:1, serverengine/common/session/doc.go:1, serverengine/common/state/doc.go:1, serverengine/common/validation/doc.go:1, serverengine/common/observability/doc.go:1, serverengine/common/monitoring/doc.go:1.
- Blocked Goal: Cross-game reusable architecture remains mostly aspirational.
- Implementation Path:
  1. Define one minimal shared primitive per package (for example, session token lifecycle in common/session).
  2. Migrate one live call path from serverengine facade to each new primitive incrementally.
  3. Add package-level tests and import graph checks to prevent regression.
- Dependencies: Prioritization by architecture owner.
- Effort: medium/large.

## 5) Arkham Scenarios Package Exists Without Runtime Surface
- Intended Behavior: Arkham scenarios package should expose scenario templates/types/loading APIs per its package contract.
- Current State: Package contains only doc.go; scenario behavior is carried in content package.
  - Evidence: serverengine/arkhamhorror/scenarios/doc.go:1.
- Blocked Goal: Package responsibility is unclear and migration boundaries are blurred.
- Implementation Path:
  1. Choose one owner: keep scenarios in content or implement scenarios package API.
  2. If implementing scenarios package: add scenario model, resolver, and startup integration.
  3. Update docs and migration map accordingly.
- Dependencies: S5/S2 ownership decision.
- Effort: medium.

## 6) Platform Command Wiring Is Partially Reachable
- Status (2026-05-09): [x] Already resolved in current tree (non-WASM web command is hidden with explicit unsupported-target error).
- Intended Behavior: CLI should present commands that are executable on the current target, or clearly hide unavailable commands.
- Current State: root registers web command universally; on non-wasm builds it is a guaranteed error path.
  - Evidence: cmd/root.go:31, cmd/web_nowasm.go:11.
- Blocked Goal: Predictable command UX across supported build targets.
- Implementation Path:
  1. Register NewWebCommand conditionally by build target or runtime capability check.
  2. Keep explicit help messaging for unsupported target combinations.
  3. Add command-surface tests for desktop/server builds.
- Dependencies: None.
- Effort: small.

## 7) Missing Roadmap Artifact Breaks Planning Traceability
- Status (2026-05-09): [x] Already resolved in current tree (ROADMAP.md exists at repository root).
- Intended Behavior: Architecture and deferred scaffolds should point to a resolvable roadmap/tracker.
- Current State: ROADMAP.md is referenced broadly but does not exist in repo root.
  - Evidence: README.md:13, README.md:275, docs/RULES.md:211, serverengine/arkhamhorror/actions/doc.go:8.
- Blocked Goal: Hard to validate whether deferred work is intentionally tracked or drifted.
- Implementation Path:
  1. Restore ROADMAP.md or redirect references to an existing source-of-truth document.
  2. Add automated link checking in CI for markdown + package docs.
- Dependencies: None.
- Effort: small.

## 8) Package Documentation Drift Against Completed Migration Slices
- Intended Behavior: Package-level docs should reflect current implementation status.
- Current State: Some Arkham package docs still describe scaffold/deferred states after implementation landed.
  - Evidence: serverengine/arkhamhorror/actions/doc.go:7, serverengine/arkhamhorror/rules/doc.go:8, serverengine/arkhamhorror/model/doc.go:10, docs/MODULE_MIGRATION_MAP.md:20.
- Blocked Goal: Increases onboarding friction and can cause incorrect refactor decisions.
- Implementation Path:
  1. Update package docs to separate completed functionality from remaining deferred items.
  2. Add doc review checklist item during migration completion updates.
- Dependencies: None.
- Effort: small.

## Verification Roadmap (Suggested Order)
1. Resolve Gap 7 first (restore roadmap reference integrity).
2. Resolve Gap 1 decision (hide vs implement non-Arkham modules).
3. Wire Gap 2 broadcast adapter path for S6 completion truth.
4. Execute Gap 4 incremental common-package extraction.
5. Clean up low-effort hygiene gaps (3, 6, 8).

## Global Validation Commands
- `go build ./...`
- `go vet ./...`
- `go test -race ./serverengine/... ./transport/ws/... ./cmd/...`
- `rg -n "ROADMAP\.md"`
- `go list -deps ./...`
