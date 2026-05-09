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
		"color-bg-dark", "color-surface", "color-health", "color-sanity", "color-clues", "color-doom",
	}
	for _, key := range requiredColors {
		if tokens.GetColor(key) == nil {
			t.Fatalf("missing required color token: %s", key)
		}
	}
	if tokens.GetTypography("body") == nil {
		t.Fatal("missing required typography token: body")
	}
}
