> **⚠️ Intellectual Property Notice**
> BostonFear is a **rules-only game engine** designed to execute the mechanics of the
> Arkham Horror series of games. This repository contains **no copyrighted content**
> produced by Fantasy Flight Games. No card text, scenario narratives, investigator
> stories, artwork, encounter text, or any other proprietary material owned by
> Fantasy Flight Games (an Asmodee brand) is, or will ever be, reproduced here.
> *Arkham Horror* is a trademark of Fantasy Flight Games. This project is an
> independent, fan-made rules engine and is not affiliated with or endorsed by
> Fantasy Flight Games or Asmodee.

**BostonFear** implements the rule systems for Arkham Horror Third Edition (AH3e), a cooperative
board game published by Fantasy Flight Games. This document specifies the engine's compliance
against those rule systems and defines the canonical implementation requirements.

## Engine Implementation Status

> This section tracks the compliance of the BostonFear game engine against the AH3e
> rule systems specified below. Updated April 24, 2026.

| Rule System | Specified in RULES.md | Implemented in Engine | Test Coverage | Gap Reference |
|---|---|---|---|---|
| Action System (2 actions/turn, 12 action types) | ✅ | ✅ All 12 (Move, Gather, Investigate, Ward, Focus, Research, Trade, Encounter, Component, Attack, Evade, CloseGate) | ✅ `TestRulesFullActionSet` | — |
| Dice Resolution (pool, focus spend, tentacle) | ✅ | ✅ `rollDicePool` with focus spend and rerolls | ✅ `TestRulesDicePoolFocusModifier` | — |
| Mythos Phase (draw 2 events, place, spread, cup token) | ✅ | ✅ `runMythosPhase` with event types and token cup | ✅ `TestRulesMythosPhaseEventPlacement` | — |
| Resource Management (money, clues, remnants, focus) | ✅ | ✅ Health, Sanity, Clues, Money, Remnants, Focus (6 types) | ✅ `TestRulesResourceTypes` | — |
| Encounter Resolution | ✅ | ✅ Deck-based draws with typed effects | ✅ `TestRulesEncounterResolution` | — |
| Act/Agenda Deck Progression | ✅ | ✅ Clue thresholds scale with player count | ✅ `TestRulesActAgendaProgression` | — |
| Investigator Defeat/Recovery | ✅ | ✅ LostInTimeAndSpace state + auto-recovery at Mythos Phase | ✅ `TestRulesDefeatRecovery`, `TestInvestigatorAutoRecovery` | — |
| Scenario System (setup, victory/defeat) | ✅ | ✅ `DefaultScenario` with custom win/lose conditions | ✅ `TestRulesScenarioSystem` | — |
| Modular Difficulty Settings | ✅ | ✅ Easy/Normal/Hard presets via `ActionSetDifficulty` | ✅ `TestDifficulty_*`, `TestProcessAction_SetDifficulty` | — |
| 1–6 Player Support | ✅ | ✅ Min 1, Max 6, join-in-progress, act deck rescaling | ✅ `TestRescaleActDeck_LateJoin` | — |
| Attack/Evade (enemies) | ✅ | ✅ `performAttack`, `performEvade`, enemy spawn | ✅ `TestProcessAction_Attack`, `TestProcessAction_Evade` | — |
| Gate/Anomaly Mechanics | ✅ | ✅ `openGate`, `performCloseGate`, anomaly spawns | ✅ `TestGateMechanics_OpenAndClose` | — |
| Investigator Selection | ✅ | ✅ 6 archetypes via `ActionSelectInvestigator` | ✅ `TestProcessAction_SelectInvestigator` | — |

**Legend**: ✅ = Complete, ⚠️ = Partial, ❌ = Missing/None

**Status**: 13/13 core rule systems fully implemented and covered by automated tests.

## Engine Design Summary

The BostonFear engine models a cooperative investigator game as a state machine. Each connected
client drives one investigator. The server enforces all rules centrally; clients are display and
input terminals only.

**Turn structure**: The engine alternates between two phases per round. During the investigator
phase, each player submits actions in turn order. Each investigator may submit exactly two actions
before the server advances the turn pointer. Once all investigators have acted, the server
automatically executes the mythos phase and then loops back to the first investigator.

**Action model**: An action is a typed intent (`ActionType`) paired with an optional target
identifier. The engine validates the action against current state (location adjacency, resource
availability, turn ownership) and either applies effects or returns a rejection event. Twelve
action types are supported; see the implementation table above for full coverage.

**Dice resolution**: Actions that require a skill test receive a dice pool determined by the
action type and any focus tokens the investigator spends. Each die resolves independently to one
of three outcomes: success, blank, or tentacle. The engine counts successes against a per-action
threshold and counts tentacle results for doom accumulation, then emits a `diceResult` message
before the `gameState` broadcast.

