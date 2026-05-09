# API AUDIT — May 9, 2026

## API Surface Summary

**Module:** `github.com/opd-ai/bostonfear`  
**Go Version:** 1.24.1  
**Project Maturity:** Alpha (pre-v1); stated in README as "alpha — placeholder sprites" with stable WebSocket protocol  
**Target Consumers:** Game server operators, Go game client developers, WebSocket transport implementers  

**Public API Packages (30 total):**
- **protocol**: Message types, location/action/dice enums, game state DTOs  
- **serverengine**: Core GameServer, game mechanics, state validation, scenario setup  
- **serverengine/arkhamhorror, eldersign, eldritchhorror, finalhour**: Game module implementations  
- **serverengine/common**: Shared contracts, messaging, validation, observability  
- **monitoring**: Health and performance metrics HTTP handlers  
- **monitoringdata**: Metrics value types  
- **transport/ws**: WebSocket handler, server setup, route registration  
- **client/ebiten**: Go game client for desktop/WASM/mobile  
- **cmd**: CLI commands (server, desktop, web)  

**Exported Symbols:**
- 234 exported functions (86.8% documented; 13.2% lack GoDoc comments)
- 124 structs (mostly protocol DTOs or metrics types)
- 9 interfaces (Broadcaster, StateValidator, Provider, SessionEngine, etc.)
- **Note:** No `internal/` package boundary; all packages are API surface

**API Evolution Stage:** 
- v0.0.0 (no version tags in go.mod)
- Breaking changes documented only via git history (no CHANGELOG.md)
- API stability not formally guaranteed but WebSocket contract stated as stable in README

---

## API Quality Scorecard

| Category | Rating | Notes |
|----------|--------|-------|
| **Surface Minimality** | ⚠️ | No `internal/` encapsulation; implementation details (Broadcaster, ConnectionQuality, metrics types) exposed in public API |
| **Signature Design** | ⚠️ | Missing `context.Context` parameters for I/O-bound methods; no timeout/cancellation support in GameServer.Start, HandleConnection, etc. |
| **Type Design** | ✅ | Struct zero-values mostly useful; factory functions (NewGameServer) provided for non-trivial types; type aliases reduce schema drift |
| **Documentation** | ⚠️ | 86.8% of exported functions have GoDoc comments but quality average is 0.31/1.0; 31 functions undocumented; no example functions; parameter constraints, error contracts, and concurrency safety rarely documented |
| **Backward Compatibility** | ✅ | Pre-v1 project status documented; no version tags released yet; no breaking change history to audit |
| **Error Contract** | ⚠️ | ValidationError type exported with no documented error handling pattern; no sentinel errors defined; error types returned inconsistently documented |
| **Consumer Ergonomics** | ⚠️ | GameServer API requires manual channel wiring (broadcastCh, actionCh); no convenience builders; ConnectionQuality, per-connection mutexes are internal concern leaked to synchronization APIs |

---

## Findings

### CRITICAL

- [x] **Missing context.Context for I/O-bound operations** — `serverengine.GameServer.Start()` [game_server.go:168], `GameServer.HandleConnection()` [connection.go:24], `transport/ws.SetupServer()` [server.go:13] — Methods that perform network I/O (WebSocket reads, message processing) lack `context.Context` parameter, preventing graceful shutdown, cancellation, and timeout enforcement. — **Consumer Impact:** Operators cannot implement request timeouts, graceful shutdown with deadline, or cancellation propagation from higher-level contexts. — **Remediation:** 
  1. Add `ctx context.Context` as first parameter to `Start()`, `HandleConnection()`, and `SetupServer()`  
  2. In `broadcastHandler()`, respect context cancellation: `select { case <-ctx.Done(): return, case payload := <-gs.broadcastCh: ... }`  
  3. Pass context to `conn.SetReadDeadline()`, `conn.SetWriteDeadline()` derived from `ctx`  
  4. Document that caller is responsible for context lifecycle  
  5. Example: `func (gs *GameServer) Start(ctx context.Context) error { go gs.broadcastHandler(ctx); ... }`

