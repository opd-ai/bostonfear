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
	"math"
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

var boardLocationOrder = []ebclient.Location{
	"Downtown",
	"University",
	"Rivertown",
	"Northside",
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
	{R: 238, G: 198, B: 72, A: 255},  // Player 1: gold/yellow
	{R: 92, G: 214, B: 238, A: 255},  // Player 2: cyan/light blue
	{R: 226, G: 104, B: 188, A: 255}, // Player 3: magenta/pink
	{R: 162, G: 220, B: 92, A: 255},  // Player 4: lime/green
	{R: 236, G: 132, B: 96, A: 255},  // Player 5: orange/coral
	{R: 164, G: 132, B: 222, A: 255}, // Player 6: purple/violet
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
	animState   hudAnimationState
	coachTicks  int
}

type playerResourceSnapshot struct {
	health int
	sanity int
	clues  int

	actions int
}

type hudAnimationState struct {
	lastDoom        int
	doomFlashFrames int

	lastCurrentPlayer string
	turnFlashFrames   int

	resourceSnapshot map[string]playerResourceSnapshot
	resourceFlash    map[string]int

	lastOutcomeKey   string
	resultRollFrames int
	resultFadeFrames int

	actionResult      string
	actionResultOK    bool
	actionResultFlash int
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
		animState: hudAnimationState{
			resourceSnapshot: make(map[string]playerResourceSnapshot),
			resourceFlash:    make(map[string]int),
		},
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
	g.advanceHUDAnimations(gs)
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
	g.drawInvestigatorTokens(screen, gs, playerID)

	// Layer 3 — UI: text overlays drawn directly on screen after sprite flush.
	g.drawStatusRail(screen, gs, playerID, connected)
	g.drawConnectionBanner(screen, connected)
	g.drawStateBanner(screen)
	g.drawDoomCounter(screen, gs)
	g.drawResourceRail(screen, gs, playerID)
	g.drawPlayerPanel(screen, gs, playerID)
	g.drawLocationPanel(screen, gs, playerID)
	g.drawResultsPanel(screen)
	g.drawEventLog(screen)
	g.drawInputHints(screen, gs, playerID)
	g.drawCoachMarks(screen, gs, playerID)
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
	g.animState.lastOutcomeKey = key
	g.animState.resultRollFrames = 20
	g.animState.resultFadeFrames = 28
	g.animState.actionResult = strings.ToLower(strings.TrimSpace(update.Event))
	g.animState.actionResultOK = update.Result != "fail"
	g.animState.actionResultFlash = 18
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
	ebitenutil.DrawRect(screen, 8, 24, 420, 118, color.RGBA{R: 30, G: 30, B: 40, A: 220})

	// Draw outcome, resource delta, doom change, and dice text with wrapping instead of truncation.
	y := 30
	y = drawWrappedText(screen, g.results.OutcomeText(), 400, 14, y, color.White) + 3
	y = drawWrappedText(screen, g.results.ResourceDeltaText(), 400, 14, y, color.RGBA{R: 210, G: 230, B: 255, A: 255}) + 2
	y = drawWrappedText(screen, g.results.DoomChangeText(), 400, 14, y, color.RGBA{R: 255, G: 210, B: 180, A: 255}) + 2
	drawWrappedText(screen, g.results.DiceText(), 400, 14, y, color.RGBA{R: 220, G: 240, B: 255, A: 255})
	g.drawDiceResultVisualization(screen)
}

func (g *Game) drawDiceResultVisualization(screen *ebiten.Image) {
	outcome := g.results.CurrentOutcome()
	if outcome == nil || outcome.DiceRoll == nil || len(outcome.DiceRoll.Dice) == 0 {
		return
	}

	const (
		diceSize = 26
		diceGap  = 8
		baseY    = 110
	)

	count := len(outcome.DiceRoll.Dice)
	rowW := count*diceSize + (count-1)*diceGap
	startX := 16 + (404-rowW)/2

	rollFrames := g.animState.resultRollFrames
	fadeFrames := g.animState.resultFadeFrames
	alpha := uint8(255)
	if fadeFrames > 0 {
		alpha = uint8(min(255, 120+fadeFrames*4))
	}

	for i, die := range outcome.DiceRoll.Dice {
		x := float64(startX + i*(diceSize+diceGap))
		y := float64(baseY)
		if rollFrames > 0 {
			phase := float64(g.frameCount+int64(i*5)) / 3.8
			y += math.Sin(phase) * 3
		}
		g.drawDiceFace(screen, x, y, float64(diceSize), strings.ToLower(strings.TrimSpace(die)), alpha)
	}

	status := "Success!"
	statusColor := color.RGBA{R: 150, G: 235, B: 168, A: alpha}
	if !outcome.DiceRoll.Passed {
		status = "Failed - Doom +1"
		statusColor = color.RGBA{R: 255, G: 168, B: 150, A: alpha}
	}
	drawUIText(screen, status, 282, 112, statusColor)
}

func (g *Game) drawDiceFace(screen *ebiten.Image, x, y, size float64, outcome string, alpha uint8) {
	fill := color.RGBA{R: 82, G: 92, B: 108, A: alpha}
	border := color.RGBA{R: 188, G: 198, B: 216, A: alpha}
	glyph := "?"
	switch outcome {
	case "success":
		fill = color.RGBA{R: 54, G: 132, B: 82, A: alpha}
		border = color.RGBA{R: 170, G: 250, B: 190, A: alpha}
		glyph = "S"
	case "tentacle":
		fill = color.RGBA{R: 132, G: 56, B: 78, A: alpha}
		border = color.RGBA{R: 242, G: 178, B: 200, A: alpha}
		glyph = "T"
	default:
		fill = color.RGBA{R: 92, G: 96, B: 110, A: alpha}
		border = color.RGBA{R: 198, G: 204, B: 220, A: alpha}
		glyph = "-"
	}

	rect := image.Rect(int(x), int(y), int(x+size), int(y+size))
	drawRoundedRect(screen, rect, 6, fill)
	drawRoundedBorder(screen, rect, 6, border)
	drawUIText(screen, glyph, rect.Min.X+8, rect.Min.Y+9, color.RGBA{R: 245, G: 246, B: 250, A: alpha})
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
	slide := max(0, 36-int(g.frameCount/2))
	panelX := 120 + slide
	ebitenutil.DrawRect(screen, float64(panelX), 90, 560, 108, color.RGBA{R: 12, G: 12, B: 22, A: 240})
	drawTileBorder(screen, float64(panelX), 90, 560, 108, color.RGBA{R: 140, G: 154, B: 182, A: 255})
	g.drawOnboardingHighlight(screen, step)
	currentStep, totalSteps := g.onboarding.Progress()
	progress := "Step " + strconv.Itoa(currentStep) + " of " + strconv.Itoa(totalSteps)
	drawUIText(screen, progress, panelX+16, 94, color.RGBA{R: 208, G: 220, B: 246, A: 255})

	// Draw title and description with wrapping for better readability.
	y := 110
	y = drawWrappedText(screen, step.Title, 540, panelX+16, y, color.RGBA{R: 230, G: 230, B: 255, A: 255}) + 3
	y = drawWrappedText(screen, step.Description, 540, panelX+16, y, color.White) + 1
	drawUIText(screen, "Use NEXT and SKIP TUTORIAL buttons (keyboard shortcuts are optional)", panelX+16, y, color.RGBA{R: 200, G: 200, B: 220, A: 255})

	nextFill := color.RGBA{R: 58, G: 78, B: 114, A: 245}
	nextBorder := color.RGBA{R: 210, G: 225, B: 255, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.next.Min.X), float64(controls.next.Min.Y), float64(controls.next.Dx()), float64(controls.next.Dy()), nextFill)
	drawTileBorder(screen, float64(controls.next.Min.X), float64(controls.next.Min.Y), float64(controls.next.Dx()), float64(controls.next.Dy()), nextBorder)
	drawUIText(screen, "NEXT", controls.next.Min.X+52, controls.next.Min.Y+18, color.White)

	skipFill := color.RGBA{R: 78, G: 46, B: 48, A: 245}
	skipBorder := color.RGBA{R: 245, G: 195, B: 195, A: 255}
	ebitenutil.DrawRect(screen, float64(controls.skip.Min.X), float64(controls.skip.Min.Y), float64(controls.skip.Dx()), float64(controls.skip.Dy()), skipFill)
	drawTileBorder(screen, float64(controls.skip.Min.X), float64(controls.skip.Min.Y), float64(controls.skip.Dx()), float64(controls.skip.Dy()), skipBorder)
	drawUIText(screen, "SKIP TUTORIAL", controls.skip.Min.X+26, controls.skip.Min.Y+18, color.White)
}

