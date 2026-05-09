package ui

// BoardView projects board nodes/rectangles for rendering.
type BoardView struct {
	camera  *Camera
	centerX float64
	centerY float64
}

// NewBoardView creates a projection helper for a camera.
func NewBoardView(camera *Camera, centerX, centerY float64) *BoardView {
	return &BoardView{camera: camera, centerX: centerX, centerY: centerY}
}

// ProjectPoint maps a board-space point to screen-space.
func (bv *BoardView) ProjectPoint(x, y float64) (float64, float64, float64) {
	if bv == nil {
		return x, y, 1.0
	}
	return bv.camera.Project(x, y, bv.centerX, bv.centerY)
}
