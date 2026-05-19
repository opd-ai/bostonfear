package rules

import "testing"

func TestDefaultCountdown(t *testing.T) {
	cd := DefaultCountdown()
	if cd.Current != 12 {
		t.Errorf("expected countdown current to be 12, got %d", cd.Current)
	}
	if cd.Max != 12 {
		t.Errorf("expected countdown max to be 12, got %d", cd.Max)
	}
}

func TestCountdownDecrement(t *testing.T) {
	cd := DefaultCountdown()

	err := cd.Decrement(3)
	if err != nil {
		t.Fatalf("decrement failed: %v", err)
	}
	if cd.Current != 9 {
		t.Errorf("expected countdown to be 9 after decrement, got %d", cd.Current)
	}
}

func TestCountdownDecrementNegative(t *testing.T) {
	cd := DefaultCountdown()

	err := cd.Decrement(-1)
	if err == nil {
		t.Error("expected error for negative decrement, got nil")
	}
}

func TestCountdownIsExpired(t *testing.T) {
	cd := CountdownToken{Current: 0, Max: 12}
	if !cd.IsExpired() {
		t.Error("expected countdown to be expired at 0")
	}

	cd.Current = 1
	if cd.IsExpired() {
		t.Error("expected countdown not to be expired at 1")
	}
}

func TestCountdownDecrementToZero(t *testing.T) {
	cd := CountdownToken{Current: 3, Max: 12}

	err := cd.Decrement(5)
	if err != nil {
		t.Fatalf("decrement failed: %v", err)
	}
	if cd.Current != 0 {
		t.Errorf("expected countdown to clamp at 0, got %d", cd.Current)
	}
	if !cd.IsExpired() {
		t.Error("expected countdown to be expired after decrementing below zero")
	}
}
