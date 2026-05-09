package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/protocol"
)

// touchLocationRects maps each location name to its board rectangle (x, y, w, h)
// for touch input region detection on mobile platforms.
var touchLocationRects = map[protocol.Location]struct{ x, y, w, h int }{
	protocol.Downtown:   {40, 60, 160, 100},
	protocol.University: {220, 60, 160, 100},
	protocol.Rivertown:  {40, 220, 160, 100},
	protocol.Northside:  {220, 220, 160, 100},
}

// actionKey maps a keyboard key to the action string sent to the server.
type actionKey struct {
	key    ebiten.Key
	action string
	target string // location name for move actions, empty for others
}

// keyBindings defines all keyboard shortcuts available to the player.
// These are fixed mappings; the UI renders labels next to each location/action.
var keyBindings = []actionKey{
	// Movement — numeric keys 1-4 select a destination.
	{ebiten.Key1, "move", "Downtown"},
	{ebiten.Key2, "move", "University"},
	{ebiten.Key3, "move", "Rivertown"},
	{ebiten.Key4, "move", "Northside"},

	// Actions — letter keys.
	{ebiten.KeyG, "gather", ""},      // G — Gather Resources
	{ebiten.KeyI, "investigate", ""}, // I — Investigate
	{ebiten.KeyW, "ward", ""},        // W — Cast Ward
	{ebiten.KeyF, "focus", ""},       // F — Focus (gain focus point)
	{ebiten.KeyR, "research", ""},    // R — Research (improved investigation)
	{ebiten.KeyT, "trade", ""},       // T — Trade (with co-located player if available)
	{ebiten.KeyC, "component", ""},   // C — Component ability (archetype-specific)
	{ebiten.KeyA, "attack", ""},      // A — Attack (engaged enemy)
	{ebiten.KeyE, "evade", ""},       // E — Evade (from engaged enemy)
	{ebiten.KeyX, "closegate", ""},   // X — Close Gate (seal gate at location)
}

// InputHandler processes keyboard input each frame and sends actions to the server.
// It is safe for use from the Ebitengine Update loop (single goroutine).
type InputHandler struct {
	net   *ebclient.NetClient
	state *ebclient.LocalState
}

// NewInputHandler creates an InputHandler wired to the given client and state.
func NewInputHandler(net *ebclient.NetClient, state *ebclient.LocalState) *InputHandler {
	return &InputHandler{net: net, state: state}
}

// Update is called once per frame by the game loop.
// It checks each bound key and, if just pressed, queues the corresponding action.
// It also checks for touch input on mobile platforms.
func (h *InputHandler) Update() {
	gs, playerID, connected := h.state.Snapshot()
	if !connected || gs.CurrentPlayer != playerID {
		// Only process input when connected and it is this player's turn.
		return
	}

	// Handle keyboard input.
	for _, kb := range keyBindings {
		if inpututil.IsKeyJustPressed(kb.key) {
			// Special handling for Trade: find a co-located player.
			target := kb.target
			if kb.action == "trade" {
				target = h.findColocatedPlayer(gs, playerID)
				if target == "" {
					// No co-located player; skip the action.
					continue
				}
			}

			h.net.SendAction(ebclient.PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: playerID,
				Action:   protocol.ActionType(kb.action),
				Target:   target,
			})
		}
	}

	// Handle touch input on mobile platforms.
	h.handleTouchInput(gs, playerID)
}

// handleTouchInput processes touch events and maps them to game actions.
// Location taps trigger Move actions; HUD tap regions trigger other actions.
func (h *InputHandler) handleTouchInput(gs ebclient.GameState, playerID string) {
	// Get newly pressed touch IDs.
	touchIDs := inpututil.JustPressedTouchIDs()
	if len(touchIDs) == 0 {
		return
	}

	// Process each touch.
	for _, touchID := range touchIDs {
		x, y := ebiten.TouchPosition(touchID)

		// Check if the tap is within a location rectangle.
		for location, rect := range touchLocationRects {
			if x >= rect.x && x < rect.x+rect.w && y >= rect.y && y < rect.y+rect.h {
				// Tap is within this location; send Move action.
				h.net.SendAction(ebclient.PlayerActionMessage{
					Type:     "playerAction",
					PlayerID: playerID,
					Action:   protocol.ActionMove,
					Target:   string(location),
				})
				return
			}
		}

		// Check HUD tap regions (bottom strip, 550-600px).
		if y >= 550 {
			// Divide the HUD into 6 regions for common actions.
			region := int(float64(x) / float64(screenWidth) * 6)
			actions := []struct {
				action protocol.ActionType
				target string
			}{
				{protocol.ActionGather, ""},
				{protocol.ActionInvestigate, ""},
				{protocol.ActionCastWard, ""},
				{protocol.ActionFocus, ""},
				{protocol.ActionResearch, ""},
				{protocol.ActionCloseGate, ""},
			}
			if region >= 0 && region < len(actions) {
				action := actions[region]
				h.net.SendAction(ebclient.PlayerActionMessage{
					Type:     "playerAction",
					PlayerID: playerID,
					Action:   action.action,
					Target:   action.target,
				})
				return
			}
		}
	}
}

// findColocatedPlayer returns the ID of the first other player at the same location,
// or an empty string if none is found.
func (h *InputHandler) findColocatedPlayer(gs ebclient.GameState, playerID string) string {
	thisPlayer, exists := gs.Players[playerID]
	if !exists {
		return ""
	}

	for otherID, otherPlayer := range gs.Players {
		if otherID != playerID && otherPlayer.Location == thisPlayer.Location && !otherPlayer.Defeated {
			return otherID
		}
	}
	return ""
}