**Mythos phase**: The engine draws from an event queue, places events on neighborhood nodes
following placement priority rules, resolves spread interactions, draws a mythos cup token, and
updates doom totals. Enemy and anomaly spawns are emitted as state deltas before the consolidated
`gameState` broadcast.

**Resource model**: Six resource dimensions are tracked per investigator: health, sanity, money,
clues, remnants, and focus tokens. Each has a defined minimum, maximum, and set of actions that
produce or consume it. The engine clamps all values at the configured bounds and emits a defeat
event when health or sanity reaches zero.

**Encounter resolution**: When an investigator activates an encounter token, the engine selects a
card from the neighborhood-specific encounter deck, evaluates any embedded skill test, applies
effects, and discards the token. No encounter card text is stored or transmitted; only typed
effect payloads are used.

**Win and loss**: Victory is evaluated per-scenario; the default scenario requires act deck
completion before the agenda deck is exhausted. Loss fires when doom reaches the configured
maximum (default 12) or when a scenario-specific loss condition triggers. Both conditions emit
a `gameUpdate` event with the appropriate outcome flag before the final `gameState` broadcast.

**Difficulty**: Three named presets (Easy, Normal, Hard) adjust starting doom placement and the
composition of the mythos cup token distribution. Presets are applied at scenario start and do
not change mid-game.

Arkham Horror Third Edition - Game Engine Specification
Core Game State

    Players: 1-6 player support
    Board: Modular neighborhood tile system with dynamic placement
    Turn Structure: Investigator Phase → Mythos Phase → cycle
    Resources Per Investigator: Health, Sanity, Money, Clues, Remnants, Focus Tokens
    Global State: Doom tokens on locations, Mythos Cup contents, Active Scenario

Game Components Data Structures
dts

Investigator {
  id: string
  name: string
  health: number (max)
  sanity: number (max)
  startingLocation: NeighborhoodID
  specialAbility: AbilityFunction
  personalStory: StoryCard
  inventory: Item[]
  currentHealth: number
  currentSanity: number
  resources: {money, clues, remnants, focus}
}

Neighborhood {
  id: string
  name: string
  connections: NeighborhoodID[]
  spaces: Space[]
  encounterDeck: EncounterCard[]
  doomTokens: number
  events: EventCard[]
}

DiceResult {
  success: number
  blank: number
  tentacle: number
}

Action System

Each investigator receives 2 actions per turn:

    Move: Travel to adjacent neighborhood
    Gather Resources: Gain $1
    Focus: Gain 1 focus token
    Ward: Remove 1 doom from location
    Research: Spend resources to gain clues
    Trade: Exchange resources/items with investigator in same space
    Component Action: Activate card/space abilities
    Attack/Evade: Engage enemies

Dice Resolution System

    Custom dice pool: Base dice + skill modifiers + focus tokens spent
    Results: Success (✓), Blank, Tentacle (trigger mythos token effect)
    Test difficulty: Number of successes required

Mythos Phase Sequence

    Draw 2 event cards
    Place events in neighborhoods (following placement rules)
    Resolve event spread (doom + event = escalation)
    Draw and resolve mythos token
    Advance doom/agenda if conditions met
    Spawn anomalies per scenario rules

Scenario System Requirements

    Branching narrative through Codex entries (numbered text blocks)
    Act/Agenda deck progression
    Unique setup configuration per scenario
    Victory/defeat conditions
    Triggered story events based on game state

Resource Management Rules

    Money: Purchase items/allies from display (refresh during mythos)
    Clues: Advance act deck, fulfill scenario objectives
    Remnants: Spend for powerful effects, gained from supernatural encounters
    Focus: Spend to reroll dice or convert results

Encounter Resolution

    Investigator engages encounter token
    Draw from appropriate neighborhood deck
    Resolve skill test or choice
    Apply consequences/rewards
    Discard encounter token

Defeat Conditions

    Investigator: Health OR Sanity reaches 0
    Team: Final agenda card reached OR scenario-specific loss condition
    Defeated investigators: Become "lost in time and space" (limited actions)

Victory Conditions

    Advance through all act cards
    Complete scenario-specific objectives
    Prevent agenda completion

Modular Difficulty Settings

    Starting doom placement
    Mythos cup token composition
    Resource availability
    Timer pressure (agenda advancement rate)

---

## Non-Goals

This document specifies **engine rule requirements only**. The following are explicitly
out of scope for this project:

- Game content creation (card text, encounter narratives, scenario scripts)
- Card or scenario data files (JSON/YAML card definitions, codex entries)
- Investigator flavor text, personal story writing, or thematic copy
- Art assets, card layout, or print-ready materials
- Expansion content (Under Dark Waves, Dead of Night, etc.)

For the full list of project non-goals, see `ROADMAP.md` § Non-Goals.
