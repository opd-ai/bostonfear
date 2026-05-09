package ui

// TurnWidget displays the current turn holder and actions remaining.
type TurnWidget struct {
	CurrentPlayer      string
	ActionsRemaining   int
	MaxActions         int
	IsYourTurn         bool
	ConnectedPlayerIDs []string
}

// NewTurnWidget creates a widget for turn display.
func NewTurnWidget() *TurnWidget {
	return &TurnWidget{
		CurrentPlayer:      "",
		ActionsRemaining:   0,
		MaxActions:         2,
		IsYourTurn:         false,
		ConnectedPlayerIDs: []string{},
	}
}

// Update synchronizes the widget with current game state.
// Called whenever game state changes.
func (w *TurnWidget) Update(currentPlayer string, actionsRemaining int, yourPlayerID string, connectedPlayerIDs []string) {
	if w == nil {
		return
	}
	w.CurrentPlayer = currentPlayer
	w.ActionsRemaining = actionsRemaining
	w.IsYourTurn = (currentPlayer == yourPlayerID)
	w.ConnectedPlayerIDs = connectedPlayerIDs
}

// StatusText returns a human-readable status string.
func (w *TurnWidget) StatusText() string {
	if w == nil {
		return ""
	}
	if w.IsYourTurn {
		return "Your Turn (" + string(rune(w.ActionsRemaining)) + "/" + string(rune(w.MaxActions)) + " actions)"
	}
	return w.CurrentPlayer + "'s Turn"
}

// ProgressPercent returns a 0-100 progress fraction for actions used.
func (w *TurnWidget) ProgressPercent() int {
	if w == nil || w.MaxActions == 0 {
		return 0
	}
	actionsUsed := w.MaxActions - w.ActionsRemaining
	return (actionsUsed * 100) / w.MaxActions
}

// ActionsRemainingText returns a string like "2/2" or "1/2".
func (w *TurnWidget) ActionsRemainingText() string {
	if w == nil {
		return "0/0"
	}
	return string(rune('0'+w.ActionsRemaining)) + "/" + string(rune('0'+w.MaxActions))
}

// AvailableActionsWidget lists which actions can be taken.
type AvailableActionsWidget struct {
	Actions map[string]ActionInfo
}

// ActionInfo describes an available action.
type ActionInfo struct {
	ID          string
	Label       string
	Icon        string // e.g., "move", "gather", "investigate"
	Description string
	CostHealth  int
	CostSanity  int
	CostClues   int
	Enabled     bool
	Reason      string // Why disabled, if applicable.
}

// NewAvailableActionsWidget creates a widget for action display.
func NewAvailableActionsWidget() *AvailableActionsWidget {
	return &AvailableActionsWidget{
		Actions: make(map[string]ActionInfo),
	}
}

// RegisterAction adds or updates an action in the list.
func (w *AvailableActionsWidget) RegisterAction(info ActionInfo) {
	if w != nil {
		w.Actions[info.ID] = info
	}
}

// SetActionEnabled marks an action as available or unavailable with optional reason.
func (w *AvailableActionsWidget) SetActionEnabled(actionID string, enabled bool, reason string) {
	if w == nil {
		return
	}
	if info, exists := w.Actions[actionID]; exists {
		info.Enabled = enabled
		info.Reason = reason
		w.Actions[actionID] = info
	}
}

// GetAction returns info for a given action ID.
func (w *AvailableActionsWidget) GetAction(actionID string) (ActionInfo, bool) {
	if w == nil {
		return ActionInfo{}, false
	}
	info, exists := w.Actions[actionID]
	return info, exists
}

// AllActionsEnabled reports whether all actions are currently available.
func (w *AvailableActionsWidget) AllActionsEnabled() bool {
	if w == nil || len(w.Actions) == 0 {
		return false
	}
	for _, info := range w.Actions {
		if !info.Enabled {
			return false
		}
	}
	return true
}

// DisabledActionsCount returns how many actions are unavailable.
func (w *AvailableActionsWidget) DisabledActionsCount() int {
	if w == nil {
		return 0
	}
	count := 0
	for _, info := range w.Actions {
		if !info.Enabled {
			count++
		}
	}
	return count
}
