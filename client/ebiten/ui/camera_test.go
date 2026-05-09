package ui

import "testing"

func TestCamera_OrbitCyclesEightDirections(t *testing.T) {
	c := NewCamera()
	for i := 0; i < 8; i++ {
		c.OrbitCW()
	}
	if c.Direction() != 0 {
		t.Fatalf("direction after full cycle = %d, want 0", c.Direction())
	}

	c.OrbitCCW()
	if c.Direction() != 7 {
		t.Fatalf("direction after CCW from 0 = %d, want 7", c.Direction())
	}
}

func TestCamera_ToggleViewMode(t *testing.T) {
	c := NewCamera()
	if c.Mode() != ViewModePseudo3D {
		t.Fatalf("default mode = %s, want %s", c.Mode(), ViewModePseudo3D)
	}
	c.ToggleViewMode()
	if c.Mode() != ViewModeTopDown {
		t.Fatalf("mode after toggle = %s, want %s", c.Mode(), ViewModeTopDown)
	}
}
