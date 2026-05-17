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
