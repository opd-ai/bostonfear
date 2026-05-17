// Package app — scene state machine for the Arkham Horror Ebitengine client.
//
// CLIENT_SPEC.md §1 requires four scenes:
//
//	SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver
//
// All four scenes are implemented here.
package app

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/client/ebiten/ui"
	"github.com/opd-ai/bostonfear/protocol"
)

// Scene is the interface implemented by each full-screen game scene.
// Update is called every tick; Draw is called every frame.
type Scene interface {
	Update() error
	Draw(screen *ebiten.Image)
}

// SceneConnect is shown while the WebSocket connection is being established.
// It captures server address and display name while showing connection status.
type SceneConnect struct {
	game          *Game
	tick          int
	activeField   int
	reconnectTick int
}

type connectControlID string

const (
	connectControlNone          connectControlID = ""
	connectControlAddressField  connectControlID = "address-field"
	connectControlAddressClear  connectControlID = "address-clear"
	connectControlNameField     connectControlID = "name-field"
	connectControlNameClear     connectControlID = "name-clear"
	connectControlConnectButton connectControlID = "connect-button"
)

type connectLayout struct {
	addressField  image.Rectangle
	addressClear  image.Rectangle
	nameField     image.Rectangle
	nameClear     image.Rectangle
	connectButton image.Rectangle
}

func newConnectLayout() connectLayout {
	return connectLayout{
		addressField:  image.Rect(180, 182, 620, 230),
		addressClear:  image.Rect(542, 194, 610, 218),
		nameField:     image.Rect(180, 244, 620, 292),
		nameClear:     image.Rect(542, 256, 610, 280),
		connectButton: image.Rect(322, 316, 478, 354),
	}
}

func (l connectLayout) hitTest(x, y int) connectControlID {
	pt := image.Pt(x, y)
	switch {
	case pt.In(l.addressClear):
		return connectControlAddressClear
	case pt.In(l.nameClear):
		return connectControlNameClear
	case pt.In(l.addressField):
		return connectControlAddressField
	case pt.In(l.nameField):
		return connectControlNameField
	case pt.In(l.connectButton):
		return connectControlConnectButton
	default:
		return connectControlNone
	}
}

const (
	connectFieldAddress = iota
	connectFieldName
)

// Update handles connect-form input and reconnect countdown state.
func (s *SceneConnect) Update() error {
	s.tick++
	s.handleFieldSwitch()
	s.handleFieldEdits()
	s.handlePointerInput()
	s.updateReconnectTick()
	s.confirmConnectForm()
	return nil
}

// Draw renders the connecting screen using anchor-based layout.
func (s *SceneConnect) Draw(screen *ebiten.Image) {
	gs, playerID, connected := s.game.state.Snapshot()
	address, displayName := s.game.state.ConnectFormSnapshot()
	token := s.game.state.GetReconnectToken()
	layout := newConnectLayout()

	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})

	// Create viewport for anchor-based positioning.
	vp := &ui.Viewport{
		LogicalWidth:   screenWidth,
		LogicalHeight:  screenHeight,
		PhysicalWidth:  screenWidth, // 1:1 scale for now
		PhysicalHeight: screenHeight,
		Scale:          1.0,
		SafeArea:       ui.SafeArea{},
	}

	dots := "...."[:s.tick/15%5]

	// Title: centered near top.
	titleConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 120,
		Width:   240,
		Height:  16,
	}
	titleBounds := titleConstraint.Bounds(vp)
	drawUIText(screen, "Boston Fear - Arkham Horror", titleBounds.Min.X, titleBounds.Min.Y, color.White)

	// Status: below title.
	status := "Connecting" + dots
	if connected {
		status = "Connected"
	}
	statusConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 160,
		Width:   240,
		Height:  16,
	}
	statusBounds := statusConstraint.Bounds(vp)
	drawUIText(screen, "Status: "+status, statusBounds.Min.X, statusBounds.Min.Y, color.White)

	// Server address field.
	s.drawConnectField(screen, vp, layout.addressField, layout.addressClear, "Server address", address, s.activeField == connectFieldAddress)

	// Display name field.
	s.drawConnectField(screen, vp, layout.nameField, layout.nameClear, "Display name", displayName, s.activeField == connectFieldName)

	buttonColor := color.RGBA{R: 85, G: 130, B: 200, A: 255}
	buttonBorder := color.RGBA{R: 190, G: 220, B: 255, A: 255}
	buttonLabel := "CONNECT"
	if connected {
		buttonLabel = "RECONNECT"
	}
	hovered, pressed := pointerState(layout.connectButton)
	if hovered {
		buttonColor = color.RGBA{R: 98, G: 146, B: 220, A: 255}
		buttonBorder = color.RGBA{R: 218, G: 236, B: 255, A: 255}
	}
	if pressed {
		buttonColor = color.RGBA{R: 114, G: 166, B: 240, A: 255}
		buttonBorder = color.RGBA{R: 238, G: 245, B: 255, A: 255}
	}
	if s.activeField == connectFieldAddress || s.activeField == connectFieldName {
		buttonBorder = color.RGBA{R: 255, G: 220, B: 120, A: 255}
	}
	if layout.connectButton.Empty() {
		return
	}
	ebitenutil.DrawRect(screen, float64(layout.connectButton.Min.X), float64(layout.connectButton.Min.Y), float64(layout.connectButton.Dx()), float64(layout.connectButton.Dy()), buttonColor)
	ebitenutil.DrawRect(screen, float64(layout.connectButton.Min.X), float64(layout.connectButton.Min.Y), float64(layout.connectButton.Dx()), 2, buttonBorder)
	ebitenutil.DrawRect(screen, float64(layout.connectButton.Min.X), float64(layout.connectButton.Max.Y-2), float64(layout.connectButton.Dx()), 2, buttonBorder)
	ebitenutil.DrawRect(screen, float64(layout.connectButton.Min.X), float64(layout.connectButton.Min.Y), 2, float64(layout.connectButton.Dy()), buttonBorder)
	ebitenutil.DrawRect(screen, float64(layout.connectButton.Max.X-2), float64(layout.connectButton.Min.Y), 2, float64(layout.connectButton.Dy()), buttonBorder)
	labelX := layout.connectButton.Min.X + layout.connectButton.Dx()/2 - textWidth(buttonLabel)/2
	labelY := layout.connectButton.Min.Y + 10
	drawUIText(screen, buttonLabel, labelX, labelY, color.White)

	// Player slots indicator.
	connectedPlayers := countConnectedPlayers(gs)
	slotsConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 245,
		Width:   240,
		Height:  16,
	}
	slotsBounds := slotsConstraint.Bounds(vp)
	drawUIText(screen, "Slots: "+strconv.Itoa(connectedPlayers)+"/6", slotsBounds.Min.X, slotsBounds.Min.Y, color.White)

	if connectedPlayers >= 6 {
		fullConstraint := ui.Constraint{
			Anchor:  ui.AnchorTopCenter,
			OffsetY: 260,
			Width:   240,
			Height:  16,
		}
		fullBounds := fullConstraint.Bounds(vp)
		drawUIText(screen, "Game Full (6/6)", fullBounds.Min.X, fullBounds.Min.Y, color.RGBA{R: 255, G: 190, B: 190, A: 255})
	}

	// Reconnection timer (if saved session exists but not connected).
	if token != "" && !connected {
		remaining := 60 - s.reconnectTick/60
		if remaining < 0 {
			remaining = 0
		}
		reconnectConstraint := ui.Constraint{
			Anchor:  ui.AnchorTopCenter,
			OffsetY: 290,
			Width:   400,
			Height:  16,
		}
		reconnectBounds := reconnectConstraint.Bounds(vp)
		drawUIText(screen,
			"Reconnecting with saved session... "+strconv.Itoa(remaining)+"s",
			reconnectBounds.Min.X, reconnectBounds.Min.Y, color.White)
	}

	// Field editing instruction.
	fieldLabel := "address"
	if s.activeField == connectFieldName {
		fieldLabel = "display name"
	}
	editConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 330,
		Width:   400,
		Height:  16,
	}
	editBounds := editConstraint.Bounds(vp)
	drawUIText(screen, "Editing: "+fieldLabel+" (TAB to switch)", editBounds.Min.X, editBounds.Min.Y, color.White)

	// Input instructions.
	instrConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 366,
		Width:   400,
		Height:  16,
	}
	instrBounds := instrConstraint.Bounds(vp)
	drawUIText(screen, "Click fields to edit, CLEAR to blank them, ENTER or CONNECT to join", instrBounds.Min.X, instrBounds.Min.Y, color.RGBA{R: 220, G: 220, B: 220, A: 255})

	// Player ID display (if assigned).
	if playerID != "" {
		playerIDConstraint := ui.Constraint{
			Anchor:  ui.AnchorTopCenter,
			OffsetY: 380,
			Width:   320,
			Height:  16,
		}
		playerIDBounds := playerIDConstraint.Bounds(vp)
		drawUIText(screen, trimToWidth("Player ID: "+playerID, 320), playerIDBounds.Min.X, playerIDBounds.Min.Y, color.White)
	}
}

