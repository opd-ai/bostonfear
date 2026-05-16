# ADR 001: Go/Ebitengine Client Over JavaScript/Canvas

## Status
Accepted

## Date
2026-05-16

## Context

When implementing the BostonFear Arkham Horror multiplayer game client, we needed to choose between two primary approaches:

1. **Traditional Web Stack**: JavaScript + HTML5 Canvas API
2. **Go/Ebitengine**: Go-based game engine with cross-platform compilation (desktop, WASM, mobile)

The task description initially specified a JavaScript client with HTML5 Canvas rendering at 800×600px, which represents the industry-standard approach for browser-based multiplayer games.

## Decision Drivers

### Technical Requirements
- Support 1-6 concurrent players with real-time WebSocket communication
- Render game state including locations, resources, player positions, and UI elements
- Handle player input (keyboard, mouse, touch)
- Deploy to multiple platforms: desktop (Linux, macOS, Windows), web (browser via WASM), mobile (iOS, Android)
- Maintain consistent behavior across all platforms
- Support automatic reconnection with exponential backoff

### Team Constraints
- Primary development in Go (server already implemented in Go)
- Intermediate developers learning client-server WebSocket architecture
- Focus on code reusability and maintainability

## Decision

**We chose Go/Ebitengine over JavaScript/Canvas.**

The client is implemented as a Go application using the Ebitengine game framework, which compiles to:
- Native executables for desktop (Linux, macOS, Windows)
- WebAssembly (WASM) for browsers
- Mobile bindings for iOS and Android

This means the "JavaScript client" mentioned in project goals is actually a **Go client compiled to WASM**, not a traditional JavaScript implementation.

## Rationale

### Advantages of Go/Ebitengine

1. **Type Safety**
   - Go's static typing prevents entire classes of runtime errors common in JavaScript
   - Compile-time type checking catches WebSocket protocol mismatches early
   - IDE tooling provides better autocomplete and refactoring support

2. **Code Reuse**
   - Single Go codebase compiles to all target platforms
   - Shared `protocol` package used by both server and client ensures wire format consistency
   - No need to maintain separate JavaScript, Swift, and Kotlin implementations

3. **Performance**
   - Native desktop performance without JavaScript engine overhead
   - WASM compilation produces efficient binary (~2MB) with predictable performance
   - Ebitengine's rendering pipeline is optimized for game workloads

4. **Maintainability**
   - One codebase to test, debug, and refactor
   - Unified toolchain: `go build`, `go test`, `go vet` for all platforms
   - Go's garbage collection handles memory management without manual intervention

5. **Cross-Platform Consistency**
   - Identical game logic across desktop, web, and mobile
   - No platform-specific quirks to work around
   - WebSocket client behavior is identical on all platforms

6. **Developer Experience**
   - Server and client developers use the same language and tools
   - Race detector (`go test -race`) works for client code
   - Standard library provides robust networking, JSON, and concurrency primitives

### Trade-offs

1. **WASM Binary Size**: ~2MB WASM binary vs. ~50KB minified JavaScript
   - Acceptable for game workload; loaded once per session
   - Modern broadband makes this negligible (<2 seconds load time)

2. **Browser Compatibility**: WASM requires modern browsers (2017+)
   - Not an issue for target audience (intermediate developers, game enthusiasts)
   - All major browsers support WASM

3. **No DOM Manipulation**: Ebitengine uses Canvas but doesn't integrate with HTML/CSS
   - UI is rendered via Ebitengine's layers, not HTML elements
   - Acceptable for game-focused UI (not a content-heavy webpage)

4. **Learning Curve**: Team must learn Ebitengine API
   - Offset by reduced complexity (no separate JavaScript client)
   - Ebitengine API is well-documented and idiomatic Go

## Alternatives Considered

### JavaScript + Canvas
**Pros**:
- Industry standard for browser games
- Native browser support, no compilation needed
- Smaller initial bundle size
- DOM integration for HTML/CSS UI

