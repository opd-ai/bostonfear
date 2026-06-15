# Project Overview
BostonFear is a rules-only, copyright-safe Arkham Horror engine for cooperative multiplayer play. The project centers on a Go server that owns game rules, turn progression, dice resolution, resources, doom escalation, and reconnection state, plus a Go/Ebitengine client that renders the board and UI for desktop, web (WASM), and mobile from one codebase. Game-family modules are selected through runtime configuration; `arkhamhorror` is the default, with additional modules available under `serverengine/eldersign`, `serverengine/eldritchhorror`, and `serverengine/finalhour`.

The primary audience is Go developers extending the engine, transport, content, or cross-platform client. Typical work includes adding mechanics, refining UI scenes, expanding Nightglass scenario content, or improving monitoring and validation. Keep all contributions aligned with the ADR-backed architecture: interface-based networking, modular game-family boundaries, structured logging, centralized server authority, and original content only.

## Technical Stack
- **Primary Language**: Go 1.25
- **Frameworks**: gorilla/websocket v1.5.3 for WebSocket transport adapters; Ebitengine v2.9.9 for desktop/WASM/mobile rendering; Cobra v1.10.2 for CLI commands; Viper v1.21.0 for config loading; go.yaml.in/yaml/v3 v3.0.4 for YAML-backed content/assets
- **Testing**: Go's built-in testing package with table-driven tests, race detection, `go vet`, display-gated Ebitengine tests (`-tags=requires_display`), documentation coverage checks via `go-stats-generator`, and server benchmark coverage for broadcast latency
- **Build/Deploy**: `go build ./cmd/server`, `go build ./cmd/desktop`, `GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web`, ebitenmobile bindings for `cmd/mobile`, config via root `config.toml`, CI workflows in `.github/workflows/`

## Code Assistance Guidelines
1. Keep the server authoritative: game rules, turn validation, dice outcomes, doom changes, and reconnection/session state belong in `serverengine` or game-family modules, not in transport handlers or Ebitengine UI code.
2. Preserve the modular architecture. Put shared runtime contracts in `serverengine/common`, Arkham-specific behavior in `serverengine/arkhamhorror`, and transport concerns in `transport/ws`; do not hardcode WebSocket details into engine packages.
3. Follow the repository's Go standards from `CONTRIBUTING.md`: document exported identifiers, wrap errors with context, prefer channels and careful goroutine cleanup, and keep naming idiomatic (`HTTP`, `JSON`, `HandleConnection`, `Broadcaster`).
4. Maintain project validation gates. New code should be formatted with `gofmt`, pass `go vet ./...`, keep documentation coverage at or above the enforced 80% threshold, and include tests for new logic or regressions.
5. Treat the Ebitengine client as a multi-scene, cross-platform Go application. Reuse shared client state and networking code, preserve the current 800x600 logical layout used by `client/ebiten`, and avoid introducing platform-specific forks when the same behavior can stay in shared Go code unless you are intentionally migrating the renderer.
6. Respect project security and deployment constraints: keep the repository rules-only and copyright-safe, preserve origin validation through `network.allowed_origins`, and prefer structured logging through `serverengine/common/logging` rather than ad hoc `log.Printf`-style output.
7. When changing content, config, or startup behavior, verify the Nightglass default scenario flow, `BOSTONFEAR_GAME`/`server.game` module selection, and the documented fallback rules for `scenario.default_id`.
8. **Complete Feature Implementations**: Always prefer completing the full implementation of any feature rather than leaving partial or placeholder code. When a complete implementation is not feasible, insert clear inline `TODO` comments describing what remains, why it was deferred, and any known constraints (e.g., `// TODO: Implement retry logic once the error categorization schema is finalized`). Never leave code in a silently incomplete state.

## Project Context
- **Domain**: A fan-made, original-content, rules-only engine for Arkham Horror-style cooperative investigation gameplay with turn-based actions, dice tests, mythos escalation, and scenario-driven progression.
- **Architecture**: Centralized server authority with JSON protocol types in `protocol/`, WebSocket adaptation in `transport/ws`, monitoring endpoints in `monitoring/`, and an ADR-backed modular engine that defaults to `serverengine/arkhamhorror`.
- **Key Directories**: `cmd/server`, `cmd/desktop`, and `cmd/web` are platform entrypoints; `client/ebiten/` contains scenes, rendering, and networking; `serverengine/` contains engine/runtime logic; `serverengine/arkhamhorror/content/nightglass/` contains default scenario content; `docs/adr/` records architecture decisions.
- **Configuration**: Root `config.toml` controls `server.game`, listen settings, desktop/web server URLs, `network.allowed_origins`, and `scenario.default_id`; production changes should keep config/docs/CLI behavior in sync.

## Quality Standards
- Run and preserve the existing validation flow: `go vet ./...`, the relevant `go test` commands (including `-race` and `-tags=requires_display` when applicable), and Web/WASM or desktop builds for touched areas.
- CI enforces documentation coverage with `scripts/check-doc-coverage.sh` (minimum 80%), dependency direction checks with `scripts/check-common-deps.sh`, and benchmark scrutiny for broadcast latency; changes affecting broadcast behavior should stay below the 200ms threshold referenced in CI.
- Update README, ADRs, runbooks, or content specifications whenever behavior, configuration, architecture, or operator workflows change so repository guidance stays aligned with the implementation.

## Networking Best Practices (for Go projects)
When declaring network variables, always use interface types:
- Never use `net.UDPAddr`, `net.IPAddr`, or `net.TCPAddr`. Use `net.Addr` only instead.
- Never use `net.UDPConn`, use `net.PacketConn` instead
- Never use `net.TCPConn`, use `net.Conn` instead
- Never use `net.UDPListener` or `net.TCPListener`, use `net.Listener` instead
- Never use a type switch or type assertion to convert from an interface type to a concrete type. Use the interface methods instead.

This approach enhances testability and flexibility when working with different network implementations or mocks.
