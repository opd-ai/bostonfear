package ui

import "time"

// MotionPreset defines a reusable transition profile.
type MotionPreset struct {
	Name     string
	Duration time.Duration
	Easing   string
}

// MotionCatalog stores named transition presets.
type MotionCatalog struct {
	presets map[string]MotionPreset
}

// NewMotionCatalog creates default transition presets for UI consistency.
func NewMotionCatalog() *MotionCatalog {
	return &MotionCatalog{presets: map[string]MotionPreset{
		"fade-fast": {Name: "fade-fast", Duration: 120 * time.Millisecond, Easing: "ease-out"},
		"fade":      {Name: "fade", Duration: 220 * time.Millisecond, Easing: "ease-in-out"},
		"pulse":     {Name: "pulse", Duration: 600 * time.Millisecond, Easing: "ease-in-out"},
		"slide":     {Name: "slide", Duration: 180 * time.Millisecond, Easing: "ease-out"},
	}}
}

// Get returns a motion preset by name.
func (c *MotionCatalog) Get(name string) MotionPreset {
	if c == nil || c.presets == nil {
		return MotionPreset{}
	}
	return c.presets[name]
}
