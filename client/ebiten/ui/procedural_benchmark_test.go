package ui

import "testing"

func BenchmarkProceduralGenerate_Low(b *testing.B) {
	benchProceduralGenerate(b, QualityLow)
}

func BenchmarkProceduralGenerate_Medium(b *testing.B) {
	benchProceduralGenerate(b, QualityMedium)
}

func BenchmarkProceduralGenerate_High(b *testing.B) {
	benchProceduralGenerate(b, QualityHigh)
}

func benchProceduralGenerate(b *testing.B, tier QualityTier) {
	b.Helper()
	gen := NewProceduralGenerator(20260509)
	profile := EffectProfileForTier(tier)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.Generate(profile, 1280, 720, int64(i))
	}
}
