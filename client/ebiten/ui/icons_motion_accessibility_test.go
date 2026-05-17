package ui

import "testing"

func TestIconRegistry_Defaults(t *testing.T) {
	r := NewIconRegistry()
	if r.Get(IconMove) == "" {
		t.Fatal("expected IconMove to be registered")
	}
	if r.Get(IconDoom) == "" {
		t.Fatal("expected IconDoom to be registered")
	}
}

func TestMotionCatalog_Defaults(t *testing.T) {
	c := NewMotionCatalog()
	if c.Get("fade").Duration == 0 {
		t.Fatal("expected fade preset duration to be non-zero")
	}
	if c.Get("pulse").Easing == "" {
		t.Fatal("expected pulse preset easing to be set")
	}
}

func TestThemeContrastBaseline(t *testing.T) {
	tokens := NewDefaultArkhamTheme()
	bg := tokens.GetColor("color-bg-dark")
	text := tokens.GetColor("color-primary-light")
	contrast := ContrastRatio(bg, text)
	if contrast < 3.0 {
		t.Fatalf("contrast baseline too low: got %.2f, want >= 3.0", contrast)
	}
}

func TestTokenRegistryRequiredEntries(t *testing.T) {
	tokens := NewDefaultArkhamTheme()
	requiredColors := []string{
		"color-bg-dark", "color-surface", "color-surface-elevated", "color-health", "color-sanity", "color-clues", "color-doom",
	}
	fallback := Color{R: 255, G: 255, B: 255, A: 255} // Default white fallback from GetColor
	for _, key := range requiredColors {
		col := tokens.GetColor(key)
		if col.R == fallback.R && col.G == fallback.G && col.B == fallback.B && col.A == fallback.A {
			t.Fatalf("missing required color token (got fallback): %s", key)
		}
	}
	if tokens.GetTypography("body") == nil {
		t.Fatal("missing required typography token: body")
	}
	if got := tokens.GetSpacing("md"); got <= 0 {
		t.Fatalf("missing required spacing token: md (got %f)", got)
	}
	if tokens.GetElevation("surface-raised") == nil {
		t.Fatal("missing required elevation token: surface-raised")
	}
	if tokens.GetIconStyle("icon-action") == nil {
		t.Fatal("missing required icon style token: icon-action")
	}
	base := tokens.GetSemanticColor("surface-base")
	elevated := tokens.GetSemanticColor("surface-elevated")
	if base == elevated {
		t.Fatal("surface semantic colors should differ for hierarchy")
	}
}