func (s *SceneConnect) drawConnectField(screen *ebiten.Image, vp *ui.Viewport, fieldRect, clearRect image.Rectangle, label, value string, active bool) {
	border := color.RGBA{R: 110, G: 130, B: 170, A: 255}
	fill := color.RGBA{R: 18, G: 20, B: 32, A: 235}
	labelColor := color.RGBA{R: 210, G: 220, B: 240, A: 255}
	if active {
		border = color.RGBA{R: 255, G: 220, B: 120, A: 255}
		fill = color.RGBA{R: 30, G: 28, B: 18, A: 240}
	}
	ebitenutil.DrawRect(screen, float64(fieldRect.Min.X), float64(fieldRect.Min.Y), float64(fieldRect.Dx()), float64(fieldRect.Dy()), fill)
	ebitenutil.DrawRect(screen, float64(fieldRect.Min.X), float64(fieldRect.Min.Y), float64(fieldRect.Dx()), 2, border)
	ebitenutil.DrawRect(screen, float64(fieldRect.Min.X), float64(fieldRect.Max.Y-2), float64(fieldRect.Dx()), 2, border)
	ebitenutil.DrawRect(screen, float64(fieldRect.Min.X), float64(fieldRect.Min.Y), 2, float64(fieldRect.Dy()), border)
	ebitenutil.DrawRect(screen, float64(fieldRect.Max.X-2), float64(fieldRect.Min.Y), 2, float64(fieldRect.Dy()), border)

	labelY := fieldRect.Min.Y + 8
	drawUIText(screen, label, fieldRect.Min.X+10, labelY, labelColor)

	valueText := trimToWidth(value, fieldRect.Dx()-120)
	valueY := fieldRect.Min.Y + 26
	if valueText == "" {
		valueText = "(click and type)"
		if active {
			valueText = ""
		}
		labelColor = color.RGBA{R: 140, G: 150, B: 170, A: 255}
	}
	drawUIText(screen, valueText, fieldRect.Min.X+10, valueY, labelColor)

	if active {
		caretX := fieldRect.Min.X + 10 + textWidth(valueText) + 2
		caretTop := fieldRect.Min.Y + 28
		caretBottom := fieldRect.Min.Y + 42
		if caretBottom > fieldRect.Max.Y-6 {
			caretBottom = fieldRect.Max.Y - 6
		}
		ebitenutil.DrawRect(screen, float64(caretX), float64(caretTop), 2, float64(caretBottom-caretTop), color.RGBA{R: 255, G: 230, B: 160, A: 255})
	}

	clearLabel := "CLEAR"
	clearFill, clearBorder := clearButtonColors(clearRect)
	if !clearRect.Empty() {
		ebitenutil.DrawRect(screen, float64(clearRect.Min.X), float64(clearRect.Min.Y), float64(clearRect.Dx()), float64(clearRect.Dy()), clearFill)
		ebitenutil.DrawRect(screen, float64(clearRect.Min.X), float64(clearRect.Min.Y), float64(clearRect.Dx()), 2, clearBorder)
		ebitenutil.DrawRect(screen, float64(clearRect.Min.X), float64(clearRect.Max.Y-2), float64(clearRect.Dx()), 2, clearBorder)
		ebitenutil.DrawRect(screen, float64(clearRect.Min.X), float64(clearRect.Min.Y), 2, float64(clearRect.Dy()), clearBorder)
		ebitenutil.DrawRect(screen, float64(clearRect.Max.X-2), float64(clearRect.Min.Y), 2, float64(clearRect.Dy()), clearBorder)
		labelX := clearRect.Min.X + clearRect.Dx()/2 - textWidth(clearLabel)/2
		labelY := clearRect.Min.Y + 4
		drawUIText(screen, clearLabel, labelX, labelY, color.White)
	}
	_ = vp
}

