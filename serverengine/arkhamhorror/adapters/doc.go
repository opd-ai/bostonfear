// Package adapters translates between Arkham Horror-specific game events and the shared
// runtime contracts in serverengine/common/contracts. This layer decouples game rules
// from network protocol details and enables testing of rules in isolation.
//
// During migration, adapters will convert between protocol wire types (JSON) and
// internal Arkham types (Location, InvestigatorType, etc.).
//
// NOTE: This package is a scaffold. Implementation is deferred. See ROADMAP.md.
package adapters
