// Package session provides shared session lifecycle primitives for cross-engine use.
// Token identifies a player session and can be validated before re-use.
//
// Example usage for token validation:
//
//	token := session.Token("abc123")
//	if token.Validate() {
//	    // Token is structurally valid (non-empty, no whitespace)
//	}
//
// Example usage for reconnection flow:
//
//	store := session.Default
//	record := session.Record{
//	    PlayerID:       "player1",
//	    Token:          session.Token("abc123"),
//	    Connected:      false,
//	    DisconnectedAt: time.Now().Add(-10 * time.Second),
//	}
//	if store.CanRestore(record, session.Token("abc123"), time.Now(), 30*time.Second) {
//	    // Session can be restored within grace period
//	}
package session

import (
	"strings"
	"time"
)

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

// Record captures the minimal session lifecycle data used by reconnect flows.
type Record struct {
	PlayerID       string
	Token          Token
	Connected      bool
	DisconnectedAt time.Time
}

// Store defines minimal session lifecycle checks shared across engines.
// Implementations can be in-memory or backed by external storage.
type Store interface {
	CanRestore(record Record, candidate Token, now time.Time, gracePeriod time.Duration) bool
	IsExpired(disconnectedAt, now time.Time, gracePeriod time.Duration) bool
}

// DefaultStore provides stateless lifecycle checks over session records.
type DefaultStore struct{}

// CanRestore reports whether a disconnected session can be reclaimed.
func (DefaultStore) CanRestore(record Record, candidate Token, now time.Time, gracePeriod time.Duration) bool {
	if !candidate.Validate() || !record.Token.Validate() {
		return false
	}
	if record.Connected || record.Token != candidate {
		return false
	}
	if record.DisconnectedAt.IsZero() {
		return false
	}
	return !DefaultStore{}.IsExpired(record.DisconnectedAt, now, gracePeriod)
}

// IsExpired reports whether a disconnect has exceeded the reconnection grace period.
func (DefaultStore) IsExpired(disconnectedAt, now time.Time, gracePeriod time.Duration) bool {
	if disconnectedAt.IsZero() {
		return false
	}
	return now.Sub(disconnectedAt) > gracePeriod
}

// Default is the shared session lifecycle checker used by serverengine reconnect logic.
var Default Store = DefaultStore{}