func clearButtonColors(clearRect image.Rectangle) (color.RGBA, color.RGBA) {
	fill := color.RGBA{R: 70, G: 70, B: 90, A: 255}
	border := color.RGBA{R: 150, G: 160, B: 190, A: 255}
	hovered, pressed := pointerState(clearRect)
	if pressed {
		return color.RGBA{R: 104, G: 108, B: 140, A: 255}, color.RGBA{R: 212, G: 220, B: 248, A: 255}
	}
	if hovered {
		return color.RGBA{R: 90, G: 92, B: 120, A: 255}, color.RGBA{R: 184, G: 196, B: 228, A: 255}
	}
	return fill, border
}

func (s *SceneConnect) handleBackspace() {
	address, displayName := s.game.state.ConnectFormSnapshot()
	if s.activeField == connectFieldAddress {
		if len(address) > 0 {
			s.game.state.SetConnectAddress(address[:len(address)-1])
		}
		return
	}

	if len(displayName) > 0 {
		s.game.state.SetDisplayName(displayName[:len(displayName)-1])
	}
}

func (s *SceneConnect) handleFieldSwitch() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		s.activeField = (s.activeField + 1) % 2
	}
}

func (s *SceneConnect) handleFieldEdits() {
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		s.handleBackspace()
	}

	for _, r := range ebiten.AppendInputChars(nil) {
		s.appendRune(r)
	}
}

func (s *SceneConnect) updateReconnectTick() {
	_, _, connected := s.game.state.Snapshot()
	token := s.game.state.GetReconnectToken()
	if !connected && token != "" {
		s.reconnectTick++
		return
	}
	s.reconnectTick = 0
}

func (s *SceneConnect) confirmConnectForm() {
	if !inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return
	}

	address, name := s.game.state.ConnectFormSnapshot()
	if strings.TrimSpace(address) != "" {
		s.game.state.SetConnectAddress(address)
	}
	if strings.TrimSpace(name) == "" {
		s.game.state.SetDisplayName("Investigator")
	}
	// Abort the current backoff sleep and redial immediately with the new address.
	s.game.net.Reconnect()
}

func (s *SceneConnect) handlePointerInput() {
	layout := newConnectLayout()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.activateConnectControl(layout.hitTest(x, y))
	}
	for _, touchID := range inpututil.JustPressedTouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x < 0 || y < 0 {
			continue
		}
		s.activateConnectControl(layout.hitTest(x, y))
	}
}

func (s *SceneConnect) activateConnectControl(control connectControlID) {
	switch control {
	case connectControlAddressField:
		s.activeField = connectFieldAddress
	case connectControlNameField:
		s.activeField = connectFieldName
	case connectControlAddressClear:
		s.game.state.ClearConnectAddress()
		s.activeField = connectFieldAddress
	case connectControlNameClear:
		s.game.state.ClearDisplayName()
		s.activeField = connectFieldName
	case connectControlConnectButton:
		s.confirmConnectForm()
	}
}

func (s *SceneConnect) appendRune(r rune) {
	if r < 32 || r > 126 {
		return
	}

	address, displayName := s.game.state.ConnectFormSnapshot()
	if s.activeField == connectFieldAddress {
		s.game.state.SetConnectAddress(address + string(r))
		return
	}
	s.game.state.SetDisplayName(displayName + string(r))
}

func countConnectedPlayers(gs ebclient.GameState) int {
	count := 0
	for _, p := range gs.Players {
		if p != nil && p.Connected {
			count++
		}
	}
	return count
}

// SceneGame renders the in-progress game board. It delegates all drawing to
// the Game's existing draw helpers so the full HUD remains available.
type SceneGame struct {
	game *Game
}

type onboardingControls struct {
	next image.Rectangle
	skip image.Rectangle
}

type cameraControls struct {
	left   image.Rectangle
	right  image.Rectangle
	toggle image.Rectangle
}

type cameraControlID int

const (
	cameraControlNone cameraControlID = iota
	cameraControlLeft
	cameraControlRight
	cameraControlToggle
)

func newOnboardingControls() onboardingControls {
	return onboardingControls{
		next: image.Rect(404, 154, 516, 182),
		skip: image.Rect(528, 154, 664, 182),
	}
}

func newCameraControls() cameraControls {
	y := bottomPanelY() - 28
	return cameraControls{
		left:   image.Rect(374, y, 454, y+24),
		right:  image.Rect(462, y, 542, y+24),
		toggle: image.Rect(550, y, 692, y+24),
	}
}

func (c cameraControls) hitTest(x, y int) cameraControlID {
	pt := image.Pt(x, y)
	switch {
	case pt.In(c.left):
		return cameraControlLeft
	case pt.In(c.right):
		return cameraControlRight
	case pt.In(c.toggle):
		return cameraControlToggle
	default:
		return cameraControlNone
	}
}

// Update processes player input and handles per-tick game logic.
func (s *SceneGame) Update() error {
	s.game.input.Update()
	s.handleCameraControls()
	s.handleOnboarding()
	return nil
}

func (s *SceneGame) handleOnboarding() {
	if s.game.onboarding == nil {
		return
	}
	if !s.game.onboarding.IsActive() && !s.game.onboarding.IsCompleted() {
		s.game.onboarding.Start()
	}
	s.handleOnboardingPointerInput()
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		s.game.onboarding.Skip()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && s.game.onboarding.IsActive() {
		s.game.onboarding.AdvanceStep()
	}
	s.game.onboarding.Update()
}

