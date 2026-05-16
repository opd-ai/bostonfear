# Contributing to BostonFear

Thank you for your interest in contributing to BostonFear! This guide will help you get started with development, testing, and submitting contributions.

## Table of Contents
1. [Quick Start](#quick-start)
2. [Development Workflow](#development-workflow)
3. [Code Style](#code-style)
4. [Testing](#testing)
5. [Pull Request Process](#pull-request-process)
6. [Architecture Overview](#architecture-overview)
7. [Getting Help](#getting-help)

## Quick Start

### Prerequisites
- Go 1.24+ installed
- Git configured with your GitHub credentials
- (Optional) Xvfb for display tests on Linux

### Setup (< 10 minutes)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/opd-ai/bostonfear.git
   cd bostonfear
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run tests**:
   ```bash
   go test -race ./...
   ```

4. **Start the server**:
   ```bash
   go run . server
   ```

5. **Run the desktop client** (in another terminal):
   ```bash
   go run ./cmd/desktop -server ws://localhost:8080/ws
   ```

You're ready to contribute!

## Development Workflow

### 1. Create a Branch

Always work on a feature branch:
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number-description
```

Branch naming conventions:
- `feature/` for new features
- `fix/` for bug fixes
- `docs/` for documentation updates
- `refactor/` for code refactoring

### 2. Make Your Changes

Follow the [Code Style](#code-style) guidelines below.

### 3. Test Your Changes

```bash
# Run all tests with race detector
go test -race ./...

# Run specific package tests
go test -race ./serverengine/...

# Run display tests (requires X server or Xvfb)
go test -race -tags=requires_display ./client/ebiten/app/... ./client/ebiten/render/...

# Run with verbose output
go test -v -race ./serverengine/...
```

### 4. Lint and Vet

```bash
# Run go vet (required)
go vet ./...

# Check gofmt
gofmt -l .

# Apply gofmt
gofmt -w .
```

### 5. Commit Your Changes

Write clear, descriptive commit messages:
```bash
git add .
git commit -m "Add structured logging with slog

- Replace log.Printf with logging.Info/Error/Warn/Debug
- Add serverengine/common/logging package
- Emit JSON-formatted logs with structured fields
- Add tests for log level configuration

Closes #123"
```

Commit message format:
- **First line**: Imperative mood, ~50 characters, capitalize first letter
- **Body**: Explain what and why (not how), wrap at 72 characters
- **Footer**: Reference issues/PRs (`Closes #123`, `Fixes #456`)

### 6. Push and Open a Pull Request

```bash
git push origin feature/your-feature-name
```

Then open a PR on GitHub with:
- **Title**: Clear, concise summary
- **Description**: 
  - What changes were made?
  - Why were they necessary?
  - How were they tested?
  - Any breaking changes?
- **Link to issue** (if applicable)

## Code Style

### Go Conventions

Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments):

1. **Naming**:
   - Exported identifiers: `GameServer`, `HandleConnection`
   - Unexported identifiers: `validateState`, `playerCount`
   - Interfaces: `-er` suffix when possible (`Broadcaster`, `HealthChecker`)
   - Acronyms: all caps (`HTTPServer`, `JSONDecoder`)

2. **Error Handling**:
   ```go
   // ✅ Good: Explicit error checking
   data, err := json.Marshal(state)
   if err != nil {
       return fmt.Errorf("marshal game state: %w", err)
   }

   // ❌ Bad: Ignoring errors
   data, _ := json.Marshal(state)
   ```

3. **Comments**:
   - Document all exported functions, types, and constants
   - Use complete sentences with proper punctuation
   - Start with the name of the thing being documented
   ```go
   // HandleConnection manages a player session via the provided net.Conn.
   // It blocks until the connection closes or an error occurs.
   func (gs *GameServer) HandleConnection(conn net.Conn, token string) error {
   ```

4. **Concurrency**:
   - Use goroutines with care; ensure proper cleanup
   - Prefer channels over shared memory
   - Use `sync.Mutex` for shared state
   - Run tests with `-race` to detect data races

### Documentation Coverage

Maintain ≥70% doc coverage (currently 81.7%):
```bash
# Check doc coverage (requires go-stats-generator)
go-stats-generator analyze . --sections documentation
```

Document:
- All exported functions, methods, types, constants
- Package-level documentation in `doc.go` files
- Non-obvious implementation decisions

### Function Complexity

Target thresholds (enforced by CI):
- **Max function length**: 30 lines (prefer smaller)
- **Max cyclomatic complexity**: 10
- **Min doc coverage**: 70%

Check complexity:
```bash
go-stats-generator analyze . --sections functions
```

## Testing

### Test Coverage

- Write tests for new functions and bug fixes
- Aim for >70% coverage on new code
- Use table-driven tests for multiple scenarios

Example:
```go
func TestValidateResources(t *testing.T) {
    tests := []struct {
        name      string
        resources Resources
        want      Resources
    }{
        {
            name:      "clamp health above max",
            resources: Resources{Health: 15, Sanity: 5, Clues: 2},
            want:      Resources{Health: 10, Sanity: 5, Clues: 2},
        },
        {
            name:      "clamp sanity below min",
            resources: Resources{Health: 5, Sanity: -1, Clues: 2},
            want:      Resources{Health: 5, Sanity: 0, Clues: 2},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gs := NewGameServer()
            gs.ValidateResources(&tt.resources)
            if tt.resources != tt.want {
                t.Errorf("got %+v, want %+v", tt.resources, tt.want)
            }
        })
    }
}
```

### Test Categories

1. **Unit Tests** (standard, always run):
   ```bash
   go test ./serverengine/...
   ```

2. **Integration Tests** (require server setup):
   ```bash
   go test ./serverengine/integration_test.go
   ```

3. **Display Tests** (require X server):
   ```bash
   # On Linux with display
   go test -tags=requires_display ./client/ebiten/app/...

   # On Linux without display (CI)
   Xvfb :99 -screen 0 1280x720x24 &
   DISPLAY=:99 go test -tags=requires_display ./client/ebiten/app/...
   ```

4. **Race Detection** (always use in CI):
   ```bash
   go test -race ./...
   ```

### Benchmark Tests

For performance-critical code:
```go
func BenchmarkBroadcastLatency(b *testing.B) {
    gs := NewGameServer()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        gs.broadcastGameState()
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./serverengine/...
```

## Pull Request Process

### Before Submitting

- [ ] All tests pass: `go test -race ./...`
- [ ] Code is formatted: `gofmt -w .`
- [ ] No vet warnings: `go vet ./...`
- [ ] Documentation updated (README, ROADMAP, ADRs if needed)
- [ ] Commit messages follow convention
- [ ] Branch is up to date with `main`

### PR Checklist

Your PR description should include:
- [ ] Summary of changes
- [ ] Motivation (why was this change needed?)
- [ ] Testing performed (unit tests, manual testing)
- [ ] Breaking changes (if any)
- [ ] Related issues/PRs

### Review Process

1. Automated CI checks run (tests, vet, benchmarks)
2. Maintainers review code for:
   - Correctness
   - Code style compliance
   - Test coverage
   - Documentation
3. Address feedback in new commits (don't force-push until approved)
4. Once approved, maintainer will merge

### CI Requirements

All PRs must pass:
- `go test -race ./...` (all tests)
- `go vet ./...` (no warnings)
- Documentation coverage ≥70%
- Benchmark latency <200ms (for broadcast changes)
- Dependency direction check (no circular imports)

## Architecture Overview

Understanding the project structure helps you contribute effectively.

### Key Packages

| Package | Purpose | Key Files |
|---------|---------|-----------|
| `serverengine` | Game logic, state management | `game_server.go`, `actions.go`, `connection.go` |
| `serverengine/arkhamhorror` | Arkham Horror rules | `module.go`, `rules/`, `actions/` |
| `serverengine/common` | Shared contracts and utilities | `contracts/`, `logging/`, `runtime/` |
| `transport/ws` | WebSocket HTTP handler | `server.go`, `websocket_handler.go` |
| `protocol` | Wire message types | `protocol.go` |
| `monitoring` | Health and metrics handlers | `handlers.go` |
| `client/ebiten` | Ebitengine game client | `game.go`, `net.go`, `render/` |

### Design Principles

1. **Interface-based networking**: Use `net.Conn`, not `*websocket.Conn` ([ADR 002](docs/adr/002-interface-based-networking.md))
2. **Modular game families**: Each game is a pluggable module ([ADR 003](docs/adr/003-modular-game-architecture.md))
3. **Go/Ebitengine client**: Cross-platform Go client, not JavaScript ([ADR 001](docs/adr/001-ebitengine-client.md))
4. **Explicit error handling**: No ignored errors, always check and wrap
5. **Structured logging**: Use `logging.Info/Error/Warn/Debug`, not `log.Printf`

### Architecture Decision Records

For major design decisions, consult [docs/adr/](docs/adr/README.md).

## Getting Help

- **Questions**: Open a [GitHub Discussion](https://github.com/opd-ai/bostonfear/discussions)
- **Bugs**: Open a [GitHub Issue](https://github.com/opd-ai/bostonfear/issues)
- **Security**: See [SECURITY.md](SECURITY.md)
- **Design decisions**: See [docs/adr/](docs/adr/README.md)

## Code of Conduct

Be respectful, constructive, and collaborative. We welcome contributors of all skill levels.

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see [LICENSE](LICENSE)).

---

**Thank you for contributing to BostonFear!** 🎲🐙
