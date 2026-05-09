# ORGANIZATION AUDIT — 2026-05-08

## Architecture Summary

**Module**: `github.com/opd-ai/bostonfear` (Go 1.24.1)

### Entrypoints
| Entrypoint | Purpose |
|------------|---------|
| `cmd/server/main.go` | HTTP+WebSocket game server |
| `cmd/desktop/main.go` | Native Ebitengine desktop client |
| `cmd/mobile/binding.go` | Ebitengine mobile binding (package `mobile`, not `main`) |
| `cmd/web/main.go` | WASM web client entrypoint |

### Library Packages
| Package | Responsibility |
|---------|---------------|
| `serverengine` | Core game state, mechanics, connections, metrics, health |
| `protocol` | Shared JSON wire types (game state, actions, dice, etc.) |
| `transport/ws` | WebSocket upgrade adapter; defines `SessionEngine` interface |
| `monitoring` | HTTP health/metrics handlers; defines `Provider` interface |
| `monitoringdata` | Shared data-transfer types for monitoring snapshots |
| `client/ebiten` | WebSocket NetClient + LocalState for Go clients |
| `client/ebiten/app` | Ebitengine `Game` implementation (update/draw loop) |
| `client/ebiten/render` | Sprite/shader/atlas rendering utilities |

### Dependency Flow
```
cmd/server ──► serverengine ──► protocol
                            ──► monitoringdata
cmd/server ──► transport/ws ──► (net, gorilla/websocket)
cmd/server ──► monitoring   ──► monitoringdata

cmd/desktop ──► client/ebiten/app ──► client/ebiten ──► protocol
cmd/mobile  ──► client/ebiten/app

client/ebiten/app ──► client/ebiten/render
```

No circular dependencies detected. The dependency arrow never points from a library package back to a `cmd` package.

---

## Organization Scorecard

| Category | Rating | Evidence |
|----------|--------|----------|
| Library-Forward Design | ✅ | All game mechanics, connection handling, metrics, and health checking live in `serverengine`. `cmd/server/main.go` is 58 lines and contains zero game logic. |
| Entrypoint Thinness | ✅ | Server `main()` delegates immediately to `run()` (25 lines: parse flags, construct server, listen, route). Desktop `main()` is 20 lines: parse flag, construct `Game`, call `RunGame`. |
| Struct/Interface Boundaries | ⚠️ | `transport/ws` and `monitoring` each define local interfaces (`SessionEngine`, `Provider`) allowing injection of any implementation — good. However, `serverengine.GameServer` has no exported interface of its own; `cmd/server` depends on the concrete struct directly, and the `actionProcessor`/`sessionManager`/`metricsCollector` component wrappers inside `serverengine` add a layer of indirection without providing exported interfaces, limiting external testability. |
| Separation of Concerns | ⚠️ | `serverengine/connection.go` owns connection lifecycle, goroutine management, broadcast handler, action handler, and game-state broadcast. This is a single 350-line file driving I/O, goroutine orchestration, and state mutation. `serverengine/health.go` owns both health snapshot logic AND the public adapter methods (`CollectPerformanceMetrics`, etc.) that satisfy the `monitoring.Provider` interface — two distinct responsibilities in one file. |
| Extensibility | ✅ | New mechanics extend `serverengine/actions.go` and `serverengine/game_mechanics.go`. New transport adapters implement `SessionEngine`. New monitoring endpoints implement `Provider`. The `Scenario` struct allows injecting custom setup/win/lose logic without touching core code. |

---

## Findings

### HIGH

- [x] **`GameServer` has no exported interface** — `serverengine/game_server.go:18–67` — The concrete `*GameServer` type is injected into `transport/ws` and `monitoring` only because those packages define their own narrow interfaces (`SessionEngine`, `Provider`). However, code that needs to test or mock the full game lifecycle (e.g., an integration harness, a future admin API, or an alternative transport) must depend on the concrete struct. An exported `GameEngine` interface in `serverengine` capturing the public surface would close this gap without breaking existing code. — **Remediation:** 1. Define `type GameEngine interface { HandleConnection(net.Conn, string) error; Start() error; SetAllowedOrigins([]string); AllowedOrigins() []string; … }` in a new `serverengine/engine.go`. 2. Verify `*GameServer` satisfies it via `var _ GameEngine = (*GameServer)(nil)`. 3. Update `cmd/server/main.go` to accept `GameEngine`. Validate: `go build ./...`, `go test -race ./...`.

- [x] **`actionProcessor`/`sessionManager`/`metricsCollector` wrap without exporting interfaces** — `serverengine/components.go:1–42` — These three component types are unexported concrete wrappers around `*GameServer` with no corresponding interfaces. They were apparently introduced as a "separation of concerns" step but do not fulfil that goal: each wrapper simply calls back into the parent `GameServer`, making them aliasing indirection rather than dependency boundaries. Test code in `serverengine` cannot replace these with fakes. — **Remediation:** Either (a) remove the wrappers and retain the `Core` methods directly (net-zero change), or (b) define unexported interfaces (`actionExecutor`, `sessionHandler`, `metricsProvider`) and inject them into `GameServer`, enabling unit tests to substitute fakes without spinning up a full server. Choose (a) unless independent substitution is a stated need. Validate: `go test -race ./serverengine/...` must still pass.

### MEDIUM

