# IMPLEMENTATION GAP AUDIT — 2026-05-09

## Project Architecture Overview

BostonFear is a Go module (`github.com/opd-ai/bostonfear`, Go `1.24.1`) with a multiplayer Arkham Horror runtime and Go/Ebitengine clients. The intended architecture (from `README.md`, `config.toml`, and `docs/MODULE_MIGRATION_MAP.md`) is:

- `cmd`: Cobra/Viper command wiring and config ingestion.
- `transport/ws`: listener setup and WebSocket upgrade/dispatch.
- `monitoring` + `monitoringdata`: health and Prometheus metrics payloads.
- `protocol`: shared JSON wire contracts.
- `serverengine`: current runtime facade and active gameplay implementation.
- `serverengine/arkhamhorror/*`: intended long-term Arkham ownership boundary.
- `serverengine/common/*`: shared cross-game contracts/runtime primitives.
- `serverengine/{eldersign,eldritchhorror,finalhour}`: future game-family modules.

Audit evidence used:

- `go-stats-generator analyze . --skip-tests --format json --sections functions,documentation,packages,patterns,interfaces,structs,duplication`
- `go-stats-generator analyze . --skip-tests`
- `go list ./...`
- `go build ./...` (clean)
- `go vet ./...` (clean)
- Brief public GitHub check: no open issues, no open PRs, no milestones.

## Gap Summary

| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 2 | 0 | 1 | 1 | 0 |
| Dead Code | 1 | 0 | 0 | 1 | 0 |
| Partially Wired | 3 | 0 | 1 | 2 | 0 |
| Interface Gaps | 1 | 0 | 0 | 0 | 1 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Implementation Completeness by Package

Coverage is exported-surface completeness for this audit (`(Implemented / Exported Functions) * 100`).

| Package | Exported Functions | Implemented | Stubs | Dead | Coverage |
|---------|-------------------|-------------|-------|------|----------|
| `cmd` | 7 | 6 | 0 | 1 | 85.7% |
| `serverengine` | 21 | 21 | 0 | 0 | 100% |
| `serverengine/arkhamhorror` | 4 | 3 | 1 | 0 | 75.0% |
| `serverengine/common/runtime` | 17 | 16 | 1 | 0 | 94.1% |
| `serverengine/eldersign` | 4 | 3 | 1 | 0 | 75.0% |
| `serverengine/eldritchhorror` | 4 | 3 | 1 | 0 | 75.0% |
| `serverengine/finalhour` | 4 | 3 | 1 | 0 | 75.0% |
| `transport/ws` | 13 | 13 | 0 | 0 | 100% |
| `monitoring` | 3 | 3 | 0 | 0 | 100% |
| `protocol` | 0 | 0 | 0 | 0 | 100% |

## Findings

### CRITICAL

- [ ] No CRITICAL implementation gaps remained after false-positive filtering.

### HIGH

- [ ] Arkham ownership migration is still structurally planned but not implemented in executable paths — `docs/MODULE_MIGRATION_MAP.md:27`, `docs/MODULE_MIGRATION_MAP.md:45`, `serverengine/arkhamhorror/module.go:66`, `serverengine/arkhamhorror/actions/doc.go:7`, `serverengine/arkhamhorror/phases/doc.go:7`, `serverengine/arkhamhorror/scenarios/doc.go:7`. The migration map keeps slices `S1`-`S6` as `Planned`, and `arkhamhorror.Module.NewEngine()` still returns `serverengine.NewGameServer()` directly. This blocks the stated package-ownership architecture in `README.md`.
  - **Remediation:** Implement migration slice-by-slice (`S1` first), move live logic into `serverengine/arkhamhorror/*`, keep `serverengine` as facade only.
  - **Validation:** `go test -race ./serverengine ./serverengine/arkhamhorror/... && go vet ./... && go build ./...`.
  - **Dependencies:** Requires parity tests before each slice move.

