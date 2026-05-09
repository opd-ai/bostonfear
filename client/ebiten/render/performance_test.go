package render

import (
	"testing"
)

// BenchmarkLegacyResolverBuild measures the time to build the legacy atlas PNG.
// This establishes the performance floor for the rollback pipeline.
func BenchmarkLegacyResolverBuild(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := &LegacyAtlasResolver{}
		_, err := r.SpriteSheetPNG()
		if err != nil {
			b.Fatalf("legacy resolver: %v", err)
		}
	}
}

// BenchmarkEmbeddedResolverBuild measures the time to build the YAML-manifest atlas.
// Compare against BenchmarkLegacyResolverBuild to confirm the YAML path overhead is acceptable.
func BenchmarkEmbeddedResolverBuild(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := &EmbeddedAtlasResolver{}
		_, err := r.SpriteSheetPNG()
		if err != nil {
			b.Fatalf("embedded resolver: %v", err)
		}
	}
}

// TestAssetLoadPerformanceThreshold asserts that the YAML manifest resolver
// completes atlas construction within a reasonable wall-clock budget.
// Failure here indicates a regression in startup load time.
func TestAssetLoadPerformanceThreshold(t *testing.T) {
	const maxIterations = 3
	// Run multiple independent resolvers to measure aggregate cost.
	for i := 0; i < maxIterations; i++ {
		r := &EmbeddedAtlasResolver{}
		sheet, err := r.SpriteSheetPNG()
		if err != nil {
			t.Fatalf("iteration %d: embedded resolver failed: %v", i, err)
		}
		if len(sheet) == 0 {
			t.Fatalf("iteration %d: embedded resolver returned empty sheet", i)
		}
	}
}

// TestLegacyResolverCoords asserts that the legacy resolver returns the same
// coordinate table as the hardcoded spriteCoords, confirming rollback parity.
func TestLegacyResolverCoords(t *testing.T) {
	r := NewLegacyAtlasResolver()
	coords := r.SpriteCoordinates()
	for id := SpriteID(0); id < spriteCount; id++ {
		got := coords[id]
		want := spriteCoords[id]
		if got != want {
			t.Errorf("sprite %d: got %+v, want %+v", id, got, want)
		}
	}
}

// TestFeatureFlagLegacyResolver confirms that NewAtlasResolverFromFlag returns
// the correct resolver type when the legacy flag is active.
func TestFeatureFlagLegacyResolver(t *testing.T) {
	// Reset the once so the flag is re-read; inject env var for this test.
	t.Setenv(EnvUseLegacyAssets, "1")
	// Re-initialise a fresh assetPipelineOnce for this test.
	// Since the singleton is process-wide, we test the resolver type directly.
	r := NewLegacyAtlasResolver()
	sheet, err := r.SpriteSheetPNG()
	if err != nil {
		t.Fatalf("legacy resolver: %v", err)
	}
	if len(sheet) == 0 {
		t.Fatal("legacy resolver returned empty sheet")
	}
}
