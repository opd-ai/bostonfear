package app

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/client/ebiten/ui"
	"github.com/opd-ai/bostonfear/protocol"
)

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
		// Count blocked retries when the player attempts input out of turn/disconnected.
		if hasActionInputAttempted() {
			h.state.RecordInvalidActionRetry("out-of-turn-or-disconnected")
		}
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
					// No co-located player; treat as invalid retry.
					h.state.RecordInvalidActionRetry("trade-no-colocated-player")
					continue
				}
			}

			h.state.RecordValidActionSent()
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

// handleTouchInput processes touch events and maps them to game actions using hit box registry.
// Location taps trigger Move actions; HUD tap regions trigger other actions.
func (h *InputHandler) handleTouchInput(gs ebclient.GameState, playerID string) {
	// Create viewport for input coordinate transforms.
	vp := &ui.Viewport{
		LogicalWidth:   screenWidth,
		LogicalHeight:  screenHeight,
		PhysicalWidth:  screenWidth,
		PhysicalHeight: screenHeight,
		Scale:          1.0,
		SafeArea:       ui.SafeArea{},
	}

	// Get newly pressed touch IDs from Ebitengine.
	touchIDs := inpututil.JustPressedTouchIDs()
	if len(touchIDs) == 0 {
		return
	}
	handleTouchIDs(h, gs, playerID, touchIDs, vp)
}

func handleTouchIDs(h *InputHandler, gs ebclient.GameState, playerID string, touchIDs []ebiten.TouchID, vp *ui.Viewport) {
	mapper := buildTouchInputMapper(vp)
	for _, touchID := range touchIDs {
		if !dispatchTouchID(h, gs, playerID, mapper, touchID) {
			continue
		}
	}
}

func dispatchTouchID(h *InputHandler, gs ebclient.GameState, playerID string, mapper *ui.InputMapper, touchID ebiten.TouchID) bool {
	x, y := ebiten.TouchPosition(touchID)
	if x < 0 || y < 0 || x >= screenWidth || y >= screenHeight {
		return false
	}

	hitBox := mapper.HitTest(float64(x), float64(y))
	if hitBox == nil {
		return false
	}

	return h.dispatchTouchHitBox(gs, playerID, hitBox.ID)
}

func (h *InputHandler) dispatchTouchHitBox(gs ebclient.GameState, playerID, id string) bool {
	if id == "trade" {
		target := h.findColocatedPlayer(gs, playerID)
		if target == "" {
			h.state.RecordInvalidActionRetry("trade-no-colocated-player")
			return true
		}
		h.state.RecordValidActionSent()
		h.net.SendAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionTrade,
			Target:   target,
		})
		return true
	}

	switch protocol.Location(id) {
	case protocol.Downtown, protocol.University, protocol.Rivertown, protocol.Northside:
		h.state.RecordValidActionSent()
		h.net.SendAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionMove,
			Target:   id,
		})
		return true
	}

	actionMap := map[string]protocol.ActionType{
		"gather":      protocol.ActionGather,
		"investigate": protocol.ActionInvestigate,
		"ward":        protocol.ActionCastWard,
		"focus":       protocol.ActionFocus,
		"research":    protocol.ActionResearch,
		"closegate":   protocol.ActionCloseGate,
		"component":   protocol.ActionComponent,
		"attack":      protocol.ActionAttack,
		"evade":       protocol.ActionEvade,
	}

	if action, exists := actionMap[id]; exists {
		h.state.RecordValidActionSent()
		h.net.SendAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   action,
			Target:   "",
		})
		return true
	}
	return false
}

func buildTouchInputMapper(vp *ui.Viewport) *ui.InputMapper {
	mapper := ui.NewInputMapper()
	registerTouchLocationHitBoxes(mapper, vp)
	registerTouchActionHitBoxes(mapper)

	return mapper
}

func registerTouchLocationHitBoxes(mapper *ui.InputMapper, vp *ui.Viewport) {
	locationConstraints := map[protocol.Location]*ui.Constraint{
		protocol.Downtown: {
			Anchor:  ui.AnchorTopLeft,
			OffsetX: 40,
			OffsetY: 60,
			Width:   160,
			Height:  100,
		},
		protocol.University: {
			Anchor:  ui.AnchorTopLeft,
			OffsetX: 220,
			OffsetY: 60,
			Width:   160,
			Height:  100,
		},
		protocol.Rivertown: {
			Anchor:  ui.AnchorTopLeft,
			OffsetX: 40,
			OffsetY: 220,
			Width:   160,
			Height:  100,
		},
		protocol.Northside: {
			Anchor:  ui.AnchorTopLeft,
			OffsetX: 220,
			OffsetY: 220,
			Width:   160,
			Height:  100,
		},
	}

	for location, constraint := range locationConstraints {
		mapper.Register(string(location), constraint.Bounds(vp), 44)
	}
}

func registerTouchActionHitBoxes(mapper *ui.InputMapper) {
	actionGridOriginY := screenHeight - 220
	actionGridCellWidth := 170
	actionGridCellHeight := 44
	actionGrid := [][]string{
		{"gather", "investigate"},
		{"ward", "focus"},
		{"research", "trade"},
		{"component", "attack"},
		{"evade", "closegate"},
	}
	for row, rowActions := range actionGrid {
		for col, actionName := range rowActions {
			regionBounds := image.Rect(
				10+col*actionGridCellWidth,
				actionGridOriginY+row*actionGridCellHeight,
				10+(col+1)*actionGridCellWidth,
				actionGridOriginY+(row+1)*actionGridCellHeight,
			)
			mapper.Register(actionName, regionBounds, 44)
		}
	}
}

func hasActionInputAttempted() bool {
	for _, kb := range keyBindings {
		if inpututil.IsKeyJustPressed(kb.key) {
			return true
		}
	}
	return len(inpututil.JustPressedTouchIDs()) > 0
}

type touchLayout struct {
	width           int
	actionBarTop    int
	actionBarBottom int
	actionRegions   int
}

func currentTouchLayout() touchLayout {
	return touchLayout{
		width:           screenWidth,
		actionBarTop:    screenHeight - 50,
		actionBarBottom: screenHeight,
		actionRegions:   6,
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
