# Arkham Horror — Engine & Client Infrastructure Roadmap

> **Supersedes:** This document replaces the prior `ROADMAP.md` dated June 7, 2025
> (Version 1.0 — "Enhancement Roadmap" covering production ops, gameplay expansion,
> and social features). That roadmap is fully retired.

*Revision date: March 15, 2026*

---

## Non-Goals

The following are **explicitly out of scope** for every phase in this roadmap:

- **No game content creation** — no new cards, scenarios, investigators, items, or encounter decks.
- **No card or scenario data** — no JSON/YAML card definitions, no scenario scripting.
- **No lore or flavor text** — no narrative writing, codex entries, or thematic copy.

This roadmap covers **engine, client, and rules-engine infrastructure only**.

---

## Phase 1 — Ebitengine Client Foundation

### Objective

Replace the existing HTML5 Canvas / JavaScript client (`client/index.html`,
`client/game.js`) with a Go-based game client built on
[Ebitengine](https://ebitengine.org), while keeping the current `gorilla/websocket`
server completely unchanged.

### Scope Boundaries

| In Scope | Excluded |
|---|---|
| New `client/ebiten/` Go package with Ebitengine game loop | Modifications to `cmd/server/` or the WebSocket protocol |
| WebSocket client using `gorilla/websocket` from Go | Removal of the legacy `client/` HTML/JS files (retained for reference) |
| Minimal placeholder rendering (colored rectangles for locations, text labels) | Final art assets, sprites, or shaders |
| Module dependency addition (`github.com/hajimehoshi/ebiten/v2`) | Mobile or WASM build targets (Phase 3–4) |

### Technical Implementation

- Create package `client/ebiten/` containing:
  - `game.go` — implements `ebiten.Game` interface (`Update`, `Draw`, `Layout`).
  - `net.go` — `gorilla/websocket` dial, JSON encode/decode matching existing
    `gameState` / `playerAction` / `diceResult` / `connectionStatus` / `gameUpdate`
    message types.
  - `state.go` — local mirror of server game state, updated on each `gameState`
    message.
  - `input.go` — keyboard/mouse input mapping to player actions.
- Retain the existing JSON message protocol byte-for-byte; the server must not
  require any changes to support the new client.
- Implement automatic reconnection (5-second retry) matching the current JS client
  behavior.
- Add `github.com/hajimehoshi/ebiten/v2` to `go.mod` (minimum v2.7).
- Add `github.com/gorilla/websocket` as a client-side dependency (already in
  `go.mod` for the server).

### Dependencies

| Package | Purpose |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | Game loop, rendering, input |
| `github.com/gorilla/websocket` | WebSocket client (already present) |
| Go standard library (`encoding/json`, `sync`, `time`, `log`) | Protocol handling, concurrency |

### Success Criteria

1. `go build ./client/ebiten/...` compiles without errors on Go 1.24+.
2. The Ebitengine client connects to the unmodified server, sends `playerAction`
   messages, and receives `gameState` updates.
3. A placeholder game board renders in an 800×600 window showing player positions,
   resource levels, doom counter, and current-turn indicator.
4. Two instances of the Ebitengine client can join the same game and observe
   real-time state synchronization.

---

## Phase 2 — Desktop Build Target

### Objective

Produce native desktop binaries for Linux, macOS, and Windows from a single
`cmd/desktop/` entrypoint using standard `go build` cross-compilation.

### Scope Boundaries

| In Scope | Excluded |
|---|---|
| `cmd/desktop/main.go` entrypoint importing `client/ebiten` | WASM or mobile builds |
| Build tags / constraints for platform-specific code (if any) | Installer packages (`.msi`, `.dmg`, `.deb`) |
| CLI flags for server address (`-server ws://host:port`) | Auto-update mechanism |
| Static linking where feasible (CGO_ENABLED=0 on Linux/Windows) | macOS code signing / notarization |

### Technical Implementation

- Create `cmd/desktop/main.go`:
  - Parse `-server` flag (default `ws://localhost:8080/ws`).
  - Instantiate the `client/ebiten` game, pass server URL.
  - Call `ebiten.RunGame(game)`.
- Add a `Makefile` (or document `go build` invocations) for three targets:
  ```makefile
  build-linux:
      GOOS=linux GOARCH=amd64 go build -o dist/bostonfear-linux ./cmd/desktop
  build-macos:
      GOOS=darwin GOARCH=amd64 go build -o dist/bostonfear-macos ./cmd/desktop
  build-windows:
      GOOS=windows GOARCH=amd64 go build -o dist/bostonfear.exe ./cmd/desktop
  ```
- Ensure Ebitengine's platform dependencies are documented (e.g., X11/Wayland dev
  headers on Linux, Xcode command-line tools on macOS).
