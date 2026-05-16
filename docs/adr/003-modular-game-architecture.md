# ADR 003: Modular Game-Family Architecture

## Status
Accepted

## Date
2026-05-16

## Context

The BostonFear project initially focused on implementing Arkham Horror 3rd Edition rules. However, Fantasy Flight Games publishes multiple cooperative horror investigation games with overlapping mechanics:

- **Arkham Horror 3rd Edition**: City exploration, doom counter, investigators
- **Elder Sign**: Dice-focused, museum setting, shorter games
- **Eldritch Horror**: Global scale, travel mechanics, epic scope
- **Final Hour**: Real-time cooperative, experimental mechanics

These games share common patterns (resource tracking, dice resolution, cooperative win conditions) but differ in specific rules (movement systems, action sets, scenario structures).

The question: Should we hardcode Arkham Horror rules into `serverengine`, or design a modular system that supports multiple game families?

## Decision

**We implement a modular game-family architecture where each Fantasy Flight game is a pluggable module.**

The architecture has three layers:

1. **Common Layer** (`serverengine/common`): Shared contracts and primitives
   - `Engine` interface (health checks, metrics, allowed origins)
   - `SessionHandler` interface (connection lifecycle)
   - `StateValidator` interface (game state validation)
   - Shared types: `Location`, `Resources`, `DiceResult`

2. **Game-Family Modules** (`serverengine/arkhamhorror`, `serverengine/eldersign`, etc.):
   - Each module implements `Engine` and `SessionHandler` interfaces
   - Encapsulates game-specific rules, actions, and scenarios
   - Registers itself in a global module registry

3. **Runtime Selection** (`serverengine/common/runtime`):
   - `BOSTONFEAR_GAME` environment variable selects module at startup
   - `arkhamhorror` is the default and only fully implemented module
   - Other modules are scaffolded placeholders

## Architecture

### Module Structure

```
serverengine/
├── common/
│   ├── contracts/       # Engine, SessionHandler, StateValidator interfaces
│   ├── runtime/         # Module registry and selection
│   ├── logging/         # Shared structured logging
│   └── ...
├── arkhamhorror/        # Arkham Horror 3e implementation (COMPLETE)
│   ├── module.go        # Implements Engine + SessionHandler
│   ├── rules/           # Movement, dice, resource rules
│   ├── actions/         # 12 action types
│   ├── content/         # Scenarios, investigators, encounters
│   └── README.md        # Arkham-specific documentation
├── eldersign/           # Elder Sign module (SCAFFOLDED)
│   └── module.go        # Returns "not implemented" error
├── eldritchhorror/      # Eldritch Horror module (SCAFFOLDED)
│   └── module.go        # Returns "not implemented" error
└── finalhour/           # Arkham Horror: Final Hour module (SCAFFOLDED)
    └── module.go        # Returns "not implemented" error
```

### Module Registration

Each game module registers itself in `init()`:

```go
// serverengine/arkhamhorror/module.go
func init() {
    runtime.RegisterModule("arkhamhorror", NewEngine)
}

// serverengine/eldersign/module.go
func init() {
    runtime.RegisterModule("eldersign", NewEngine)
}
```

### Runtime Selection

Server startup resolves the module:

```go
// cmd/server/main.go
gameModule := viper.GetString("server.game")  // from config or BOSTONFEAR_GAME
engine, err := runtime.LoadModule(gameModule)
if err != nil {
    log.Fatal(err)
}

// engine is an Engine interface (works for any module)
ws.SetupServer(listener, handlers)
```

### Interface Contracts

**serverengine/common/contracts/engine.go**:
```go
type Engine interface {
    SessionHandler
    HealthChecker
    MetricsProvider
    
    SetAllowedOrigins(origins []string)
}

type SessionHandler interface {
    HandleConnection(conn net.Conn, reconnectToken string) error
    HandleConnectionWithContext(ctx context.Context, conn net.Conn, token string) error
}

type HealthChecker interface {
    HealthStatus() HealthReport
}

type MetricsProvider interface {
    Metrics() map[string]interface{}
}
```

Each game module implements these interfaces, ensuring uniform behavior across transports and monitoring systems.

## Rationale

### 1. Support Multiple Game Families

