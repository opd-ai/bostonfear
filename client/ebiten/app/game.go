// Package app implements the Ebitengine game loop for the Arkham Horror client.
// It wires together LocalState, NetClient, InputHandler, and the layered
// Compositor into an ebiten.Game implementation. cmd/desktop, cmd/web, and
// cmd/mobile all call NewGame to obtain a ready-to-run game object.
//
// This package is separated from the parent client/ebiten package so that the
// pure network/state logic in that package can be tested without a display.
package app

import (
	"image"
	"image/color"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/client/ebiten/render"
	"github.com/opd-ai/bostonfear/client/ebiten/ui"
	"github.com/opd-ai/bostonfear/client/ebiten/ui/feedback"
	"github.com/opd-ai/bostonfear/client/ebiten/ui/hud"
	"github.com/opd-ai/bostonfear/client/ebiten/ui/onboarding"
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

// boardAdjacency mirrors the server's four-location movement graph so the UI
// can highlight legal movement targets without querying the server for a list.
var boardAdjacency = map[ebclient.Location][]ebclient.Location{
	"Downtown":   {"University", "Rivertown"},
	"University": {"Downtown", "Northside"},
	"Rivertown":  {"Downtown", "Northside"},
	"Northside":  {"University", "Rivertown"},
}

var moveShortcutHints = map[ebclient.Location]string{
	"Downtown":   "1",
	"University": "2",
	"Rivertown":  "3",
	"Northside":  "4",
}

var actionShortcutHints = map[string]string{
	"gather":      "G",
	"investigate": "I",
	"ward":        "W",
	"focus":       "F",
	"research":    "R",
	"trade":       "T",
	"component":   "C",
	"attack":      "A",
	"evade":       "E",
	"closegate":   "X",
	"encounter":   "N",
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
	quality     ui.QualityTier
	effects     ui.EffectProfile
	theme       ui.ThemePack
	tokens      *ui.DesignTokenRegistry
	icons       *ui.IconRegistry
	onboarding  *onboarding.OnboardingController
	stateBanner *ui.StateVisibilityWidget
	results     *hud.ResultsPanel
	lastOutcome string
	procedural  *ui.ProceduralGenerator
	camera      *ui.Camera
	boardView   *ui.BoardView
	procSeed    uint64
	procFrame   ui.ProceduralFrame
	procAtFrame int64
	startedAt   time.Time
	frameCount  int64
	activeScene Scene // current full-screen scene; managed by updateScene
	sceneFade   int
	uiCache     uiStringCache
}

type uiStringCache struct {
	doomValue  int
	doomLabel  string
	phaseValue string
	phaseLabel string
	statusKey  string
	statusText string
}

// NewGame creates a Game connected to the given server URL.
// Call ebiten.RunGame(game) to start the event loop.
func NewGame(serverURL string) *Game {
	state := ebclient.NewLocalState(serverURL)
	net := ebclient.NewNetClient(state)
	input := NewInputHandler(net, state)
	net.Connect()
	quality := ui.ParseQualityTier(os.Getenv("BOSTONFEAR_RENDER_QUALITY"))
	tokens := ui.NewDefaultArkhamTheme()
	g := &Game{
		state:       state,
		net:         net,
		input:       input,
		renderer:    render.NewCompositor(),
		quality:     quality,
		effects:     ui.EffectProfileForTier(quality),
		theme:       ui.ResolveThemePack(tokens),
		tokens:      tokens,
		icons:       ui.NewIconRegistry(),
		onboarding:  onboarding.NewOnboardingController(onboarding.DefaultArkhamHorrorOnboarding()),
		stateBanner: ui.NewStateVisibilityWidget(),
		results:     hud.NewResultsPanel(),
		camera:      ui.NewCamera(),
		startedAt:   time.Now(),
	}
	g.boardView = ui.NewBoardView(g.camera, 210, 190)
	g.activeScene = &SceneConnect{game: g}
	return g
}

// Close releases GPU resources owned by Game.
func (g *Game) Close() {
	if g.shaders != nil {
		g.shaders.Deallocate()
		g.shaders = nil
	}
}

// Update is called every tick (60 TPS by default).
// It transitions between scenes based on connection and game-end state,
// then delegates per-tick logic to the active scene.
func (g *Game) Update() error {
	g.updateScene()
	g.stepSceneFade()
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
		g.drawSceneFade(screen)
		return
	}
	// Fallback when activeScene is unset (e.g. in unit tests that build Game directly).
	g.drawGameContent(screen)
}

// drawGameContent renders the in-game view. Called from SceneGame.Draw and
// from the Draw fallback path when no active scene is set.
func (g *Game) drawGameContent(screen *ebiten.Image) {
	g.frameCount++
	g.ensureShaders()

	screen.Fill(g.theme.Background)

	gs, playerID, connected := g.state.Snapshot()
	g.updateUXWidgets(gs, connected)
	g.ensureProceduralSeed(gs)
	g.drawProceduralAtmosphere(screen)

	// Layer 0 — Board: location tiles.
	g.enqueueBoard(gs)
	g.drawBoardOverlay(screen, gs)

	// Layer 1 — Tokens: investigator tokens on their location tiles.
	g.enqueueTokens(gs, playerID)

	// Layer 2 — Effects: doom bar segment.
	g.enqueueDoomEffect(gs)

	// Flush layers 0-2 and 4 (no animation sprites yet) before overlaying text UI.
	g.renderer.Flush(screen)

	// Layer 3 — UI: text overlays drawn directly on screen after sprite flush.
	g.drawConnectionBanner(screen, connected)
	g.drawStateBanner(screen)
	g.drawDoomCounter(screen, gs)
	g.drawPlayerPanel(screen, gs, playerID)
	g.drawResultsPanel(screen)
	g.drawEventLog(screen)
	g.drawInputHints(screen, gs, playerID)
	g.drawCameraControls(screen)
	g.drawOnboarding(screen)
	g.applyPostProcess(screen, gs.Doom)
}

func (g *Game) ensureShaders() {
	if g.shaders != nil {
		return
	}
	ss, err := render.NewShaderSet()
	if err == nil {
		g.shaders = ss
		return
	}
	log.Printf("shader compilation failed (vignette disabled): %v", err)
	// Assign a sentinel non-nil value so we don't retry every frame.
	g.shaders = &render.ShaderSet{}
}

func (g *Game) stepSceneFade() {
	if g.sceneFade > 0 {
		g.sceneFade--
	}
}

func (g *Game) startSceneFade() {
	g.sceneFade = 12
}

func (g *Game) drawSceneFade(screen *ebiten.Image) {
	if g.sceneFade <= 0 {
		return
	}
	alpha := uint8(float64(g.sceneFade) / 12.0 * 210.0)
	ebitenutil.DrawRect(screen, 0, 0, screenWidth, screenHeight, color.RGBA{R: 8, G: 10, B: 16, A: alpha})
}