func (g *Game) drawOnboardingHighlight(screen *ebiten.Image, step *onboarding.OnboardingStep) {
	if step == nil {
		return
	}
	rect, ok := g.onboardingHighlightRect(step)
	if !ok {
		return
	}
	ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), color.RGBA{R: 54, G: 74, B: 112, A: 70})
	drawTileBorder(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), color.RGBA{R: 214, G: 230, B: 255, A: 255})
}

func (g *Game) onboardingHighlightRect(step *onboarding.OnboardingStep) (image.Rectangle, bool) {
	if step.Highlight != nil {
		r := step.Highlight
		return image.Rect(int(r.X), int(r.Y), int(r.X+r.Width), int(r.Y+r.Height)), true
	}
	controls := newCameraControls()
	switch step.ID {
	case "resources":
		return image.Rect(10, 32, 190, 116), true
	case "doom":
		return image.Rect(screenWidth-190, 4, screenWidth-8, 64), true
	case "locations":
		return image.Rect(28, 44, 376, 348), true
	case "actions":
		return image.Rect(8, screenHeight-actionGridTotalHeight(), screenWidth-8, screenHeight-8), true
	case "roll_dice":
		return image.Rect(8, 24, 432, 108), true
	case "ready":
		return image.Rect(controls.left.Min.X-4, controls.left.Min.Y-4, controls.toggle.Max.X+4, controls.toggle.Max.Y+4), true
	default:
		return image.Rectangle{}, false
	}
}

