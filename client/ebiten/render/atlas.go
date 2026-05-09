// Package render provides a layered 2-D sprite system for the Arkham Horror
// Ebitengine client. It organises draw calls into five ordered layers so that
// board elements, tokens, effects, UI, and animations always composite in the
// correct z-order.
package render

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// spritesheetPNG is the 512×512 sprite-sheet atlas embedded at build time.
// Artists replace assets/sprites.png to update visuals without any code change.
//
//go:embed assets/sprites.png
var spritesheetPNG []byte

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

// spriteCoords is the authoritative sprite-sheet coordinate table.
// Each entry maps a SpriteID to the pixel region it occupies in sprites.png.
// The atlas is laid out as a single row of 64×64 tiles on a 512×512 sheet:
//
//	col 0  col 1       col 2         col 3        col 4         col 5       col 6        col 7
//	Bg     Downtown    University    Rivertown     Northside     PlayerToken DoomMarker   ActionOverlay
//
// Updating real art assets only requires replacing assets/sprites.png while
// keeping the pixel regions in this table the same.
var spriteCoords = [spriteCount]spriteRect{
	SpriteBackground:         {x: 0, y: 0, w: 64, h: 64},
	SpriteLocationDowntown:   {x: 64, y: 0, w: 64, h: 64},
	SpriteLocationUniversity: {x: 128, y: 0, w: 64, h: 64},
	SpriteLocationRivertown:  {x: 192, y: 0, w: 64, h: 64},
	SpriteLocationNorthside:  {x: 256, y: 0, w: 64, h: 64},
	SpritePlayerToken:        {x: 320, y: 0, w: 64, h: 64},
	SpriteDoomMarker:         {x: 384, y: 0, w: 64, h: 64},
	SpriteActionOverlay:      {x: 448, y: 0, w: 64, h: 64},
}

// Atlas manages the single shared texture image used for all game sprites.
// The atlas avoids repeated texture uploads and keeps draw-call counts low.
//
// At startup generateAtlas decodes the embedded assets/sprites.png sprite
// sheet and records each sprite's pixel region from the spriteCoords table.
// Replace assets/sprites.png with production art to update visuals without
// any code change; the coordinate table in spriteCoords must be kept in sync
// with the art layout.
type Atlas struct {
	image    *ebiten.Image
	entries  [spriteCount]spriteRect
	sources  [spriteCount]*ebiten.Image
	resolver AtlasAssetResolver
}

// NewAtlas creates an Atlas by calling generateAtlas to build or load the
// texture image. The returned Atlas is ready to use immediately.
func NewAtlas() *Atlas {
	return NewAtlasWithResolver(NewEmbeddedAtlasResolver())
}

// NewAtlasWithResolver creates an Atlas using the provided asset resolver.
func NewAtlasWithResolver(resolver AtlasAssetResolver) *Atlas {
	if resolver == nil {
		resolver = NewEmbeddedAtlasResolver()
	}
	a := &Atlas{}
	a.resolver = resolver
	a.generateAtlas()
	return a
}

// DrawSprite blits the named sprite onto dst at pixel position (dx, dy).
// scaleX and scaleY scale the sprite around its top-left origin before translation.
// An optional colour tint is applied via ebiten's ColorScale.
func (a *Atlas) DrawSprite(dst *ebiten.Image, id SpriteID, dx, dy, scaleX, scaleY float64, tint color.RGBA) {
	if id < 0 || int(id) >= len(a.entries) {
		return
	}
	r := a.entries[id]
	src := a.sources[id]
	if src == nil && a.image != nil {
		src = a.image.SubImage(imageRect(r.x, r.y, r.w, r.h)).(*ebiten.Image)
		a.sources[id] = src
	}
	if src == nil {
		return
	}

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(dx, dy)
	if tint != (color.RGBA{}) {
		op.ColorScale.SetR(float32(tint.R) / 255)
		op.ColorScale.SetG(float32(tint.G) / 255)
		op.ColorScale.SetB(float32(tint.B) / 255)
		op.ColorScale.SetA(float32(tint.A) / 255)
	}
	dst.DrawImage(src, &op)
}

// generateAtlas populates the atlas image from the embedded sprites.png sprite
// sheet and records each sprite's position from the spriteCoords table.
// The fallback path generates solid-colour placeholders only when PNG decoding
// fails (e.g. a corrupt embed), which preserves correctness during development.
func (a *Atlas) generateAtlas() {
	if a.resolver == nil {
		a.resolver = NewEmbeddedAtlasResolver()
	}

	data, err := a.resolver.SpriteSheetPNG()
	if err != nil {
		log.Printf("render: resolver failed to provide sprite sheet, using placeholder atlas: %v", err)
		a.generatePlaceholderAtlas()
		return
	}

	img, err := decodeSpritesheet(data)
	if err == nil {
		a.image = img
		a.entries = a.resolver.SpriteCoordinates()
		a.initSources()
		return
	}
	// Log the decode failure so developers can diagnose corrupt or missing embeds.
	log.Printf("render: failed to decode embedded sprite sheet, using placeholder atlas: %v", err)
	// Fallback: programmatically generated placeholder tiles.
	a.generatePlaceholderAtlas()
}

// decodeSpritesheet decodes a PNG-encoded sprite sheet into an *ebiten.Image.
func decodeSpritesheet(data []byte) (*ebiten.Image, error) {
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(decoded), nil
}

// generatePlaceholderAtlas creates a solid-colour placeholder atlas when the
// embedded sprite sheet cannot be decoded. Each sprite is a distinct colour so
// that all game elements remain visually distinguishable during development.
func (a *Atlas) generatePlaceholderAtlas() {
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
	a.initSources()
}

func (a *Atlas) initSources() {
	for id := SpriteID(0); id < spriteCount; id++ {
		r := a.entries[id]
		a.sources[id] = a.image.SubImage(imageRect(r.x, r.y, r.w, r.h)).(*ebiten.Image)
	}
}

// imageRect returns an image.Rectangle for use in SubImage calls.
func imageRect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}
