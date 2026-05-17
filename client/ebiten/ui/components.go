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

// ButtonVariant defines semantic button intent.
type ButtonVariant int

const (
	ButtonPrimary ButtonVariant = iota
	ButtonSecondary
	ButtonDanger
	ButtonDisabled
)

// ButtonSize defines standard button dimensions.
type ButtonSize int

const (
	ButtonSizeSmall ButtonSize = iota
	ButtonSizeMedium
	ButtonSizeLarge
)

// ButtonState defines interaction state for rendering.
type ButtonState int

const (
	ButtonStateDefault ButtonState = iota
	ButtonStateHover
	ButtonStatePressed
	ButtonStateDisabled
	ButtonStateLoading
)

// ButtonStyle captures resolved visuals for a button instance.
type ButtonStyle struct {
	Variant      ButtonVariant
	Size         ButtonSize
	State        ButtonState
	Width        float64
	Height       float64
	CornerRadius float64
	Padding      float64
	Fill         Color
	Border       Color
	Text         Color
	ShowSpinner  bool
	IconAllowed  bool
}

// ResolveButtonStyle resolves a reusable button style from semantic inputs.
// The returned style is renderer-agnostic and can be used by desktop/mobile/web.
func ResolveButtonStyle(variant ButtonVariant, size ButtonSize, state ButtonState, tokens *DesignTokenRegistry) ButtonStyle {
	if state == ButtonStateDisabled {
		variant = ButtonDisabled
	}

	style := ButtonStyle{
		Variant:      variant,
		Size:         size,
		State:        state,
		CornerRadius: 8,
		Padding:      10,
		IconAllowed:  true,
	}

	switch size {
	case ButtonSizeSmall:
		style.Width, style.Height = 32, 32
		style.Padding = 8
	case ButtonSizeLarge:
		style.Width, style.Height = 64, 64
		style.Padding = 12
	default:
		style.Width, style.Height = 48, 48
	}

	style.Fill = ColorPrimary
	style.Border = Color{R: 220, G: 232, B: 248, A: 255}
	style.Text = Color{R: 255, G: 255, B: 255, A: 255}

	switch variant {
	case ButtonSecondary:
		style.Fill = Color{R: 42, G: 54, B: 70, A: 255}
		style.Border = Color{R: 150, G: 176, B: 214, A: 255}
	case ButtonDanger:
		style.Fill = ColorDanger
		style.Border = Color{R: 255, G: 210, B: 200, A: 255}
	case ButtonDisabled:
		style.Fill = ColorNeutral
		style.Border = Color{R: 132, G: 132, B: 132, A: 255}
		style.Text = Color{R: 220, G: 220, B: 220, A: 255}
		style.IconAllowed = false
	}

	switch state {
	case ButtonStateHover:
		style.Fill = lightenColor(style.Fill, 16)
		style.Border = lightenColor(style.Border, 12)
	case ButtonStatePressed:
		style.Fill = lightenColor(style.Fill, 30)
		style.Border = lightenColor(style.Border, 20)
	case ButtonStateDisabled:
		style.Fill = Color{R: 88, G: 88, B: 96, A: 255}
		style.Border = Color{R: 122, G: 122, B: 132, A: 255}
		style.Text = Color{R: 196, G: 196, B: 204, A: 255}
		style.IconAllowed = false
	case ButtonStateLoading:
		style.ShowSpinner = true
	}

	if tokens != nil {
		if radius := tokens.GetCornerRadius("md"); radius > 0 {
			style.CornerRadius = radius
		}
		if padding := tokens.GetSpacing("button-padding"); padding > 0 {
			style.Padding = padding
		}
	}

	return style
}

func lightenColor(c Color, delta uint8) Color {
	add := func(v uint8) uint8 {
		max := int(v) + int(delta)
		if max > 255 {
			max = 255
		}
		return uint8(max)
	}
	return Color{R: add(c.R), G: add(c.G), B: add(c.B), A: c.A}
}
