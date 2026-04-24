// Package app — unit tests for the InputHandler key-binding table.
// These tests exercise pure data logic (constant inspection, table structure)
// and do NOT call any Ebitengine input-poll functions. However, because the
// app package imports Ebitengine (which initialises GLFW on package init), a
// virtual framebuffer is required to load the test binary.
//
// Run with: DISPLAY=:99 xvfb-run -a go test -race -tags=requires_display ./client/ebiten/app/...

//go:build requires_display

package app

import "testing"

// TestKeyBindings_NotEmpty verifies that the keyBindings table is populated.
// An empty table would silently disable all keyboard input.
func TestKeyBindings_NotEmpty(t *testing.T) {
	if len(keyBindings) == 0 {
		t.Error("keyBindings is empty: no keyboard input would be processed")
	}
}

// TestKeyBindings_Actions verifies that every expected action type appears at
// least once in the keyBindings table (move, gather, investigate, ward).
func TestKeyBindings_Actions(t *testing.T) {
	required := []string{"move", "gather", "investigate", "ward"}
	found := make(map[string]bool, len(required))
	for _, kb := range keyBindings {
		found[kb.action] = true
	}
	for _, action := range required {
		if !found[action] {
			t.Errorf("action %q not found in keyBindings", action)
		}
	}
}

// TestKeyBindings_MoveTargets verifies that all four canonical Arkham Horror
// neighbourhoods appear as move targets in the keyBindings table.
func TestKeyBindings_MoveTargets(t *testing.T) {
	expected := []string{"Downtown", "University", "Rivertown", "Northside"}
	targets := make(map[string]bool, len(expected))
	for _, kb := range keyBindings {
		if kb.action == "move" {
			targets[kb.target] = true
		}
	}
	for _, loc := range expected {
		if !targets[loc] {
			t.Errorf("location %q missing from move keyBindings", loc)
		}
	}
}

// TestKeyBindings_NonMoveActionsHaveEmptyTarget verifies that non-move
// actions (gather, investigate, ward) do not accidentally set a target string,
// which would be interpreted as an invalid location by the server.
func TestKeyBindings_NonMoveActionsHaveEmptyTarget(t *testing.T) {
	for _, kb := range keyBindings {
		if kb.action != "move" && kb.target != "" {
			t.Errorf("non-move action %q has non-empty target %q; expected empty",
				kb.action, kb.target)
		}
	}
}

// TestKeyBindings_UniqueKeys verifies that each keyboard key appears at most
// once in the keyBindings table. Duplicate key entries would cause the first
// matching binding to shadow all subsequent ones with the same key.
func TestKeyBindings_UniqueKeys(t *testing.T) {
	seen := make(map[int]string, len(keyBindings))
	for _, kb := range keyBindings {
		keyInt := int(kb.key)
		if prev, dup := seen[keyInt]; dup {
			t.Errorf("duplicate key %d in keyBindings: actions %q and %q", keyInt, prev, kb.action)
		}
		seen[keyInt] = kb.action
	}
}
