// Package actions provides implementations for all 12 Arkham Horror investigator actions:
// Move, Gather, Investigate, CastWard, Focus, Research, Trade, Component, Encounter, Attack, Evade, CloseGate.
//
// During the modular rules migration (ROADMAP.md), action handlers from serverengine.performAction*
// will be moved here. Each action will validate state, apply effects, and return results.
//
// NOTE: This package is a scaffold. Implementation is deferred until the Arkham rules
// decomposition begins. See ROADMAP.md Phase 2 for timeline.
package actions