---

### HIGH

- [x] **31 exported functions lack GoDoc comments** — Multiple files in `serverengine/` (e.g., state validator methods, connection monitoring) — Functions including `arkhamhorror.Key()`, `runtime.HandleConnection()`, and others offer no guidance on purpose, parameters, return semantics, or error conditions. — **Consumer Impact:** Developers integrating the game engine cannot understand exported API without reading implementation code; IDE hover documentation is empty. — **Remediation:** Add GoDoc comment lines to all 31 undocumented exported functions. Each comment must start with the function name and describe purpose, key parameters, and error contract. Example: `// HandleConnection manages a player session, accepting a net.Conn and optional reconnect token. Returns an error if registration fails or connection I/O fails.` Identify exact functions via `go doc -all ./...` filtering for `missing comment`

- [x] **No example functions for primary API entry points** — No `ExampleNewGameServer()`, `ExampleGameServer_HandleConnection()`, `ExampleSetupServer()` in any `*_example_test.go` file — Operators cannot see idiomatic usage patterns from go.dev or `go doc -all`. — **Consumer Impact:** Learning curve is high; developers resort to reading test files. — **Remediation:** Create example functions in test files:
  1. `serverengine_example_test.go`: `ExampleNewGameServer()` showing constructor, Start(), and connection handling lifecycle  
  2. `transport_ws_example_test.go`: `ExampleSetupServer()` showing listener setup, handler wiring, and route integration  
  3. `protocol_example_test.go`: `ExamplePlayerActionMessage()` and `ExampleGameState()` showing JSON marshaling round-trip  
  4. Each example must be runnable (`// Output: ...` comment or `// Output:` for silence)

- [x] **Error contract not documented for exported error types** — `serverengine.ValidationError` [error_recovery.go:10], function error returns (e.g., `GameServer.HandleConnection()` returns `error` with no documentation of which error values/types are possible) — Callers cannot distinguish recoverable errors from fatal ones or determine retry strategy. — **Consumer Impact:** Error handling code in consumers is unreliable; no way to use `errors.Is()` or `errors.As()` without reading source code. — **Remediation:**
  1. Document ValidationError struct with `// Severity field may be "CRITICAL", "HIGH", "MEDIUM", "LOW"; callers should stop processing on CRITICAL errors.`  
  2. For functions returning error, add documentation:
     - `GameServer.HandleConnection()`: `// Returns error if connection I/O fails, player count exceeds MaxPlayers, or state validation detects corruption. Specific error types (if any) are not defined; callers should log and close the connection.`  
     - Export sentinel errors if needed: `var ErrGameFull = errors.New("game is full"); var ErrInvalidPlayer = errors.New("player not found")`  
  3. Update NewGameServer docs to describe initialization failure modes.

- [x] **No concurrency safety documented on exported interfaces and types** — `serverengine.Broadcaster`, `StateValidator`, `GameServer` struct [game_server.go:23] lack concurrency safety documentation — Consumers cannot determine if methods are safe to call from multiple goroutines, which mutexes protect which fields, or what happens under concurrent access. — **Consumer Impact:** Concurrent client code may deadlock or corrupt state silently. — **Remediation:** Add concurrency safety documentation to each exported interface and struct:
  1. `Broadcaster`: `// Broadcast is safe for concurrent use and must be called from multiple goroutines (action handler and WebSocket connections).`  
  2. `StateValidator`: `// Methods are safe for concurrent use; implementations must hold no locks across method calls.`  
  3. `GameServer`: Document which methods require external synchronization (none; all use internal mutex) and which fields are accessed atomically. Example: `// Start, HandleConnection, SetAllowedOrigins are safe for concurrent use. Internal game state mutations are synchronized via internal mutex; external callers need not coordinate.`