func (g *Game) applyPostProcess(screen *ebiten.Image, doom int) {
	if g.effects.EnableFog {
		fog := render.DoomReactiveIntensity(g.effects.FogOpacity, doom, 12)
		render.DrawFogOverlay(screen, g.shaders, fog)
	}
	if g.effects.EnableGlow {
		glow := render.DoomReactiveIntensity(g.effects.GlowIntensity, doom, 12)
		render.DrawGlowOverlay(screen, g.shaders, glow, float32(time.Since(g.startedAt).Seconds()))
	}
	if doom > 0 {
		render.DrawDoomVignette(screen, g.shaders, float32(doom)/12)
	}
}

func (g *Game) ensureProceduralSeed(gs ebclient.GameState) {
	seed := ui.SeedFromGameState(gs)
	if g.procedural == nil || g.procSeed != seed {
		g.procSeed = seed
		g.procedural = ui.NewProceduralGenerator(seed)
		g.procFrame = ui.ProceduralFrame{}
		g.procAtFrame = -1
	}
}

func (g *Game) drawProceduralAtmosphere(screen *ebiten.Image) {
	if !g.effects.EnableAmbient || g.procedural == nil {
		return
	}
	step := g.effects.ProceduralStep
	if step < 1 {
		step = 1
	}
	if len(g.procFrame.Rects) == 0 || g.frameCount%int64(step) == 0 {
		g.procFrame = g.procedural.Generate(g.effects, screenWidth, screenHeight, g.frameCount)
		g.procAtFrame = g.frameCount
	}
	for _, r := range g.procFrame.Rects {
		col := g.theme.Ambient
		switch r.Layer {
		case ui.LayerFog:
			col = g.theme.FogTint
		case ui.LayerGrain:
			col = g.theme.GrainTint
		case ui.LayerSigil:
			col = g.theme.SigilTint
		}
		if col.A == 0 {
			col.A = r.Alpha
		} else {
			col.A = min8(col.A, r.Alpha)
		}
		// Keep low-doom visuals readable while increasing atmosphere under pressure.
		col.A = g.doomScaledAlpha(col.A)
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, col)
	}
}

func (g *Game) doomScaledAlpha(alpha uint8) uint8 {
	gs, _, _ := g.state.Snapshot()
	scaled := render.DoomReactiveIntensity(float32(alpha), gs.Doom, 12)
	if scaled < 0 {
		scaled = 0
	}
	if scaled > 255 {
		scaled = 255
	}
	return uint8(scaled)
}

func (g *Game) updateUXWidgets(gs ebclient.GameState, connected bool) {
	if g.stateBanner != nil {
		if connected {
			g.stateBanner.SetConnectionState(ui.Connected)
			g.stateBanner.SetSyncStatus(ui.Synchronized)
		} else {
			g.stateBanner.SetConnectionState(ui.Reconnecting)
			g.stateBanner.SetSyncStatus(ui.Syncing)
		}
	}

	if g.results == nil {
		return
	}
	update, dice := g.state.LatestEventsSnapshot()
	if update == nil {
		return
	}
	key := update.Timestamp.String() + ":" + update.Event + ":" + update.PlayerID
	if key == g.lastOutcome {
		return
	}
	g.lastOutcome = key
	outcome := &hud.ActionOutcome{
		PlayerID:    update.PlayerID,
		PlayerName:  g.playerDisplayName(gs, update.PlayerID),
		ActionType:  update.Event,
		Successful:  update.Result != "fail",
		Description: update.Result,
		ResourceDelta: feedback.ResourceDelta{
			HealthDelta: update.ResourceDelta.Health,
			SanityDelta: update.ResourceDelta.Sanity,
			ClueDelta:   update.ResourceDelta.Clues,
			DoomDelta:   update.DoomDelta,
		},
	}
	if dice != nil {
		results := make([]string, 0, len(dice.Results))
		for _, r := range dice.Results {
			results = append(results, string(r))
		}
		required := 0
		switch update.Event {
		case "investigate":
			required = 2
		case "ward", "closegate":
			required = 3
		}
		outcome.DiceRoll = &hud.DiceRollResult{
			Dice:     results,
			Required: required,
			Achieved: dice.Successes,
			Passed:   dice.Success,
		}
	}
	g.results.DisplayOutcome(outcome)
}

func (g *Game) drawStateBanner(screen *ebiten.Image) {
	if g.stateBanner == nil || !g.stateBanner.BannerVisible() {
		return
	}
	ebitenutil.DrawRect(screen, 0, 0, screenWidth, 20, color.RGBA{R: 10, G: 25, B: 40, A: 220})
	drawUIText(screen, g.stateBanner.BannerText(), 8, 6, color.White)
}

func (g *Game) drawResultsPanel(screen *ebiten.Image) {
	if g.results == nil || !g.results.IsVisible() {
		return
	}
	ebitenutil.DrawRect(screen, 8, 24, 420, 78, color.RGBA{R: 30, G: 30, B: 40, A: 220})

	// Draw outcome, resource delta, doom change, and dice text with wrapping instead of truncation.
	y := 30
	y = drawWrappedText(screen, g.results.OutcomeText(), 400, 14, y, color.White) + 3
	y = drawWrappedText(screen, g.results.ResourceDeltaText(), 400, 14, y, color.RGBA{R: 210, G: 230, B: 255, A: 255}) + 2
	y = drawWrappedText(screen, g.results.DoomChangeText(), 400, 14, y, color.RGBA{R: 255, G: 210, B: 180, A: 255}) + 2
	drawWrappedText(screen, g.results.DiceText(), 400, 14, y, color.RGBA{R: 220, G: 240, B: 255, A: 255})
}

