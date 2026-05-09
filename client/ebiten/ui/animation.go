package ui

import (
	"math"
	"time"
)

// EasingFunc defines how a value changes over time (0->1).
type EasingFunc func(t float64) float64

// Common easing functions.
var (
	// Linear: constant velocity.
	EaseLinear EasingFunc = func(t float64) float64 {
		if t < 0 {
			return 0
		}
		if t > 1 {
			return 1
		}
		return t
	}

	// EaseInQuad: acceleration from 0.
	EaseInQuad EasingFunc = func(t float64) float64 {
		if t < 0 {
			return 0
		}
		if t > 1 {
			return 1
		}
		return t * t
	}

	// EaseOutQuad: deceleration to stop.
	EaseOutQuad EasingFunc = func(t float64) float64 {
		if t < 0 {
			return 0
		}
		if t > 1 {
			return 1
		}
		return 1 - (1-t)*(1-t)
	}

	// EaseInOutQuad: acceleration then deceleration.
	EaseInOutQuad EasingFunc = func(t float64) float64 {
		if t < 0 {
			return 0
		}
		if t > 1 {
			return 1
		}
		if t < 0.5 {
			return 2 * t * t
		}
		return 1 - 2*(1-t)*(1-t)
	}

	// EasePulse: elasticoscillation effect.
	EasePulse EasingFunc = func(t float64) float64 {
		if t < 0 {
			return 0
		}
		if t > 1 {
			return 1
		}
		// Sine pulse: starts at 1, dips, returns to 1.
		const pi = 3.14159265359
		return 1 - 0.3*EaseOutQuad(t)*quadraticBounce(t*2*pi*1.5)
	}
)

// quadraticBounce helper for pulse effect.
func quadraticBounce(angle float64) float64 {
	// Simplified sine-based oscillation.
	return (1 + sinApprox(angle)) / 2
}

// sinApprox approximates sine(x) for animation purposes.
func sinApprox(x float64) float64 {
	// Reduced to [0, 2*pi] range for efficiency.
	const pi = 3.14159265359
	const twoPi = 6.28318530718
	for x < 0 {
		x += twoPi
	}
	for x > twoPi {
		x -= twoPi
	}
	// Taylor series approximation (good enough for UI).
	return x - (x*x*x)/6.0 + (x*x*x*x*x)/120.0
}

// Transition represents an animated value change over time.
type Transition struct {
	startTime  time.Time
	duration   time.Duration
	easing     EasingFunc
	startVal   float64
	endVal     float64
	onComplete func()
}

// NewTransition creates a new transition from startVal to endVal over duration.
func NewTransition(duration time.Duration, startVal, endVal float64, easing EasingFunc) *Transition {
	if easing == nil {
		easing = EaseLinear
	}
	return &Transition{
		startTime: time.Now(),
		duration:  duration,
		easing:    easing,
		startVal:  startVal,
		endVal:    endVal,
	}
}

// OnComplete registers a callback to fire when transition ends.
func (t *Transition) OnComplete(fn func()) *Transition {
	t.onComplete = fn
	return t
}

// Value returns the interpolated value at the current time.
func (t *Transition) Value() float64 {
	if t == nil {
		return 0
	}
	if t.duration <= 0 {
		return t.endVal
	}
	elapsed := time.Since(t.startTime)
	progress := clamp01(float64(elapsed.Nanoseconds()) / float64(t.duration.Nanoseconds()))

	// Apply easing.
	eased := t.easing(progress)

	// Interpolate between start and end.
	return t.startVal + (t.endVal-t.startVal)*eased
}

// IsComplete reports whether the transition has finished.
func (t *Transition) IsComplete() bool {
	if t == nil {
		return true
	}
	elapsed := time.Since(t.startTime)
	return elapsed >= t.duration
}

// Update advances the transition and calls onComplete if finished.
// Should be called once per frame. Returns true if complete.
func (t *Transition) Update() bool {
	if t == nil {
		return true
	}
	if t.IsComplete() {
		if t.onComplete != nil {
			t.onComplete()
		}
		return true
	}
	return false
}

