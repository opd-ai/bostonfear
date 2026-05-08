# Organization Gaps — 2026-05-08

## Gap 1: Transport-Agnostic Engine Boundary
- **Desired Organization**: `serverengine` should expose transport-neutral orchestration and rules APIs, while WebSocket upgrade/session lifecycle lives in `transport/ws`.
- **Current State**: `serverengine` owns websocket upgrader state and handler/session loop logic (`serverengine/game_server.go:26`, `serverengine/game_server.go:29`, `serverengine/connection.go:22`, `serverengine/connection.go:285`).
- **Impact**: Adding another transport (for example, in-process bots, alternate sockets, test harness transport) requires invasive edits in the core engine package.
- **Closing the Gap**:
  1. Define a transport-neutral engine interface in `serverengine` for player lifecycle/action submission/state snapshots.
  2. Move websocket-specific upgrade/read/write/reconnect token parsing into `transport/ws`.
  3. Keep `cmd/server` as composition root only.
  4. Validate with `go build ./...`, `go test -race ./...`, and `go-stats-generator analyze . --sections packages,interfaces,structs`.

## Gap 2: Coordinator vs Multi-Concern God Type
- **Desired Organization**: Keep orchestration coordinator small, with observability, connection/session management, and action execution delegated to focused components.
- **Current State**: `GameServer` holds domain state, ws connection maps, origin policy, connection quality data, and performance counters in one type (`serverengine/game_server.go:22-70`), plus monitoring-policy methods in engine (`serverengine/health.go:150`, `serverengine/health.go:207`).
- **Impact**: Feature work on rules, transport quality, and telemetry converges on the same type/files, increasing merge conflict frequency and reducing independent testability.
- **Closing the Gap**:
  1. Extract package-local components (`actionProcessor`, `sessionManager`, `metricsCollector`) from `GameServer`.
  2. Keep `GameServer` as orchestrator over those components.
  3. Shift system-alert policy composition to monitoring-side component with data-only snapshots from engine.
  4. Validate with `go test -race ./...` and compare hotspot concentration in `go-stats-generator` function/pattern sections.

## Gap 3: Public API Coupled to Concrete WebSocket Type
- **Desired Organization**: Public package boundaries should accept/return interfaces (`net.Conn`, package-specific interfaces) unless concrete exposure is a deliberate extension seam.
- **Current State**: `ConnectionWrapper` and `NewConnectionWrapper` are exported from `serverengine` and constructor requires `*websocket.Conn` (`serverengine/connection_wrapper.go:13`, `serverengine/connection_wrapper.go:21`).
- **Impact**: API surface leaks transport detail and limits substitution ergonomics for downstream users/tests.
- **Closing the Gap**:
  1. Move connection wrapper to `transport/ws` as unexported implementation.
  2. Expose only interface-shaped construction (`net.Conn`) where cross-package handoff is needed.
  3. Ensure `serverengine` APIs remain websocket-agnostic.
  4. Validate with `go build ./...`, `go test -race ./...`, and signature scan from `go-stats-generator` for exported symbols.

## Gap 4: Package-Header Convention Drift
- **Desired Organization**: Package-level docs should accurately reflect ownership/boundary to preserve maintainability and onboarding clarity.
- **Current State**: Several `serverengine` files still declare comment headers as `Package main` (`serverengine/actions.go:1`, `serverengine/connection.go:1`, `serverengine/game_server.go:1`, `serverengine/health.go:1`).
- **Impact**: Low technical risk but recurring cognitive friction during architecture work and code review.
- **Closing the Gap**:
  1. Normalize headers to `Package serverengine` with concise role descriptions.
  2. Add a lightweight doc consistency check in review checklist.
  3. Validate with grep and `go test -race ./...`.
