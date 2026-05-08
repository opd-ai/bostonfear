# ORGANIZATION AUDIT — 2026-05-08

## Architecture Summary

- Module: `github.com/opd-ai/bostonfear` on Go `1.24.1` with two direct runtime dependencies: `github.com/gorilla/websocket` and `github.com/hajimehoshi/ebiten/v2`.
- Baseline: `go build ./...` passes and `go test -race ./...` passes. This audit therefore focuses on organization and extensibility, not incidental breakage.
- Package map from `go list ./...`: `client/ebiten`, `client/ebiten/app`, `client/ebiten/render`, `cmd/desktop`, `cmd/mobile`, `cmd/server`. There is no `internal/` tree and no importable Go server package.
- Entrypoints: `cmd/desktop/main.go`, `cmd/web/main.go`, and `cmd/mobile/binding.go` are thin adapters over the Ebitengine client packages. `cmd/server/main.go` is itself small, but the surrounding `cmd/server` command package owns the entire server runtime.
- Business/domain logic ownership: the server command package owns `GameServer`, `GameState`, action dispatch, dice resolution, Mythos flow, validation, metrics, dashboard handlers, and connection management (`cmd/server/game_server.go:22`, `cmd/server/game_server.go:270`, `cmd/server/game_mechanics.go:1`, `cmd/server/connection.go:1`, `cmd/server/metrics.go:1`).
- Integration ownership: HTTP/WebSocket setup also lives in the same server command package (`cmd/server/game_utils.go:9`, `cmd/server/connection.go:256`).
- Client-side layering is cleaner: `cmd/{desktop,web,mobile}` depend on `client/ebiten/app`, which composes reusable `client/ebiten` state/network code and `client/ebiten/render` rendering code (`client/ebiten/app/game.go:69`, `client/ebiten/net.go:39`, `client/ebiten/state.go:189`).
- Primary stats evidence from `go-stats-generator analyze . --skip-tests`:
  - 5 Go packages, 27 analyzed files, 53 functions, 157 methods, 59 structs, 3 interfaces.
  - The `main` package dominates the codebase with 142 functions, 41 structs, and 18 files.
  - Only 3 interfaces exist total: `Broadcaster`, `StateValidator`, and client `Scene`.
- External context: repository issue/PR history shows migration and audit work, but no substantive public issue trail about package-boundary pain. Go’s own module-layout guidance for server projects recommends `cmd/` entrypoints with supporting logic in `internal/` packages. I treated the README’s explicit “entry point + game logic” documentation as intentional context, not an automatic failure, and only recorded findings where that choice materially limits reuse or extension.

## Organization Scorecard

| Category | Rating | Evidence |
|----------|--------|----------|
| Library-Forward Design | ❌ | Client code is library-forward, but the entire server engine remains in `cmd/server` `package main`; there is no importable server package despite the repo positioning itself as a rules engine. |
| Entrypoint Thinness | ⚠️ | `cmd/desktop`, `cmd/web`, and `cmd/mobile` are thin. `cmd/server/main.go` is also thin, but the command package around it is not thinly layered because gameplay, transport, and observability stay inside the same command package. |
| Struct/Interface Boundaries | ⚠️ | There are only three interfaces total. The server’s two abstractions are defined inside `package main`, so they help local testing but do not form reusable package-boundary contracts. |
| Separation of Concerns | ⚠️ | Files are grouped by topic, but package boundaries stop at the file level on the server. Mechanics, transport, monitoring, and recovery all compile into one package. |
| Extensibility | ⚠️ | New client entrypoints can reuse library packages cleanly. New server hosts, protocols, or external tools would need extraction work first because the server engine is trapped in `cmd/server`. |

## Findings

### CRITICAL