// AnimationQueue manages multiple sequential transitions.
type AnimationQueue struct {
	current   *Transition
	queued    []*Transition
	onAllDone func()
	isRunning bool
	isDone    bool
}

// NewAnimationQueue creates a queue for chaining animations.
func NewAnimationQueue() *AnimationQueue {
	return &AnimationQueue{
		queued: make([]*Transition, 0),
	}
}

// Queue appends a transition to the queue.
func (aq *AnimationQueue) Queue(t *Transition) *AnimationQueue {
	if t != nil {
		aq.queued = append(aq.queued, t)
	}
	return aq
}

// OnAllDone registers callback when entire queue finishes.
func (aq *AnimationQueue) OnAllDone(fn func()) *AnimationQueue {
	aq.onAllDone = fn
	return aq
}

// Start begins playback of the queued animations.
func (aq *AnimationQueue) Start() *AnimationQueue {
	aq.isRunning = true
	aq.isDone = false
	if len(aq.queued) > 0 {
		aq.current = aq.queued[0]
		aq.queued = aq.queued[1:]
	}
	return aq
}

// Update advances the current animation and chains to next if complete.
// Returns true if the entire queue has completed.
func (aq *AnimationQueue) Update() bool {
	if aq == nil || aq.isDone {
		return true
	}
	if !aq.isRunning {
		return false
	}
	if aq.current == nil {
		return aq.finishQueue()
	}
	if !aq.current.Update() {
		return false
	}
	aq.advanceQueue()
	return aq.isDone
}

func (aq *AnimationQueue) finishQueue() bool {
	aq.isDone = true
	if aq.onAllDone != nil {
		aq.onAllDone()
	}
	return true
}

func (aq *AnimationQueue) advanceQueue() {
	if len(aq.queued) > 0 {
		aq.current = aq.queued[0]
		aq.queued = aq.queued[1:]
		return
	}
	aq.current = nil
	aq.finishQueue()
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

// CurrentValue returns the value of the current transition.
func (aq *AnimationQueue) CurrentValue() float64 {
	if aq.current == nil {
		return 0
	}
	return aq.current.Value()
}

// IsRunning reports whether any animation is currently active.
func (aq *AnimationQueue) IsRunning() bool {
	return aq.isRunning && !aq.isDone
}

// AnimationPresets provides named transition presets for common UI effects.
type AnimationPresets struct {
	// Turn transition: 300ms fade + slide in.
	TurnTransition struct {
		FadeDuration  time.Duration
		SlideDuration time.Duration
	}

	// Dice roll reveal: 500ms scale-in with bounce.
	DiceReveal struct {
		Duration time.Duration
		Easing   EasingFunc
	}

	// Doom spike: 200ms pulse + color shift.
	DoomSpike struct {
		PulseDuration time.Duration
		Easing        EasingFunc
	}

	// Invalid action: 400ms shake + fade-out.
	InvalidAction struct {
		ShakeDuration time.Duration
		FadeDuration  time.Duration
	}
}

// NewAnimationPresets returns sensible defaults for game animations.
func NewAnimationPresets() *AnimationPresets {
	return &AnimationPresets{
		TurnTransition: struct {
			FadeDuration  time.Duration
			SlideDuration time.Duration
		}{
			FadeDuration:  300 * time.Millisecond,
			SlideDuration: 400 * time.Millisecond,
		},
		DiceReveal: struct {
			Duration time.Duration
			Easing   EasingFunc
		}{
			Duration: 500 * time.Millisecond,
			Easing:   EaseOutQuad,
		},
		DoomSpike: struct {
			PulseDuration time.Duration
			Easing        EasingFunc
		}{
			PulseDuration: 200 * time.Millisecond,
			Easing:        EasePulse,
		},
		InvalidAction: struct {
			ShakeDuration time.Duration
			FadeDuration  time.Duration
		}{
			ShakeDuration: 200 * time.Millisecond,
			FadeDuration:  400 * time.Millisecond,
		},
	}
}
