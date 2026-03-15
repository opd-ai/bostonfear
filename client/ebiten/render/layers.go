package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LayerID identifies one of the five compositing layers drawn in z-order.
type LayerID int

const (
	// LayerBoard is the lowest layer: static board tiles and location labels.
	LayerBoard LayerID = iota
	// LayerTokens draws investigator and enemy tokens on top of the board.
	LayerTokens
	// LayerEffects draws transient visual effects (dice flashes, doom pulses).
	LayerEffects
	// LayerUI draws the HUD overlay: doom bar, resource panels, turn indicator.
	LayerUI
	// LayerAnimation draws frame-by-frame animations above everything else.
	LayerAnimation

	layerCount
)

// DrawCmd is a single draw instruction queued on a layer.
type DrawCmd struct {
	// SpriteID selects the atlas tile to draw.
	Sprite SpriteID
	// X, Y are the top-left destination pixel coordinates.
	X, Y float64
	// Tint optionally modulates the sprite colour. Zero value = no tint.
	Tint color.RGBA
	// ScaleX, ScaleY default to 1.0 when zero.
	ScaleX, ScaleY float64
}

// Renderer manages the five-layer compositing pipeline.
// Each frame, callers enqueue DrawCmds on the appropriate layer via Enqueue,
// then call Flush to composite all layers onto the screen in z-order.
type Renderer struct {
	atlas  *Atlas
	layers [layerCount][]DrawCmd
}

// NewRenderer allocates a Renderer with a shared texture atlas.
func NewRenderer() *Renderer {
	return &Renderer{
		atlas: NewAtlas(),
	}
}

// Enqueue adds a draw command to the given layer's queue for this frame.
func (r *Renderer) Enqueue(layer LayerID, cmd DrawCmd) {
	if layer < 0 || layer >= layerCount {
		return
	}
	// Default scale to 1.0 when callers leave it at zero.
	if cmd.ScaleX == 0 {
		cmd.ScaleX = 1
	}
	if cmd.ScaleY == 0 {
		cmd.ScaleY = 1
	}
	r.layers[layer] = append(r.layers[layer], cmd)
}

// Flush composites all queued layers onto screen in ascending LayerID order,
// then resets the per-frame queues. Call once per Draw frame.
func (r *Renderer) Flush(screen *ebiten.Image) {
	for id := LayerID(0); id < layerCount; id++ {
		for _, cmd := range r.layers[id] {
			r.atlas.DrawSprite(screen, cmd.Sprite, cmd.X, cmd.Y, cmd.Tint)
		}
		// Reset slice length but keep capacity to avoid allocations next frame.
		r.layers[id] = r.layers[id][:0]
	}
}

// Atlas returns the shared texture atlas for callers that need direct access.
func (r *Renderer) Atlas() *Atlas {
	return r.atlas
}

// LocationSpriteID returns the atlas SpriteID for the given location name.
func LocationSpriteID(loc string) SpriteID {
	switch loc {
	case "Downtown":
		return SpriteLocationDowntown
	case "University":
		return SpriteLocationUniversity
	case "Rivertown":
		return SpriteLocationRivertown
	case "Northside":
		return SpriteLocationNorthside
	default:
		return SpriteBackground
	}
}
