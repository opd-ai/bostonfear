// Package app — scene state machine for the Arkham Horror Ebitengine client.
//
// CLIENT_SPEC.md §1 requires four scenes:
//
//	SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver
//
// All four scenes are implemented here.
package app

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
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

const (
	connectFieldAddress = iota
	connectFieldName
)

// Update handles connect-form input and reconnect countdown state.
func (s *SceneConnect) Update() error {
	s.tick++
	s.handleFieldSwitch()
	s.handleFieldEdits()
	s.updateReconnectTick()
	s.confirmConnectForm()
	return nil
}

// Draw renders the connecting screen using anchor-based layout.
func (s *SceneConnect) Draw(screen *ebiten.Image) {
	gs, playerID, connected := s.game.state.Snapshot()
	address, displayName := s.game.state.ConnectFormSnapshot()
	token := s.game.state.GetReconnectToken()

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
	serverConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 190,
		Width:   320,
		Height:  16,
	}
	serverBounds := serverConstraint.Bounds(vp)
	drawUIText(screen, trimToWidth("Server: "+address, 320), serverBounds.Min.X, serverBounds.Min.Y, color.White)

	// Display name field.
	nameConstraint := ui.Constraint{
		Anchor:  ui.AnchorTopCenter,
		OffsetY: 215,
		Width:   320,
		Height:  16,
	}
	nameBounds := nameConstraint.Bounds(vp)
	drawUIText(screen, trimToWidth("Display Name: "+displayName, 320), nameBounds.Min.X, nameBounds.Min.Y, color.White)

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
		OffsetY: 350,
		Width:   400,
		Height:  16,
	}
	instrBounds := instrConstraint.Bounds(vp)
	drawUIText(screen, "Type to edit fields - BACKSPACE deletes - ENTER confirms", instrBounds.Min.X, instrBounds.Min.Y, color.RGBA{R: 220, G: 220, B: 220, A: 255})

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

// Update processes player input and handles per-tick game logic.
func (s *SceneGame) Update() error {
	s.game.input.Update()
	return nil
}

// Draw composites the full game board via the layered renderer.
func (s *SceneGame) Draw(screen *ebiten.Image) {
	s.game.drawGameContent(screen)
}

// SceneCharacterSelect is shown after connection but before the game begins,
// allowing the player to choose their investigator archetype from 6 options.
// It transitions to SceneGame once the selection is confirmed.
type SceneCharacterSelect struct {
	game *Game

	// investigators holds the list of available investigator archetypes.
	// They are indexed 0-5 and map to keys 1-6.
	investigators []struct {
		name    string // "researcher", "detective", etc.
		display string // "Researcher", "Detective", etc.
	}
}

// NewSceneCharacterSelect creates a new character selection scene.
func NewSceneCharacterSelect(g *Game) *SceneCharacterSelect {
	return &SceneCharacterSelect{
		game: g,
		investigators: []struct {
			name    string
			display string
		}{
			{"researcher", "Researcher"},
			{"detective", "Detective"},
			{"occultist", "Occultist"},
			{"soldier", "Soldier"},
			{"mystic", "Mystic"},
			{"survivor", "Survivor"},
		},
	}
}

// Update checks for key presses 1-6 and sends the selection to the server.
func (s *SceneCharacterSelect) Update() error {
	gs, playerID, _ := s.game.state.Snapshot()

	// Check if this player has already selected an investigator.
	if player, exists := gs.Players[playerID]; exists && player.InvestigatorType != "" {
		// Selection already made, state will transition to SceneGame in the next updateScene call.
		return nil
	}

	keys := []ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3,
		ebiten.Key4, ebiten.Key5, ebiten.Key6,
	}

	for i, key := range keys {
		if i < len(s.investigators) && inpututil.IsKeyJustPressed(key) {
			investigator := s.investigators[i]
			s.game.net.SendAction(ebclient.PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: playerID,
				Action:   protocol.ActionSelectInvestigator,
				Target:   investigator.name,
			})
		}
	}

	return nil
}

// Draw renders the character selection menu with 6 options and their key bindings.
func (s *SceneCharacterSelect) Draw(screen *ebiten.Image) {
	gs, _, _ := s.game.state.Snapshot()
	selected, waiting := investigatorSelectionStatus(gs)

	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	drawUIText(screen, "Select Your Investigator", screenWidth/2-100, screenHeight/2-80, color.White)

	for i, investigator := range s.investigators {
		yOffset := screenHeight/2 - 50 + i*20
		keyLabel := strconv.Itoa(i + 1)
		text := "[" + keyLabel + "] " + investigator.display
		drawUIText(screen, text, screenWidth/2-80, yOffset, color.RGBA{R: 220, G: 220, B: 220, A: 255})
	}

	drawUIText(screen, "Selected: "+strconv.Itoa(selected)+"  Waiting: "+strconv.Itoa(waiting), screenWidth/2-100, screenHeight/2+90, color.White)
	drawUIText(screen, "Scene advances when all connected players confirm.", screenWidth/2-150, screenHeight/2+110, color.RGBA{R: 220, G: 220, B: 220, A: 255})
}

// SceneGameOver is shown when the game reaches a win or lose condition.
// It displays the outcome and prompts the player to close the window.
type SceneGameOver struct {
	game *Game
}

// Update checks for Enter or Space key to restart the game.
// When pressed, it resets the local state and initiates reconnection.
func (s *SceneGameOver) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.game.state.Reset()
		s.game.net.Connect()
		s.game.activeScene = &SceneConnect{game: s.game}
	}
	return nil
}

// Draw renders the game-over overlay with the outcome message and restart prompt.
func (s *SceneGameOver) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	gs, _, _ := s.game.state.Snapshot()

	if gs.WinCondition {
		screen.Fill(color.RGBA{R: 10, G: 30, B: 10, A: 255})
		drawUIText(screen, "* INVESTIGATORS WIN! *", screenWidth/2-90, screenHeight/2-10, color.RGBA{R: 140, G: 255, B: 140, A: 255})
		drawUIText(screen, "The Ancient One's influence has been sealed.", screenWidth/2-130, screenHeight/2+10, color.White)
	} else {
		screen.Fill(color.RGBA{R: 30, G: 10, B: 10, A: 255})
		drawUIText(screen, "* ANCIENT ONE AWAKENS - YOU LOSE *", screenWidth/2-140, screenHeight/2-10, color.RGBA{R: 255, G: 140, B: 140, A: 255})
		drawUIText(screen, "Doom has consumed Arkham.", screenWidth/2-80, screenHeight/2+10, color.White)
	}
	drawUIText(screen, "Press ENTER to play again - Close window to exit.", screenWidth/2-150, screenHeight/2+30, color.White)
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
		g.activeScene = &SceneGameOver{game: g}
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
		g.activeScene = &SceneConnect{game: g}
	}
}

func (g *Game) setCharacterSelectScene() {
	if _, ok := g.activeScene.(*SceneCharacterSelect); !ok {
		g.activeScene = NewSceneCharacterSelect(g)
	}
}

func (g *Game) setGameScene() {
	if _, ok := g.activeScene.(*SceneGame); !ok {
		g.activeScene = &SceneGame{game: g}
	}
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

func investigatorSelectionStatus(gs ebclient.GameState) (selected int, waiting int) {
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
