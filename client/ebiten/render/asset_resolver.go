package render

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

const embeddedVisualManifestPath = "assets/visuals.yaml"

//go:embed assets/*
var embeddedAssetsFS embed.FS

var requiredComponentKeys = map[SpriteID]string{
	SpriteBackground:         "board.background",
	SpriteLocationDowntown:   "location.downtown",
	SpriteLocationUniversity: "location.university",
	SpriteLocationRivertown:  "location.rivertown",
	SpriteLocationNorthside:  "location.northside",
	SpritePlayerToken:        "token.investigator.default",
	SpriteDoomMarker:         "hud.doom.marker",
	SpriteActionOverlay:      "ui.action.button.default",
}

// AtlasAssetResolver abstracts sprite sheet and coordinate lookup for rendering.
// Implementations can source assets from embed, filesystem, or manifest-backed providers.
type AtlasAssetResolver interface {
	SpriteSheetPNG() ([]byte, error)
	SpriteCoordinates() [spriteCount]spriteRect
}

// EmbeddedAtlasResolver is the default resolver backed by embedded YAML-configured assets.
type EmbeddedAtlasResolver struct {
	once       sync.Once
	sheetPNG   []byte
	coords     [spriteCount]spriteRect
	resolveErr error
}

// NewEmbeddedAtlasResolver returns the default atlas resolver implementation.
func NewEmbeddedAtlasResolver() AtlasAssetResolver {
	return &EmbeddedAtlasResolver{}
}

// SpriteSheetPNG returns the manifest-resolved PNG sprite sheet bytes.
func (r *EmbeddedAtlasResolver) SpriteSheetPNG() ([]byte, error) {
	return r.sheetPNGOrErr()
}

// SpriteCoordinates returns coordinates matching SpriteSheetPNG output.
func (r *EmbeddedAtlasResolver) SpriteCoordinates() [spriteCount]spriteRect {
	return r.coordsOrFallback()
}

func (r *EmbeddedAtlasResolver) sheetPNGOrErr() ([]byte, error) {
	if err := r.ensureResolved(); err != nil {
		return nil, err
	}
	if len(r.sheetPNG) == 0 {
		return nil, fmt.Errorf("embedded resolver produced empty sprite sheet")
	}
	return r.sheetPNG, nil
}

func (r *EmbeddedAtlasResolver) coordsOrFallback() [spriteCount]spriteRect {
	if err := r.ensureResolved(); err != nil {
		return spriteCoords
	}
	return r.coords
}

func (r *EmbeddedAtlasResolver) ensureResolved() error {
	r.once.Do(r.resolveFromManifest)
	if r.resolveErr != nil {
		return r.resolveErr
	}
	return nil
}

