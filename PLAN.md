# Control Accessibility & Modernization Plan

## Phase 1 - Accessibility Blockers (CRITICAL)

### Milestone
- [x] Every gameplay and scene action is executable by click/tap with no keyboard knowledge.

### Technical Checklist
- [x] Connect form keyboard dependency (Tab, Enter, Backspace, typed input) -> clickable Address and Display Name fields with visible focus, caret, clear, and Connect button. Targets: client/ebiten/app/scenes.go, client/ebiten/app/game.go.
- [x] Investigator selection dependency (1-6) -> clickable investigator cards with selected state and confirm affordance. Targets: client/ebiten/app/scenes.go.
- [x] Difficulty dependency (E, S, H) -> visible Easy/Standard/Hard segmented control. Targets: client/ebiten/app/scenes.go.
- [x] Movement dependency (1, 2, 3, 4) -> adjacent-location move buttons/chips rendered from current position. Targets: client/ebiten/app/input.go, client/ebiten/app/game.go.
- [x] Core action dependency (G, I, W) -> visible Gather/Investigate/Ward controls with enabled/disabled reason text. Targets: client/ebiten/app/input.go, client/ebiten/app/game.go.
- [x] Advanced action dependency (F, R, T, C, A, E, X, N) -> visible Focus/Research/Trade/Component/Attack/Evade/Close Gate/Encounter controls. Targets: client/ebiten/app/input.go, client/ebiten/app/game.go.
- [x] Encounter discoverability gap (N) -> explicit Encounter button and pointer hit-box parity in the same action cluster. Targets: client/ebiten/app/input.go, client/ebiten/app/game.go.
- [x] Focus navigation fallback (Tab, Shift+Tab, Enter) -> pointer-first hover/focus/pressed state system for all actionable controls. Targets: client/ebiten/app/input.go, client/ebiten/app/game.go.
- [x] Onboarding dependency (Enter, H) -> visible Next and Skip Tutorial buttons. Targets: client/ebiten/app/scenes.go.
- [x] Camera dependency ([, ], V) -> visible orbit left/right and view toggle controls. Targets: client/ebiten/app/scenes.go, client/ebiten/app/game.go.
- [x] Game-over dependency (Enter, Space) -> visible Play Again and Return to Lobby buttons. Targets: client/ebiten/app/scenes.go.
- [x] Keep shortcuts for advanced players -> retain existing key binds and display shortcut hints on matching controls. Targets: client/ebiten/app/input.go, client/ebiten/app/doc.go.

### UX Acceptance Criteria
- [x] New players can complete Connect -> Select -> First Turn -> Game Over flow with pointer only.
- [x] No scene presents hidden keyboard-only progression.
- [x] Disabled actions always show why they are disabled.

## Phase 2 - Visual Modernization (HIGH)

### Milestone
- [ ] Interface and board visuals reach modern indie readability and atmosphere standards.

### Technical Checklist
- [x] Establish visual direction tokens (color roles, typography scale, icon style, spacing scale, elevation/surface rules). Targets: client/ebiten/ui, client/ebiten/render.
- [x] Refresh board readability (clear district boundaries, stronger interactable contrast, reduced decorative noise). Targets: client/ebiten/render, client/ebiten/app/game.go.
- [x] Upgrade entity readability (distinct silhouettes and contrast for players, gates, enemies, interactables). Targets: client/ebiten/render.
- [x] Modernize HUD hierarchy (icon-first status, grouped panels, reduced text wall, stronger turn/action emphasis). Targets: client/ebiten/app/game.go, client/ebiten/ui.
- [x] Improve atmospheric coherence (doom-reactive fog, vignette, and lighting tuned for clarity-first). Targets: client/ebiten/render/shaders, client/ebiten/render.
- [x] Add scene/camera transition system (smooth fades and camera interpolation rather than abrupt state jumps). Targets: client/ebiten/app/scenes.go, client/ebiten/app/game.go.

### UX Acceptance Criteria
- [ ] Critical game information is readable at gameplay distance in <=2 seconds.
- [ ] Players can distinguish actionable vs non-actionable elements without trial-and-error.
- [ ] Atmosphere supports tension without obscuring interactable information.

## Phase 3 - Interaction Polish (MEDIUM)

### Milestone
- [ ] Interaction quality feels responsive, consistent, and teachable across all scenes.

### Technical Checklist
- [x] Add full interaction state feedback (hover, pressed, disabled, invalid, success, failure) across all interactive controls. Targets: client/ebiten/app/game.go, client/ebiten/ui.
- [x] Add action outcome feedback loops (dice anticipation, result reveal, doom/resource delta callouts). Targets: client/ebiten/app/game.go, client/ebiten/render.
- [x] Add micro-animations for state changes (health, sanity, clues, doom, turn handoff). Targets: client/ebiten/render, client/ebiten/app/game.go.
- [x] Add contextual first-session guidance (coach marks for turn order, legal moves, action costs, objective cues). Targets: client/ebiten/app/scenes.go, client/ebiten/app/game.go.
- [x] Execute consistency pass (labels, icon semantics, timing, motion curves, error wording) across connect/select/play/game-over. Targets: client/ebiten/app/scenes.go, client/ebiten/app/game.go, client/ebiten/ui.

### UX Acceptance Criteria
- [ ] User always receives visible feedback within one frame cycle for every interaction.
- [ ] Error and invalid-action messages are immediate, specific, and actionable.
- [ ] First-time players can understand turn flow and available actions within one onboarding session.

## Success Criteria
- [ ] Playable without knowing any keyboard shortcuts.
- [ ] Every current shortcut dependency has an explicit visual replacement in the same scene and context.
- [ ] Keyboard shortcuts remain available for experienced players.
- [ ] Visual presentation is modern, coherent, and clearly readable during moment-to-moment play.
- [ ] Interaction patterns are consistent, learnable, and provide immediate visual feedback for every action.