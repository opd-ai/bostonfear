package render

import (
	"log"
	"os"
	"sync"
)

// AssetPipelineMode selects which asset loading strategy the renderer uses.
type AssetPipelineMode int

const (
	// AssetPipelineYAML is the default: assets are resolved from the embedded
	// YAML visual manifest (content/visuals section). This is the production path.
	AssetPipelineYAML AssetPipelineMode = iota

	// AssetPipelineLegacy bypasses the YAML manifest and uses the hardcoded
	// spriteCoords table with a procedurally generated placeholder atlas.
	// Enable this via BOSTONFEAR_USE_LEGACY_ASSETS=1 for rollback purposes.
	AssetPipelineLegacy
)

const (
	// EnvUseLegacyAssets is the environment variable name that activates the
	// legacy (hardcoded) asset pipeline when set to a non-empty value.
	EnvUseLegacyAssets = "BOSTONFEAR_USE_LEGACY_ASSETS"
)

var (
	assetPipelineOnce sync.Once
	assetPipelineMode AssetPipelineMode
)

// ActiveAssetPipeline returns the asset pipeline mode resolved from the
// environment at first call and then cached for the lifetime of the process.
// Set BOSTONFEAR_USE_LEGACY_ASSETS=1 to enable the legacy rollback path.
func ActiveAssetPipeline() AssetPipelineMode {
	assetPipelineOnce.Do(func() {
		if os.Getenv(EnvUseLegacyAssets) != "" {
			assetPipelineMode = AssetPipelineLegacy
			log.Printf("asset pipeline: LEGACY mode active (BOSTONFEAR_USE_LEGACY_ASSETS is set); YAML manifest skipped")
		} else {
			assetPipelineMode = AssetPipelineYAML
			log.Printf("asset pipeline: YAML manifest mode active")
		}
	})
	return assetPipelineMode
}

// NewAtlasResolverFromFlag returns the appropriate AtlasAssetResolver based on
// the active asset pipeline feature flag. Call this instead of constructing a
// resolver directly so that the flag is honoured.
func NewAtlasResolverFromFlag() AtlasAssetResolver {
	if ActiveAssetPipeline() == AssetPipelineLegacy {
		return NewLegacyAtlasResolver()
	}
	return NewEmbeddedAtlasResolver()
}
