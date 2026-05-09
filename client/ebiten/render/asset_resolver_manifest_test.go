package render

import (
	"os"
	"testing"
)

func TestEmbeddedVisualManifest_ContainsAllRequiredComponentKeys(t *testing.T) {
	manifestBytes, err := os.ReadFile("assets/visuals.yaml")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	manifest, err := ParseVisualManifest(manifestBytes)
	if err != nil {
		t.Fatalf("ParseVisualManifest returned error: %v", err)
	}

	for _, key := range requiredComponentKeys {
		if _, ok := manifest.Content.Visuals.Components[key]; !ok {
			t.Fatalf("manifest missing required component key %q", key)
		}
	}
}

func TestEmbeddedAtlasResolver_BuildsAtlasFromManifest(t *testing.T) {
	resolver := NewEmbeddedAtlasResolver()
	if resolver == nil {
		t.Fatal("NewEmbeddedAtlasResolver returned nil")
	}

	sheet, err := resolver.SpriteSheetPNG()
	if err != nil {
		t.Fatalf("SpriteSheetPNG returned error: %v", err)
	}
	if len(sheet) == 0 {
		t.Fatal("SpriteSheetPNG returned empty image bytes")
	}

	coords := resolver.SpriteCoordinates()
	for id := SpriteID(0); id < spriteCount; id++ {
		c := coords[id]
		if c.w <= 0 || c.h <= 0 {
			t.Fatalf("sprite %d has invalid dimensions %+v", id, c)
		}
	}
}
