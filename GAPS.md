# Organization Gaps — 2026-05-08

## GAP-1: No Exported Interface for the Core Game Engine

- **Desired Organization**: `serverengine` exports a `GameEngine` (or similarly named) interface capturing the public behaviour of `*GameServer`. Consumers (`cmd/server`, future admin APIs, integration test harnesses) depend on the interface, not the concrete struct.
- **Current State**: `*GameServer` is the only type available at the `serverengine` package boundary. `transport/ws` and `monitoring` work around this by defining their own narrow interfaces (`SessionEngine`, `Provider`) locally, but code that needs to stub the *full* engine lifecycle cannot do so without the concrete struct.
- **Impact**: Integration tests that want to replace the game engine with a controlled fake must import and construct a real `*GameServer`. Adding a second server implementation (e.g., a replay engine or a rules-only headless engine) requires duplicating the full method set rather than implementing a declared contract. Discovering the public surface of `serverengine` requires reading all 14 source files rather than a single interface declaration.
- **Closing the Gap**:
  1. Create `serverengine/engine.go` with `type GameEngine interface { … }` listing all methods currently called from outside the package.
  2. Add a compile-time assertion: `var _ GameEngine = (*GameServer)(nil)`.
  3. Update `cmd/server/main.go` to pass a `GameEngine` value to `transportws.NewWebSocketHandler` and `monitoring.HealthHandler` instead of `*GameServer`.
  4. Validate: `go build ./...`, `go test -race ./...`.

---

## GAP-2: Component Wrappers Provide Indirection Without Interface Contracts

- **Desired Organization**: The `actionProcessor`, `sessionManager`, and `metricsCollector` types in `serverengine/components.go` either (a) are removed as unnecessary pass-through wrappers, or (b) represent genuine seams backed by unexported interfaces that allow unit-test fakes to be injected into `GameServer`.
- **Current State**: All three types are unexported concrete structs that call back into `*GameServer` via a stored pointer. Neither the structs nor their methods are exposed as interfaces. `GameServer` stores them as concrete fields (`actionProcessor *actionProcessor`, etc.), so the indirection cannot be used to substitute a fake in tests. The wrappers' only observable effect is one extra call-stack frame per action.
- **Impact**: Tests that want to validate action processing in isolation still need a full `GameServer`. The wrapper pattern signals intent to separate concerns but does not deliver it, which can mislead future contributors into adding logic to the wrapper layer (thinking it is a proper boundary) rather than the correct owner.
- **Closing the Gap** (choose one path):
  - **Path A — Remove**: Delete `components.go`. Move any logic that lives exclusively in wrapper methods back to `GameServer`. Update `processAction` to call `gs.processActionCore` directly. Net result: simpler code, no change in behaviour.
  - **Path B — Promote to real seams**: Define unexported interfaces `type actionExecutor interface { Process(PlayerActionMessage) error }` etc. in `serverengine/interfaces.go`. Change `GameServer` fields to use these interfaces. Provide mock implementations in `*_test.go` files. This is the heavier investment and should be deferred until test coverage gaps justify it.
  - Validate either path: `go test -race ./serverengine/...` must pass and the function-count total in `go-stats-generator` should decrease (Path A) or the interface count should increase (Path B).

---

## GAP-3: `connection.go` Conflates Connection Lifecycle with Broadcast Infrastructure

