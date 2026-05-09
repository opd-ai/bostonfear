// Package app — scene state machine for the Arkham Horror Ebitengine client.
//
// CLIENT_SPEC.md §1 requires four scenes:
//
//	SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver
//
// All four scenes are implemented here.
package app

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
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

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		s.activeField = (s.activeField + 1) % 2
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		s.handleBackspace()
	}

	for _, r := range ebiten.AppendInputChars(nil) {
		s.appendRune(r)
	}

	gs, _, connected := s.game.state.Snapshot()
	token := s.game.state.GetReconnectToken()
	if !connected && token != "" {
		s.reconnectTick++
	} else {
		s.reconnectTick = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		address, name := s.game.state.ConnectFormSnapshot()
		if strings.TrimSpace(address) != "" {
			s.game.state.SetConnectAddress(address)
		}
		if strings.TrimSpace(name) == "" {
			s.game.state.SetDisplayName("Investigator")
		}
	}

	_ = gs
	return nil
}

// Draw renders the connecting screen.
func (s *SceneConnect) Draw(screen *ebiten.Image) {
	gs, playerID, connected := s.game.state.Snapshot()
	address, displayName := s.game.state.ConnectFormSnapshot()
	token := s.game.state.GetReconnectToken()

	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	dots := "...."[:s.tick/15%5]
	ebitenutil.DebugPrintAt(screen, "Boston Fear — Arkham Horror", screenWidth/2-120, 120)

	status := fmt.Sprintf("Connecting%s", dots)
	if connected {
		status = "Connected"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", status), screenWidth/2-120, 160)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Server: %s", address), screenWidth/2-120, 190)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Display Name: %s", displayName), screenWidth/2-120, 215)

	connectedPlayers := countConnectedPlayers(gs)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Slots: %d/6", connectedPlayers), screenWidth/2-120, 245)
	if connectedPlayers >= 6 {
		ebitenutil.DebugPrintAt(screen, "Game Full (6/6)", screenWidth/2-120, 260)
	}

	if token != "" && !connected {
		remaining := 60 - s.reconnectTick/60
		if remaining < 0 {
			remaining = 0
		}
		ebitenutil.DebugPrintAt(screen,
			fmt.Sprintf("Reconnecting with saved session... %ds", remaining),
			screenWidth/2-120, 290)
	}

	fieldLabel := "address"
	if s.activeField == connectFieldName {
		fieldLabel = "display name"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Editing: %s (TAB to switch)", fieldLabel), screenWidth/2-120, 330)
	ebitenutil.DebugPrintAt(screen, "Type to edit fields • BACKSPACE deletes • ENTER confirms", screenWidth/2-120, 350)

	if playerID != "" {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Player ID: %s", playerID), screenWidth/2-120, 380)
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
	ebitenutil.DebugPrintAt(screen, "Select Your Investigator", screenWidth/2-100, screenHeight/2-80)

	for i, investigator := range s.investigators {
		yOffset := screenHeight/2 - 50 + i*20
		keyLabel := fmt.Sprintf("%d", i+1)
		text := fmt.Sprintf("[%s] %s", keyLabel, investigator.display)
		ebitenutil.DebugPrintAt(screen, text, screenWidth/2-80, yOffset)
	}

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Selected: %d  Waiting: %d", selected, waiting), screenWidth/2-100, screenHeight/2+90)
	ebitenutil.DebugPrintAt(screen, "Scene advances when all connected players confirm.", screenWidth/2-150, screenHeight/2+110)
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
		ebitenutil.DebugPrintAt(screen, "✦ INVESTIGATORS WIN! ✦", screenWidth/2-90, screenHeight/2-10)
		ebitenutil.DebugPrintAt(screen, "The Ancient One's influence has been sealed.", screenWidth/2-130, screenHeight/2+10)
	} else {
		screen.Fill(color.RGBA{R: 30, G: 10, B: 10, A: 255})
		ebitenutil.DebugPrintAt(screen, "✦ ANCIENT ONE AWAKENS — YOU LOSE ✦", screenWidth/2-140, screenHeight/2-10)
		ebitenutil.DebugPrintAt(screen, "Doom has consumed Arkham.", screenWidth/2-80, screenHeight/2+10)
	}
	ebitenutil.DebugPrintAt(screen, "Press ENTER to play again • Close window to exit.", screenWidth/2-150, screenHeight/2+30)
}

// updateScene evaluates current state and transitions to the appropriate scene.
// Called from Game.Update() every tick.
func (g *Game) updateScene() {
	gs, playerID, connected := g.state.Snapshot()
	_, displayName := g.state.ConnectFormSnapshot()

	// Game over takes priority.
	if gs.WinCondition || gs.LoseCondition {
		if _, ok := g.activeScene.(*SceneGameOver); !ok {
			g.activeScene = &SceneGameOver{game: g}
		}
		return
	}

	// Connected, named, but investigator not selected → SceneCharacterSelect.
	if connected {
		if strings.TrimSpace(displayName) == "" {
			if _, ok := g.activeScene.(*SceneConnect); !ok {
				g.activeScene = &SceneConnect{game: g}
			}
			return
		}

		if player, exists := gs.Players[playerID]; exists && player.InvestigatorType == "" {
			if _, ok := g.activeScene.(*SceneCharacterSelect); !ok {
				g.activeScene = NewSceneCharacterSelect(g)
			}
			return
		}
	}

	// Connected and local character selected, but others are still picking.
	if connected {
		if !allConnectedPlayersSelected(gs) {
			if _, ok := g.activeScene.(*SceneCharacterSelect); !ok {
				g.activeScene = NewSceneCharacterSelect(g)
			}
			return
		}

		if _, ok := g.activeScene.(*SceneGame); !ok {
			g.activeScene = &SceneGame{game: g}
		}
		return
	}

	// Not connected → SceneConnect (initial state or reconnecting).
	if _, ok := g.activeScene.(*SceneConnect); !ok {
		g.activeScene = &SceneConnect{game: g}
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