func (s *SceneGame) handleOnboardingPointerInput() {
	if s.game.onboarding == nil || !s.game.onboarding.IsActive() {
		return
	}
	controls := newOnboardingControls()
	handleOnboardingActivate := func(x, y int) {
		pt := image.Pt(x, y)
		switch {
		case pt.In(controls.next):
			s.game.onboarding.AdvanceStep()
		case pt.In(controls.skip):
			s.game.onboarding.Skip()
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		handleOnboardingActivate(x, y)
	}
	for _, touchID := range inpututil.JustPressedTouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x < 0 || y < 0 {
			continue
		}
		handleOnboardingActivate(x, y)
	}
}

func (s *SceneGame) handleCameraControls() {
	if s.game.camera == nil {
		return
	}
	if s.handleCameraMouseControls() {
		return
	}
	s.handleCameraKeyboardShortcuts()
	s.handleCameraWheelControls()
	s.handleTouchCameraControls()
}

func (s *SceneGame) handleCameraMouseControls() bool {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return false
	}
	controls := newCameraControls()
	x, y := ebiten.CursorPosition()
	return s.applyCameraControl(controls.hitTest(x, y))
}

func (s *SceneGame) handleCameraKeyboardShortcuts() {
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		s.game.camera.OrbitCCW()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		s.game.camera.OrbitCW()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		s.game.camera.ToggleViewMode()
	}
}

func (s *SceneGame) handleCameraWheelControls() {
	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		s.game.camera.OrbitCW()
	}
	if wheelY < 0 {
		s.game.camera.OrbitCCW()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		s.game.camera.ToggleViewMode()
	}
}

func (s *SceneGame) applyCameraControl(control cameraControlID) bool {
	switch control {
	case cameraControlLeft:
		s.game.camera.OrbitCCW()
		return true
	case cameraControlRight:
		s.game.camera.OrbitCW()
		return true
	case cameraControlToggle:
		s.game.camera.ToggleViewMode()
		return true
	default:
		return false
	}
}

func (s *SceneGame) handleTouchCameraControls() {
	// Touch gesture: only touches outside gameplay hit boxes may orbit or toggle.
	controls := newCameraControls()
	for _, id := range inpututil.JustPressedTouchIDs() {
		s.handleSingleTouchCameraControl(id, controls)
	}
}

func (s *SceneGame) handleSingleTouchCameraControl(id ebiten.TouchID, controls cameraControls) {
	if s.isInteractiveTouch(id) {
		return
	}
	x, y := ebiten.TouchPosition(id)
	if s.applyCameraControl(controls.hitTest(x, y)) {
		return
	}
	if x < screenWidth/3 {
		s.game.camera.OrbitCCW()
		return
	}
	if x > screenWidth*2/3 {
		s.game.camera.OrbitCW()
		return
	}
	s.game.camera.ToggleViewMode()
}

func (s *SceneGame) isInteractiveTouch(id ebiten.TouchID) bool {
	x, y := ebiten.TouchPosition(id)
	if !isWithinScreenBounds(x, y) {
		return false
	}
	return s.touchHitsGameplayControl(x, y) ||
		s.touchHitsCameraControl(x, y) ||
		s.touchHitsOnboardingControl(x, y)
}

func isWithinScreenBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < screenWidth && y < screenHeight
}

func (s *SceneGame) touchHitsGameplayControl(x, y int) bool {
	vp := &ui.Viewport{
		LogicalWidth:   screenWidth,
		LogicalHeight:  screenHeight,
		PhysicalWidth:  screenWidth,
		PhysicalHeight: screenHeight,
		Scale:          1.0,
		SafeArea:       ui.SafeArea{},
	}
	mapper := buildTouchInputMapper(vp)
	return mapper.HitTest(float64(x), float64(y)) != nil
}

func (s *SceneGame) touchHitsCameraControl(x, y int) bool {
	return newCameraControls().hitTest(x, y) != cameraControlNone
}

func (s *SceneGame) touchHitsOnboardingControl(x, y int) bool {
	if s.game.onboarding == nil || !s.game.onboarding.IsActive() {
		return false
	}
	onboardingButtons := newOnboardingControls()
	pt := image.Pt(x, y)
	return pt.In(onboardingButtons.next) || pt.In(onboardingButtons.skip)
}

// Draw composites the full game board via the layered renderer.
func (s *SceneGame) Draw(screen *ebiten.Image) {
	s.game.drawGameContent(screen)
}

// SceneCharacterSelect is shown after connection but before the game begins,
// allowing the player to choose their investigator archetype from 6 options
// and select a difficulty level.
// It transitions to SceneGame once the selection is confirmed.
type SceneCharacterSelect struct {
	game *Game

	// investigators holds the list of available investigator archetypes.
	// They are indexed 0-5 and map to keys 1-6.
	investigators []struct {
		name        string // "researcher", "detective", etc.
		display     string // "Researcher", "Detective", etc.
		description string // "Gathers clues and unravels mysteries" etc.
	}

	// selectedDifficulty tracks the player's difficulty choice: "easy", "standard", or "hard".
	// Empty string means not yet selected.
	selectedDifficulty string

	// selectedInvestigator tracks the locally highlighted investigator card.
	// It is confirmed either by the button below the cards or by the keyboard shortcuts.
	selectedInvestigator string
}

type difficultyButton struct {
	value string
	label string
	rect  image.Rectangle
}

// NewSceneCharacterSelect creates a new character selection scene.
func NewSceneCharacterSelect(g *Game) *SceneCharacterSelect {
	return &SceneCharacterSelect{
		game: g,
		investigators: []struct {
			name        string
			display     string
			description string
		}{
			{"researcher", "Researcher", "Gathers clues steadily; excels at investigation"},
			{"detective", "Detective", "Questions suspects and finds hidden motives"},
			{"occultist", "Occultist", "Masters eldritch lore and ancient secrets"},
			{"soldier", "Soldier", "Stands firm against horrors and physical threats"},
			{"mystic", "Mystic", "Channels arcane power through ritual and will"},
			{"survivor", "Survivor", "Adapts to adversity and learns from hardship"},
		},
	}
}

