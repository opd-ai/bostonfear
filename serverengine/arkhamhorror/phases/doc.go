// Package phases implements Arkham Horror turn structure orchestration.
//
// The package owns phase sequencing and turn progression logic while the
// serverengine facade supplies callbacks for concrete state mutation, metrics,
// and compatibility with the existing public API.
//
// This keeps Arkham-specific phase flow under serverengine/arkhamhorror without
// introducing circular dependencies on the compatibility layer.
package phases