func (g *Game) drawOnboarding(screen *ebiten.Image) {
	if g.onboarding == nil || !g.onboarding.IsActive() {
		return
	}
	step := g.onboarding.CurrentStep()
	if step == nil {
		return
	}
	controls := newOnboardingControls()
	ebitenutil.DrawRect(screen, 120, 90, 560, 96, color.RGBA{R: 12, G: 12, B: 22, A: 240})

	// Draw title and description with wrapping for better readability.
	y := 102
	y = drawWrappedText(screen, step.Title, 540, 136, y, color.RGBA{R: 230, G: 230, B: 255, A: 255}) + 3
	y = drawWrappedText(screen, step.Description, 540, 136, y, color.White) + 1
	drawUIText(screen, "Use NEXT/SKIP buttons or ENTER/H shortcuts", 136, y, color.RGBA{R: 200, G: 200, B: 220, A: 255})

	nextFill := color.RGBA{R: 58, G: 78, B: 114, A: 245}
	nextBorder := color.RGBA{R: 210, G: 225, B: 255, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.next.Min.X), float64(controls.next.Min.Y), float64(controls.next.Dx()), float64(controls.next.Dy()), nextFill)
	ebitenutil.DrawRect(screen, float64(controls.next.Min.X), float64(controls.next.Min.Y), float64(controls.next.Dx()), 2, nextBorder)
	ebitenutil.DrawRect(screen, float64(controls.next.Min.X), float64(controls.next.Max.Y-2), float64(controls.next.Dx()), 2, nextBorder)
	ebitenutil.DrawRect(screen, float64(controls.next.Min.X), float64(controls.next.Min.Y), 2, float64(controls.next.Dy()), nextBorder)
	ebitenutil.DrawRect(screen, float64(controls.next.Max.X-2), float64(controls.next.Min.Y), 2, float64(controls.next.Dy()), nextBorder)
	drawUIText(screen, "NEXT", controls.next.Min.X+36, controls.next.Min.Y+8, color.White)

	skipFill := color.RGBA{R: 78, G: 46, B: 48, A: 245}
	skipBorder := color.RGBA{R: 245, G: 195, B: 195, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.skip.Min.X), float64(controls.skip.Min.Y), float64(controls.skip.Dx()), float64(controls.skip.Dy()), skipFill)
	ebitenutil.DrawRect(screen, float64(controls.skip.Min.X), float64(controls.skip.Min.Y), float64(controls.skip.Dx()), 2, skipBorder)
	ebitenutil.DrawRect(screen, float64(controls.skip.Min.X), float64(controls.skip.Max.Y-2), float64(controls.skip.Dx()), 2, skipBorder)
	ebitenutil.DrawRect(screen, float64(controls.skip.Min.X), float64(controls.skip.Min.Y), 2, float64(controls.skip.Dy()), skipBorder)
	ebitenutil.DrawRect(screen, float64(controls.skip.Max.X-2), float64(controls.skip.Min.Y), 2, float64(controls.skip.Dy()), skipBorder)
	drawUIText(screen, "SKIP TUTORIAL", controls.skip.Min.X+12, controls.skip.Min.Y+8, color.White)
}

func (g *Game) drawCameraControls(screen *ebiten.Image) {
	if g.camera == nil {
		return
	}
	controls := newCameraControls()

	leftFill := color.RGBA{R: 34, G: 38, B: 52, A: 245}
	leftBorder := color.RGBA{R: 156, G: 176, B: 218, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.left.Min.X), float64(controls.left.Min.Y), float64(controls.left.Dx()), float64(controls.left.Dy()), leftFill)
	ebitenutil.DrawRect(screen, float64(controls.left.Min.X), float64(controls.left.Min.Y), float64(controls.left.Dx()), 2, leftBorder)
	ebitenutil.DrawRect(screen, float64(controls.left.Min.X), float64(controls.left.Max.Y-2), float64(controls.left.Dx()), 2, leftBorder)
	ebitenutil.DrawRect(screen, float64(controls.left.Min.X), float64(controls.left.Min.Y), 2, float64(controls.left.Dy()), leftBorder)
	ebitenutil.DrawRect(screen, float64(controls.left.Max.X-2), float64(controls.left.Min.Y), 2, float64(controls.left.Dy()), leftBorder)
	drawUIText(screen, "[", controls.left.Min.X+34, controls.left.Min.Y+6, color.White)

	rightFill := color.RGBA{R: 34, G: 38, B: 52, A: 245}
	rightBorder := color.RGBA{R: 156, G: 176, B: 218, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.right.Min.X), float64(controls.right.Min.Y), float64(controls.right.Dx()), float64(controls.right.Dy()), rightFill)
	ebitenutil.DrawRect(screen, float64(controls.right.Min.X), float64(controls.right.Min.Y), float64(controls.right.Dx()), 2, rightBorder)
	ebitenutil.DrawRect(screen, float64(controls.right.Min.X), float64(controls.right.Max.Y-2), float64(controls.right.Dx()), 2, rightBorder)
	ebitenutil.DrawRect(screen, float64(controls.right.Min.X), float64(controls.right.Min.Y), 2, float64(controls.right.Dy()), rightBorder)
	ebitenutil.DrawRect(screen, float64(controls.right.Max.X-2), float64(controls.right.Min.Y), 2, float64(controls.right.Dy()), rightBorder)
	drawUIText(screen, "]", controls.right.Min.X+34, controls.right.Min.Y+6, color.White)

	toggleFill := color.RGBA{R: 40, G: 58, B: 80, A: 245}
	toggleBorder := color.RGBA{R: 188, G: 214, B: 255, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.toggle.Min.X), float64(controls.toggle.Min.Y), float64(controls.toggle.Dx()), float64(controls.toggle.Dy()), toggleFill)
	ebitenutil.DrawRect(screen, float64(controls.toggle.Min.X), float64(controls.toggle.Min.Y), float64(controls.toggle.Dx()), 2, toggleBorder)
	ebitenutil.DrawRect(screen, float64(controls.toggle.Min.X), float64(controls.toggle.Max.Y-2), float64(controls.toggle.Dx()), 2, toggleBorder)
	ebitenutil.DrawRect(screen, float64(controls.toggle.Min.X), float64(controls.toggle.Min.Y), 2, float64(controls.toggle.Dy()), toggleBorder)
	ebitenutil.DrawRect(screen, float64(controls.toggle.Max.X-2), float64(controls.toggle.Min.Y), 2, float64(controls.toggle.Dy()), toggleBorder)
	drawUIText(screen, "Toggle View (V)", controls.toggle.Min.X+18, controls.toggle.Min.Y+6, color.White)
}

// enqueueBoard adds one board-layer draw command per location.
func (g *Game) enqueueBoard(gs ebclient.GameState) {
	for loc, rect := range locationRects {
		px, py, scale := g.projectPoint(float64(rect.x), float64(rect.y))
		g.renderer.Enqueue(render.LayerBoard, render.DrawCmd{
			Sprite: render.LocationSpriteID(string(loc)),
			X:      px,
			Y:      py,
			ScaleX: scale,
			ScaleY: scale,
		})
		_ = gs // board layout is static; gs unused here
	}
}