// Update checks for key presses 1-6 (investigator) and E/S/H (difficulty) and sends selections to the server.
func (s *SceneCharacterSelect) Update() error {
	gs, playerID, _ := s.game.state.Snapshot()

	// Check if this player has already selected an investigator.
	if player, exists := gs.Players[playerID]; exists && player.InvestigatorType != "" {
		// Selection already made, state will transition to SceneGame in the next updateScene call.
		return nil
	}

	s.handlePointerInput(gs, playerID)

	// Handle difficulty selection with E (easy), S (standard), H (hard).
	difficultyKeys := map[ebiten.Key]string{
		ebiten.KeyE: "easy",
		ebiten.KeyS: "standard",
		ebiten.KeyH: "hard",
	}
	for key, difficulty := range difficultyKeys {
		if inpututil.IsKeyJustPressed(key) && s.selectedDifficulty != difficulty {
			s.selectedDifficulty = difficulty
			s.game.net.SendAction(ebclient.PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: playerID,
				Action:   protocol.ActionSetDifficulty,
				Target:   difficulty,
			})
		}
	}

	s.handleDifficultyPointerInput(playerID)

	keys := []ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3,
		ebiten.Key4, ebiten.Key5, ebiten.Key6,
	}

	for i, key := range keys {
		if i < len(s.investigators) && inpututil.IsKeyJustPressed(key) {
			investigator := s.investigators[i]
			s.selectedInvestigator = investigator.name
			s.game.net.SendAction(ebclient.PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: playerID,
				Action:   protocol.ActionSelectInvestigator,
				Target:   investigator.name,
			})
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) && s.selectedInvestigator != "" {
		s.game.net.SendAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionSelectInvestigator,
			Target:   s.selectedInvestigator,
		})
	}

	return nil
}

