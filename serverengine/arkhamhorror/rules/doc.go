// Package rules owns all Arkham Horror rule logic extracted from serverengine.GameServer:
// location adjacency validation, resource bound checks, action legality (2 per turn),
// dice threshold resolution (1-3 successes required), and doom counter management.
//
// Pure rule functions accept game state and return validation errors or effect results.
// This enables offline rule testing and easier debugging.
//
// NOTE: This package is a scaffold. Implementation is deferred. See ROADMAP.md.
package rules
