package render

import (
	_ "embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed shaders/fog.kage
var fogShaderSrc []byte

//go:embed shaders/glow.kage
var glowShaderSrc []byte

//go:embed shaders/doom.kage
var doomShaderSrc []byte

// ShaderSet holds the compiled Kage shaders used for post-processing effects.
// Compile once via NewShaderSet and reuse each frame.
type ShaderSet struct {
	Fog  *ebiten.Shader // fog-of-war vignette
	Glow *ebiten.Shader // player-token glow pulse
	Doom *ebiten.Shader // doom-level desaturation vignette
}

// NewShaderSet compiles all three Kage shaders from embedded source.
// Returns an error if any shader fails to compile.
func NewShaderSet() (*ShaderSet, error) {
	fog, err := ebiten.NewShader(fogShaderSrc)
	if err != nil {
		return nil, fmt.Errorf("compile fog shader: %w", err)
	}
	glow, err := ebiten.NewShader(glowShaderSrc)
	if err != nil {
		fog.Deallocate()
		return nil, fmt.Errorf("compile glow shader: %w", err)
	}
	doom, err := ebiten.NewShader(doomShaderSrc)
	if err != nil {
		fog.Deallocate()
		glow.Deallocate()
		return nil, fmt.Errorf("compile doom shader: %w", err)
	}
	return &ShaderSet{Fog: fog, Glow: glow, Doom: doom}, nil
}

// Deallocate releases GPU resources for all shaders in the set.
func (s *ShaderSet) Deallocate() {
	if s.Fog != nil {
		s.Fog.Deallocate()
	}
	if s.Glow != nil {
		s.Glow.Deallocate()
	}
	if s.Doom != nil {
		s.Doom.Deallocate()
	}
}

// DrawDoomVignette composites the doom vignette over dst at the given doom fraction.
// doomFraction should be doom/12 in the range [0.0, 1.0].
func DrawDoomVignette(dst *ebiten.Image, shaders *ShaderSet, doomFraction float32) {
	if shaders == nil || shaders.Doom == nil || doomFraction <= 0 {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"DoomFraction": doomFraction,
	}
	op.Images[0] = dst
	dst.DrawRectShader(w, h, shaders.Doom, op)
}

// DrawFogOverlay composites a subtle full-screen fog effect over dst.
// opacity should be in the range [0.0, 1.0].
func DrawFogOverlay(dst *ebiten.Image, shaders *ShaderSet, opacity float32) {
	if shaders == nil || shaders.Fog == nil || opacity <= 0 {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"Opacity": opacity,
	}
	op.Images[0] = dst
	dst.DrawRectShader(w, h, shaders.Fog, op)
}

// DrawGlowOverlay composites a soft pulse used for atmosphere and interaction highlights.
// intensity should be in [0.0, 1.0], timeSeconds is elapsed real time in seconds.
func DrawGlowOverlay(dst *ebiten.Image, shaders *ShaderSet, intensity, timeSeconds float32) {
	if shaders == nil || shaders.Glow == nil || intensity <= 0 {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"Intensity": intensity,
		"Time":      timeSeconds,
	}
	op.Images[0] = dst
	dst.DrawRectShader(w, h, shaders.Glow, op)
}
