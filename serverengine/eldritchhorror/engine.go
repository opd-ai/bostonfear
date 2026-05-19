package eldritchhorror

import (
	"log"

	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/model"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/phases"
)

// Engine is the Eldritch Horror module-owned runtime wrapper.
// It provides a functional server loop with Eldritch Horror-specific game mechanics
// including monster movement phases between player turns.
type Engine struct {
	*serverengine.GameServer
	eldritchState *model.EldritchGameState
}

// InitializeEldritchState sets up the Eldritch Horror-specific game state.
// This should be called after engine creation and before starting the game.
func (e *Engine) InitializeEldritchState() {
	e.eldritchState = &model.EldritchGameState{
		Investigators: make(map[string]model.InvestigatorState),
		Monsters:      []model.MonsterState{},
		Gates:         []model.GateState{},
	}
}

// runMonsterPhase executes monster movement between player turns.
// This method is registered as a post-turn callback with the base GameServer.
// It must not block or acquire additional locks to avoid deadlock.
func (e *Engine) runMonsterPhase() {
	// Skip if state is not initialized or no monsters present
	if e.eldritchState == nil || len(e.eldritchState.Monsters) == 0 {
		return
	}

	monsterPhase := phases.NewMonsterMovementPhase(e.eldritchState)
	events, err := monsterPhase.Execute()
	if err != nil {
		log.Printf("Eldritch Horror monster phase error: %v", err)
		return
	}

	// Log monster movement events
	for _, event := range events {
		log.Printf("Monster movement: %s", event.String())
	}
}
