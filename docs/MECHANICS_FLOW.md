# Game Mechanics Flow Documentation

This document maps each player action to its game mechanics effects, showing how the 5 core systems (Location, Resources, Actions, Doom, Dice) interact.

## Complete Action Resolution Flows

### 1. Move Action
**System Dependencies**: Location System + Action System

**Pre-Action Validation**:
- Verify player has `ActionsRemaining > 0`
- Verify target location is adjacent (4-location neighborhood graph)
- Location must satisfy `adjacencyMap[currentLocation]` contains `targetLocation`

**Execution** (`performMove`):
1. Validate adjacency with `validateMovement(from, to)`
2. Update `player.Location = targetLocation`
3. No resources consumed
4. No doom interaction

**Post-Action**:
- Decrement `player.ActionsRemaining--` in `processActionCore`
- If `ActionsRemaining == 0`, advance turn with `advanceTurn()`
- Broadcast `GameState` with new location

### 2. Gather Resources Action
**System Dependencies**: Dice Resolution + Resource Tracking + Doom Counter + Action System

**Pre-Action Validation**:
- Verify player has `ActionsRemaining > 0`
- Verify `focusSpend <= player.Resources.Focus`
- Optional: Allow focus pooling from teammates

**Execution** (`performGather`):
1. Call `rollDicePool(2, focusSpend, player)` → returns `(results []DiceResult, successes, tentacles)`
2. Deduct `player.Resources.Focus -= focusSpend`
3. **Success Path** (if `successes >= 1`):
   - `player.Resources.Health = min(Health + 1, MaxHealth=10)`
   - `player.Resources.Sanity = min(Sanity + 1, MaxSanity=10)`
   - `player.Resources.Money = min(Money + 1, MaxMoney=unlimited)`
4. **Tentacle Handling** (unconditional):
   - For each Tentacle in results: `gameState.Doom = min(Doom + 1, MaxDoom=12)`
   - Example: Roll `[Success, Blank, Tentacle]` → 1 success (gain +1 Health/Sanity), +1 doom

**Return Value**:
```go
&DiceResultMessage{
    Results:      results,      // ["success", "blank", "tentacle"]
    Successes:    successes,    // int
    Tentacles:    tentacles,    // int (each is +1 doom)
    Success:      successes >= 1,
    DoomIncrease: tentacles,
}
```

**Post-Action**:
- Decrement `player.ActionsRemaining--`
- Broadcast `GameState` showing updated resources and doom
- Validate resources stay within bounds via `validateResources()`

### 3. Investigate Action
**System Dependencies**: Dice Resolution + Resource Tracking + Doom Counter + Action System

**Pre-Action Validation**:
- Verify player has `ActionsRemaining > 0`
- Verify `focusSpend <= player.Resources.Focus`

**Execution** (`performInvestigate`):
1. Require `successes >= 2` (difficulty threshold)
2. Roll `rollDicePool(3, focusSpend, player)` → `(results, successes, tentacles)`
3. Deduct `player.Resources.Focus -= focusSpend`
4. **Success Path** (if `successes >= 2`):
   - `player.Resources.Clues = min(Clues + 1, MaxClues=5)`
   - Return result `"success"`
5. **Failure Path** (if `successes < 2`):
   - No clue gained
   - Return result `"fail"`
6. **Tentacle Handling**:
   - For each Tentacle: increment doom unconditionally
   - Tentacles increment doom REGARDLESS of success/failure

**Example Rolls**:
- `[Success, Success, Blank]` → 2 successes → PASS → gain +1 Clue
- `[Success, Blank, Tentacle]` → 1 success, 1 tentacle → FAIL → gain 0 Clues, +1 Doom
- `[Tentacle, Tentacle, Blank]` → 0 successes, 2 tentacles → FAIL → gain 0 Clues, +2 Doom

**Return Values**:
```go
&DiceResultMessage{ Results, Successes, Tentacles, Success: (successes >= 2), DoomIncrease: tentacles }
result := "success" or "fail"
```

**Post-Action**:
- Decrement `player.ActionsRemaining--`
- Broadcast `GameState` with updated doom, clues, and turn order
- If clues reach threshold, trigger win condition check

