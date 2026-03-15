// Package render provides a layered 2-D sprite system for the Arkham Horror
// Ebitengine client. It organises draw calls into five ordered layers so that
// board elements, tokens, effects, UI, and animations always composite in the
// correct z-order.
package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// SpriteID identifies a named entry in the texture atlas.
type SpriteID int

const (
	SpriteBackground SpriteID = iota // full-board background
	SpriteLocationDowntown
	SpriteLocationUniversity
	SpriteLocationRivertown
	SpriteLocationNorthside
	SpritePlayerToken   // generic player token (tinted per player)
	SpriteDoomMarker    // doom-track filled segment
	SpriteActionOverlay // semi-transparent action-hint overlay
	spriteCount
)

// spriteRect describes the position and size of one sprite in the atlas image.
type spriteRect struct {
	x, y, w, h int
}

// Atlas manages the single shared texture image used for all game sprites.
// The atlas avoids repeated texture uploads and keeps draw-call counts low.
//
// In this placeholder implementation every sprite is a solid-colour rectangle
// rendered onto a shared offscreen image. Real asset integration can replace
// the generateAtlas function without changing any other render code.
type Atlas struct {
	image   *ebiten.Image
	entries [spriteCount]spriteRect
}

// NewAtlas creates an Atlas by calling generateAtlas to build or load the
// texture image. The returned Atlas is ready to use immediately.
func NewAtlas() *Atlas {
	a := &Atlas{}
	a.generateAtlas()
	return a
}

// DrawSprite blits the named sprite onto dst at pixel position (dx, dy).
// An optional colour tint is applied via ebiten's ColorScale.
func (a *Atlas) DrawSprite(dst *ebiten.Image, id SpriteID, dx, dy float64, tint color.RGBA) {
	if id < 0 || int(id) >= len(a.entries) {
		return
	}
	r := a.entries[id]

	src := a.image.SubImage(imageRect(r.x, r.y, r.w, r.h)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(dx, dy)
	if tint != (color.RGBA{}) {
		op.ColorScale.SetR(float32(tint.R) / 255)
		op.ColorScale.SetG(float32(tint.G) / 255)
		op.ColorScale.SetB(float32(tint.B) / 255)
		op.ColorScale.SetA(float32(tint.A) / 255)
	}
	dst.DrawImage(src, op)
}

// generateAtlas populates the atlas image with solid-colour placeholder tiles.
// Each sprite occupies a 64×64 region on a 512×512 texture.
func (a *Atlas) generateAtlas() {
	const tileSize = 64
	const cols = 8 // 512 / 64
	img := ebiten.NewImage(512, 512)

	placeholderColours := [spriteCount]color.RGBA{
		SpriteBackground:         {R: 15, G: 15, B: 25, A: 255},
		SpriteLocationDowntown:   {R: 60, G: 80, B: 140, A: 255},
		SpriteLocationUniversity: {R: 60, G: 120, B: 80, A: 255},
		SpriteLocationRivertown:  {R: 120, G: 60, B: 60, A: 255},
		SpriteLocationNorthside:  {R: 100, G: 80, B: 140, A: 255},
		SpritePlayerToken:        {R: 255, G: 220, B: 50, A: 255},
		SpriteDoomMarker:         {R: 200, G: 40, B: 40, A: 255},
		SpriteActionOverlay:      {R: 255, G: 255, B: 255, A: 80},
	}

	for id := SpriteID(0); id < spriteCount; id++ {
		col := int(id) % cols
		row := int(id) / cols
		rx := col * tileSize
		ry := row * tileSize
		a.entries[id] = spriteRect{rx, ry, tileSize, tileSize}

		tile := img.SubImage(imageRect(rx, ry, tileSize, tileSize)).(*ebiten.Image)
		tile.Fill(placeholderColours[id])
	}

	a.image = img
}

// imageRect returns an image.Rectangle for use in SubImage calls.
func imageRect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}
