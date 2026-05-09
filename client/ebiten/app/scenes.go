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
// It renders an animated "Connecting…" banner and transitions to SceneGame
// automatically once the connection is confirmed.
type SceneConnect struct {
	game *Game
	tick int
}

// Update increments the tick counter used for the connecting animation.
func (s *SceneConnect) Update() error {
	s.tick++
	return nil
}

// Draw renders the connecting screen.
func (s *SceneConnect) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	dots := "...."[:s.tick/15%5]
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Connecting to server%s", dots),
		screenWidth/2-100, screenHeight/2)
	ebitenutil.DebugPrintAt(screen, "Boston Fear — Arkham Horror", screenWidth/2-100, screenHeight/2-20)
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
	screen.Fill(color.RGBA{R: 10, G: 10, B: 20, A: 255})
	ebitenutil.DebugPrintAt(screen, "Select Your Investigator", screenWidth/2-100, screenHeight/2-80)

	for i, investigator := range s.investigators {
		yOffset := screenHeight/2 - 50 + i*20
		keyLabel := fmt.Sprintf("%d", i+1)
		text := fmt.Sprintf("[%s] %s", keyLabel, investigator.display)
		ebitenutil.DebugPrintAt(screen, text, screenWidth/2-80, yOffset)
	}
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

	// Game over takes priority.
	if gs.WinCondition || gs.LoseCondition {
		if _, ok := g.activeScene.(*SceneGameOver); !ok {
			g.activeScene = &SceneGameOver{game: g}
		}
		return
	}

	// Connected but investigator not selected → SceneCharacterSelect.
	if connected {
		if player, exists := gs.Players[playerID]; exists && player.InvestigatorType == "" {
			if _, ok := g.activeScene.(*SceneCharacterSelect); !ok {
				g.activeScene = NewSceneCharacterSelect(g)
			}
			return
		}
	}

	// Connected and character selected → SceneGame.
	if connected {
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
