package hud

import "github.com/opd-ai/bostonfear/client/ebiten/ui"

// HUD represents the overall heads-up display layout with three major zones.
type HUD struct {
	// StatusRail occupies the top of the screen and displays current turn, doom, and player resources.
	StatusRail StatusRail

	// Board occupies the center of the screen and displays the game board and player positions.
	Board BoardZone

	// ActionRail occupies the bottom of the screen and provides action buttons.
	ActionRail ActionRail

	Viewport *ui.Viewport
}

// StatusRail displays turn information, doom counter, and win condition progress at the top.
type StatusRail struct {
	// CurrentTurnDisplay shows whose turn it is and how many actions remain.
	CurrentTurnDisplay *PlayerTurnIndicator

	// DoomDisplay shows the current doom level and max threshold
	DoomDisplay *ui.DoomDisplay

	// ObjectiveDisplay shows the collected clues progress toward win condition
	ObjectiveDisplay *ObjectiveCard

	// PlayerStrip shows icons/avatars for all connected players
	PlayerStrip *PlayerStrip

	Bounds ui.Rect
}

// PlayerTurnIndicator shows the current acting player and action count.
type PlayerTurnIndicator struct {
	PlayerName        string
	ActionsRemaining  int
	MaxActions        int
	IsCurrentPlayer   bool
	CurrentPlayerName string
	Bounds            ui.Rect
}

// ObjectiveCard displays the clue collection goal.
type ObjectiveCard struct {
	CollectedClues int
	RequiredClues  int
	PlayerCount    int
	Bounds         ui.Rect
}

// PlayerStrip shows small avatars/indicators for all players in turn order.
type PlayerStrip struct {
	Players []*PlayerAvatar
	Bounds  ui.Rect
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
	Color       ui.Color
	Bounds      ui.Rect
}

// BoardZone occupies the center of the screen and shows the game board.
type BoardZone struct {
	LocationGrids    []*LocationGrid // 4 neighborhoods
	PlayerTokens     []*PlayerToken
	InteractionHints []*InteractionHint
	Bounds           ui.Rect
}

// LocationGrid represents a single location on the board with encounter/resource info.
type LocationGrid struct {
	Location      string
	DisplayName   string
	EncounterText string
	Resources     []ResourceIcon
	Enemies       []EnemyToken
	Bounds        ui.Rect
}

// ResourceIcon shows a specific resource at a location (clues, health, sanity, etc).
type ResourceIcon struct {
	Type   ui.ResourceType
	Count  int
	Bounds ui.Rect
}

// EnemyToken represents a supernatural threat at a location.
type EnemyToken struct {
	EnemyName string
	Health    int
	Bounds    ui.Rect
}

// PlayerToken represents a player's physical position on the board.
type PlayerToken struct {
	PlayerID     string
	Location     string
	Color        ui.Color
	IsActive     bool
	IsCurrent    bool
	HealthStatus int // for visual feedback
	Bounds       ui.Rect
}

// InteractionHint provides visual guidance for available actions.
type InteractionHint struct {
	ActionType ui.ActionHintType
	TargetID   string
	Text       string
	Bounds     ui.Rect
}

// ActionRail displays available actions at the bottom of the screen.
type ActionRail struct {
	ActionButtons []*ActionButton
	Bounds        ui.Rect
}

// ActionButton represents a single action the player can take.
type ActionButton struct {
	Action     ui.ActionType
	Label      string
	Icon       string
	IsEnabled  bool
	IsPending  bool
	IsSelected bool
	Tooltip    string
	Bounds     ui.Rect
}
