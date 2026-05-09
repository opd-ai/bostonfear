package ui

import (
	"testing"

	"github.com/opd-ai/bostonfear/protocol"
)

func TestSeedFromGameState_Deterministic(t *testing.T) {
	gs := protocol.GameState{
		Difficulty: "normal",
		Players: map[string]*protocol.Player{
			"p1": {ID: "p1"},
			"p2": {ID: "p2"},
		},
		ActDeck:    []protocol.ActCard{{Title: "Signal in the Harbor"}},
		AgendaDeck: []protocol.AgendaCard{{Title: "Dark Tide"}},
	}

	seedA := SeedFromGameState(gs)
	seedB := SeedFromGameState(gs)
	if seedA != seedB {
		t.Fatalf("SeedFromGameState() not deterministic: %d != %d", seedA, seedB)
	}
}

func TestProceduralGenerate_Deterministic(t *testing.T) {
	genA := NewProceduralGenerator(42)
	genB := NewProceduralGenerator(42)
	profile := EffectProfileForTier(QualityMedium)

	a := genA.Generate(profile, 800, 600, 123)
	b := genB.Generate(profile, 800, 600, 123)
	if len(a.Rects) != len(b.Rects) {
		t.Fatalf("rect count mismatch: %d != %d", len(a.Rects), len(b.Rects))
	}
	for i := range a.Rects {
		if a.Rects[i] != b.Rects[i] {
			t.Fatalf("rect %d mismatch: %+v != %+v", i, a.Rects[i], b.Rects[i])
		}
	}
}

func TestEffectProfileForTier_ProgressiveLayers(t *testing.T) {
	low := EffectProfileForTier(QualityLow)
	med := EffectProfileForTier(QualityMedium)
	high := EffectProfileForTier(QualityHigh)

	if !(low.AmbientLayers < med.AmbientLayers && med.AmbientLayers < high.AmbientLayers) {
		t.Fatalf("expected ambient layers low < medium < high, got %d, %d, %d", low.AmbientLayers, med.AmbientLayers, high.AmbientLayers)
	}
	if !(low.ProceduralStep > med.ProceduralStep && med.ProceduralStep > high.ProceduralStep) {
		t.Fatalf("expected procedural step low > medium > high, got %d, %d, %d", low.ProceduralStep, med.ProceduralStep, high.ProceduralStep)
	}
}
