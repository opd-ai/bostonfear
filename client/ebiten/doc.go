// Package ebiten provides a Go/Ebitengine-powered game client for Arkham Horror,
// supporting desktop, web (WASM), and mobile platforms with a shared codebase.
//
// This package implements the client-side game loop, state management, rendering,
// and network synchronization for the multiplayer server playable over WebSocket.
//
// Subpackages:
//
// - app: Game loop, scene management, input handling, UI rendering
// - net: WebSocket connection, message marshaling, reconnection logic
// - render: Sprite rendering, layer management, shader effects, animation
// - state: Local game state mirror, win/lose detection, player status
//
// Complete Client Architecture:
//
// The client is structured in layers, each with clear ownership:
//
//  1. Network Layer (net.go):
//     - Manages WebSocket connection with exponential backoff
//     - Handles reconnection using server-issued tokens
//     - Deserializes GameState messages from server
//     - Queues actions for transmission to server
//
//  2. State Layer (state.go):
//     - Holds a local copy of GameState received from server
//     - Computes win/lose conditions locally (optimistic rendering)
//     - Tracks which player is "us" (our player ID)
//     - Manages AnimationFrame and visual effect timers
//
//  3. Rendering Layer (render/*):
//     - Renders board and location tokens
//     - Displays player tokens at their locations
//     - Overlays resource bars (health, sanity, clues)
//     - Renders action buttons and status indicators
//     - Applies shader effects (fog-of-war, doom vignette)
//
//  4. Game Loop Layer (app/game.go):
//     - Implements Update() and Draw() for Ebitengine
//     - Processes input and sends actions to net layer
//     - Advances state transitions (connecting → playing → game over)
//     - Manages scene transitions (ConnectingScene, PlayingScene, etc.)
//
// Data Flow:
//
// Server → WebSocket → net.go → state.go → app/game.go → render/* → Player Display
// Player Input → app/input.go → net.go → WebSocket → Server
//
// Platform-Specific Behavior:
//
// The codebase compiles to three targets with minimal platform-specific code:
//
// - Desktop (cmd/desktop/main.go):
//   - Creates SDL2 window via Ebitengine
//   - Keyboard and mouse input
//   - Persistent session token in ~/.bostonfear/session.json
//
// - Web (cmd/web/main.go):
//   - Compiles to WebAssembly
//   - Runs in browser canvas
//   - Session token stored in browser localStorage
//   - Network operations via browser WebSocket API
//
// - Mobile (cmd/mobile/binding.go):
//   - Uses ebitenmobile to generate Android AAR and iOS framework
//   - Touch input with multi-pointer support
//   - Session token in platform-specific secure storage
//   - (Alpha: binding scaffolding exists; not verified on devices)
//
// Reconnection & Session Persistence:
//
// When the client connects, the server issues a reconnectToken. On disconnect:
//  1. Exponential backoff retry loop starts (5s, 10s, 20s, 30s, ...)
//  2. On reconnect, send token to server as ?token=<token> query parameter
//  3. Server validates token and restores player session, resources, and turn state
//  4. Client updates local state.go with restored GameState
//  5. UI transitions back to playing scene
//
// The token is persistent (stored in localStorage or file) so session survives
// browser refresh or app restart.
//
// Example Connection Flow:
//
// Desktop Build:
//
//	go run ./cmd/desktop -server ws://localhost:8080/ws
//
// Browser:
//
//	http://localhost:8080/play (or navigate to WASM bundle)
//
// Mobile:
//
//	(From Xcode or Android Studio after ebitenmobile build)
//
// Testing & Debugging:
//
// - Game.state is exported, allowing tests to inject state without server
// - net.Game interface is mockable for testing UI without WebSocket
// - Render pipeline can be tested by passing mock Assets
// - Scene tests use injected Game.state to verify rendering logic
//
// Performance Targets:
//
// - Frame rate: 60 FPS (update every ~17ms)
// - Input latency: < 16ms click → action send
// - State sync lag: ≤ 1 frame after receiving gameState
// - Memory: ~120 MB on desktop (sprites, fonts, runtime)
// - Mobile: Graceful degradation if GPU memory is limited
//
// Known Limitations:
//
// - Sprites are placeholders (alpha status)
// - Mobile hardware acceleration may vary (tested on recent devices)
// - Web (WASM) build is ~10 MB (no compression applied yet)
// - Maximum players is 6 (server-enforced in serverengine)
//
// Future Enhancements:
//
// - Animated sprite sheets for investigator tokens
// - Voice chat integration (WebRTC)
// - Mobile accelerometer input for additional actions
// - Server-side rendering for UI elements (reduce client complexity)
//
// See README.md for build instructions and ROADMAP.md for planned improvements.
package ebiten
