// Package ui provides reusable UI infrastructure for multi-resolution rendering and input handling.
//
// Components:
//
// Layout: Define logical and physical coordinate spaces, safe-area insets, and anchor-based positioning.
// - Viewport: logical/physical size mapping with scale factor
// - SafeArea: notch, home bar, rounded corner inset handling
// - Anchor, Constraint: responsive layout anchors and size/position rules
//
// Scaling: Automatic device profile resolution and scale factors.
// - Profile: device form factors (phone portrait/landscape, tablet, desktop, ultrawide)
// - ResolveProfile: pick best profile from physical dimensions
// - TextScaleForProfile, IconScaleForProfile: per-device typography/icon scaling
//
// Input: Hit box registry and coordinate transformation.
// - HitBox: clickable/touchable UI regions with accessibility validation
// - InputMapper: efficient hit testing with z-order support
// - CoordinateTransformer: physical ↔ logical coordinate conversion
//
// Theme: Resolved style token packs for atmospheric rendering.
// - ThemePack: concrete color bundle used by render-time effects
// - ResolveThemePack: maps DesignTokenRegistry values into draw-ready RGBA colors
//
// Effects: Quality-tiered orchestration profile.
// - QualityTier: low, medium, high
// - EffectProfile: controls fog/glow toggles and procedural recompute throttles
//
// Procedural: Deterministic atmosphere primitives.
// - ProceduralGenerator: seeded generator for fog/grain/sigil/ambient overlays
// - SeedFromGameState: derives a stable seed from scenario-identifying state
//
// Example:
//
//	profile := ui.ResolveProfile(width, height)
//	viewport := &ui.Viewport{
//		LogicalWidth:  profile.LogicalWidth,
//		LogicalHeight: profile.LogicalHeight,
//		PhysicalWidth: width,
//		PhysicalHeight: height,
//		Scale: ui.ScaleFactor(float64(width), float64(profile.LogicalWidth)),
//	}
//	mapper := ui.NewInputMapper()
//	mapper.Register("button", rect, 44)
//	hit := mapper.HitTest(viewport.ToLogical(physX, physY))
package ui
