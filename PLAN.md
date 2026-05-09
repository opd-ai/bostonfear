# BostonFear UI/UX Strategic Plan

## Assumptions and Constraints
1. This plan is design and delivery planning only; no runtime behavior changes are applied here.
2. Current UI is the Go/Ebitengine client in [client/ebiten/app/game.go](client/ebiten/app/game.go), [client/ebiten/app/scenes.go](client/ebiten/app/scenes.go), [client/ebiten/app/input.go](client/ebiten/app/input.go), and [client/ebiten/render/layers.go](client/ebiten/render/layers.go).
3. Current Ebitengine logical/render size is 800x600 and currently hard-wired in [client/ebiten/app/game.go](client/ebiten/app/game.go) and runtime setup in [cmd/desktop.go](cmd/desktop.go), [cmd/web_wasm.go](cmd/web_wasm.go).
4. Shared message contract must remain stable and extend via backward-compatible fields in [protocol/protocol.go](protocol/protocol.go).
5. Reusability must align with existing multi-game module architecture in [serverengine/common/contracts/module.go](serverengine/common/contracts/module.go), [serverengine/common/runtime/registry.go](serverengine/common/runtime/registry.go), and family modules like [serverengine/arkhamhorror/module.go](serverengine/arkhamhorror/module.go).
6. Performance constraints:
9. Desktop/WASM target 60 FPS.
10. Mobile/low-end fallback target 30 FPS with adaptive quality.
11. UI feedback latency target under 100 ms for action acknowledgement.

## Target UX Outcomes
1. First-time players can identify current turn owner, actions remaining, and available actions within 30-60 seconds.
2. UI remains readable and usable across portrait, landscape, and widescreen, with no overlap of critical gameplay controls.
3. Visual quality upgrades move from test-like appearance to production-grade atmospheric presentation.
4. Action outcomes become explicit and trustworthy: what changed, why, and what to do next.
5. New GUI systems are reusable across Arkham-family variants by swapping content adapters instead of rewriting core UI primitives.

## Workstreams

### Workstream 1: Multi-Resolution and Orientation Support
- Remediation checklist:
- [x] Problem statement: Current clients are effectively fixed-layout (800x600 assumptions in board coordinates, touch regions, and window setup), which limits readability and control quality across device classes.
- [x] Proposed changes: Introduce a resolution/orientation profile system with named profiles (phone portrait, phone landscape, tablet, desktop 16:9, desktop ultrawide).
- [x] Proposed changes: Replace absolute UI placement with anchor-based layout primitives for HUD, board, and action rail.
- [x] Proposed changes: Add safe-area handling and dynamic typography scaling.
- [x] Proposed changes: Normalize input hit-testing against logical coordinate transforms.
- [x] Reusable component candidates: `ui/layout` (viewport model, anchors, constraints, safe-area abstraction).
- [x] Reusable component candidates: `ui/scaling` (device profile resolver, text/icon scale curves).
- [x] Reusable component candidates: `ui/inputmap` (screen-to-world transform, hitbox registry).
- [x] Dependencies: Existing scene system in [client/ebiten/app/scenes.go](client/ebiten/app/scenes.go).
- [x] Dependencies: Input routing in [client/ebiten/app/input.go](client/ebiten/app/input.go).
- [x] Acceptance criteria: Portrait, landscape, and widescreen snapshots show no clipped or overlapping critical UI.
- [x] Acceptance criteria: All action targets meet minimum touch size.
- [x] Acceptance criteria: Turn, doom, and resources remain visible at all supported sizes.
- [x] Effort estimate: M
- [x] Verification steps: Add screenshot matrix test harness for profile set.
- [x] Verification steps: Manual pass on desktop, simulated mobile, and WASM viewport resize.
- [x] Verification steps: Track layout fallback warnings and verify zero critical collisions.