Fantasy Flight's catalog has 4+ cooperative horror games. A monolithic `serverengine` hardcoded for Arkham Horror would require forking the entire codebase to support Elder Sign.

With modules:
- Add `serverengine/eldersign/module.go`
- Implement `Engine` interface
- Set `BOSTONFEAR_GAME=eldersign`
- Zero changes to `transport`, `monitoring`, or `cmd/server`

### 2. Separation of Concerns

**Without modules** (monolithic):
```go
// serverengine/actions.go
func (gs *GameServer) ProcessAction(action PlayerAction) error {
    if gs.GameType == "arkham" {
        // Arkham-specific logic
    } else if gs.GameType == "eldersign" {
        // Elder Sign-specific logic
    }
    // 500 lines of if/else per action
}
```

**With modules**:
```go
// serverengine/arkhamhorror/actions/perform.go
func Perform(action PlayerAction, state *GameState) error {
    // Only Arkham logic; no conditionals
}

// serverengine/eldersign/actions/perform.go
func Perform(action PlayerAction, state *GameState) error {
    // Only Elder Sign logic
}
```

Each module is self-contained, reducing cognitive load and merge conflicts.

### 3. Testing Isolation

**Arkham Horror tests** don't accidentally depend on Elder Sign logic:
```go
// serverengine/arkhamhorror/rules_test.go
func TestMoveAction(t *testing.T) {
    // Only tests Arkham movement rules
    // No risk of Elder Sign dice logic interfering
}
```

Each module has its own `*_test.go` files, ensuring test failures are localized.

### 4. Incremental Implementation

We can deliver value progressively:
1. ✅ Arkham Horror (fully implemented)
2. ⏳ Elder Sign (scaffolded, returns "not implemented")
3. ⏳ Eldritch Horror (scaffolded)
4. ⏳ Final Hour (scaffolded)

Scaffolded modules validate the architecture without requiring full implementation.

### 5. Configuration-Driven Deployment

Production deployments can run different modules on different servers:
```bash
# Arkham Horror server
BOSTONFEAR_GAME=arkhamhorror ./server

# Elder Sign server (when implemented)
BOSTONFEAR_GAME=eldersign ./server
```

No recompilation needed; same binary supports all modules (selected at runtime).

### 6. Shared Primitives

Common mechanics are reused across modules:
- Dice resolution (`serverengine/common/dice`)
- Resource tracking (`serverengine/common/resources`)
- State validation (`serverengine/common/validation`)

This avoids duplication while preserving flexibility for game-specific rules.

## Implementation Notes

### Default Module

`arkhamhorror` is the default if `BOSTONFEAR_GAME` is unset:

```go
// serverengine/common/runtime/registry.go
func LoadModule(name string) (contracts.Engine, error) {
    if name == "" {
        name = "arkhamhorror" // default
    }
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("module %q not registered", name)
    }
    return factory()
}
```

### Scaffolded Modules

Placeholder modules return clear errors:

```go
// serverengine/eldersign/module.go
func NewEngine() (contracts.Engine, error) {
    return nil, errors.New("Elder Sign module not yet implemented")
}
```

This prevents silent failures and documents future work.

### Content Isolation

Each module manages its own content:
- `serverengine/arkhamhorror/content/nightglass/` (Arkham scenarios)
- `serverengine/eldersign/content/base/` (Elder Sign scenarios, when implemented)
- No cross-module content dependencies

### Config Schema

`config.toml` supports module selection:
```toml
[server]
game = "arkhamhorror"  # or "eldersign", "eldritchhorror", "finalhour"
listen = ":8080"

[scenario]
default_id = "scn.nightglass.harbor-signal"  # module-specific ID
```

## Trade-offs

### Advantages
✅ **Extensibility**: Adding new games requires only a new module  
✅ **Isolation**: Arkham logic doesn't interfere with Elder Sign logic  
✅ **Testability**: Each module has independent test coverage  
✅ **Reusability**: Common primitives shared across modules  
✅ **Runtime selection**: Same binary supports multiple games  

### Disadvantages
❌ **Upfront complexity**: Requires defining interfaces before implementing modules  
❌ **Indirection**: `Engine` interface adds one layer of abstraction  
❌ **Coordination**: Shared primitives need careful API design  

