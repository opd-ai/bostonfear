package rules

import "fmt"

// Monster represents a creature spawned by gates or mythos effects.
type Monster struct {
	ID          string
	Name        string
	Description string

	// Combat attributes
	Horror    int // Sanity loss when encountering
	Damage    int // Health loss in combat
	Toughness int // Number of successes needed to defeat

	// Spawning attributes
	SpawnCity City // Where the monster spawns (or "" for any gate)
	IsElusive bool // Can be ignored with successful evade
	IsEpic    bool // Requires multiple investigators to defeat
}

// Gate represents a portal to another dimension spawning monsters.
type Gate struct {
	ID       string
	Location City
	IsOpen   bool

	// Gate state
	MonsterCount int  // Number of monsters spawned at this gate
	IsSealed     bool // Whether the gate has been permanently sealed
	ClueTokens   int  // Number of clue tokens spent on closing attempts
}

// MonsterSpawningRules defines how monsters appear on the board.
type MonsterSpawningRules struct {
	MaxMonstersPerGate int // Typically 1-2 monsters per gate
	SurgeThreshold     int // Number of gates that triggers monster surge
}

// DefaultMonsterSpawningRules returns standard Eldritch Horror spawning rules.
func DefaultMonsterSpawningRules() MonsterSpawningRules {
	return MonsterSpawningRules{
		MaxMonstersPerGate: 1,
		SurgeThreshold:     6, // Surge when 6+ gates open
	}
}

// MonsterPool tracks all active monsters and gates on the board.
type MonsterPool struct {
	Monsters []Monster
	Gates    []Gate
	Rules    MonsterSpawningRules
}

// NewMonsterPool creates an empty monster pool with default rules.
func NewMonsterPool() *MonsterPool {
	return &MonsterPool{
		Monsters: []Monster{},
		Gates:    []Gate{},
		Rules:    DefaultMonsterSpawningRules(),
	}
}

// OpenGate creates a new gate at the specified city.
func (mp *MonsterPool) OpenGate(city City) (*Gate, error) {
	// Check if gate already exists at this city
	for _, gate := range mp.Gates {
		if gate.Location == city && gate.IsOpen {
			return nil, fmt.Errorf("gate already open at %s", city)
		}
	}

	gate := Gate{
		ID:           fmt.Sprintf("gate-%s", city),
		Location:     city,
		IsOpen:       true,
		MonsterCount: 0,
		IsSealed:     false,
		ClueTokens:   0,
	}
	mp.Gates = append(mp.Gates, gate)

	// Spawn initial monster at new gate
	// Get pointer to the gate in the slice
	gateIndex := len(mp.Gates) - 1
	mp.SpawnMonsterAtGate(&mp.Gates[gateIndex])

	return &mp.Gates[gateIndex], nil
}

// CloseGate removes a gate from the specified city.
func (mp *MonsterPool) CloseGate(city City) error {
	for i, gate := range mp.Gates {
		if gate.Location == city && gate.IsOpen {
			mp.Gates[i].IsOpen = false
			return nil
		}
	}
	return fmt.Errorf("no open gate at %s", city)
}

// SealGate permanently closes a gate (requires Elder Signs).
func (mp *MonsterPool) SealGate(city City) error {
	for i, gate := range mp.Gates {
		if gate.Location == city && gate.IsOpen {
			mp.Gates[i].IsOpen = false
			mp.Gates[i].IsSealed = true
			return nil
		}
	}
	return fmt.Errorf("no open gate at %s", city)
}

// SpawnMonsterAtGate adds a monster to the specified gate.
func (mp *MonsterPool) SpawnMonsterAtGate(gate *Gate) error {
	if gate.MonsterCount >= mp.Rules.MaxMonstersPerGate {
		return fmt.Errorf("gate at %s already has maximum monsters", gate.Location)
	}

	// Create a generic monster (in full implementation, would draw from deck)
	monster := Monster{
		ID:        fmt.Sprintf("monster-%d", len(mp.Monsters)),
		Name:      "Eldritch Horror",
		SpawnCity: gate.Location,
		Horror:    1,
		Damage:    1,
		Toughness: 1,
		IsElusive: false,
		IsEpic:    false,
	}

	mp.Monsters = append(mp.Monsters, monster)
	gate.MonsterCount++
	return nil
}

// GetOpenGateCount returns the number of currently open gates.
func (mp *MonsterPool) GetOpenGateCount() int {
	count := 0
	for _, gate := range mp.Gates {
		if gate.IsOpen && !gate.IsSealed {
			count++
		}
	}
	return count
}

// ShouldTriggerMonsterSurge checks if monster surge condition is met.
func (mp *MonsterPool) ShouldTriggerMonsterSurge() bool {
	return mp.GetOpenGateCount() >= mp.Rules.SurgeThreshold
}

// TriggerMonsterSurge spawns additional monsters at all open gates.
func (mp *MonsterPool) TriggerMonsterSurge() error {
	if !mp.ShouldTriggerMonsterSurge() {
		return fmt.Errorf("surge threshold not met")
	}

	for i := range mp.Gates {
		if mp.Gates[i].IsOpen && !mp.Gates[i].IsSealed {
			// Attempt to spawn additional monster (may fail if gate at capacity)
			mp.SpawnMonsterAtGate(&mp.Gates[i])
		}
	}

	return nil
}

// GetMonstersAtCity returns all monsters currently at the specified city.
func (mp *MonsterPool) GetMonstersAtCity(city City) []Monster {
	var monsters []Monster
	for _, monster := range mp.Monsters {
		if monster.SpawnCity == city {
			monsters = append(monsters, monster)
		}
	}
	return monsters
}

// DefeatMonster removes a monster from the pool.
func (mp *MonsterPool) DefeatMonster(monsterID string) error {
	for i, monster := range mp.Monsters {
		if monster.ID == monsterID {
			// Remove monster from pool
			mp.Monsters = append(mp.Monsters[:i], mp.Monsters[i+1:]...)

			// Update gate monster count
			for j := range mp.Gates {
				if mp.Gates[j].Location == monster.SpawnCity {
					mp.Gates[j].MonsterCount--
					break
				}
			}
			return nil
		}
	}
	return fmt.Errorf("monster %s not found", monsterID)
}

// GetGateAtCity returns the gate at the specified city if one exists.
func (mp *MonsterPool) GetGateAtCity(city City) (*Gate, error) {
	for i := range mp.Gates {
		if mp.Gates[i].Location == city && mp.Gates[i].IsOpen {
			return &mp.Gates[i], nil
		}
	}
	return nil, fmt.Errorf("no open gate at %s", city)
}

// CombatResult represents the outcome of monster combat.
type CombatResult struct {
	Success      bool
	MonsterID    string
	HorrorLoss   int // Sanity lost
	DamageTaken  int // Health lost
	MonsterSlain bool
}

// ResolveCombat executes combat between an investigator and a monster.
// successCount is the number of successes rolled in the combat test.
func ResolveCombat(monster Monster, successCount int) CombatResult {
	result := CombatResult{
		MonsterID:    monster.ID,
		HorrorLoss:   monster.Horror,
		DamageTaken:  0,
		MonsterSlain: false,
	}

	// Check if investigator achieved enough successes
	if successCount >= monster.Toughness {
		result.Success = true
		result.MonsterSlain = true
		result.DamageTaken = 0 // No damage if monster is defeated
	} else {
		result.Success = false
		result.MonsterSlain = false
		result.DamageTaken = monster.Damage // Take damage on failure
	}

	return result
}
