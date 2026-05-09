// Package phases implements Arkham Horror turn structure: investigator phase (player actions),
// mythos phase (enemy spawns, gate spreads, doom advancement), and resolution phase (check win/lose).
//
// Each phase is a state machine that validates preconditions, applies transitions,
// and posts state updates to the shared runtime.
//
// NOTE: This package is a scaffold. Implementation is deferred. See ROADMAP.md.
package phases
