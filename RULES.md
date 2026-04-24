**Arkham Horror Third Edition** is the latest edition of Fantasy Flight Games' flagship cooperative horror board game, released in 2018. This edition represents a significant reimagining of the classic Lovecraftian investigation game, streamlining many mechanics while maintaining the thematic depth the series is known for.

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

The game supports 1-6 players who take on the roles of investigators exploring the city of Arkham, Massachusetts in the 1920s. The core objective involves uncovering clues and thwarting the machinations of an Ancient One before doom spreads throughout the city. Unlike previous editions, Third Edition uses a modular board consisting of neighborhood tiles that are placed during setup based on the chosen scenario, creating a more focused and narrative-driven experience.

The action system has been completely overhauled from previous editions. Each investigator receives two actions per turn, which can be spent to move between neighborhoods, gather resources, focus to improve dice rolls, ward locations to remove doom, research to gain clues, trade with other investigators, or activate special abilities. Combat and skill tests now use a custom dice system featuring success, blank, and tentacle results, with the tentacle faces potentially triggering negative effects based on the current mythos token.

The game employs a scenario-based structure with four included in the base game. Each scenario features unique setup instructions, victory conditions, and narrative elements delivered through event cards and the codex - a companion booklet containing numbered entries that are read when specific game conditions are met. This creates branching storylines where player choices directly impact how the scenario unfolds.

During the mythos phase, the game uses an innovative event deck system. Two event cards are drawn and placed in neighborhood spaces, creating localized threats that investigators must address. If events spread to neighborhoods already containing doom tokens, they trigger increasingly severe consequences. The mythos cup contains tokens that modify how mythos effects resolve, adding an element of escalating tension as more dangerous tokens are added throughout the game.

Investigators accumulate various resources including money, clues, remnants, and focus tokens. Money purchases items and allies from the display, clues advance the scenario objectives, remnants are supernatural currency for powerful effects, and focus tokens improve dice rolls. The game also features a sanity and stamina system where investigators can suffer horror and damage, potentially becoming defeated if either track is fully depleted.

The encounter system has been streamlined compared to earlier editions. When investigators engage with encounter tokens on the board, they draw from neighborhood-specific encounter decks that provide thematically appropriate challenges and rewards. Anomalies represent tears in reality that spawn throughout the game, requiring investigators to seal them before they overwhelm the city with doom.

Victory conditions vary by scenario but typically involve collecting a certain number of clues to advance through act cards that tell the story, while simultaneously preventing the agenda deck from advancing too far. The agenda deck functions as a timer, with doom tokens accumulating on locations pushing the Ancient One's plans forward. If the final agenda card is reached, the investigators typically face a climactic confrontation with severely reduced chances of success.

The game includes variable player powers through unique investigator abilities and starting equipment, asymmetric starting positions based on the scenario setup, and a personal story system where investigators can complete individual objectives for rewards. The difficulty can be adjusted through various mechanisms including the initial doom placement and the composition of the mythos cup, allowing groups to tailor the challenge to their preference.

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
