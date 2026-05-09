# MURMUR Ebitengine UI Transitions + Input Audit (No-Edit, Autonomous)

## Mission
Run a deep, autonomous audit of MURMUR’s Ebitengine UI/UX implementation with primary emphasis on transition quality and input handling correctness. Produce a structured diagnostic report only. Do not modify files.

## Execution Mode
- Autonomous static audit report generation
- Read/analyze code only
- No patches, no refactors, no file writes

## Repository Context (MURMUR-specific)
- MURMUR uses Ebitengine for Pulse Map rendering and UI interaction flows
- UX quality depends heavily on smooth scene transitions, stable state machines, and deterministic input behavior across keyboard, mouse, touch
- Focus on user-facing interaction surfaces first: scene changes, overlays, menus, camera navigation, text entry, and modal states

## Scope Rules
- Include only files that import or directly interact with Ebitengine packages:
  github.com/hajimehoshi/ebiten/v2 and related packages (inpututil, ebitenutil, text, audio, mobile, vector)
- Also include files that do not import Ebitengine directly but are called by Ebitengine Update/Draw/Layout paths for UI state, transitions, or input mapping
- Exclude non-UI backend logic unless it directly impacts transition timing/input behavior
- Exclude server-only and unrelated networking code

## Priority Order (strict)
1. UI transitions and animation/state continuity
2. Input handling and action routing
3. Layout/responsiveness and interaction targets
4. Text legibility and feedback clarity
5. Performance hot paths affecting smoothness
6. Resource lifecycle risks affecting UI reliability

## Audit Procedure
### Step 1: Build a UI call graph
- Identify all entry points: Update, Draw, Layout, scene/router/state manager calls
- Trace transition triggers: key/mouse/touch events, timers, async callbacks
- Trace input propagation: global handlers, scene handlers, widget-level handlers

### Step 2: Transition integrity checks (primary)
- Missing transition state machine guards (double-enter, re-entrant transitions, stale previous scene references)
- Transition interruption bugs (new scene starts before old transition completion)
- Non-monotonic animation clocks or delta misuse causing jitter/teleporting
- Easing misuse (frame-dependent step increments instead of time-based interpolation)
- Camera/scroll state leakage across scene changes
- Overlay/modal transition order bugs (z-order pops, flicker, accidental click-through during fade)
- Focus handoff bugs between scenes/widgets during enter/exit
- Async completion race conditions that switch scene after user already navigated elsewhere

### Step 3: Input correctness checks (primary)
- Mode isolation failures:
  shortcuts active while text input/composition is active
  gameplay/pan/zoom controls active while modal/dialog is open
- Binding conflicts:
  same key mapped to multiple actions in same state without deterministic priority
- Event consumption defects:
  one handler processes input but does not consume it, causing duplicate action
- Coordinate transform defects:
  pointer/touch hit-tests not adjusted for camera, zoom, DPI scaling, or scroll offset
- Edge-state handling:
  drag/hover/focus not reset on focus lost, resize, minimize, scene switch, touch cancel
- Text input buffering defects:
  AppendInputChars not fully consumed per frame
  IME/composition flow ignored or mixed with shortcut handling
- Repeat/debounce defects:
  held keys causing multiple unintended transitions
  no gating for one-shot actions
- Gesture ambiguity:
  tap interpreted as drag, drag threshold too low/high, wheel zoom and pan conflicts

### Step 4: MURMUR UX-specific checks
- Pulse Map navigation:
  pan/zoom inertia and bounds clamping consistency after transitions
- Overlay interaction:
  anonymous/surface layer overlays do not steal input unexpectedly
- Transition continuity:
  switching views preserves intentional context (selection/focus), resets only what should reset
- Interaction latency risk:
  any Update/Draw hot-path operation likely to produce visible stutter during transition/input moments

### Step 5: Secondary technical checks
- Per-frame allocations in Update/Draw (images, strings, fmt/log churn)
- Hardcoded dimensions breaking dynamic layout or touch target minimums (44x44 logical)
- Layout return mismatches and stale maxScroll clamping
- Resource lifecycle:
  goroutines/channels linked to UI scenes not shut down when scene exits

## False-Positive Controls
- Do not flag intentional patterns unless concrete user-visible risk is shown
- For each finding, cite exact evidence (condition, call order, state variable behavior)
- If uncertain, mark as Needs Runtime Validation and provide minimal reproduction scenario

## Required Output Structure
### Section A: Coverage
- List every audited Ebitengine-related file and status: Audited or Skipped (with reason)
- Confirm no findings outside Ebitengine interaction boundary

### Section B: Findings (sorted)
- Sort by severity descending, then file path alphabetically

For each finding, use this exact template:

```markdown
### [SEVERITY] Short description
- File: path/to/file.go#Lstart-Lend
- Category: Transition | Input | Layout | Text | Performance | Resource
- User Impact: One sentence describing what user sees/feels
- Problem: One sentence defect statement
- Evidence: Concrete code pattern, branch, value, or call sequence
- Fix: Specific, testable change (include guard conditions/order/state reset rules)
- Validation: How to reproduce and verify fix
```

### Severity definitions
- CRITICAL: crash, freeze, stuck input state, or unrecoverable navigation lock
- HIGH: visible transition/input bug in normal use
- MEDIUM: intermittent or edge-case UX breakage/perf hitch
- LOW: polish/defensive hardening

### Section C: Transition Quality Summary
- Transition state machine integrity: Pass/Fail with rationale
- Interruptibility and re-entrancy safety: Pass/Fail with rationale
- Animation timing consistency: Pass/Fail with rationale
- Focus and context continuity across scenes: Pass/Fail with rationale

### Section D: Input Reliability Summary
- Mode guards: Pass/Fail
- Binding conflicts resolved deterministically: Pass/Fail
- Coordinate transforms and hit-testing correctness: Pass/Fail
- Drag/hover/focus reset robustness: Pass/Fail
- Text input/IME buffering correctness: Pass/Fail

### Section E: Category Status
- Transition: findings or No issues found.
- Input: findings or No issues found.
- Layout: findings or No issues found.
- Text: findings or No issues found.
- Performance: findings or No issues found.
- Resource: findings or No issues found.

## Acceptance Criteria
- Every Ebitengine-relevant file is audited or explicitly skipped with reason
- Transition and Input categories receive the deepest analysis and most detailed evidence
- All CRITICAL/HIGH findings include testable fixes and validation steps
- No findings reference unrelated backend-only code
- Report is deterministic, ordered, and actionable