func (g *Game) drawCameraControls(screen *ebiten.Image) {
	if g.camera == nil {
		return
	}
	controls := newCameraControls()
	leftHovered, leftPressed := pointerState(controls.left)
	rightHovered, rightPressed := pointerState(controls.right)
	toggleHovered, togglePressed := pointerState(controls.toggle)

	leftFill := color.RGBA{R: 32, G: 36, B: 50, A: 245}
	leftBorder := color.RGBA{R: 156, G: 176, B: 218, A: 255}
	if leftHovered {
		leftFill = color.RGBA{R: 42, G: 50, B: 72, A: 248}
	}
	if leftPressed {
		leftFill = color.RGBA{R: 56, G: 66, B: 94, A: 252}
	}
	ebitenutil.DrawRect(screen, float64(controls.left.Min.X), float64(controls.left.Min.Y), float64(controls.left.Dx()), float64(controls.left.Dy()), leftFill)
	drawTileBorder(screen, float64(controls.left.Min.X), float64(controls.left.Min.Y), float64(controls.left.Dx()), float64(controls.left.Dy()), leftBorder)
	drawUIText(screen, "<", controls.left.Min.X+18, controls.left.Min.Y+16, color.White)

	rightFill := color.RGBA{R: 32, G: 36, B: 50, A: 245}
	rightBorder := color.RGBA{R: 156, G: 176, B: 218, A: 255}
	if rightHovered {
		rightFill = color.RGBA{R: 42, G: 50, B: 72, A: 248}
	}
	if rightPressed {
		rightFill = color.RGBA{R: 56, G: 66, B: 94, A: 252}
	}
	ebitenutil.DrawRect(screen, float64(controls.right.Min.X), float64(controls.right.Min.Y), float64(controls.right.Dx()), float64(controls.right.Dy()), rightFill)
	drawTileBorder(screen, float64(controls.right.Min.X), float64(controls.right.Min.Y), float64(controls.right.Dx()), float64(controls.right.Dy()), rightBorder)
	drawUIText(screen, ">", controls.right.Min.X+18, controls.right.Min.Y+16, color.White)

	toggleFill := color.RGBA{R: 40, G: 58, B: 80, A: 245}
	toggleBorder := color.RGBA{R: 188, G: 214, B: 255, A: 255}
	if toggleHovered {
		toggleFill = color.RGBA{R: 52, G: 74, B: 102, A: 248}
	}
	if togglePressed {
		toggleFill = color.RGBA{R: 64, G: 90, B: 122, A: 252}
	}
	ebitenutil.DrawRect(screen, float64(controls.toggle.Min.X), float64(controls.toggle.Min.Y), float64(controls.toggle.Dx()), float64(controls.toggle.Dy()), toggleFill)
	drawTileBorder(screen, float64(controls.toggle.Min.X), float64(controls.toggle.Min.Y), float64(controls.toggle.Dx()), float64(controls.toggle.Dy()), toggleBorder)
	drawUIText(screen, "V", controls.toggle.Min.X+18, controls.toggle.Min.Y+16, color.White)

	tooltip := ""
	if leftHovered {
		tooltip = "Orbit left ["
	}
	if rightHovered {
		tooltip = "Orbit right ]"
	}
	if toggleHovered {
		tooltip = "Toggle view V"
	}
	if tooltip != "" {
		drawUIText(screen, tooltip, controls.left.Min.X-4, controls.left.Max.Y+4, color.RGBA{R: 220, G: 232, B: 255, A: 255})
	}
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
		px, py, _, height := g.drawLocationTileOverlay(screen, loc, rect, currentLocation, legalSet)

		labelX, labelY := ui.ProjectLabelPosition(float64(rect.x), float64(rect.y), float64(rect.w), float64(rect.h), g.boardView)
		label := string(loc)
		labelW := float64(textWidth(label) + 8)
		ebitenutil.DrawRect(screen, labelX-4, labelY-2, labelW, 16, color.RGBA{R: 8, G: 8, B: 16, A: 214})
		drawUIText(screen, label, int(labelX), int(labelY), color.White)

		g.drawLocationEntityBadges(screen, gs, loc, px, py, height)
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

type tileVisual struct {
	border color.RGBA
	fill   color.RGBA
	note   string
}

func (g *Game) drawLocationTileOverlay(screen *ebiten.Image, loc ebclient.Location, rect struct{ x, y, w, h int }, currentLocation ebclient.Location, legalSet map[ebclient.Location]struct{}) (float64, float64, float64, float64) {
	px, py, scale := g.projectPoint(float64(rect.x), float64(rect.y))
	width := float64(rect.w) * scale
	height := float64(rect.h) * scale
	v := locationTileVisual(loc, currentLocation, legalSet)
	ebitenutil.DrawRect(screen, px, py, width, height, v.fill)
	g.drawLocationPattern(screen, loc, px, py, width, height)
	drawTileBorder(screen, px, py, width, height, v.border)

	noteW := float64(textWidth(v.note) + 6)
	ebitenutil.DrawRect(screen, px+width-noteW-4, py+4, noteW, 14, locationNoteBG(v.note))
	drawUIText(screen, strings.ToUpper(v.note), int(px+width-noteW), int(py+6), color.RGBA{R: 235, G: 240, B: 250, A: 255})
	return px, py, width, height
}

func locationTileVisual(loc, currentLocation ebclient.Location, legalSet map[ebclient.Location]struct{}) tileVisual {
	base := locationColours[loc]
	v := tileVisual{
		border: color.RGBA{R: 225, G: 226, B: 235, A: 165},
		fill:   withAlpha(blendRGBA(base, color.RGBA{R: 12, G: 14, B: 20, A: 255}, 0.42), 155),
		note:   "locked",
	}
	if loc == currentLocation {
		v.border = color.RGBA{R: 252, G: 220, B: 104, A: 250}
		v.fill = withAlpha(blendRGBA(base, color.RGBA{R: 190, G: 145, B: 44, A: 255}, 0.45), 190)
		v.note = "current"
		return v
	}
	if _, ok := legalSet[loc]; ok {
		v.border = color.RGBA{R: 98, G: 234, B: 255, A: 238}
		v.fill = withAlpha(blendRGBA(base, color.RGBA{R: 44, G: 130, B: 176, A: 255}, 0.40), 180)
		v.note = "move"
	}
	return v
}

func locationNoteBG(note string) color.RGBA {
	if note == "move" {
		return color.RGBA{R: 9, G: 40, B: 53, A: 225}
	}
	if note == "current" {
		return color.RGBA{R: 74, G: 55, B: 12, A: 230}
	}
	return color.RGBA{R: 8, G: 10, B: 16, A: 210}
}

// drawLocationPattern adds district-specific line art so tiles read as stylized
// neighborhoods rather than flat placeholder fills.
func (g *Game) drawLocationPattern(screen *ebiten.Image, loc ebclient.Location, x, y, w, h float64) {
	line := color.RGBA{R: 230, G: 236, B: 246, A: 34}
	shadow := color.RGBA{R: 0, G: 0, B: 0, A: 22}
	switch loc {
	case "Downtown":
		for i := 1; i < 5; i++ {
			xi := x + float64(i)*w/5
			ebitenutil.DrawLine(screen, xi, y+6, xi, y+h-8, line)
		}
		for i := 1; i < 3; i++ {
			yi := y + float64(i)*h/3
			ebitenutil.DrawLine(screen, x+6, yi, x+w-6, yi, line)
		}
	case "University":
		for i := 0; i < 6; i++ {
			x0 := x + float64(i)*w/6
			ebitenutil.DrawLine(screen, x0, y+h-6, x+w/2, y+8, line)
		}
		ebitenutil.DrawLine(screen, x+10, y+h-20, x+w-10, y+h-20, shadow)
	case "Rivertown":
		for i := 0; i < 6; i++ {
			yi := y + float64(i)*h/6
			ebitenutil.DrawLine(screen, x+6, yi, x+w-6, yi+4, line)
		}
		ebitenutil.DrawLine(screen, x+12, y+10, x+w-12, y+h-12, shadow)
	case "Northside":
		for i := 0; i < 6; i++ {
			x0 := x + float64(i)*w/6
			ebitenutil.DrawLine(screen, x0, y+8, x0+8, y+h-8, line)
		}
		ebitenutil.DrawLine(screen, x+10, y+12, x+w-12, y+h-10, shadow)
	}
}

func (g *Game) drawLocationEntityBadges(screen *ebiten.Image, gs ebclient.GameState, loc ebclient.Location, px, py, height float64) {
	badgeX := px + 6
	badgeY := py + height - 18
	badgeX = drawCountBadge(screen, badgeX, badgeY, "P", g.countPlayersAt(gs, loc), color.RGBA{R: 230, G: 226, B: 135, A: 230})
	badgeX = drawCountBadge(screen, badgeX, badgeY, "G", g.countGatesAt(gs, loc), color.RGBA{R: 146, G: 90, B: 211, A: 230})
	drawCountBadge(screen, badgeX, badgeY, "E", g.countEnemiesAt(gs, loc), color.RGBA{R: 214, G: 87, B: 87, A: 230})
}

func drawCountBadge(screen *ebiten.Image, x, y float64, prefix string, value int, tint color.RGBA) float64 {
	if value <= 0 {
		return x
	}
	drawEntityBadge(screen, x, y, prefix+":"+strconv.Itoa(value), tint)
	return x + 34
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

// enqueueTokens is kept for layer compatibility; token visuals now draw in
// drawInvestigatorTokens to support richer badge patterns and state styling.
func (g *Game) enqueueTokens(gs ebclient.GameState, myID string) {
	_ = gs
	_ = myID
}

// drawInvestigatorTokens renders investigator-themed board badges with distinct
// patterns per player and state styling for active and disconnected players.
func (g *Game) drawInvestigatorTokens(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	occupants := make(map[ebclient.Location]int)
	for _, pid := range gs.TurnOrder {
		p, ok := gs.Players[pid]
		if !ok || p == nil {
			continue
		}
		rect, ok := locationRects[p.Location]
		if !ok {
			continue
		}

		offset := occupants[p.Location]
		occupants[p.Location]++
		colourIdx := playerColourIndex(pid, gs.TurnOrder)
		base := atmosphericPlayerColour(colourIdx)

		tokenX := float64(rect.x + 26 + (offset%3)*38)
		tokenY := float64(rect.y + rect.h - 22 - (offset/3)*36)
		px, py, scale := g.projectPoint(tokenX, tokenY)
		size := 44.0 * scale
		bob := math.Sin(float64(g.frameCount+int64(colourIdx*7))/14.0) * 0.9 * scale

		isCurrent := pid == gs.CurrentPlayer && gs.GamePhase == "playing"
		isGone := !p.Connected
		if pid == myID {
			size += 2 * scale
		}
		g.drawInvestigatorTokenBadge(screen, px+size/2, py+bob, size, base, investigatorSymbol(p), colourIdx, isCurrent, isGone)
	}
}

func (g *Game) drawInvestigatorTokenBadge(screen *ebiten.Image, cx, cy, size float64, base color.RGBA, symbol string, pattern int, isCurrent, isGone bool) {
	radius := size / 2
	alpha := uint8(245)
	if isGone {
		alpha = 118
	}
	base.A = alpha
	border := blendRGBA(base, color.RGBA{R: 245, G: 245, B: 235, A: alpha}, 0.35)

	if isCurrent {
		pulse := 0.45 + 0.55*math.Sin(float64(g.frameCount)/7.0)
		glow := color.RGBA{R: 255, G: uint8(180 + int(50*pulse)), B: 98, A: uint8(120 + int(65*pulse))}
		ebitenutil.DrawRect(screen, cx-radius-4, cy-radius-4, size+8, size+8, glow)
	}

	ebitenutil.DrawRect(screen, cx-radius, cy-radius, size, size, base)
	drawTileBorder(screen, cx-radius, cy-radius, size, size, border)
	g.drawTokenPattern(screen, cx, cy, radius, pattern, isGone)

	textColor := color.RGBA{R: 246, G: 246, B: 240, A: 255}
	if isGone {
		textColor = color.RGBA{R: 192, G: 192, B: 188, A: 220}
	}
	drawUIText(screen, symbol, int(cx-4), int(cy+4), textColor)
}

func (g *Game) drawTokenPattern(screen *ebiten.Image, cx, cy, radius float64, pattern int, isGone bool) {
	overlay := color.RGBA{R: 10, G: 14, B: 18, A: 78}
	if isGone {
		overlay.A = 48
	}

	switch pattern % 6 {
	case 0:
		for i := -2; i <= 2; i++ {
			ebitenutil.DrawRect(screen, cx-radius+float64(i*6), cy-radius, 2, radius*2, overlay)
		}
	case 1:
		for i := -2; i <= 2; i++ {
			ebitenutil.DrawRect(screen, cx-radius, cy-radius+float64(i*6), radius*2, 2, overlay)
		}
	case 2:
		for i := -2; i <= 2; i++ {
			ebitenutil.DrawRect(screen, cx-radius+float64(i*6), cy-radius+float64(i*6), 2, radius*2, overlay)
		}
	case 3:
		for i := -2; i <= 2; i++ {
			ebitenutil.DrawRect(screen, cx-radius+float64(i*6), cy-radius-float64(i*6), 2, radius*2, overlay)
		}
	case 4:
		for y := -1; y <= 1; y++ {
			for x := -1; x <= 1; x++ {
				ebitenutil.DrawRect(screen, cx+float64(x*7)-1, cy+float64(y*7)-1, 3, 3, overlay)
			}
		}
	default:
		ebitenutil.DrawRect(screen, cx-radius, cy-1, radius*2, 2, overlay)
		ebitenutil.DrawRect(screen, cx-1, cy-radius, 2, radius*2, overlay)
	}
}

func investigatorSymbol(p *ebclient.Player) string {
	if p == nil {
		return "?"
	}
	if name := strings.TrimSpace(string(p.InvestigatorType)); name != "" {
		r := []rune(strings.ToUpper(name))
		if len(r) > 0 {
			return string(r[0])
		}
	}
	return "I"
}

func atmosphericPlayerColour(index int) color.RGBA {
	base := playerColours[index%len(playerColours)]
	shadow := color.RGBA{R: 18, G: 20, B: 28, A: 255}
	return blendRGBA(base, shadow, 0.18)
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
	drawUIText(screen, "[ Connecting to server... ]", screenWidth/2-90, 38, color.White)
}

// drawDoomCounter renders the global doom track (0–12) on the right side.
func (g *Game) drawDoomCounter(screen *ebiten.Image, gs ebclient.GameState) {
	panel := image.Rect(rightPanelX()-10, 42, rightPanelX()-10+386, 98)
	drawStyledPanel(screen, panel, panelStyle{radius: 10}, "Doom Track", "Global pressure")
	flash := g.animState.doomFlashFrames
	if gs.Doom >= 12 {
		g.drawMaxDoomGlow(screen)
	}
	drawUITextScaled(screen, g.doomLabel(gs.Doom), rightPanelX(), 64, g.doomLabelColor(flash), textScaleHeader)
	g.drawDoomTrackSegments(screen, gs.Doom, flash)
	drawUITextScaled(screen, strconv.Itoa(gs.Doom)+"/12", 652, 82, color.RGBA{R: 242, G: 246, B: 255, A: 255}, textScaleCaption)
}

func (g *Game) doomLabelColor(flash int) color.RGBA {
	if flash <= 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	boost := uint8(min(70, flash*3))
	return color.RGBA{R: 255, G: uint8(205 + boost/3), B: uint8(205 + boost/3), A: 255}
}

func (g *Game) drawDoomTrackSegments(screen *ebiten.Image, doom, flash int) {
	const (
		segments     = 12
		segmentW     = 14
		segmentGap   = 2
		segmentH     = 14
		segmentY     = 82
		segmentStart = 430
	)
	for i := 0; i < segments; i++ {
		x := segmentStart + i*(segmentW+segmentGap)
		col := g.doomSegmentFill(i, doom, flash)
		ebitenutil.DrawRect(screen, float64(x), segmentY, segmentW, segmentH, col)
		drawTileBorder(screen, float64(x), segmentY, segmentW, segmentH, color.RGBA{R: 94, G: 104, B: 122, A: 255})
	}
}

func (g *Game) doomSegmentFill(index, doom, flash int) color.RGBA {
	bg := color.RGBA{R: 34, G: 38, B: 46, A: 255}
	if index >= doom {
		return bg
	}
	return brightenDoomColor(g.doomSegmentColor(index), flash)
}

func brightenDoomColor(col color.RGBA, flash int) color.RGBA {
	if flash <= 0 {
		return col
	}
	boost := uint8(min(80, flash*4))
	return color.RGBA{
		R: uint8(min(255, int(col.R)+int(boost/3))),
		G: uint8(min(255, int(col.G)+int(boost/4))),
		B: uint8(min(255, int(col.B)+int(boost/4))),
		A: 255,
	}
}

func (g *Game) drawMaxDoomGlow(screen *ebiten.Image) {
	pulse := 0.55 + 0.45*math.Sin(float64(g.frameCount)/8.0)
	glow := color.RGBA{R: 170, G: 36, B: 30, A: uint8(70 + int(50*pulse))}
	ebitenutil.DrawRect(screen, 422, 78, 194, 22, glow)
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
	panel := image.Rect(panelX, panelY, panelX+panelW, panelY+188)
	drawStyledPanel(screen, panel, panelStyle{radius: 10}, "Turn Overview", g.phaseLabel(gs.GamePhase))

	if len(gs.TurnOrder) == 0 {
		drawUIText(screen, "Waiting for players...", rightPanelX(), panelY+28, color.White)
		return
	}

	y := panelY + 34
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
	pulse := 0
	if pid == currentPlayer {
		pulse = g.animState.turnFlashFrames
	}
	fill, border := playerRowStyle(pid == currentPlayer, pulse)
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
	drawUITextScaled(screen, turnGlyph+" "+name, rightPanelX(), y+5, color.White, textScaleHeader)

	pillX := rightPanelX() + 162
	pillX = g.drawResourcePill(screen, pillX, y+4, g.iconLabel(ui.IconHealth, "HP"), p.Resources.Health, color.RGBA{R: 200, G: 82, B: 82, A: 255}, g.resourceFlashLevel(pid, "health"))
	pillX = g.drawResourcePill(screen, pillX, y+4, g.iconLabel(ui.IconSanity, "SN"), p.Resources.Sanity, color.RGBA{R: 90, G: 160, B: 232, A: 255}, g.resourceFlashLevel(pid, "sanity"))
	pillX = g.drawResourcePill(screen, pillX, y+4, g.iconLabel(ui.IconClues, "CL"), p.Resources.Clues, color.RGBA{R: 86, G: 194, B: 122, A: 255}, g.resourceFlashLevel(pid, "clues"))
	g.drawResourcePill(screen, pillX, y+4, "ACT", p.ActionsRemaining, color.RGBA{R: 228, G: 197, B: 102, A: 255}, g.resourceFlashLevel(pid, "actions"))
	return cardH + 5
}

func playerRowStyle(isCurrent bool, pulse int) (color.RGBA, color.RGBA) {
	if isCurrent {
		fill := color.RGBA{R: 58, G: 70, B: 110, A: 228}
		border := color.RGBA{R: 220, G: 232, B: 255, A: 250}
		if pulse > 0 {
			boost := uint8(min(95, pulse*3))
			fill = color.RGBA{R: uint8(min(255, int(fill.R)+int(boost/5))), G: uint8(min(255, int(fill.G)+int(boost/5))), B: uint8(min(255, int(fill.B)+int(boost/3))), A: fill.A}
			border = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		return fill, border
	}
	return color.RGBA{R: 24, G: 28, B: 38, A: 210}, color.RGBA{R: 98, G: 112, B: 148, A: 230}
}

func (g *Game) drawResourcePill(screen *ebiten.Image, x, y int, icon string, value int, accent color.RGBA, flash int) int {
	if icon == "" {
		icon = "?"
	}
	label := icon + ":" + strconv.Itoa(value)
	width := textWidth(label) + 12
	bg := color.RGBA{R: 10, G: 12, B: 20, A: 220}
	if flash > 0 {
		boost := uint8(min(80, flash*4))
		bg = color.RGBA{R: uint8(min(255, int(bg.R)+int(boost/6))), G: uint8(min(255, int(bg.G)+int(boost/6))), B: uint8(min(255, int(bg.B)+int(boost/4))), A: bg.A}
		accent = color.RGBA{R: uint8(min(255, int(accent.R)+int(boost/3))), G: uint8(min(255, int(accent.G)+int(boost/3))), B: uint8(min(255, int(accent.B)+int(boost/3))), A: 255}
	}
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(width), 14, bg)
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(width), 2, accent)
	drawUIText(screen, label, x+4, y+3, color.RGBA{R: 245, G: 247, B: 252, A: 255})
	return x + width + 4
}

func (g *Game) advanceHUDAnimations(gs ebclient.GameState) {
	if gs.Doom != g.animState.lastDoom {
		g.animState.lastDoom = gs.Doom
		g.animState.doomFlashFrames = 22
	}
	if gs.CurrentPlayer != g.animState.lastCurrentPlayer {
		g.animState.lastCurrentPlayer = gs.CurrentPlayer
		g.animState.turnFlashFrames = 24
	}

	for pid, p := range gs.Players {
		if p == nil {
			continue
		}
		prev := g.animState.resourceSnapshot[pid]
		if prev.health != p.Resources.Health {
			g.animState.resourceFlash[pid+":health"] = 18
		}
		if prev.sanity != p.Resources.Sanity {
			g.animState.resourceFlash[pid+":sanity"] = 18
		}
		if prev.clues != p.Resources.Clues {
			g.animState.resourceFlash[pid+":clues"] = 18
		}
		if prev.actions != p.ActionsRemaining {
			g.animState.resourceFlash[pid+":actions"] = 14
		}
		g.animState.resourceSnapshot[pid] = playerResourceSnapshot{
			health:  p.Resources.Health,
			sanity:  p.Resources.Sanity,
			clues:   p.Resources.Clues,
			actions: p.ActionsRemaining,
		}
	}

	if g.animState.doomFlashFrames > 0 {
		g.animState.doomFlashFrames--
	}
	if g.animState.turnFlashFrames > 0 {
		g.animState.turnFlashFrames--
	}
	for key, frames := range g.animState.resourceFlash {
		if frames <= 1 {
			delete(g.animState.resourceFlash, key)
			continue
		}
		g.animState.resourceFlash[key] = frames - 1
	}
	if g.animState.resultRollFrames > 0 {
		g.animState.resultRollFrames--
	}
	if g.animState.resultFadeFrames > 0 {
		g.animState.resultFadeFrames--
	}
	if g.animState.actionResultFlash > 0 {
		g.animState.actionResultFlash--
	}
}

func (g *Game) resourceFlashLevel(playerID, stat string) int {
	if g == nil {
		return 0
	}
	return g.animState.resourceFlash[playerID+":"+stat]
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

// drawEventLog renders a compact snapshot of the latest events above the action dock.
func (g *Game) drawEventLog(screen *ebiten.Image) {
	entries := g.state.EventLogSnapshot()
	y := screenHeight - 130
	drawUIText(screen, "-- Event Log --", rightPanelX(), y, color.White)
	y += 12

	start := 0
	if len(entries) > 3 {
		start = len(entries) - 3
	}
	for _, e := range entries[start:] {
		drawUIText(screen, trimToWidth(e.Text, 360), rightPanelX(), y, color.RGBA{R: 220, G: 220, B: 220, A: 255})
		y += 12
	}
}

// drawInputHints renders the bottom action dock and its compact status line.
func (g *Game) drawInputHints(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	panelX := 10
	panelH := actionGridTotalHeight()
	panelY := screenHeight - panelH
	panelW := screenWidth - 20
	panel := image.Rect(panelX, panelY, panelX+panelW, panelY+panelH)
	drawStyledPanel(screen, panel, panelStyle{radius: 12}, "Action Bar", "Available actions and shortcuts")
	// Split header and action grid into clear sections for readability.
	headerY := panelY + 26
	ebitenutil.DrawRect(screen, float64(panelX+2), float64(headerY), float64(panelW-4), 20, color.RGBA{R: 30, G: 34, B: 48, A: 182})
	ebitenutil.DrawRect(screen, float64(panelX+2), float64(headerY+20), float64(panelW-4), float64(panelH-22), color.RGBA{R: 14, G: 18, B: 28, A: 176})
	drawUIText(screen, g.actionDockSummary(gs, myID), panelX+10, panelY+30, color.RGBA{R: 232, G: 238, B: 248, A: 255})
	drawUIText(screen, trimToWidth(g.actionDockHint(gs, myID), panelW-20), panelX+10, panelY+42, color.RGBA{R: 194, G: 212, B: 238, A: 255})
	g.drawVisibleActionButtons(screen, gs, myID)
	g.drawEndBanner(screen, gs)
}

func (g *Game) drawLocationPanel(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	panel := locationPanelRect()
	drawStyledPanel(screen, panel, panelStyle{radius: 12}, "Location Panel", "Click or tap to move. Press 1-4 for shortcuts.")

	hovered := g.state.HoveredActionHint()
	focused := g.state.FocusedActionHint()
	pressed := g.state.PressedActionHint()
	for _, button := range locationPanelButtons(gs, myID) {
		fill, border, labelColor := locationPanelButtonStyle(button, focused, hovered, pressed)
		ebitenutil.DrawRect(screen, float64(button.rect.Min.X), float64(button.rect.Min.Y), float64(button.rect.Dx()), float64(button.rect.Dy()), fill)
		drawTileBorder(screen, float64(button.rect.Min.X), float64(button.rect.Min.Y), float64(button.rect.Dx()), float64(button.rect.Dy()), border)
		drawUIText(screen, string(button.location), button.rect.Min.X+10, button.rect.Min.Y+10, labelColor)
		drawUIText(screen, button.detail, button.rect.Min.X+10, button.rect.Min.Y+28, color.RGBA{R: 208, G: 220, B: 238, A: 255})
		drawUIText(screen, "Adjacency: "+g.locationAdjacencyLabel(button.location), button.rect.Min.X+10, button.rect.Min.Y+42, color.RGBA{R: 174, G: 190, B: 214, A: 255})
		drawUIText(screen, "["+button.shortcut+"]", button.rect.Max.X-28, button.rect.Min.Y+10, color.RGBA{R: 220, G: 228, B: 252, A: 255})
		if button.population > 0 {
			drawUIText(screen, "P:"+strconv.Itoa(button.population), button.rect.Max.X-34, button.rect.Min.Y+42, color.RGBA{R: 236, G: 224, B: 152, A: 255})
		}
	}
}

type panelStyle struct {
	radius int
}

func drawStyledPanel(screen *ebiten.Image, rect image.Rectangle, style panelStyle, title string, subtitle ...string) {
	if rect.Empty() {
		return
	}
	radius := style.radius
	if radius <= 0 {
		radius = 8
	}

	shadow := image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X+2, rect.Max.Y+2)
	drawRoundedRect(screen, shadow, radius, color.RGBA{R: 0, G: 0, B: 0, A: 72})
	drawRoundedRect(screen, rect, radius, color.RGBA{R: 12, G: 14, B: 24, A: 214})
	drawRoundedBorder(screen, rect, radius, color.RGBA{R: 110, G: 130, B: 165, A: 245})

	titleBar := image.Rect(rect.Min.X+1, rect.Min.Y+1, rect.Max.X-1, rect.Min.Y+24)
	drawRoundedRect(screen, titleBar, max(2, radius-3), color.RGBA{R: 34, G: 40, B: 58, A: 220})
	drawUIText(screen, title, rect.Min.X+10, rect.Min.Y+8, color.RGBA{R: 242, G: 244, B: 252, A: 255})
	if len(subtitle) > 0 && strings.TrimSpace(subtitle[0]) != "" {
		drawUIText(screen, trimToWidth(subtitle[0], rect.Dx()-20), rect.Min.X+136, rect.Min.Y+8, color.RGBA{R: 196, G: 214, B: 240, A: 255})
	}
}

func (g *Game) drawStatusRail(screen *ebiten.Image, gs ebclient.GameState, myID string, connected bool) {
	rail := image.Rect(10, 8, screenWidth-10, 36)
	drawStyledPanel(screen, rail, panelStyle{radius: 10}, "Status Rail")

	turnLabel := g.turnStatusLabel(gs, myID, gs.CurrentPlayer == myID)
	actionsLeft := 0
	if current, ok := gs.Players[gs.CurrentPlayer]; ok && current != nil {
		actionsLeft = current.ActionsRemaining
	}
	connection := "online"
	if !connected {
		connection = "reconnecting"
	}
	if g.animState.turnFlashFrames > 0 {
		pulse := 0.4 + 0.6*math.Sin(float64(g.frameCount)/5.0)
		w := 230 + int(14*pulse)
		h := 18 + int(4*pulse)
		x := rail.Min.X + 98 - (w-230)/2
		y := rail.Min.Y + 9 - (h-18)/2
		ebitenutil.DrawRect(screen, float64(x), float64(y), float64(w), float64(h), color.RGBA{R: 76, G: 104, B: 154, A: uint8(80 + int(40*pulse))})
		drawTileBorder(screen, float64(x), float64(y), float64(w), float64(h), color.RGBA{R: 188, G: 214, B: 255, A: 228})
	}
	status := "Turn: " + trimToWidth(turnLabel, 220) + " | Doom: " + strconv.Itoa(gs.Doom) + "/12 | Connection: " + connection
	statusColor := color.RGBA{R: 246, G: 248, B: 255, A: 255}
	if g.animState.turnFlashFrames > 0 {
		statusColor = color.RGBA{R: 255, G: 246, B: 214, A: 255}
	}
	drawUIText(screen, trimToWidth(status, rail.Dx()-250), rail.Min.X+104, rail.Min.Y+10, statusColor)

	// Action dots mirror remaining action economy for the current turn.
	dotX := rail.Max.X - 98
	drawUIText(screen, "ACT", dotX-26, rail.Min.Y+10, color.RGBA{R: 220, G: 228, B: 252, A: 255})
	for i := 0; i < 2; i++ {
		col := color.RGBA{R: 66, G: 72, B: 86, A: 255}
		if i < actionsLeft {
			col = color.RGBA{R: 226, G: 190, B: 98, A: 255}
		}
		ebitenutil.DrawRect(screen, float64(dotX+i*16), float64(rail.Min.Y+12), 10, 10, col)
		drawTileBorder(screen, float64(dotX+i*16), float64(rail.Min.Y+12), 10, 10, color.RGBA{R: 188, G: 196, B: 220, A: 255})
	}
}

func (g *Game) drawResourceRail(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	player, ok := gs.Players[myID]
	if !ok || player == nil {
		return
	}
	rail := image.Rect(10, 42, 236, 140)
	drawStyledPanel(screen, rail, panelStyle{radius: 10}, "Resources", "Current investigator")

	healthCol := g.tokenOrDefault("color-health", color.RGBA{R: 194, G: 72, B: 66, A: 255})
	sanityCol := g.tokenOrDefault("color-sanity", color.RGBA{R: 86, G: 150, B: 218, A: 255})
	clueCol := g.tokenOrDefault("color-clues", color.RGBA{R: 214, G: 176, B: 78, A: 255})

	drawUIText(screen, g.iconLabel(ui.IconHealth, "HP")+" "+strconv.Itoa(player.Resources.Health), rail.Min.X+10, rail.Min.Y+30, color.White)
	g.drawSegmentedBar(screen, rail.Min.X+74, rail.Min.Y+32, 10, player.Resources.Health, 12, 4, healthCol)

	drawUIText(screen, g.iconLabel(ui.IconSanity, "SN")+" "+strconv.Itoa(player.Resources.Sanity), rail.Min.X+10, rail.Min.Y+52, color.White)
	g.drawSegmentedBar(screen, rail.Min.X+74, rail.Min.Y+54, 10, player.Resources.Sanity, 12, 4, sanityCol)

	drawUIText(screen, g.iconLabel(ui.IconClues, "CL")+" "+strconv.Itoa(player.Resources.Clues), rail.Min.X+10, rail.Min.Y+74, color.White)
	g.drawSegmentedBar(screen, rail.Min.X+74, rail.Min.Y+76, 5, player.Resources.Clues, 24, 6, clueCol)
}

func (g *Game) drawSegmentedBar(screen *ebiten.Image, x, y, maxSegments, filled, segW, gap int, fill color.RGBA) {
	for i := 0; i < maxSegments; i++ {
		col := color.RGBA{R: 36, G: 40, B: 52, A: 255}
		if i < filled {
			col = fill
		}
		xi := x + i*(segW+gap)
		ebitenutil.DrawRect(screen, float64(xi), float64(y), float64(segW), 10, col)
		drawTileBorder(screen, float64(xi), float64(y), float64(segW), 10, color.RGBA{R: 116, G: 126, B: 144, A: 255})
	}
}

func (g *Game) tokenOrDefault(name string, fallback color.RGBA) color.RGBA {
	if g.tokens == nil {
		return fallback
	}
	tok := g.tokens.GetColor(name)
	if tok == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		return fallback
	}
	return tok
}

