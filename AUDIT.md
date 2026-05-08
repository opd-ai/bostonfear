# ORGANIZATION AUDIT — 2026-05-08

## Architecture Summary
- Module: `github.com/opd-ai/bostonfear` (Go `1.24.1`) with two direct dependencies (`gorilla/websocket`, `ebiten/v2`).
- Package inventory (`go list ./...`): 10 packages.
- Entrypoints: `cmd/server`, `cmd/desktop`, `cmd/web` (all `package main`; `cmd/mobile` is binding package, not a `main` command).
- Library packages:
  - `serverengine`: game rules, state transitions, turn orchestration, connection lifecycle, monitoring metric collection.
  - `protocol`: shared wire/data schema.
  - `monitoring`: HTTP handlers over provider interface.
  - `monitoringdata`: DTOs for monitoring payloads.
  - `transport/ws`: HTTP route registration over `net.Listener`.
  - `client/ebiten`, `client/ebiten/app`, `client/ebiten/render`: reusable Go client components.
- Internal packages: none present (`internal/` not used).
- Responsibility map:
  - Orchestration owner: `serverengine` (`GameServer`, action/broadcast goroutines, turn progression).
  - Domain/business logic owner: `serverengine` (actions, dice, doom, encounters, mythos).
  - Integration owner: split, but with overlap.
    - `transport/ws` owns route wiring.
    - `monitoring` owns HTTP monitoring endpoints.
    - `serverengine` still owns WebSocket upgrade/session loop and observability aggregation logic.
- Baseline evidence:
  - `go-stats-generator analyze . --skip-tests --sections packages,functions,interfaces,structs,patterns,duplication`
  - `go build ./...` passed.
  - `go test -race ./...` passed.
  - Metrics snapshot: 10 packages, 55 structs, 4 interfaces, 58 functions, 163 methods.

## Online Research Notes (Brief)
- GitHub issue/PR query for architecture terms showed only a small set of closed items; no strong externally reported boundary/extensibility pain pattern was visible from accessible public metadata.
- Go organization guidance (`go.dev/doc/modules/layout`) emphasizes:
  - thin `cmd/*` commands,
  - reusable packages for shared behavior,
  - optional `internal/` for non-exportable implementation details,
  - clear command/package split in mixed repositories.

## Organization Scorecard
| Category | Rating | Evidence |
|----------|--------|----------|
| Library-Forward Design | WARN | Core gameplay is in `serverengine` (good), but transport and monitoring policy logic are also embedded there (`serverengine/game_server.go:22`, `serverengine/connection.go:22`, `serverengine/health.go:207`). |
| Entrypoint Thinness | PASS | Entrypoints mostly parse config, construct deps, and run (`cmd/server/main.go:23`, `cmd/desktop/main.go:19`, `cmd/web/main.go:22`). |
| Struct/Interface Boundaries | WARN | Useful interfaces exist (`monitoring/handlers.go:14`, `serverengine/interfaces.go:7`), but an exported concrete transport adapter leaks websocket details (`serverengine/connection_wrapper.go:13`, `serverengine/connection_wrapper.go:21`). |
| Separation of Concerns | WARN | `GameServer` aggregates domain state, ws transport state, and observability state in one type (`serverengine/game_server.go:22-70`), and WebSocket lifecycle remains in engine package (`serverengine/connection.go:285`). |
| Extensibility | WARN | Adding non-WebSocket transport or alternate monitoring backend likely requires edits in `serverengine`, not only adapter packages (`serverengine/connection.go:22`, `serverengine/game_server.go:29`, `serverengine/health.go:150`). |

## Findings
### CRITICAL
- [x] None.

### HIGH
- [ ] Core engine package mixes domain rules with WebSocket transport lifecycle — `serverengine/game_server.go:26`, `serverengine/game_server.go:29`, `serverengine/connection.go:22`, `serverengine/connection.go:285` — `serverengine` is not transport-agnostic because it owns upgrader/session/IO details in the same package as rules/orchestration; this raises extension cost for alternate transports and increases boundary churn risk — **Remediation:**
  1. Introduce a transport-neutral engine surface (for example: `RegisterConnection(net.Conn)`, `SubmitAction(PlayerActionMessage)`, `SnapshotState()`), implemented in `serverengine` with no `gorilla/websocket` imports.
  2. Move upgrade/session loop (`handleWebSocket`, reconnect token query parsing, ws read/write loops) into `transport/ws`.
  3. Keep `cmd/server` as pure wiring of engine + transport.
  4. Validation: run `go build ./...`, `go test -race ./...`, `go-stats-generator analyze . --sections packages,interfaces,structs` and confirm `serverengine` no longer imports websocket package.

### MEDIUM
- [ ] `GameServer` is a multi-concern aggregate (rules + transport + observability state) — `serverengine/game_server.go:22-70`, `serverengine/health.go:150`, `serverengine/health.go:207` — high-consequence changes converge on one type, reducing independent replaceability/test seam clarity for orchestration vs telemetry concerns — **Remediation:**
  1. Extract focused components (`ActionProcessor`, `SessionRegistry`, `MetricsCollector`) as package-local types.
  2. Keep `GameServer` as coordinator that composes these components.
  3. Move alert-threshold policy (`getSystemAlerts`) into monitoring-facing package or dedicated observability component.
  4. Validation: `go test -race ./...`; `go-stats-generator analyze . --sections functions,patterns` should show reduced hotspot concentration in `GameServer` methods.

- [ ] Exported concrete connection adapter leaks websocket dependency across package API — `serverengine/connection_wrapper.go:13`, `serverengine/connection_wrapper.go:21` — public API exposes `*websocket.Conn` in constructor, creating avoidable coupling for consumers and reducing interface-driven substitution at package boundaries — **Remediation:**
  1. Move wrapper type to `transport/ws` and make it unexported (`connectionWrapper`).
  2. If cross-package creation is required, expose a transport-level constructor returning `net.Conn` instead of concrete type.
  3. Keep `serverengine` consuming only `net.Conn`/engine interfaces.
  4. Validation: `go build ./...`, `go test -race ./...`; verify no exported serverengine symbol has `*websocket.Conn` in signature.

### LOW
- [ ] Package comments in multiple `serverengine` files still state `Package main`, creating misleading architectural signals — `serverengine/actions.go:1`, `serverengine/connection.go:1`, `serverengine/game_server.go:1`, `serverengine/health.go:1` — this is low runtime risk but increases maintenance confusion during boundary-oriented refactors — **Remediation:**
  1. Update package comments to `Package serverengine` with concise ownership statements.
  2. Keep comments aligned with actual package ownership after each move/refactor.
  3. Validation: `go test -race ./...` and quick grep check for stale header pattern.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| "Entrypoints are fat and contain business logic" | Rejected: `cmd/server/main.go`, `cmd/desktop/main.go`, and `cmd/web/main.go` are orchestration-focused and thin in current state. |
| "No `internal/` directory means architecture is wrong" | Rejected: README states reusable package goals (`serverengine`, shared protocol/client packages), so exportable packages are intentional rather than accidental leakage. |
| "Every exported struct must have a matching interface" | Rejected: only extension seams require abstraction; pure data DTOs in `protocol`/`monitoringdata` do not benefit from interface indirection. |
| "Concrete usage inside package is always bad" | Rejected: package-local concrete types are acceptable; findings were limited to package-boundary/API coupling that impacts substitution or reuse. |
| "go-stats-generator placement warnings are all actionable architecture issues" | Rejected: auto-suggestions were filtered; only findings confirmed by package-goal context and line-level boundary evidence were kept. |
