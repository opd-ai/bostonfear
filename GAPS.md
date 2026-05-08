# Organization Gaps — 2026-05-08

## Server Engine Lives In A Command Package

- **Desired Organization**: `cmd/server` should only parse config, construct dependencies, register handlers, and start the process. Core gameplay state and rules execution should live in an importable package such as `serverengine` or `game`.
- **Current State**: The server engine lives directly in `cmd/server`, including `GameServer` (`cmd/server/game_server.go:22`), the action pipeline (`cmd/server/game_server.go:270`), state types (`cmd/server/game_types.go:154`), and mechanic files (`cmd/server/game_mechanics.go:1`, `cmd/server/actions.go:1`).
- **Impact**: Alternate hosts, integration tools, simulations, and future non-HTTP entrypoints cannot reuse the engine without first extracting code from a command package. This is the main blocker to a library-forward architecture.
- **Closing the Gap**: Create `serverengine`, move state/types/mechanics/mythos/validation into it, then update `cmd/server/main.go` and `cmd/server/game_utils.go` to compose that package. Keep the refactor incremental by moving whole files together first, then refining package boundaries. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections packages,interfaces,structs`.

## Server Interfaces Stop At The Command Boundary

- **Desired Organization**: Interfaces should be owned by the importable package that consumes them, so other packages can provide implementations and depend on contracts instead of concrete command-local types.
- **Current State**: `Broadcaster` and `StateValidator` are defined in `cmd/server/interfaces.go:7` and `cmd/server/interfaces.go:13`, then consumed by `GameServer` fields in `cmd/server/game_server.go:31` and `cmd/server/game_server.go:34`.
- **Impact**: The project does use interfaces, but only inside the command package. That limits substitution, testing from outside the package, and future composition of the server engine in a different runtime.
- **Closing the Gap**: Move the interfaces with the extracted server engine, keep concrete broadcaster and validator implementations in provider packages, and inject them via constructors. A safe sequence is: extract interfaces, move the engine package, then move provider implementations. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections interfaces,packages`.

## Server Separation Of Concerns Ends At Files

- **Desired Organization**: Separate packages should own rules/state, transport, monitoring, and recovery concerns, with dependency flow from `cmd/server` into those packages rather than lateral coordination inside one large package.
- **Current State**: The same `cmd/server` package owns mechanics (`cmd/server/game_mechanics.go:1`), connection handling (`cmd/server/connection.go:1`), dashboard logic (`cmd/server/dashboard.go:1`), metrics export (`cmd/server/metrics.go:1`), and recovery/validation (`cmd/server/error_recovery.go:10`).
- **Impact**: The package remains understandable today, but change blast radius is larger than necessary. Rules changes, observability changes, and transport changes all happen in one broad namespace, which will become harder to evolve as the server grows.
- **Closing the Gap**: After extracting the engine, split remaining code into focused packages such as `transport/ws`, `monitoring`, and `validation`. Keep HTTP route registration in `cmd/server` or a thin transport package. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections packages,functions,structs`.

## Shared Go Protocol Types Lack A Single Owner

- **Desired Organization**: Shared JSON message contracts and protocol enums should live in one Go package so the server and Go client compile against the same wire schema.
- **Current State**: The server defines protocol and state structs in `cmd/server/game_types.go:20`, `cmd/server/game_types.go:61`, `cmd/server/game_types.go:154`, and `cmd/server/game_types.go:190`, while the Ebitengine client mirrors them in `client/ebiten/state.go:81`, `client/ebiten/state.go:91`, `client/ebiten/state.go:100`, and `client/ebiten/net.go:19`.
- **Impact**: Every protocol change requires coordinated edits across packages. That raises schema-drift risk and makes the stable server/client protocol harder to evolve safely.
- **Closing the Gap**: Introduce a shared Go protocol package, likely `protocol`, move message types and shared enums there, and keep client-only view state local to `client/ebiten`. Add compatibility-focused tests around marshal/unmarshal boundaries after the move. Validate with `go build ./...`, `go test -race ./...`, and `/home/user/go/bin/go-stats-generator analyze . --sections structs,packages`.