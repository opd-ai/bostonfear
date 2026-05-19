package phases

import (
	"testing"

	"github.com/opd-ai/bostonfear/protocol"
)

// makeGameState creates a minimal GameState for tests.
func makeGameState(players map[string]*protocol.Player, turnOrder []string, current string) *protocol.GameState {
	return &protocol.GameState{
		Players:            players,
		TurnOrder:          turnOrder,
		CurrentPlayer:      current,
		LocationDoomTokens: map[string]int{},
		MythosEventDeck:    []protocol.MythosEvent{},
		MythosEvents:       []protocol.MythosEvent{},
		ActiveEvents:       []string{},
		GamePhase:          gamePhasePlaying,
	}
}

func connectedPlayer() *protocol.Player {
	return &protocol.Player{
		Connected:        true,
		Defeated:         false,
		ActionsRemaining: 2,
	}
}

// --- AdvanceTurn tests ---

func TestAdvanceTurn_RotatesToNextPlayer(t *testing.T) {
	p1 := connectedPlayer()
	p2 := connectedPlayer()
	state := makeGameState(
		map[string]*protocol.Player{"p1": p1, "p2": p2},
		[]string{"p1", "p2"},
		"p1",
	)

	AdvanceTurn(state, TurnCallbacks{})

	if state.CurrentPlayer != "p2" {
		t.Errorf("expected CurrentPlayer p2, got %s", state.CurrentPlayer)
	}
	if p2.ActionsRemaining != 2 {
		t.Errorf("expected p2 ActionsRemaining 2, got %d", p2.ActionsRemaining)
	}
}

func TestAdvanceTurn_WrapsAroundAndTriggersMyths(t *testing.T) {
	p1 := connectedPlayer()
	p2 := connectedPlayer()
	state := makeGameState(
		map[string]*protocol.Player{"p1": p1, "p2": p2},
		[]string{"p1", "p2"},
		"p2",
	)

	mythosRan := false
	cb := TurnCallbacks{RunMythosPhase: func() { mythosRan = true }}

	AdvanceTurn(state, cb)

	if state.CurrentPlayer != "p1" {
		t.Errorf("expected CurrentPlayer p1 after wrap, got %s", state.CurrentPlayer)
	}
	if !mythosRan {
		t.Error("expected RunMythosPhase to be called on wrap-around")
	}
}

func TestAdvanceTurn_EmptyTurnOrder(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "p1")
	// Must not panic
	AdvanceTurn(state, TurnCallbacks{})
}

func TestAdvanceTurn_SkipsDefeatedPlayer(t *testing.T) {
	defeated := connectedPlayer()
	defeated.Defeated = true
	p3 := connectedPlayer()

	state := makeGameState(
		map[string]*protocol.Player{"p1": connectedPlayer(), "p2": defeated, "p3": p3},
		[]string{"p1", "p2", "p3"},
		"p1",
	)

	AdvanceTurn(state, TurnCallbacks{})

	if state.CurrentPlayer != "p3" {
		t.Errorf("expected CurrentPlayer p3 (skipping defeated p2), got %s", state.CurrentPlayer)
	}
}

// --- RunMythosPhase tests ---

func TestRunMythosPhase_DoomIncrementsForEventPlacement(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Doom = 3
	state.MythosEventsPerRound = 1
	state.MythosEventDeck = []protocol.MythosEvent{
		{LocationID: "Downtown", MythosEventType: mythosEventBlank, Effect: "test"},
	}

	RunMythosPhase(state, MythosCallbacks{})

	if state.Doom != 4 {
		t.Errorf("expected doom 4 after event placement, got %d", state.Doom)
	}
	if len(state.MythosEvents) != 1 {
		t.Errorf("expected 1 mythos event, got %d", len(state.MythosEvents))
	}
}

func TestRunMythosPhase_DoomCapAt12(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Doom = 11
	state.MythosEventsPerRound = 2
	state.MythosEventDeck = []protocol.MythosEvent{
		{LocationID: "Downtown", MythosEventType: mythosEventBlank, Effect: "e1"},
		{LocationID: "University", MythosEventType: mythosEventBlank, Effect: "e2"},
	}

	RunMythosPhase(state, MythosCallbacks{})

	if state.Doom > 12 {
		t.Errorf("doom must not exceed 12, got %d", state.Doom)
	}
}

