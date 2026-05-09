// Package content defines Arkham Horror map topology and game constants.
// S5 Migration: Scenario setup, decks, constants, and adjacency rules.
package content

// LocationAdjacency defines which locations are adjacent (can move in 1 action).
// This is arkhamhorror module-owned and used for movement legality validation.
var LocationAdjacency = map[string][]string{
	"downtown":   {"university", "rivertown"},
	"university": {"downtown", "northside"},
	"rivertown":  {"downtown", "northside"},
	"northside":  {"university", "rivertown"},
}

// IsAdjacentLocation checks if two locations are adjacent
func IsAdjacentLocation(from, to string) bool {
	if from == to {
		return false
	}
	adjacent, ok := LocationAdjacency[from]
	if !ok {
		return false
	}
	for _, loc := range adjacent {
		if loc == to {
			return true
		}
	}
	return false
}

// LocationNames returns the canonical list of playable locations
func LocationNames() []string {
	return []string{"downtown", "university", "rivertown", "northside"}
}

// DoomLocationConstants define doom token placement rules
const (
	// MaxDoomPerLocation limits doom that can be placed at any one location
	MaxDoomPerLocation = 4

	// DoomThresholdPerLocation triggers anomaly spawn when reached
	DoomThresholdPerLocation = 2
)

// S5: Game constants for encounters and mythos are now arkhamhorror-owned
// to keep all Arkham-specific rules together and support scenario customization