- [x] **Undocumented parameter constraints and defaults** — Most function parameters lack constraints documentation (e.g., `reconnectToken` in `HandleConnection` can be empty string, `locations` in `SetAllowedOrigins` are normalized but behavior on invalid input is not documented) — Consumers must test to discover behavior (empty string handling, nil safety, length bounds, etc.). — **Consumer Impact:** Developers pass invalid arguments and rely on trial-and-error testing. — **Remediation:**
  1. Document `SetAllowedOrigins(origins []string)` behavior: "Each entry is lowercased, trimmed, and empty strings are silently dropped. A nil or empty slice disables origin validation (allows any origin). Returns no error; invalid entries do not cause failure."  
  2. Document `HandleConnection(conn net.Conn, reconnectToken string)` behavior: "If reconnectToken is empty, a new player is registered. If non-empty and valid, the player's session is restored. Non–existent tokens cause new player registration. conn must be non-nil and readable/writable."  
  3. Use parameter names to clarify intent (already done) and document bounds in comment.

---

### MEDIUM

- [x] **Exposed internal coordination concerns via public API** — `serverengine/game_server.go` exposes `wsWriteMu`, `latencySamples`, `playerSessions` internal state details in `GameServer` struct definition (lines 23–100) — Consumers might inspect or rely on these fields, creating fragile coupling to implementation details. — **Consumer Impact:** API consumers may depend on internal structure; refactoring internals breaks their code. — **Remediation:** Unexport fields that are purely internal (`wsWriteMu`, `latencyHead`, `latencySampleCount`, `latencyMu`). Methods that need to expose metrics should do so via dedicated getter functions or dedicated metrics struct. Example: add `func (gs *GameServer) LatencyPercentiles() map[string]float64 { ... }` rather than exposing raw `latencySamples` array. This is backwards compatible via getter methods.

- [x] **Large exported interface definitions create implementation burden** — `transport/ws.SessionEngine` interface [server.go or implicit in handler] requires implementations to satisfy multiple methods — New transport adapters (TCP, in-process) must implement all methods even if not needed. — **Consumer Impact:** Difficult to add alternative transports or test with minimal mocks. — **Remediation:** Document SessionEngine interface contract and break into smaller role-based interfaces if needed. Alternatively, ensure interface is only 3-5 methods for minimal burden. If already necessary, document which methods are mandatory vs optional (with example no-op implementations).

- [x] **No validation guidance in public API** — `serverengine.Player` struct and `protocol.GameState` are exported JSON DTOs but lack validation documentation — Consumers constructing these structs manually cannot know valid ranges (resources 0-10, clues 0-5, locations one of four, etc.). — **Consumer Impact:** Manual state construction or test setup becomes trial-and-error. — **Remediation:** Add struct field tags or documentation:  
  1. Minimal: Update GoDoc comment on `Player` struct: `Player represents an investigator. Health and Sanity must be in [1, 10] or player is defeated; Clues must be in [0, 5]; Location must be one of Downtown, University, Rivertown, Northside.`  
  2. Medium: Add `json` tags with validation constraints in comments: `Health int // Health; range [1, 10]; defeat if 0`  
  3. Maximum: Implement `Validate() error` method on Player and GameState (not JSON-serializable but helpful for consumers).

