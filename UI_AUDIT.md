# MURMUR Ebitengine UI/UX Clarity Audit

## Objective
Autonomously audit all MURMUR Ebitengine-dependent UI code for defects that make the interface hard to understand, hard to navigate, or hard to trust. Prioritize whether the UI is intuitive and obvious to use for a first-time user, then identify technical issues that undermine that goal.

Produce a structured diagnostic report only. Do not modify files.

## Execution Mode
- Autonomous static audit report generation
- Read and analyze code only
- No patches, no refactors, no file writes

## Product Context
MURMUR is not a conventional app. The UI is expected to teach users a new mental model:
- The Pulse Map is the primary interface, not a feed
- Users navigate a spatial network, not a list
- Surface and Anonymous layers must remain understandable and visually distinct
- Onboarding should teach through action, not dense explanation
- The UI should make it obvious how to pan, zoom, select, publish a Wave, inspect a node, switch layers, and recover orientation

The audit should therefore focus on discoverability, feedback, interaction clarity, and progressive disclosure before lower-level polish.

## Scope
Audit only code that imports or directly participates in Ebitengine UI paths:
- github.com/hajimehoshi/ebiten/v2
- related packages: ebitenutil, inpututil, text, audio, mobile, vector
- files called by Update, Draw, Layout, onboarding screen controllers, Pulse Map interaction flows, overlay systems, scene/state routing, and input mapping that directly affect the UI

Ignore:
- unrelated backend logic
- server-side logic
- non-rendering tests unless they directly validate UI behavior
- non-UI code outside the Ebitengine interaction boundary

## Primary Audit Question
Would a new user, without prior MURMUR knowledge, be able to infer:
- where they are
- what is interactive
- how to move around
- how to perform the next meaningful action
- whether an action succeeded
- how to back out of a state or recover from confusion

If the code suggests the answer is no, treat that as a UX defect even if the underlying implementation is technically correct.

## Priority Order
1. Discoverability and mental-model clarity
2. Onboarding and first-run comprehension
3. Pulse Map navigation and orientation
4. Interaction feedback and state transitions
5. Input correctness and mode isolation
6. Layout, readability, accessibility, and responsiveness
7. Performance or resource issues that visibly harm usability

## Audit Checklist

### 1. Discoverability and Affordance
Look for UI that works only if the user already knows how it works.
- Critical actions hidden behind unlabeled gestures or invisible hotspots
- No visible cue for pan, zoom, select, compose, inspect, switch layer, or return
- Clickable elements rendered without hover, highlight, focus, pressed, or selected states
- Overlays, panels, or radial menus that appear without enough contextual explanation
- Important controls placed off to the side with no signposting
- Icons used without adjacent labels where meaning is not obvious
- Features introduced only through docs or assumptions, not the interface itself
- Empty states that fail to tell the user what to do next
- Failure to distinguish decorative animation from actionable UI

### 2. Onboarding and First-Run Clarity
Assess whether the UI teaches MURMUR's unusual model through interaction.
- Welcome, identity, mode selection, bootstrap, and completion flows that overload the user
- Screens that explain concepts abstractly without showing the related control or visual element
- Missing "what happens next" cues after identity creation, mode selection, or first bootstrap
- Hybrid/Fortress concepts introduced without enough differentiation in the UI
- No reinforcement of the user's own node, own Specter, or current layer
- First action flows that do not clearly guide the user to publish, explore, or connect
- Skipped or collapsed onboarding states that leave users in the Pulse Map without orientation
- Recovery or returning-user flows that do not clearly explain current status or next steps

### 3. Pulse Map Navigation and Orientation
Assess whether users can understand the spatial UI without guessing.
- No obvious indication of the user's current position or selected node
- Camera pan/zoom controls that lack visible affordances, hints, or reset options
- Minimap, layer blend, node detail, or overlay controls that are hard to discover
- Double-click, drag, scroll-wheel, or touch gestures required but not signposted
- Selection state that is too subtle, short-lived, or visually ambiguous
- Context loss when opening/closing panels or switching between views
- No clear way to return to ego-centric view or recover after disorientation
- Layer changes that make Surface vs Anonymous context unclear
- Motion, pulse, or effects that attract attention but do not explain significance

### 4. Interaction Feedback and State Transitions
Assess whether the UI clearly answers "what just happened?"
- Clicking or tapping produces no visible confirmation
- Transition timing obscures whether a panel opened, loading began, or input was accepted
- Scene changes, overlays, and modal states that interrupt context without explanation
- Missing loading, publishing, sealing, connecting, or routing status indicators
- Async completion that changes state after the user has moved on
- Click-through or focus leakage during transitions
- No clear success/failure feedback after key user actions
- Errors shown as logs/debug text rather than user-comprehensible feedback
- Subtle transient banners or toasts that disappear before they can be read

