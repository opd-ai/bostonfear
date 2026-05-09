# Organization Gaps — 2026-05-08

## Gap 1: Domain-to-Monitoring Dependency Inversion
- **Desired Organization**: `serverengine` should expose state/metrics through stable library APIs; `monitoring` should consume those APIs without `serverengine` importing handler/policy packages.
- **Current State**: `serverengine` imports `monitoring` in `serverengine/health.go:11` and calls `monitoring.BuildSystemAlerts` at `serverengine/health.go:194`.
- **Impact**: Core engine becomes less reusable in non-HTTP environments and adapter replacement becomes harder because domain package carries monitoring-policy coupling.
- **Closing the Gap**:
  - Move alert derivation calls to `monitoring` handlers only.
  - Remove `getSystemAlerts` from `serverengine` and keep engine outputs as neutral metrics/state.
  - If common alert threshold helpers are needed, place pure data logic in `monitoringdata` (or a new neutral package) instead of importing `monitoring` into engine.
  - Validate with `go build ./...`, `go test -race ./...`, `go-stats-generator analyze . --skip-tests --sections packages,interfaces`.

## Gap 2: Filesystem Path Ownership in Core Engine
- **Desired Organization**: Entrypoints/transports should own deployment-specific paths (static assets, dashboard files), while `serverengine` stays path-agnostic and rules-focused.
- **Current State**: Asset path constant is defined in `serverengine/game_constants.go:8` and exported through `serverengine/game_constants.go:12`; server bootstrap consumes it at `cmd/server/main.go:44-45`.
- **Impact**: Engine package contains environment/layout assumptions (`../client`) that reduce portability and complicate embedding in alternative hosts.
- **Closing the Gap**:
  - Move asset path configuration to `cmd/server` flags/config or `transport/ws` route setup parameters.
  - Keep `serverengine` free of repo-relative path constants.
  - Add explicit server wiring tests in `cmd/server` (or transport package tests) for route setup using injected paths.
  - Validate with `go build ./...`, `go test -race ./...`, `go-stats-generator analyze . --skip-tests --sections packages,patterns`.

## Gap 3: Duplicated Origin-Validation Ownership
- **Desired Organization**: A single package should own WebSocket origin validation rules to avoid policy drift.
- **Current State**: Origin policy logic appears in `serverengine/game_server.go:168-199` and independently in `transport/ws/websocket_handler.go:48-83`.
- **Impact**: Future policy changes risk divergence between transport behavior and engine-side helper behavior, increasing maintenance overhead and ambiguity.
- **Closing the Gap**:
  - Keep transport package as the single owner for request-origin checks.
  - Limit engine to storing normalized allow-list (`AllowedOrigins`) or extract a small pure shared validator used by transport and tests.
  - Add focused tests around one canonical validator path.
  - Validate with `go test -race ./...` and `go-stats-generator analyze . --skip-tests --sections duplication,packages`.
