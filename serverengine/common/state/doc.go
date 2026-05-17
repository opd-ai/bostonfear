// Package state provides shared state primitives for cross-engine use.
// ResourceBounds defines min/max constraints for investigator resource values.
package state

// ResourceBounds holds the inclusive min and max for a single investigator resource.
type ResourceBounds struct {
	Min int
	Max int
}

var (
	// HealthBounds allows defeated investigators at 0 and caps at 10.
	HealthBounds = ResourceBounds{Min: 0, Max: 10}
	// SanityBounds allows defeated investigators at 0 and caps at 10.
	SanityBounds = ResourceBounds{Min: 0, Max: 10}
	// ClueBounds tracks per-investigator clue capacity.
	ClueBounds = ResourceBounds{Min: 0, Max: 5}
)

// Clamp returns v clamped to [Min, Max].
func (b ResourceBounds) Clamp(v int) int {
	if v < b.Min {
		return b.Min
	}
	if v > b.Max {
		return b.Max
	}
	return v
}

// InBounds reports whether v is within [Min, Max].
func (b ResourceBounds) InBounds(v int) bool { return v >= b.Min && v <= b.Max }

// ClampCoreResources applies canonical Arkham investigator bounds to health, sanity, and clues.
func ClampCoreResources(health int, sanity int, clues int) (int, int, int) {
	return HealthBounds.Clamp(health), SanityBounds.Clamp(sanity), ClueBounds.Clamp(clues)
}
