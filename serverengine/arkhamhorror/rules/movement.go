package rules

import (
"strings"

"github.com/opd-ai/bostonfear/protocol"
"github.com/opd-ai/bostonfear/serverengine/arkhamhorror/content"
)

// IsAdjacent reports whether a move from one location to another is legal.
// S5: Delegates to arkhamhorror/content module for location topology definitions.
func IsAdjacent(from, to protocol.Location) bool {
// Convert protocol.Location to lowercase string for arkhamhorror module
// (which uses lowercase location names for consistency with map adjacency data).
fromStr := strings.ToLower(string(from))
toStr := strings.ToLower(string(to))

// Check adjacency using arkhamhorror/content location rules
return content.IsAdjacentLocation(fromStr, toStr)
}