func (g *Game) doomSegmentColor(segment int) color.RGBA {
	if segment <= 3 {
		return g.tokenOrDefault("color-success", color.RGBA{R: 78, G: 176, B: 108, A: 255})
	}
	if segment <= 7 {
		return g.tokenOrDefault("color-warning", color.RGBA{R: 226, G: 184, B: 78, A: 255})
	}
	return g.tokenOrDefault("color-doom", color.RGBA{R: 198, G: 76, B: 72, A: 255})
}

func (g *Game) drawCoachMarks(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	if g.onboarding != nil && g.onboarding.IsActive() {
		return
	}
	if gs.GamePhase != "playing" {
		return
	}
	if g.coachTicks > 60*90 {
		return
	}
	g.coachTicks++

	hints := g.buildCoachHints(gs, myID)
	if len(hints) == 0 {
		return
	}

	x := 10
	y := 402
	w := 410
	h := 18 + len(hints)*14
	ebitenutil.DrawRect(screen, float64(x), float64(y), float64(w), float64(h), color.RGBA{R: 10, G: 14, B: 26, A: 212})
	border := color.RGBA{R: 188, G: 210, B: 255, A: 235}
	drawTileBorder(screen, float64(x), float64(y), float64(w), float64(h), border)
	drawUIText(screen, "Coach marks", x+8, y+5, color.RGBA{R: 238, G: 246, B: 255, A: 255})
	lineY := y + 19
	for _, hint := range hints {
		drawUIText(screen, trimToWidth("- "+hint, w-14), x+7, lineY, color.RGBA{R: 225, G: 235, B: 252, A: 255})
		lineY += 14
	}
}

