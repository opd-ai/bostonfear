package input

import (
	"image"
	"sort"

	uipkg "github.com/opd-ai/bostonfear/client/ebiten/ui"
)

// HitBox represents a clickable/touchable UI region with a semantic label and bounds.
type HitBox struct {
	ID           string
	Bounds       image.Rectangle
	MinTouchSize int
}

// Contains reports whether the point (in logical coordinates) is within this hit box.
func (h *HitBox) Contains(logicalX, logicalY float64) bool {
	x, y := int(logicalX), int(logicalY)
	return x >= h.Bounds.Min.X && x < h.Bounds.Max.X &&
		y >= h.Bounds.Min.Y && y < h.Bounds.Max.Y
}

// IsAccessibleTouchTarget reports whether this hit box meets minimum touch size requirements.
func (h *HitBox) IsAccessibleTouchTarget() bool {
	w := h.Bounds.Dx()
	hsize := h.Bounds.Dy()
	return w >= h.MinTouchSize && hsize >= h.MinTouchSize
}

// InputMapper manages a registry of hit boxes and performs efficient hit testing.
type InputMapper struct {
	hitboxes map[string]*HitBox
	sorted   []*HitBox
}

// NewInputMapper creates an empty hit box registry.
func NewInputMapper() *InputMapper {
	return &InputMapper{
		hitboxes: make(map[string]*HitBox),
		sorted:   []*HitBox{},
	}
}

// Register adds or updates a hit box in the registry.
func (im *InputMapper) Register(id string, bounds image.Rectangle, minTouchSize int) {
	hb := &HitBox{
		ID:           id,
		Bounds:       bounds,
		MinTouchSize: minTouchSize,
	}
	im.hitboxes[id] = hb
	im.sorted = nil
}

// Unregister removes a hit box by ID.
func (im *InputMapper) Unregister(id string) {
	delete(im.hitboxes, id)
	im.sorted = nil
}

// HitTest returns the hit box (if any) at the logical coordinate (x, y).
// Returns the topmost hit box if multiple overlap; nil if no hit.
func (im *InputMapper) HitTest(logicalX, logicalY float64) *HitBox {
	im.ensureSorted()

	for i := len(im.sorted) - 1; i >= 0; i-- {
		if im.sorted[i].Contains(logicalX, logicalY) {
			return im.sorted[i]
		}
	}
	return nil
}

// AllHitBoxes returns all registered hit boxes.
func (im *InputMapper) AllHitBoxes() []*HitBox {
	result := make([]*HitBox, 0, len(im.hitboxes))
	for _, hb := range im.hitboxes {
		result = append(result, hb)
	}
	return result
}

// InaccessibleHitBoxes returns hit boxes that do not meet minimum touch size.
func (im *InputMapper) InaccessibleHitBoxes() []*HitBox {
	var result []*HitBox
	for _, hb := range im.hitboxes {
		if !hb.IsAccessibleTouchTarget() {
			result = append(result, hb)
		}
	}
	return result
}

// ensureSorted maintains a sorted list of hit boxes for efficient top-most hit testing.
func (im *InputMapper) ensureSorted() {
	if im.sorted != nil {
		return
	}

	im.sorted = make([]*HitBox, 0, len(im.hitboxes))
	for _, hb := range im.hitboxes {
		im.sorted = append(im.sorted, hb)
	}

	sort.Slice(im.sorted, func(i, j int) bool {
		areaI := im.sorted[i].Bounds.Dx() * im.sorted[i].Bounds.Dy()
		areaJ := im.sorted[j].Bounds.Dx() * im.sorted[j].Bounds.Dy()
		return areaI < areaJ
	})
}

// CoordinateTransformer adapts between physical and logical coordinate spaces.
type CoordinateTransformer struct {
	viewport *uipkg.Viewport
}

// NewCoordinateTransformer creates a transformer for the given viewport.
func NewCoordinateTransformer(v *uipkg.Viewport) *CoordinateTransformer {
	return &CoordinateTransformer{viewport: v}
}

// PhysicalToLogical converts physical screen coordinates to logical game coordinates.
func (ct *CoordinateTransformer) PhysicalToLogical(physicalX, physicalY int) (float64, float64) {
	return ct.viewport.ToLogical(physicalX, physicalY)
}

// LogicalToPhysical converts logical coordinates to physical screen pixels.
func (ct *CoordinateTransformer) LogicalToPhysical(logicalX, logicalY float64) (int, int) {
	return ct.viewport.ToPhysical(logicalX, logicalY)
}