// Draw renders the character selection menu with 6 options and their key bindings.
func (s *SceneCharacterSelect) Draw(screen *ebiten.Image) {
	gs, _, _ := s.game.state.Snapshot()
	selected, waiting := investigatorSelectionStatus(gs)
	layout := newCharacterSelectLayout()

	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	drawUIText(screen, "Select Your Investigator", screenWidth/2-100, screenHeight/2-110, color.White)
	drawUIText(screen, "Click a card or press [1-6] to choose", screenWidth/2-135, screenHeight/2-95, color.RGBA{R: 180, G: 180, B: 180, A: 255})

	for i, investigator := range s.investigators {
		rect := layout.cardRect(i)
		isSelected := s.selectedInvestigator == investigator.name
		hovered, pressed := pointerState(rect)
		if playerID := gs.CurrentPlayer; playerID != "" {
			if player, exists := gs.Players[playerID]; exists && player.InvestigatorType == protocol.InvestigatorType(investigator.name) && player.InvestigatorType != "" {
				isSelected = true
			}
		}
		keyLabel := strconv.Itoa(i + 1)

		fill := color.RGBA{R: 18, G: 20, B: 32, A: 240}
		border := color.RGBA{R: 115, G: 128, B: 165, A: 255}
		if hovered {
			fill = color.RGBA{R: 28, G: 32, B: 48, A: 245}
			border = color.RGBA{R: 150, G: 176, B: 220, A: 255}
		}
		if pressed {
			fill = color.RGBA{R: 40, G: 46, B: 66, A: 248}
			border = color.RGBA{R: 204, G: 220, B: 255, A: 255}
		}
		if isSelected {
			fill = color.RGBA{R: 38, G: 32, B: 18, A: 245}
			border = color.RGBA{R: 255, G: 220, B: 120, A: 255}
		}
		ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), fill)
		ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), 2, border)
		ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Max.Y-2), float64(rect.Dx()), 2, border)
		ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), 2, float64(rect.Dy()), border)
		ebitenutil.DrawRect(screen, float64(rect.Max.X-2), float64(rect.Min.Y), 2, float64(rect.Dy()), border)

		// Choose color based on selection state.
		titleColor := color.RGBA{R: 220, G: 220, B: 220, A: 255}
		descColor := color.RGBA{R: 160, G: 160, B: 160, A: 255}
		if isSelected {
			titleColor = color.RGBA{R: 255, G: 215, B: 100, A: 255} // Gold highlight
			descColor = color.RGBA{R: 200, G: 180, B: 100, A: 255}
		}

		// Draw the key, name, and description.
		text := "[" + keyLabel + "] " + investigator.display
		if isSelected {
			text += " ✓"
		}
		drawUIText(screen, text, rect.Min.X+12, rect.Min.Y+8, titleColor)
		drawUIText(screen, "    "+investigator.description, rect.Min.X+12, rect.Min.Y+22, descColor)
	}

	// Draw difficulty selection.
	drawUIText(screen, "Confirm Selection", layout.confirmButton.Min.X+20, layout.confirmButton.Min.Y+10, color.White)
	confirmHovered, confirmPressed := pointerState(layout.confirmButton)
	confirmEnabled := s.selectedInvestigator != ""
	confirmColor := color.RGBA{R: 80, G: 120, B: 195, A: 255}
	confirmBorder := color.RGBA{R: 220, G: 230, B: 255, A: 255}
	if confirmEnabled {
		confirmColor = color.RGBA{R: 120, G: 170, B: 240, A: 255}
	}
	if !confirmEnabled {
		confirmColor = color.RGBA{R: 60, G: 72, B: 96, A: 255}
		confirmBorder = color.RGBA{R: 132, G: 145, B: 182, A: 255}
	}
	if confirmEnabled && confirmHovered {
		confirmColor = color.RGBA{R: 132, G: 184, B: 255, A: 255}
		confirmBorder = color.RGBA{R: 236, G: 244, B: 255, A: 255}
	}
	if confirmEnabled && confirmPressed {
		confirmColor = color.RGBA{R: 154, G: 204, B: 255, A: 255}
		confirmBorder = color.RGBA{R: 250, G: 250, B: 255, A: 255}
	}
	ebitenutil.DrawRect(screen, float64(layout.confirmButton.Min.X), float64(layout.confirmButton.Min.Y), float64(layout.confirmButton.Dx()), float64(layout.confirmButton.Dy()), confirmColor)
	ebitenutil.DrawRect(screen, float64(layout.confirmButton.Min.X), float64(layout.confirmButton.Min.Y), float64(layout.confirmButton.Dx()), 2, confirmBorder)
	ebitenutil.DrawRect(screen, float64(layout.confirmButton.Min.X), float64(layout.confirmButton.Max.Y-2), float64(layout.confirmButton.Dx()), 2, confirmBorder)
	ebitenutil.DrawRect(screen, float64(layout.confirmButton.Min.X), float64(layout.confirmButton.Min.Y), 2, float64(layout.confirmButton.Dy()), confirmBorder)
	ebitenutil.DrawRect(screen, float64(layout.confirmButton.Max.X-2), float64(layout.confirmButton.Min.Y), 2, float64(layout.confirmButton.Dy()), confirmBorder)
	confirmLabel := "Tap a card first"
	if s.selectedInvestigator != "" {
		confirmLabel = "Tap here or press ENTER"
	}
	drawUIText(screen, confirmLabel, layout.confirmButton.Min.X+20, layout.confirmButton.Min.Y+22, color.White)

	drawUIText(screen, "Select Difficulty:", screenWidth/2-120, 468, color.White)
	for _, option := range layout.difficultyButtons() {
		hovered, pressed := pointerState(option.rect)
		fill := color.RGBA{R: 25, G: 28, B: 40, A: 240}
		border := color.RGBA{R: 130, G: 145, B: 175, A: 255}
		if hovered {
			fill = color.RGBA{R: 34, G: 42, B: 58, A: 245}
			border = color.RGBA{R: 165, G: 186, B: 224, A: 255}
		}
		if pressed {
			fill = color.RGBA{R: 44, G: 56, B: 76, A: 250}
			border = color.RGBA{R: 208, G: 220, B: 250, A: 255}
		}
		if s.selectedDifficulty == option.value {
			fill = color.RGBA{R: 45, G: 62, B: 96, A: 255}
			border = color.RGBA{R: 255, G: 220, B: 120, A: 255}
		}
		ebitenutil.DrawRect(screen, float64(option.rect.Min.X), float64(option.rect.Min.Y), float64(option.rect.Dx()), float64(option.rect.Dy()), fill)
		ebitenutil.DrawRect(screen, float64(option.rect.Min.X), float64(option.rect.Min.Y), float64(option.rect.Dx()), 2, border)
		ebitenutil.DrawRect(screen, float64(option.rect.Min.X), float64(option.rect.Max.Y-2), float64(option.rect.Dx()), 2, border)
		ebitenutil.DrawRect(screen, float64(option.rect.Min.X), float64(option.rect.Min.Y), 2, float64(option.rect.Dy()), border)
		ebitenutil.DrawRect(screen, float64(option.rect.Max.X-2), float64(option.rect.Min.Y), 2, float64(option.rect.Dy()), border)
		drawUIText(screen, option.label, option.rect.Min.X+18, option.rect.Min.Y+13, color.White)
	}
	drawUIText(screen, "Tap one of the three buttons or use E/S/H", screenWidth/2-135, 528, color.RGBA{R: 180, G: 180, B: 180, A: 255})

	// Highlight selected difficulty.
	if s.selectedDifficulty != "" {
		diffColor := color.RGBA{R: 255, G: 215, B: 100, A: 255} // Gold
		drawUIText(screen, "Current: "+strings.ToTitle(s.selectedDifficulty), screenWidth/2-120, screenHeight/2+170, diffColor)
	}

	drawUIText(screen, "", screenWidth/2-100, screenHeight/2+190, color.White) // Spacing
	drawUIText(screen, "Selected: "+strconv.Itoa(selected)+"  Waiting: "+strconv.Itoa(waiting), screenWidth/2-100, screenHeight/2+205, color.White)
	drawUIText(screen, "Scene advances when all connected players confirm.", screenWidth/2-150, screenHeight/2+220, color.RGBA{R: 180, G: 180, B: 180, A: 255})
}

func (s *SceneCharacterSelect) handlePointerInput(gs ebclient.GameState, playerID string) {
	layout := newCharacterSelectLayout()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.activateCharacterChoice(layout, x, y, gs, playerID)
		s.activateDifficultyChoice(layout, x, y, playerID)
	}
	for _, touchID := range inpututil.JustPressedTouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x < 0 || y < 0 {
			continue
		}
		s.activateCharacterChoice(layout, x, y, gs, playerID)
		s.activateDifficultyChoice(layout, x, y, playerID)
	}
}

func (s *SceneCharacterSelect) activateCharacterChoice(layout characterSelectLayout, x, y int, gs ebclient.GameState, playerID string) {
	if idx, ok := layout.cardIndexAt(x, y); ok && idx < len(s.investigators) {
		s.selectedInvestigator = s.investigators[idx].name
		return
	}
	if image.Pt(x, y).In(layout.confirmButton) && s.selectedInvestigator != "" {
		s.game.net.SendAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionSelectInvestigator,
			Target:   s.selectedInvestigator,
		})
	}
	_ = gs
}

func (s *SceneCharacterSelect) handleDifficultyPointerInput(playerID string) {
	_ = playerID
}

func (s *SceneCharacterSelect) activateDifficultyChoice(layout characterSelectLayout, x, y int, playerID string) {
	pt := image.Pt(x, y)
	for _, option := range layout.difficultyButtons() {
		if !pt.In(option.rect) || s.selectedDifficulty == option.value {
			continue
		}
		s.selectedDifficulty = option.value
		s.game.net.SendAction(ebclient.PlayerActionMessage{
			Type:     "playerAction",
			PlayerID: playerID,
			Action:   protocol.ActionSetDifficulty,
			Target:   option.value,
		})
		return
	}
}

