// Package app — unit tests for the Ebitengine game-loop wiring.
//
// Tests that call Ebitengine rendering APIs require a display context.
// Build with -tags=requires_display and DISPLAY=:99 (or any accessible X11
// display) in headless CI environments; without the tag these files are
// excluded so that `go test ./...` succeeds in pure-headless environments
// without triggering the GLFW init() panic.

//go:build requires_display

package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
)

// newTestGame builds a bare Game struct wired to an in-memory LocalState.
// It does NOT call NewGame (which dials the WebSocket server).
func newTestGame(t *testing.T) *Game {
	t.Helper()
	state := ebclient.NewLocalState("ws://localhost:8080/ws")
	net := ebclient.NewNetClient(state)
	input := NewInputHandler(net, state)
	return &Game{
		state: state,
		net:   net,
		input: input,
	}
}

// TestLayout verifies that Layout always returns the fixed 800×600 logical resolution.
func TestLayout(t *testing.T) {
	g := &Game{}
	w, h := g.Layout(0, 0)
	if w != screenWidth || h != screenHeight {
		t.Errorf("Layout() = (%d, %d), want (%d, %d)", w, h, screenWidth, screenHeight)
	}
}

// TestUpdate_NoopWhenNotConnected verifies that Update() returns nil and
// does not panic when the client is not yet connected to the server.
func TestUpdate_NoopWhenNotConnected(t *testing.T) {
	g := newTestGame(t)
	// LocalState starts in disconnected state; Update must be safe.
	if err := g.Update(); err != nil {
		t.Errorf("Update() returned non-nil error while disconnected: %v", err)
	}
}

// TestDrawPlayerPanel_NoPlayers verifies that drawPlayerPanel renders the
// "Waiting for players…" fallback without panicking when no players are present.
func TestDrawPlayerPanel_NoPlayers(t *testing.T) {
	g := newTestGame(t)
	screen := ebiten.NewImage(screenWidth, screenHeight)
	gs := ebclient.GameState{
		Players:   make(map[string]*ebclient.Player),
		TurnOrder: []string{},
		GamePhase: "waiting",
	}
	// Must not panic.
	g.drawPlayerPanel(screen, gs, "")
}

// TestDrawPlayerPanel_SinglePlayer verifies that drawPlayerPanel renders
// a player row without panicking when one player is present.
func TestDrawPlayerPanel_SinglePlayer(t *testing.T) {
	g := newTestGame(t)
	screen := ebiten.NewImage(screenWidth, screenHeight)
	gs := ebclient.GameState{
		Players: map[string]*ebclient.Player{
			"p1": {
				ID:               "p1",
				Location:         "Downtown",
				Resources:        ebclient.Resources{Health: 8, Sanity: 6, Clues: 2},
				ActionsRemaining: 1,
				Connected:        true,
			},
		},
		CurrentPlayer: "p1",
		TurnOrder:     []string{"p1"},
		GamePhase:     "playing",
	}
	// Must not panic.
	g.drawPlayerPanel(screen, gs, "p1")
}

// TestPlayerColourIndex_InOrder verifies that playerColourIndex returns the
// position of the player within the turn order, modulo palette size.
func TestPlayerColourIndex_InOrder(t *testing.T) {
	order := []string{"p1", "p2", "p3"}
	if got := playerColourIndex("p1", order); got != 0 {
		t.Errorf("playerColourIndex(p1) = %d, want 0", got)
	}
	if got := playerColourIndex("p2", order); got != 1 {
		t.Errorf("playerColourIndex(p2) = %d, want 1", got)
	}
	if got := playerColourIndex("p3", order); got != 2 {
		t.Errorf("playerColourIndex(p3) = %d, want 2", got)
	}
}

// TestPlayerColourIndex_Missing verifies that playerColourIndex returns 0
// when the player ID is not found in the turn order.
func TestPlayerColourIndex_Missing(t *testing.T) {
	if got := playerColourIndex("ghost", []string{"p1", "p2"}); got != 0 {
		t.Errorf("playerColourIndex(ghost) = %d, want 0 (default)", got)
	}
}

// TestMin8 verifies the min8 helper for uint8 clamping.
func TestMin8(t *testing.T) {
	if got := min8(10, 20); got != 10 {
		t.Errorf("min8(10,20) = %d, want 10", got)
	}
	if got := min8(200, 50); got != 50 {
		t.Errorf("min8(200,50) = %d, want 50", got)
	}
	if got := min8(255, 255); got != 255 {
		t.Errorf("min8(255,255) = %d, want 255", got)
	}
}
