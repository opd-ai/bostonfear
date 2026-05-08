# Implementation Gaps — 2026-05-08

## Legacy JS Client Parse Failure (Critical Runtime Blocker)
- **Intended Behavior**: Legacy browser client remains functional during migration (`README.md` marks legacy browser target active).
- **Current State**: Duplicate code fragment exists in class body and breaks JavaScript parsing (`client/game.js:397-434`).
- **Blocked Goal**: Browser client cannot run, so multiplayer access via `client/index.html` is broken.
- **Implementation Path**: Remove duplicated fragment, keep single `displayDiceResult` implementation, then re-check class structure for balanced braces/method boundaries.
- **Dependencies**: None.
- **Effort**: small

## Missing SceneCharacterSelect in Ebitengine Scene State Machine
- **Intended Behavior**: CLIENT_SPEC requires `SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver`.
- **Current State**: Code explicitly defers SceneCharacterSelect and transitions directly to SceneGame when connected (`client/ebiten/app/scenes.go:7-8`, `108-112`).
- **Blocked Goal**: Planned investigator-selection UX and phased migration completeness.
- **Implementation Path**: Implement `SceneCharacterSelect` UI/state, add readiness criteria from server state, and update transition logic/tests to enforce four-scene sequencing.
- **Dependencies**: Requires stable client handling for `selectInvestigator` action and selection readiness state.
- **Effort**: medium

## Connection Quality Self-Indicator Not Wired
- **Intended Behavior**: Client updates own status badge from `connectionQuality` payloads.
- **Current State**: Client checks `qualityMessage.playerID`, but server emits `playerId` (`client/game.js:462`, `cmd/server/dashboard.go:31`).
- **Blocked Goal**: Real-time own-connection quality feedback from implemented ping/pong system.
- **Implementation Path**: Normalize key usage to `playerId` on client path; add a message-shape regression test for quality update handling.
- **Dependencies**: None.
- **Effort**: small

## Difficulty System Partially Wired (ExtraDoomTokens Unused)
- **Intended Behavior**: Difficulty should adjust initial conditions including extra doom-token pressure (`DifficultySetup`).
- **Current State**: `applyDifficulty` only applies `InitialDoom`; `ExtraDoomTokens` is never consumed (`cmd/server/game_constants.go:215-224`, `cmd/server/game_mechanics.go:113-120`).
- **Blocked Goal**: Full modular difficulty behavior promised by architecture/spec docs.
- **Implementation Path**: Add token-cup representation to game state and consume `ExtraDoomTokens` in setup/mythos token draw logic; add deterministic tests proving per-difficulty token composition effects.
- **Dependencies**: Requires explicit Mythos cup state modeling if currently implicit/random-only.
- **Effort**: medium

## Runtime-Dead Helpers in Ebitengine App Layer
- **Intended Behavior**: Production helpers in `app/game.go` should support runtime rendering logic.
- **Current State**: `playerColourIndex` and `min8` are defined in production file but only referenced in tests (`client/ebiten/app/game.go:324`, `334`).
- **Blocked Goal**: None directly; adds maintenance burden and obscures active code paths.
- **Implementation Path**: Remove dead helpers or wire them into draw paths that currently duplicate equivalent logic.
- **Dependencies**: None.
- **Effort**: small

## Unused Base Protocol Type in Server
- **Intended Behavior**: Protocol types should reflect actively used server message contracts.
- **Current State**: `Message` envelope type is declared but not used in processing flow (`cmd/server/game_types.go:31`).
- **Blocked Goal**: None directly; ambiguous API surface can mislead contributors.
- **Implementation Path**: Either remove the type or standardize decode/encode flow around it and update handlers accordingly.
- **Dependencies**: None.
- **Effort**: small