type characterSelectLayout struct {
	cards         []image.Rectangle
	confirmButton image.Rectangle
}

func (l characterSelectLayout) difficultyButtons() []difficultyButton {
	if len(l.cards) == 0 {
		return nil
	}
	centerX := screenWidth / 2
	y := 500
	buttonWidth := 126
	buttonHeight := 36
	gap := 12
	left := centerX - buttonWidth - gap - buttonWidth/2
	return []difficultyButton{
		{value: "easy", label: "Easy", rect: image.Rect(left, y, left+buttonWidth, y+buttonHeight)},
		{value: "standard", label: "Standard", rect: image.Rect(centerX-buttonWidth/2, y, centerX+buttonWidth/2, y+buttonHeight)},
		{value: "hard", label: "Hard", rect: image.Rect(centerX+gap+buttonWidth/2, y, centerX+gap+buttonWidth/2+buttonWidth, y+buttonHeight)},
	}
}

func newCharacterSelectLayout() characterSelectLayout {
	cards := make([]image.Rectangle, 0, 6)
	startY := 120
	for i := 0; i < 6; i++ {
		y := startY + i*46
		cards = append(cards, image.Rect(screenWidth/2-220, y, screenWidth/2+220, y+40))
	}
	return characterSelectLayout{
		cards:         cards,
		confirmButton: image.Rect(screenWidth/2-110, 414, screenWidth/2+110, 452),
	}
}

func (l characterSelectLayout) cardRect(index int) image.Rectangle {
	if index < 0 || index >= len(l.cards) {
		return image.Rectangle{}
	}
	return l.cards[index]
}

func (l characterSelectLayout) cardIndexAt(x, y int) (int, bool) {
	pt := image.Pt(x, y)
	for i, rect := range l.cards {
		if pt.In(rect) {
			return i, true
		}
	}
	return 0, false
}

// SceneGameOver is shown when the game reaches a win or lose condition.
// It displays the outcome and prompts the player to close the window.
type SceneGameOver struct {
	game *Game
}

type gameOverLayout struct {
	playAgain image.Rectangle
	toLobby   image.Rectangle
}

func newGameOverLayout() gameOverLayout {
	return gameOverLayout{
		playAgain: image.Rect(screenWidth/2-172, screenHeight/2+54, screenWidth/2-16, screenHeight/2+92),
		toLobby:   image.Rect(screenWidth/2+16, screenHeight/2+54, screenWidth/2+188, screenHeight/2+92),
	}
}

func (l gameOverLayout) hitTest(x, y int) string {
	pt := image.Pt(x, y)
	switch {
	case pt.In(l.playAgain):
		return "play"
	case pt.In(l.toLobby):
		return "lobby"
	default:
		return ""
	}
}

// Update checks for Enter or Space key to restart the game.
// When pressed, it resets the local state and initiates reconnection.
func (s *SceneGameOver) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.restartGame()
	}

	layout := newGameOverLayout()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		s.activateGameOverControl(layout.hitTest(x, y))
	}
	for _, touchID := range inpututil.JustPressedTouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x < 0 || y < 0 {
			continue
		}
		s.activateGameOverControl(layout.hitTest(x, y))
	}
	return nil
}

func (s *SceneGameOver) activateGameOverControl(control string) {
	switch control {
	case "play":
		s.restartGame()
	case "lobby":
		s.returnToLobby()
	}
}

func (s *SceneGameOver) restartGame() {
	s.game.state.Reset()
	s.game.net.Connect()
	s.game.activateScene(&SceneConnect{game: s.game})
}

func (s *SceneGameOver) returnToLobby() {
	s.game.state.Reset()
	s.game.state.ClearDisplayName()
	s.game.activateScene(&SceneConnect{game: s.game})
}

