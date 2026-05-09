package ui

import (
	"strconv"
	"time"
)

// StateVisibilityWidget displays sync status, reconnection state, and pending actions.
type StateVisibilityWidget struct {
	ConnectionState ConnectionState
	SyncStatus      SyncStatus
	PendingActions  []PendingAction
	LastSyncTime    time.Time
	ShowBanner      bool
}

// ConnectionState indicates the network state.
type ConnectionState int

const (
	Connected ConnectionState = iota
	Connecting
	Reconnecting
	Disconnected
)

// StateText returns a display string for connection state.
func (cs ConnectionState) String() string {
	switch cs {
	case Connected:
		return "Connected"
	case Connecting:
		return "Connecting..."
	case Reconnecting:
		return "Reconnecting..."
	case Disconnected:
		return "Disconnected"
	default:
		return "Unknown"
	}
}

// SyncStatus indicates whether game state is synchronized.
type SyncStatus int

const (
	Synchronized SyncStatus = iota
	Syncing
	OutOfSync
)

// StatusText returns a display string for sync status.
func (ss SyncStatus) String() string {
	switch ss {
	case Synchronized:
		return "In Sync"
	case Syncing:
		return "Syncing..."
	case OutOfSync:
		return "Out of Sync"
	default:
		return "Unknown"
	}
}

// PendingAction represents an action awaiting server confirmation.
type PendingAction struct {
	ID        string
	Action    string
	Target    string
	Timestamp time.Time
	Status    PendingStatus
}

// PendingStatus indicates what happened to a pending action.
type PendingStatus int

const (
	PendingSubmitted PendingStatus = iota
	PendingConfirmed
	PendingFailed
	PendingRetrying
)

// StatusText returns a display string for pending action status.
func (ps PendingStatus) String() string {
	switch ps {
	case PendingSubmitted:
		return "Submitted..."
	case PendingConfirmed:
		return "Confirmed"
	case PendingFailed:
		return "Failed"
	case PendingRetrying:
		return "Retrying..."
	default:
		return "Unknown"
	}
}

// NewStateVisibilityWidget creates a widget for state display.
func NewStateVisibilityWidget() *StateVisibilityWidget {
	return &StateVisibilityWidget{
		ConnectionState: Disconnected,
		SyncStatus:      OutOfSync,
		PendingActions:  make([]PendingAction, 0),
		ShowBanner:      false,
	}
}

// SetConnectionState updates the connection status.
func (w *StateVisibilityWidget) SetConnectionState(state ConnectionState) {
	if w != nil {
		w.ConnectionState = state
		w.ShowBanner = (state != Connected)
	}
}

// SetSyncStatus updates the synchronization status.
func (w *StateVisibilityWidget) SetSyncStatus(status SyncStatus) {
	if w != nil {
		w.SyncStatus = status
		w.LastSyncTime = time.Now()
		w.ShowBanner = (status != Synchronized)
	}
}

// AddPendingAction registers a pending action.
func (w *StateVisibilityWidget) AddPendingAction(action PendingAction) {
	if w != nil {
		action.Timestamp = time.Now()
		w.PendingActions = append(w.PendingActions, action)
	}
}

// UpdatePendingAction updates the status of a pending action.
func (w *StateVisibilityWidget) UpdatePendingAction(id string, status PendingStatus) {
	if w == nil {
		return
	}
	for i := range w.PendingActions {
		if w.PendingActions[i].ID == id {
			w.PendingActions[i].Status = status
			return
		}
	}
}

// RemovePendingAction removes a completed pending action.
func (w *StateVisibilityWidget) RemovePendingAction(id string) {
	if w == nil {
		return
	}
	for i := range w.PendingActions {
		if w.PendingActions[i].ID == id {
			w.PendingActions = append(w.PendingActions[:i], w.PendingActions[i+1:]...)
			return
		}
	}
}

// GetPendingAction retrieves a pending action by ID.
func (w *StateVisibilityWidget) GetPendingAction(id string) (PendingAction, bool) {
	if w == nil {
		return PendingAction{}, false
	}
	for _, action := range w.PendingActions {
		if action.ID == id {
			return action, true
		}
	}
	return PendingAction{}, false
}

// HasPendingActions reports whether any actions are awaiting confirmation.
func (w *StateVisibilityWidget) HasPendingActions() bool {
	return w != nil && len(w.PendingActions) > 0
}

// BannerVisible reports whether the status banner should be shown.
func (w *StateVisibilityWidget) BannerVisible() bool {
	return w != nil && (w.ShowBanner || w.HasPendingActions())
}

// BannerText returns text to display on the status banner.
func (w *StateVisibilityWidget) BannerText() string {
	if w == nil {
		return ""
	}
	if w.ConnectionState != Connected {
		return w.ConnectionState.String()
	}
	if w.SyncStatus != Synchronized {
		return w.SyncStatus.String()
	}
	if w.HasPendingActions() {
		count := len(w.PendingActions)
		if count == 1 {
			return w.PendingActions[0].Action + " pending..."
		}
		return strconv.Itoa(count) + " actions pending..."
	}
	return ""
}