func TestRunMythosPhase_EmptyEventDeckUsesDefault(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Doom = 0
	state.MythosEventsPerRound = 1

	defaultCalled := false
	cb := MythosCallbacks{
		DefaultEventDeck: func() []protocol.MythosEvent {
			defaultCalled = true
			return []protocol.MythosEvent{
				{LocationID: "Northside", MythosEventType: mythosEventBlank, Effect: "e"},
			}
		},
	}

	RunMythosPhase(state, cb)

	if !defaultCalled {
		t.Error("expected DefaultEventDeck callback to be called when deck is empty")
	}
}

func TestRunMythosPhase_GamePhaseRestoredToPlaying(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.GamePhase = gamePhasePlaying

	RunMythosPhase(state, MythosCallbacks{})

	if state.GamePhase != gamePhasePlaying {
		t.Errorf("expected GamePhase %q after RunMythosPhase, got %q", gamePhasePlaying, state.GamePhase)
	}
}

// mythosEventBlank is a stand-in event type for tests that do not need a specific type.
const mythosEventBlank = "blank_event"

// --- ResolveEventEffect tests ---

func TestResolveEventEffect_Anomaly(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	evt := protocol.MythosEvent{
		LocationID:      "Downtown",
		MythosEventType: mythosEventAnomaly,
	}

	spawned := ""
	cb := EventCallbacks{
		SpawnAnomaly: func(locID string) {
			spawned = locID
		},
	}

	ResolveEventEffect(state, evt, cb)

	if spawned != "Downtown" {
		t.Errorf("expected SpawnAnomaly called with Downtown, got %s", spawned)
	}
}

func TestResolveEventEffect_FogMadness(t *testing.T) {
	p1 := connectedPlayer()
	p1.Resources.Sanity = 5
	p2 := connectedPlayer()
	p2.Resources.Sanity = 3
	defeated := connectedPlayer()
	defeated.Defeated = true
	defeated.Resources.Sanity = 4

	state := makeGameState(
		map[string]*protocol.Player{"p1": p1, "p2": p2, "p3": defeated},
		[]string{"p1", "p2", "p3"},
		"p1",
	)

	evt := protocol.MythosEvent{MythosEventType: mythosEventFogMadness}
	ResolveEventEffect(state, evt, EventCallbacks{})

	if p1.Resources.Sanity != 4 {
		t.Errorf("expected p1 sanity 4, got %d", p1.Resources.Sanity)
	}
	if p2.Resources.Sanity != 2 {
		t.Errorf("expected p2 sanity 2, got %d", p2.Resources.Sanity)
	}
	if defeated.Resources.Sanity != 4 {
		t.Errorf("expected defeated p3 sanity unchanged at 4, got %d", defeated.Resources.Sanity)
	}
}

func TestResolveEventEffect_ClueDrought(t *testing.T) {
	p1 := connectedPlayer()
	p1.Resources.Clues = 3
	p2 := connectedPlayer()
	p2.Resources.Clues = 0

	state := makeGameState(
		map[string]*protocol.Player{"p1": p1, "p2": p2},
		[]string{"p1", "p2"},
		"p1",
	)

	evt := protocol.MythosEvent{MythosEventType: mythosEventClueDrought}
	ResolveEventEffect(state, evt, EventCallbacks{})

	if p1.Resources.Clues != 2 {
		t.Errorf("expected p1 clues 2, got %d", p1.Resources.Clues)
	}
	if p2.Resources.Clues != 0 {
		t.Errorf("expected p2 clues 0 (already at min), got %d", p2.Resources.Clues)
	}
}

func TestResolveEventEffect_DoomSpread(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Doom = 5
	state.OpenGates = []protocol.Gate{
		{ID: "gate1", Location: protocol.Downtown},
		{ID: "gate2", Location: protocol.University},
	}

	evt := protocol.MythosEvent{MythosEventType: mythosEventDoomSpread}
	ResolveEventEffect(state, evt, EventCallbacks{})

	if state.Doom != 7 {
		t.Errorf("expected doom 7 (5+2 gates), got %d", state.Doom)
	}
}

func TestResolveEventEffect_DoomSpreadNoGates(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Doom = 5
	state.OpenGates = []protocol.Gate{}

	evt := protocol.MythosEvent{MythosEventType: mythosEventDoomSpread}
	ResolveEventEffect(state, evt, EventCallbacks{})

	if state.Doom != 6 {
		t.Errorf("expected doom 6 (minimum increment 1), got %d", state.Doom)
	}
}

