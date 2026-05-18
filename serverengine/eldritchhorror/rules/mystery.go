package rules

import "fmt"

// Mystery represents a multi-stage objective in Eldritch Horror.
// Mysteries require worldwide cooperation to solve and are the primary
// win condition. Each mystery has 3 stages that must be completed in order.
type Mystery struct {
	ID          string
	Name        string
	Description string
	Stages      []MysteryStage
	Reward      MysteryReward
}

// MysteryStage represents one stage of a mystery objective.
// Investigators must complete tasks and meet requirements to advance.
type MysteryStage struct {
	StageNumber  int
	Description  string
	Requirements MysteryRequirements
	Completed    bool
}

// MysteryRequirements defines what's needed to complete a mystery stage.
type MysteryRequirements struct {
	// CluesRequired is the number of clues that must be spent to advance.
	CluesRequired int

	// LocationRequired specifies a city where the clue must be spent (optional).
	// Empty string means any city is valid.
	LocationRequired City

	// InvestigatorsRequired is the minimum number of investigators who must
	// participate in completing this stage (for cooperative stages).
	InvestigatorsRequired int

	// EncountersRequired is the number of encounters that must be successfully
	// resolved before this stage can complete (optional).
	EncountersRequired int
}

// MysteryReward defines bonuses granted when a mystery is solved.
type MysteryReward struct {
	Description string
	ElderSigns  int // Elder Sign tokens granted (can be used to seal gates)
	GatesClosed int // Number of gates automatically closed
}

// MysteryDeck manages the active mystery and completed mysteries.
type MysteryDeck struct {
	ActiveMystery      *Mystery
	CompletedMysteries []Mystery
	MysteriesToSolve   int // Number of mysteries required to win (usually 3)
}

// NewMysteryDeck creates a mystery deck with the specified win condition.
func NewMysteryDeck(mysteriesToSolve int) *MysteryDeck {
	return &MysteryDeck{
		ActiveMystery:      nil,
		CompletedMysteries: []Mystery{},
		MysteriesToSolve:   mysteriesToSolve,
	}
}

// ActivateMystery sets the current active mystery.
func (md *MysteryDeck) ActivateMystery(mystery Mystery) error {
	if md.ActiveMystery != nil {
		return fmt.Errorf("cannot activate mystery: another mystery is already active")
	}
	md.ActiveMystery = &mystery
	return nil
}

// GetCurrentStage returns the first incomplete stage of the active mystery.
// Returns nil if no mystery is active or all stages are complete.
func (md *MysteryDeck) GetCurrentStage() *MysteryStage {
	if md.ActiveMystery == nil {
		return nil
	}
	for i := range md.ActiveMystery.Stages {
		if !md.ActiveMystery.Stages[i].Completed {
			return &md.ActiveMystery.Stages[i]
		}
	}
	return nil
}

// CompleteCurrentStage marks the current mystery stage as complete.
// Returns an error if no stage is available to complete.
func (md *MysteryDeck) CompleteCurrentStage() error {
	stage := md.GetCurrentStage()
	if stage == nil {
		return fmt.Errorf("no incomplete stage to complete")
	}
	stage.Completed = true
	return nil
}

// IsMysteryComplete checks if all stages of the active mystery are done.
func (md *MysteryDeck) IsMysteryComplete() bool {
	if md.ActiveMystery == nil {
		return false
	}
	for _, stage := range md.ActiveMystery.Stages {
		if !stage.Completed {
			return false
		}
	}
	return true
}

// FinalizeMystery moves the active mystery to completed and clears it.
// Returns the completed mystery and its reward.
func (md *MysteryDeck) FinalizeMystery() (*Mystery, error) {
	if md.ActiveMystery == nil {
		return nil, fmt.Errorf("no active mystery to finalize")
	}
	if !md.IsMysteryComplete() {
		return nil, fmt.Errorf("cannot finalize incomplete mystery")
	}

	completed := *md.ActiveMystery
	md.CompletedMysteries = append(md.CompletedMysteries, completed)
	md.ActiveMystery = nil
	return &completed, nil
}

// IsVictoryConditionMet checks if enough mysteries have been solved to win.
func (md *MysteryDeck) IsVictoryConditionMet() bool {
	return len(md.CompletedMysteries) >= md.MysteriesToSolve
}

// GetProgress returns the number of mysteries solved and required.
func (md *MysteryDeck) GetProgress() (solved, required int) {
	return len(md.CompletedMysteries), md.MysteriesToSolve
}

// CanSpendClues checks if the requirements allow spending clues at a location.
func (req MysteryRequirements) CanSpendClues(location City, cluesAvailable int) bool {
	if cluesAvailable < req.CluesRequired {
		return false
	}
	if req.LocationRequired != "" && req.LocationRequired != location {
		return false
	}
	return true
}

// String implements Stringer for Mystery.
func (m Mystery) String() string {
	return fmt.Sprintf("Mystery: %s (%d stages)", m.Name, len(m.Stages))
}
