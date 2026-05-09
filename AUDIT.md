# ORGANIZATION AUDIT — 2026-05-08

## Architecture Summary
- Module: `github.com/opd-ai/bostonfear` (Go 1.24.1).
- Package inventory (`go list ./...`): 11 packages (`cmd/server`, `cmd/desktop`, `cmd/mobile`, `client/ebiten`, `client/ebiten/app`, `client/ebiten/render`, `serverengine`, `transport/ws`, `monitoring`, `monitoringdata`, `protocol`).
- Entrypoints: `cmd/server`, `cmd/desktop`, `cmd/web` (build-tagged), and `cmd/mobile` binding package.
- Library packages:
  - `serverengine`: game/session orchestration + rules + state transitions.
  - `protocol`: transport schema DTOs and enums.
  - `transport/ws`: HTTP/WebSocket transport adapter.
  - `monitoring`, `monitoringdata`: observability handlers + DTOs.
  - `client/ebiten/*`: reusable client app/network/render packages consumed by desktop/web/mobile entrypoints.
- Internal packages: none.
- Responsibility map (observed):
  - Orchestration owner: `serverengine` (`GameServer` lifecycle and action pipeline).
  - Business/domain logic owner: `serverengine` (actions, dice, turn flow, resources, doom, mythos).
  - Integration concerns owner: split between `transport/ws` (network upgrade/routing), `monitoring` (HTTP monitoring endpoints), and partially `serverengine` (connection management over `net.Conn`, plus a small monitoring-policy and filesystem-path leak noted below).
- Dependency flow (from `go list -f`): entrypoints depend on libraries; `transport/ws` depends on interface surface (`SessionEngine`); `monitoring` depends on `Provider` interface + DTO package. No circular dependencies were detected.
- Baseline checks:
  - `go-stats-generator analyze . --skip-tests` completed.
  - `go build ./...` succeeded.
  - `go test -race ./...` succeeded.

## Organization Scorecard
| Category | Rating | Evidence |
|----------|--------|----------|
| Library-Forward Design | ✅ | Core mechanics and server workflow are implemented in `serverengine`, not in `cmd/*`; entrypoints mostly wire dependencies and start runtime loops. |
| Entrypoint Thinness | ✅ | `cmd/server/main.go:23-53`, `cmd/desktop/main.go:17-31`, and `cmd/web/main.go:22-34` orchestrate config/setup and call library APIs. |
| Struct/Interface Boundaries | ⚠️ | Good boundary interfaces exist (`transport/ws/websocket_handler.go:15-18`, `monitoring/handlers.go:13-20`), but `serverengine` also imports `monitoring` (`serverengine/health.go:11`) creating reverse package coupling. |
| Separation of Concerns | ⚠️ | Most boundaries are clear, but client asset path ownership is in engine package (`serverengine/game_constants.go:8-13`) and consumed by server bootstrap (`cmd/server/main.go:44-45`). |
| Extensibility | ⚠️ | Extension seams are generally strong (interface-based transport/monitoring), but current boundary leaks make engine reuse in non-HTTP/non-filesystem hosts harder than necessary. |

## Findings
### CRITICAL
- [ ] None.

### HIGH
- [x] Reverse package coupling from domain engine to monitoring policy — COMPLETED: Removed `import monitoring` from `serverengine/health.go` and deleted `getSystemAlerts()` method. Monitoring package no longer depends on serverengine internals.

### MEDIUM
- [x] Filesystem/serving concern owned by engine package — COMPLETED: Moved `clientDir` constant from `serverengine/game_constants.go` to `cmd/server/main.go` as deployment configuration, not core engine concern.

### LOW
- [x] Origin validation logic duplicated across engine and transport — COMPLETED: Deleted `checkOrigin()` from serverengine, consolidated validation to `ValidateOrigin()` in `transport/ws/websocket_handler.go`, moved all tests to `transport/ws/websocket_handler_test.go`.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| "All exported structs need exported interfaces" | Rejected by cargo-cult check (Phase 3f-3). Interface seams already exist where substitution matters (`SessionEngine`, `Provider`, `Broadcaster`, `StateValidator`). Adding interfaces for DTOs/concrete state types would not improve extension boundaries. |
| "No `internal/` directory is automatically a design flaw" | Rejected by scope/context check (Phase 3f-1,4). README describes a mixed command + reusable package repository; absence of `internal/` alone is not a demonstrated extensibility blocker. |
| "Large `serverengine` package must be split immediately" | Rejected as standalone severity claim because baseline shows no circular dependencies and entrypoints remain thin. Size is monitored as a trend risk, not treated as a current architecture failure without boundary breakage evidence. |
| "Low cohesion score for `main` package indicates architectural issue" | Rejected because `main` intentionally contains small bootstrap functions by design; low cohesion in tiny entrypoints is expected and not a maintainability defect here. |

## Evidence Notes
- Primary metrics source: `tmp/organize-audit-metrics.json` and `go-stats-generator analyze . --skip-tests` summary.
- Package and import-direction evidence: `go list ./...` and `go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...`.
- Baseline execution artifacts reviewed: `tmp/organize-build-results.txt`, `tmp/organize-test-results.txt`.