// drawBoardOverlay adds readable location labels and movement cues on top of
// the board tiles. This keeps the four-neighbourhood map understandable even
// when placeholder art is still in use.
func (g *Game) drawBoardOverlay(screen *ebiten.Image, gs ebclient.GameState) {
	currentPlayer, ok := gs.Players[gs.CurrentPlayer]
	if !ok {
		g.drawDistrictGuides(screen)
		g.drawLocationLabels(screen)
		return
	}

	currentLocation := currentPlayer.Location
	legalMoves := boardAdjacency[currentLocation]
	legalSet := make(map[ebclient.Location]struct{}, len(legalMoves))
	for _, loc := range legalMoves {
		legalSet[loc] = struct{}{}
	}

	centers := make(map[ebclient.Location][2]float64, len(locationRects))
	for loc, rect := range locationRects {
		cx, cy, _ := g.projectPoint(float64(rect.x+rect.w/2), float64(rect.y+rect.h/2))
		centers[loc] = [2]float64{cx, cy}
	}

	g.drawDistrictGuides(screen)

	if currentCenter, ok := centers[currentLocation]; ok {
		for _, target := range legalMoves {
			if targetCenter, ok := centers[target]; ok {
				ebitenutil.DrawLine(screen, currentCenter[0], currentCenter[1], targetCenter[0], targetCenter[1], color.RGBA{R: 110, G: 230, B: 255, A: 160})
			}
		}
	}

	for loc, rect := range locationRects {
		px, py, scale := g.projectPoint(float64(rect.x), float64(rect.y))
		width := float64(rect.w) * scale
		height := float64(rect.h) * scale
		base := locationColours[loc]
		border := color.RGBA{R: 225, G: 226, B: 235, A: 165}
		fill := withAlpha(blendRGBA(base, color.RGBA{R: 12, G: 14, B: 20, A: 255}, 0.42), 155)
		note := "locked"
		switch {
		case loc == currentLocation:
			border = color.RGBA{R: 252, G: 220, B: 104, A: 250}
			fill = withAlpha(blendRGBA(base, color.RGBA{R: 190, G: 145, B: 44, A: 255}, 0.45), 190)
			note = "current"
		case func() bool { _, ok := legalSet[loc]; return ok }():
			border = color.RGBA{R: 98, G: 234, B: 255, A: 238}
			fill = withAlpha(blendRGBA(base, color.RGBA{R: 44, G: 130, B: 176, A: 255}, 0.40), 180)
			note = "move"
		}

		ebitenutil.DrawRect(screen, px, py, width, height, fill)
		drawTileBorder(screen, px, py, width, height, border)

		noteW := float64(textWidth(note) + 6)
		noteBg := color.RGBA{R: 8, G: 10, B: 16, A: 210}
		if note == "move" {
			noteBg = color.RGBA{R: 9, G: 40, B: 53, A: 225}
		}
		if note == "current" {
			noteBg = color.RGBA{R: 74, G: 55, B: 12, A: 230}
		}
		ebitenutil.DrawRect(screen, px+width-noteW-4, py+4, noteW, 14, noteBg)
		drawUIText(screen, strings.ToUpper(note), int(px+width-noteW), int(py+6), color.RGBA{R: 235, G: 240, B: 250, A: 255})

		labelX, labelY := ui.ProjectLabelPosition(float64(rect.x), float64(rect.y), float64(rect.w), float64(rect.h), g.boardView)
		label := string(loc)
		labelW := float64(textWidth(label) + 8)
		ebitenutil.DrawRect(screen, labelX-4, labelY-2, labelW, 16, color.RGBA{R: 8, G: 8, B: 16, A: 214})
		drawUIText(screen, label, int(labelX), int(labelY), color.White)

		gatesAtLoc := g.countGatesAt(gs, loc)
		enemiesAtLoc := g.countEnemiesAt(gs, loc)
		playersAtLoc := g.countPlayersAt(gs, loc)
		badgeX := px + 6
		badgeY := py + height - 18
		if playersAtLoc > 0 {
			drawEntityBadge(screen, badgeX, badgeY, "P:"+strconv.Itoa(playersAtLoc), color.RGBA{R: 230, G: 226, B: 135, A: 230})
			badgeX += 34
		}
		if gatesAtLoc > 0 {
			drawEntityBadge(screen, badgeX, badgeY, "G:"+strconv.Itoa(gatesAtLoc), color.RGBA{R: 146, G: 90, B: 211, A: 230})
			badgeX += 34
		}
		if enemiesAtLoc > 0 {
			drawEntityBadge(screen, badgeX, badgeY, "E:"+strconv.Itoa(enemiesAtLoc), color.RGBA{R: 214, G: 87, B: 87, A: 230})
		}
	}

	if currentLocation != "" {
		movesText := "Legal moves: none"
		if len(legalMoves) > 0 {
			movesText = "Legal moves: " + string(legalMoves[0])
			for i := 1; i < len(legalMoves); i++ {
				movesText += ", " + string(legalMoves[i])
			}
		}
		ebitenutil.DrawRect(screen, 32, 540, 320, 22, color.RGBA{R: 8, G: 10, B: 18, A: 200})
		drawUIText(screen, trimToWidth(movesText, 300), 40, 544, color.RGBA{R: 220, G: 240, B: 255, A: 255})
	}
}

func (g *Game) drawDistrictGuides(screen *ebiten.Image) {
	x, y, scale := g.projectPoint(200, 48)
	verticalW := 12.0 * scale
	horizontalW := 12.0 * scale
	boardH := 292.0 * scale
	boardW := 352.0 * scale
	ebitenutil.DrawRect(screen, x, y, verticalW, boardH, color.RGBA{R: 6, G: 8, B: 12, A: 188})
	ebitenutil.DrawRect(screen, x-(160.0*scale), y+(106.0*scale), boardW, horizontalW, color.RGBA{R: 6, G: 8, B: 12, A: 188})
}

// drawLocationLabels renders a readable label for each neighborhood tile.
func (g *Game) drawLocationLabels(screen *ebiten.Image) {
	for loc, rect := range locationRects {
		labelX, labelY := ui.ProjectLabelPosition(float64(rect.x), float64(rect.y), float64(rect.w), float64(rect.h), g.boardView)
		label := string(loc)
		labelW := float64(textWidth(label) + 8)
		ebitenutil.DrawRect(screen, labelX-4, labelY-2, labelW, 16, color.RGBA{R: 8, G: 8, B: 16, A: 200})
		drawUIText(screen, label, int(labelX), int(labelY), color.White)
	}
}

// drawTileBorder draws a simple rectangular outline using four thin fills.
func drawTileBorder(screen *ebiten.Image, x, y, w, h float64, clr color.RGBA) {
	const thickness = 2.0
	ebitenutil.DrawRect(screen, x, y, w, thickness, clr)
	ebitenutil.DrawRect(screen, x, y+h-thickness, w, thickness, clr)
	ebitenutil.DrawRect(screen, x, y, thickness, h, clr)
	ebitenutil.DrawRect(screen, x+w-thickness, y, thickness, h, clr)
}

func blendRGBA(a, b color.RGBA, factor float64) color.RGBA {
	if factor < 0 {
		factor = 0
	}
	if factor > 1 {
		factor = 1
	}
	inv := 1.0 - factor
	return color.RGBA{
		R: uint8(float64(a.R)*inv + float64(b.R)*factor),
		G: uint8(float64(a.G)*inv + float64(b.G)*factor),
		B: uint8(float64(a.B)*inv + float64(b.B)*factor),
		A: uint8(float64(a.A)*inv + float64(b.A)*factor),
	}
}

func withAlpha(col color.RGBA, alpha uint8) color.RGBA {
	col.A = alpha
	return col
}

