package rules

import (
	"testing"
)

// mockSpender is a simple FocusTokenSpender for tests.
type mockSpender struct {
	focus int
	spent int
}

func (m *mockSpender) GetFocus() int        { return m.focus }
func (m *mockSpender) SpendFocus(n int) int { m.spent += n; m.focus -= n; return n }

func TestRollDice_ZeroDice(t *testing.T) {
	results, successes, tentacles := RollDice(0)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
	if successes != 0 || tentacles != 0 {
		t.Errorf("expected 0 successes and tentacles, got %d/%d", successes, tentacles)
	}
}

func TestRollDice_NegativeDice(t *testing.T) {
	results, successes, tentacles := RollDice(-5)
	if len(results) != 0 || successes != 0 || tentacles != 0 {
		t.Error("negative dice should return empty results")
	}
}

func TestRollDice_ResultLength(t *testing.T) {
	for _, n := range []int{1, 3, 6} {
		results, _, _ := RollDice(n)
		if len(results) != n {
			t.Errorf("expected %d results, got %d", n, len(results))
		}
	}
}

func TestRollDice_ValidFaceValues(t *testing.T) {
	valid := map[DieResult]bool{
		DiceSuccess:  true,
		DiceBlank:    true,
		DiceTentacle: true,
	}
	results, _, _ := RollDice(30)
	for _, r := range results {
		if !valid[r] {
			t.Errorf("unexpected die result: %q", r)
		}
	}
}

func TestRollDice_CountConsistency(t *testing.T) {
	results, successes, tentacles := RollDice(100)
	actualSuccesses := 0
	actualTentacles := 0
	for _, r := range results {
		switch r {
		case DiceSuccess:
			actualSuccesses++
		case DiceTentacle:
			actualTentacles++
		}
	}
	if actualSuccesses != successes {
		t.Errorf("success count mismatch: reported %d, counted %d", successes, actualSuccesses)
	}
	if actualTentacles != tentacles {
		t.Errorf("tentacle count mismatch: reported %d, counted %d", tentacles, actualTentacles)
	}
}

func TestRollDicePoolWithFocus_NoFocus(t *testing.T) {
	spender := &mockSpender{focus: 0}
	results, _, _ := RollDicePoolWithFocus(3, 0, spender)
	if len(results) != 3 {
		t.Errorf("expected 3 results with no focus, got %d", len(results))
	}
	if spender.spent != 0 {
		t.Errorf("expected 0 focus spent, got %d", spender.spent)
	}
}

func TestRollDicePoolWithFocus_SpendsClampsToAvailable(t *testing.T) {
	spender := &mockSpender{focus: 2}
	RollDicePoolWithFocus(3, 5, spender) // ask for 5 but only 2 available
	if spender.spent > 2 {
		t.Errorf("spent more focus than available: spent %d", spender.spent)
	}
}

func TestRollDicePoolWithFocus_NegativeFocusIsIgnored(t *testing.T) {
	spender := &mockSpender{focus: 3}
	results, _, _ := RollDicePoolWithFocus(2, -1, spender)
	if len(results) != 2 {
		t.Errorf("expected 2 results with negative focus, got %d", len(results))
	}
}
