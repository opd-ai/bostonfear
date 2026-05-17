package session

import (
	"testing"
	"time"
)

func TestTokenValidate(t *testing.T) {
	cases := []struct {
		name string
		tok  Token
		want bool
	}{
		{name: "valid", tok: Token("tok_abc"), want: true},
		{name: "empty", tok: Token(""), want: false},
		{name: "whitespace", tok: Token("  tok  "), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tok.Validate(); got != tc.want {
				t.Fatalf("Validate() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDefaultStoreIsExpiredEdges(t *testing.T) {
	store := DefaultStore{}
	now := time.Now()
	grace := 30 * time.Second

	if got := store.IsExpired(time.Time{}, now, grace); got {
		t.Fatal("IsExpired(zero) = true, want false")
	}
	if got := store.IsExpired(now.Add(-grace), now, grace); got {
		t.Fatal("IsExpired(at grace boundary) = true, want false")
	}
	if got := store.IsExpired(now.Add(-(grace + time.Millisecond)), now, grace); !got {
		t.Fatal("IsExpired(just beyond grace) = false, want true")
	}
}

func TestDefaultStoreCanRestore(t *testing.T) {
	store := DefaultStore{}
	now := time.Now()
	grace := 30 * time.Second

	record := Record{
		PlayerID:       "player_1",
		Token:          Token("tok_abc"),
		Connected:      false,
		DisconnectedAt: now.Add(-5 * time.Second),
	}
	if !store.CanRestore(record, Token("tok_abc"), now, grace) {
		t.Fatal("CanRestore(valid disconnected record) = false, want true")
	}

	record.Connected = true
	if store.CanRestore(record, Token("tok_abc"), now, grace) {
		t.Fatal("CanRestore(connected record) = true, want false")
	}

	record.Connected = false
	record.DisconnectedAt = now.Add(-40 * time.Second)
	if store.CanRestore(record, Token("tok_abc"), now, grace) {
		t.Fatal("CanRestore(expired record) = true, want false")
	}
}
