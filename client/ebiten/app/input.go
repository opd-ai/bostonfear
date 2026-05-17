package app

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/client/ebiten/ui"
	"github.com/opd-ai/bostonfear/client/ebiten/ui/input"
	"github.com/opd-ai/bostonfear/protocol"
)

const actionDebounceWindow = 120 * time.Millisecond

var touchActionMap = map[string]protocol.ActionType{
	"gather":      protocol.ActionGather,
	"investigate": protocol.ActionInvestigate,
	"ward":        protocol.ActionCastWard,
	"focus":       protocol.ActionFocus,
	"research":    protocol.ActionResearch,
	"closegate":   protocol.ActionCloseGate,
	"component":   protocol.ActionComponent,
	"attack":      protocol.ActionAttack,
	"evade":       protocol.ActionEvade,
	"encounter":   protocol.ActionEncounter,
}

// actionKey maps a keyboard key to the action string sent to the server.
type actionKey struct {
	key    ebiten.Key
	action string
	target string // location name for move actions, empty for others
}

type moveChip struct {
	target protocol.Location
	rect   image.Rectangle
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
	{ebiten.KeyN, "encounter", ""},   // N — Encounter (draw location encounter card)
}

// InputHandler processes keyboard input each frame and sends actions to the server.
// It is safe for use from the Ebitengine Update loop (single goroutine).
type InputHandler struct {
	net              *ebclient.NetClient
	state            *ebclient.LocalState
	focusOrder       []string
	focusIndex       int
	lastActionSentAt time.Time
	lastPressedAt    time.Time
}

// NewInputHandler creates an InputHandler wired to the given client and state.
func NewInputHandler(net *ebclient.NetClient, state *ebclient.LocalState) *InputHandler {
	h := &InputHandler{
		net:   net,
		state: state,
		focusOrder: []string{
			"Downtown", "University", "Rivertown", "Northside",
			"gather", "investigate", "ward", "focus", "research", "trade", "component", "attack", "evade", "closegate", "encounter",
		},
	}
	h.setInitialFocusHint()
	return h
}

func (h *InputHandler) setInitialFocusHint() {
	if h.state == nil || len(h.focusOrder) == 0 {
		return
	}
	h.state.SetFocusedActionHint(h.focusOrder[0])
}

// Update is called once per frame by the game loop.
// It checks each bound key and, if just pressed, queues the corresponding action.
// It also checks for touch input on mobile platforms.
func (h *InputHandler) Update() {
	h.handleFocusNavigation()

	gs, playerID, connected := h.state.Snapshot()
	h.updateHoverHint(gs, playerID)
	h.maybeExpirePressedHint()
	if h.blockedForTurn(gs, playerID, connected) {
		return
	}

	h.handleFocusedActionActivate(gs, playerID)
	h.handleKeyboardInput(gs, playerID)

	// Handle mouse input on desktop/browser.
	h.handleMouseInput(gs, playerID)

	// Handle touch input on mobile platforms.
	h.handleTouchInput(gs, playerID)
}

func (h *InputHandler) blockedForTurn(gs ebclient.GameState, playerID string, connected bool) bool {
	if connected && gs.CurrentPlayer == playerID {
		return false
	}
	if hasActionInputAttempted() {
		h.state.RecordInvalidActionRetry("out-of-turn-or-disconnected")
	}
	return true
}

func (h *InputHandler) handleKeyboardInput(gs ebclient.GameState, playerID string) {
	for _, kb := range keyBindings {
		if !inpututil.IsKeyJustPressed(kb.key) {
			continue
		}
		target := kb.target
		if kb.action == "trade" {
			target = h.findColocatedPlayer(gs, playerID)
			if target == "" {
				h.state.RecordInvalidActionRetry("trade-no-colocated-player")
				continue
			}
		}
		h.markPressedHint(kb.targetOrAction())
		h.sendPlayerAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionType(kb.action),
			Target:   target,
		})
	}
}

