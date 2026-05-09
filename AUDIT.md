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
- [ ] Reverse package coupling from domain engine to monitoring policy — `serverengine/health.go:11`, `serverengine/health.go:188-194` — `serverengine` imports `monitoring` only to compute alerts, which inverts the intended dependency direction where monitoring should consume engine state, not be required by engine internals. This increases substitution cost for alternate hosts and makes the core engine less transport/adapter-neutral. — **Remediation:** Move alert policy composition out of `serverengine` by (1) deleting `getSystemAlerts` from `serverengine`, (2) keeping alert construction exclusively in `monitoring` using existing provider snapshots, and (3) if shared threshold logic is needed, place pure data helpers in `monitoringdata` (no handler imports). Validate with: `go build ./...`, `go test -race ./...`, `go-stats-generator analyze . --skip-tests --sections packages,interfaces,structs`.

### MEDIUM
- [ ] Filesystem/serving concern owned by engine package — `serverengine/game_constants.go:8-13`, consumed at `cmd/server/main.go:44-45` — `serverengine.ClientDir()` exposes a repo-layout-specific asset path used by HTTP handlers, mixing runtime integration configuration into the rules/orchestration package. This reduces library portability (embedding engine into another binary requires inherited path assumptions). — **Remediation:** Move client asset path ownership to server bootstrap (`cmd/server`) or transport adapter configuration (`transport/ws` route setup args), and keep `serverengine` free of filesystem path constants. Validate with: `go build ./...`, `go test -race ./...`, `go-stats-generator analyze . --skip-tests --sections packages,patterns`.

### LOW
- [ ] Origin validation logic duplicated across engine and transport — `serverengine/game_server.go:166-200` and `transport/ws/websocket_handler.go:48-83` — the same policy logic exists in two packages, while only transport path is exercised at upgrade time. This creates maintenance drift risk and unclear ownership for security policy behavior. — **Remediation:** Keep a single authoritative origin validator in `transport/ws`; reduce `serverengine` responsibility to normalized origin config (`AllowedOrigins` state) only, or extract a tiny shared pure function package consumed by transport and tests. Validate with: `go test -race ./...`, `go-stats-generator analyze . --skip-tests --sections duplication,packages`.

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
