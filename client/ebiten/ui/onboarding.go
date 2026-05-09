package ui

import (
	"time"
)

// OnboardingScript defines a sequence of onboarding hints and checkpoints.
type OnboardingScript struct {
	Steps []OnboardingStep
}

// OnboardingStep represents one hint or tutorial checkpoint.
type OnboardingStep struct {
	ID          string        // Unique identifier for this step.
	Title       string        // Display title.
	Description string        // Longer explanation.
	Duration    time.Duration // How long to display (0 = manual advance).
	Action      string        // Optional action required to advance (none = auto-advance).
	Highlight   *Highlight    // Optional UI element to highlight.
}

// Highlight defines an area to visually emphasize during onboarding.
type Highlight struct {
	X             float64
	Y             float64
	Width         float64
	Height        float64
	Color         [4]uint8 // RGBA
	PulseDuration time.Duration
	ArrowTarget   string // "none", "up", "down", "left", "right"
	ArrowOffsetX  float64
	ArrowOffsetY  float64
}

// OnboardingController manages playback and progression through a script.
type OnboardingController struct {
	script        *OnboardingScript
	currentStep   int
	isActive      bool
	isCompleted   bool
	stepStartTime time.Time
	visited       map[string]bool // Tracks visited steps.
	canReplay     bool
	onStepChange  func(step *OnboardingStep)
	onComplete    func()
}

// NewOnboardingController creates a controller for the given script.
func NewOnboardingController(script *OnboardingScript) *OnboardingController {
	if script == nil {
		script = &OnboardingScript{}
	}
	return &OnboardingController{
		script:      script,
		currentStep: 0,
		isActive:    false,
		isCompleted: false,
		visited:     make(map[string]bool),
		canReplay:   true,
	}
}

// Start begins playback of the onboarding sequence.
func (oc *OnboardingController) Start() {
	if oc == nil || len(oc.script.Steps) == 0 {
		return
	}
	oc.isActive = true
	oc.isCompleted = false
	oc.currentStep = 0
	oc.stepStartTime = time.Now()
	oc.notifyStepChange()
}

// Skip ends onboarding immediately.
func (oc *OnboardingController) Skip() {
	if oc == nil {
		return
	}
	oc.isActive = false
	oc.isCompleted = true
	if oc.onComplete != nil {
		oc.onComplete()
	}
}

// AdvanceStep moves to the next step (if action requirement is met).
func (oc *OnboardingController) AdvanceStep() {
	if oc == nil || !oc.isActive {
		return
	}
	oc.currentStep++
	if oc.currentStep >= len(oc.script.Steps) {
		// Script finished.
		oc.isActive = false
		oc.isCompleted = true
		if oc.onComplete != nil {
			oc.onComplete()
		}
		return
	}
	oc.stepStartTime = time.Now()
	oc.notifyStepChange()
}

// OnStepChange registers a callback for when the current step changes.
func (oc *OnboardingController) OnStepChange(fn func(step *OnboardingStep)) *OnboardingController {
	oc.onStepChange = fn
	return oc
}

// OnComplete registers a callback for when onboarding finishes.
func (oc *OnboardingController) OnComplete(fn func()) *OnboardingController {
	oc.onComplete = fn
	return oc
}

// notifyStepChange fires the step-change callback.
func (oc *OnboardingController) notifyStepChange() {
	if oc.onStepChange != nil && oc.currentStep < len(oc.script.Steps) {
		step := oc.script.Steps[oc.currentStep]
		oc.visited[step.ID] = true
		oc.onStepChange(&step)
	}
}

// Update advances time-based steps. Returns true if onboarding is complete.
func (oc *OnboardingController) Update() bool {
	if oc == nil || !oc.isActive || oc.currentStep >= len(oc.script.Steps) {
		return oc != nil && oc.isCompleted
	}

	step := oc.script.Steps[oc.currentStep]

	// If step has a duration and no action requirement, auto-advance.
	if step.Duration > 0 && step.Action == "" {
		if time.Since(oc.stepStartTime) >= step.Duration {
			oc.AdvanceStep()
		}
	}

	return !oc.isActive
}

