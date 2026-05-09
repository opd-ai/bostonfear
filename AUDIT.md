# IMPLEMENTATION GAP AUDIT — 2026-05-09

## Project Architecture Overview
BostonFear is positioned as a rules-driven multiplayer Arkham Horror engine with a Go WebSocket server, Go/Ebitengine clients (desktop, WASM, mobile scaffolding), and module-based runtime selection.

Stated architecture (from README and migration docs):
- serverengine/common owns cross-game contracts/primitives.
- serverengine/arkhamhorror owns Arkham-specific rules and runtime behavior.
- serverengine is a compatibility facade while migration completes.
- transport/ws owns HTTP + WebSocket adaptation.
- monitoring owns health/metrics handlers.

Phase-0/Phase-2 evidence used:
- README and docs/MODULE_MIGRATION_MAP.md, docs/MODULE_MIGRATION_COMPLETION.md.
- go.mod (module: github.com/opd-ai/bostonfear, go 1.24.1).
- go list ./... (45 packages).
- go-stats-generator metrics:
  - LOC: 6532
  - Functions: 216, Methods: 398
  - Structs: 159, Interfaces: 18
  - Packages: 30, Files: 127
  - Documentation overall coverage: 81.56%
- Baseline quality gates:
  - go build ./... : clean (no output)
  - go vet ./... : clean (no output)

External brief research (<=10 min):
- GitHub repo has no open issues, milestones, projects, or pull requests.
- No external tracker was discoverable to disambiguate deferred items from completed items.

## Gap Summary
| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 4 | 0 | 1 | 2 | 1 |
| Dead Code | 1 | 0 | 0 | 1 | 0 |
| Partially Wired | 3 | 0 | 1 | 1 | 1 |
| Interface Gaps | 1 | 0 | 0 | 1 | 0 |
| Dependency Gaps | 1 | 0 | 0 | 0 | 1 |

## Implementation Completeness by Package
(Exported counts derived from go-stats function inventory; completeness is implementation-oriented, not behavior-completeness.)

| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|-------------------:|------------:|------:|-----:|---------:|
| serverengine | 31 | 31 | 0 | 0 | 100% |
| cmd | 7 | 6 | 1 | 0 | 85.7% |
| rules | 5 | 4 | 0 | 1 | 80.0% |
| runtime | 17 | 17 | 0 | 0 | 100% |
| actions | 1 | 1 | 0 | 0 | 100% |
| content | 4 | 4 | 0 | 0 | 100% |
| eldersign | 4 | 0 | 4 | 0 | 0% |
| eldritchhorror | 4 | 0 | 4 | 0 | 0% |
| finalhour | 4 | 0 | 4 | 0 | 0% |
| scenarios | 0 | 0 | 1 package scaffold | 0 | 0% |
| common/messaging | 0 | 0 | 1 package scaffold | 0 | 0% |
| common/session | 0 | 0 | 1 package scaffold | 0 | 0% |
| common/state | 0 | 0 | 1 package scaffold | 0 | 0% |
| common/validation | 0 | 0 | 1 package scaffold | 0 | 0% |
| common/observability | 0 | 0 | 1 package scaffold | 0 | 0% |
| common/monitoring | 0 | 0 | 1 package scaffold | 0 | 0% |

## Findings
### HIGH
- [ ] Multi-family runtime path remains non-runnable despite module registration wiring — cmd/server.go:60, cmd/server.go:135, serverengine/eldersign/module.go:44, serverengine/eldritchhorror/module.go:46, serverengine/finalhour/module.go:45 — non-Arkham modules are selectable (when experimental is enabled) but always fail startup via unimplemented engines, blocking the stated module-based runtime selection vision.
  - Remediation: Either (A) keep only arkhamhorror registerable in production code paths until runtimes exist, or (B) implement minimum runnable engine boot paths for each registered family (Start/HandleConnection/scenario loading).
  - Validation: `go run . server --game arkhamhorror`, `BOSTONFEAR_EXPERIMENTAL=1 go run . server --game eldersign`, module-selection integration tests.
  - Dependency: Align with module migration documentation updates (Finding 4).

### MEDIUM
- [ ] Broadcast payload abstraction is declared complete but not wired into execution — serverengine/arkhamhorror/adapters/broadcast.go:7, docs/MODULE_MIGRATION_COMPLETION.md:40, docs/MODULE_MIGRATION_MAP.md:30, serverengine/broadcast.go:51 — BroadcastPayloadAdapter/ActionResultPayload/DiceResultPayload exist, but shaping still happens in facade broadcast flow and no adapter implementation is invoked.
  - Remediation: Add concrete adapter implementation in arkhamhorror/adapters and inject it into serverengine broadcast path (or explicitly mark S6 as partial in docs).
  - Validation: unit tests asserting adapter invocation and wire-shape parity; `go test -race ./serverengine/...`.
  - Dependency: None.

