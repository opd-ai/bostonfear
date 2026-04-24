// Package app — scene state machine for the Arkham Horror Ebitengine client.
//
// CLIENT_SPEC.md §1 requires four scenes:
//
//	SceneConnect → SceneCharacterSelect → SceneGame → SceneGameOver
//
// SceneCharacterSelect is deferred until the selectInvestigator server action is
// fully wired on the client side. The remaining three scenes are implemented here.
package app

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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

// SceneGameOver is shown when the game reaches a win or lose condition.
// It displays the outcome and prompts the player to close the window.
type SceneGameOver struct {
	game *Game
}

// Update is a no-op; the game-over screen accepts no player actions.
func (s *SceneGameOver) Update() error {
	return nil
}

// Draw renders the game-over overlay with the outcome message.
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
	ebitenutil.DebugPrintAt(screen, "Close the window to exit.", screenWidth/2-80, screenHeight/2+30)
}

// updateScene evaluates current state and transitions to the appropriate scene.
// Called from Game.Update() every tick.
func (g *Game) updateScene() {
	gs, _, connected := g.state.Snapshot()

	// Game over takes priority.
	if gs.WinCondition || gs.LoseCondition {
		if _, ok := g.activeScene.(*SceneGameOver); !ok {
			g.activeScene = &SceneGameOver{game: g}
		}
		return
	}

	// Connected and in-game → SceneGame.
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
