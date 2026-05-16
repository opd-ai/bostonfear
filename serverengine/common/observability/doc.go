// Package observability provides shared observability primitives for cross-engine use.
// Hook is the pluggable interface for game engine event observation.
package observability

import "time"

// Event carries contextual data for a single observable game engine event.
type Event struct {
	Name      string            // Event identifier (e.g. "doom.increment", "action.move")
	Timestamp time.Time         // Wall-clock time of the event
	Labels    map[string]string // Key/value metadata for filtering or routing
}

// Hook receives engine events for logging, metrics, or alerting.
type Hook interface {
	// Observe is called once per engine event. Implementations must be goroutine-safe.
	Observe(evt Event)
}

// NoopHook is a Hook that discards all events. Use as a safe default.
type NoopHook struct{}

func (NoopHook) Observe(Event) {}
