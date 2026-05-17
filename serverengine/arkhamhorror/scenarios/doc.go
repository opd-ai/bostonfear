// Package scenarios defines Arkham Horror scenario templates: starting investigator count,
// initial location distribution, difficulty thresholds, act/agenda decks, and win conditions.
//
// Scenarios are immutable content that GameServer loads on startup. Each scenario
// governs the flow from initial setup through act/agenda progression to game end.
//
// Content ownership: live scenario definitions (YAML/JSON) reside in
// serverengine/arkhamhorror/content/nightglass/ and are loaded by the content loader
// registered in serverengine/arkhamhorror/module.go. This package is reserved for
// Go type definitions (ScenarioTemplate, ActCard, AgendaCard, etc.) used by the loader
// and by future non-Arkham scenario subtypes that require different Go representations.
//
// Implementation status: Go type definitions are deferred. The content loader uses
// the YAML schema directly; see serverengine/arkhamhorror/content/ for active content.
// See ROADMAP.md Phase 1 for the complete Arkham content inventory.
package scenarios
