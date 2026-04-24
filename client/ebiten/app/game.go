// Package app implements the Ebitengine game loop for the Arkham Horror client.
// It wires together LocalState, NetClient, InputHandler, and the layered
// Compositor into an ebiten.Game implementation. cmd/desktop, cmd/web, and
// cmd/mobile all call NewGame to obtain a ready-to-run game object.
//
// This package is separated from the parent client/ebiten package so that the
// pure network/state logic in that package can be tested without a display.
package app

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/client/ebiten/render"
)

// screenWidth and screenHeight define the logical resolution (800×600 minimum).
const (
	screenWidth  = 800
	screenHeight = 600
)

// locationRects maps each location name to its board rectangle (x, y, w, h).
// The layout places four neighbourhoods in a 2×2 grid with gutters.
var locationRects = map[ebclient.Location]struct{ x, y, w, h int }{
	"Downtown":   {40, 60, 160, 100},
	"University": {220, 60, 160, 100},
	"Rivertown":  {40, 220, 160, 100},
	"Northside":  {220, 220, 160, 100},
}

// locationColours assigns a distinct background colour to each neighbourhood.
var locationColours = map[ebclient.Location]color.RGBA{
	"Downtown":   {R: 60, G: 80, B: 140, A: 255},
	"University": {R: 60, G: 120, B: 80, A: 255},
	"Rivertown":  {R: 120, G: 60, B: 60, A: 255},
	"Northside":  {R: 100, G: 80, B: 140, A: 255},
}

// playerColours cycles through distinctive colours for up to 6 players.
var playerColours = []color.RGBA{
	{R: 255, G: 220, B: 50, A: 255},
	{R: 50, G: 220, B: 255, A: 255},
	{R: 255, G: 100, B: 50, A: 255},
	{R: 150, G: 255, B: 100, A: 255},
	{R: 255, G: 100, B: 200, A: 255},
	{R: 200, G: 200, B: 255, A: 255},
}

// Game implements the ebiten.Game interface and drives the Arkham Horror client.
// Update processes input and network events each tick; Draw renders the board.
// The layered renderer composites board, tokens, effects, UI, and animations in
// correct z-order every frame.
type Game struct {
	state       *ebclient.LocalState
	net         *ebclient.NetClient
	input       *InputHandler
	renderer    *render.Compositor
	shaders     *render.ShaderSet // lazily compiled on first Draw call; nil until then
	activeScene Scene             // current full-screen scene; managed by updateScene
}

// NewGame creates a Game connected to the given server URL.
// Call ebiten.RunGame(game) to start the event loop.
func NewGame(serverURL string) *Game {
	state := ebclient.NewLocalState(serverURL)
	net := ebclient.NewNetClient(state)
	input := NewInputHandler(net, state)
	net.Connect()
	g := &Game{
		state:    state,
		net:      net,
		input:    input,
		renderer: render.NewCompositor(),
	}
	g.activeScene = &SceneConnect{game: g}
	return g
}

// Update is called every tick (60 TPS by default).
// It transitions between scenes based on connection and game-end state,
// then delegates per-tick logic to the active scene.
func (g *Game) Update() error {
	g.updateScene()
	if g.activeScene != nil {
		return g.activeScene.Update()
	}
	// Fallback when activeScene is unset (e.g. in unit tests that build Game directly).
	g.input.Update()
	return nil
}

// Layout returns the logical screen dimensions regardless of the window size.
func (g *Game) Layout(_, _ int) (int, int) {
	return screenWidth, screenHeight
}

// Draw renders the full game board each frame using the five-layer pipeline.
// Layers are flushed in z-order: board → tokens → effects → UI → animation.
func (g *Game) Draw(screen *ebiten.Image) {
	if g.activeScene != nil {
		g.activeScene.Draw(screen)
		return
	}
	// Fallback when activeScene is unset (e.g. in unit tests that build Game directly).
	g.drawGameContent(screen)
}

