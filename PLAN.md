# PLAN.md - Graphics Quality and Configurability Upgrade Plan for Go Game Project

## Scope and assumptions
This plan defines a practical, implementation-ready rollout to improve visual quality and configurability by loading static PNG assets from a content YAML section rather than hardcoded paths, while also improving layout responsiveness and mouse + keyboard usability.

Assumptions:
- The project is a Go game client with a render loop and input handling already in place.
- YAML content loading already exists or can be added with standard Go libraries plus a YAML package.
- PNG files are the required asset format for this phase.
- The UI currently supports at least one baseline resolution and needs broader scaling behavior.
- The team can add snapshot/image-based checks and automated tests to CI.

Dependencies:
- Asset pipeline agreement (naming, folder structure, export dimensions).
- YAML parser dependency selected and approved.
- One source of truth for content config (single YAML file or merged content packs).
- Basic test harness for rendering and input behavior.

Out of scope (for this phase):
- Runtime skin editor or mod marketplace.
- Animated sprite atlases and advanced shader effects beyond static PNG display.
- Full accessibility compliance audit (only practical interaction clarity covered here).

---

## Current-state questions to validate
1. Which rendering backend is used (Ebiten, SDL wrapper, custom OpenGL, other)?
2. How are assets loaded today (hardcoded paths, embedded files, remote fetch)?
3. Is there an existing content YAML structure that can host an assets section?
4. Which game components must be covered first (board, tokens, HUD, buttons, overlays, icons)?
5. What are target resolutions and aspect ratios (for example 1280x720, 1920x1080, 2560x1440, ultrawide)?
6. Is DPI scaling currently handled?
7. What is the current input model and command routing (action map, raw key handling, click hitboxes)?
8. Are there existing fallback visuals when an asset is missing?
9. Does the game run in fullscreen, windowed, and resizable modes?
10. What test tooling exists for rendering validation and regression detection?

---

## Phased plan

## Phase 1 - Baseline audit and target definition
Goal:
Create a clear inventory of renderable components, current layout behavior, and current input flows to avoid ambiguous implementation.

Tasks:
1. Enumerate all game components that must render static visuals.
2. Document current hardcoded asset references and where they are resolved.
3. Define resolution targets and minimum supported viewport.
4. Define input actions that must be available by mouse and keyboard.
5. Define acceptance criteria for visual clarity and usability.

Completion checklist:
- [x] Complete component inventory with owner and priority.
- [x] All current hardcoded asset references identified.
- [x] Resolution matrix approved.
- [x] Mouse and keyboard interaction list approved.
- [x] Phase acceptance criteria signed off.

Deliverables:
- Component-to-asset inventory document.
- Resolution and interaction support matrix.
- Baseline screenshots and interaction notes.

---

## Phase 2 - Asset configuration design and loader architecture
Goal:
Introduce an idiomatic, testable content-driven asset mapping layer in Go.

Tasks:
1. Define YAML assets section schema (component IDs, file path, fallback, optional scale/anchor metadata).
2. Create Go structs for YAML decoding with validation methods.
3. Add content validation rules for missing keys, duplicate IDs, invalid paths, and unsupported formats.
4. Implement asset manifest loader package with explicit error returns.
5. Add a resolver interface for asset lookup to decouple render code from file system details.
6. Add fallback behavior for missing/invalid assets (placeholder texture + warning log).
7. Add unit tests for YAML parsing and validation edge cases.

Completion checklist:
- [x] YAML schema drafted and approved.
- [x] Loader package compiles with explicit validation errors.
- [x] Resolver interface introduced and used by rendering layer.
- [x] Missing asset fallback works deterministically.
- [x] Unit tests cover happy path and failure modes.

Deliverables:
- YAML schema specification.
- Asset loader and resolver package.
- Validation and parser test suite.

---

## Phase 3 - Static PNG integration across all game components
Goal:
Render static visual assets for all prioritized game components using YAML-driven references only.

Tasks:
1. Replace hardcoded asset paths with resolver lookups.
2. Integrate asset lookup in board, entities, HUD, controls, and state overlays.
3. Ensure each component has a manifest key and explicit fallback.
4. Add startup preflight check that reports missing assets before gameplay starts.
5. Add lightweight runtime diagnostics for unresolved asset IDs.
6. Verify no direct hardcoded paths remain in gameplay rendering code.

Completion checklist:
- [x] All target components render PNG assets from YAML references.
- [x] No hardcoded component asset paths remain in render paths.
- [x] Startup preflight reports missing assets clearly.
- [x] Runtime fallback visuals are visible and non-blocking.
- [x] Coverage report confirms manifest key usage for all components.

Deliverables:
- Updated render integration using resolver-based lookups.
- Asset preflight report command or startup log output.
- Component mapping report (component ID to YAML key).

---

## Phase 4 - Layout and screen real-estate optimization
Goal:
Use screen space effectively across common resolutions with predictable, readable UI composition.

Tasks:
1. Define layout zones (playfield, player panel, actions, status, notifications).
2. Implement resolution-aware layout calculator returning rectangles for each zone.
3. Introduce scale rules based on viewport size and aspect ratio.
4. Add safe margins and minimum touch/click target dimensions.
5. Add text and icon sizing rules for readability at all target resolutions.
6. Add letterboxing or adaptive panel behavior for extreme aspect ratios.
7. Add automated layout assertions for key breakpoints.

Completion checklist:
- [ ] Layout calculator implemented and unit tested.
- [ ] Target resolutions render without overlap/clipping.
- [ ] Readability thresholds met for text and icons.
- [ ] Extreme aspect ratios handled by approved fallback strategy.
- [ ] Layout assertions run in CI.

