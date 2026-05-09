package ui

import "math"

// ViewMode controls whether board rendering is top-down or pseudo-3D.
type ViewMode string

const (
	ViewModeTopDown  ViewMode = "topdown"
	ViewModePseudo3D ViewMode = "pseudo3d"
)

// Camera tracks board orientation with 8 directional presets.
type Camera struct {
	direction int
	mode      ViewMode
}

// NewCamera creates a camera in pseudo-3D mode at direction 0.
func NewCamera() *Camera {
	return &Camera{direction: 0, mode: ViewModePseudo3D}
}

// Direction returns the active directional preset in [0, 7].
func (c *Camera) Direction() int {
	if c == nil {
		return 0
	}
	return c.direction
}

// Mode returns the current view mode.
func (c *Camera) Mode() ViewMode {
	if c == nil {
		return ViewModeTopDown
	}
	return c.mode
}

// OrbitCW rotates the camera clockwise to the next preset.
func (c *Camera) OrbitCW() {
	if c == nil {
		return
	}
	c.direction = (c.direction + 1) % 8
}

// OrbitCCW rotates the camera counter-clockwise to the previous preset.
func (c *Camera) OrbitCCW() {
	if c == nil {
		return
	}
	c.direction = (c.direction + 7) % 8
}

// ToggleViewMode switches between pseudo-3D and top-down fallback modes.
func (c *Camera) ToggleViewMode() {
	if c == nil {
		return
	}
	if c.mode == ViewModeTopDown {
		c.mode = ViewModePseudo3D
		return
	}
	c.mode = ViewModeTopDown
}

// Project transforms board-space coordinates into screen-space coordinates.
// The board center should be provided so transforms preserve readability.
func (c *Camera) Project(x, y, centerX, centerY float64) (float64, float64, float64) {
	if c == nil || c.mode == ViewModeTopDown {
		return x, y, 1.0
	}

	angle := float64(c.direction) * (math.Pi / 4.0)
	relX := x - centerX
	relY := y - centerY

	rotx := relX*math.Cos(angle) - relY*math.Sin(angle)
	roty := relX*math.Sin(angle) + relY*math.Cos(angle)

	// Isometric-like skew with mild vertical compression to keep labels readable.
	px := centerX + rotx + roty*0.35
	py := centerY + roty*0.65
	scale := 0.85 + 0.15*(1.0-(roty/600.0))
	if scale < 0.70 {
		scale = 0.70
	}
	if scale > 1.15 {
		scale = 1.15
	}
	return px, py, scale
}
