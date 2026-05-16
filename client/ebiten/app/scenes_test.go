// Package app — scene-level tests for pointer-accessible connect controls.
//
// Run with: DISPLAY=:99 xvfb-run -a go test -race -tags=requires_display ./client/ebiten/app/...

//go:build requires_display

package app

import (
	"testing"

	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
)

func TestConnectLayoutHitTestRoutesControls(t *testing.T) {
	layout := newConnectLayout()
	if got := layout.hitTest(layout.addressField.Min.X+4, layout.addressField.Min.Y+4); got != connectControlAddressField {
		t.Fatalf("address field hitTest = %q, want %q", got, connectControlAddressField)
	}
	if got := layout.hitTest(layout.addressClear.Min.X+1, layout.addressClear.Min.Y+1); got != connectControlAddressClear {
		t.Fatalf("address clear hitTest = %q, want %q", got, connectControlAddressClear)
	}
	if got := layout.hitTest(layout.nameField.Min.X+4, layout.nameField.Min.Y+4); got != connectControlNameField {
		t.Fatalf("name field hitTest = %q, want %q", got, connectControlNameField)
	}
	if got := layout.hitTest(layout.nameClear.Min.X+1, layout.nameClear.Min.Y+1); got != connectControlNameClear {
		t.Fatalf("name clear hitTest = %q, want %q", got, connectControlNameClear)
	}
	if got := layout.hitTest(layout.connectButton.Min.X+2, layout.connectButton.Min.Y+2); got != connectControlConnectButton {
		t.Fatalf("connect button hitTest = %q, want %q", got, connectControlConnectButton)
	}
}

func TestSceneConnectActivateClearControls(t *testing.T) {
	g := newTestGame(t)
	g.state.SetConnectAddress("example.org:8080")
	g.state.SetDisplayName("Dana")
	scene := &SceneConnect{game: g}

	scene.activateConnectControl(connectControlAddressClear)
	address, displayName := g.state.ConnectFormSnapshot()
	if address != "" {
		t.Fatalf("address after clear = %q, want empty", address)
	}
	if displayName != "Dana" {
		t.Fatalf("display name after address clear = %q, want %q", displayName, "Dana")
	}
	if scene.activeField != connectFieldAddress {
		t.Fatalf("activeField after address clear = %d, want %d", scene.activeField, connectFieldAddress)
	}

	scene.activateConnectControl(connectControlNameClear)
	_, displayName = g.state.ConnectFormSnapshot()
	if displayName != "" {
		t.Fatalf("display name after clear = %q, want empty", displayName)
	}
	if scene.activeField != connectFieldName {
		t.Fatalf("activeField after name clear = %d, want %d", scene.activeField, connectFieldName)
	}

	// The connect button should reuse the existing reconnect path without panicking.
	gs := ebclient.GameState{Players: map[string]*ebclient.Player{}}
	g.state.UpdateGame(gs)
	scene.activateConnectControl(connectControlConnectButton)
}