Deliverables:
- Layout engine/module with deterministic sizing logic.
- Resolution screenshots set for each breakpoint.
- Layout regression test cases.

---

## Phase 5 - Input UX clarity for mouse and keyboard
Goal:
Provide clear, obvious, and conflict-free mouse and keyboard interactions with strong visual feedback.

Tasks:
1. Define canonical action map (primary/secondary/select/cancel/next/previous/hotkeys).
2. Implement shared action-command layer used by both mouse and keyboard.
3. Add focus model for keyboard navigation through interactable UI elements.
4. Add hover, focus, pressed, disabled, and selected visual states.
5. Add on-screen hints for available key bindings and click affordances.
6. Prevent duplicate submissions from rapid click/key repeat.
7. Resolve input conflicts with deterministic priority and clear lock states.
8. Add integration tests for action parity between mouse and keyboard flows.

Completion checklist:
- [ ] Action map documented and implemented.
- [ ] Mouse and keyboard both trigger same command path.
- [ ] Focus navigation is visible and predictable.
- [ ] Input conflict handling and debouncing verified.
- [ ] Interaction parity tests pass.

Deliverables:
- Input action map specification.
- Unified command routing implementation.
- Interaction test cases and demo capture.

---

## Phase 6 - Hardening, rollout, and observability
Goal:
Deploy safely with rollback options, monitoring, and confidence checks.

Tasks:
1. Add feature flag to switch between legacy and YAML-driven asset pipeline.
2. Add startup and runtime metrics for asset load failures and fallback usage.
3. Add visual regression checks for key scenes.
4. Add performance checks for load time and frame timing impact.
5. Execute staged rollout (internal, beta, full) with acceptance gates.
6. Prepare rollback procedure and incident response checklist.

Completion checklist:
- [ ] Feature flag available and documented.
- [ ] Asset and input telemetry visible in logs/metrics.
- [ ] Visual and performance checks pass thresholds.
- [ ] Staged rollout gates completed.
- [ ] Rollback playbook tested.

Deliverables:
- Rollout playbook.
- Metrics and alert definitions.
- Final release readiness report.

---

## Suggested YAML schema for asset mapping (concise example)
```yaml
content:
  visuals:
    version: 1
    basePath: assets/png
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/board_main.png
        scaleMode: cover
      location.downtown:
        file: locations/downtown.png
      token.investigator.default:
        file: tokens/investigator_default.png
      hud.health.icon:
        file: hud/health.png
      hud.sanity.icon:
        file: hud/sanity.png
      hud.clues.icon:
        file: hud/clues.png
      button.endTurn.default:
        file: ui/end_turn.png
        hover: ui/end_turn_hover.png
        pressed: ui/end_turn_pressed.png
        disabled: ui/end_turn_disabled.png
    inputHints:
      keyboard:
        endTurn: Enter
        nextTarget: Tab
        cancel: Escape
      mouse:
        primary: leftClick
        secondary: rightClick
```

Schema notes:
- components keys are stable IDs used by renderer and UI systems.
- file paths are relative to basePath.
- optional state variants (hover, pressed, disabled) support clear affordance.
- placeholders.missing is mandatory to guarantee graceful degradation.

---

## Validation and QA checklist
- [ ] Every renderable component has a mapped YAML component ID.
- [ ] Game runs when one or more assets are missing, with visible placeholder and warning.
- [ ] No hardcoded asset paths remain in rendering code (verified by static scan).
- [ ] Startup preflight catches malformed YAML and unresolved files.
- [ ] All target resolutions render without overlap, clipping, or unusable empty space.
- [ ] Mouse-only flow can complete key gameplay actions.
- [ ] Keyboard-only flow can complete key gameplay actions.
- [ ] Mixed-input flow does not create duplicate actions or conflicting state.
- [ ] Focus/hover/disabled states are visually distinct and understandable.
- [ ] Frame time and load time remain within agreed thresholds after asset integration.
- [ ] Visual regression snapshots match approved baselines.
- [ ] CI includes parser tests, layout tests, and input integration tests.

---

## Risks and mitigations
1. Risk: Missing or misnamed PNG files break visuals.
   Mitigation: Startup manifest preflight, strict YAML validation, mandatory placeholder.
2. Risk: Layout regressions on uncommon aspect ratios.
   Mitigation: Deterministic layout calculator, breakpoint tests, screenshot regression checks.
3. Risk: Mouse and keyboard command conflicts.
   Mitigation: Single command-routing layer, explicit input priority rules, debounce/lock protections.
4. Risk: Asset load performance spikes and stutter.
   Mitigation: Preload critical assets, lazy-load non-critical assets, cache decoded textures.
5. Risk: Inconsistent component IDs between code and YAML.
   Mitigation: Centralized constants/enums for IDs, compile-time references where possible, manifest lint check.
6. Risk: Large rollout causes broad regressions.
   Mitigation: Feature flag, staged rollout, clear rollback playbook.

---

## Success criteria
1. Static PNG visuals are displayed for all in-scope game components using YAML-configured references only.
2. Asset selection and overrides are driven by content YAML and validated at startup with clear errors.
3. UI uses screen space effectively across approved resolutions without overlap/clipping and with readable controls.
4. Mouse and keyboard interactions are both fully supported, discoverable, and consistent through a shared command path.
5. Automated tests cover parser validation, layout breakpoints, and input parity, and pass in CI.
6. Rollout includes monitoring, fallback behavior, and rollback readiness with no major clarification needed by implementers.
