package feedback

import "github.com/opd-ai/bostonfear/client/ebiten/ui"

import "time"

// FeedbackQueue manages transient UI notifications (toasts, confirmations, action results).
type FeedbackQueue struct {
	items []*Feedback
}

// Feedback represents a single notification message.
type Feedback struct {
	// ID is a unique identifier for this feedback item.
	ID string

	// Type indicates what kind of feedback this is.
	Type FeedbackType

	// Title is the primary message text.
	Title string

	// Description is optional additional context.
	Description string

	// Severity controls visual prominence and color scheme.
	Severity Severity

	// CreatedAt marks when this feedback was queued.
	CreatedAt time.Time

	// Duration indicates how long to display before auto-dismissing (0 = no auto-dismiss).
	Duration time.Duration

	// ActionDeltas summarize state changes for this feedback (e.g., dice roll results, resource changes).
	ActionDeltas *ActionDelta

	// State tracks the feedback lifecycle.
	State FeedbackState

	// Bounds define the on-screen position (computed at render time).
	Bounds ui.Rect
}

// FeedbackType categorizes the kind of notification.
type FeedbackType int

const (
	FeedbackActionPending FeedbackType = iota
	FeedbackActionResolved
	FeedbackDiceResult
	FeedbackResourceChanged
	FeedbackDoomIncremented
	FeedbackInvalidAction
	FeedbackGameEvent
	FeedbackConnectionStatus
	FeedbackGameOver
)

// Severity indicates the visual prominence and color of a feedback item.
type Severity int

const (
	SeverityInfo Severity = iota
	SeveritySuccess
	SeverityWarning
	SeverityError
)

// FeedbackState tracks the lifecycle of a feedback item.
type FeedbackState int

const (
	FeedbackStateEntering FeedbackState = iota
	FeedbackStateDisplayed
	FeedbackStateExiting
	FeedbackStateDismissed
)

// ActionDelta summarizes the state changes resulting from an action.
type ActionDelta struct {
	Action         ui.ActionType
	ActionLabel    string
	DiceResults    []string // "success", "blank", "tentacle"
	ResourceDelta  ResourceDelta
	LocationChange *LocationChange
	DoomDelta      int
	Success        bool
	FailureReason  string
}

// ResourceDelta tracks health, sanity, clue changes.
type ResourceDelta struct {
	HealthDelta int
	SanityDelta int
	ClueDelta   int
	DoomDelta   int
}

// LocationChange tracks a move action.
type LocationChange struct {
	From string
	To   string
}

// NewFeedback creates a new feedback item with defaults.
func NewFeedback(id string, typ FeedbackType, title string) *Feedback {
	return &Feedback{
		ID:        id,
		Type:      typ,
		Title:     title,
		CreatedAt: time.Now(),
		State:     FeedbackStateEntering,
	}
}

// SetSeverity sets the severity level.
func (f *Feedback) SetSeverity(sev Severity) *Feedback {
	f.Severity = sev
	return f
}

// SetDescription sets optional description text.
func (f *Feedback) SetDescription(desc string) *Feedback {
	f.Description = desc
	return f
}

// SetDuration sets auto-dismiss duration (0 = never auto-dismiss).
func (f *Feedback) SetDuration(d time.Duration) *Feedback {
	f.Duration = d
	return f
}

// SetActionDelta sets the state change summary for this feedback.
func (f *Feedback) SetActionDelta(delta *ActionDelta) *Feedback {
	f.ActionDeltas = delta
	return f
}

// Enqueue adds a feedback item to the queue.
func (fq *FeedbackQueue) Enqueue(feedback *Feedback) {
	fq.items = append(fq.items, feedback)
}

// AllItems returns all feedback items in the queue.
func (fq *FeedbackQueue) AllItems() []*Feedback {
	return fq.items
}

// Dismiss removes a feedback item by ID.
func (fq *FeedbackQueue) Dismiss(id string) {
	for i, f := range fq.items {
		if f.ID == id {
			fq.items = append(fq.items[:i], fq.items[i+1:]...)
			return
		}
	}
}

// Prune removes old feedback items based on their display duration.
func (fq *FeedbackQueue) Prune(now time.Time) {
	var remaining []*Feedback
	for _, f := range fq.items {
		if f.Duration > 0 && now.Sub(f.CreatedAt) > f.Duration {
			continue
		}
		remaining = append(remaining, f)
	}
	fq.items = remaining
}

// Toast is a simple notification that appears briefly and auto-dismisses.
type Toast struct {
	Message  string
	Duration time.Duration
	Severity Severity
	Bounds   ui.Rect
}

// Confirmation is a modal dialog requesting user confirmation for a significant action.
type Confirmation struct {
	Title       string
	Description string
	YesLabel    string
	NoLabel     string
	OnYes       func()
	OnNo        func()
	Bounds      ui.Rect
}
