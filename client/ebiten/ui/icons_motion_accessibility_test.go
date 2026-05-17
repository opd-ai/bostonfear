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
	if r.Count() < 20 {
		t.Fatalf("expected at least 20 icon semantics, got %d", r.Count())
	}

	ids := []IconID{IconMove, IconGather, IconInvestigate, IconWard, IconFocus, IconResearch, IconTrade, IconComponent,
		IconAttack, IconEvade, IconCloseGate, IconEncounter, IconHealth, IconSanity, IconClues, IconDoom,
		IconDifficulty, IconConnection, IconTurn, IconArrowLeft, IconArrowRight, IconCameraTop, IconCamera3D, IconPlayer}
	for _, id := range ids {
		spec := r.Spec(id)
		if spec.Glyph == "" {
			t.Fatalf("missing glyph for %q", id)
		}
		if spec.SizePx != 32 && spec.SizePx != 64 {
			t.Fatalf("icon %q size must be 32 or 64, got %d", id, spec.SizePx)
		}
		if spec.StrokePx < 1.0 || spec.StrokePx > 3.0 {
			t.Fatalf("icon %q stroke must be in [1.0, 3.0], got %.2f", id, spec.StrokePx)
		}
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

func TestIconLegibilityAgainstCommonBackgrounds(t *testing.T) {
	tokens := NewDefaultArkhamTheme()
	r := NewIconRegistry()

	bgKeys := []string{"color-bg-dark", "color-surface", "color-surface-elevated"}
	fgKeys := []string{"color-primary-light", "color-secondary-light", "color-info"}
	for _, bgKey := range bgKeys {
		bg := tokens.GetColor(bgKey)
		for _, fgKey := range fgKeys {
			fg := tokens.GetColor(fgKey)
			if ratio := ContrastRatio(bg, fg); ratio < 3.0 {
				t.Fatalf("insufficient icon contrast: bg=%s fg=%s ratio=%.2f", bgKey, fgKey, ratio)
			}
		}
	}

	if r.Spec(IconMove).SizePx != 32 {
		t.Fatalf("expected minimum legibility size to default to 32px, got %d", r.Spec(IconMove).SizePx)
	}
}
