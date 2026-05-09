package ui

import "testing"

func TestCalculateLayoutZones_TargetResolutions_NoOverlapOrClipping(t *testing.T) {
	testCases := []struct {
		name string
		w    int
		h    int
	}{
		{name: "minimum-800x600", w: 800, h: 600},
		{name: "hd-1280x720", w: 1280, h: 720},
		{name: "fullhd-1920x1080", w: 1920, h: 1080},
		{name: "qhd-2560x1440", w: 2560, h: 1440},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vp := &Viewport{LogicalWidth: tc.w, LogicalHeight: tc.h, Scale: 1.0}
			zones := CalculateLayoutZones(vp)
			bounds := vp.LogicalBounds()

			if !zones.StatusRail.In(bounds) {
				t.Fatalf("status rail out of bounds: %v in %v", zones.StatusRail, bounds)
			}
			if !zones.Board.In(bounds) {
				t.Fatalf("board out of bounds: %v in %v", zones.Board, bounds)
			}
			if !zones.ActionRail.In(bounds) {
				t.Fatalf("action rail out of bounds: %v in %v", zones.ActionRail, bounds)
			}

			if zones.StatusRail.Max.Y > zones.Board.Min.Y {
				t.Fatalf("status rail overlaps board: %v vs %v", zones.StatusRail, zones.Board)
			}
			if zones.Board.Max.Y > zones.ActionRail.Min.Y {
				t.Fatalf("board overlaps action rail: %v vs %v", zones.Board, zones.ActionRail)
			}
		})
	}
}

func TestCalculateLayoutZones_ReadabilityThresholds(t *testing.T) {
	vp := &Viewport{LogicalWidth: 800, LogicalHeight: 600, Scale: 1.0}
	zones := CalculateLayoutZones(vp)

	if zones.ReadabilityHint.BodyTextPx < minBodyTextPx {
		t.Fatalf("body text too small: %d", zones.ReadabilityHint.BodyTextPx)
	}
	if zones.ReadabilityHint.IconPx < minIconPx {
		t.Fatalf("icon too small: %d", zones.ReadabilityHint.IconPx)
	}
}

func TestCalculateLayoutZones_ExtremeAspectFallback(t *testing.T) {
	cases := []struct {
		name string
		w    int
		h    int
	}{
		{name: "ultrawide", w: 3440, h: 1440},
		{name: "very-tall", w: 900, h: 1600},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vp := &Viewport{LogicalWidth: tc.w, LogicalHeight: tc.h, Scale: 1.0}
			zones := CalculateLayoutZones(vp)
			if !zones.UsesLetterbox {
				t.Fatalf("expected letterbox fallback for %s (%dx%d)", tc.name, tc.w, tc.h)
			}
		})
	}
}

func TestCalculateLayoutZones_MinimumZoneHeights(t *testing.T) {
	vp := &Viewport{LogicalWidth: 800, LogicalHeight: 600, Scale: 1.0}
	zones := CalculateLayoutZones(vp)

	if zones.StatusRail.Dy() < minStatusRailHeight {
		t.Fatalf("status rail below minimum: got %d, want >= %d", zones.StatusRail.Dy(), minStatusRailHeight)
	}
	if zones.ActionRail.Dy() < minActionRailHeight {
		t.Fatalf("action rail below minimum: got %d, want >= %d", zones.ActionRail.Dy(), minActionRailHeight)
	}
	if zones.Board.Dy() < minBoardHeight {
		t.Fatalf("board zone below minimum: got %d, want >= %d", zones.Board.Dy(), minBoardHeight)
	}
}
