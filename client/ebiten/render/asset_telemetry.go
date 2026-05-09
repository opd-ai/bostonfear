package render

import (
	"log"
	"sync/atomic"
)

// AssetTelemetry collects runtime counters for asset pipeline events.
// All counter operations are safe for concurrent use.
type AssetTelemetry struct {
	// ManifestParseErrors counts the number of times the YAML manifest could
	// not be decoded or failed schema validation.
	ManifestParseErrors atomic.Int64

	// ComponentLoadFailures counts the number of individual component assets
	// that could not be resolved (file missing, unreadable, or invalid).
	ComponentLoadFailures atomic.Int64

	// FallbacksUsed counts the number of times a placeholder image was
	// substituted for a missing or unresolvable component asset.
	FallbacksUsed atomic.Int64

	// AtlasBuildErrors counts the number of times atlas PNG construction
	// failed after all component images were resolved.
	AtlasBuildErrors atomic.Int64
}

// globalAssetTelemetry is the package-level telemetry sink. Import this
// from other packages using AssetMetrics().
var globalAssetTelemetry AssetTelemetry

// AssetMetrics returns a pointer to the package-level AssetTelemetry
// instance. Values are reset to zero at process start only.
func AssetMetrics() *AssetTelemetry {
	return &globalAssetTelemetry
}

// LogSummary emits a single log line with all current counter values.
func (t *AssetTelemetry) LogSummary() {
	log.Printf(
		"asset telemetry: manifest_parse_errors=%d component_load_failures=%d fallbacks_used=%d atlas_build_errors=%d",
		t.ManifestParseErrors.Load(),
		t.ComponentLoadFailures.Load(),
		t.FallbacksUsed.Load(),
		t.AtlasBuildErrors.Load(),
	)
}

// Snapshot returns a point-in-time copy of all counters as a plain struct.
func (t *AssetTelemetry) Snapshot() AssetTelemetrySnapshot {
	return AssetTelemetrySnapshot{
		ManifestParseErrors:   t.ManifestParseErrors.Load(),
		ComponentLoadFailures: t.ComponentLoadFailures.Load(),
		FallbacksUsed:         t.FallbacksUsed.Load(),
		AtlasBuildErrors:      t.AtlasBuildErrors.Load(),
	}
}

// AssetTelemetrySnapshot is an immutable copy of AssetTelemetry counters
// suitable for comparison, export, or health-check thresholds.
type AssetTelemetrySnapshot struct {
	ManifestParseErrors   int64
	ComponentLoadFailures int64
	FallbacksUsed         int64
	AtlasBuildErrors      int64
}