### 4. Cast Ward Action
**System Dependencies**: Dice Resolution + Resource Tracking + Doom Counter + Action System

**Pre-Action Validation**:
- Verify player has `ActionsRemaining > 0`
- Verify `player.Resources.Sanity >= 1` (cost)
- Verify `focusSpend <= player.Resources.Focus`

**Execution** (`performCastWard`):
1. Require `successes >= 3` (hardest difficulty)
2. Deduct 1 Sanity immediately: `player.Resources.Sanity -= 1`
3. Revert Sanity if action fails (see failure path)
4. Roll `rollDicePool(3, focusSpend, player)` → `(results, successes, tentacles)`
5. Deduct `player.Resources.Focus -= focusSpend`
6. **Success Path** (if `successes >= 3`):
   - `gameState.Doom = max(Doom - 2, MinDoom=0)`
   - Return result `"success"`
7. **Failure Path** (if `successes < 3`):
   - Restore Sanity: `player.Resources.Sanity += 1` (refund the cost)
   - Return result `"fail"`
8. **Tentacle Handling**:
   - Increment doom for each Tentacle (in addition to success/failure outcome)
   - Example: Roll `[Success, Blank, Tentacle]` → Fail (1 < 3), restore 1 Sanity, +1 Doom (net: no sanity change, sanity cost wasted, +1 doom)

**Return Values**:
```go
&DiceResultMessage{ Results, Successes, Tentacles, Success: (successes >= 3), DoomIncrease: max(tentacles - 2, 0) }
// Note: if success, doom decreased by 2 but increased by tentacles
// Example: 3 successes, 1 tentacle → net doom change = -2 + 1 = -1
```

**Post-Action**:
- Check resource bounds: Sanity must be in [1, 10]
- Broadcast `GameState` with updated sanity and doom
- If ward succeeds, check if doom < 12 (lose condition)

### 5. Focus Action
**System Dependencies**: Resource Tracking + Action System

**Pre-Action Validation**:
- Verify player has `ActionsRemaining > 0`

**Execution** (`performFocus`):
1. Gain `+1 Focus`: `player.Resources.Focus = min(Focus + 1, unlimited)`
2. No dice roll
3. No doom interaction
4. No resource transfer

**Post-Action**:
- Decrement `player.ActionsRemaining--`
- Broadcast `GameState` with updated Focus

### 6. Research Action
**System Dependencies**: Dice Resolution + Resource Tracking + Doom Counter + Action System

**Pre-Action Validation**:
- Verify player has `ActionsRemaining > 0`
- Verify `focusSpend <= player.Resources.Focus`

**Execution** (`performResearch`):
1. Require `successes >= 1` (easiest difficulty)
2. Roll `rollDicePool(3, focusSpend, player)` → `(results, successes, tentacles)`
3. Deduct `player.Resources.Focus -= focusSpend`
4. **Success Path** (if `successes >= 1`):
   - Gain `+1 Money`: `player.Resources.Money = unlimited`
   - Return result `"success"`
5. **Failure Path** (if `successes < 1`):
   - No money gained
   - Return result `"fail"`
6. **Tentacle Handling**:
   - Increment doom unconditionally for each Tentacle

**Example Rolls**:
- `[Success, Blank, Blank]` → 1 success → PASS → gain +1 Money
- `[Blank, Blank, Tentacle]` → 0 successes, 1 tentacle → FAIL → gain 0 Money, +1 Doom

**Post-Action**:
- Decrement `player.ActionsRemaining--`
- Broadcast `GameState`

## Turn Structure and Action Validation

### Per-Turn Enforcement
- **Start of Turn** (`startTurn`):
  - Identify current player from `gameState.TurnOrder`
  - Set `currentPlayer.ActionsRemaining = 2`
  - Broadcast turn indicator

- **Action Execution** (player submits `PlayerActionMessage`):
  1. Verify `player.ID == gameState.CurrentPlayer`
  2. Call `dispatchAction(action, player)` → executes appropriate `performX` method
  3. Call `processActionCore(...)` to:
     - Apply dice results to doom/resources
     - Decrement `player.ActionsRemaining--`
     - Validate all resource bounds
     - Check for game-ending conditions

