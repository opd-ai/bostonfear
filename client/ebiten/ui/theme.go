package ui

import "image/color"

// ThemePack contains pre-resolved colors for atmospheric rendering.
type ThemePack struct {
	Background color.RGBA
	FogTint    color.RGBA
	GrainTint  color.RGBA
	SigilTint  color.RGBA
	Ambient    color.RGBA
}

// ResolveThemePack builds a rendering-friendly theme pack from design tokens.
func ResolveThemePack(registry *DesignTokenRegistry) ThemePack {
	if registry == nil {
		registry = NewDefaultArkhamTheme()
	}
	return ThemePack{
		Background: asRGBA(registry.GetColor("color-bg-dark"), color.RGBA{R: 20, G: 20, B: 30, A: 255}),
		FogTint:    asRGBA(registry.GetColor("color-bg-darker"), color.RGBA{R: 8, G: 6, B: 10, A: 255}),
		GrainTint:  asRGBA(registry.GetColor("color-surface-light"), color.RGBA{R: 40, G: 35, B: 45, A: 28}),
		SigilTint:  asRGBA(registry.GetColor("color-primary"), color.RGBA{R: 155, G: 89, B: 182, A: 34}),
		Ambient:    asRGBA(registry.GetColor("color-warning"), color.RGBA{R: 241, G: 196, B: 15, A: 18}),
	}
}

func asRGBA(in color.Color, fallback color.RGBA) color.RGBA {
	if in == nil {
		return fallback
	}
	r, g, b, a := in.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}
