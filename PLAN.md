# Control Accessibility & Modernization Plan

**Objective**: Make the game playable without keyboard knowledge while modernizing visuals/interactions to contemporary indie game standards.

**Current State**: Game is operable via keyboard shortcuts only (1-4, G, I, W, F, R, T, C, A, E, X, N). Placeholder colored rectangles used for visuals. Touch infrastructure exists but is not visually discoverable.

---

## 1. CRITICAL — Replace Keyboard-Only Controls

### 1.1 Location Selection UI (Movement)
- [x] Implement persistent **Location Panel** showing all 4 accessible destinations
  - Display as 4 large buttons with location names + adjacency rules
  - Button states: available (normal) vs. inaccessible (disabled/greyed)
  - Placement: right side of board or collapsible sidebar
  - Replace: keyboard shortcuts 1-4 (Downtown, University, Rivertown, Northside)
- [x] Add tooltip text showing "Press 1-4" for experienced players
- [x] Ensure all location buttons are 64x64px minimum touch targets
- [x] Highlight current location with distinct visual state
- [x] Show adjacency lines/labels to teach movement restrictions

### 1.2 Action Buttons Grid (Game Actions)
- [x] Replace hardcoded bottom action grid with modern button row/radial menu
  - Gather (G) → labeled button with "gather clues" text
  - Investigate (I) → labeled button with "investigate" text
  - Ward (W) → labeled button with "cast ward" text
  - Focus (F) → labeled button with "gain focus" text
  - Research (R) → labeled button with "research" text
  - Trade (T) → labeled button with "trade" text
  - Component (C) → labeled button with "ability" text
  - Attack (A) → labeled button with "attack" text
  - Evade (E) → labeled button with "evade" text
  - Close Gate (X) → labeled button with "close gate" text
  - Encounter (N) → labeled button with "draw" text
  - Replace: keyboard shortcuts G, I, W, F, R, T, C, A, E, X, N
- [x] Button styling: consistent size (40-50px), rounded corners, clear labels + icons
- [x] Button states: enabled/disabled/hovered/pressed with distinct visuals
- [x] Tooltip: show keyboard shortcut on hover (e.g., "Press G")                                                                                                                                                                                                                                                                                                                                                qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqtttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttttt1p;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;;grr
- [x] Visibility: always visible during player's turn, disabled/greyed when unavailable
- [x] Mobile: arrange in 2-3 rows (portrait) or single row (landscape)
- [x] Desktop: horizontal row at bottom or vertical column on right side

### 1.3 Character Selection UI
- [x] Redesign investigator selection scene
  - Replace: keyboard shortcuts 1-6 (investigator selection)
  - Current: text list with key hints
  - Proposed: card-based interface with investigator portraits
  - Card states: available/selected/disabled with clear visual feedback
  - Add large "Select" button for each card alongside or below name
  - Replace: E/S/H difficulty shortcuts
  - Proposed: three clearly labeled difficulty buttons (Easy, Standard, Hard)
  - Add confirmation button with clear label: "Start Game" or "Confirm Selection"
  - Persist selected investigator + difficulty visually until submission
- [x] Ensure all interactive elements are 48x48px minimum on mobile

### 1.4 Camera Control UI
- [x] Add permanent camera control buttons to HUD (bottom-right or top-right)
  - Left arrow: orbit counter-clockwise (replace: [)
  - Right arrow: orbit clockwise (replace: ])
  - Toggle button: switch top-down ↔ pseudo-3D (replace: V)
  - Replace: mouse wheel, middle click
  - Button styling: compact but clearly distinguishable
  - Icon-based with text labels on hover
  - Mobile: make 40x40px minimum, place in safe area
- [x] Add keyboard shortcut labels on hover for power users

### 1.5 Onboarding UI
- [x] Enhance existing tutorial flow (currently has Next/Skip buttons)
  - Verify: Next and Skip buttons are clearly visible and touchable (44x48px)
  - Add: visual step counter (e.g., "Step 2 of 5")
  - Add: highlight overlay on interactive elements being taught
  - Ensure: tutorial text is readable without relying on keyboard shortcuts
  - Replace: H character skip (add explicit "Skip Tutorial" button label)

