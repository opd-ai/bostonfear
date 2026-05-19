package app

// Token layout constants define spacing for investigator tokens on location tiles.
const (
	TokenBaseDiameter    = 44.0 // Base token diameter in logical pixels
	TokenHorizontalPitch = 52   // Horizontal spacing: diameter + 8px gap
	TokenVerticalPitch   = 50   // Vertical spacing: diameter + 6px gap
)

// Player panel layout constants define the right-side player roster positioning.
const (
	PlayerPanelY      = 110 // Top Y coordinate of player panel
	PlayerPanelHeight = 204 // Total height: header(34) + 6×(24+4) rows
	PlayerRowHeight   = 24  // Height of each player row card
	PlayerRowGap      = 4   // Vertical gap between player rows
	PlayerPanelGap    = 8   // Vertical gap between player and location panels
)

// Action grid layout constants define the action dock button grid.
const (
	ActionGridOriginX      = 10  // Left margin of action grid
	ActionGridGap          = 6   // Spacing between buttons
	ActionGridCellHeight   = 44  // Button height
	ActionGridMinCellWidth = 110 // Minimum button width (increased from 96px)
	ActionGridHeader       = 48  // Header space for title and status text
)

// Resource widget spacing constants define margins for health/sanity/clue displays.
const (
	ResourceTrackSpacing = 8 // Spacing after resource tracks
	ResourcePillSpacing  = 8 // Spacing after resource pills (standardized)
	ResourceSegmentWidth = 8 // Width of each segment in a resource bar
	ResourceSegmentGap   = 2 // Gap between segments in a resource bar
)

// Right panel layout constants define the common dimensions for right-side panels.
const (
	RightPanelMargin = 10  // Horizontal margin from screen edge
	RightPanelWidth  = 386 // Fixed width for all right-side panels
)