func (g *Game) buildCoachHints(gs ebclient.GameState, myID string) []string {
	hints := make([]string, 0, 4)
	currentName := g.playerDisplayName(gs, gs.CurrentPlayer)
	if gs.CurrentPlayer != myID {
		hints = append(hints, "Wait for your turn. Current player: "+currentName)
		return hints
	}

	remaining := g.remainingActions(gs, myID)
	legalMoves := boardAdjacency[g.playerLocation(gs, myID)]
	hints = append(hints, "You have "+strconv.Itoa(remaining)+" actions this turn. Move only to highlighted adjacent locations.")
	if len(legalMoves) > 0 {
		moves := string(legalMoves[0])
		for i := 1; i < len(legalMoves); i++ {
			moves += ", " + string(legalMoves[i])
		}
		hints = append(hints, "Legal moves now: "+moves)
	}

	if sanity := g.playerSanity(gs, myID); sanity <= 1 {
		hints = append(hints, "Ward costs 1 sanity. Gather or recover sanity before casting.")
	} else {
		hints = append(hints, "Ward costs 1 sanity and can reduce doom on a successful roll.")
	}

	needed := len(gs.Players) * 4
	hints = append(hints, "Objective: collect "+strconv.Itoa(needed)+" clues before doom reaches 12.")
	return hints
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
			rect = animatedActionRect(rect, actionName, focused, hovered, pressed)
			fill, border, labelColor, detailColor := actionButtonStyle(action.Available, actionName, focused, hovered, pressed)
			fill, border = g.actionResultFeedbackColors(actionName, fill, border)
			if strings.EqualFold(actionName, hovered) {
				drawRoundedRect(screen, image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X+2, rect.Max.Y+2), 6, color.RGBA{R: 8, G: 10, B: 18, A: 170})
			}

			// Rounded cards and an explicit icon badge improve affordance and readability.
			drawRoundedRect(screen, rect, 6, fill)
			drawRoundedBorder(screen, rect, 6, border)

			iconRect := image.Rect(rect.Min.X+6, rect.Min.Y+6, rect.Min.X+30, rect.Min.Y+30)
			iconFill := color.RGBA{R: 18, G: 22, B: 34, A: 248}
			if action.Available {
				iconFill = color.RGBA{R: 22, G: 42, B: 34, A: 248}
			}
			if strings.EqualFold(actionName, hovered) || strings.EqualFold(actionName, focused) {
				iconFill = color.RGBA{R: 34, G: 44, B: 72, A: 248}
			}
			drawRoundedRect(screen, iconRect, 4, iconFill)
			drawRoundedBorder(screen, iconRect, 4, border)
			iconLabel := trimToWidth(g.actionGlyph(actionName), iconRect.Dx()-8)
			drawUIText(screen, iconLabel, iconRect.Min.X+4, iconRect.Min.Y+8, labelColor)
			if strings.EqualFold(actionName, pressed) {
				g.drawActionPendingIndicator(screen, iconRect)
			}

			labelX := iconRect.Max.X + 6
			label := trimToWidth(g.actionButtonLabel(actionName), rect.Max.X-labelX-8)
			drawUIText(screen, label, labelX, rect.Min.Y+5, labelColor)
			detail := trimToWidth(action.Detail, rect.Max.X-labelX-8)
			if strings.EqualFold(actionName, hovered) || strings.EqualFold(actionName, focused) {
				detail = trimToWidth(actionDisplayLabel(actionName)+": "+action.Detail+g.actionShortcutTooltip(actionName), rect.Max.X-labelX-8)
			}
			drawUIText(screen, detail, labelX, rect.Min.Y+19, detailColor)
		}
	}
}

