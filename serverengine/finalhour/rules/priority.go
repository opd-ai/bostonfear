package rules

import (
	"fmt"
	"sort"
)

// PriorityBid represents a player's action submission with priority value.
type PriorityBid struct {
	PlayerID  string
	Action    string
	Priority  int
	FocusCost int
}

// PriorityResolver handles conflict resolution when multiple investigators
// attempt to act on the same target simultaneously.
type PriorityResolver struct {
	bids []PriorityBid
}

// NewPriorityResolver creates a resolver for a set of simultaneous action bids.
func NewPriorityResolver() *PriorityResolver {
	return &PriorityResolver{
		bids: make([]PriorityBid, 0),
	}
}

// AddBid registers a player's action with priority value.
func (r *PriorityResolver) AddBid(playerID, action string, priority, focusCost int) error {
	if priority < 0 {
		return fmt.Errorf("priority must be non-negative: %d", priority)
	}
	if focusCost < 0 {
		return fmt.Errorf("focus cost must be non-negative: %d", focusCost)
	}
	r.bids = append(r.bids, PriorityBid{
		PlayerID:  playerID,
		Action:    action,
		Priority:  priority,
		FocusCost: focusCost,
	})
	return nil
}

// ResolveOrder returns bids sorted by priority (highest first).
// Ties are broken by player ID alphabetically for deterministic resolution.
func (r *PriorityResolver) ResolveOrder() []PriorityBid {
	sorted := make([]PriorityBid, len(r.bids))
	copy(sorted, r.bids)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].PlayerID < sorted[j].PlayerID
	})

	return sorted
}

// Clear removes all bids from the resolver, preparing for the next round.
func (r *PriorityResolver) Clear() {
	r.bids = r.bids[:0]
}
