// Package state provides shared state primitives for cross-engine use.
// ResourceBounds defines min/max constraints for investigator resource values.
package state

// ResourceBounds holds the inclusive min and max for a single investigator resource.
type ResourceBounds struct {
	Min int
	Max int
}

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
