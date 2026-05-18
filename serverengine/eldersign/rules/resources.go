package rules

// ResourceBounds defines Elder Sign resource limits.
// Unlike Arkham Horror's fixed 1-10 bounds, Elder Sign uses
// different ranges for Stamina and Sanity (1-8).
type ResourceBounds struct {
	MinStamina int
	MaxStamina int
	MinSanity  int
	MaxSanity  int
}

// DefaultResourceBounds returns the standard Elder Sign resource limits.
// Stamina and Sanity both range from 1-8, different from Arkham's 1-10.
func DefaultResourceBounds() ResourceBounds {
	return ResourceBounds{
		MinStamina: 1,
		MaxStamina: 8,
		MinSanity:  1,
		MaxSanity:  8,
	}
}

// ValidateStamina checks if a stamina value is within Elder Sign bounds.
func (b ResourceBounds) ValidateStamina(stamina int) error {
	if stamina < b.MinStamina {
		return ErrInsufficientStamina
	}
	if stamina > b.MaxStamina {
		return nil // Capping at max is allowed
	}
	return nil
}

// ValidateSanity checks if a sanity value is within Elder Sign bounds.
func (b ResourceBounds) ValidateSanity(sanity int) error {
	if sanity < b.MinSanity {
		return ErrInsufficientSanity
	}
	if sanity > b.MaxSanity {
		return nil // Capping at max is allowed
	}
	return nil
}

// ClampStamina ensures a stamina value stays within bounds.
func (b ResourceBounds) ClampStamina(stamina int) int {
	if stamina < b.MinStamina {
		return b.MinStamina
	}
	if stamina > b.MaxStamina {
		return b.MaxStamina
	}
	return stamina
}

// ClampSanity ensures a sanity value stays within bounds.
func (b ResourceBounds) ClampSanity(sanity int) int {
	if sanity < b.MinSanity {
		return b.MinSanity
	}
	if sanity > b.MaxSanity {
		return b.MaxSanity
	}
	return sanity
}

// ResourceEvent represents a change to investigator resources.
// Used for tracking stamina/sanity gains and losses during gameplay.
type ResourceEvent struct {
	PlayerID     string
	StaminaDelta int // Positive for gain, negative for loss
	SanityDelta  int // Positive for gain, negative for loss
	Reason       string
	IsFatal      bool // True if investigator is defeated (stamina or sanity reached 0)
}

// ApplyResourceEvent modifies stamina and sanity values based on an event.
// Returns the updated values after applying deltas and clamping to bounds.
func ApplyResourceEvent(currentStamina, currentSanity int, event ResourceEvent, bounds ResourceBounds) (int, int, error) {
	newStamina := currentStamina + event.StaminaDelta
	newSanity := currentSanity + event.SanityDelta

	// Check for fatal conditions before clamping
	if newStamina <= 0 || newSanity <= 0 {
		return 0, 0, nil // Investigator is defeated
	}

	// Clamp to bounds
	newStamina = bounds.ClampStamina(newStamina)
	newSanity = bounds.ClampSanity(newSanity)

	return newStamina, newSanity, nil
}