// drawGameContent renders the in-game view. Called from SceneGame.Draw and
// from the Draw fallback path when no active scene is set.
func (g *Game) drawGameContent(screen *ebiten.Image) {
	// Lazily compile Kage shaders on first draw. Errors are logged but non-fatal:
	// the game renders correctly without shader effects, just without the doom vignette.
	if g.shaders == nil {
		if ss, err := render.NewShaderSet(); err == nil {
			g.shaders = ss
		} else {
			log.Printf("shader compilation failed (vignette disabled): %v", err)
			// Assign a sentinel non-nil value so we don't retry every frame.
			g.shaders = &render.ShaderSet{}
		}
	}

	screen.Fill(color.RGBA{R: 20, G: 20, B: 30, A: 255})

	gs, playerID, connected := g.state.Snapshot()

	// Layer 0 — Board: location tiles.
	g.enqueueBoard(gs)

	// Layer 1 — Tokens: investigator tokens on their location tiles.
	g.enqueueTokens(gs, playerID)

	// Layer 2 — Effects: doom bar segment.
	g.enqueueDoomEffect(gs)

	// Flush layers 0-2 and 4 (no animation sprites yet) before overlaying text UI.
	g.renderer.Flush(screen)

	// Layer 3 — UI: text overlays drawn directly on screen after sprite flush.
	g.drawConnectionBanner(screen, connected)
	g.drawDoomCounter(screen, gs)
	g.drawPlayerPanel(screen, gs, playerID)
	g.drawEventLog(screen)
	g.drawInputHints(screen, gs, playerID)

	// Post-process: doom vignette shader composited over the fully rendered frame.
	if gs.Doom > 0 {
		render.DrawDoomVignette(screen, g.shaders, float32(gs.Doom)/12)
	}
}

// enqueueBoard adds one board-layer draw command per location.
func (g *Game) enqueueBoard(gs ebclient.GameState) {
	for loc, rect := range locationRects {
		g.renderer.Enqueue(render.LayerBoard, render.DrawCmd{
			Sprite: render.LocationSpriteID(string(loc)),
			X:      float64(rect.x),
			Y:      float64(rect.y),
		})
		_ = gs // board layout is static; gs unused here
	}
}

// enqueueTokens adds token-layer draw commands for each connected player.
func (g *Game) enqueueTokens(gs ebclient.GameState, myID string) {
	occupants := make(map[ebclient.Location]int) // count tokens per location for offset
	for i, pid := range gs.TurnOrder {
		p, ok := gs.Players[pid]
		if !ok || !p.Connected {
			continue
		}
		rect := locationRects[p.Location]
		offset := occupants[p.Location]
		occupants[p.Location]++

		col := playerColours[i%len(playerColours)]
		if pid == myID {
			// Slightly enlarge own token to make it stand out.
			col.A = 255
		}
		g.renderer.Enqueue(render.LayerTokens, render.DrawCmd{
			Sprite: render.SpritePlayerToken,
			X:      float64(rect.x + 4 + offset*14),
			Y:      float64(rect.y + rect.h - 16),
			Tint:   col,
		})
	}
}

// enqueueDoomEffect adds an effect-layer doom-marker segment scaled to doom level.
func (g *Game) enqueueDoomEffect(gs ebclient.GameState) {
	if gs.Doom <= 0 {
		return
	}
	fraction := float64(gs.Doom) / 12.0
	g.renderer.Enqueue(render.LayerEffects, render.DrawCmd{
		Sprite: render.SpriteDoomMarker,
		X:      420,
		Y:      76,
		ScaleX: fraction * (200.0 / 64), // stretch 64-px tile to fill bar width
		ScaleY: 14.0 / 64,
	})
}

// drawConnectionBanner shows a "Connecting…" overlay when the WebSocket is down.
func (g *Game) drawConnectionBanner(screen *ebiten.Image, connected bool) {
	if connected {
		return
	}
	ebitenutil.DebugPrintAt(screen, "[ Connecting to server… ]", screenWidth/2-90, 8)
}