### Workstream 2: UI Redesign (Readability, Hierarchy, Feedback)
- Remediation checklist:
- [x] Problem statement: Current UI is functional but test-oriented (heavy text overlays, sparse hierarchy, limited visual affordances), reducing playability and confidence.
- [x] Proposed changes: Introduce a production HUD shell with three fixed zones (top status rail, center board, bottom action rail).
- [x] Proposed changes: Replace text-heavy status blocks with compact cards (turn, objective, doom, player strip).
- [x] Proposed changes: Convert transient updates into a unified notification system with action preview, submitted state, and resolved state.
- [x] Proposed changes: Implement animation language for turn transitions, dice outcomes, doom spikes, and invalid actions.
- [x] Reusable component candidates: `ui/hud` (status rail, player strip, action rail primitives).
- [x] Reusable component candidates: `ui/feedback` (toasts, confirmations, transient update queue).
- [x] Reusable component candidates: `ui/components` (badges, pills, counters, segmented bars).
- [x] Dependencies: Event stream from `gameUpdate`, `diceResult`, and `gameState` in [client/ebiten/net.go](client/ebiten/net.go).
- [x] Dependencies: Shared action semantics in [protocol/protocol.go](protocol/protocol.go).
- [x] Acceptance criteria: Current turn, actions remaining, and available actions are identifiable in under 3 seconds.
- [x] Acceptance criteria: Every user action gets immediate pending feedback and explicit result feedback.
- [x] Acceptance criteria: Invalid actions always display reason and recovery guidance.
- [x] Effort estimate: L
- [x] Verification steps: First-time-user moderated task tests for one full turn (protocol and worksheet documented in [docs/UX_FIRST_TURN_MODERATED_TEST.md](docs/UX_FIRST_TURN_MODERATED_TEST.md)).
- [x] Verification steps: Instrument time-to-first-valid-action and invalid-action retry count.
- [x] Verification steps: UX regression checklist against onboarding and clarity states.

### Workstream 3: Procedural Visual Atmosphere
- Remediation checklist:
- [x] Problem statement: Visuals currently rely on placeholders and static primitives; atmosphere is weak and inconsistent across clients.
- [x] Proposed changes: Add deterministic procedural layers (fog, grain, sigils, ambient accents) with per-scenario seeds.
- [x] Proposed changes: Expand shader pipeline for subtle scene effects via [client/ebiten/render/shaders.go](client/ebiten/render/shaders.go).
- [x] Proposed changes: Build style token packs (palette, contrast, glow, line style) for visual consistency.
- [x] Proposed changes: Add quality tiers (low/medium/high) with runtime performance throttles.
- [x] Reusable component candidates: `ui/theme` (token packs and style resolver).
- [x] Reusable component candidates: `ui/procedural` (seeded background and sigil generator).
- [x] Reusable component candidates: `ui/effects` (shader/effect orchestration and quality gating).
- [x] Dependencies: Render compositor in [client/ebiten/render/layers.go](client/ebiten/render/layers.go).
- [x] Dependencies: Atlas content in [client/ebiten/render/atlas.go](client/ebiten/render/atlas.go).
- [x] Dependencies: Scenario identity from module-specific content packages.
- [x] Acceptance criteria: Visual atmosphere is deterministic for a given seed/scenario.
- [x] Acceptance criteria: Theme consistency score passes design QA checklist.
- [x] Acceptance criteria: FPS remains within target for each quality tier.
- [x] Effort estimate: M
- [x] Verification steps: Seed determinism tests for generated layers.
- [x] Verification steps: Frame-time and memory benchmarks per quality tier.
- [x] Verification steps: Snapshot diff review for thematic consistency (documented in [docs/PROCEDURAL_VISUAL_ATMOSPHERE_VERIFICATION.md](docs/PROCEDURAL_VISUAL_ATMOSPHERE_VERIFICATION.md)).