func drawEntityBadge(screen *ebiten.Image, x, y float64, label string, tint color.RGBA) {
	bg := color.RGBA{R: 10, G: 12, B: 16, A: 210}
	ebitenutil.DrawRect(screen, x, y, 30, 12, bg)
	ebitenutil.DrawRect(screen, x, y, 30, 2, tint)
	ebitenutil.DrawRect(screen, x, y+10, 30, 2, tint)
	drawUIText(screen, label, int(x+2), int(y+2), color.RGBA{R: 245, G: 245, B: 245, A: 255})
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
		px, py, scale := g.projectPoint(float64(rect.x+4+offset*14), float64(rect.y+rect.h-16))
		g.renderer.Enqueue(render.LayerTokens, render.DrawCmd{
			Sprite: render.SpritePlayerToken,
			X:      px,
			Y:      py,
			ScaleX: scale,
			ScaleY: scale,
			Tint:   col,
		})
	}
}

func (g *Game) projectPoint(x, y float64) (float64, float64, float64) {
	if g.boardView == nil {
		return x, y, 1.0
	}
	return g.boardView.ProjectPoint(x, y)
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
	drawUIText(screen, "[ Connecting to server... ]", screenWidth/2-90, 8, color.White)
}

// drawDoomCounter renders the global doom track (0–12) on the right side.
func (g *Game) drawDoomCounter(screen *ebiten.Image, gs ebclient.GameState) {
	ebitenutil.DrawRect(screen, float64(rightPanelX()-10), 42, 386, 56, color.RGBA{R: 12, G: 14, B: 24, A: 214})
	label := g.doomLabel(gs.Doom)
	drawUIText(screen, label, rightPanelX(), 60, color.White)

	// Draw a simple bar.
	barW := 200.0
	filled := float64(gs.Doom) / 12.0 * barW
	bg, fg := g.doomBarColors()
	ebitenutil.DrawRect(screen, float64(rightPanelX()), 76, barW, 14, bg)
	ebitenutil.DrawRect(screen, float64(rightPanelX()), 76, filled, 14, fg)
}

func (g *Game) doomBarColors() (bg, fg color.RGBA) {
	bg = color.RGBA{R: 60, G: 60, B: 60, A: 255}
	fg = color.RGBA{R: 200, G: 40, B: 40, A: 255}
	if g.tokens == nil {
		return bg, fg
	}
	bg = ui.ResolveThemePack(g.tokens).FogTint
	doomToken := g.tokens.GetColor("color-doom")
	r, gg, b, a := doomToken.RGBA()
	fg = color.RGBA{R: uint8(r >> 8), G: uint8(gg >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
	return bg, fg
}

// drawPlayerPanel renders resource levels for all players on the right side.
func (g *Game) drawPlayerPanel(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	panelX := rightPanelX() - 10
	panelY := 110
	panelW := 386
	ebitenutil.DrawRect(screen, float64(panelX), float64(panelY), float64(panelW), 188, color.RGBA{R: 12, G: 14, B: 24, A: 214})
	drawUIText(screen, "Turn Overview", rightPanelX(), panelY+8, color.RGBA{R: 238, G: 240, B: 252, A: 255})
	drawUIText(screen, g.phaseLabel(gs.GamePhase), rightPanelX()+120, panelY+8, color.RGBA{R: 182, G: 212, B: 255, A: 255})

	if len(gs.TurnOrder) == 0 {
		drawUIText(screen, "Waiting for players...", rightPanelX(), panelY+28, color.White)
		return
	}

	y := panelY + 26
	for i, pid := range gs.TurnOrder {
		p, ok := gs.Players[pid]
		if !ok {
			continue
		}
		y += g.drawPlayerPanelRow(screen, y, pid, p, myID, gs.CurrentPlayer)
		if i >= 5 {
			break
		}
	}
}

func (g *Game) drawPlayerPanelRow(screen *ebiten.Image, y int, pid string, p *ebclient.Player, myID, currentPlayer string) int {
	const cardH = 24
	fill, border := playerRowStyle(pid == currentPlayer)
	ebitenutil.DrawRect(screen, float64(rightPanelX()-4), float64(y), 374, float64(cardH), fill)
	drawTileBorder(screen, float64(rightPanelX()-4), float64(y), 374, float64(cardH), border)

	name := trimToWidth(g.playerDisplayNameFromPlayer(pid, p), 150)
	if pid == myID {
		name += " (you)"
	}
	turnGlyph := " "
	if pid == currentPlayer {
		turnGlyph = g.iconLabel(ui.IconTurn, ">")
	}
	drawUIText(screen, turnGlyph+" "+name, rightPanelX(), y+6, color.White)

	pillX := rightPanelX() + 162
	pillX = g.drawResourcePill(screen, pillX, y+4, g.iconLabel(ui.IconHealth, "HP"), p.Resources.Health, color.RGBA{R: 200, G: 82, B: 82, A: 255})
	pillX = g.drawResourcePill(screen, pillX, y+4, g.iconLabel(ui.IconSanity, "SN"), p.Resources.Sanity, color.RGBA{R: 90, G: 160, B: 232, A: 255})
	pillX = g.drawResourcePill(screen, pillX, y+4, g.iconLabel(ui.IconClues, "CL"), p.Resources.Clues, color.RGBA{R: 86, G: 194, B: 122, A: 255})
	g.drawResourcePill(screen, pillX, y+4, "ACT", p.ActionsRemaining, color.RGBA{R: 228, G: 197, B: 102, A: 255})
return cardH + 5
}

func playerRowStyle(isCurrent bool) (color.RGBA, color.RGBA) {
	if isCurrent {
		return color.RGBA{R: 58, G: 70, B: 110, A: 228}, color.RGBA{R: 220, G: 232, B: 255, A: 250}
	}
	return color.RGBA{R: 24, G: 28, B: 38, A: 210}, color.RGBA{R: 98, G: 112, B: 148, A: 230}
}

func (g *Game) drawResourcePill(screen *ebiten.Image, x, y int, icon string, value int, accent color.RGBA) int {
	if icon == "" {
		icon = "?"
	}
	label := icon + ":" + strconv.Itoa(value)
	width := textWidth(label) + 12
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(width), 14, color.RGBA{R: 10, G: 12, B: 20, A: 220})
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(width), 2, accent)
	drawUIText(screen, label, x+4, y+3, color.RGBA{R: 245, G: 247, B: 252, A: 255})
	return x + width + 4
}

func (g *Game) iconLabel(id ui.IconID, fallback string) string {
	if g.icons == nil {
		return fallback
	}
	if label := g.icons.Get(id); label != "" {
		return label
	}
	return fallback
}

