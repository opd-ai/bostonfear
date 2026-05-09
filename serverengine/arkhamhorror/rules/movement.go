package rules

import "github.com/opd-ai/bostonfear/protocol"

// locationAdjacency defines legal movement edges in the canonical Arkham board.
var locationAdjacency = map[protocol.Location][]protocol.Location{
	protocol.Downtown:   {protocol.University, protocol.Rivertown},
	protocol.University: {protocol.Downtown, protocol.Northside},
	protocol.Rivertown:  {protocol.Downtown, protocol.Northside},
	protocol.Northside:  {protocol.University, protocol.Rivertown},
}

// IsAdjacent reports whether a move from one location to another is legal.
func IsAdjacent(from, to protocol.Location) bool {
	adjacentLocations, exists := locationAdjacency[from]
	if !exists {
		return false
	}

	for _, location := range adjacentLocations {
		if location == to {
			return true
		}
	}
	return false
}