### Neutral
⚖️ **Module count**: 1 implemented + 3 scaffolded (manageable at current scale)  
⚖️ **Binary size**: All modules compiled in (~10MB total)  

## Alternatives Considered

### Alternative 1: Monolithic GameServer

**Code**:
```go
// serverengine/actions.go
func (gs *GameServer) ProcessAction(action PlayerAction) error {
    if gs.gameType == "arkham" {
        return gs.processArkhamAction(action)
    } else if gs.gameType == "eldersign" {
        return gs.processElderSignAction(action)
    }
    // ...
}
```

**Pros**:
- Simpler initial implementation
- No interface abstraction

**Cons**:
- 1000+ line `actions.go` with nested conditionals
- Arkham and Elder Sign logic tightly coupled
- Tests become integration tests (test all games at once)
- Cannot deploy Arkham-only server

**Why rejected**: Violates single responsibility principle; unmaintainable beyond 2 game families.

### Alternative 2: Separate Repositories

**Code**:
```
github.com/opd-ai/bostonfear-arkham
github.com/opd-ai/bostonfear-eldersign
github.com/opd-ai/bostonfear-eldritchhorror
```

**Pros**:
- Complete isolation between games
- Independent versioning

**Cons**:
- Cannot share `protocol`, `transport`, or `monitoring` packages
- Code duplication across repos
- No unified deployment

**Why rejected**: Loses code reuse benefits; creates maintenance burden.

### Alternative 3: Plugin System (Go Plugins)

**Code**:
```go
plugin.Open("modules/arkhamhorror.so")
symbol, _ := plugin.Lookup("NewEngine")
engine := symbol.(func() Engine)()
```

**Pros**:
- Dynamic loading at runtime
- No recompilation for new modules

**Cons**:
- Go plugins are fragile (strict version matching)
- Cross-platform issues (plugins not supported on all platforms)
- Harder to test

**Why rejected**: Go plugins are experimental and not production-ready.

## Consequences

### Positive
✅ Arkham Horror fully functional with clear architecture  
✅ Elder Sign, Eldritch Horror, Final Hour scaffolded for future work  
✅ `transport` and `monitoring` work with any module (interface-based)  
✅ Tests localized to each module  
✅ Single `bostonfear` binary supports multiple games via config  

### Negative
⚠️ Requires upfront interface design (Engine, SessionHandler contracts)  
⚠️ New modules must implement 4+ interfaces  
⚠️ Shared primitives need careful versioning to avoid breaking modules  

### Neutral
⚖️ Module count: 1 complete + 3 scaffolded (expected to grow slowly)  
⚖️ Learning curve: Contributors must understand module system  

## Validation

The architecture is validated by:

1. **Arkham Horror module** fully functional:
   - 12 actions, 4 locations, 6 investigators
   - 13/13 rule systems implemented
   - Playable end-to-end

2. **Scaffolded modules** register successfully:
   - `BOSTONFEAR_GAME=eldersign` fails with clear error message
   - No runtime crashes, just "not implemented" error

3. **Interface compliance**:
   - `arkhamhorror.Engine` implements `contracts.Engine`
   - `transport/ws` works with any `Engine` implementation

4. **Tests isolated**:
   - `serverengine/arkhamhorror/rules_test.go` has zero Elder Sign dependencies
   - No test pollution across modules

## Future Work

- Implement `eldersign` module (estimated 4-6 weeks)
- Implement `eldritchhorror` module (estimated 8-10 weeks)
- Add content versioning for modules (v1, v2 scenarios)
- Support multiple modules running simultaneously (federated server)

## Related Decisions

- **ADR 001**: Go/Ebitengine client (clients connect to any module via `Engine` interface)
- **ADR 002**: Interface-based networking (modules use `net.Conn`, not WebSocket-specific types)

## References

- [Fantasy Flight Games Catalog](https://www.fantasyflightgames.com/)
- [BostonFear Module Registry](../../serverengine/common/runtime/registry.go)
- [Arkham Horror Module](../../serverengine/arkhamhorror/README.md)
- [Elder Sign Module Scaffold](../../serverengine/eldersign/module.go)
