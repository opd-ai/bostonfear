package ui

// ProjectLabelPosition returns a projected label anchor near the top center
// of a projected board tile.
func ProjectLabelPosition(tileX, tileY, tileW, tileH float64, view *BoardView) (float64, float64) {
	if view == nil {
		return tileX + tileW/2, tileY - 10
	}
	px, py, _ := view.ProjectPoint(tileX+tileW/2, tileY)
	return px, py - 10
}
