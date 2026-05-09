package ui

import "image"

const (
	// Minimum readable UI dimensions at the project floor resolution.
	minStatusRailHeight = 88
	minActionRailHeight = 176
	minBoardHeight      = 120
	minBodyTextPx       = 14
	minIconPx           = 18
)

// LayoutZones represents the three primary HUD zones used by the client scene.
type LayoutZones struct {
	StatusRail      image.Rectangle
	Board           image.Rectangle
	ActionRail      image.Rectangle
	UsesLetterbox   bool
	ReadabilityHint ReadabilityProfile
}

// ReadabilityProfile defines minimum sizes used by text/icon rendering.
type ReadabilityProfile struct {
	BodyTextPx int
	IconPx     int
}

// CalculateLayoutZones computes deterministic status/board/action rectangles for a viewport.
func CalculateLayoutZones(v *Viewport) LayoutZones {
	if v == nil {
		return LayoutZones{}
	}

	bounds := v.LogicalBounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return LayoutZones{}
	}

	statusHeight := clamp(height/6, minStatusRailHeight, 156)
	actionHeight := clamp(height/4, minActionRailHeight, 260)
	boardHeight := height - statusHeight - actionHeight
	if boardHeight < minBoardHeight {
		deficit := minBoardHeight - boardHeight
		actionHeight = max(minActionRailHeight, actionHeight-deficit)
		boardHeight = height - statusHeight - actionHeight
		if boardHeight < minBoardHeight {
			statusHeight = max(minStatusRailHeight, statusHeight-(minBoardHeight-boardHeight))
			boardHeight = height - statusHeight - actionHeight
		}
	}

	statusRect := image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+statusHeight)
	boardRect := image.Rect(bounds.Min.X, statusRect.Max.Y, bounds.Max.X, statusRect.Max.Y+boardHeight)
	actionRect := image.Rect(bounds.Min.X, boardRect.Max.Y, bounds.Max.X, bounds.Max.Y)

	zones := LayoutZones{
		StatusRail:      statusRect,
		Board:           boardRect,
		ActionRail:      actionRect,
		ReadabilityHint: computeReadabilityProfile(bounds.Dx(), bounds.Dy()),
	}

	// For very wide/tall screens, keep board in a predictable 16:9 envelope.
	aspect := float64(width) / float64(height)
	if aspect > 2.2 {
		targetW := int(float64(boardRect.Dy()) * 16.0 / 9.0)
		if targetW > 0 && targetW < boardRect.Dx() {
			leftPad := (boardRect.Dx() - targetW) / 2
			zones.Board = image.Rect(boardRect.Min.X+leftPad, boardRect.Min.Y, boardRect.Min.X+leftPad+targetW, boardRect.Max.Y)
			zones.UsesLetterbox = true
		}
	}
	if aspect < 1.2 {
		targetH := int(float64(boardRect.Dx()) * 9.0 / 16.0)
		if targetH > 0 && targetH < boardRect.Dy() {
			topPad := (boardRect.Dy() - targetH) / 2
			zones.Board = image.Rect(boardRect.Min.X, boardRect.Min.Y+topPad, boardRect.Max.X, boardRect.Min.Y+topPad+targetH)
			zones.UsesLetterbox = true
		}
	}

	return zones
}

func computeReadabilityProfile(width, height int) ReadabilityProfile {
	smallest := min(width, height)
	body := clamp(smallest/45, minBodyTextPx, 24)
	icon := clamp(smallest/35, minIconPx, 32)
	return ReadabilityProfile{BodyTextPx: body, IconPx: icon}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
