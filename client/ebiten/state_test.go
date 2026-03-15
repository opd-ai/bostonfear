package ebiten

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTokenPersistence_RoundTrip verifies that a reconnect token saved by one
// LocalState instance is correctly loaded by a freshly constructed instance.
func TestTokenPersistence_RoundTrip(t *testing.T) {
	// Redirect the home directory so we don't touch the real ~/.bostonfear.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if err := os.Setenv("USERPROFILE", tmpHome); err != nil {
		t.Fatal(err)
	}

	const want = "tok-abc-123"

	// Save via SetReconnectToken.
	ls1 := NewLocalState("ws://localhost:8080/ws")
	ls1.SetReconnectToken(want)

	// Verify the file was written.
	expected := filepath.Join(tmpHome, ".bostonfear", "session.json")
	if _, err := os.Stat(expected); os.IsNotExist(err) {
		t.Fatalf("session file not created at %s", expected)
	}

	// Load via a brand-new LocalState.
	ls2 := NewLocalState("ws://localhost:8080/ws")
	if got := ls2.GetReconnectToken(); got != want {
		t.Errorf("loaded token = %q, want %q", got, want)
	}
}

// TestTokenPersistence_MissingFile verifies that a missing session file
// does not cause NewLocalState to fail.
func TestTokenPersistence_MissingFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	ls := NewLocalState("ws://localhost:8080/ws")
	if got := ls.GetReconnectToken(); got != "" {
		t.Errorf("expected empty token without file, got %q", got)
	}
}

// TestTokenPersistence_OverwriteExisting verifies that calling SetReconnectToken
// again replaces the previously persisted token.
func TestTokenPersistence_OverwriteExisting(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	ls := NewLocalState("ws://localhost:8080/ws")
	ls.SetReconnectToken("first-token")
	ls.SetReconnectToken("second-token")

	ls2 := NewLocalState("ws://localhost:8080/ws")
	if got := ls2.GetReconnectToken(); got != "second-token" {
		t.Errorf("loaded token = %q, want %q", got, "second-token")
	}
}
