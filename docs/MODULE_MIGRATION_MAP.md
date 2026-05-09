# Module Migration Map

This document tracks the migration of Arkham runtime ownership from the
compatibility layer in serverengine into module-owned packages under
serverengine/arkhamhorror.

## Goal

Align implementation ownership with documented architecture:

- serverengine/common: cross-game runtime primitives only
- serverengine/arkhamhorror: Arkham rules and gameplay implementation
- serverengine: temporary compatibility facade until slices are migrated

## Migration Rules

- Preserve wire protocol and public API behavior while moving internals.
- Move vertical slices with tests, not broad package rewrites.
- Add parity tests before and after each move.
- Keep dependency direction one-way:
	common <- arkhamhorror <- serverengine facade/wiring

## Slice Backlog

| ID | Capability Slice | Current Primary Files | Target Module Package | Status | Notes |
|---|---|---|---|---|---|
| S1 | Action dispatch and legality gates | serverengine/actions.go, serverengine/game_mechanics.go | serverengine/arkhamhorror/actions | Planned | Start here for smallest high-value vertical path |
| S2 | Turn progression and mythos transitions | serverengine/mythos.go, serverengine/game_mechanics.go | serverengine/arkhamhorror/phases | Planned | Includes connected/defeated fallback handling |
| S3 | Dice resolution and doom coupling | serverengine/dice.go, serverengine/game_mechanics.go | serverengine/arkhamhorror/rules | Planned | Keep tentacle -> doom invariants identical |
| S4 | Investigator and resource model rules | serverengine/game_types.go, serverengine/health.go | serverengine/arkhamhorror/model | Planned | Preserve clamp and defeat invariants |
| S5 | Scenario setup, decks, constants, adjacency | serverengine/game_constants.go, serverengine/mythos.go | serverengine/arkhamhorror/scenarios, serverengine/arkhamhorror/content | Planned | Keep RULES.md ownership boundary explicit |
| S6 | Broadcast payload shaping and mechanic events | serverengine/broadcast.go, serverengine/game_server.go | serverengine/arkhamhorror/adapters | Planned | Keep protocol message ordering unchanged |

## Execution Template Per Slice

1. Add parity tests against current behavior in serverengine tests.
2. Introduce module-owned implementation behind an internal interface.
3. Switch serverengine facade call site to module implementation.
4. Re-run race tests and vet.
5. Mark slice status as In Progress, then Completed when parity holds.

## Tracking

- Last updated: 2026-05-09
- Current active slice: none
- Completed slices: none
