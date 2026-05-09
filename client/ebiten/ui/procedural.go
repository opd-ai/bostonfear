package ui

import (
	"fmt"
	"hash/fnv"

	"github.com/opd-ai/bostonfear/protocol"
)

// ProceduralLayer identifies an atmosphere pass category.
type ProceduralLayer string

const (
	LayerFog     ProceduralLayer = "fog"
	LayerGrain   ProceduralLayer = "grain"
	LayerSigil   ProceduralLayer = "sigil"
	LayerAmbient ProceduralLayer = "ambient"
)

// ProceduralRect is a deterministic transient overlay primitive.
type ProceduralRect struct {
	Layer ProceduralLayer
	X     float64
	Y     float64
	W     float64
	H     float64
	Alpha uint8
}

// ProceduralFrame is the generated atmosphere payload for a render frame.
type ProceduralFrame struct {
	Rects []ProceduralRect
}

// ProceduralGenerator generates deterministic overlays from a stable seed.
type ProceduralGenerator struct {
	seed uint64
}

// NewProceduralGenerator creates a seeded deterministic generator.
func NewProceduralGenerator(seed uint64) *ProceduralGenerator {
	if seed == 0 {
		seed = 1
	}
	return &ProceduralGenerator{seed: seed}
}

// SeedFromGameState derives a deterministic scenario seed from wire state.
// It uses scenario-identifying values sourced from module-owned content fields
// that are already present in protocol.GameState (difficulty + act/agenda deck).
func SeedFromGameState(gs protocol.GameState) uint64 {
	identity := gs.Difficulty
	if len(gs.ActDeck) > 0 {
		identity += "|act:" + gs.ActDeck[0].Title
	}
	if len(gs.AgendaDeck) > 0 {
		identity += "|agenda:" + gs.AgendaDeck[0].Title
	}
	identity += fmt.Sprintf("|players:%d", len(gs.Players))

	h := fnv.New64a()
	_, _ = h.Write([]byte(identity))
	seed := h.Sum64()
	if seed == 0 {
		seed = 1
	}
	return seed
}

// Generate builds a deterministic set of atmosphere rectangles for a frame.
func (g *ProceduralGenerator) Generate(profile EffectProfile, width, height int, frame int64) ProceduralFrame {
	if g == nil || width <= 0 || height <= 0 || profile.AmbientLayers <= 0 {
		return ProceduralFrame{}
	}
	state := g.seed ^ uint64(frame+1)*0x9e3779b97f4a7c15
	rects := make([]ProceduralRect, 0, profile.AmbientLayers)

	for i := 0; i < profile.AmbientLayers; i++ {
		state = xorshift64(state)
		x := float64(state % uint64(width))
		state = xorshift64(state)
		y := float64(state % uint64(height))
		state = xorshift64(state)
		w := float64(6 + state%36)
		state = xorshift64(state)
		h := float64(6 + state%30)

		layer := LayerAmbient
		switch i % 4 {
		case 0:
			layer = LayerFog
		case 1:
			layer = LayerGrain
		case 2:
			layer = LayerSigil
		}

		rects = append(rects, ProceduralRect{
			Layer: layer,
			X:     x,
			Y:     y,
			W:     w,
			H:     h,
			Alpha: alphaForLayer(layer),
		})
	}

	return ProceduralFrame{Rects: rects}
}

func alphaForLayer(layer ProceduralLayer) uint8 {
	switch layer {
	case LayerFog:
		return 20
	case LayerGrain:
		return 14
	case LayerSigil:
		return 28
	default:
		return 10
	}
}

func xorshift64(v uint64) uint64 {
	v ^= v << 13
	v ^= v >> 7
	v ^= v << 17
	if v == 0 {
		return 1
	}
	return v
}