### 1.6 Game Over UI
- [x] Redesign game over scene
  - Current: text + keyboard (Enter/Space to restart)
  - Proposed: large "Play Again" button + "Return to Lobby" button
  - Styling: prominent, clear large text (36-48pt)
  - Add: game summary stats (final doom, clues, survivors)
  - Replace: keyboard shortcuts (Enter, Space)
  - Show: win/lose condition clearly (banner + icon)

### 1.7 Connection/Lobby UI
- [x] Verify connection form is fully clickable (already has button UI)
  - Address field: label + text input + clear button
  - Name field: label + text input + clear button
  - Connect button: clearly labeled, large touch target
  - Add: status display while connecting (spinner + progress text)
  - Add: error messages with clear instructions for retry

---

## 2. HIGH — Visual UI System Foundation

### 2.1 Design Token System
- [x] Establish **color palette** for contemporary indie game aesthetic
  - Primary action color (e.g., gold/amber for Arkham theme)
  - Secondary action color (subtle accent)
  - Backgrounds: dark theme with slight gradient or pattern
  - Borders: subtle but readable contrast
  - Hover state: slight brightening or glow
  - Disabled: greyed out with reduced opacity
  - Success: green highlight
  - Danger/Doom: red accent
  - Information/Sanity: blue accent
- [x] Finalize **typography scale**
  - HUD titles: 32-48pt (turn indicator, doom counter)
  - Panel titles: 24-32pt (location panel, action panel)
  - Button labels: 18-24pt
  - Body text: 14-18pt (resource names, tooltips)
  - Caption: 12-14pt (keyboard shortcuts)
- [x] Define **spacing/padding standards**
  - Button padding: 8-12px
  - Panel margins: 16-24px
  - Grid gaps: 8-12px
- [x] Establish **rounded corner radius** (4px, 8px, 12px for hierarchy)

### 2.2 Button Component Library
- [x] Create reusable button styles (code + visuals)
  - Primary button: filled with primary color
  - Secondary button: outline or muted fill
  - Danger button: red-themed for negative actions
  - Disabled button: greyed out, no interaction hint
  - Icon + label variants
  - Size variants: small (32x32), medium (48x48), large (64x64)
  - States: default, hover, pressed, disabled, loading
- [x] Implement in `client/ebiten/ui/components.go` if not already present
- [x] Add icon assets for each button action
- [x] Test hit targets: minimum 44x44px for touch, 40x40px for mouse

### 2.3 Panel/Container Styling
- [x] Create consistent panel styles for game UI regions
  - Background: dark with subtle border
  - Title bar: slightly darker or contrasting background
  - Content padding: consistent spacing
  - Borders: 1-2px outline
  - Optional drop shadow for depth
- [x] Location panel: semi-transparent dark background, rounded corners
- [x] Action bar: gradient or solid bar at bottom/side with clear sections
- [x] Status rail (top): high contrast for visibility
  - Torn/vintage paper edge texture (optional, for Arkham atmosphere)

### 2.4 Icon Set
- [x] Design 20+ icons for all game actions and resources
  - Movement icons (location-specific symbols)
  - Action icons: gather, investigate, ward, focus, research, trade, component, attack, evade, close gate, encounter
  - Resource icons: health (heart), sanity (mind), clues (lightbulb), doom (skull)
  - Status icons: turn indicator, player avatars, connection status
  - Directional icons: arrow (for camera/navigation)
  - Camera mode icons: top-down view, pseudo-3D view
  - Difficulty icons: easy/medium/hard stars
- [x] Ensure icons are 32x32 or 64x64px, with consistent stroke weight
- [x] Test legibility at minimum sizes on various backgrounds

### 2.5 Enhanced HUD Widget Update
- [x] Upgrade doom counter display
  - Current: simple text "DOOM: X / 12"
  - Proposed: visual bar (filled segments 0-12) + numeric label
  - Colors: gradient from green (low doom) to red (high doom)
  - Animation: subtle pulse when doom increases
  - Visual proximity: positioned prominently at top-right or center-top
- [x] Upgrade resource displays
  - Health: heart icon + segmented bar (10 segments) + numeric value
  - Sanity: mind icon + segmented bar (10 segments) + numeric value
  - Clues: lightbulb icon + counter (0-5) with progress indication
  - Placement: vertically stacked on left or horizontally on top rail
  - Color coding: health=red, sanity=blue, clues=gold