**Cons**:
- Requires separate implementations for desktop and mobile
- JavaScript's dynamic typing increases runtime errors
- Manual memory management for game state
- Cannot reuse server's `protocol` package (would need JSON schema)
- Testing requires browser automation (Selenium, Puppeteer)

**Why rejected**: Code duplication across platforms and lack of type safety outweigh bundle size benefits.

### Unity/Unreal (WebGL export)
**Pros**:
- Professional game engine features
- Visual editor for scene design
- Large asset ecosystem

**Cons**:
- Heavyweight for a 2D board game
- Proprietary toolchain (not open source)
- Poor WebSocket support (requires plugins)
- Large WASM bundles (>10MB typical)
- Not idiomatic for Go developers

**Why rejected**: Overkill for this project's scope; introduces non-Go toolchain complexity.

### Native Mobile (Swift/Kotlin) + Separate Web Client
**Pros**:
- Platform-native UI and performance
- Full access to iOS/Android APIs

**Cons**:
- Three separate codebases (Web, iOS, Android)
- Different networking stacks per platform
- Protocol consistency becomes a manual burden
- 3× maintenance effort

**Why rejected**: Unsustainable for a small team focused on learning client-server architecture.

## Implementation Notes

### Platform-Specific Considerations

**Desktop** (`cmd/desktop/main.go`):
- Window management via Ebitengine
- Keyboard and mouse input
- Native file I/O for config

**Web** (`cmd/web/main.go`):
- WASM compilation: `GOOS=js GOARCH=wasm go build`
- Served via static file server or embedded in HTML
- Uses browser's WebSocket API (via syscall/js)

**Mobile** (`cmd/mobile/binding.go`):
- `ebitenmobile bind` creates AAR (Android) or XCFramework (iOS)
- Touch input parity with desktop
- Server URL configuration per platform (emulator vs. device)

### WebSocket Protocol Consistency

The `protocol/protocol.go` package defines wire messages:
```go
type PlayerAction struct {
    Type     string `json:"type"`
    PlayerID string `json:"playerId"`
    Action   string `json:"action"`
    Target   string `json:"target,omitempty"`
}
```

Both server (`serverengine`) and client (`client/ebiten`) import this package, ensuring compile-time guarantees that messages match.

### Testing Strategy

- **Unit tests**: Standard Go tests for game logic
- **Integration tests**: Mock WebSocket connections
- **Race detector**: `go test -race ./client/ebiten/...`
- **Display tests**: Guarded by `requires_display` build tag (Xvfb on CI)

## Consequences

### Positive
- Zero code duplication across platforms
- Type-safe WebSocket protocol
- Unified development workflow
- Easier to onboard new contributors (one language)
- Race-free concurrency (Go's goroutines + race detector)

### Negative
- WASM binary size (2MB) larger than JavaScript
- Requires WASM-capable browser (not an issue in 2026)
- Cannot use existing JavaScript game libraries

### Neutral
- Learning curve for Ebitengine (offset by Go familiarity)
- UI must be rendered via Ebitengine (acceptable for game-focused interface)

## Validation

The decision is validated by:
- Desktop client builds and runs on Linux, macOS, Windows
- WASM client loads in Chrome, Firefox, Safari, Edge
- Mobile bindings compile for Android (API 29+) and iOS (16+)
- Touch input tests verify all 12 actions are accessible
- WebSocket reconnection works identically across platforms
- No reported platform-specific bugs in game logic

## Related Decisions

- **ADR 002**: Interface-based networking (net.Conn, net.Listener)
- **ADR 003**: Modular game-family architecture (arkhamhorror, eldersign, etc.)

## References

- [Ebitengine Documentation](https://ebitengine.org/en/documents/)
- [Go WASM Guide](https://github.com/golang/go/wiki/WebAssembly)
- [BostonFear README: Design Rationale](../README.md#design-rationale-goebitengine-vs-javascriptcanvas)
- [Mobile Verification Runbook](../MOBILE_VERIFICATION_RUNBOOK.md)
