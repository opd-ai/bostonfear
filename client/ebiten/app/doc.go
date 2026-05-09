// Package app implements the Ebitengine Game struct and UI rendering for the Arkham Horror client.
//
// This package bridges the Ebitengine game loop with Arkham Horror game state,
// handling screen updates, input processing, and scene transitions.
//
// Core Components:
//
// - Game: Implements ebiten.Game (Update, Draw, Layout methods for the main loop)
// - InputHandler: Captures keyboard, mouse, and touch input; translates to player actions
// - Scene: Abstraction layer for different UI states (connecting, waiting, playing, game-over)
// - TextUI: Immediate-mode UI rendering using NotoSans bitmap font
// - Rendering System: Board sprites, location tokens, resource overlays, doom counter
//
// Game Loop Integration:
//
// The Ebitengine runtime calls Update() and Draw() repeatedly at 60 FPS.
//
//  1. Update(): Process input, update game state from WebSocket messages, tick timers
//  2. Draw(): Render board, sprites, UI overlays, animations
//  3. Layout(): Return desired screen dimensions (1280x720 logical, scaled to display)
//
// Input Handling:
//
// InputHandler listens for:
//   - Keyboard: Arrow keys (movement), Space (action confirm), Esc (cancel)
//   - Mouse: Click on action buttons, drag to select targets
//   - Touch: Multi-pointer support on mobile; safe areas for notch/homebaroffsets
//
// Inputs are translated to PlayerActionMessage and sent via the WebSocket connection
// (client/ebiten/net.go).
//
// Scene System:
//
// Scenes represent different UI states:
//   - ConnectingScene: Display connection spinner, retry count, error messages
//   - WaitingScene: List connected players, wait for host to start game
//   - PlayingScene: Render board, locations, current player indicator, action buttons
//   - GameOverScene: Display win/lose text, final statistics, reconnect button
//
// Each scene handles its own input and rendering, enabling clean separation
// between different game phases.
//
// Rendering Layers:
//
// The render subsystem (client/ebiten/render/) manages visual layers:
//   - Layer 0: Background/board
//   - Layer 1: Location tokens and static elements
//   - Layer 2: Player tokens and moving sprites
//   - Layer 3: UI overlays (buttons, resource bars)
//   - Layer 4: Fog-of-war and shader effects
//   - Layer 5: Text and debug overlays
//
// Each layer uses an Atlas for sprite batching, reducing per-frame draw calls.
// Shaders (Kage language) handle fog, vignettes, and interactive highlights.
//
// State Synchronization:
//
// Game state flows from the server:
//  1. WebSocket receives GameState JSON
//  2. client/ebiten/net.go decodes and updates Game.state
//  3. Update() triggers scene transitions if state changed
//  4. Draw() renders the current scene using Game.state
//
// The binding between Game.state and scene rendering is immediate-mode:
// no caching, no deferred updates. This simplifies reasoning about correctness.
//
// Mobile Considerations:
//
// On iOS/Android (via ebitenmobile):
//   - Layout() returns safe-area insets for notch/home bar
//   - Touch input uses PointerID to track individual fingers
//   - Action buttons are expanded to 48x48px minimum (touch targets)
//   - Battery consumption is minimized by reducing animation FPS when idle
//
// Platform Behavior:
//
// - Desktop (Linux, macOS, Windows): Mouse and keyboard input
// - WASM (browser): Touch support, no file I/O (login tokens in sessionStorage)
// - Mobile (iOS, Android): Full touch and sensor support (accelerometer future)
//
// Testing:
//
// InputHandler and Scene methods can be tested without Ebitengine by passing
// mock InputState structures. Game struct test cases mock the net.Game interface
// to inject simulated server events.
//
// Performance Notes:
//
// - Sprites are allocated once in init(); reused each frame
// - Camera is applied via a transform matrix, not per-pixel offset
// - AnimationFrame counter drives animation state transitions every Nt frames (configurable)
// - Safe to update Game.state concurrently from WebSocket goroutine using a channel-based update
package app
