// Package session provides shared session lifecycle primitives for cross-engine use.
// Token identifies a player session and can be validated before re-use.
package session

import "strings"

// Token represents a player's session credential used for reconnection and slot reclaim.
// Tokens are opaque strings; use Validate to check structural validity before use.
type Token string

// Validate reports whether the token is structurally valid (non-empty, no whitespace).
func (t Token) Validate() bool {
	s := string(t)
	return s != "" && s == strings.TrimSpace(s)
}

// String returns the raw token string.
func (t Token) String() string { return string(t) }
