# Arkham Horror WebSocket Game - Network & Game State Audit (No-Edit, Autonomous)

## Mission
Run a deep, autonomous audit of the Arkham Horror multiplayer game implementation with primary emphasis on WebSocket communication reliability, game state synchronization, mechanic integration correctness, and Go idiom compliance. Produce a structured diagnostic report only. Do not modify files.

## Execution Mode
- Autonomous static audit report generation
- Read/analyze code only
- No patches, no refactors, no file writes

## Repository Context (bostonfear)
- Game uses Go WebSocket server with JavaScript client for 1-6 concurrent players
- Implementation depends on: reliable state synchronization, proper turn order enforcement, net interface abstraction, correct mechanic interactions (Location, Resources, Actions, Doom, Dice)
- Focus on reliability: turn progression, action validation, dice resolution, doom counter updates, client-server consistency, and Go idiom adherence

## Scope Rules
- Include only server files in `/server` implementing: WebSocket handlers, game state management, turn order, mechanic systems, and net.Conn/net.Listener interface usage
- Include client files in `/client` managing: connection lifecycle, game state rendering, input capture, action transmission
- Include mechanic integration where systems interact: dice affecting doom, actions consuming resources, movement validating location adjacency
- Exclude scaffold/template code unrelated to core game mechanics

## Priority Order (strict)
1. Net interface compliance (net.Conn, net.Listener, net.Addr usage)
2. Turn order and action system correctness
3. Game state synchronization across clients
4. Mechanic integration and cross-system validation
5. WebSocket connection reliability and reconnection handling
6. Dice resolution, doom counter, and resource bounds enforcement

## Audit Procedure
### Step 1: Go idiom and network interface checks (primary)
- Verify all network operations use net.Conn, net.Listener, net.Addr interfaces
- Check for concrete type usage (net.TCPConn, net.TCPListener, net.TCPAddr) instead of abstractions
- Validate goroutine and channel usage for concurrent connection management
- Confirm explicit error handling with nil checks on all network operations
- Verify no manual memory management or finalizers

### Step 2: Turn order and action integrity checks (primary)
- Missing action count enforcement (players exceeding 2 actions per turn)
- Turn advancement race conditions (multiple players advancing simultaneously)
- Stale player state references between turns
- Action validation bypass (invalid moves, resource overages, location adjacency)
- Turn timeout and reconnection edge cases (player drops mid-turn, reconnects)
- Action execution order guarantees across concurrent WebSocket connections

### Step 3: State synchronization checks (primary)
- Client-server state divergence (updates not reaching all clients within 500ms)
- Message delivery ordering violations (gameState before playerAction confirmation)
- Partial state updates causing inconsistency (doom incremented but not broadcast)
- Reconnection state recovery (rejoining player receives correct current state)
- Concurrent action conflict resolution (simultaneous actions on same resource)
- JSON protocol compliance with required message types

### Step 4: Mechanic integration checks
- Dice roll with Tentacle outcomes not incrementing doom counter
- Actions not validating and consuming resource costs
- Location adjacency validation for movement restrictions
- Resource bounds violations (Health/Sanity 1-10, Clues 0-5, Doom 0-12)
- Cross-mechanic state corruption (resource loss without action validation)
- Turn order progression after exactly 2 actions per player

### Step 5: Secondary technical checks
- Per-action allocations in message handlers
- JSON marshaling/unmarshaling errors not handled
- Goroutine leaks from abandoned or dropped connections
- Channel deadlocks in concurrent action processing
- WebSocket frame size limits for large state updates
- Broadcast message ordering to multiple connected clients

## False-Positive Controls
- Do not flag intentional patterns unless concrete gameplay impact is shown
- For each finding, cite exact evidence (code pattern, message type, state variable)
- If uncertain, mark as Needs Runtime Validation with minimal reproduction scenario

## Required Output Structure

### Section A: Coverage
- List every audited file and status: Audited or Skipped (with reason)
- Confirm no findings outside game mechanics and WebSocket boundary

### Section B: Findings (sorted)
- Sort by severity descending, then file path alphabetically

For each finding, use this exact template:

```markdown
### [SEVERITY] Short description
- File: path/to/file.go#Lstart-Lend or path/to/file.js#Lstart-Lend
- Category: Interface | TurnOrder | Synchronization | Mechanic | Connection | Performance | Resource
- User Impact: One sentence describing error/freeze/desync
- Problem: One sentence defect statement
- Evidence: Concrete code pattern, race condition, or call sequence
- Fix: Specific, testable change (include guard conditions, validation order, state reset)
- Validation: How to reproduce and verify fix
```

### Severity definitions
- CRITICAL: crash, freeze, stuck turn, state divergence, or mechanic violation
- HIGH: visible synchronization/turn/mechanic bug in normal play
- MEDIUM: intermittent or edge-case gameplay breakage
- LOW: defensive hardening or polish

### Section C: Go Idiom Compliance Summary
- Net interface usage (Conn/Listener/Addr): Pass/Fail with evidence
- Goroutine and channel safety: Pass/Fail with rationale
- Error handling consistency: Pass/Fail with examples
- Exported/unexported naming conventions: Pass/Fail

### Section D: Turn & Action System Summary
- Action count enforcement: Pass/Fail
- Turn order progression correctness: Pass/Fail
- Action validation (resources, location, bounds): Pass/Fail
- Timeout and reconnection handling: Pass/Fail

### Section E: State Synchronization Summary
- Message ordering and delivery guarantees: Pass/Fail
- Client-server consistency: Pass/Fail
- Concurrent action safety: Pass/Fail
- Reconnection state recovery: Pass/Fail

### Section F: Mechanic Integration Summary
- Dice roll doom counter updates: Pass/Fail
- Resource cost validation and enforcement: Pass/Fail
- Location adjacency validation: Pass/Fail
- Resource bounds and win/lose conditions: Pass/Fail

### Section G: Category Status
- Interface Compliance: findings or No issues found.
- TurnOrder: findings or No issues found.
- Synchronization: findings or No issues found.
- Mechanic: findings or No issues found.
- Connection: findings or No issues found.
- Performance: findings or No issues found.
- Resource: findings or No issues found.

## Acceptance Criteria
- Every audited file is explicitly listed with status
- Interface compliance receives equal priority to mechanics
- All CRITICAL/HIGH findings include testable fixes and validation steps
- No findings reference unrelated infrastructure
- Report is deterministic, ordered, and actionable for gameplay correctness
