// Package actions owns Arkham Horror action dispatch logic used by the serverengine
// compatibility facade.
//
// The dispatcher in perform.go routes investigator actions (move, gather, investigate,
// castWard, focus, research, trade, component, encounter, attack, evade, closeGate)
// through callback interfaces so rule logic can be tested independently from the
// facade server implementation.
package actions
