package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"
)

// LegacyAtlasResolver implements AtlasAssetResolver using the hardcoded
// spriteCoords table and a procedurally generated placeholder atlas.
// It is the rollback path activated by BOSTONFEAR_USE_LEGACY_ASSETS=1.
type LegacyAtlasResolver struct {
	once       sync.Once
	sheetPNG   []byte
	coords     [spriteCount]spriteRect
	resolveErr error
}

// NewLegacyAtlasResolver returns an AtlasAssetResolver backed by the
// hardcoded spriteCoords table and a procedurally generated sprite sheet.
func NewLegacyAtlasResolver() AtlasAssetResolver {
	return &LegacyAtlasResolver{}
}

// SpriteSheetPNG returns a procedurally generated PNG using hardcoded coords.
func (r *LegacyAtlasResolver) SpriteSheetPNG() ([]byte, error) {
	r.once.Do(r.build)
	if r.resolveErr != nil {
		return nil, r.resolveErr
	}
	return r.sheetPNG, nil
}

// SpriteCoordinates returns the hardcoded spriteCoords table.
func (r *LegacyAtlasResolver) SpriteCoordinates() [spriteCount]spriteRect {
	r.once.Do(r.build)
	return r.coords
}

func (r *LegacyAtlasResolver) build() {
	r.coords = spriteCoords

	// Determine atlas dimensions from the hardcoded table.
	totalW := 0
	maxH := 0
	for i := SpriteID(0); i < spriteCount; i++ {
		rc := spriteCoords[i]
		right := rc.x + rc.w
		if right > totalW {
			totalW = right
		}
		bottom := rc.y + rc.h
		if bottom > maxH {
			maxH = bottom
		}
	}
	if totalW <= 0 || maxH <= 0 {
		r.resolveErr = fmt.Errorf("legacy resolver: invalid sprite coord dimensions")
		return
	}

	atlas := image.NewRGBA(image.Rect(0, 0, totalW, maxH))

	// Paint each sprite region with a distinct placeholder colour.
	placeholderColors := [spriteCount]color.RGBA{
		{R: 60, G: 60, B: 80, A: 255},    // Background — dark blue-grey
		{R: 40, G: 100, B: 160, A: 255},  // Downtown — blue
		{R: 60, G: 140, B: 60, A: 255},   // University — green
		{R: 160, G: 80, B: 40, A: 255},   // Rivertown — brown
		{R: 100, G: 60, B: 140, A: 255},  // Northside — purple
		{R: 220, G: 180, B: 60, A: 255},  // PlayerToken — gold
		{R: 200, G: 40, B: 40, A: 255},   // DoomMarker — red
		{R: 180, G: 180, B: 180, A: 128}, // ActionOverlay — translucent grey
	}

	for id := SpriteID(0); id < spriteCount; id++ {
		rc := spriteCoords[id]
		rect := image.Rect(rc.x, rc.y, rc.x+rc.w, rc.y+rc.h)
		var col color.RGBA
		if int(id) < len(placeholderColors) {
			col = placeholderColors[id]
		} else {
			col = color.RGBA{R: 128, G: 128, B: 128, A: 255}
		}
		draw.Draw(atlas, rect, &image.Uniform{C: col}, image.Point{}, draw.Src)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, atlas); err != nil {
		r.resolveErr = fmt.Errorf("legacy resolver: encode atlas png: %w", err)
		return
	}
	r.sheetPNG = buf.Bytes()
}