- [x] Upgrade turn indicator
  - Current player: large bold text "Turn: Player Name"
  - Proposed: player avatar + name + "Actions Left: 2/2" + visual action dots
  - Highlight current player row/card visually
- [x] All HUD widgets: use design tokens (colors, fonts, spacing)

---

## 3. HIGH — Graphics Modernization

### 3.1 Board & Location Visuals
- [x] Replace placeholder 64x64 colored tiles with stylized location art
  - Downtown: urban architecture, shops, street corner
  - University: academic buildings, gothic style, library vibes
  - Rivertown: dock/warehouse area, industrial feel
  - Northside: residential area with tension/atmosphere
  - Art style: isometric or top-down view consistent across all
  - Color palette: tied to established design tokens
  - Grid overlay: subtle grid pattern to enhance clarity
- [x] Add location labels inside or below each tile
  - Font: bold, readable at 1280x720 logical resolution
  - Color: contrasting against background
  - Optional: location-specific subtitle (e.g., "Downtown - Market District")
- [x] Add subtle ambient effects
  - Fog overlay (subtle, transparency 0.1-0.2)
  - Lighting gradient to show depth
  - Optional: animated elements (fog animation loop, moving shadows)

### 3.2 Player Token Visuals
- [x] Replace placeholder colored circles with investigator-themed tokens
  - Design: circular badges with player avatar/symbol
  - Differentiation: each player color distinct + pattern/symbol overlay
  - States: normal, current turn (glow/border highlight), gone (faded)
  - Size: 40-48px on board, scale appropriately
  - Animation: subtle idle animation, larger animation when selected

### 3.3 Doom Counter Visual Enhancement
- [x] Upgrade from simple text to visual track
  - Design: 12-segment horizontal or circular track
  - Filled segments: solid color (red/dark), 1px gaps between
  - Empty segments: dark outline only
  - Color gradient: green (0-4 doom) → yellow (5-8 doom) → red (9-12 doom)
  - Animation: fade-in on increase, subtle glow at max

### 3.4 Dice Result Visualization
- [x] Design dice result display for investigation/ward actions
  - Show 3 individual dice with success/blank/tentacle symbols
  - Styling: cube or rounded square shape
  - Colors: success=green, blank=grey, tentacle=red/purple
  - Layout: horizontal row, center-aligned
  - Animation: roll animation (tumble/spin effect), fade-in result
  - Text: "Success!" / "Failed - Doom +1" below dice

### 3.5 Player Color Scheme
- [x] Define primary player colors with atmospheric variants
  - Player 1: gold/yellow → warm tone
  - Player 2: cyan/light blue → cool tone
  - Player 3: magenta/pink → warm accent
  - Player 4: lime/green → natural tone
  - Player 5: orange/coral → warm tone
  - Player 6: purple/violet → cool tone
  - Add pattern overlay (stripe, dot, hatch) for color-blind accessibility
  - Apply: player tokens, resource bars, turn indicators, chat/names

### 3.6 Font & Typography
- [x] Replace default Ebitengine font with custom web font or bitmap font
  - Suggested: serif or slab-serif for Arkham Horror theme (e.g., "Crimson Text", "Playfair Display")
  - Fallback: ensure all sizes remain readable with default
  - Sizes: implement scaling per design token system (32pt, 24pt, 18pt, 14pt, 12pt)
  - Test readability at 800x600 and 1920x1080

---

## 4. MEDIUM — Polish & Feel

### 4.1 Animations & Transitions
- [x] Button interactions
  - Hover: scale 1.05x + slight shadow
  - Press: scale 0.95x + color shift
  - Duration: 100-150ms easing (ease-out)
- [x] Action submission
  - Disable button briefly (100-200ms) after click
  - Show pending indicator (small spinner or highlight)
  - On success: flash green briefly, reset button
  - On failure: flash red, show error tooltip, re-enable
- [x] Scene transitions
  - Fade in/out between scenes (200-300ms)
  - Slide in from left/right for modals
- [x] Turn indicator
  - Highlight change: scale 1.1x for 500ms then settle
  - Color shift: brief glow effect on player change
- [x] Doom increment
  - Pulse animation on counter (200-300ms)
  - Optional: screen vignette flicker (subtle red overlay)

