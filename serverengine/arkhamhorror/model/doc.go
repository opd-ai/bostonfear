// Package model owns Arkham Horror-specific type aliases and constants (Location, InvestigatorType, etc.)
// that encapsulate Arkham game family concerns.
//
// During migration, these types will replace protocol wire types in serverengine game_constants.go
// to create a clear boundary between protocol and domain logic.
//
// Dependencies: serverengine/common (runtime contracts only).
// Forbidden: Importing serverengine package (monolith).
//
// NOTE: This package is a scaffold. Implementation is deferred. See ROADMAP.md.
package model