### 5. Input Handling and Mode Isolation
Audit correctness where it affects usability.
- Shortcuts active during text entry or composition
- Map controls active while modal, dialog, or text input is focused
- Key/mouse/touch bindings that conflict in the same state
- Pointer hit-testing not adjusted for camera, zoom, DPI, or scroll offset
- Drag, hover, or focus not reset on blur, resize, minimize, scene switch, or touch cancel
- AppendInputChars or IME flows mishandled
- Held-key repeat causing duplicate submissions or navigation
- Touch targets smaller than 44x44 logical pixels on mobile
- Gesture thresholds that make taps look like drags or vice versa

### 6. Layout, Readability, and Accessibility
Assess whether users can comfortably read and operate the UI.
- Hardcoded dimensions or positions that break on resize or non-default resolutions
- Layout() mismatches or stale maxScroll logic
- Overlapping panels, clipped content, or off-screen actions
- Long strings without truncation, wrapping, or scroll behavior
- Text/background contrast likely below WCAG AA
- Font sizes too small at common desktop or handheld scales
- Dense information clusters without visual grouping
- Overuse of debug rendering in production-facing UI
- Color-only state distinctions without shape, label, or motion reinforcement

### 7. Performance and Reliability That Harm Usability
Only flag these when they are likely to be user-visible.
- Per-frame allocations in Draw or Update causing likely hitching
- Logging or string formatting on hot paths
- Repeated image creation without reuse
- Dirty-region or redraw mistakes causing avoidable full-screen work
- Asset, sprite, or text caches without limits where memory growth can degrade UX
- Goroutines, timers, or channels tied to scenes that outlive the scene and create stale UI behavior
- Silent backpressure drops for notification or event channels that affect visible feedback

## MURMUR-Specific Review Areas
Explicitly inspect these interaction surfaces if present:
- Pulse Map core navigation
- Node selection and node detail panel behavior
- Compose / publish Wave flow
- Layer blending and Surface vs Anonymous distinction
- Minimap and viewport controls
- Onboarding screens and returning-user flows
- Search, settings, passphrase, recovery, and device-management panels
- Anonymous mechanics overlays only insofar as their UI remains understandable to a normal user

## False-Positive Controls
- Do not report issues that are purely stylistic unless they impair clarity or usability
- Do not flag intentional complexity if the UI explains it well enough in context
- Prefer user-visible defects over purely architectural critique
- If uncertain, mark the issue as Needs Runtime Validation and describe the concrete ambiguity
- Do not report backend-only issues unless they manifest directly in the UI experience

## Required Output

### Section A: Coverage
List every audited Ebitengine-relevant file with one of:
- Audited
- Skipped - not user-facing rendering/input code
- Skipped - test only
- Skipped - stub/platform guard
- Skipped - outside effective UI control path

Also confirm that no findings reference code outside the Ebitengine interaction boundary.

### Section B: Executive UX Summary
Summarize, in plain language:
- Is the UI generally obvious to use for a first-time user?
- What are the top 3 reasons a user might get confused?
- Which area most needs improvement: onboarding, navigation, feedback, or input reliability?

### Section C: Findings
Sort findings by severity descending, then file path alphabetically.

Use this exact template for each finding:

### [SEVERITY] Short description
- File: path/to/file.go#Lstart-Lend
- Category: Discoverability | Onboarding | Navigation | Feedback | Input | Layout | Text | Performance | Resource
- User Goal At Risk: What the user is trying to do
- User Impact: What the user is likely to feel, misunderstand, or fail to complete
- Problem: One-sentence defect statement
- Evidence: Concrete code pattern, branch, value, or call sequence
- Fix: Specific, testable change
- Validation: Minimal way to verify the fix in the UI

### Severity Levels
- CRITICAL: user can become stuck, lose control, or misfire destructive/irrecoverable action
- HIGH: normal users will likely get confused, fail a task, or mistrust the UI
- MEDIUM: friction, inconsistency, or edge-case failure that noticeably harms usability
- LOW: polish, defensive hardening, or minor clarity improvement

### Section D: User Journey Assessment
Assess these journeys with Pass, Fail, or Needs Runtime Validation:
- First launch: user understands what MURMUR is enough to continue
- Identity setup: user understands what they are creating and why it matters
- Mode selection: user understands Open vs Hybrid vs Fortress
- Network bootstrap: user understands whether the app is connecting and when it is ready
- Pulse Map entry: user understands where they are and what to do first
- Basic navigation: user can pan, zoom, select, and recover orientation
- First action: user can publish a Wave or inspect a node without guessing
- Layer comprehension: user understands Surface vs Anonymous context when applicable

### Section E: Category Status
For each category, either list findings or emit:
- No issues found.

Categories:
- Discoverability
- Onboarding
- Navigation
- Feedback
- Input
- Layout
- Text
- Performance
- Resource

## Acceptance Criteria
- Every Ebitengine-relevant file is audited or explicitly skipped with reason
- The report prioritizes intuitive use and first-time comprehension over generic engine advice
- All CRITICAL and HIGH findings include testable fixes and validation steps
- The audit stays inside the Ebitengine/UI interaction boundary
- The final report makes it clear whether MURMUR's UI currently teaches itself or depends on prior knowledge
