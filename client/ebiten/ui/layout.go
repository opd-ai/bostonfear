package ui

import "image"

// Viewport represents the logical and physical screen dimensions with safe-area insets.
// All game coordinates are in logical units; rendering scales to physical device pixels.
type Viewport struct {
	LogicalWidth   int
	LogicalHeight  int
	PhysicalWidth  int
	PhysicalHeight int
	SafeArea       SafeArea
	Scale          float64
}

// SafeArea describes regions to avoid on displays with notches, home bars, or rounded corners.
type SafeArea struct {
	Top    int
	Bottom int
	Left   int
	Right  int
}

// LogicalBounds returns the safe-area-adjusted game bounds in logical coordinates.
func (v *Viewport) LogicalBounds() image.Rectangle {
	top := int(float64(v.SafeArea.Top) / v.Scale)
	bottom := int(float64(v.SafeArea.Bottom) / v.Scale)
	left := int(float64(v.SafeArea.Left) / v.Scale)
	right := int(float64(v.SafeArea.Right) / v.Scale)

	return image.Rect(
		left,
		top,
		v.LogicalWidth-right,
		v.LogicalHeight-bottom,
	)
}

// ToPhysical converts a logical coordinate to physical screen pixels.
func (v *Viewport) ToPhysical(logicalX, logicalY float64) (int, int) {
	return int(logicalX * v.Scale), int(logicalY * v.Scale)
}

// ToLogical converts physical screen pixels to logical game coordinates.
func (v *Viewport) ToLogical(physicalX, physicalY int) (float64, float64) {
	return float64(physicalX) / v.Scale, float64(physicalY) / v.Scale
}

// Anchor defines how a UI element is positioned relative to viewport edges.
type Anchor int

const (
	AnchorTopLeft Anchor = iota
	AnchorTopCenter
	AnchorTopRight
	AnchorCenterLeft
	AnchorCenter
	AnchorCenterRight
	AnchorBottomLeft
	AnchorBottomCenter
	AnchorBottomRight
)

// Constraint defines size and position rules for a UI region.
type Constraint struct {
	Anchor        Anchor
	OffsetX       float64
	OffsetY       float64
	Width         float64
	Height        float64
	PaddingTop    float64
	PaddingBottom float64
	PaddingLeft   float64
	PaddingRight  float64
}

// Bounds computes the final position and size of a UI region within the viewport.
func (c *Constraint) Bounds(v *Viewport) image.Rectangle {
	bounds := v.LogicalBounds()
	cx := bounds.Dx() / 2
	cy := bounds.Dy() / 2

	var x, y int
	switch c.Anchor {
	case AnchorTopLeft:
		x, y = 0, 0
	case AnchorTopCenter:
		x, y = cx, 0
	case AnchorTopRight:
		x, y = bounds.Dx(), 0
	case AnchorCenterLeft:
		x, y = 0, cy
	case AnchorCenter:
		x, y = cx, cy
	case AnchorCenterRight:
		x, y = bounds.Dx(), cy
	case AnchorBottomLeft:
		x, y = 0, bounds.Dy()
	case AnchorBottomCenter:
		x, y = cx, bounds.Dy()
	case AnchorBottomRight:
		x, y = bounds.Dx(), bounds.Dy()
	}

	x = bounds.Min.X + x + int(c.OffsetX)
	y = bounds.Min.Y + y + int(c.OffsetY)
	w := int(c.Width)
	h := int(c.Height)

	if w == 0 {
		w = bounds.Dx() - int(c.PaddingLeft+c.PaddingRight)
	}
	if h == 0 {
		h = bounds.Dy() - int(c.PaddingTop+c.PaddingBottom)
	}

	return image.Rect(x, y, x+w, y+h)
}