// Draw renders the game-over overlay with the outcome message and restart prompt.
func (s *SceneGameOver) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	gs, _, _ := s.game.state.Snapshot()
	layout := newGameOverLayout()

	if gs.WinCondition {
		screen.Fill(color.RGBA{R: 10, G: 30, B: 10, A: 255})
		drawUIText(screen, "* INVESTIGATORS WIN! *", screenWidth/2-90, screenHeight/2-10, color.RGBA{R: 140, G: 255, B: 140, A: 255})
		drawUIText(screen, "The Ancient One's influence has been sealed.", screenWidth/2-130, screenHeight/2+10, color.White)
	} else {
		screen.Fill(color.RGBA{R: 30, G: 10, B: 10, A: 255})
		drawUIText(screen, "* ANCIENT ONE AWAKENS - YOU LOSE *", screenWidth/2-140, screenHeight/2-10, color.RGBA{R: 255, G: 140, B: 140, A: 255})
		drawUIText(screen, "Doom has consumed Arkham.", screenWidth/2-80, screenHeight/2+10, color.White)
	}
	playFill := color.RGBA{R: 60, G: 92, B: 140, A: 255}
	playBorder := color.RGBA{R: 206, G: 222, B: 255, A: 255}
	playHovered, playPressed := pointerState(layout.playAgain)
	if playHovered {
		playFill = color.RGBA{R: 78, G: 116, B: 168, A: 255}
		playBorder = color.RGBA{R: 226, G: 236, B: 255, A: 255}
	}
	if playPressed {
		playFill = color.RGBA{R: 96, G: 138, B: 192, A: 255}
		playBorder = color.RGBA{R: 242, G: 246, B: 255, A: 255}
	}
	ebitenutil.DrawRect(screen, float64(layout.playAgain.Min.X), float64(layout.playAgain.Min.Y), float64(layout.playAgain.Dx()), float64(layout.playAgain.Dy()), playFill)
	ebitenutil.DrawRect(screen, float64(layout.playAgain.Min.X), float64(layout.playAgain.Min.Y), float64(layout.playAgain.Dx()), 2, playBorder)
	ebitenutil.DrawRect(screen, float64(layout.playAgain.Min.X), float64(layout.playAgain.Max.Y-2), float64(layout.playAgain.Dx()), 2, playBorder)
	ebitenutil.DrawRect(screen, float64(layout.playAgain.Min.X), float64(layout.playAgain.Min.Y), 2, float64(layout.playAgain.Dy()), playBorder)
	ebitenutil.DrawRect(screen, float64(layout.playAgain.Max.X-2), float64(layout.playAgain.Min.Y), 2, float64(layout.playAgain.Dy()), playBorder)
	drawUIText(screen, "PLAY AGAIN", layout.playAgain.Min.X+30, layout.playAgain.Min.Y+11, color.White)

	lobbyFill := color.RGBA{R: 52, G: 54, B: 70, A: 255}
	lobbyBorder := color.RGBA{R: 184, G: 192, B: 225, A: 255}
	lobbyHovered, lobbyPressed := pointerState(layout.toLobby)
	if lobbyHovered {
		lobbyFill = color.RGBA{R: 68, G: 72, B: 94, A: 255}
		lobbyBorder = color.RGBA{R: 208, G: 214, B: 240, A: 255}
	}
	if lobbyPressed {
		lobbyFill = color.RGBA{R: 86, G: 90, B: 114, A: 255}
		lobbyBorder = color.RGBA{R: 228, G: 232, B: 248, A: 255}
	}
	ebitenutil.DrawRect(screen, float64(layout.toLobby.Min.X), float64(layout.toLobby.Min.Y), float64(layout.toLobby.Dx()), float64(layout.toLobby.Dy()), lobbyFill)
	ebitenutil.DrawRect(screen, float64(layout.toLobby.Min.X), float64(layout.toLobby.Min.Y), float64(layout.toLobby.Dx()), 2, lobbyBorder)
	ebitenutil.DrawRect(screen, float64(layout.toLobby.Min.X), float64(layout.toLobby.Max.Y-2), float64(layout.toLobby.Dx()), 2, lobbyBorder)
	ebitenutil.DrawRect(screen, float64(layout.toLobby.Min.X), float64(layout.toLobby.Min.Y), 2, float64(layout.toLobby.Dy()), lobbyBorder)
	ebitenutil.DrawRect(screen, float64(layout.toLobby.Max.X-2), float64(layout.toLobby.Min.Y), 2, float64(layout.toLobby.Dy()), lobbyBorder)
	drawUIText(screen, "RETURN TO LOBBY", layout.toLobby.Min.X+18, layout.toLobby.Min.Y+11, color.White)

	drawUIText(screen, "ENTER/SPACE: play again", screenWidth/2-102, screenHeight/2+104, color.White)
}

// updateScene evaluates current state and transitions to the appropriate scene.
// Called from Game.Update() every tick.
func (g *Game) updateScene() {
	gs, playerID, connected := g.state.Snapshot()
	_, displayName := g.state.ConnectFormSnapshot()

	if g.ensureGameOverScene(gs) {
		return
	}

	if !connected {
		g.setConnectScene()
		return
	}

	if strings.TrimSpace(displayName) == "" {
		g.setConnectScene()
		return
	}

	if g.localPlayerNeedsSelection(gs, playerID) || !allConnectedPlayersSelected(gs) {
		g.setCharacterSelectScene()
		return
	}

	g.setGameScene()
}

func (g *Game) ensureGameOverScene(gs ebclient.GameState) bool {
	if !gs.WinCondition && !gs.LoseCondition {
		return false
	}
	if _, ok := g.activeScene.(*SceneGameOver); !ok {
		g.activateScene(&SceneGameOver{game: g})
	}
	return true
}

func (g *Game) localPlayerNeedsSelection(gs ebclient.GameState, playerID string) bool {
	player, exists := gs.Players[playerID]
	if !exists || player == nil {
		return true
	}
	return player.InvestigatorType == ""
}

func (g *Game) setConnectScene() {
	if _, ok := g.activeScene.(*SceneConnect); !ok {
		g.activateScene(&SceneConnect{game: g})
	}
}

func (g *Game) setCharacterSelectScene() {
	if _, ok := g.activeScene.(*SceneCharacterSelect); !ok {
		g.activateScene(NewSceneCharacterSelect(g))
	}
}

func (g *Game) setGameScene() {
	if _, ok := g.activeScene.(*SceneGame); !ok {
		g.activateScene(&SceneGame{game: g})
	}
}

func (g *Game) activateScene(next Scene) {
	if next == nil {
		return
	}
	if sceneTypeName(g.activeScene) == sceneTypeName(next) {
		g.activeScene = next
		return
	}
	g.activeScene = next
	g.startSceneFade()
}

func sceneTypeName(scene Scene) string {
	switch scene.(type) {
	case *SceneConnect:
		return "connect"
	case *SceneCharacterSelect:
		return "select"
	case *SceneGame:
		return "game"
	case *SceneGameOver:
		return "gameover"
	default:
		return "unknown"
	}
}

func pointerState(rect image.Rectangle) (hovered, pressed bool) {
	if rect.Empty() {
		return false, false
	}
	x, y := ebiten.CursorPosition()
	pt := image.Pt(x, y)
	hovered = pt.In(rect)
	pressed = hovered && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	for _, touchID := range ebiten.TouchIDs() {
		tx, ty := ebiten.TouchPosition(touchID)
		if image.Pt(tx, ty).In(rect) {
			hovered = true
			pressed = true
			break
		}
	}
	return hovered, pressed
}

func allConnectedPlayersSelected(gs ebclient.GameState) bool {
	connectedPlayers := 0
	selectedPlayers := 0
	for _, pid := range gs.TurnOrder {
		player, ok := gs.Players[pid]
		if !ok || player == nil || !player.Connected {
			continue
		}
		connectedPlayers++
		if player.InvestigatorType != "" {
			selectedPlayers++
		}
	}

	return connectedPlayers > 0 && selectedPlayers == connectedPlayers
}

func investigatorSelectionStatus(gs ebclient.GameState) (selected, waiting int) {
	for _, pid := range gs.TurnOrder {
		player, ok := gs.Players[pid]
		if !ok || player == nil || !player.Connected {
			continue
		}
		if player.InvestigatorType == "" {
			waiting++
			continue
		}
		selected++
	}

	return selected, waiting
}