func animatedActionRect(rect image.Rectangle, actionName, focused, hovered, pressed string) image.Rectangle {
	if strings.EqualFold(actionName, pressed) {
		return image.Rect(rect.Min.X+2, rect.Min.Y+1, rect.Max.X-2, rect.Max.Y-1)
	}
	if strings.EqualFold(actionName, hovered) || strings.EqualFold(actionName, focused) {
		return image.Rect(rect.Min.X-2, rect.Min.Y-1, rect.Max.X+2, rect.Max.Y+1)
	}
	return rect
}

func (g *Game) drawActionPendingIndicator(screen *ebiten.Image, rect image.Rectangle) {
	phase := (g.frameCount / 3) % 8
	x := rect.Max.X - 7
	y := rect.Min.Y + 6
	for i := int64(0); i < 8; i++ {
		alpha := uint8(60)
		if i == phase {
			alpha = 220
		}
		ebitenutil.DrawRect(screen, float64(x), float64(y+int(i*2)), 2, 1, color.RGBA{R: 238, G: 246, B: 255, A: alpha})
	}
}

func (g *Game) actionResultFeedbackColors(actionName string, fill, border color.RGBA) (color.RGBA, color.RGBA) {
	if g.animState.actionResultFlash <= 0 {
		return fill, border
	}
	if !strings.EqualFold(strings.TrimSpace(actionName), g.animState.actionResult) {
		return fill, border
	}
	blend := float64(g.animState.actionResultFlash) / 18.0
	if g.animState.actionResultOK {
		return blendRGBA(fill, color.RGBA{R: 58, G: 120, B: 78, A: fill.A}, blend*0.8), blendRGBA(border, color.RGBA{R: 178, G: 244, B: 190, A: border.A}, blend)
	}
	return blendRGBA(fill, color.RGBA{R: 132, G: 52, B: 52, A: fill.A}, blend*0.8), blendRGBA(border, color.RGBA{R: 242, G: 168, B: 160, A: border.A}, blend)
}