- [ ] `scenario.default_id` is documented and configured but not consumed by runtime startup — `README.md:145`, `README.md:159`, `config.toml:41`, `cmd/server.go:68`, `serverengine/game_server.go:111`. Startup installs content then constructs `DefaultScenario` via `NewGameServer()` without reading `scenario.default_id`.
  - **Remediation:** Read `viper.GetString("scenario.default_id")`, resolve against `serverengine/arkhamhorror/content/nightglass/scenarios/index.yaml`, and pass resolved scenario into server creation.
  - **Validation:** Add tests for valid/missing/invalid configured scenario IDs and run `go test ./cmd ./serverengine/...`.
  - **Dependencies:** Best aligned with scenario migration slice (`S5`).

### MEDIUM

- [ ] `web.server` is an advertised config key but is not wired into WASM URL resolution — `README.md:147`, `config.toml:25`, `cmd/web_wasm.go:41`, `cmd/web_wasm.go:43`. Current code only uses `window.__serverURL` or `window.location`.
  - **Remediation:** Bind/read `web.server` in command config path and prioritize it before browser-origin fallback.
  - **Validation:** `GOOS=js GOARCH=wasm go build ./cmd/web` plus resolver unit tests.
  - **Dependencies:** None.

- [ ] Placeholder game-family modules are registered as normal runtime options, then fail later with not-implemented engine behavior — `cmd/server.go:58`, `cmd/server.go:59`, `cmd/server.go:60`, `serverengine/eldersign/module.go:43`, `serverengine/eldritchhorror/module.go:45`, `serverengine/finalhour/module.go:44`, `serverengine/common/runtime/unimplemented_engine.go:24`.
  - **Remediation:** Hide placeholders from default module registry or gate behind explicit experimental flag and fail at selection time.
  - **Validation:** command-level tests for module-selection behavior in `./cmd`.
  - **Dependencies:** None.

- [ ] `serverengine/common/{messaging,session,state,observability,monitoring,validation}` are currently doc-only scaffold packages and unreferenced by internal package graph — `serverengine/common/messaging/doc.go:5`, `serverengine/common/session/doc.go:5`, `serverengine/common/state/doc.go:5`, `serverengine/common/observability/doc.go:5`, `serverengine/common/monitoring/doc.go:5`, `serverengine/common/validation/doc.go:5`.
  - **Remediation:** Either remove until first real extraction, or migrate concrete shared primitives into each package as slices land.
  - **Validation:** `go list ./...` + import-graph diff after migration/removal.
  - **Dependencies:** Depends on arkham ownership migration.

### LOW

- [ ] `UnimplementedEngine` satisfies full `contracts.Engine` but includes no-op origin methods, weakening contract consistency (`AllowedOrigins()` returns `nil`, `SetAllowedOrigins()` is empty) — `serverengine/common/runtime/unimplemented_engine.go:31`, `serverengine/common/runtime/unimplemented_engine.go:33`, `serverengine/common/contracts/engine.go:40`.
  - **Remediation:** Store origins in placeholder engine or explicitly document that placeholder engines do not support origin filtering semantics.
  - **Validation:** `go test ./serverengine/common/runtime` with origin-set/read behavior assertions.
  - **Dependencies:** None.

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| `client/ebiten/host_status.go:5` empty function body | Build-tag shim for non-WASM platforms; DOM host status updates are intentionally JS-only. |
| `cmd/web_nowasm.go:10` returns non-WASM error | Correct platform guard, not an unfinished implementation. |
| Placeholder sprite atlas fallback (`client/ebiten/render/atlas.go:120`) | Explicit alpha-state behavior documented in README; does not block core gameplay runtime. |
| One-line exported constructors/getters from go-stats | Verified as valid thin wrappers/accessors, not stubs by behavior or tests. |
| Direct dependencies in `go.mod` potentially unused | `go mod why -m` confirms active import paths for all direct dependencies. |