func TestResolveEventEffect_Resurgence(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Enemies = map[string]*protocol.Enemy{
		"e1": {ID: "e1", Health: 3, MaxHealth: 5, Engaged: []string{"p1"}},
		"e2": {ID: "e2", Health: 4, MaxHealth: 4, Engaged: []string{}},
	}

	evt := protocol.MythosEvent{MythosEventType: mythosEventResurgence}
	ResolveEventEffect(state, evt, EventCallbacks{})

	if state.Enemies["e1"].Health != 4 {
		t.Errorf("expected engaged enemy health 4, got %d", state.Enemies["e1"].Health)
	}
	if state.Enemies["e2"].Health != 4 {
		t.Errorf("expected unengaged enemy health unchanged at 4, got %d", state.Enemies["e2"].Health)
	}
}

// --- DrawMythosToken tests ---

func TestDrawMythosToken_ReturnsValidToken(t *testing.T) {
	validTokens := map[string]bool{
		mythosTokenDoom:     true,
		mythosTokenBlessing: true,
		mythosTokenCurse:    true,
		mythosTokenBlank:    true,
	}

	for i := 0; i < 20; i++ {
		token := DrawMythosToken()
		if !validTokens[token] {
			t.Errorf("DrawMythosToken returned invalid token: %s", token)
		}
	}
}

// --- ResolveMythosToken tests ---

func TestResolveMythosToken_Doom(t *testing.T) {
	state := makeGameState(map[string]*protocol.Player{}, []string{}, "")
	state.Doom = 5

	ResolveMythosToken(state, mythosTokenDoom, TokenCallbacks{})

	if state.Doom != 6 {
		t.Errorf("expected doom 6, got %d", state.Doom)
	}
}

func TestResolveMythosToken_Blessing(t *testing.T) {
	p1 := connectedPlayer()
	p1.Resources.Health = 8

	state := makeGameState(
		map[string]*protocol.Player{"p1": p1},
		[]string{"p1"},
		"p1",
	)

	cb := TokenCallbacks{MaxHealth: 10}
	ResolveMythosToken(state, mythosTokenBlessing, cb)

	if p1.Resources.Health != 9 {
		t.Errorf("expected p1 health 9, got %d", p1.Resources.Health)
	}
}

func TestResolveMythosToken_BlessingCapped(t *testing.T) {
	p1 := connectedPlayer()
	p1.Resources.Health = 10

	state := makeGameState(
		map[string]*protocol.Player{"p1": p1},
		[]string{"p1"},
		"p1",
	)

	cb := TokenCallbacks{MaxHealth: 10}
	ResolveMythosToken(state, mythosTokenBlessing, cb)

	if p1.Resources.Health != 10 {
		t.Errorf("expected p1 health capped at 10, got %d", p1.Resources.Health)
	}
}

func TestResolveMythosToken_Curse(t *testing.T) {
	p1 := connectedPlayer()
	p1.Resources.Sanity = 5

	state := makeGameState(
		map[string]*protocol.Player{"p1": p1},
		[]string{"p1"},
		"p1",
	)

	defeatChecked := false
	cb := TokenCallbacks{
		CheckInvestigatorDefeat: func(pid string) {
			defeatChecked = true
			if pid != "p1" {
				t.Errorf("expected defeat check for p1, got %s", pid)
			}
		},
	}

	ResolveMythosToken(state, mythosTokenCurse, cb)

	if p1.Resources.Sanity != 4 {
		t.Errorf("expected p1 sanity 4, got %d", p1.Resources.Sanity)
	}
	if !defeatChecked {
		t.Error("expected CheckInvestigatorDefeat to be called")
	}
}

func TestResolveMythosToken_CurseFloored(t *testing.T) {
	p1 := connectedPlayer()
	p1.Resources.Sanity = 0

	state := makeGameState(
		map[string]*protocol.Player{"p1": p1},
		[]string{"p1"},
		"p1",
	)

	ResolveMythosToken(state, mythosTokenCurse, TokenCallbacks{})

	if p1.Resources.Sanity != 0 {
		t.Errorf("expected p1 sanity floored at 0, got %d", p1.Resources.Sanity)
	}
}
