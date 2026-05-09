package ui

// HUD represents the overall heads-up display layout with three major zones.
type HUD struct {
	// StatusRail occupies the top of the screen and displays current turn, doom, and player resources.
	StatusRail StatusRail

	// Board occupies the center of the screen and displays the game board and player positions.
	Board BoardZone

	// ActionRail occupies the bottom of the screen and provides action buttons.
	ActionRail ActionRail

	Viewport *Viewport
}

// StatusRail displays turn information, doom counter, and win condition progress at the top.
type StatusRail struct {
	// CurrentTurnDisplay shows whose turn it is and how many actions remain.
	CurrentTurnDisplay *PlayerTurnIndicator

	// DoomDisplay shows the current doom level and max threshold
	DoomDisplay *DoomDisplay

	// ObjectiveDisplay shows the collected clues progress toward win condition
	ObjectiveDisplay *ObjectiveCard

	// PlayerStrip shows icons/avatars for all connected players
	PlayerStrip *PlayerStrip

	Bounds Rect
}

// PlayerTurnIndicator shows the current acting player and action count.
type PlayerTurnIndicator struct {
	PlayerName        string
	ActionsRemaining  int
	MaxActions        int
	IsCurrentPlayer   bool
	CurrentPlayerName string
	Bounds            Rect
}

// ObjectiveCard displays the clue collection goal.
type ObjectiveCard struct {
	CollectedClues int
	RequiredClues  int
	PlayerCount    int
	Bounds         Rect
}

// PlayerStrip shows small avatars/indicators for all players in turn order.
type PlayerStrip struct {
	Players []*PlayerAvatar
	Bounds  Rect
}

// PlayerAvatar represents a single player in the player strip.
type PlayerAvatar struct {
	PlayerID    string
	DisplayName string
	Archetype   string
	Health      int
	Sanity      int
	IsActive    bool
	IsCurrent   bool
	Color       Color
	Bounds      Rect
}

// BoardZone occupies the center of the screen and shows the game board.
type BoardZone struct {
	LocationGrids    []*LocationGrid // 4 neighborhoods
	PlayerTokens     []*PlayerToken
	InteractionHints []*InteractionHint
	Bounds           Rect
}

// LocationGrid represents a single location on the board with encounter/resource info.
type LocationGrid struct {
	Location      string
	DisplayName   string
	EncounterText string
	Resources     []ResourceIcon
	Enemies       []EnemyToken
	Bounds        Rect
}

// ResourceIcon shows a specific resource at a location (clues, health, sanity, etc).
type ResourceIcon struct {
	Type   ResourceType
	Count  int
	Bounds Rect
}

// ResourceType identifies what kind of resource is displayed.
type ResourceType int

const (
	ResourceClue ResourceType = iota
	ResourceHealth
	ResourceSanity
	ResourceFocus
)

// EnemyToken represents a supernatural threat at a location.
type EnemyToken struct {
	EnemyName string
	Health    int
	Bounds    Rect
}

// PlayerToken represents a player's physical position on the board.
type PlayerToken struct {
	PlayerID     string
	Location     string
	Color        Color
	IsActive     bool
	IsCurrent    bool
	HealthStatus int // for visual feedback
	Bounds       Rect
}

// InteractionHint provides visual guidance for available actions.
type InteractionHint struct {
	ActionType ActionHintType
	TargetID   string
	Text       string
	Bounds     Rect
}

// ActionHintType identifies what action the hint suggests.
type ActionHintType int

const (
	ActionHintMove ActionHintType = iota
	ActionHintInvestigate
	ActionHintGather
	ActionHintWard
)

// ActionRail displays available actions at the bottom of the screen.
type ActionRail struct {
	ActionButtons []*ActionButton
	Bounds        Rect
}

// ActionButton represents a single action the player can take.
type ActionButton struct {
	Action     ActionType
	Label      string
	Icon       string
	IsEnabled  bool
	IsPending  bool
	IsSelected bool
	Tooltip    string
	Bounds     Rect
}

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

// ActionTypeString returns the string name of an action type.
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