func drawRoundedRect(screen *ebiten.Image, rect image.Rectangle, radius int, fill color.RGBA) {
	if rect.Empty() {
		return
	}
	w := rect.Dx()
	h := rect.Dy()
	if radius < 1 {
		ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), fill)
		return
	}
	if radius*2 > w {
		radius = w / 2
	}
	if radius*2 > h {
		radius = h / 2
	}
	if radius < 1 {
		ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), fill)
		return
	}

	ebitenutil.DrawRect(screen, float64(rect.Min.X+radius), float64(rect.Min.Y), float64(w-2*radius), float64(h), fill)
	ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y+radius), float64(radius), float64(h-2*radius), fill)
	ebitenutil.DrawRect(screen, float64(rect.Max.X-radius), float64(rect.Min.Y+radius), float64(radius), float64(h-2*radius), fill)

	r2 := radius * radius
	for dy := 0; dy < radius; dy++ {
		for dx := 0; dx < radius; dx++ {
			xd := radius - 1 - dx
			yd := radius - 1 - dy
			if xd*xd+yd*yd > r2 {
				continue
			}
			ebitenutil.DrawRect(screen, float64(rect.Min.X+dx), float64(rect.Min.Y+dy), 1, 1, fill)
			ebitenutil.DrawRect(screen, float64(rect.Max.X-radius+dx), float64(rect.Min.Y+dy), 1, 1, fill)
			ebitenutil.DrawRect(screen, float64(rect.Min.X+dx), float64(rect.Max.Y-radius+dy), 1, 1, fill)
			ebitenutil.DrawRect(screen, float64(rect.Max.X-radius+dx), float64(rect.Max.Y-radius+dy), 1, 1, fill)
		}
	}
}

func drawRoundedBorder(screen *ebiten.Image, rect image.Rectangle, radius int, border color.RGBA) {
	if rect.Empty() {
		return
	}
	w := rect.Dx()
	h := rect.Dy()
	if radius < 1 {
		drawTileBorder(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), border)
		return
	}
	if radius*2 > w {
		radius = w / 2
	}
	if radius*2 > h {
		radius = h / 2
	}
	if radius < 1 {
		drawTileBorder(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), border)
		return
	}

	ebitenutil.DrawRect(screen, float64(rect.Min.X+radius), float64(rect.Min.Y), float64(w-2*radius), 1, border)
	ebitenutil.DrawRect(screen, float64(rect.Min.X+radius), float64(rect.Max.Y-1), float64(w-2*radius), 1, border)
	ebitenutil.DrawRect(screen, float64(rect.Min.X), float64(rect.Min.Y+radius), 1, float64(h-2*radius), border)
	ebitenutil.DrawRect(screen, float64(rect.Max.X-1), float64(rect.Min.Y+radius), 1, float64(h-2*radius), border)

	r2 := radius * radius
	for dy := 0; dy < radius; dy++ {
		for dx := 0; dx < radius; dx++ {
			xd := radius - 1 - dx
			yd := radius - 1 - dy
			d := xd*xd + yd*yd
			if d > r2 || d < (radius-2)*(radius-2) {
				continue
			}
			ebitenutil.DrawRect(screen, float64(rect.Min.X+dx), float64(rect.Min.Y+dy), 1, 1, border)
			ebitenutil.DrawRect(screen, float64(rect.Max.X-radius+dx), float64(rect.Min.Y+dy), 1, 1, border)
			ebitenutil.DrawRect(screen, float64(rect.Min.X+dx), float64(rect.Max.Y-radius+dy), 1, 1, border)
			ebitenutil.DrawRect(screen, float64(rect.Max.X-radius+dx), float64(rect.Max.Y-radius+dy), 1, 1, border)
		}
	}
}

