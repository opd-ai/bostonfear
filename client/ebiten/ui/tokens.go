package ui

import "image/color"

// DesignToken holds a single visual property (color, size, duration, etc.).
type DesignToken interface {
	// Value returns the token's computed value for rendering.
	Value() interface{}
}

// ColorToken represents a semantic color in the design system.
type ColorToken struct {
	Name  string     // e.g., "primary", "danger", "success"
	Light color.RGBA // color.RGBA value
}

// Value returns the RGBA color.
func (ct *ColorToken) Value() interface{} {
	return ct.Light
}

// SpacingToken represents a dimension (padding, margin, gap).
type SpacingToken struct {
	Name string  // e.g., "xs", "sm", "md", "lg"
	Size float64 // In logical pixels
}

// Value returns the spacing value.
func (st *SpacingToken) Value() interface{} {
	return st.Size
}

// TypographyToken represents a text style (font size, weight, line height).
type TypographyToken struct {
	Name       string // e.g., "heading1", "body", "caption"
	FontSize   float64
	LineHeight float64
	Weight     int // 400=normal, 700=bold
}

// Value returns the font size.
func (tt *TypographyToken) Value() interface{} {
	return tt.FontSize
}

// ElevationToken represents depth styling for layered UI surfaces.
type ElevationToken struct {
	Name    string
	Blur    float64
	Spread  float64
	Opacity float64
}

// Value returns a compact scalar value suitable for generic token handling.
func (et *ElevationToken) Value() interface{} {
	return et.Opacity
}

// IconStyleToken captures icon styling rules for consistent semantics.
type IconStyleToken struct {
	Name      string
	StrokePx  float64
	Roundness float64
	Emphasis  float64
}

// Value returns a compact scalar value suitable for generic token handling.
func (it *IconStyleToken) Value() interface{} {
	return it.StrokePx
}

// DesignTokenRegistry centralizes all design tokens for a theme.
type DesignTokenRegistry struct {
	colors     map[string]*ColorToken
	spacing    map[string]*SpacingToken
	typography map[string]*TypographyToken
	elevation  map[string]*ElevationToken
	iconStyles map[string]*IconStyleToken
	semantics  map[string]string // Maps semantic name to token name.
}

// NewDesignTokenRegistry creates an empty token registry.
func NewDesignTokenRegistry() *DesignTokenRegistry {
	return &DesignTokenRegistry{
		colors:     make(map[string]*ColorToken),
		spacing:    make(map[string]*SpacingToken),
		typography: make(map[string]*TypographyToken),
		elevation:  make(map[string]*ElevationToken),
		iconStyles: make(map[string]*IconStyleToken),
		semantics:  make(map[string]string),
	}
}

// RegisterColor adds a color token.
func (dtr *DesignTokenRegistry) RegisterColor(name string, col color.RGBA) {
	if dtr != nil {
		dtr.colors[name] = &ColorToken{Name: name, Light: col}
	}
}

// GetColor retrieves a color token by name.
func (dtr *DesignTokenRegistry) GetColor(name string) color.RGBA {
	if dtr != nil {
		if token, exists := dtr.colors[name]; exists {
			return token.Light
		}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255} // Default white fallback.
}

// RegisterSpacing adds a spacing token.
func (dtr *DesignTokenRegistry) RegisterSpacing(name string, size float64) {
	if dtr != nil {
		dtr.spacing[name] = &SpacingToken{Name: name, Size: size}
	}
}

// GetSpacing retrieves a spacing token by name.
func (dtr *DesignTokenRegistry) GetSpacing(name string) float64 {
	if dtr != nil {
		if token, exists := dtr.spacing[name]; exists {
			return token.Size
		}
	}
	return 0
}

// RegisterTypography adds a typography token.
func (dtr *DesignTokenRegistry) RegisterTypography(name string, fontSize, lineHeight float64, weight int) {
	if dtr != nil {
		dtr.typography[name] = &TypographyToken{
			Name:       name,
			FontSize:   fontSize,
			LineHeight: lineHeight,
			Weight:     weight,
		}
	}
}

// GetTypography retrieves a typography token by name.
func (dtr *DesignTokenRegistry) GetTypography(name string) *TypographyToken {
	if dtr != nil {
		return dtr.typography[name]
	}
	return nil
}

// RegisterElevation adds an elevation token.
func (dtr *DesignTokenRegistry) RegisterElevation(name string, blur, spread, opacity float64) {
	if dtr == nil {
		return
	}
	dtr.elevation[name] = &ElevationToken{
		Name:    name,
		Blur:    blur,
		Spread:  spread,
		Opacity: opacity,
	}
}

// GetElevation retrieves an elevation token by name.
func (dtr *DesignTokenRegistry) GetElevation(name string) *ElevationToken {
	if dtr == nil {
		return nil
	}
	return dtr.elevation[name]
}

// RegisterIconStyle adds an icon style token.
func (dtr *DesignTokenRegistry) RegisterIconStyle(name string, strokePx, roundness, emphasis float64) {
	if dtr == nil {
		return
	}
	dtr.iconStyles[name] = &IconStyleToken{
		Name:      name,
		StrokePx:  strokePx,
		Roundness: roundness,
		Emphasis:  emphasis,
	}
}

