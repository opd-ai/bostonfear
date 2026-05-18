package rules

import "fmt"

// Resources represents investigator resources in Eldritch Horror.
// Uses the same bounds as Arkham Horror (Health 1-10, Sanity 1-10)
// but with different acquisition and loss mechanics.
type Resources struct {
	Health     int // Current health (1-10)
	MaxHealth  int // Maximum health capacity
	Sanity     int // Current sanity (1-10)
	MaxSanity  int // Maximum sanity capacity
	Clues      int // Clue tokens (0+, no upper bound)
	Money      int // Money resources for trading
	Tickets    int // Travel tickets (train/ship)
	ElderSigns int // Elder Sign tokens for sealing gates
}

// NewResources creates a resource pool with specified maximums.
func NewResources(maxHealth, maxSanity int) Resources {
	return Resources{
		Health:     maxHealth,
		MaxHealth:  maxHealth,
		Sanity:     maxSanity,
		MaxSanity:  maxSanity,
		Clues:      0,
		Money:      0,
		Tickets:    0,
		ElderSigns: 0,
	}
}

// RestoreHealth increases health up to maximum.
func (r *Resources) RestoreHealth(amount int) int {
	if amount <= 0 {
		return 0
	}
	before := r.Health
	r.Health += amount
	if r.Health > r.MaxHealth {
		r.Health = r.MaxHealth
	}
	return r.Health - before
}

// LoseHealth decreases health.
// Returns true if investigator is eliminated (health reaches 0).
func (r *Resources) LoseHealth(amount int) bool {
	if amount <= 0 {
		return false
	}
	r.Health -= amount
	if r.Health < 0 {
		r.Health = 0
	}
	return r.Health == 0
}

// RestoreSanity increases sanity up to maximum.
func (r *Resources) RestoreSanity(amount int) int {
	if amount <= 0 {
		return 0
	}
	before := r.Sanity
	r.Sanity += amount
	if r.Sanity > r.MaxSanity {
		r.Sanity = r.MaxSanity
	}
	return r.Sanity - before
}

// LoseSanity decreases sanity.
// Returns true if investigator is eliminated (sanity reaches 0).
func (r *Resources) LoseSanity(amount int) bool {
	if amount <= 0 {
		return false
	}
	r.Sanity -= amount
	if r.Sanity < 0 {
		r.Sanity = 0
	}
	return r.Sanity == 0
}

// GainClues adds clue tokens.
func (r *Resources) GainClues(amount int) {
	if amount > 0 {
		r.Clues += amount
	}
}

// SpendClues removes clue tokens.
// Returns error if insufficient clues.
func (r *Resources) SpendClues(amount int) error {
	if amount < 0 {
		return fmt.Errorf("cannot spend negative clues")
	}
	if r.Clues < amount {
		return fmt.Errorf("insufficient clues: have %d, need %d", r.Clues, amount)
	}
	r.Clues -= amount
	return nil
}

// GainMoney adds money resources.
func (r *Resources) GainMoney(amount int) {
	if amount > 0 {
		r.Money += amount
	}
}

// SpendMoney removes money resources.
// Returns error if insufficient money.
func (r *Resources) SpendMoney(amount int) error {
	if amount < 0 {
		return fmt.Errorf("cannot spend negative money")
	}
	if r.Money < amount {
		return fmt.Errorf("insufficient money: have %d, need %d", r.Money, amount)
	}
	r.Money -= amount
	return nil
}

// GainTickets adds travel tickets.
func (r *Resources) GainTickets(amount int) {
	if amount > 0 {
		r.Tickets += amount
	}
}

// SpendTicket removes one travel ticket.
// Returns error if no tickets available.
func (r *Resources) SpendTicket() error {
	if r.Tickets <= 0 {
		return fmt.Errorf("no tickets available")
	}
	r.Tickets--
	return nil
}

// GainElderSign adds an Elder Sign token.
func (r *Resources) GainElderSign() {
	r.ElderSigns++
}

// SpendElderSign removes an Elder Sign token.
// Returns error if none available.
func (r *Resources) SpendElderSign() error {
	if r.ElderSigns <= 0 {
		return fmt.Errorf("no elder signs available")
	}
	r.ElderSigns--
	return nil
}

// IsEliminated checks if investigator has been eliminated (0 health or sanity).
func (r *Resources) IsEliminated() bool {
	return r.Health == 0 || r.Sanity == 0
}

// GetHealthRatio returns current health as a fraction of maximum.
func (r *Resources) GetHealthRatio() float64 {
	if r.MaxHealth == 0 {
		return 0
	}
	return float64(r.Health) / float64(r.MaxHealth)
}

// GetSanityRatio returns current sanity as a fraction of maximum.
func (r *Resources) GetSanityRatio() float64 {
	if r.MaxSanity == 0 {
		return 0
	}
	return float64(r.Sanity) / float64(r.MaxSanity)
}

// RestAction performs a rest action to recover resources.
// Different cities provide different rest benefits.
type RestAction struct {
	HealthRestore int
	SanityRestore int
}

// DefaultRestAction returns standard rest recovery values.
func DefaultRestAction() RestAction {
	return RestAction{
		HealthRestore: 2,
		SanityRestore: 2,
	}
}

// CityRestAction returns rest benefits for a specific city.
// Some cities (with hospitals, lodges, etc.) provide better recovery.
func CityRestAction(city City) RestAction {
	switch city {
	case CityArkham, CityLondon, CityTokyo:
		// Major cities with good facilities
		return RestAction{HealthRestore: 3, SanityRestore: 3}
	case CityCairo, CityIstanbul, CityShanghai:
		// Cities with moderate facilities
		return RestAction{HealthRestore: 2, SanityRestore: 3}
	case CityAntarctica, CityAmazon, CityHimalayas:
		// Remote expedition sites with poor facilities
		return RestAction{HealthRestore: 1, SanityRestore: 1}
	default:
		return DefaultRestAction()
	}
}

// ApplyRestAction applies rest benefits to resources.
func (r *Resources) ApplyRestAction(action RestAction) {
	r.RestoreHealth(action.HealthRestore)
	r.RestoreSanity(action.SanityRestore)
}

// String implements Stringer for Resources.
func (r Resources) String() string {
	return fmt.Sprintf("Health: %d/%d, Sanity: %d/%d, Clues: %d, Money: %d, Tickets: %d, Elder Signs: %d",
		r.Health, r.MaxHealth, r.Sanity, r.MaxSanity, r.Clues, r.Money, r.Tickets, r.ElderSigns)
}
