package ui

import "strings"

// QualityTier controls visual-effect fidelity and runtime cost.
type QualityTier string

const (
	QualityLow    QualityTier = "low"
	QualityMedium QualityTier = "medium"
	QualityHigh   QualityTier = "high"
)

// EffectProfile defines which atmosphere effects are enabled and how often
// expensive effect state should be recomputed.
type EffectProfile struct {
	EnableFog      bool
	EnableGlow     bool
	EnableAmbient  bool
	FogOpacity     float32
	GlowIntensity  float32
	AmbientLayers  int
	ProceduralStep int // recompute every Nth frame (1 = every frame)
}

// ParseQualityTier normalizes an input string to a supported tier.
func ParseQualityTier(raw string) QualityTier {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(QualityLow):
		return QualityLow
	case string(QualityHigh):
		return QualityHigh
	default:
		return QualityMedium
	}
}

// EffectProfileForTier returns conservative defaults for each tier.
func EffectProfileForTier(tier QualityTier) EffectProfile {
	switch tier {
	case QualityLow:
		return EffectProfile{
			EnableFog:      true,
			EnableGlow:     false,
			EnableAmbient:  true,
			FogOpacity:     0.05,
			GlowIntensity:  0,
			AmbientLayers:  6,
			ProceduralStep: 3,
		}
	case QualityHigh:
		return EffectProfile{
			EnableFog:      true,
			EnableGlow:     true,
			EnableAmbient:  true,
			FogOpacity:     0.12,
			GlowIntensity:  0.16,
			AmbientLayers:  20,
			ProceduralStep: 1,
		}
	default:
		return EffectProfile{
			EnableFog:      true,
			EnableGlow:     true,
			EnableAmbient:  true,
			FogOpacity:     0.08,
			GlowIntensity:  0.10,
			AmbientLayers:  12,
			ProceduralStep: 2,
		}
	}
}