### Workstream 4: Pseudo-3D Board Visualization (8 Directions)
- Remediation checklist:
- [x] Problem statement: Current board is strictly top-down 2D; spatial understanding and immersion are limited.
- [x] Proposed changes: Implement pseudo-3D camera projection over node/edge board representation.
- [x] Proposed changes: Support at least 8 direction presets plus smooth orbit between adjacent presets.
- [x] Proposed changes: Add camera controls for mouse, keyboard, and touch gestures.
- [x] Proposed changes: Add top-down accessibility fallback mode.
- [x] Reusable component candidates: `ui/camera` (orbit, preset snapping, reset controls).
- [x] Reusable component candidates: `ui/boardview` (projection mapper for board nodes/edges/tokens).
- [x] Reusable component candidates: `ui/labels` (occlusion-aware location and token labels).
- [x] Dependencies: Location graph contract from [protocol/protocol.go](protocol/protocol.go).
- [x] Dependencies: Input abstraction from Workstream 1.
- [x] Dependencies: Layered renderer integration in [client/ebiten/render/layers.go](client/ebiten/render/layers.go).
- [x] Acceptance criteria: User can rotate across 8 directions without losing action readability.
- [x] Acceptance criteria: Move/action interactions remain consistent regardless of camera direction.
- [x] Acceptance criteria: Top-down fallback can be toggled at runtime.
- [x] Effort estimate: L
- [x] Verification steps: Camera usability tests across desktop and touch (documented in [docs/CAMERA_USABILITY_VERIFICATION.md](docs/CAMERA_USABILITY_VERIFICATION.md)).
- [x] Verification steps: Hit-test accuracy checks in all camera presets.
- [x] Verification steps: Comparison of action error rates versus top-down mode.

### Workstream 5: General Visual Upgrades (Typography, Iconography, Motion, Contrast)
- Remediation checklist:
- [x] Problem statement: Typography, spacing, iconography, and contrast are inconsistent across current clients.
- [x] Proposed changes: Define a design token system (spacing scale, typography scale, semantic colors, motion durations, elevation).
- [x] Proposed changes: Introduce a shared icon system for actions, resources, status, and outcomes.
- [x] Proposed changes: Standardize contrast-safe palettes and non-color state cues.
- [x] Proposed changes: Refactor critical screens to consume tokens only.
- [x] Reusable component candidates: `ui/tokens` (centralized design token registry).
- [x] Reusable component candidates: `ui/icons` (vector/sprite icon map).
- [x] Reusable component candidates: `ui/motion` (transition presets and easing catalog).
- [x] Dependencies: Scene and HUD refactor from Workstream 2.
- [x] Dependencies: Render atlas pipeline from [client/ebiten/render/atlas.go](client/ebiten/render/atlas.go).
- [x] Acceptance criteria: Token adoption for all primary game screens.
- [x] Acceptance criteria: Accessibility baseline met for contrast and scalable text.
- [x] Acceptance criteria: Critical state changes perceivable without color dependence.
- [x] Effort estimate: M
- [x] Verification steps: Automated token usage linting.
- [x] Verification steps: Accessibility contrast checks.
- [x] Verification steps: Visual regression snapshots across all core scenes (documented in [docs/UI_TOKEN_ACCESSIBILITY_VERIFICATION.md](docs/UI_TOKEN_ACCESSIBILITY_VERIFICATION.md)).