func (g *Game) playerPanelLabel(pid, currentPlayer, myID string, p *ebclient.Player) string {
	marker := " "
	if pid == currentPlayer {
		marker = ">"
		if g.icons != nil {
			marker = g.icons.Get(ui.IconTurn)
		}
	}
	me := ""
	if pid == myID {
		me = " (you)"
	}
	name := g.playerDisplayNameFromPlayer(pid, p)
	hp, sn, cl := "HP", "SN", "CL"
	if g.icons != nil {
		hp = g.icons.Get(ui.IconHealth)
		sn = g.icons.Get(ui.IconSanity)
		cl = g.icons.Get(ui.IconClues)
	}
	return marker + " " + name + " (" + pid + ")" + me + "  " + hp + ":" + strconv.Itoa(p.Resources.Health) +
		" " + sn + ":" + strconv.Itoa(p.Resources.Sanity) +
		" " + cl + ":" + strconv.Itoa(p.Resources.Clues) +
		" ACT:" + strconv.Itoa(p.ActionsRemaining)
}

// drawEventLog renders the last 8 events in the lower-right corner.
func (g *Game) drawEventLog(screen *ebiten.Image) {
	entries := g.state.EventLogSnapshot()
	y := bottomPanelY()
	drawUIText(screen, "-- Event Log --", rightPanelX(), y, color.White)
	y += 12

	// Show the last 8 entries.
	start := 0
	if len(entries) > 8 {
		start = len(entries) - 8
	}
	for _, e := range entries[start:] {
		drawUIText(screen, trimToWidth(e.Text, 360), rightPanelX(), y, color.RGBA{R: 220, G: 220, B: 220, A: 255})
		y += 12
	}
}

// drawInputHints renders a state-driven action panel in the lower-left corner.
func (g *Game) drawInputHints(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	panelX := 10
	panelY := bottomPanelY()
	panelW := 360
	panelH := 130
	ebitenutil.DrawRect(screen, float64(panelX-2), float64(panelY-2), float64(panelW), float64(panelH), color.RGBA{R: 10, G: 12, B: 20, A: 210})
	g.drawMoveChips(screen, gs, myID, panelX, panelY-28)
	g.drawVisibleActionButtons(screen, gs, myID)
	g.drawActionPanelSummary(screen, gs, myID, panelX, panelY)
	g.drawAvailableActionList(screen, gs, myID, panelX, panelY+92)
	g.drawEndBanner(screen, gs)
}

func (g *Game) drawMoveChips(screen *ebiten.Image, gs ebclient.GameState, myID string, panelX, panelY int) {
	moves := legalMoveChips(gs, myID, panelX, panelY)
	if len(moves) == 0 {
		drawUIText(screen, "Move chips: unavailable", panelX, panelY, color.RGBA{R: 200, G: 220, B: 255, A: 255})
		return
	}
	hovered := g.state.HoveredActionHint()
	focused := g.state.FocusedActionHint()
	pressed := g.state.PressedActionHint()
	drawUIText(screen, "Move chips", panelX, panelY-12, color.White)
	for _, move := range moves {
		fill, border := moveChipStyle(string(move.target), focused, hovered, pressed)
		ebitenutil.DrawRect(screen, float64(move.rect.Min.X), float64(move.rect.Min.Y), float64(move.rect.Dx()), float64(move.rect.Dy()), fill)
		ebitenutil.DrawRect(screen, float64(move.rect.Min.X), float64(move.rect.Min.Y), float64(move.rect.Dx()), 2, border)
		ebitenutil.DrawRect(screen, float64(move.rect.Min.X), float64(move.rect.Max.Y-2), float64(move.rect.Dx()), 2, border)
		ebitenutil.DrawRect(screen, float64(move.rect.Min.X), float64(move.rect.Min.Y), 2, float64(move.rect.Dy()), border)
		ebitenutil.DrawRect(screen, float64(move.rect.Max.X-2), float64(move.rect.Min.Y), 2, float64(move.rect.Dy()), border)
		drawUIText(screen, string(move.target), move.rect.Min.X+10, move.rect.Min.Y+5, color.White)
		drawMoveShortcutHint(screen, move)
	}
}

func drawMoveShortcutHint(screen *ebiten.Image, move moveChip) {
	key := moveShortcutHints[move.target]
	if key == "" {
		return
	}
	drawUIText(screen, "["+key+"]", move.rect.Max.X-30, move.rect.Min.Y+5, color.RGBA{R: 216, G: 228, B: 255, A: 255})
}

func moveChipStyle(target, focused, hovered, pressed string) (color.RGBA, color.RGBA) {
	fill := color.RGBA{R: 28, G: 36, B: 52, A: 245}
	border := color.RGBA{R: 120, G: 180, B: 230, A: 255}
	if strings.EqualFold(target, focused) {
		fill = color.RGBA{R: 42, G: 52, B: 82, A: 245}
		border = color.RGBA{R: 168, G: 206, B: 255, A: 255}
	}
	if strings.EqualFold(target, hovered) {
		fill = color.RGBA{R: 46, G: 64, B: 92, A: 245}
		border = color.RGBA{R: 186, G: 222, B: 255, A: 255}
	}
	if strings.EqualFold(target, pressed) {
		fill = color.RGBA{R: 60, G: 86, B: 118, A: 245}
		border = color.RGBA{R: 220, G: 238, B: 255, A: 255}
	}
	return fill, border
}

func (g *Game) drawVisibleActionButtons(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	actions := g.availableActions(gs, myID)
	availability := make(map[string]actionAvailability, len(actions))
	for _, action := range actions {
		availability[actionLookupKey(action.Name)] = action
	}
	hovered := g.state.HoveredActionHint()
	focused := g.state.FocusedActionHint()
	pressed := g.state.PressedActionHint()
	for row, rowActions := range actionGridRows() {
		for col, actionName := range rowActions {
			if actionName == "" {
				continue
			}
			action := availability[actionLookupKey(actionName)]
			rect := actionGridRect(row, col)
			fill, border, labelColor := actionButtonStyle(action.Available, actionName, focused, hovered, pressed)
			ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), fill)
			ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), 2, border)
			ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Max.Y-2), float64(rect.Dx()), 2, border)
			ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), 2, float64(rect.Dy()), border)
			ebitenutil.DrawRect(screen, float64(rect.Max.X-2), float64(rect.Min.Y), 2, float64(rect.Dy()), border)
			drawUIText(screen, strings.Title(strings.ReplaceAll(actionName, "closegate", "close gate")), rect.Min.X+10, rect.Min.Y+5, labelColor)
			if key := actionShortcutHints[actionName]; key != "" {
				drawUIText(screen, "["+key+"]", rect.Max.X-30, rect.Min.Y+5, color.RGBA{R: 216, G: 228, B: 255, A: 255})
			}
			if action.Detail != "" {
				drawUIText(screen, trimToWidth(action.Detail, rect.Dx()-16), rect.Min.X+10, rect.Min.Y+17, color.RGBA{R: 200, G: 220, B: 235, A: 255})
			}
		}
	}
}

