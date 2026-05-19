package rules

import "testing"

func TestNewPriorityResolver(t *testing.T) {
	r := NewPriorityResolver()
	if r == nil {
		t.Fatal("expected non-nil resolver")
	}
}

func TestAddBid(t *testing.T) {
	r := NewPriorityResolver()

	err := r.AddBid("player1", "reposition", 5, 2)
	if err != nil {
		t.Fatalf("AddBid failed: %v", err)
	}
}

func TestAddBidNegativePriority(t *testing.T) {
	r := NewPriorityResolver()

	err := r.AddBid("player1", "reposition", -1, 2)
	if err == nil {
		t.Error("expected error for negative priority, got nil")
	}
}

func TestAddBidNegativeFocus(t *testing.T) {
	r := NewPriorityResolver()

	err := r.AddBid("player1", "reposition", 5, -1)
	if err == nil {
		t.Error("expected error for negative focus cost, got nil")
	}
}

func TestResolveOrder(t *testing.T) {
	r := NewPriorityResolver()

	r.AddBid("player1", "action1", 3, 1)
	r.AddBid("player2", "action2", 5, 2)
	r.AddBid("player3", "action3", 1, 0)

	ordered := r.ResolveOrder()

	if len(ordered) != 3 {
		t.Fatalf("expected 3 bids, got %d", len(ordered))
	}

	// Should be sorted by priority descending
	if ordered[0].PlayerID != "player2" || ordered[0].Priority != 5 {
		t.Errorf("expected player2 with priority 5 first, got %s with %d", ordered[0].PlayerID, ordered[0].Priority)
	}
	if ordered[1].PlayerID != "player1" || ordered[1].Priority != 3 {
		t.Errorf("expected player1 with priority 3 second, got %s with %d", ordered[1].PlayerID, ordered[1].Priority)
	}
	if ordered[2].PlayerID != "player3" || ordered[2].Priority != 1 {
		t.Errorf("expected player3 with priority 1 third, got %s with %d", ordered[2].PlayerID, ordered[2].Priority)
	}
}

func TestResolveOrderTieBreaker(t *testing.T) {
	r := NewPriorityResolver()

	r.AddBid("bob", "action1", 3, 1)
	r.AddBid("alice", "action2", 3, 1)

	ordered := r.ResolveOrder()

	// Ties broken alphabetically by player ID
	if ordered[0].PlayerID != "alice" {
		t.Errorf("expected alice to win tie-breaker, got %s", ordered[0].PlayerID)
	}
	if ordered[1].PlayerID != "bob" {
		t.Errorf("expected bob to lose tie-breaker, got %s", ordered[1].PlayerID)
	}
}

func TestClear(t *testing.T) {
	r := NewPriorityResolver()

	r.AddBid("player1", "action1", 3, 1)
	r.AddBid("player2", "action2", 5, 2)

	r.Clear()

	ordered := r.ResolveOrder()
	if len(ordered) != 0 {
		t.Errorf("expected 0 bids after clear, got %d", len(ordered))
	}
}
