package ui

import "testing"

func TestResolveButtonStyle_SizeVariants(t *testing.T) {
	tokens := NewDefaultArkhamTheme()

	small := ResolveButtonStyle(ButtonPrimary, ButtonSizeSmall, ButtonStateDefault, tokens)
	if small.Width != 32 || small.Height != 32 {
		t.Fatalf("small button size = %.0fx%.0f, want 32x32", small.Width, small.Height)
	}

	medium := ResolveButtonStyle(ButtonPrimary, ButtonSizeMedium, ButtonStateDefault, tokens)
	if medium.Width != 48 || medium.Height != 48 {
		t.Fatalf("medium button size = %.0fx%.0f, want 48x48", medium.Width, medium.Height)
	}

	large := ResolveButtonStyle(ButtonPrimary, ButtonSizeLarge, ButtonStateDefault, tokens)
	if large.Width != 64 || large.Height != 64 {
		t.Fatalf("large button size = %.0fx%.0f, want 64x64", large.Width, large.Height)
	}
}

func TestResolveButtonStyle_StateAndVariantBehavior(t *testing.T) {
	tokens := NewDefaultArkhamTheme()

	danger := ResolveButtonStyle(ButtonDanger, ButtonSizeMedium, ButtonStateDefault, tokens)
	if danger.Fill != ColorDanger {
		t.Fatalf("danger button fill mismatch: got %+v, want %+v", danger.Fill, ColorDanger)
	}

	disabled := ResolveButtonStyle(ButtonPrimary, ButtonSizeMedium, ButtonStateDisabled, tokens)
	if disabled.IconAllowed {
		t.Fatal("disabled button should not allow icon emphasis")
	}

	loading := ResolveButtonStyle(ButtonPrimary, ButtonSizeMedium, ButtonStateLoading, tokens)
	if !loading.ShowSpinner {
		t.Fatal("loading button should enable spinner")
	}

	if loading.CornerRadius != 8 {
		t.Fatalf("corner radius mismatch: got %.1f, want 8.0", loading.CornerRadius)
	}
	if loading.Padding != 10 {
		t.Fatalf("padding mismatch: got %.1f, want 10.0", loading.Padding)
	}
}

func TestIconRegistry_ActionIconsPresent(t *testing.T) {
	r := NewIconRegistry()
	required := []IconID{
		IconMove,
		IconGather,
		IconInvestigate,
		IconWard,
		IconFocus,
		IconResearch,
		IconTrade,
		IconComponent,
		IconAttack,
		IconEvade,
		IconCloseGate,
		IconEncounter,
		IconHealth,
		IconSanity,
		IconClues,
		IconDoom,
		IconTurn,
		IconDifficulty,
		IconConnection,
	}
	for _, id := range required {
		if glyph := r.Get(id); glyph == "" {
			t.Fatalf("missing default icon glyph for %q", id)
		}
	}
}