func actionButtonStyle(available bool, actionName, focused, hovered, pressed string) (color.RGBA, color.RGBA, color.RGBA) {
	fill := color.RGBA{R: 24, G: 28, B: 38, A: 245}
	border := color.RGBA{R: 110, G: 130, B: 165, A: 255}
	labelColor := color.RGBA{R: 220, G: 220, B: 235, A: 255}
	if available {
		fill = color.RGBA{R: 28, G: 62, B: 50, A: 245}
		border = color.RGBA{R: 170, G: 240, B: 190, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if strings.EqualFold(actionName, focused) {
		fill = color.RGBA{R: 52, G: 54, B: 82, A: 245}
		border = color.RGBA{R: 190, G: 202, B: 255, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if strings.EqualFold(actionName, hovered) {
		fill = color.RGBA{R: 58, G: 64, B: 98, A: 245}
		border = color.RGBA{R: 206, G: 220, B: 255, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if strings.EqualFold(actionName, pressed) {
		fill = color.RGBA{R: 78, G: 88, B: 124, A: 245}
		border = color.RGBA{R: 232, G: 236, B: 255, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return fill, border, labelColor
}

func actionLookupKey(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

func (g *Game) drawActionPanelSummary(screen *ebiten.Image, gs ebclient.GameState, myID string, panelX, panelY int) {
	drawUIText(screen, "-- Available Actions --", panelX, panelY, color.White)
	panelY += 12

	isMyTurn := gs.CurrentPlayer == myID && gs.GamePhase == "playing"
	drawUIText(screen, g.turnStatusLabel(gs, myID, isMyTurn), panelX, panelY, color.White)
	panelY += 12

	metrics := g.state.UXMetrics()
	drawUIText(screen, "Actions left: "+strconv.Itoa(g.remainingActions(gs, myID)), panelX, panelY, color.RGBA{R: 180, G: 220, B: 255, A: 255})
	panelY += 12
	drawUIText(screen, g.firstActionStatusText(metrics), panelX, panelY, color.RGBA{R: 180, G: 220, B: 255, A: 255})
	panelY += 12
	drawUIText(screen, "Invalid retries: "+strconv.Itoa(metrics.InvalidActionRetries), panelX, panelY,
		color.RGBA{R: 255, G: 200, B: 170, A: 255})
	panelY += 12
	drawUIText(screen, trimToWidth(g.invalidActionHint(metrics.LastInvalidReason), 350), panelX, panelY,
		color.RGBA{R: 255, G: 220, B: 180, A: 255})
	panelY += 12
	drawUIText(screen, "Focus: "+g.focusHintLabel()+" (Tab/Shift+Tab + Enter)", panelX, panelY,
		color.RGBA{R: 200, G: 220, B: 255, A: 255})
	panelY += 12
	drawUIText(screen, g.cameraStatusText(), panelX, panelY, color.RGBA{R: 200, G: 220, B: 255, A: 255})
}

func (g *Game) focusHintLabel() string {
	focusHint := strings.TrimSpace(g.state.FocusedActionHint())
	if focusHint == "" {
		return "none"
	}
	return focusHint
}

func (g *Game) drawAvailableActionList(screen *ebiten.Image, gs ebclient.GameState, myID string, panelX, panelY int) {
	actions := g.availableActions(gs, myID)
	leftX := panelX
	rightX := panelX + 180
	startY := panelY
	for i, action := range actions {
		x := leftX
		y := startY + (i%6)*10
		if i >= 6 {
			x = rightX
		}
		prefix := "[x]"
		clr := color.RGBA{R: 255, G: 180, B: 160, A: 255}
		if action.Available {
			prefix = "[✓]"
			clr = color.RGBA{R: 160, G: 255, B: 180, A: 255}
		}
		line := prefix + " " + action.Name
		if action.Detail != "" {
			line += " - " + action.Detail
		}
		drawUIText(screen, trimToWidth(line, 165), x, y, clr)
	}
}

type actionAvailability struct {
	Name      string
	Detail    string
	Available bool
}

func (g *Game) availableActions(gs ebclient.GameState, myID string) []actionAvailability {
	actions := make([]actionAvailability, 0, 10)
	turnActive := gs.GamePhase == "playing" && gs.CurrentPlayer == myID
	remaining := g.remainingActions(gs, myID)
	currentLocation := g.playerLocation(gs, myID)
	legalMoves := boardAdjacency[currentLocation]
	coLocated := g.colocatedPlayer(gs, myID)
	openGate := g.hasOpenGateAt(gs, currentLocation)
	hasEnemy := g.hasEnemyAtLocation(gs, currentLocation)
	sanity := g.playerSanity(gs, myID)

	add := func(name, detail string, available bool) {
		actions = append(actions, actionAvailability{Name: name, Detail: detail, Available: available})
	}

	moveDetail := g.moveDetail(legalMoves)
	add("Move", moveDetail, turnActive && remaining > 0)
	add("Gather", "gain resources", turnActive && remaining > 0)
	add("Investigate", "2-success clue test", turnActive && remaining > 0)
	add("Ward", g.wardDetail(turnActive, remaining, sanity), turnActive && remaining > 0 && sanity > 1)
	add("Focus", "gain focus", turnActive && remaining > 0)
	add("Research", "improved clue test", turnActive && remaining > 0)
	add("Trade", g.tradeDetail(turnActive, remaining, coLocated), turnActive && remaining > 0 && coLocated != "")
	add("Component", "archetype ability", turnActive && remaining > 0)
	add("Attack", g.attackDetail(turnActive, remaining, hasEnemy), turnActive && remaining > 0 && hasEnemy)
	add("Evade", g.attackDetail(turnActive, remaining, hasEnemy), turnActive && remaining > 0 && hasEnemy)
	add("Close Gate", g.closeGateDetail(turnActive, remaining, openGate), turnActive && remaining > 0 && openGate)
	add("Encounter", "draw location encounter card", turnActive && remaining > 0)

	return actions
}

func (g *Game) moveDetail(legalMoves []ebclient.Location) string {
	if len(legalMoves) == 0 {
		return "no adjacent location"
	}
	detail := "to " + string(legalMoves[0])
	for i := 1; i < len(legalMoves); i++ {
		detail += ", " + string(legalMoves[i])
	}
	return detail
}

func legalMoveChips(gs ebclient.GameState, myID string, panelX, panelY int) []moveChip {
	current := ebclient.Location("")
	if player, ok := gs.Players[myID]; ok && player != nil {
		current = player.Location
	}
	legalMoves := boardAdjacency[current]
	if len(legalMoves) == 0 {
		return nil
	}
	chips := make([]moveChip, 0, len(legalMoves))
	chipW := 84
	chipH := 22
	gap := 8
	for i, target := range legalMoves {
		x := panelX + i*(chipW+gap)
		chips = append(chips, moveChip{target: target, rect: image.Rect(x, panelY, x+chipW, panelY+chipH)})
	}
	return chips
}

func (g *Game) remainingActions(gs ebclient.GameState, myID string) int {
	if p, ok := gs.Players[myID]; ok {
		return p.ActionsRemaining
	}
	return 0
}

func (g *Game) playerLocation(gs ebclient.GameState, myID string) ebclient.Location {
	if p, ok := gs.Players[myID]; ok {
		return p.Location
	}
	return ""
}

func (g *Game) playerSanity(gs ebclient.GameState, myID string) int {
	if p, ok := gs.Players[myID]; ok {
		return p.Resources.Sanity
	}
	return 0
}

func (g *Game) colocatedPlayer(gs ebclient.GameState, myID string) string {
	me, ok := gs.Players[myID]
	if !ok {
		return ""
	}
	for otherID, other := range gs.Players {
		if otherID != myID && other.Location == me.Location && !other.Defeated {
			return otherID
		}
	}
	return ""
}

func (g *Game) hasEnemyAtLocation(gs ebclient.GameState, loc ebclient.Location) bool {
	for _, enemy := range gs.Enemies {
		if enemy != nil && enemy.Location == loc {
			return true
		}
	}
	return false
}

func (g *Game) hasOpenGateAt(gs ebclient.GameState, loc ebclient.Location) bool {
	for _, gate := range gs.OpenGates {
		if gate.Location == loc {
			return true
		}
	}
	return false
}

func (g *Game) countEnemiesAt(gs ebclient.GameState, loc ebclient.Location) int {
	count := 0
	for _, enemy := range gs.Enemies {
		if enemy != nil && enemy.Location == loc {
			count++
		}
	}
	return count
}

func (g *Game) countGatesAt(gs ebclient.GameState, loc ebclient.Location) int {
	count := 0
	for _, gate := range gs.OpenGates {
		if gate.Location == loc {
			count++
		}
	}
	return count
}

func (g *Game) countPlayersAt(gs ebclient.GameState, loc ebclient.Location) int {
	count := 0
	for _, pid := range gs.TurnOrder {
		p := gs.Players[pid]
		if p != nil && p.Connected && p.Location == loc {
			count++
		}
	}
	return count
}

func (g *Game) wardDetail(turnActive bool, remaining, sanity int) string {
	if !turnActive {
		return "wait for your turn"
	}
	if remaining <= 0 {
		return "no actions left"
	}
	if sanity <= 1 {
		return "need 1 sanity to spend"
	}
	return "costs 1 sanity"
}

func (g *Game) tradeDetail(turnActive bool, remaining int, colocated string) string {
	if !turnActive {
		return "wait for your turn"
	}
	if remaining <= 0 {
		return "no actions left"
	}
	if colocated == "" {
		return "need a co-located ally"
	}
	return "with " + colocated
}

func (g *Game) attackDetail(turnActive bool, remaining int, hasEnemy bool) string {
	if !turnActive {
		return "wait for your turn"
	}
	if remaining <= 0 {
		return "no actions left"
	}
	if !hasEnemy {
		return "need an enemy here"
	}
	return "enemy at location"
}

func (g *Game) closeGateDetail(turnActive bool, remaining int, openGate bool) string {
	if !turnActive {
		return "wait for your turn"
	}
	if remaining <= 0 {
		return "no actions left"
	}
	if !openGate {
		return "need an open gate here"
	}
	return "seal the gate"
}

func (g *Game) invalidActionHint(reason string) string {
	switch reason {
	case "":
		return "Last invalid: none yet"
	case "out-of-turn-or-disconnected":
		return "Last invalid: wait for your turn or reconnect"
	case "trade-no-colocated-player":
		return "Last invalid: trade needs a co-located ally; move to the same location first"
	default:
		return "Last invalid: " + reason
	}
}

func (g *Game) firstActionStatusText(metrics ebclient.UXMetricsSnapshot) string {
	if metrics.HasFirstValidAction {
		return "First valid action: " + metrics.TimeToFirstValidAction.Round(time.Millisecond).String()
	}
	return "First valid action: pending"
}

func (g *Game) cameraStatusText() string {
	if g.camera == nil {
		return ""
	}
	mode := "topdown"
	if g.camera.Mode() == ui.ViewModePseudo3D {
		mode = "pseudo3d"
	}
	return "Camera: " + mode + " dir=" + strconv.Itoa(g.camera.Direction()+1) + "/8"
}

func (g *Game) drawEndBanner(screen *ebiten.Image, gs ebclient.GameState) {
	if gs.WinCondition {
		drawUIText(screen, "* INVESTIGATORS WIN! *", 10, screenHeight-20, color.RGBA{R: 140, G: 255, B: 140, A: 255})
		return
	}
	if gs.LoseCondition {
		drawUIText(screen, "* ANCIENT ONE AWAKENS - YOU LOSE *", 10, screenHeight-20, color.RGBA{R: 255, G: 140, B: 140, A: 255})
	}
}

func rightPanelX() int {
	return screenWidth - 380
}

func bottomPanelY() int {
	return screenHeight - 130
}

func (g *Game) doomLabel(doom int) string {
	if g.uiCache.doomLabel == "" || g.uiCache.doomValue != doom {
		g.uiCache.doomValue = doom
		g.uiCache.doomLabel = "DOOM: " + strconv.Itoa(doom) + " / 12"
	}
	return g.uiCache.doomLabel
}

func (g *Game) phaseLabel(phase string) string {
	if g.uiCache.phaseLabel == "" || g.uiCache.phaseValue != phase {
		g.uiCache.phaseValue = phase
		g.uiCache.phaseLabel = "Phase: " + phase
	}
	return g.uiCache.phaseLabel
}

func (g *Game) turnStatusLabel(gs ebclient.GameState, myID string, isMyTurn bool) string {
	actions := 0
	if p, ok := gs.Players[myID]; ok {
		actions = p.ActionsRemaining
	}
	key := strconv.FormatBool(isMyTurn) + ":" + strconv.Itoa(actions)
	if g.uiCache.statusKey != key {
		g.uiCache.statusKey = key
		if !isMyTurn {
			g.uiCache.statusText = "waiting for your turn"
		} else {
			g.uiCache.statusText = "YOUR TURN (" + strconv.Itoa(actions) + " actions left)"
		}
	}
	return g.uiCache.statusText
}

func (g *Game) playerDisplayName(gs ebclient.GameState, playerID string) string {
	if p, ok := gs.Players[playerID]; ok {
		return g.playerDisplayNameFromPlayer(playerID, p)
	}
	return playerID
}

func (g *Game) playerDisplayNameFromPlayer(playerID string, p *ebclient.Player) string {
	if p == nil {
		return playerID
	}
	if name := strings.TrimSpace(p.DisplayName); name != "" {
		return name
	}
	return playerID
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