func (r *EmbeddedAtlasResolver) resolveFromManifest() {
	manifestBytes, err := embeddedAssetsFS.ReadFile(embeddedVisualManifestPath)
	if err != nil {
		r.resolveErr = fmt.Errorf("read embedded visual manifest: %w", err)
		return
	}

	manifest, err := ParseVisualManifest(manifestBytes)
	if err != nil {
		r.resolveErr = fmt.Errorf("parse embedded visual manifest: %w", err)
		return
	}

	basePath := strings.TrimSpace(manifest.Content.Visuals.BasePath)
	placeholderPath := filepath.ToSlash(filepath.Join(basePath, manifest.Content.Visuals.Placeholders.Missing))
	placeholderImage := r.loadImageOrNil(placeholderPath)

	images := make(map[SpriteID]image.Image, spriteCount)
	preflightIssues := make([]string, 0)

	for id := SpriteID(0); id < spriteCount; id++ {
		componentKey, ok := requiredComponentKeys[id]
		if !ok {
			preflightIssues = append(preflightIssues, fmt.Sprintf("missing required key mapping for sprite id %d", id))
			images[id] = solidPlaceholderImage(64, 64)
			continue
		}

		asset, exists := manifest.Content.Visuals.Components[componentKey]
		if !exists {
			preflightIssues = append(preflightIssues, fmt.Sprintf("manifest missing required component key %q", componentKey))
			images[id] = fallbackImage(placeholderImage)
			continue
		}

		componentPath := filepath.ToSlash(filepath.Join(basePath, asset.File))
		img := r.loadImageOrNil(componentPath)
		if img == nil {
			preflightIssues = append(preflightIssues, fmt.Sprintf("component %q missing/unreadable at %q", componentKey, componentPath))
			images[id] = fallbackImage(placeholderImage)
			continue
		}

		if asset.W > 0 && asset.H > 0 {
			cropped, cropErr := cropImage(img, asset.X, asset.Y, asset.W, asset.H)
			if cropErr != nil {
				preflightIssues = append(preflightIssues, fmt.Sprintf("component %q invalid crop rect: %v", componentKey, cropErr))
				images[id] = fallbackImage(placeholderImage)
				continue
			}
			img = cropped
		}

		images[id] = img
	}

	r.sheetPNG, r.coords, r.resolveErr = buildAtlasPNG(images)
	if r.resolveErr != nil {
		r.resolveErr = fmt.Errorf("build atlas from manifest assets: %w", r.resolveErr)
		return
	}

	if len(preflightIssues) > 0 {
		log.Printf("render preflight: %d asset issue(s) detected", len(preflightIssues))
		for _, issue := range preflightIssues {
			log.Printf("render preflight: %s", issue)
		}
	}
}

func (r *EmbeddedAtlasResolver) loadImageOrNil(path string) image.Image {
	data, err := embeddedAssetsFS.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

func buildAtlasPNG(images map[SpriteID]image.Image) ([]byte, [spriteCount]spriteRect, error) {
	coords := [spriteCount]spriteRect{}
	totalWidth := 0
	maxHeight := 0

	for id := SpriteID(0); id < spriteCount; id++ {
		img, ok := images[id]
		if !ok || img == nil {
			return nil, coords, fmt.Errorf("missing image for sprite id %d", id)
		}
		b := img.Bounds()
		w := b.Dx()
		h := b.Dy()
		if w <= 0 || h <= 0 {
			return nil, coords, fmt.Errorf("invalid dimensions for sprite id %d", id)
		}
		coords[id] = spriteRect{x: totalWidth, y: 0, w: w, h: h}
		totalWidth += w
		if h > maxHeight {
			maxHeight = h
		}
	}

	atlas := image.NewRGBA(image.Rect(0, 0, totalWidth, maxHeight))
	for id := SpriteID(0); id < spriteCount; id++ {
		img := images[id]
		dstRect := image.Rect(coords[id].x, coords[id].y, coords[id].x+coords[id].w, coords[id].y+coords[id].h)
		draw.Draw(atlas, dstRect, img, img.Bounds().Min, draw.Over)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, atlas); err != nil {
		return nil, coords, fmt.Errorf("encode atlas png: %w", err)
	}
	return out.Bytes(), coords, nil
}

func cropImage(img image.Image, x, y, w, h int) (image.Image, error) {
	b := img.Bounds()
	if x < 0 || y < 0 || w <= 0 || h <= 0 {
		return nil, fmt.Errorf("x/y/w/h must be non-negative with positive width and height")
	}
	if x+w > b.Dx() || y+h > b.Dy() {
		return nil, fmt.Errorf("crop rectangle out of bounds")
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), img, image.Point{X: b.Min.X + x, Y: b.Min.Y + y}, draw.Src)
	return out, nil
}

func fallbackImage(placeholder image.Image) image.Image {
	if placeholder != nil {
		return placeholder
	}
	return solidPlaceholderImage(64, 64)
}

func solidPlaceholderImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 200, G: 40, B: 40, A: 255}}, image.Point{}, draw.Src)
	return img
}