// GetIconStyle retrieves an icon style token by name.
func (dtr *DesignTokenRegistry) GetIconStyle(name string) *IconStyleToken {
	if dtr == nil {
		return nil
	}
	return dtr.iconStyles[name]
}

// MapSemantic aliases a semantic name to a token name.
// Example: MapSemantic("danger-background", "color-red-700")
func (dtr *DesignTokenRegistry) MapSemantic(semanticName, tokenName string) {
	if dtr != nil {
		dtr.semantics[semanticName] = tokenName
	}
}

// GetSemanticColor resolves a semantic name to a color.
func (dtr *DesignTokenRegistry) GetSemanticColor(semanticName string) color.RGBA {
	if dtr != nil {
		if tokenName, exists := dtr.semantics[semanticName]; exists {
			return dtr.GetColor(tokenName)
		}
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255} // Fallback.
}

// NewDefaultArkhamTheme creates a dark, eldritch-themed token registry.
func NewDefaultArkhamTheme() *DesignTokenRegistry {
	dtr := NewDesignTokenRegistry()

	// Core colors
	dtr.RegisterColor("color-bg-dark", color.RGBA{R: 15, G: 12, B: 18, A: 255})       // Deep dark background
	dtr.RegisterColor("color-bg-darker", color.RGBA{R: 8, G: 6, B: 10, A: 255})       // Even darker
	dtr.RegisterColor("color-surface", color.RGBA{R: 25, G: 20, B: 30, A: 255})       // Panel background
	dtr.RegisterColor("color-surface-light", color.RGBA{R: 40, G: 35, B: 45, A: 255}) // Lighter surface
	dtr.RegisterColor("color-surface-elevated", color.RGBA{R: 50, G: 44, B: 59, A: 255})
	dtr.RegisterColor("color-border-strong", color.RGBA{R: 126, G: 110, B: 152, A: 255})

	// Primary (eldritch)
	dtr.RegisterColor("color-primary", color.RGBA{R: 155, G: 89, B: 182, A: 255}) // Purple
	dtr.RegisterColor("color-primary-light", color.RGBA{R: 200, G: 150, B: 210, A: 255})

	// Status colors
	dtr.RegisterColor("color-success", color.RGBA{R: 46, G: 213, B: 115, A: 255}) // Green
	dtr.RegisterColor("color-danger", color.RGBA{R: 231, G: 76, B: 60, A: 255})   // Red
	dtr.RegisterColor("color-warning", color.RGBA{R: 241, G: 196, B: 15, A: 255}) // Gold
	dtr.RegisterColor("color-info", color.RGBA{R: 52, G: 152, B: 219, A: 255})    // Blue

	// Resource colors
	dtr.RegisterColor("color-health", color.RGBA{R: 192, G: 57, B: 43, A: 255})  // Red
	dtr.RegisterColor("color-sanity", color.RGBA{R: 52, G: 152, B: 219, A: 255}) // Blue
	dtr.RegisterColor("color-clues", color.RGBA{R: 46, G: 213, B: 115, A: 255})  // Green
	dtr.RegisterColor("color-doom", color.RGBA{R: 230, G: 126, B: 34, A: 255})   // Orange

	// Semantics
	dtr.MapSemantic("bg-primary", "color-bg-dark")
	dtr.MapSemantic("bg-secondary", "color-surface")
	dtr.MapSemantic("surface-base", "color-surface")
	dtr.MapSemantic("surface-elevated", "color-surface-elevated")
	dtr.MapSemantic("text-primary", "color-primary")
	dtr.MapSemantic("border-primary", "color-primary-light")
	dtr.MapSemantic("border-strong", "color-border-strong")
	dtr.MapSemantic("status-success", "color-success")
	dtr.MapSemantic("status-danger", "color-danger")

	// Spacing scale (0.5x, 1x, 2x, 4x, 8x)
	scales := []string{"xs", "sm", "md", "lg", "xl"}
	bases := []float64{4, 8, 16, 32, 64}
	for i, name := range scales {
		dtr.RegisterSpacing(name, bases[i])
	}

	// Typography scale  (heading1, heading2, body, caption)
	dtr.RegisterTypography("heading1", 32, 40, 700)
	dtr.RegisterTypography("heading2", 24, 32, 700)
	dtr.RegisterTypography("body", 14, 20, 400)
	dtr.RegisterTypography("caption", 12, 16, 400)

	// Elevation scale for layered surfaces.
	dtr.RegisterElevation("surface-flat", 0, 0, 0.00)
	dtr.RegisterElevation("surface-raised", 2, 1, 0.18)
	dtr.RegisterElevation("surface-overlay", 6, 2, 0.30)

	// Icon styling roles for consistent silhouette and emphasis.
	dtr.RegisterIconStyle("icon-action", 2.0, 0.25, 0.90)
	dtr.RegisterIconStyle("icon-resource", 1.5, 0.20, 0.80)
	dtr.RegisterIconStyle("icon-alert", 2.5, 0.15, 1.00)

	return dtr
}
