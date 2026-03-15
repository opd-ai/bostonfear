// Package mobile — smoke tests for the mobile binding entry point.
//
// Require DISPLAY=:99 (or equivalent) in headless CI environments because
// the package init() wires Ebitengine, which initialises GLFW.
package mobile

import (
	"testing"
)

// TestSetServerURL verifies that SetServerURL correctly updates the global
// server URL used by the mobile binding.
func TestSetServerURL(t *testing.T) {
	original := serverURL
	defer func() { serverURL = original }()

	const want = "ws://192.168.1.100:8080/ws"
	SetServerURL(want)
	if serverURL != want {
		t.Errorf("serverURL = %q, want %q", serverURL, want)
	}
}

// TestDefaultServerURL verifies that the default server URL targets the
// Android emulator host (10.0.2.2) as documented.
func TestDefaultServerURL(t *testing.T) {
	const want = "ws://10.0.2.2:8080/ws"
	if defaultServerURL != want {
		t.Errorf("defaultServerURL = %q, want %q", defaultServerURL, want)
	}
}

// TestDummy verifies that Dummy is callable and returns without panicking.
// Dummy is required by ebitenmobile's binding generator.
func TestDummy(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Dummy() panicked: %v", r)
		}
	}()
	Dummy()
}
