package rules

import (
	"testing"

	"github.com/opd-ai/bostonfear/protocol"
)

func TestIsAdjacent_ValidPairs(t *testing.T) {
	valid := [][2]protocol.Location{
		{protocol.Downtown, protocol.University},
		{protocol.Downtown, protocol.Rivertown},
		{protocol.University, protocol.Northside},
		{protocol.Rivertown, protocol.Northside},
	}
	for _, pair := range valid {
		if !IsAdjacent(pair[0], pair[1]) {
			t.Errorf("expected %q -> %q to be adjacent", pair[0], pair[1])
		}
	}
}

func TestIsAdjacent_InvalidPairs(t *testing.T) {
	invalid := [][2]protocol.Location{
		{protocol.Downtown, protocol.Northside},
		{protocol.University, protocol.Rivertown},
		{protocol.Location("Unknown"), protocol.Downtown},
	}
	for _, pair := range invalid {
		if IsAdjacent(pair[0], pair[1]) {
			t.Errorf("expected %q -> %q to be non-adjacent", pair[0], pair[1])
		}
	}
}