- Verify that the binary embeds no server code — it is a pure client.

### Dependencies

| Package / Tool | Purpose |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | Desktop windowing and rendering |
| Go cross-compilation toolchain | Multi-OS builds |
| Platform headers (Linux: `libgl1-mesa-dev`, `xorg-dev`) | CGo dependencies for Ebitengine on Linux |

### Success Criteria

1. `go build ./cmd/desktop` produces a single binary on each of Linux, macOS, and
   Windows.
2. The binary launches a native window (no browser required), connects to the
   existing Go server, and is fully playable.
3. All three platform builds pass a manual smoke test: connect, take two actions,
   observe state sync.

---

## Phase 3 — Web (WASM) Build Target

### Objective

Produce a browser-playable WebAssembly build of the Ebitengine client, served from
the existing Go server or any static file host.

### Scope Boundaries

| In Scope | Excluded |
|---|---|
| `cmd/web/main.go` entrypoint compiled with `GOOS=js GOARCH=wasm` | Service workers, offline support |
| `wasm_exec.js` integration (from `$(go env GOROOT)/misc/wasm/`) | Progressive Web App (PWA) manifest |
| HTML host page (`client/wasm/index.html`) that loads the WASM artifact | Legacy JS client maintenance |
| Optional: serve WASM files from the existing Go HTTP server | CDN deployment automation |

### Technical Implementation

- Create `cmd/web/main.go`:
  - Same initialization as `cmd/desktop/main.go` but compiled for `js/wasm`.
  - Server URL derived from `window.location` via `syscall/js` or passed as a JS
    global.
- Build command:
  ```bash
  GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web
  ```
- Copy `wasm_exec.js`:
  ```bash
  cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" client/wasm/
  ```
- Create `client/wasm/index.html`:
  - Load `wasm_exec.js`, instantiate `game.wasm`.
  - Provide a `<canvas>` element for Ebitengine's WebGL renderer.
- Optionally register a route in `cmd/server/` to serve `client/wasm/` at `/play`
  (non-breaking addition).
- Document expected WASM binary size and any `wasm-opt` post-processing.

### Dependencies

| Package / Tool | Purpose |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | WebGL rendering in WASM |
| Go WASM toolchain (`GOOS=js GOARCH=wasm`) | Compilation target |
| `wasm_exec.js` (bundled with Go) | Go runtime bootstrap in browser |
| (Optional) `wasm-opt` from Binaryen | WASM binary size optimization |

### Success Criteria

1. `GOOS=js GOARCH=wasm go build -o game.wasm ./cmd/web` succeeds.
2. Opening `client/wasm/index.html` in Chrome, Firefox, or Safari loads the game
   without plugins.
3. The WASM client connects to the server via WebSocket, sends actions, and receives
   state updates identically to the desktop client.
4. Two browser tabs running the WASM client can play a full game together.

---

## Phase 4 — Mobile Build Target

### Objective

Build installable iOS and Android applications using `ebitenmobile` (the Ebitengine
mobile binding tool) and `gomobile`.

### Scope Boundaries

| In Scope | Excluded |
|---|---|
| `cmd/mobile/` entrypoint using `ebitenmobile` binding API | App Store / Play Store submission |
| Touch input handling within the Ebitengine render loop | Push notifications |
| Screen-size-adaptive layout for phone and tablet displays | In-app purchases |
| Minimum targets: iOS 16+, Android 10+ (API 29) | Bluetooth or local-network discovery |

### Technical Implementation

- Create `cmd/mobile/mobile.go`:
  - Export a `Game` struct via `ebitenmobile` binding conventions.
  - Implement `ebitenmobile`-compatible `Update` / `Draw` / `Layout`.
- Touch input mapping:
  - Tap on location → move action.
  - Tap on action button → execute action.
  - Translate `ebiten.TouchID` events to the same action vocabulary as
    keyboard/mouse input.
- Build commands:
  ```bash
  # Android AAR
  ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile

  # iOS Framework
  ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile
  ```
- Create minimal Android Studio / Xcode wrapper projects that embed the generated
  artifact and launch the game activity / view controller.