### 4.2 Visual Feedback for Actions
- [x] Action outcome toast/notification
  - Success: green background, white text, 2-3s duration
  - Failure: red background, white text, 2-3s duration
  - Position: center-top or bottom-right, non-blocking
  - Message: "Investigated successfully - Gained 1 clue" or "Investigation failed - Doom +1"
  - Animation: fade in (200ms), pause 2s, fade out (200ms)
- [x] Resource change feedback
  - Show delta next to resource (e.g., "+1" in green, "-2" in red)
  - Fade after 1s
  - Optional: brief numeric counter animation

### 4.3 Loading & Connection States
- [x] Spinner animation during connection
  - Design: rotating ring or animated dots
  - Color: use primary action color
  - Text: "Connecting...", "Waiting for players...", "Reconnecting..."
- [x] Connection lost indicator
  - Position: top-right or as bar overlay
  - Color: orange/warning yellow
  - Text: "Connection lost - Attempting to reconnect..."
  - Animate: subtle pulse or breath effect

### 4.4 Accessibility Enhancements
- [x] Add icon + text to all buttons (not icons alone)
- [x] Ensure 4.5:1 contrast minimum for all text on backgrounds
- [x] Implement colorblind-friendly mode
  - Option: palette swap to CVD-friendly colors
  - Icons + text labels (not color only)
  - Pattern overlays on player colors (stripes, dots)
- [x] Keyboard navigation
  - Tab cycles through buttons in logical order
  - Enter/Space activates focused button
  - Arrow keys for grid navigation (action buttons, location selection)
- [x] Focus indicators
  - 2-3px border or glow around focused element
  - Color: primary action color

### 4.5 Responsive Layout
- [x] Portrait mobile layout (320-480px)
  - Vertical location panel (1 column)
  - Horizontal action bar (2-3 rows of buttons)
  - Stacked resource displays
- [x] Landscape mobile layout (480-800px)
  - Side-by-side layout (board + reduced side panels)
  - Horizontal action bar with 3-4 buttons per row
- [x] Tablet layout (800-1200px)
  - Comfortable spacing, larger buttons
  - Board centered with side panels
- [x] Desktop layout (1200px+)
  - Full HUD with all UI visible
  - Optional: collapsible/dockable panels

---

## 5. Keyboard Shortcut Preservation (Power Users)

- [x] Maintain all existing keyboard shortcuts
  - Shortcuts remain functional but now also available via UI
  - Display shortcut hint on button hover/focus
  - Example: Location button shows "1" label on hover
  - Example: Action button shows "G" label on hover
- [x] Add keyboard focus navigation (Tab/Shift+Tab)
  - Cycle through actionable elements
  - Arrow keys for grid-based selection (locations, actions)
  - Enter/Space to activate focused element
- [x] Document shortcuts in help/settings menu
  - Quick reference guide accessible from HUD
  - Optional: remappable keybinds (low priority)

---

## 6. Implementation Phases

### Phase 1: Critical UI Compliance (Blocking)
- Location panel with 4 buttons
- Action button grid replacement (all 11 actions)
- Character select redesign
- Camera control buttons
- Ensures game is playable without keyboard knowledge
- Timeline: Week 1-2

### Phase 2: Visual Modernization (High)
- Design token system implementation
- Board + location art (or improved placeholders)
- Player token redesign
- Doom counter + resource displays upgrade
- Player color scheme with patterns
- Timeline: Week 3-4

### Phase 3: Polish & Feel (Medium)
- Button animations + transitions
- Action feedback (toasts, resource deltas)
- Loading states + connection indicators
- Accessibility enhancements (contrast, focus, colorblind)
- Responsive layout refinement
- Timeline: Week 5-6

---

## 7. Success Criteria

- [x] **Accessibility**: Game is fully playable by clicking/touching UI only. No keyboard shortcuts required.
- [x] **Visual Coherence**: All UI elements follow consistent design token system (colors, fonts, spacing).
- [x] **Discoverability**: Every action has a labeled button visible. Keyboard shortcuts shown as secondary hints.
- [x] **Touch Targets**: All interactive elements are 44x44px minimum on mobile.
- [x] **Visual Feedback**: Every action shows clear success/failure/loading state.
- [x] **Responsive**: Playable and usable on mobile (portrait/landscape), tablet, and desktop.
- [x] **Modernization**: Visual style is competitive with indie games (2024 standard). Not placeholder.
- [x] **Performance**: No FPS degradation from animation/effect additions.
- [x] **Testing**: 
  - First-time players can understand all actions without tutorial
  - Experienced players prefer UI buttons for non-trivial selections (location, action type)
  - Touch gameplay works smoothly on 3-6 player games

