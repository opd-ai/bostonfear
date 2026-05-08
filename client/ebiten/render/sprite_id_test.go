// Package render — headless-safe unit tests for atlas sprite IDs and
// location-to-sprite mapping. These tests exercise pure logic (no Ebitengine
// draw calls) but must run with a virtual framebuffer because the render
// package imports Ebitengine, which initialises GLFW on package init.
//
// Run with: DISPLAY=:99 xvfb-run -a go test -race -tags=requires_display ./client/ebiten/render/...
// Tests that allocate Ebitengine images are in atlas_test.go (same build tag).

//go:build requires_display

package render

import "testing"

// TestSpriteIDs_Range verifies that all exported SpriteID constants are in the
// valid range [0, spriteCount) so that DrawSprite never receives an out-of-range
// value from a correctly-written caller.
func TestSpriteIDs_Range(t *testing.T) {
	ids := []struct {
		name string
		id   SpriteID
	}{
		{"SpriteBackground", SpriteBackground},
		{"SpriteLocationDowntown", SpriteLocationDowntown},
		{"SpriteLocationUniversity", SpriteLocationUniversity},
		{"SpriteLocationRivertown", SpriteLocationRivertown},
		{"SpriteLocationNorthside", SpriteLocationNorthside},
		{"SpritePlayerToken", SpritePlayerToken},
		{"SpriteDoomMarker", SpriteDoomMarker},
		{"SpriteActionOverlay", SpriteActionOverlay},
	}
	for _, tc := range ids {
		if tc.id < 0 || tc.id >= spriteCount {
			t.Errorf("%s = %d: out of range [0, %d)", tc.name, tc.id, spriteCount)
		}
	}
}

// TestSpriteIDs_Unique verifies that no two exported SpriteID constants share
// the same numeric value (duplicate IDs would alias atlas entries).
func TestSpriteIDs_Unique(t *testing.T) {
	ids := []struct {
		name string
		id   SpriteID
	}{
		{"SpriteBackground", SpriteBackground},
		{"SpriteLocationDowntown", SpriteLocationDowntown},
		{"SpriteLocationUniversity", SpriteLocationUniversity},
		{"SpriteLocationRivertown", SpriteLocationRivertown},
		{"SpriteLocationNorthside", SpriteLocationNorthside},
		{"SpritePlayerToken", SpritePlayerToken},
		{"SpriteDoomMarker", SpriteDoomMarker},
		{"SpriteActionOverlay", SpriteActionOverlay},
	}
	seen := make(map[SpriteID]string, len(ids))
	for _, tc := range ids {
		if prev, dup := seen[tc.id]; dup {
			t.Errorf("SpriteID collision: %s and %s both equal %d", prev, tc.name, tc.id)
		}
		seen[tc.id] = tc.name
	}
}

// TestLocationSpriteID_AllLocations verifies that LocationSpriteID maps each
// of the four canonical Arkham Horror neighbourhoods to its dedicated sprite.
func TestLocationSpriteID_AllLocations(t *testing.T) {
	cases := map[string]SpriteID{
		"Downtown":   SpriteLocationDowntown,
		"University": SpriteLocationUniversity,
		"Rivertown":  SpriteLocationRivertown,
		"Northside":  SpriteLocationNorthside,
	}
	for loc, want := range cases {
		if got := LocationSpriteID(loc); got != want {
			t.Errorf("LocationSpriteID(%q) = %d, want %d", loc, got, want)
		}
	}
}

// TestLocationSpriteID_FallbackToBackground verifies that an unrecognised
// location name returns SpriteBackground (the safe fallback).
func TestLocationSpriteID_FallbackToBackground(t *testing.T) {
	unknowns := []string{"", "Dunwich", "Arkham", "BeyondTheVeil"}
	for _, loc := range unknowns {
		if got := LocationSpriteID(loc); got != SpriteBackground {
			t.Errorf("LocationSpriteID(%q) = %d, want SpriteBackground (%d)",
				loc, got, SpriteBackground)
		}
	}
}

