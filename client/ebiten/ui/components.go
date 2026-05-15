package ui

// Badge represents a small label or status indicator.
type Badge struct {
	Text    string
	Color   Color
	Variant BadgeVariant
	Bounds  Rect
}

// BadgeVariant defines the visual style for a badge.
type BadgeVariant int

const (
	BadgeDefault BadgeVariant = iota
	BadgePrimary
	BadgeSuccess
	BadgeWarning
	BadgeDanger
)

// Pill represents a larger label with optional icon, commonly used for categorization.
type Pill struct {
	Text     string
	Icon     string
	Selected bool
	Bounds   Rect
}

// Counter displays a numeric value with increment/decrement UI.
type Counter struct {
	Value     int
	Min       int
	Max       int
	Label     string
	Color     Color
	ShowLabel bool
	Bounds    Rect
}

// SegmentedBar represents progress or resource levels as a filled/unfilled bar.
type SegmentedBar struct {
	Current int // filled segments
	Max     int // total segments
	Color   Color
	Bounds  Rect
}

// HealthDisplay combines a segmented bar with health-specific styling.
type HealthDisplay struct {
	Value      int // 1-10
	MaxHealth  int // 10
	Bar        SegmentedBar
	HarmEffect int // additional harm indicator
	Bounds     Rect
}

// SanityDisplay combines a segmented bar with sanity-specific styling and madness indicator.
type SanityDisplay struct {
	Value        int // 1-10
	MaxSanity    int // 10
	Bar          SegmentedBar
	MadnessLevel int // 0-5
	Bounds       Rect
}

// ClueDisplay shows collected clues with a counter and target progress.
type ClueDisplay struct {
	Collected int // 0-5
	Target    int // varies by player count
	Counter   Counter
	Bounds    Rect
}

// DoomDisplay shows the global doom counter with prominence scaling.
type DoomDisplay struct {
	Current   int // 0-12
	Max       int // 12
	Counter   Counter
	Intensity float64 // 0.0-1.0, for shader/animation intensity
	Bounds    Rect
}

// Color represents an RGBA color value for UI elements.
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// Rect represents a rectangular region in logical coordinates.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Contains reports whether the point (px, py) is within this rectangle.
func (r Rect) Contains(px, py float64) bool {
	return px >= r.X && px < r.X+r.Width &&
		py >= r.Y && py < r.Y+r.Height
}

// Standard semantic colors for the game UI.
var (
	ColorPrimary = Color{R: 100, G: 150, B: 200, A: 255}
	ColorSuccess = Color{R: 76, G: 175, B: 80, A: 255}
	ColorWarning = Color{R: 255, G: 193, B: 7, A: 255}
	ColorDanger  = Color{R: 244, G: 67, B: 54, A: 255}
	ColorNeutral = Color{R: 158, G: 158, B: 158, A: 255}
	ColorHealth  = Color{R: 76, G: 175, B: 80, A: 255}
	ColorSanity  = Color{R: 156, G: 39, B: 176, A: 255}
	ColorClue    = Color{R: 33, G: 150, B: 243, A: 255}
	ColorDoom    = Color{R: 244, G: 67, B: 54, A: 255}
)

// ActionType identifies what action can be performed.
type ActionType int

const (
	ActionMove ActionType = iota
	ActionGather
	ActionInvestigate
	ActionCastWard
	ActionFocus
	ActionResearch
	ActionTrade
	ActionEncounter
	ActionComponent
	ActionAttack
	ActionEvade
	ActionCloseGate
)

// String returns the string name of an action type.
func (a ActionType) String() string {
	switch a {
	case ActionMove:
		return "Move"
	case ActionGather:
		return "Gather"
	case ActionInvestigate:
		return "Investigate"
	case ActionCastWard:
		return "Cast Ward"
	case ActionFocus:
		return "Focus"
	case ActionResearch:
		return "Research"
	case ActionTrade:
		return "Trade"
	case ActionEncounter:
		return "Encounter"
	case ActionComponent:
		return "Component"
	case ActionAttack:
		return "Attack"
	case ActionEvade:
		return "Evade"
	case ActionCloseGate:
		return "Close Gate"
	default:
		return "Unknown"
	}
}

// ResourceType identifies what kind of resource is displayed.
type ResourceType int

const (
	ResourceClue ResourceType = iota
	ResourceHealth
	ResourceSanity
	ResourceFocus
)

// ActionHintType identifies what action the hint suggests.
type ActionHintType int

const (
	ActionHintMove ActionHintType = iota
	ActionHintInvestigate
	ActionHintGather
	ActionHintWard
)