// drawDoomCounter renders the global doom track (0–12) on the right side.
func (g *Game) drawDoomCounter(screen *ebiten.Image, gs ebclient.GameState) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DOOM: %d / 12", gs.Doom), 420, 60)

	// Draw a simple bar.
	barW := 200.0
	filled := float64(gs.Doom) / 12.0 * barW
	ebitenutil.DrawRect(screen, 420, 76, barW, 14, color.RGBA{R: 60, G: 60, B: 60, A: 255})
	ebitenutil.DrawRect(screen, 420, 76, filled, 14, color.RGBA{R: 200, G: 40, B: 40, A: 255})
}

// drawPlayerPanel renders resource levels for all players on the right side.
func (g *Game) drawPlayerPanel(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	y := 110
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Phase: %s", gs.GamePhase), 420, y)
	y += 14

	for i, pid := range gs.TurnOrder {
		p, ok := gs.Players[pid]
		if !ok {
			continue
		}
		col := playerColours[i%len(playerColours)]
		marker := " "
		if pid == gs.CurrentPlayer {
			marker = "►"
		}
		me := ""
		if pid == myID {
			me = " (you)"
		}
		label := fmt.Sprintf("%s %s%s  HP:%d SN:%d CL:%d ACT:%d",
			marker, pid, me,
			p.Resources.Health, p.Resources.Sanity, p.Resources.Clues,
			p.ActionsRemaining)

		// Draw background highlight for current player.
		if pid == gs.CurrentPlayer {
			ebitenutil.DrawRect(screen, 418, float64(y-1), 370, 13,
				color.RGBA{R: col.R / 4, G: col.G / 4, B: col.B / 4, A: 255})
		}
		ebitenutil.DebugPrintAt(screen, label, 420, y)
		y += 14
	}

	if len(gs.TurnOrder) == 0 {
		ebitenutil.DebugPrintAt(screen, "Waiting for players…", 420, y)
	}
}

// drawEventLog renders the last 8 events in the lower-right corner.
func (g *Game) drawEventLog(screen *ebiten.Image) {
	entries := g.state.EventLogSnapshot()
	y := screenHeight - 130
	ebitenutil.DebugPrintAt(screen, "── Event Log ──", 420, y)
	y += 12

	// Show the last 8 entries.
	start := 0
	if len(entries) > 8 {
		start = len(entries) - 8
	}
	for _, e := range entries[start:] {
		ebitenutil.DebugPrintAt(screen, e.Text, 420, y)
		y += 12
	}
}

// drawInputHints renders the keyboard shortcut legend in the lower-left corner.
func (g *Game) drawInputHints(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	y := screenHeight - 130
	ebitenutil.DebugPrintAt(screen, "── Controls ──", 10, y)
	y += 12

	isMyTurn := gs.CurrentPlayer == myID && gs.GamePhase == "playing"
	status := "waiting for your turn"
	if isMyTurn {
		status = fmt.Sprintf("YOUR TURN  (%d actions left)", func() int {
			if p, ok := gs.Players[myID]; ok {
				return p.ActionsRemaining
			}
			return 0
		}())
	}
	ebitenutil.DebugPrintAt(screen, status, 10, y)
	y += 12

	hints := []string{
		"1 Move→Downtown  2 Move→University",
		"3 Move→Rivertown 4 Move→Northside",
		"G Gather  I Investigate  W Ward",
	}
	for _, h := range hints {
		ebitenutil.DebugPrintAt(screen, h, 10, y)
		y += 12
	}

	// Win / lose banner.
	if gs.WinCondition {
		ebitenutil.DebugPrintAt(screen, "✦ INVESTIGATORS WIN! ✦", 10, screenHeight-20)
	} else if gs.LoseCondition {
		ebitenutil.DebugPrintAt(screen, "✦ ANCIENT ONE AWAKENS — YOU LOSE ✦", 10, screenHeight-20)
	}
}

// playerColourIndex returns a stable colour index for pid based on turn order.
func playerColourIndex(pid string, order []string) int {
	for i, id := range order {
		if id == pid {
			return i % len(playerColours)
		}
	}
	return 0
}

// min8 returns the smaller of a and b, clamped to [0,255].
func min8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}