func (h *InputHandler) handleFocusNavigation() {
	if len(h.focusOrder) == 0 || h.state == nil {
		return
	}
	if !inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		return
	}
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		h.focusIndex = (h.focusIndex - 1 + len(h.focusOrder)) % len(h.focusOrder)
	} else {
		h.focusIndex = (h.focusIndex + 1) % len(h.focusOrder)
	}
	h.state.SetFocusedActionHint(h.focusOrder[h.focusIndex])
}

func (h *InputHandler) handleFocusedActionActivate(gs ebclient.GameState, playerID string) {
	if !inpututil.IsKeyJustPressed(ebiten.KeyEnter) || len(h.focusOrder) == 0 {
		return
	}
	id := h.focusOrder[h.focusIndex]
	h.markPressedHint(id)
	_ = h.dispatchTouchHitBox(gs, playerID, id)
}

func (h *InputHandler) handleMouseInput(gs ebclient.GameState, playerID string) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	vp := &ui.Viewport{
		LogicalWidth:   screenWidth,
		LogicalHeight:  screenHeight,
		PhysicalWidth:  screenWidth,
		PhysicalHeight: screenHeight,
		Scale:          1.0,
		SafeArea:       ui.SafeArea{},
	}
	mapper := buildTouchInputMapper(vp)
	x, y := ebiten.CursorPosition()
	hitBox := mapper.HitTest(float64(x), float64(y))
	if hitBox == nil {
		if h.dispatchMoveChip(gs, playerID, x, y) {
			return
		}
		return
	}
	_ = h.dispatchTouchHitBox(gs, playerID, hitBox.ID)
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

func dispatchTouchID(h *InputHandler, gs ebclient.GameState, playerID string, mapper *input.InputMapper, touchID ebiten.TouchID) bool {
	x, y := ebiten.TouchPosition(touchID)
	if x < 0 || y < 0 || x >= screenWidth || y >= screenHeight {
		return false
	}

	hitBox := mapper.HitTest(float64(x), float64(y))
	if hitBox == nil {
		return h.dispatchMoveChip(gs, playerID, x, y)
	}

	return h.dispatchTouchHitBox(gs, playerID, hitBox.ID)
}

func (h *InputHandler) dispatchMoveChip(gs ebclient.GameState, playerID string, x, y int) bool {
	for _, chip := range legalMoveChips(gs, playerID, 10, bottomPanelY()-28) {
		if image.Pt(x, y).In(chip.rect) {
			h.markPressedHint(string(chip.target))
			h.sendPlayerAction(ebclient.PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: playerID,
				Action:   protocol.ActionMove,
				Target:   string(chip.target),
			})
			return true
		}
	}
	return false
}

func (h *InputHandler) dispatchTouchHitBox(gs ebclient.GameState, playerID, id string) bool {
	if id == "trade" {
		target := h.findColocatedPlayer(gs, playerID)
		if target == "" {
			h.state.RecordInvalidActionRetry("trade-no-colocated-player")
			return true
		}
		h.markPressedHint(id)
		h.sendPlayerAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionTrade,
			Target:   target,
		})
		return true
	}

	switch protocol.Location(id) {
	case protocol.Downtown, protocol.University, protocol.Rivertown, protocol.Northside:
		h.markPressedHint(id)
		h.sendPlayerAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionMove,
			Target:   id,
		})
		return true
	}

	if action, exists := touchActionMap[id]; exists {
		h.markPressedHint(id)
		h.sendPlayerAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   action,
			Target:   "",
		})
		return true
	}
	return false
}

func (h *InputHandler) sendPlayerAction(msg ebclient.PlayerActionMessage) {
	if h.state == nil || h.net == nil {
		return
	}
	now := time.Now()
	if !h.lastActionSentAt.IsZero() && now.Sub(h.lastActionSentAt) < actionDebounceWindow {
		h.state.RecordInvalidActionRetry("input-debounced")
		return
	}
	h.lastActionSentAt = now
	h.state.RecordValidActionSent()
	h.net.SendAction(msg)
}