- **End of Turn** (when `ActionsRemaining == 0`):
  - Call `advanceTurn()` to cycle to next player
  - Broadcast updated `GameState`

### Turn Order Rotation
```go
turnOrder := []string{player1ID, player2ID, player3ID}
currentIndex := findIndex(gameState.CurrentPlayer, turnOrder)
gameState.CurrentPlayer = turnOrder[(currentIndex + 1) % len(turnOrder)]
```

## Mythos Phase (Doom Progression)

The Mythos phase is triggered after all players have completed their turns (or after timeout). It advances global state:

1. **Event Draw**: Pull from `defaultMythosEventDeck()` and place at location
2. **Enemy Spawn**: Random enemy from `enemyTemplates` to random location
3. **Gate Spread**: If doom is high, spawn gates and anomalies
4. **Doom Increment**: Check for timeout-triggered doom (players exceed 30s inactivity)
5. **Lose Condition Check**: If `gameState.Doom >= 12`, end game with loss

## Resource Bounds and Validation

**Hard constraints** (checked in `validateResources`):
```
Health ∈ [1, 10]   — drop below 1 = defeated
Sanity ∈ [1, 10]   — drop below 1 = defeated
Clues  ∈ [0, 5]    — capped at 5 per investigator
Money  ∈ [0, ∞)    — unlimited
Doom   ∈ [0, 12]   — reach 12 = lose condition
Focus  ∈ [0, ∞)    — unlimited; spent per action
```

**Defeat Logic**:
- If `Health < 1` OR `Sanity < 1`: investigator defeated
- Enter `LostInTimeAndSpace` state; cannot act until recovered
- Recovery occurs automatically at start of Mythos phase if conditions allow

## Dice System Integration

**Core Function**: `rollDicePool(baseDice int, focusSpend int, player *Player) ([]DiceResult, int, int)`

- Input: base dice count (2 or 3), focus tokens to spend, player instance
- Output: `(results, successes, tentacles)` tuple
- Each die is 3-sided: Success, Blank, or Tentacle
- Focus spending adds extra dice and allows re-rolling Blanks
- Tentacle results always increment doom; success/blank depends on action threshold

**Thresholds by Action**:
| Action | Base Dice | Min Successes | Cost | Effect |
|--------|-----------|---------------|------|--------|
| Gather | 2 | 1 | Free | +1 Health, +1 Sanity |
| Investigate | 3 | 2 | Free | +1 Clue |
| CastWard | 3 | 3 | 1 Sanity | -2 Doom (if success) |
| Research | 3 | 1 | Free | +1 Money |

## Broadcasting and Client Synchronization

**Message Flow** (for every action):
1. Player submits `PlayerActionMessage` via WebSocket
2. Server calls `processActionCore(...)` which:
   - Applies effects (resources, doom, location)
   - Calls `gs.broadcaster.Broadcast(...)` within 500ms
   - Emits:
     - `GameUpdateMessage` (lightweight delta with action result)
     - `GameState` (full snapshot)
3. All connected clients receive both messages and update UI

**Example Investigate Flow**:
```
Client sends:   {"type": "playerAction", "playerId": "p1", "action": "investigate"}
Server rolls:   [Success, Blank, Tentacle]
Server state:   p1 gains Clue, Doom += 1
Server emits:   GameUpdateMessage { event: "investigate", result: "success", doomDelta: 1, resourceDelta: {clues: +1} }
Server emits:   GameState { doom: 4, players: {p1: {clues: 3}} }
All clients:    Render new doom counter and clue count
```

## Educational Takeaways

- **Action-Dice Integration**: Every action outcome is determined by dice rolls; tentacles always increment doom
- **Resource Economy**: Health and Sanity are precious (1-10 bounds); actions that consume them (Ward) require risk assessment
- **Turn-Based Fairness**: Exactly 2 actions per player, enforced in code to prevent bugs
- **State Consistency**: All clients see the same game state within 500ms via broadcast
- **Modular Validation**: Each mechanic (Location, Resources, Doom) is validated independently but integrated in `processActionCore`