---

## 8. Tracking

Use this checklist to track progress per phase. Update status as items are completed.

**Phase 1 Status**: [x] Completed
**Phase 2 Status**: [x] Completed
**Phase 3 Status**: [x] Completed

---

## Appendices

### A. Files to Modify/Create

**UI Components**
- `client/ebiten/ui/components.go` — enhance button/panel/badge types
- `client/ebiten/ui/hud/hud.go` — update HUD layout with new widgets
- `client/ebiten/app/scenes.go` — redesign character select, game over scenes
- `client/ebiten/app/input.go` — maintain keyboard shortcut support alongside UI

**Styling & Theming**
- `client/ebiten/ui/theme.go` — (create) unified design token system
- `client/ebiten/ui/colors.go` — (create) named color palette
- `client/ebiten/ui/typography.go` — (create) font metrics and scales

**Assets**
- `assets/sprites.png` — update with modern art (location tiles, buttons, icons)
- `assets/visuals.yaml` — map asset files to components
- `assets/fonts/` — (create) embed custom font file

**Input & Navigation**
- `client/ebiten/app/input.go` — add keyboard focus navigation (Tab, arrows)

### B. Reference Design (Rough Sketch)

```
┌─────────────────────────────────────────────────────────────────┐
│ TURN: Player 1 | DOOM: ████░░░░░░ 4/12 | Actions: ●● | Clues: ◐◐◐ │
├────────────────────────────────────────────────────────────────┤
│                                                  ┌─ Location ─────┐
│                                                  │ □ Downtown (1) │
│        [Downtown]  [University]                │ □ University(2)│
│              ↕         ↕                         │ □ Rivertown(3)│
│        [Rivertown] [Northside]                │ □ Northside(4)│
│                                                  └────────────────┘
│         (Board area with player tokens)
│
├────────────────────────────────────────────────────────────────┤
│ [G Gather] [I Investigate] [W Ward] [F Focus] [R Research]     │
│ [T Trade]  [C Component]   [A Attack] [E Evade] [X Gate]       │
│ (+ encounter button not shown)                                   │
│ Camera: [◀] [▶] [⬍] (bottom right)                             │
└────────────────────────────────────────────────────────────────┘
```

### C. Color Palette (Example)

| Purpose | Color | Hex | Usage |
|---------|-------|-----|-------|
| Primary Action | Gold/Amber | #D4AF37 | Buttons, highlights |
| Success | Green | #2ECC71 | Positive feedback, success text |
| Danger/Doom | Red | #E74C3C | Doom counter, failure state |
| Sanity | Blue | #3498DB | Sanity bar, information |
| Clues | Yellow | #F1C40F | Clues counter, neutral |
| Background | Dark Slate | #1A1A2E | HUD backgrounds |
| Border | Light Grey | #34495E | Outlines, separators |
| Text Primary | Off-White | #ECF0F1 | Main text |
| Text Secondary | Medium Grey | #95A5A6 | Secondary text, captions |
| Disabled | Dark Grey | #7F8C8D | Disabled buttons, inactive |

### D. Typography Scale (Example)

| Size | Use | Font | Weight |
|------|-----|------|--------|
| 48pt | Turn indicator, game over banner | Slab Serif | Bold |
| 32pt | HUD panel titles | Slab Serif | Bold |
| 24pt | Button labels (large) | Sans-Serif | Semi-Bold |
| 18pt | Button labels (medium), action names | Sans-Serif | Regular |
| 14pt | Body text, resource labels, tooltips | Sans-Serif | Regular |
| 12pt | Keyboard shortcut hints, captions | Mono or Sans-Serif | Regular |

---

**Document Version**: 1.0  
**Created**: 2024-2025  
**Last Updated**: [date]  
**Owner**: Boston Fear Dev Team  
**Status**: Planning Phase