- [ ] Dead duplicate movement function remains in maintained path — serverengine/arkhamhorror/rules/movement_fixed.go:12 — IsAdjacentFixed has no call sites and duplicates movement.go logic.
  - Remediation: remove movement_fixed.go or route all adjacency checks through a single exported helper with tests.
  - Validation: `go test ./serverengine/arkhamhorror/rules ./serverengine/...` and grep for removed symbol references.
  - Dependency: None.

- [ ] Reserved common packages are scaffold-only and not integrated — serverengine/common/messaging/doc.go:1, serverengine/common/session/doc.go:1, serverengine/common/state/doc.go:1, serverengine/common/validation/doc.go:1, serverengine/common/observability/doc.go:1, serverengine/common/monitoring/doc.go:1 — all six packages are doc-only scaffolds with no runtime imports, indicating unfinished modularization slices.
  - Remediation: either implement minimal shared primitives and wire at least one consuming path, or collapse/remove packages until needed.
  - Validation: import graph check (`go list -deps ./...`), package-level tests for extracted primitives.
  - Dependency: Depends on explicit backlog ownership (roadmap/tracker, Finding 6).

- [ ] Arkham scenarios package is structurally present but functionally empty — serverengine/arkhamhorror/scenarios/doc.go:1 — package claims scenario templates but contains no types/functions; scenario logic currently resides in content.
  - Remediation: consolidate documentation to content package or implement scenario types/loader APIs in scenarios and migrate call sites.
  - Validation: scenario package tests + startup scenario resolution tests.
  - Dependency: Migration slice ownership (S5/S2 boundaries).

### LOW
- [ ] Root command exposes web subcommand on non-WASM builds even though it always errors — cmd/root.go:31, cmd/web_nowasm.go:11 — user-visible command registration is partial/non-executable on primary server/desktop targets.
  - Remediation: hide command for non-wasm builds or move to platform-specific command registration.
  - Validation: `go run . --help` and platform-specific command tests.
  - Dependency: None.

- [ ] Documentation drift: packages marked “scaffold/deferred” despite implemented code paths — serverengine/arkhamhorror/actions/doc.go:7 (implemented dispatcher in perform.go), serverengine/arkhamhorror/rules/doc.go:8 (implemented dice/movement), serverengine/arkhamhorror/model/doc.go:10 (implemented clamping/resource helpers), docs/MODULE_MIGRATION_MAP.md:20 (slices marked completed).
  - Remediation: update package docs to current ownership/completeness state; preserve deferred notes only for truly unimplemented surfaces.
  - Validation: doc review + CI doc-link check.
  - Dependency: None.

### DEPENDENCY/REFERENCE
- [ ] ROADMAP.md is referenced broadly but missing from repository root — README.md:13, README.md:275, docs/RULES.md:211, serverengine/arkhamhorror/actions/doc.go:8 (and other references) — planning references cannot be resolved, weakening implementation traceability.
  - Remediation: restore ROADMAP.md or update all references to an existing planning source (for example docs/MODULE_MIGRATION_MAP.md).
  - Validation: link-check script over markdown/go comments; `rg -n "ROADMAP\.md"` should only return valid targets.
  - Dependency: None.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|-----------------|
| Placeholder sprite/atlas fallbacks in render package | Intentional alpha behavior, documented in README/client docs; not a mechanics implementation gap. |
| Empty build/vet outputs imply hidden dead code | `go build` and `go vet` both pass cleanly; no compiler/vet dead-code warning to support such a finding. |
| Arkham core mechanics missing | Core mechanics are implemented and covered by active serverengine tests; no stub-level evidence on critical Arkham path. |
| Unimplemented experimental engines as CRITICAL | They are gated (`BOSTONFEAR_EXPERIMENTAL`) and explicitly documented as placeholders; classified HIGH for architectural gap, not CRITICAL for current Arkham objective. |

## Notes on Methodology
False-positive prevention checks were applied to all findings:
1. Verified against documented intent before labeling incomplete.
2. Confirmed intentional minimalism where applicable.
3. Considered external caller possibility for exported symbols.
4. Distinguished tracked/deferred notes from undiscovered gaps.
5. Corroborated dead-code claims with concrete no-reference evidence.