- [x] **`serverengine/connection.go` owns I/O, goroutine orchestration, and game-state broadcast** — `serverengine/connection.go:302–370` — `broadcastHandler`, `actionHandler`, `broadcastGameState`, and the WebSocket read loop (`runMessageLoop`) all reside in one file alongside player registration and reconnect logic. This means a change to broadcast buffering requires understanding player registration code in the same file, and vice versa. — **Remediation:** Split into `serverengine/broadcast.go` (broadcast channel, broadcastHandler, broadcastGameState) and keep `connection.go` for connection lifecycle (register, reconnect, disconnect). Each file should then have a single, articulable responsibility. Validate: `go build ./...`, `go test -race ./serverengine/...`.

- [x] **`serverengine/health.go` conflates engine diagnostics with `monitoring.Provider` adapter** — `serverengine/health.go:157–183` — Six thin adapter methods (`CollectPerformanceMetrics`, `CollectConnectionAnalytics`, etc.) exist in `health.go` alongside the health snapshot and error-rate logic. The adapter methods exist solely to satisfy `monitoring.Provider`; they belong with the interface-implementation concern, not with game-health diagnostics. — **Remediation:** Move the six adapter methods to a new `serverengine/monitoring_adapter.go`. The file comment should read "implements monitoring.Provider". `health.go` then only contains `SnapshotHealth`, `validateAndRecoverState`, `calculateErrorRate`, and game-statistics helpers. Validate: `go build ./...`, `go test -race ./...`.

- [ ] **`serverengine` package is monolithic (138 functions, 44 structs, 14 files)** — `serverengine/` — The package hosts game mechanics, dice resolution, connection management, session metrics, health diagnostics, Mythos phase, gate mechanics, and error recovery. While individual files are well-named, the single package means all of this code shares a namespace and mutual visibility. Adding game features, transport changes, and monitoring changes all touch the same package. — **Remediation:** This is a LOW-friction issue in the current codebase; do not split prematurely. The natural next split, when the package exceeds ~200 functions or circular test dependencies appear, would be to extract `serverengine/mechanics` (actions, dice, mythos, gate) as a pure-logic sub-package with no network imports. Apply only when the complexity cost is felt.

### LOW

- [ ] **`go-stats-generator` suggests moving `InvestigatorDetective`, `InvestigatorOccultist`, etc. from `protocol/protocol.go` to `serverengine/game_constants.go`** — `protocol/protocol.go` — The `InvestigatorType` constants are defined in `protocol` and aliased into `serverengine`. Since the `protocol` package is also consumed by the Go client (`client/ebiten`), moving these constants to `serverengine` would create a client→serverengine dependency, which is architecturally backward. The current placement is intentional and correct. — **Remediation:** No change needed. The tool's suggestion is a false positive; see "False Positives" section.

- [ ] **`go-stats-generator` suggests moving `NewGameServer` to `cmd/server/main.go`** — `serverengine/game_server.go:75` — Placing a constructor in a `main` package would destroy reusability. The current placement is correct for a library-forward design. — **Remediation:** No change needed. Tool suggestion is a false positive.

- [ ] **`cmd/web/main.go` has no test file** — `cmd/web/` — All other entrypoints (`cmd/server`, `cmd/desktop`) lack test files too, which is acceptable for thin entry points. The web WASM main lacks documentation on what it differs from the desktop main. — **Remediation:** Add a doc comment to `cmd/web/main.go` explaining the WASM target. No structural change needed.

- [ ] **`monitoring` package has low cohesion score (0.7)** — `monitoring/handlers.go`, `monitoring/alerts.go` — The two files define HTTP handlers and alert-building utilities. These are related concerns; the low cohesion score is a tool artifact from the small function count. — **Remediation:** No structural change needed. Verify the tool score is driven by small package size, not genuine responsibility scatter.

---

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|-----------------|
| `go-stats-generator`: move `InvestigatorType` constants to `serverengine/game_constants.go` | `protocol` is the shared wire schema consumed by both server and Go clients. Moving game constants to `serverengine` would force client packages to import `serverengine`, reversing the intended dependency direction. Current placement is correct. |
| `go-stats-generator`: move `NewGameServer` to `cmd/server/main.go` | Placing a library constructor in a `main` package breaks reuse. The suggestion is algorithmically generated without architectural context. |
| `go-stats-generator`: move `MetricsHandler` to `cmd/server/main.go` | HTTP handlers belong in the `monitoring` library package; entrypoints should not own handler logic. Current placement is correct. |
| `go-stats-generator`: move `DefaultScenario` to `serverengine/game_server.go` | `game_constants.go` is an appropriate home for game configuration constants. Moving it into `game_server.go` would mix construction logic with constants. No value added. |
| Low cohesion in `main` package (0.3 score) | `main` packages are naturally low-cohesion by Go convention — they orchestrate dependencies without contributing domain logic. Not an actionable finding. |
| `processActionCore` complexity (16.6 overall) | The function handles a turn pipeline: pre-game actions, turn validation, dispatch, doom update, defeat check, turn advance, and broadcast. Each step delegates to a named helper. Complexity is inherent to coordinating five mechanics, not a sign of poor organization. |
| Connection handling in `serverengine` rather than `transport/ws` | The project architecture deliberately keeps game session state (player map, turn order) in `serverengine` and restricts `transport/ws` to the HTTP→WebSocket upgrade. `serverengine.HandleConnection` is the documented `SessionEngine` implementation entry point. |
| `client/ebiten` package structure (0 interfaces, 27 functions) | The Go client is in active migration (README notes "alpha — placeholder sprites"). Applying strict interface-boundary standards to an in-flight rewrite would produce premature abstractions. |