func (h *InputHandler) updateHoverHint(gs ebclient.GameState, playerID string) {
	if h.state == nil {
		return
	}
	vp := &ui.Viewport{
		LogicalWidth:   screenWidth,
		LogicalHeight:  screenHeight,
		PhysicalWidth:  screenWidth,
		PhysicalHeight: screenHeight,
		Scale:          1.0,
		SafeArea:       ui.SafeArea{},
	}
	mapper := buildTouchInputMapper(vp)
	x, y := ebiten.CursorPosition()
	hitBox := mapper.HitTest(float64(x), float64(y))
	if hitBox != nil {
		h.state.SetHoveredActionHint(hitBox.ID)
		return
	}
	h.state.SetHoveredActionHint("")
}

func (h *InputHandler) markPressedHint(id string) {
	if h.state == nil {
		return
	}
	h.lastPressedAt = time.Now()
	h.state.SetPressedActionHint(id)
}

func (h *InputHandler) maybeExpirePressedHint() {
	if h.state == nil {
		return
	}
	if h.lastPressedAt.IsZero() {
		return
	}
	if time.Since(h.lastPressedAt) < 150*time.Millisecond {
		return
	}
	h.state.SetPressedActionHint("")
	h.lastPressedAt = time.Time{}
}

func (k actionKey) targetOrAction() string {
	if k.action == "move" {
		return k.target
	}
	return k.action
}

func buildTouchInputMapper(vp *ui.Viewport) *input.InputMapper {
	mapper := input.NewInputMapper()
	registerTouchLocationHitBoxes(mapper, vp)
	registerTouchActionHitBoxes(mapper)

	return mapper
}

func registerTouchLocationHitBoxes(mapper *input.InputMapper, vp *ui.Viewport) {
	_ = vp
	for index, location := range boardLocationOrder {
		mapper.Register(string(location), locationPanelButtonRect(index), 64)
	}
}

func registerTouchActionHitBoxes(mapper *input.InputMapper) {
	for row, rowActions := range actionGridRows() {
		for col, actionName := range rowActions {
			if actionName == "" {
				continue
			}
			mapper.Register(actionName, actionGridRect(row, col), 44)
		}
	}
}

func actionGridRows() [][]string {
	actions := []string{"gather", "investigate", "ward", "focus", "research", "trade", "component", "attack", "evade", "closegate", "encounter"}
	if screenWidth >= screenHeight {
		return [][]string{actions}
	}
	return [][]string{
		actions[0:4],
		actions[4:8],
		actions[8:11],
	}
}

func actionGridRect(row, col int) image.Rectangle {
	const (
		actionGridOriginX    = 10
		actionGridGap        = 6
		actionGridCellHeight = 44
	)
	rows := actionGridRows()
	y := screenHeight - actionGridTotalHeight() + row*(actionGridCellHeight+actionGridGap)
	buttonsInRow := len(rows[row])
	if buttonsInRow == 0 {
		return image.Rect(actionGridOriginX, y, actionGridOriginX, y+actionGridCellHeight)
	}
	availableWidth := screenWidth - 20 - (buttonsInRow-1)*actionGridGap
	cellWidth := availableWidth / buttonsInRow
	if cellWidth < 96 {
		cellWidth = 96
	}
	rowWidth := buttonsInRow*cellWidth + (buttonsInRow-1)*actionGridGap
	x := actionGridOriginX + (screenWidth-20-rowWidth)/2 + col*(cellWidth+actionGridGap)
	if len(rows) == 1 {
		x = actionGridOriginX + col*(cellWidth+actionGridGap)
	}
	return image.Rect(
		x,
		y,
		x+cellWidth,
		y+actionGridCellHeight,
	)
}

func actionGridTotalHeight() int {
	const (
		actionGridGap        = 6
		actionGridCellHeight = 44
		actionGridHeader     = 28
	)
	rows := len(actionGridRows())
	if rows == 0 {
		return actionGridHeader
	}
	return actionGridHeader + rows*actionGridCellHeight + (rows-1)*actionGridGap + 8
}

func hasActionInputAttempted() bool {
	return actionKeyJustPressed() ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyTab) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.JustPressedTouchIDs()) > 0
}

func actionKeyJustPressed() bool {
	for _, kb := range keyBindings {
		if inpututil.IsKeyJustPressed(kb.key) {
			return true
		}
	}
	return false
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