- **Desired Organization**: `serverengine/connection.go` owns player registration, reconnect-token handling, message-loop, and disconnect cleanup. A separate `serverengine/broadcast.go` owns the outbound channel pipeline: `broadcastHandler`, `actionHandler`, and `broadcastGameState`.
- **Current State**: All of the above reside in a single 350-line `connection.go`. The file's leading comment says it handles "WebSocket connection handling" but the `broadcastHandler` goroutine (line 302) and `actionHandler` goroutine (line 327) are infrastructure concerns orthogonal to connection lifecycle.
- **Impact**: A developer changing broadcast buffering (e.g., increasing channel capacity, adding backpressure) must navigate through player-registration and reconnect logic to locate relevant code. Likewise, a developer adding a new connection lifecycle hook (e.g., rate-limiting new connections) must scroll past broadcast infrastructure.
- **Closing the Gap**:
  1. Create `serverengine/broadcast.go`. Move `broadcastHandler`, `actionHandler`, `broadcastGameState`, `buildGameUpdateMessage`, and `broadcastActionResults` into it.
  2. `connection.go` retains: `HandleConnection`, `registerPlayer`, `removeConnection`, `runMessageLoop`, `handleIncomingMessage`, `handlePlayerDisconnect`, `restorePlayerByToken`, `cleanupDisconnectedPlayers`.
  3. Validate: `go build ./...`, `go test -race ./serverengine/...`.

---

## GAP-4: `health.go` Owns Both Engine Diagnostics and the `monitoring.Provider` Adapter

- **Desired Organization**: Engine health diagnostics (`SnapshotHealth`, `validateAndRecoverState`, `calculateErrorRate`, `getGameStatisticsCore`) live in `serverengine/health.go`. The six thin adapter methods that implement `monitoring.Provider` live in a separate `serverengine/monitoring_adapter.go`.
- **Current State**: `serverengine/health.go` lines 157–183 contain `CollectPerformanceMetrics`, `CollectConnectionAnalytics`, `CollectMemoryMetrics`, `CollectGCMetrics`, `CollectMessageThroughput`, and `GameStatistics`. These are one-liner delegations that exist solely to satisfy the `monitoring.Provider` interface. They have no domain logic and belong to the integration-adapter concern, not the health-diagnostics concern.
- **Impact**: When changing the `monitoring.Provider` contract (e.g., adding a new method), a developer must locate the relevant code in `health.go` rather than an adapter file, conflating two separate change reasons for a single file.
- **Closing the Gap**:
  1. Create `serverengine/monitoring_adapter.go` with package doc comment: "// monitoring_adapter.go implements the monitoring.Provider interface for GameServer."
  2. Move the six adapter methods to the new file.
  3. `health.go` should then contain only diagnostics: `SnapshotHealth`, `validateAndRecoverState`, `measureHealthCheckResponseTime`, `calculateErrorRate`, `getGameStatistics`, `getGameStatisticsCore`, and the pure helpers.
  4. Validate: `go build ./...`, `go test -race ./...`.

---

## GAP-5: `serverengine` Has No `internal/` Sub-boundary Despite Mixed Exported / Unexported Surface

- **Desired Organization**: Methods and types that are purely internal implementation helpers (e.g., `rollDicePool`, `validateResources`, `cleanupDisconnectedPlayers`, the three component wrappers) are not accidentally reachable from consumer packages.
- **Current State**: Because everything is in the one `serverengine` package, all unexported identifiers are visible within the package but not outside it. This is Go-standard behaviour and appropriate for the current codebase size. However, there is no `serverengine/internal/` sub-package barrier, meaning that if `serverengine` is ever split, previously internal helpers become exported identifiers requiring a deprecation cycle.
- **Impact**: Low risk at present. If the package grows significantly or a mechanics sub-package is extracted, the lack of `internal/` means all current methods would become part of the public API by default, requiring careful auditing.
- **Closing the Gap** (deferred — apply when `serverengine` exceeds ~200 functions or a sub-package extraction is planned):
  1. Identify pure-logic helpers with no transport or metrics dependencies (dice, resource validation, adjacency checks).
  2. Move them to `serverengine/internal/mechanics` with no exported types that reference `net.Conn` or monitoring data.
  3. `serverengine` imports `internal/mechanics`; external packages cannot.
  4. Validate: `go build ./...`, `go test -race ./...`, `go-stats-generator analyze . --sections packages,interfaces`.