- Document `gomobile` and `ebitenmobile` installation steps.

### Dependencies

| Package / Tool | Purpose |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | Mobile rendering and input |
| `ebitenmobile` CLI | Generate mobile framework / AAR |
| `gomobile` | Underlying Go → mobile bridge |
| Android SDK (API 29+), NDK | Android build |
| Xcode 15+, iOS 16+ SDK | iOS build |

### Success Criteria

1. `ebitenmobile bind -target android` produces an AAR that compiles into an APK
   installable on Android 10+ devices/emulators.
2. `ebitenmobile bind -target ios` produces an xcframework that compiles into an IPA
   installable on iOS 16+ devices/simulators.
3. Touch input allows a player to complete a full turn (move + investigate) on both
   platforms.
4. The mobile client connects to the server over the network and synchronizes state
   with desktop and WASM clients in the same game.

---

## Phase 5 — Enhanced Graphics & Presentation

### Objective

Upgrade the placeholder rectangle rendering from Phase 1 to a layered sprite-based
rendering system with shader effects, targeting consistent visual quality across
desktop, web, and mobile.

### Scope Boundaries

| In Scope | Excluded |
|---|---|
| Rendering layer architecture (board, tokens, UI overlays, animations) | Creating original art assets (use programmer-art placeholders) |
| Ebitengine shader (Kage) support for visual effects | 3D rendering or perspective transforms |
| Resolution and aspect-ratio handling for all three platforms | Audio system (separate effort) |
| Sprite atlas loading and frame-based animation | Narrative cutscenes or cinematics |

### Technical Implementation

- Define rendering layers (back-to-front draw order):
  1. **Board layer** — neighborhood tiles, connections, background.
  2. **Token layer** — player pawns, doom tokens, clue tokens.
  3. **Effect layer** — fog of war, location highlights, card glow (Kage shaders).
  4. **UI overlay layer** — HUD (health, sanity, clues, doom counter), action
     buttons, turn indicator, dice results.
  5. **Animation layer** — movement tweens, dice roll animation, doom increment
     flash.
- Implement a sprite atlas loader (`client/ebiten/render/atlas.go`) supporting PNG
  sprite sheets.
- Implement Kage shaders (`client/ebiten/render/shaders/`):
  - `fog.kage` — fog-of-war effect over unexplored / distant neighborhoods.
  - `glow.kage` — pulsing glow on interactive elements.
  - `doom.kage` — screen-edge vignette that intensifies as doom approaches 12.
- Resolution strategy:
  - Logical resolution: 1280×720 (16:9).
  - Ebitengine `Layout` returns logical size; framework handles scaling.
  - Mobile: respect safe-area insets via `ebiten.DeviceScaleFactor`.
- Target 60 FPS on desktop/web, 30 FPS minimum on mobile.

### Dependencies

| Package / Tool | Purpose |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | Sprite drawing, shader compilation |
| Kage shader language (built into Ebitengine) | Custom visual effects |
| Placeholder PNG assets (programmer art) | Development-time rendering |

### Success Criteria

1. All five rendering layers draw correctly with proper z-ordering on desktop, web,
   and mobile.
2. At least two Kage shaders (fog-of-war and doom vignette) compile and render
   without errors.
3. The game maintains ≥ 60 FPS on a mid-range desktop, ≥ 30 FPS on a 2022-era
   mobile device.
4. Logical resolution scales correctly to 1080p, 1440p, 4K, and common mobile
   display sizes without layout breakage.

---

## Phase 6 — Arkham Horror 3rd Edition Rules Compliance

### Objective

Bring the server-side game engine into full compliance with the Arkham Horror 3rd
Edition (AH3e) **core rulebook** mechanics (non-expansion). This phase addresses the
gaps identified in `GAPS.md` and aligns the engine with the rule systems documented
in `RULES.md`.

### Scope Boundaries

| In Scope | Excluded |
|---|---|
| Core rulebook game-loop and action sequencing | Expansion content (Under Dark Waves, Dead of Night, etc.) |
| Doom track, gate/anomaly mechanics, encounter resolution | Scenario narrative text or codex entries |
| Mythos phase: event placement, mythos token draw, agenda advancement | Investigator flavor text or personal stories |
| Dice pool modifiers (focus tokens, skill bonuses) | Card art, card layout, or print-ready assets |
| Rules-compliance automated test suite | Balancing or difficulty tuning beyond rulebook defaults |

### Technical Implementation

