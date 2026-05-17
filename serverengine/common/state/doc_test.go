package state

import "testing"

func TestResourceBoundsClamp(t *testing.T) {
	b := ResourceBounds{Min: 1, Max: 3}

	if got := b.Clamp(0); got != 1 {
		t.Fatalf("Clamp(0) = %d, want 1", got)
	}
	if got := b.Clamp(2); got != 2 {
		t.Fatalf("Clamp(2) = %d, want 2", got)
	}
	if got := b.Clamp(9); got != 3 {
		t.Fatalf("Clamp(9) = %d, want 3", got)
	}
}

func TestClampCoreResources(t *testing.T) {
	h, s, c := ClampCoreResources(-2, 11, 7)
	if h != 0 || s != 10 || c != 5 {
		t.Fatalf("ClampCoreResources() = (%d,%d,%d), want (0,10,5)", h, s, c)
	}
}

func TestBoundInvariants(t *testing.T) {
	if !HealthBounds.InBounds(0) || !HealthBounds.InBounds(10) {
		t.Fatal("HealthBounds should include [0,10]")
	}
	if !SanityBounds.InBounds(0) || !SanityBounds.InBounds(10) {
		t.Fatal("SanityBounds should include [0,10]")
	}
	if !ClueBounds.InBounds(0) || !ClueBounds.InBounds(5) {
		t.Fatal("ClueBounds should include [0,5]")
	}
}