func actionButtonStyle(available bool, actionName, focused, hovered, pressed string) (color.RGBA, color.RGBA, color.RGBA, color.RGBA) {
	fill := color.RGBA{R: 24, G: 28, B: 38, A: 245}
	border := color.RGBA{R: 110, G: 130, B: 165, A: 255}
	labelColor := color.RGBA{R: 220, G: 220, B: 235, A: 255}
	detailColor := color.RGBA{R: 180, G: 198, B: 220, A: 255}
	if available {
		fill = color.RGBA{R: 28, G: 62, B: 50, A: 245}
		border = color.RGBA{R: 170, G: 240, B: 190, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		detailColor = color.RGBA{R: 214, G: 238, B: 222, A: 255}
	}
	if strings.EqualFold(actionName, focused) {
		fill = color.RGBA{R: 52, G: 54, B: 82, A: 245}
		border = color.RGBA{R: 190, G: 202, B: 255, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		detailColor = color.RGBA{R: 220, G: 228, B: 255, A: 255}
	}
	if strings.EqualFold(actionName, hovered) {
		fill = color.RGBA{R: 58, G: 64, B: 98, A: 245}
		border = color.RGBA{R: 206, G: 220, B: 255, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		detailColor = color.RGBA{R: 232, G: 238, B: 255, A: 255}
	}
	if strings.EqualFold(actionName, pressed) {
		fill = color.RGBA{R: 78, G: 88, B: 124, A: 245}
		border = color.RGBA{R: 232, G: 236, B: 255, A: 255}
		labelColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		detailColor = color.RGBA{R: 242, G: 246, B: 255, A: 255}
	}
	return fill, border, labelColor, detailColor
}

func (g *Game) actionGlyph(actionName string) string {
	switch strings.ToLower(actionName) {
	case "gather":
		return g.iconLabel(ui.IconGather, "G")
	case "investigate":
		return g.iconLabel(ui.IconInvestigate, "I")
	case "ward":
		return g.iconLabel(ui.IconWard, "W")
	case "focus":
		return "F"
	case "research":
		return "R"
	case "trade":
		return "T"
	case "component":
		return "C"
	case "attack":
		return "A"
	case "evade":
		return "E"
	case "closegate":
		return "X"
	case "encounter":
		return "N"
	default:
		return "+"
	}
}

func (g *Game) actionDockSummary(gs ebclient.GameState, myID string) string {
	layout := "portrait rows"
	if screenWidth >= screenHeight {
		layout = "desktop row"
	}
	return "Action Dock | " + layout + " | " + g.turnStatusLabel(gs, myID, gs.CurrentPlayer == myID && gs.GamePhase == "playing") + " | Actions left: " + strconv.Itoa(g.remainingActions(gs, myID))
}

func (g *Game) actionButtonLabel(actionName string) string {
	switch strings.ToLower(actionName) {
	case "gather":
		return "Gather Clues"
	case "investigate":
		return "Investigate"
	case "ward":
		return "Cast Ward"
	case "focus":
		return "Gain Focus"
	case "research":
		return "Research"
	case "trade":
		return "Trade"
	case "component":
		return "Ability"
	case "attack":
		return "Attack"
	case "evade":
		return "Evade"
	case "closegate":
		return "Close Gate"
	case "encounter":
		return "Draw"
	default:
		return actionDisplayLabel(actionName)
	}
}

func (g *Game) actionShortcutTooltip(actionName string) string {
	if key := actionShortcutHints[actionName]; key != "" {
		return " | Press " + key
	}
	return ""
}

func (g *Game) actionDockHint(gs ebclient.GameState, myID string) string {
	hovered := strings.TrimSpace(g.state.HoveredActionHint())
	if hovered == "" {
		hovered = strings.TrimSpace(g.state.FocusedActionHint())
	}
	if hovered == "" {
		return "Hover or focus a button to see its detail. Buttons stay visible and clickable even when unavailable."
	}
	for _, action := range g.availableActions(gs, myID) {
		if actionLookupKey(action.Name) == actionLookupKey(hovered) {
			return g.actionButtonLabel(actionLookupKey(action.Name)) + ": " + action.Detail + g.actionShortcutTooltip(actionLookupKey(action.Name))
		}
	}
	return "Hover or focus a button to see its detail."
}

func actionLookupKey(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}

func actionDisplayLabel(actionName string) string {
	switch strings.ToLower(actionName) {
	case "closegate":
		return "Close Gate"
	case "investigate":
		return "Investigate"
	case "component":
		return "Component"
	case "encounter":
		return "Encounter"
	default:
		if actionName == "" {
			return ""
		}
		return strings.ToUpper(actionName[:1]) + actionName[1:]
	}
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

func locationPanelRect() image.Rectangle {
	return image.Rect(rightPanelX()-10, 304, rightPanelX()-10+386, 472)
}

func locationPanelButtonRect(index int) image.Rectangle {
	panel := locationPanelRect()
	const (
		buttonWidth  = 178
		buttonHeight = 64
		gap          = 8
	)
	col := index % 2
	row := index / 2
	x := panel.Min.X + 10 + col*(buttonWidth+gap)
	y := panel.Min.Y + 38 + row*(buttonHeight+gap)
	return image.Rect(x, y, x+buttonWidth, y+buttonHeight)
}

type locationPanelButton struct {
	location   ebclient.Location
	rect       image.Rectangle
	shortcut   string
	detail     string
	current    bool
	available  bool
	inactive   bool
	population int
}

func locationPanelButtons(gs ebclient.GameState, myID string) []locationPanelButton {
	buttons := make([]locationPanelButton, 0, len(boardLocationOrder))
	currentLocation := ebclient.Location("")
	if player, ok := gs.Players[myID]; ok && player != nil {
		currentLocation = player.Location
	}
	legalMoves := boardAdjacency[currentLocation]
	legalSet := make(map[ebclient.Location]struct{}, len(legalMoves))
	for _, move := range legalMoves {
		legalSet[move] = struct{}{}
	}
	for index, loc := range boardLocationOrder {
		button := locationPanelButton{
			location:   loc,
			rect:       locationPanelButtonRect(index),
			shortcut:   moveShortcutHints[loc],
			population: countConnectedPlayersAt(gs, loc),
		}
		switch {
		case loc == currentLocation:
			button.current = true
			button.detail = "Current location"
		case isLegalMove(loc, legalSet):
			button.available = true
			button.detail = "Adjacent move available"
		default:
			button.inactive = true
			button.detail = "Unavailable from here"
		}
		buttons = append(buttons, button)
	}
	return buttons
}

func isLegalMove(loc ebclient.Location, legalSet map[ebclient.Location]struct{}) bool {
	_, ok := legalSet[loc]
	return ok
}

func locationPanelButtonStyle(button locationPanelButton, focused, hovered, pressed string) (color.RGBA, color.RGBA, color.RGBA) {
	fill := color.RGBA{R: 22, G: 28, B: 38, A: 245}
	border := color.RGBA{R: 102, G: 116, B: 144, A: 255}
	labelColor := color.RGBA{R: 232, G: 236, B: 244, A: 255}
	if button.available {
		fill = color.RGBA{R: 24, G: 58, B: 52, A: 245}
		border = color.RGBA{R: 122, G: 220, B: 184, A: 255}
	}
	if button.current {
		fill = color.RGBA{R: 82, G: 64, B: 20, A: 245}
		border = color.RGBA{R: 244, G: 214, B: 112, A: 255}
	}
	if button.inactive {
		labelColor = color.RGBA{R: 188, G: 194, B: 206, A: 255}
	}
	id := string(button.location)
	if strings.EqualFold(id, focused) {
		fill = color.RGBA{R: 50, G: 58, B: 82, A: 245}
		border = color.RGBA{R: 190, G: 206, B: 255, A: 255}
	}
	if strings.EqualFold(id, hovered) {
		fill = color.RGBA{R: 58, G: 72, B: 100, A: 245}
		border = color.RGBA{R: 208, G: 220, B: 255, A: 255}
	}
	if strings.EqualFold(id, pressed) {
		fill = color.RGBA{R: 74, G: 92, B: 126, A: 245}
		border = color.RGBA{R: 234, G: 238, B: 255, A: 255}
	}
	return fill, border, labelColor
}

func (g *Game) locationAdjacencyLabel(loc ebclient.Location) string {
	adjacent := boardAdjacency[loc]
	if len(adjacent) == 0 {
		return "none"
	}
	label := string(adjacent[0])
	for i := 1; i < len(adjacent); i++ {
		label += ", " + string(adjacent[i])
	}
	return trimToWidth(label, 128)
}

func countConnectedPlayersAt(gs ebclient.GameState, loc ebclient.Location) int {
	count := 0
	for _, pid := range gs.TurnOrder {
		player := gs.Players[pid]
		if player != nil && player.Connected && player.Location == loc {
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
		return "Last invalid action: none"
	case "out-of-turn-or-disconnected":
		return "Last invalid action: wait for your turn or reconnect"
	case "trade-no-colocated-player":
		return "Last invalid action: trade needs a co-located ally; move to the same location first"
	default:
		return "Last invalid action: " + reason
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
			g.uiCache.statusText = "Waiting for your turn"
		} else {
			g.uiCache.statusText = "Your turn (" + strconv.Itoa(actions) + " actions left)"
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