// IsActive reports whether onboarding is currently being shown.
func (oc *OnboardingController) IsActive() bool {
	return oc != nil && oc.isActive
}

// IsCompleted reports whether onboarding has finished.
func (oc *OnboardingController) IsCompleted() bool {
	return oc != nil && oc.isCompleted
}

// CurrentStep returns the active step, or nil if not active.
func (oc *OnboardingController) CurrentStep() *OnboardingStep {
	if oc == nil || !oc.isActive || oc.currentStep >= len(oc.script.Steps) {
		return nil
	}
	return &oc.script.Steps[oc.currentStep]
}

// Progress returns (current, total) step counts.
func (oc *OnboardingController) Progress() (int, int) {
	if oc == nil {
		return 0, 0
	}
	return oc.currentStep + 1, len(oc.script.Steps)
}

// VisitedSteps returns a map of visited step IDs.
func (oc *OnboardingController) VisitedSteps() map[string]bool {
	if oc == nil {
		return nil
	}
	copy := make(map[string]bool)
	for k, v := range oc.visited {
		copy[k] = v
	}
	return copy
}

// Replay restarts the onboarding sequence if allowed.
func (oc *OnboardingController) Replay() {
	if oc == nil || !oc.canReplay {
		return
	}
	oc.visited = make(map[string]bool)
	oc.Start()
}

// EnableReplay allows the player to restart onboarding.
func (oc *OnboardingController) EnableReplay(allow bool) {
	if oc != nil {
		oc.canReplay = allow
	}
}

// DefaultArkhamHorrorOnboarding returns a starter script for Arkham Horror.
func DefaultArkhamHorrorOnboarding() *OnboardingScript {
	return &OnboardingScript{
		Steps: []OnboardingStep{
			{
				ID:          "welcome",
				Title:       "Welcome, Investigator",
				Description: "You are in Arkham, a city plagued by supernatural forces. As an investigator, you must gather clues and cast protective wards before doom overcomes the town.",
				Duration:    4 * time.Second,
				Action:      "",
			},
			{
				ID:          "resources",
				Title:       "Your Resources",
				Description: "Your Health (red) and Sanity (blue) are critical—manage them carefully. Clues (green) help you uncover mysteries.",
				Duration:    3 * time.Second,
				Action:      "",
			},
			{
				ID:          "doom",
				Title:       "The Doom Counter",
				Description: "Watch the Doom Counter in the top right. If it reaches 12, the ancient one awakens and you lose. Failed actions increase doom for each Tentacle rolled.",
				Duration:    3 * time.Second,
				Action:      "",
			},
			{
				ID:          "locations",
				Title:       "Explore Locations",
				Description: "Four neighborhoods await: Downtown, University, Rivertown, and Northside. You can only move to adjacent locations.",
				Duration:    3 * time.Second,
				Action:      "",
			},
			{
				ID:          "actions",
				Title:       "Your Turn (2 Actions)",
				Description: "Each turn, you get 2 actions: Move, Gather Resources, Investigate (for clues), or Cast Ward (to reduce doom).",
				Duration:    3 * time.Second,
				Action:      "",
			},
			{
				ID:          "roll_dice",
				Title:       "Rolling Dice",
				Description: "Success (✓) advances your goal. Blank (○) has no effect. Tentacle (🐙) increases Doom. Different actions require different counts of Success.",
				Duration:    3 * time.Second,
				Action:      "",
			},
			{
				ID:          "win_condition",
				Title:       "How to Win",
				Description: "Gather clues equal to 4 × number of players (4 for you alone, 8 for 2 players, etc.) before Doom reaches 12.",
				Duration:    3 * time.Second,
				Action:      "",
			},
			{
				ID:          "ready",
				Title:       "Ready?",
				Description: "You can replay this guide anytime. Now, let\\'s begin your investigation!",
				Duration:    0,
				Action:      "acknowledged", // Advance manually or auto-dismiss.
			},
		},
	}
}
