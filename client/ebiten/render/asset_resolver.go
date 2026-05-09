package render

import "fmt"

// AtlasAssetResolver abstracts sprite sheet and coordinate lookup for rendering.
// Implementations can source assets from embed, filesystem, or manifest-backed providers.
type AtlasAssetResolver interface {
	SpriteSheetPNG() ([]byte, error)
	SpriteCoordinates() [spriteCount]spriteRect
}

// EmbeddedAtlasResolver is the default resolver backed by embedded atlas data.
type EmbeddedAtlasResolver struct{}

// NewEmbeddedAtlasResolver returns the default atlas resolver implementation.
func NewEmbeddedAtlasResolver() AtlasAssetResolver {
	return EmbeddedAtlasResolver{}
}

// SpriteSheetPNG returns the embedded PNG sprite sheet bytes.
func (EmbeddedAtlasResolver) SpriteSheetPNG() ([]byte, error) {
	if len(spritesheetPNG) == 0 {
		return nil, fmt.Errorf("embedded sprite sheet is empty")
	}
	return spritesheetPNG, nil
}

// SpriteCoordinates returns the static sprite coordinate table.
func (EmbeddedAtlasResolver) SpriteCoordinates() [spriteCount]spriteRect {
	return spriteCoords
}
