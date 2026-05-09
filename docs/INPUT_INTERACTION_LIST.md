# Mouse And Keyboard Interaction List

This document satisfies Phase 1 item "Mouse and keyboard interaction list approved" from PLAN.md.

## Keyboard Actions

| Input | Action | Target/Rule |
|---|---|---|
| `1` | Move | `Downtown` |
| `2` | Move | `University` |
| `3` | Move | `Rivertown` |
| `4` | Move | `Northside` |
| `G` | Gather | No target |
| `I` | Investigate | No target |
| `W` | Ward | No target |
| `F` | Focus | No target |
| `R` | Research | No target |
| `T` | Trade | Auto-target first co-located player; rejects when none exists |
| `C` | Component | No target |
| `A` | Attack | No target |
| `E` | Evade | No target |
| `X` | Close Gate | No target |

## Mouse Controls

| Input | Action |
|---|---|
| Mouse wheel up/down | Camera orbit clockwise/counter-clockwise in game scene |
| Middle click | Camera view mode toggle in game scene |

## Keyboard Camera Controls

| Input | Action |
|---|---|
| `[` | Camera orbit counter-clockwise |
| `]` | Camera orbit clockwise |
| `V` | Camera view mode toggle |

## Character Selection Inputs

| Input | Action |
|---|---|
| `1`..`6` | Select investigator archetype in character-select scene |

## Game Over Inputs

| Input | Action |
|---|---|
| `Enter` or `Space` | Restart flow (reset state and reconnect) |

## Current Validation Rules

1. Action submission is permitted only while connected and on the local player's turn.
2. Out-of-turn/disconnected action attempts are recorded as invalid retries.
3. Trade without a co-located player is rejected and recorded as invalid retry.

## Approval Notes

- Keyboard action surface is approved for all currently implemented action types.
- Mouse support currently focuses on camera controls; gameplay action dispatch is keyboard + touch.
- A future phase can add left-click action buttons if full mouse parity is required.