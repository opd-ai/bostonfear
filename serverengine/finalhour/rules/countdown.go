package rules

import "fmt"

// CountdownToken represents the time-pressure mechanic in Final Hour.
// The countdown starts at an initial value (typically 12-15) and decrements
// at the end of each round. When it reaches 0, the Ancient One wins automatically.
type CountdownToken struct {
	Current int
	Max     int
}

// DefaultCountdown creates a standard countdown token for Final Hour.
func DefaultCountdown() CountdownToken {
	return CountdownToken{
		Current: 12,
		Max:     12,
	}
}

// Decrement reduces the countdown by the specified amount.
// Returns an error if the decrement would result in a negative value.
func (c *CountdownToken) Decrement(amount int) error {
	if amount < 0 {
		return fmt.Errorf("countdown decrement amount must be non-negative: %d", amount)
	}
	if c.Current-amount < 0 {
		c.Current = 0
		return nil
	}
	c.Current -= amount
	return nil
}

// IsExpired returns true if the countdown has reached zero.
func (c CountdownToken) IsExpired() bool {
	return c.Current <= 0
}

// Remaining returns the current countdown value.
func (c CountdownToken) Remaining() int {
	return c.Current
}
