// Package model owns Arkham Horror-specific type aliases and constants
// (Location, InvestigatorType, etc.) that encapsulate game-family concerns.
//
// Migration status: this package is active and used by Arkham rules/actions slices;
// remaining protocol-to-domain boundary tightening is tracked as incremental work.
//
// Dependencies: serverengine/common (runtime contracts only).
// Forbidden: importing serverengine package (monolith).
package model
