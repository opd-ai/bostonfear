// Package app implements the Ebitengine game loop for the Arkham Horror client.
// It wires together LocalState, NetClient, InputHandler, and the layered
// Compositor into an ebiten.Game implementation. cmd/desktop, cmd/web, and
// cmd/mobile all call NewGame to obtain a ready-to-run game object.
//
// This package is separated from the parent client/ebiten package so that the
// pure network/state logic in that package can be tested without a display.
package app

import (
	"image/color"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
	"github.com/opd-ai/bostonfear/client/ebiten/render"
	"github.com/opd-ai/bostonfear/client/ebiten/ui"
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
	quality     ui.QualityTier
	effects     ui.EffectProfile
	theme       ui.ThemePack
	tokens      *ui.DesignTokenRegistry
	icons       *ui.IconRegistry
	onboarding  *ui.OnboardingController
	stateBanner *ui.StateVisibilityWidget
	results     *ui.ResultsPanel
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
		onboarding:  ui.NewOnboardingController(ui.DefaultArkhamHorrorOnboarding()),
		stateBanner: ui.NewStateVisibilityWidget(),
		results:     ui.NewResultsPanel(),
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
	g.frameCount++
	g.ensureShaders()

	screen.Fill(g.theme.Background)

	gs, playerID, connected := g.state.Snapshot()
	g.updateUXWidgets(connected)
	g.ensureProceduralSeed(gs)
	g.drawProceduralAtmosphere(screen)

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
	g.drawStateBanner(screen)
	g.drawDoomCounter(screen, gs)
	g.drawPlayerPanel(screen, gs, playerID)
	g.drawResultsPanel(screen)
	g.drawEventLog(screen)
	g.drawInputHints(screen, gs, playerID)
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

func (g *Game) applyPostProcess(screen *ebiten.Image, doom int) {
	if g.effects.EnableFog {
		render.DrawFogOverlay(screen, g.shaders, g.effects.FogOpacity)
	}
	if g.effects.EnableGlow {
		render.DrawGlowOverlay(screen, g.shaders, g.effects.GlowIntensity, float32(time.Since(g.startedAt).Seconds()))
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
		ebitenutil.DrawRect(screen, r.X, r.Y, r.W, r.H, col)
	}
}

func (g *Game) updateUXWidgets(connected bool) {
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
	outcome := &ui.ActionOutcome{
		PlayerID:    update.PlayerID,
		ActionType:  update.Event,
		Successful:  update.Result != "fail",
		Description: update.Result,
		ResourceDelta: ui.ResourceDelta{
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
		outcome.DiceRoll = &ui.DiceRollResult{
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
	ebitenutil.DrawRect(screen, 8, 24, 420, 54, color.RGBA{R: 30, G: 30, B: 40, A: 220})
	drawUIText(screen, trimToWidth(g.results.OutcomeText(), 400), 14, 30, color.White)
	drawUIText(screen, trimToWidth(g.results.ResourceDeltaText(), 400), 14, 44, color.RGBA{R: 210, G: 230, B: 255, A: 255})
	drawUIText(screen, trimToWidth(g.results.DoomChangeText(), 400), 14, 58, color.RGBA{R: 255, G: 210, B: 180, A: 255})
}

func (g *Game) drawOnboarding(screen *ebiten.Image) {
	if g.onboarding == nil || !g.onboarding.IsActive() {
		return
	}
	step := g.onboarding.CurrentStep()
	if step == nil {
		return
	}
	ebitenutil.DrawRect(screen, 120, 90, 560, 96, color.RGBA{R: 12, G: 12, B: 22, A: 240})
	drawUIText(screen, trimToWidth(step.Title, 540), 136, 102, color.RGBA{R: 230, G: 230, B: 255, A: 255})
	drawUIText(screen, trimToWidth(step.Description, 540), 136, 120, color.White)
	drawUIText(screen, "ENTER: next  H: skip tutorial", 136, 154, color.RGBA{R: 200, G: 200, B: 220, A: 255})
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
	label := g.doomLabel(gs.Doom)
	drawUIText(screen, label, rightPanelX(), 60, color.White)

	// Draw a simple bar.
	barW := 200.0
	filled := float64(gs.Doom) / 12.0 * barW
	bg, fg := g.doomBarColors()
	ebitenutil.DrawRect(screen, float64(rightPanelX()), 76, barW, 14, bg)
	ebitenutil.DrawRect(screen, float64(rightPanelX()), 76, filled, 14, fg)
}

func (g *Game) doomBarColors() (bg color.RGBA, fg color.RGBA) {
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
	y := 110
	drawUIText(screen, g.phaseLabel(gs.GamePhase), rightPanelX(), y, color.White)
	y += 14

	for i, pid := range gs.TurnOrder {
		p, ok := gs.Players[pid]
		if !ok {
			continue
		}
		col := playerColours[i%len(playerColours)]
		label := g.playerPanelLabel(pid, gs.CurrentPlayer, myID, p)
		label = trimToWidth(label, 360)

		// Draw background highlight for current player.
		if pid == gs.CurrentPlayer {
			ebitenutil.DrawRect(screen, float64(rightPanelX()-2), float64(y-1), 370, 13,
				color.RGBA{R: col.R / 4, G: col.G / 4, B: col.B / 4, A: 255})
		}
		drawUIText(screen, label, rightPanelX(), y, color.White)
		y += 14
	}

	if len(gs.TurnOrder) == 0 {
		drawUIText(screen, "Waiting for players...", rightPanelX(), y, color.White)
	}
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
	hp, sn, cl := "HP", "SN", "CL"
	if g.icons != nil {
		hp = g.icons.Get(ui.IconHealth)
		sn = g.icons.Get(ui.IconSanity)
		cl = g.icons.Get(ui.IconClues)
	}
	return marker + " " + pid + me + "  " + hp + ":" + strconv.Itoa(p.Resources.Health) +
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

// drawInputHints renders the keyboard shortcut legend in the lower-left corner.
func (g *Game) drawInputHints(screen *ebiten.Image, gs ebclient.GameState, myID string) {
	y := bottomPanelY()
	drawUIText(screen, "-- Controls --", 10, y, color.White)
	y += 12

	isMyTurn := gs.CurrentPlayer == myID && gs.GamePhase == "playing"
	drawUIText(screen, g.turnStatusLabel(gs, myID, isMyTurn), 10, y, color.White)
	y += 12

	hints := []string{
		"1 Move→Downtown  2 Move→University",
		"3 Move→Rivertown 4 Move→Northside",
		"G Gather  I Investigate  W Ward",
		"[ / ] Rotate Camera  V Toggle Top-Down",
		"H Skip Tutorial  ENTER Advance Tutorial",
	}
	for _, h := range hints {
		drawUIText(screen, h, 10, y, color.RGBA{R: 220, G: 220, B: 220, A: 255})
		y += 12
	}

	metrics := g.state.UXMetrics()
	drawUIText(screen, g.firstActionStatusText(metrics), 10, y, color.RGBA{R: 180, G: 220, B: 255, A: 255})
	y += 12
	drawUIText(screen, "Invalid retries: "+strconv.Itoa(metrics.InvalidActionRetries), 10, y,
		color.RGBA{R: 255, G: 200, B: 170, A: 255})
	y += 12
	drawUIText(screen, g.cameraStatusText(), 10, y, color.RGBA{R: 200, G: 220, B: 255, A: 255})
	g.drawEndBanner(screen, gs)
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
