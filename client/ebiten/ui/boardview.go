package ui

// BoardView projects board nodes/rectangles for rendering.
type BoardView struct {
	camera        *Camera
	centerX       float64
	centerY       float64
	screenOffsetX float64 // horizontal screen-space shift applied after projection
}

// NewBoardView creates a projection helper for a camera.
func NewBoardView(camera *Camera, centerX, centerY float64) *BoardView {
	return &BoardView{camera: camera, centerX: centerX, centerY: centerY}
}

// SetScreenOffsetX updates the horizontal screen-space shift applied after
// projection. Caller is responsible for computing the correct offset (e.g. to
// centre the board in a wider-than-base landscape canvas).
func (bv *BoardView) SetScreenOffsetX(dx float64) {
	if bv != nil {
		bv.screenOffsetX = dx
	}
}

// ProjectPoint maps a board-space point to screen-space.
func (bv *BoardView) ProjectPoint(x, y float64) (float64, float64, float64) {
	if bv == nil {
		return x, y, 1.0
	}
	sx, sy, scale := bv.camera.Project(x, y, bv.centerX, bv.centerY)
	return sx + bv.screenOffsetX, sy, scale
}