- [x] **No documented support for late-joiner scenarios** — README states "late joiners enter the turn rotation automatically" but no exported API documents how a late joiner connects or what game state they receive — Consumers implementing late-joiner support must infer implementation from code inspection. — **Consumer Impact:** Operators may misconfigure late-joiner support or overlook session persistence requirements. — **Remediation:** Document the reconnection scenario in `GameServer` package comment or a dedicated example:  
   1. `GameServer` comment: Add section "Session Recovery: Use HandleConnection(conn, reconnectToken) to restore a disconnected player's session. If the player's token is not found, a new player is registered. Late joiners always spawn at Downtown (default location)."`  
   2. Create `ExampleGameServer_LateJoiner()` example showing: new player connects → server sends reconnectToken → player disconnects → player reconnects with token → server restores state.

- [x] **Missing package-level documentation on serverengine submodules** — `serverengine/arkhamhorror`, `eldersign`, `eldritchhorror`, `finalhour` lack clear `// Package arkhamhorror ...` documentation explaining purpose, game-specific mechanics, and how to use NewEngine() or custom scenario setup — Consumers don't know which module implements which game or how to switch between them. — **Consumer Impact:** Operators cannot configure the game selection via CLI flags or code without trial-and-error. — **Remediation:** Add package-level comments to each game module file:
   ```go
   // Package arkhamhorror implements the core Arkham Horror 3rd Edition game rules,
   // including locations, investigators, doom counter mechanics, and Mythos phase.
   // Use arkhamhorror.NewModule().NewEngine() to create a game engine instance.
   ```
   Similarly for eldersign, eldritchhorror, finalhour describing era, mechanics, and difficulty.

---

### LOW

- [x] **No internal/ boundary enforcement** — All packages are public API surface; no `internal/` packages shelter implementation details — Over time, API surface becomes harder to refactor. — **Consumer Impact:** Low risk for v0 projects; acceptable. — **Remediation:** Document which packages are considered stable API: protocol, serverengine (main GameServer entry point), monitoring, transport/ws. Mark others as experimental. In v1, create `internal/` subdirectories for: serverengine/arkhamhorror/internal/state, serverengine/arkhamhorror/internal/actions, etc.

- [x] **Inconsistent receiver types across related methods** — Some methods use value receiver, others pointer receiver within the same type — Inconsistent usage confuses consumers about whether methods modify state or are read-only. — **Consumer Impact:** Embedding or copying behavior is unpredictable. — **Remediation:** Audit `GameServer` methods and ensure all use pointer receiver (they mutate state). Same for all types with any mutating method. Verify compliance with: `go vet -composites ./...`. Document: "All methods on GameServer use pointer receiver (*GameServer) because they mutate shared state. When passing GameServer, always pass by pointer or store as *GameServer."

---

## False Positives Considered and Rejected

| Candidate Finding | Reason Rejected |
|-------------------|-----------------|
| "Exported protocol types create large surface area" | Rejected: Protocol types are intentionally exposed to enable JSON interoperability with non-Go clients; schema drift prevention is the documented rationale. Not a design issue. |
| "Lack of structured logging" | Rejected: Project uses stdlib `log` package; adding structured logging (e.g., slog) is a feature request, not an API design issue. |
| "Options pattern not used universally" | Rejected: API only exposes GameServer (constructor is NewGameServer(); config done via SetAllowedOrigins). Not overly parameterized. Acceptable for small surface. |
| "GameServer has many exported methods" | Rejected: Methods are orthogonal concerns (connection handling, metrics collection, health snapshots). Not a design issue; interface is reasonable. |
| "Many internal packages without `internal/` prefix" | Rejected: Pre-v1 project; enforcing `internal/` boundary is a future refactor item (noted under LOW), not a current design flaw. |
| "No middleware or plugin support" | Rejected: Out of scope for API audit; would be a feature request, not a design issue. |

---

## Summary

**Overall Health:** ⚠️ Acceptable for alpha project, but context.Context gap and documentation deficiencies should be addressed before v1 stability claim.

**Key Strengths:**
- Type aliases reduce schema drift between server and client  
- Factory functions provided for non-trivial types  
- Struct zero-values generally useful  
- Interface-based design enables testing (Broadcaster, StateValidator)  

**Key Gaps:**
1. Missing context.Context for cancellation/timeouts (CRITICAL)  
2. 31 undocumented exported functions, no examples (HIGH)  
3. Error contract not documented (HIGH)  
4. Concurrency safety not documented (HIGH)  
5. Internal implementation details exposed (MEDIUM)  

**Recommendation:** Before v1 release, address CRITICAL and HIGH findings. Consider LOW items in next major refactor.
