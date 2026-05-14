# Boston Fear Arkham Horror Game UI/UX Clarity Audit

> **Note**: This audit framework references JavaScript client and HTML5 Canvas, but the project now uses Go/Ebitengine for all client platforms (desktop, WASM, mobile). This document is retained for historical reference. For current architecture, see README.md.

## Objective
Autonomously audit all Boston Fear WebSocket server and Go/Ebitengine client UI code for defects that make the game interface hard to understand, hard to navigate, or hard to trust. Prioritize whether the UI is intuitive and obvious for a first-time player unfamiliar with Arkham Horror mechanics, then identify technical issues that undermine that goal.

Produce a structured diagnostic report only. Do not modify files.

## Execution Mode
- Autonomous static audit report generation
- Read and analyze code only
- No patches, no refactors, no file writes

## Product Context
Boston Fear is a cooperative multiplayer Arkham Horror web game where players manage resources while exploring interconnected neighborhoods and facing supernatural threats. The UI must teach players:
- The Location System with 4 interconnected neighborhoods and restricted movement
- Resource management (Health, Sanity, Clues) with clear gain/loss feedback
- Action economy (2 actions per turn from defined set)
- Doom Counter progression and its impact on game state
- Dice resolution outcomes (Success/Blank/Tentacle) with difficulty thresholds

## Scope
Audit only code that participates in direct game UI rendering and player input:
- WebSocket client message handlers and game state rendering
- Ebitengine game rendering code (client/ebiten/)
- Go client input capture and validation for player actions
- Turn order and action availability display
- Resource level visualization and updates
- Dice roll result display and feedback
- Game state synchronization across concurrent players

Ignore:
- Go server backend processing logic
- WebSocket protocol implementation details
- Non-rendering utility functions
- Tests unless they validate critical UI behavior

## Primary Audit Question
Would a new player, without prior Arkham Horror knowledge, be able to infer:
- what resources they have and what they mean
- where they are located
- how many actions remain this turn
- what actions are available and what they do
- whether an action succeeded or failed
- what the doom counter represents
- whose turn is it
- how to recover from an invalid action

If the code suggests the answer is no, treat that as a UX defect.

## Priority Order
1. Discoverability and game mechanic clarity
2. First-time player onboarding and tutorial
3. Game state visibility and turn structure
4. Action feedback and resource updates
5. Input correctness and action validation
6. Layout, readability, and responsiveness
7. Performance issues that harm playability

## Audit Checklist

### 1. Discoverability and Affordance
- Actions hidden without visible buttons or labels
- No clear indication of which actions are available this turn
- Resource costs not shown before action confirmation
- Location names or descriptions unclear or missing
- Current player indicator hard to identify
- No visual distinction between player positions on map
- Doom counter significance unexplained
- Dice roll results unclear (Success vs Blank vs Tentacle not obvious)
- Win/lose conditions not displayed

### 2. Onboarding and First-Time Clarity
- Game rules explained only in external docs, not in UI
- Starting resources or player state not clearly displayed
- First turn flow confusing without step-by-step guidance
- Investigator selection or setup unclear
- No tutorial or guidance for location movement
- Action consequences (especially doom increment) not explained
- Multiplayer turn sequence confusing

### 3. Game State Visibility
- Current player position not highlighted
- Turn order unclear or hard to track
- Action count remaining not visible
- Resource levels hard to read or ambiguous
- Doom counter updates not acknowledged
- Location adjacency rules not shown to player
- Game phase (waiting for player, resolving action, turn complete) unclear
- No indication when state sync from server is complete

### 4. Action Feedback and State Transitions
- Action submission produces no confirmation
- Dice roll results displayed without clear success/failure judgment
- Resource changes not highlighted or explained
- Doom increments not clearly attributed to failed actions
- Invalid actions fail silently instead of showing error feedback
- Turn transitions not clearly announced
- Waiting for other players not indicated

### 5. Input Handling and Validation
- Click targets too small or overlapping
- Action buttons remain clickable when unavailable
- Multiple rapid clicks allow duplicate action submission
- Stale UI state after server disconnect/reconnect
- Canvas rendering coordinates not adjusted for device DPI or window resize
- Touch targets smaller than 44x44px on mobile
- Pointer events not cancelled/prevented appropriately

### 6. Layout, Readability, and Accessibility
- Canvas hardcoded to fixed resolution without scaling
- Text too small at 800x600 or larger displays
- Overlapping UI elements or clipped content
- Color-only distinctions (e.g., player colors) without shape or label reinforcement
- Poor text/background contrast
- No word wrapping on resource or location names
- Player name/ID display too small or unclear

### 7. Performance That Harms Playability
- Per-frame allocations in Canvas render loop
- Excessive DOM updates on state change
- Canvas redraw not optimized (full redraw every frame)
- Large uncompressed assets causing slow downloads
- Memory leaks from event listeners during game state updates

## Boston Fear-Specific Review Areas
- Canvas rendering of all 4 locations and player positions
- Resource display and update feedback
- Action button availability and selection UI
- Doom counter visualization and increment notification
- Dice roll result display and success/failure clarity
- Turn indicator and action counter
- Multiplayer player list and turn order
- Reconnection and sync status feedback
- Game over / win condition display

## False-Positive Controls
- Do not flag styling issues unrelated to clarity or usability
- Do not flag serverside mechanics unless they visibly impact UI feedback
- Prefer player-visible defects over purely architectural critique
- Mark uncertain issues as Needs Runtime Validation
- Do not report Go server logic unless it manifests as UI lag or silent failures

## Required Output

### Section A: Coverage
List every audited file with status: Audited, Skipped, or Skipped - reason

### Section B: Executive UX Summary
- Is the game UI obvious for a first-time player?
- What are the top 3 reasons a player might get confused?
- Which area needs improvement most: onboarding, visibility, feedback, or input?

### Section C: Findings
Sort by severity descending, then file path alphabetically.

Template:
### [SEVERITY] Short description
- File: path/to/file.js#Lstart-Lend
- Category: Discoverability | Onboarding | State Visibility | Feedback | Input | Layout | Performance
- Player Goal At Risk: What the player is trying to accomplish
- Player Impact: What the player may misunderstand or fail to do
- Problem: One-sentence defect
- Evidence: Concrete code pattern, missing element, or branch
- Fix: Specific, testable change
- Validation: How to verify the fix in the game UI

### Severity Levels
- CRITICAL: player becomes stuck, loses control, or misses critical information
- HIGH: normal players will likely fail a task or misunderstand the game state
- MEDIUM: friction or edge-case failure that noticeably impacts play
- LOW: polish or minor clarity improvement

### Section D: Player Journey Assessment
- Game loads: UI renders without errors
- First player connects: understands status and waiting for others
- All players ready: understands where they are and what to do
- First turn: can identify current player and available actions
- Action execution: can select and submit an action successfully
- Action resolves: understands outcome (success/failure and resource changes)
- Multiple turns: can track turn order and manage resources across turns
- Game end: understands win/lose condition

### Section E: Category Status
List findings per category or "No issues found."