- Enumerate AH3e core rule systems requiring implementation or correction (reference
  `GAPS.md` and `RULES.md`):
  1. **Turn structure** — Investigator Phase → Mythos Phase cycle (current engine
     only implements Investigator Phase).
  2. **Mythos Phase** — event card draw, event placement, event spread / doom
     escalation, mythos token resolution, anomaly spawning.
  3. **Full action set** — add Focus, Trade, and Component Action to the existing
     Move / Gather / Investigate / Ward actions.
  4. **Dice pool modifiers** — focus token spend to reroll dice, skill-based pool
     size adjustment.
  5. **Anomaly / gate mechanics** — anomaly spawning, sealing via ward action with
     adjusted difficulty.
  6. **Encounter resolution** — neighborhood-specific encounter decks, skill-test
     resolution, consequence application.
  7. **Act / Agenda deck progression** — act advancement on clue thresholds, agenda
     advancement on doom accumulation, branching conditions.
  8. **Defeat and recovery** — investigator defeat when health or sanity reaches 0,
     "lost in time and space" state, recovery mechanics.
  9. **Victory / defeat conditions** — scenario-driven act completion (win), final
     agenda advancement (lose), replacing the current simple clue-threshold win.
  10. **Resource types** — add Money, Remnants, and Focus tokens alongside existing
      Health, Sanity, Clues.
- Implement a rules-compliance test suite:
  - `cmd/server/rules_test.go` (or `rules/rules_test.go` if extracted to a separate
    package).
  - One test function per rule system above (e.g., `TestMythosPhaseEventPlacement`,
    `TestDicePoolFocusModifier`, `TestActAgendaProgression`).
  - Table-driven tests where applicable.
  - Target: every core-rulebook mechanic has at least one automated test asserting
    correct behavior.
- Refactor game state types (`cmd/server/types.go`) to support:
  - `MythosCup []MythosToken`
  - `ActDeck`, `AgendaDeck` with progression tracking.
  - `Anomalies []Anomaly` on neighborhoods.
  - Extended `Player` struct with `Money`, `Remnants`, `FocusTokens`.

### Dependencies

| Package / Tool | Purpose |
|---|---|
| Go standard library (`testing`) | Rules-compliance test suite |
| Existing `cmd/server/` packages | Game engine under test |
| `RULES.md`, `GAPS.md` | Authoritative rule references |

### Success Criteria

1. 100% of AH3e core rulebook mechanics (non-expansion) are implemented in the
   server engine.
2. Zero known rules deviations — every intentional simplification is documented with
   rationale.
3. The rules-compliance test suite passes with `go test ./...` and covers all ten
   rule systems listed above.
4. A full 2-player game can be played through to a win or loss condition following
   AH3e turn structure (Investigator Phase → Mythos Phase) without manual
   workarounds.

---

## Appendix: Proposed Package Layout

```
bostonfear/
├── cmd/
│   ├── server/          # Existing — unchanged through Phase 5
│   ├── desktop/         # Phase 2 — desktop entrypoint
│   │   └── main.go
│   ├── web/             # Phase 3 — WASM entrypoint
│   │   └── main.go
│   └── mobile/          # Phase 4 — ebitenmobile binding
│       └── mobile.go
├── client/
│   ├── ebiten/          # Phase 1 — Ebitengine client package
│   │   ├── game.go
│   │   ├── net.go
│   │   ├── state.go
│   │   ├── input.go
│   │   └── render/      # Phase 5 — rendering subsystem
│   │       ├── atlas.go
│   │       ├── layers.go
│   │       └── shaders/
│   │           ├── fog.kage
│   │           ├── glow.kage
│   │           └── doom.kage
│   ├── wasm/            # Phase 3 — WASM host files
│   │   ├── index.html
│   │   ├── wasm_exec.js
│   │   └── game.wasm    # build artifact (gitignored)
│   ├── index.html       # Legacy — retained for reference
│   ├── game.js          # Legacy — retained for reference
│   └── dashboard.html   # Existing — monitoring dashboard
├── rules/               # Phase 6 (optional extraction)
│   └── rules_test.go
├── go.mod
├── go.sum
├── ROADMAP.md           # This document
├── GAPS.md
├── RULES.md
└── README.md
```

---

*This roadmap is a living document. It will be updated as phases are completed and
new infrastructure needs are identified.*

**Last Updated**: March 15, 2026
**Document Owner**: Engine & Infrastructure Team