// TestImageRect_Correctness verifies the helper used by DrawSprite and generateAtlas
// to produce correct image.Rectangle values from (x, y, w, h) parameters.
func TestImageRect_Correctness(t *testing.T) {
	cases := []struct {
		x, y, w, h int
		wantMinX   int
		wantMinY   int
		wantMaxX   int
		wantMaxY   int
	}{
		{0, 0, 64, 64, 0, 0, 64, 64},
		{128, 64, 64, 64, 128, 64, 192, 128},
		{10, 20, 30, 40, 10, 20, 40, 60},
	}
	for _, tc := range cases {
		r := imageRect(tc.x, tc.y, tc.w, tc.h)
		if r.Min.X != tc.wantMinX || r.Min.Y != tc.wantMinY ||
			r.Max.X != tc.wantMaxX || r.Max.Y != tc.wantMaxY {
			t.Errorf("imageRect(%d,%d,%d,%d) = %v, want (%d,%d)-(%d,%d)",
				tc.x, tc.y, tc.w, tc.h, r,
				tc.wantMinX, tc.wantMinY, tc.wantMaxX, tc.wantMaxY)
		}
	}
}

// TestLayerIDs_Range verifies that LayerID constants are in [0, layerCount).
func TestLayerIDs_Range(t *testing.T) {
	layers := []struct {
		name string
		id   LayerID
	}{
		{"LayerBoard", LayerBoard},
		{"LayerTokens", LayerTokens},
		{"LayerEffects", LayerEffects},
		{"LayerUI", LayerUI},
		{"LayerAnimation", LayerAnimation},
	}
	for _, tc := range layers {
		if tc.id < 0 || tc.id >= layerCount {
			t.Errorf("%s = %d: out of range [0, %d)", tc.name, tc.id, layerCount)
		}
	}
}

// TestSpriteCoords_Coverage verifies that every exported SpriteID has an entry
// in the spriteCoords table with a non-zero tile size.
func TestSpriteCoords_Coverage(t *testing.T) {
	ids := []struct {
		name string
		id   SpriteID
	}{
		{"SpriteBackground", SpriteBackground},
		{"SpriteLocationDowntown", SpriteLocationDowntown},
		{"SpriteLocationUniversity", SpriteLocationUniversity},
		{"SpriteLocationRivertown", SpriteLocationRivertown},
		{"SpriteLocationNorthside", SpriteLocationNorthside},
		{"SpritePlayerToken", SpritePlayerToken},
		{"SpriteDoomMarker", SpriteDoomMarker},
		{"SpriteActionOverlay", SpriteActionOverlay},
	}
	for _, tc := range ids {
		r := spriteCoords[tc.id]
		if r.w <= 0 || r.h <= 0 {
			t.Errorf("%s: spriteCoords entry has zero or negative size: w=%d h=%d",
				tc.name, r.w, r.h)
		}
	}
}

// TestSpriteCoords_WithinAtlas verifies that every sprite region in the coordinate
// table lies entirely within the 512×512 atlas image dimensions.
// It iterates over the full [0, spriteCount) range (all entries, not just named
// exports) so that new SpriteID values added to the iota block are automatically
// validated before they receive an explicit name.
func TestSpriteCoords_WithinAtlas(t *testing.T) {
	const atlasW, atlasH = 512, 512
	for id := SpriteID(0); id < spriteCount; id++ {
		r := spriteCoords[id]
		if r.x < 0 || r.y < 0 || r.x+r.w > atlasW || r.y+r.h > atlasH {
			t.Errorf("spriteCoords[%d]: region (%d,%d,%d,%d) outside atlas %dx%d",
				id, r.x, r.y, r.w, r.h, atlasW, atlasH)
		}
	}
}

// TestDecodeSpritesheet_EmbeddedPNG verifies that the embedded sprites.png
// decodes without error and produces an image with the expected atlas dimensions.
func TestDecodeSpritesheet_EmbeddedPNG(t *testing.T) {
	img, err := decodeSpritesheet(spritesheetPNG)
	if err != nil {
		t.Fatalf("decodeSpritesheet(spritesheetPNG) error: %v", err)
	}
	if img == nil {
		t.Fatal("decodeSpritesheet returned nil image")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 512 || bounds.Dy() != 512 {
		t.Errorf("spritesheet size = %dx%d, want 512x512", bounds.Dx(), bounds.Dy())
	}
}

// TestDecodeSpritesheet_InvalidData verifies that decodeSpritesheet returns an
// error (and not nil, nil) for invalid PNG data.
func TestDecodeSpritesheet_InvalidData(t *testing.T) {
	_, err := decodeSpritesheet([]byte("not a PNG"))
	if err == nil {
		t.Error("decodeSpritesheet(invalid data) should return an error, got nil")
	}
}
