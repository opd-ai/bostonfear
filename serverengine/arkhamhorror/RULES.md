# Arkham Horror Rules Ownership

This document defines rules owned by the Arkham Horror module and serves as a migration target for behavior currently implemented in legacy serverengine files.

## Core Invariants

- Locations are limited to the Arkham neighborhood graph.
- Investigators perform exactly 2 actions per turn unless defeated/disconnected.
- Doom is clamped to [0, 12].
- Health and Sanity are clamped to [0, 10].
- Clues are clamped to [0, 5].

## Core Systems

- Action resolution: Move, Gather, Investigate, Cast Ward, Focus, Research, Trade, Encounter, Component, Attack, Evade, Close Gate.
- Dice resolution: Success, Blank, Tentacle with action-specific success thresholds.
- Mythos progression: event draw, spread, token draw, enemy/anomaly/gate updates.
- End conditions: Act/Agenda progression plus hard Doom lose condition.

## Validation Coverage Expectations

- Turn progression tests
- Action legality tests
- Mythos phase state transition tests
- Defeat/recovery lifecycle tests
- State recovery and corruption repair tests

## Migration Boundary

Rule constants, map adjacency, archetypes, and scenario decks belong here and must not be added to serverengine/common.