### Workstream 6: General UX Upgrades (Onboarding, Turn Clarity, Action Feedback, State Visibility)
- Remediation checklist:
- [ ] Problem statement: New players must infer many mechanics from text or external docs; turn/action feedback is present but still fragmented.
- [ ] Proposed changes: Add first-session guided onboarding with mechanic callouts and optional skip.
- [ ] Proposed changes: Add turn-intent flow (actionable highlights, disabled-reason tooltips, next-best-action hints).
- [ ] Proposed changes: Add a state visibility layer for synced/unsynced status, reconnect restoration, and pending action queues.
- [ ] Proposed changes: Expand outcome feedback with delta summaries tied to doom, resources, and location changes.
- [x] Reusable component candidates: `ui/onboarding` (scripted hints, checkpoints, replay toggle).
- [x] Reusable component candidates: `ui/turn` (active-turn and actions-remaining widgets).
- [x] Reusable component candidates: `ui/state` (sync/reconnect/pending status banner).
- [x] Reusable component candidates: `ui/results` (structured action outcome panel).
- [ ] Dependencies: Reconnect/session semantics in [client/ebiten/state.go](client/ebiten/state.go).
- [ ] Dependencies: Event and protocol messages from [protocol/protocol.go](protocol/protocol.go).
- [ ] Acceptance criteria: New player completes first full turn without external docs.
- [ ] Acceptance criteria: Every action displays pending and resolved feedback with deltas.
- [ ] Acceptance criteria: Reconnect flow clearly communicates restoration status.
- [ ] Effort estimate: M
- [ ] Verification steps: Onboarding completion funnel and dropout metrics.
- [ ] Verification steps: Forced disconnect/reconnect drills.
- [ ] Verification steps: Turn comprehension and outcome comprehension user tests.

## Architecture Notes for Go Codebase Integration
1. Create a shared GUI platform package under `client/ebiten/ui` with strict interfaces.
2. Keep reusable systems game-agnostic:
3. Layout/scaling/input/camera/tokens/feedback/onboarding live in shared package.
4. Introduce game-specific adapter interfaces:
5. `BoardAdapter`: locations, adjacency, labels, icon bindings.
6. `ActionAdapter`: action metadata, costs, prerequisites, tooltip text.
7. `ThemeAdapter`: colors, typography profiles, procedural seeds, icon overrides.
8. Bind adapters per game module key from registry (same keys used by runtime module system).
9. Preserve protocol compatibility by extending optional fields in shared wire types rather than introducing parallel schemas.
10. Keep WASM launcher compatibility while the Ebitengine UI platform matures; maintain parity on turn clarity and action feedback.
11. Test strategy:
12. Unit tests for layout transforms and adapters.
13. Golden snapshot tests per scene/profile.
14. Display-tagged integration tests remain in Ebitengine app/render packages.

## Prioritized Roadmap (Now / Next / Later)

### Now (0-6 weeks)
1. Workstream 1 foundation: responsive layout/scaling/orientation support.
2. Workstream 2 clarity-first HUD redesign and unified feedback states.
3. Workstream 6 onboarding + turn/action visibility baseline.
4. Design token and reusable component scaffolding (minimum viable shared UI kit).

### Next (6-12 weeks)
1. Workstream 5 visual consistency pass with token adoption enforcement.
2. Workstream 3 procedural atmosphere v1 with quality tiers.
3. Adapterization for at least one additional Arkham-family variant to prove reusability.

### Later (12+ weeks)
1. Workstream 4 pseudo-3D board and 8-direction camera.
2. Advanced thematic packs and scenario-driven visual modulation.
3. Deeper accessibility and localization expansion.

## Risks and Mitigations
1. Risk: UI overhaul drifts from mechanics clarity.
2. Mitigation: gate releases on turn/action comprehension tests, not visual polish only.
3. Risk: Performance regressions from procedural effects and camera transforms.
4. Mitigation: strict frame-time budgets, quality tiers, and feature flags.
5. Risk: Reusability gets bypassed by game-specific shortcuts.
6. Mitigation: enforce adapter contracts and architecture review before merge.
7. Risk: Migration churn between HTML and Ebitengine clients.
8. Mitigation: define parity checklist and keep protocol-compatible, incremental rollouts.

## Validation and Measurement
1. Usability metrics:
2. Time-to-identify current turn.
3. First-turn completion rate without external docs.
4. Invalid action rate per session.
5. Visual quality metrics:
6. Token coverage ratio across UI screens.
7. Contrast/accessibility pass rate.
8. Visual regression diff stability.
9. Performance metrics:
10. FPS median and p95 by device class.
11. Input-to-feedback latency.
12. Memory growth over 15-minute sessions.
13. Reusability metrics:
14. Shared-component adoption percentage across game variants.
15. Time to spin up second variant UI using adapters.
16. Adapter contract test pass rate across modules.