- [x] Core server engine is implemented in the command package rather than an importable library or internal package — `cmd/server/game_server.go:22`, `cmd/server/game_server.go:270`, `cmd/server/game_mechanics.go:1`, `cmd/server/actions.go:1` — The repository has no importable server package at all, and `go-stats-generator` attributes 142 functions and 41 structs to `main`. That means the gameplay engine, state model, and action pipeline cannot be reused by another command, simulation harness, or alternate host without first moving code out of `cmd/server`. This is the main architecture gap relative to the repo’s stated “rules-only game engine” positioning and to the audit’s library-forward criteria. — **Remediation:** Create an importable server-engine boundary, preferably `serverengine` or `game`, and move `GameServer`, `GameState`, action logic, dice logic, mythos logic, and validation there. Leave `cmd/server/main.go` and `setupServer` as orchestration only. Start incrementally by moving `game_types.go`, `game_server.go`, `game_mechanics.go`, `actions.go`, `dice.go`, and `mythos.go` together with unchanged behavior. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections packages,interfaces,structs`.

### HIGH

- [x] Interface seams exist, but they are owned by an unimportable command package, so the advertised interface-driven design does not extend across package boundaries — `cmd/server/interfaces.go:7`, `cmd/server/interfaces.go:13`, `cmd/server/game_server.go:31`, `cmd/server/game_server.go:34` — `Broadcaster` and `StateValidator` are the only server-specific interfaces reported by `go-stats-generator`, and both live in `package main`. That makes them useful for local substitution inside the command package, but unavailable to any other package that might want to host the server engine or provide alternative implementations. The result is a real extensibility seam that stops one package too early. — **Remediation:** After extracting the server engine package, move these contracts to the package that consumes them. Keep provider implementations private, for example a WebSocket broadcaster in `internal/transport/ws` and a validator implementation in `internal/validation`. Reconstruct `GameServer` through constructors that accept those interfaces from outside the command package. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections interfaces,packages`.

### MEDIUM

- [x] Server concern boundaries are expressed at the file level, but not at the package level — `cmd/server/game_mechanics.go:1`, `cmd/server/connection.go:1`, `cmd/server/dashboard.go:1`, `cmd/server/metrics.go:1`, `cmd/server/error_recovery.go:10` — The original server package mixed mechanics, transport, and monitoring in one namespace. The server now keeps gameplay and state orchestration in `serverengine`, registers HTTP routes in `transport/ws`, and serves health/metrics/dashboard endpoints from `monitoring`, leaving `cmd/server` as wiring. Validation and recovery remain engine-adjacent, but the transport and observability blast radius is now isolated at the package boundary. — **Validation:** `go test -race ./...`, `go vet ./...`, and `go-stats-generator` session diff.

- [ ] Go client and server packages manually mirror wire/protocol structs instead of sharing a single owner for JSON contracts — `cmd/server/game_types.go:20`, `cmd/server/game_types.go:61`, `cmd/server/game_types.go:154`, `cmd/server/game_types.go:190`, `client/ebiten/state.go:81`, `client/ebiten/state.go:91`, `client/ebiten/state.go:100`, `client/ebiten/net.go:19` — The client side is reusable, but it redefines server-facing types locally instead of importing them from a shared boundary. Because the README describes the WebSocket protocol as stable and multiple Go entrypoints already share the same client packages, protocol evolution now requires coordinated manual edits across packages and increases drift risk. I am not flagging the JS client here because that cross-language boundary legitimately needs a separate representation; the issue is the duplicated Go-to-Go contract. — **Remediation:** Introduce a shared Go protocol package, likely `protocol` unless external importability is required. Move JSON message structs and shared enums there, keep UI-only client state local to `client/ebiten`, and update both the server engine and Go client to depend on the same wire types. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections structs,packages`.

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|-----------------|
| Lack of an `internal/` directory by itself | Rejected because the README explicitly documents `cmd/server` as “entry point + game logic.” The problem is not merely missing `internal/`; it is that the current placement prevents library-forward reuse of the server engine. |
| `cmd/desktop/main.go` and `cmd/web/main.go` are fat entrypoints | Rejected because both are thin adapters that parse or derive configuration, construct `app.Game`, and hand control to Ebitengine. |
| `ConnectionWrapper` uses `*websocket.Conn`, so the net interface goal is violated | Rejected because the concrete websocket type is kept inside the provider implementation while the rest of the server works against `net.Conn`; that is an acceptable local detail. |
| Render and UI packages should expose more interfaces | Rejected because there is no demonstrated substitution seam there. Adding interfaces would be cargo-cult abstraction rather than solving a package-boundary problem. |
| No public importable server package means the project is automatically misorganized | Rejected as an absolute rule because Go’s own guidance allows app-first server repos. It becomes a finding here only because the repository also describes itself as a reusable rules engine and already benefits from library packages on the client side. |