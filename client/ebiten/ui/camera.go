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
	direction       int
	mode            ViewMode
	visualDirection float64
	modeBlend       float64
}

// NewCamera creates a camera in pseudo-3D mode at direction 0.
func NewCamera() *Camera {
	return &Camera{direction: 0, mode: ViewModePseudo3D, visualDirection: 0, modeBlend: 1}
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
	if c == nil {
		return x, y, 1.0
	}

	c.stepTransitions()
	return c.projectInterpolated(x, y, centerX, centerY)
}

func (c *Camera) projectInterpolated(x, y, centerX, centerY float64) (float64, float64, float64) {
	blend := clamp01(c.modeBlend)
	if blend <= 0.001 {
		return x, y, 1.0
	}

	angle := c.visualDirection * (math.Pi / 4.0)
	relX := x - centerX
	relY := y - centerY
	rotx := relX*math.Cos(angle) - relY*math.Sin(angle)
	roty := relX*math.Sin(angle) + relY*math.Cos(angle)

	projectedX := centerX + rotx + roty*0.35
	projectedY := centerY + roty*0.65
	scale := clampRange(0.85+0.15*(1.0-(roty/600.0)), 0.70, 1.15)
	return x + (projectedX-x)*blend, y + (projectedY-y)*blend, 1 + (scale-1)*blend
}

func (c *Camera) stepTransitions() {
	targetDir := float64(c.direction)
	delta := shortestDirectionDelta(c.visualDirection, targetDir)
	c.visualDirection += delta * 0.22

	targetBlend := 1.0
	if c.mode == ViewModeTopDown {
		targetBlend = 0
	}
	c.modeBlend += (targetBlend - c.modeBlend) * 0.25
}

func shortestDirectionDelta(current, target float64) float64 {
	delta := target - current
	for delta > 4 {
		delta -= 8
	}
	for delta < -4 {
		delta += 8
	}
	return delta
}

func clamp01(v float64) float64 {
	return clampRange(v, 0, 1)
}

func clampRange(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
