// Package render — unit tests for the sprite atlas and compositing layer system.
//
// Tests that allocate Ebitengine images require a display context.
// Build with -tags=requires_display and DISPLAY=:99 (or any accessible X11
// display) in headless CI environments; without the tag these files are
// excluded so that `go test ./...` succeeds in pure-headless environments
// without triggering the GLFW init() panic.

//go:build requires_display

package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestAtlas_InitDoesNotPanic verifies that NewAtlas() initialises the texture atlas
// without panicking. This is the regression test for nil-atlas rendering crashes.
func TestAtlas_InitDoesNotPanic(t *testing.T) {
	atlas := NewAtlas()
	if atlas == nil {
		t.Fatal("NewAtlas() returned nil")
	}
	if atlas.image == nil {
		t.Error("Atlas.image is nil after NewAtlas()")
	}
}

// TestAtlas_DrawSprite_OutOfBounds verifies that DrawSprite silently ignores
// sprite IDs that are outside the valid range (negative or >= spriteCount).
func TestAtlas_DrawSprite_OutOfBounds(t *testing.T) {
	atlas := NewAtlas()
	dst := ebiten.NewImage(64, 64)
	// Must not panic for an invalid sprite ID.
	atlas.DrawSprite(dst, SpriteID(-1), 0, 0, 1, 1, color.RGBA{})
	atlas.DrawSprite(dst, SpriteID(9999), 0, 0, 1, 1, color.RGBA{})
}

// TestAtlas_DrawSprite_ValidID verifies that DrawSprite does not panic when
// called with a valid SpriteID and an optional colour tint.
func TestAtlas_DrawSprite_ValidID(t *testing.T) {
	atlas := NewAtlas()
	dst := ebiten.NewImage(200, 200)
	// Background sprite at origin, no tint.
	atlas.DrawSprite(dst, SpriteBackground, 0, 0, 1, 1, color.RGBA{})
	// Player token with a tint colour.
	atlas.DrawSprite(dst, SpritePlayerToken, 10, 10, 1, 1, color.RGBA{R: 255, G: 0, B: 0, A: 128})
}

// TestImageRect verifies that imageRect constructs the correct image.Rectangle.
func TestImageRect(t *testing.T) {
	got := imageRect(10, 20, 30, 40)
	want := image.Rect(10, 20, 40, 60)
	if got != want {
		t.Errorf("imageRect(10,20,30,40) = %v, want %v", got, want)
	}
}

// TestLocationSpriteID_KnownLocations verifies that each of the four
// Arkham Horror neighbourhoods maps to its own distinct sprite.
func TestLocationSpriteID_KnownLocations(t *testing.T) {
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

// TestLocationSpriteID_Unknown verifies that an unrecognised location name
// falls back to the background sprite.
func TestLocationSpriteID_Unknown(t *testing.T) {
	if got := LocationSpriteID("BeyondTheVeil"); got != SpriteBackground {
		t.Errorf("LocationSpriteID(unknown) = %d, want SpriteBackground (%d)", got, SpriteBackground)
	}
}

// TestNewCompositor_NotNil verifies that NewCompositor returns a usable compositor.
func TestNewCompositor_NotNil(t *testing.T) {
	c := NewCompositor()
	if c == nil {
		t.Fatal("NewCompositor() returned nil")
	}
	// Atlas must be initialised.
	if c.Atlas() == nil {
		t.Error("Compositor.Atlas() returned nil after construction")
	}
}

// TestCompositor_Flush_EmptyDoesNotPanic verifies that flushing an empty
// compositor (no enqueued draw commands) does not panic.
func TestCompositor_Flush_EmptyDoesNotPanic(t *testing.T) {
	c := NewCompositor()
	screen := ebiten.NewImage(800, 600)
	c.Flush(screen) // must not panic
}
