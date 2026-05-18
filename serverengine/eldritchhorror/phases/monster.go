package phases

import (
	"fmt"

	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/model"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror/rules"
)

// MonsterMovementPhase handles monster movement between player turns.
// In Eldritch Horror, monsters move toward investigators at the end of each round.
// This phase is executed after all investigators have completed their actions.
type MonsterMovementPhase struct {
	gameState *model.EldritchGameState
}

// NewMonsterMovementPhase creates a new monster movement phase handler.
func NewMonsterMovementPhase(gameState *model.EldritchGameState) *MonsterMovementPhase {
	return &MonsterMovementPhase{
		gameState: gameState,
	}
}

// Execute runs the monster movement phase.
// Returns a list of movement events and any errors encountered.
func (p *MonsterMovementPhase) Execute() ([]MonsterMovementEvent, error) {
	if p.gameState == nil {
		return nil, fmt.Errorf("game state is nil")
	}

	events := []MonsterMovementEvent{}

	// Process each monster on the board
	for i := range p.gameState.Monsters {
		monster := &p.gameState.Monsters[i]

		// Skip engaged monsters (they don't move)
		if monster.IsEngaged {
			continue
		}

		// Find nearest investigator
		targetCity, targetInvestigator := p.findNearestInvestigator(monster.Location)
		if targetCity == "" {
			// No investigators on board, monster stays in place
			continue
		}

		// Move monster toward target
		if targetCity != monster.Location {
			oldLocation := monster.Location
			monster.Location = targetCity

			events = append(events, MonsterMovementEvent{
				MonsterID:          monster.ID,
				MonsterType:        monster.Type,
				From:               oldLocation,
				To:                 targetCity,
				TargetInvestigator: targetInvestigator,
			})
		}

		// If monster reaches investigator location, engage
		if targetCity == monster.Location && targetInvestigator != "" {
			monster.IsEngaged = true
			monster.EngagedWith = targetInvestigator

			events = append(events, MonsterMovementEvent{
				MonsterID:          monster.ID,
				MonsterType:        monster.Type,
				From:               monster.Location,
				To:                 monster.Location,
				TargetInvestigator: targetInvestigator,
				EngagementOccurred: true,
			})
		}
	}

	return events, nil
}

// findNearestInvestigator finds the closest investigator to a monster.
// Returns the city where the investigator is located and the investigator ID.
// If no investigators are found, returns empty strings.
func (p *MonsterMovementPhase) findNearestInvestigator(monsterLocation rules.City) (rules.City, string) {
	// Simple heuristic: find any investigator (in real game, would calculate shortest path)
	// For initial implementation, monsters move to first investigator found

	for playerID, investigator := range p.gameState.Investigators {
		if investigator.Location != "" {
			// In a full implementation, this would use the global map to find
			// the shortest path from monsterLocation to investigator.Location
			// For now, return the first investigator found
			return investigator.Location, playerID
		}
	}

	return "", ""
}

// MonsterMovementEvent represents a single monster movement action.
type MonsterMovementEvent struct {
	MonsterID          string
	MonsterType        string
	From               rules.City
	To                 rules.City
	TargetInvestigator string
	EngagementOccurred bool
}

// String returns a human-readable description of the movement event.
func (e MonsterMovementEvent) String() string {
	if e.EngagementOccurred {
		return fmt.Sprintf("%s (type: %s) engaged investigator %s at %s",
			e.MonsterID, e.MonsterType, e.TargetInvestigator, e.To)
	}
	if e.From == e.To {
		return fmt.Sprintf("%s (type: %s) remained at %s", e.MonsterID, e.MonsterType, e.To)
	}
	return fmt.Sprintf("%s (type: %s) moved from %s to %s (tracking %s)",
		e.MonsterID, e.MonsterType, e.From, e.To, e.TargetInvestigator)
}

// SpawnMonster adds a new monster at the specified location.
// This is typically called during the mythos phase when gates open.
func (p *MonsterMovementPhase) SpawnMonster(monsterType string, location rules.City) string {
	monsterID := fmt.Sprintf("%s-%d", monsterType, len(p.gameState.Monsters)+1)

	monster := model.MonsterState{
		ID:        monsterID,
		Type:      monsterType,
		Location:  location,
		Toughness: getMonsterToughness(monsterType),
		Horror:    getMonsterHorror(monsterType),
		Damage:    getMonsterDamage(monsterType),
		IsEngaged: false,
	}

	p.gameState.Monsters = append(p.gameState.Monsters, monster)
	return monsterID
}

// RemoveMonster removes a defeated monster from the board.
func (p *MonsterMovementPhase) RemoveMonster(monsterID string) error {
	for i, monster := range p.gameState.Monsters {
		if monster.ID == monsterID {
			// Remove monster by slicing it out
			p.gameState.Monsters = append(
				p.gameState.Monsters[:i],
				p.gameState.Monsters[i+1:]...,
			)
			return nil
		}
	}
	return fmt.Errorf("monster %s not found", monsterID)
}

// Helper functions for monster stats (simplified for initial implementation)

func getMonsterToughness(monsterType string) int {
	// Default toughness values by monster type
	toughnessMap := map[string]int{
		"cultist":   1,
		"zombie":    2,
		"ghoul":     2,
		"deepOne":   3,
		"starSpawn": 4,
	}
	if val, ok := toughnessMap[monsterType]; ok {
		return val
	}
	return 2 // default
}

func getMonsterHorror(monsterType string) int {
	// Default horror values by monster type
	horrorMap := map[string]int{
		"cultist":   0,
		"zombie":    1,
		"ghoul":     1,
		"deepOne":   2,
		"starSpawn": 3,
	}
	if val, ok := horrorMap[monsterType]; ok {
		return val
	}
	return 1 // default
}

func getMonsterDamage(monsterType string) int {
	// Default damage values by monster type
	damageMap := map[string]int{
		"cultist":   1,
		"zombie":    1,
		"ghoul":     2,
		"deepOne":   2,
		"starSpawn": 3,
	}
	if val, ok := damageMap[monsterType]; ok {
		return val
	}
	return 1 // default
}
