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
| S1 | Action dispatch and legality gates | serverengine/actions.go, serverengine/game_mechanics.go | serverengine/arkhamhorror/actions | Completed | Module owns dispatch logic via perform.go callbacks; facade methods in serverengine |
| S2 | Turn progression and mythos transitions | serverengine/mythos.go, serverengine/game_mechanics.go | serverengine/arkhamhorror/phases | Completed | Module owns advanceTurn, runMythosPhase, event/token resolution orchestration via callbacks; facade keeps existing API |
| S3 | Dice resolution and doom coupling | serverengine/dice.go, serverengine/game_mechanics.go | serverengine/arkhamhorror/rules | Completed | Core dice logic owned by rules/dice.go and wired through the serverengine facade |
| S4 | Investigator and resource model rules | serverengine/game_types.go, serverengine/health.go | serverengine/arkhamhorror/model | Completed | Resource bounds and clamping logic owned by arkhamhorror/model; types in protocol |
| S5 | Scenario setup, decks, constants, adjacency | serverengine/game_constants.go, serverengine/mythos.go | serverengine/arkhamhorror/scenarios, serverengine/arkhamhorror/content | Completed | Map topology and constants in content/map.go; encounter decks remain façade until S2 |
| S6 | Broadcast payload shaping and mechanic events | serverengine/broadcast.go, serverengine/game_server.go | serverengine/arkhamhorror/adapters | Completed | Broadcast adapter interface and payload shapes defined in adapters/broadcast.go |

## Execution Template Per Slice

1. Add parity tests against current behavior in serverengine tests.
2. Introduce module-owned implementation behind an internal interface.
3. Switch serverengine facade call site to module implementation.
4. Re-run race tests and vet.
5. Mark slice status as In Progress, then Completed when parity holds.

## Tracking

- Last updated: 2026-05-09
- Current active slice: none (all tracked slices completed)
- Completed slices: 
  - S1 ✅ (Action dispatch via perform.go callbacks)
  - S2 ✅ (Turn progression and mythos orchestration in phases/mythos.go)
  - S3 ✅ (Dice logic in rules/dice.go)
  - S4 ✅ (Resource bounds in model/investigator.go)
  - S5 ✅ (Map topology in content/map.go)
  - S6 ✅ (Broadcast shapes in adapters/broadcast.go)
- Next priority: Continue shrinking the compatibility facade where useful; no blocking migration slice remains
