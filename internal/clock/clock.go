package clock

import (
	"sync"
	"time"
)

// Clock tracks a simulated time based on a fixed base and an offset.
type Clock struct {
	mu     sync.RWMutex
	base   time.Time
	offset time.Duration
}

// New creates a clock seeded with the provided base time.
func New(base time.Time) *Clock {
	return &Clock{base: base}
}

// Now returns the current simulated time.
func (c *Clock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.base.Add(c.offset)
}

// Advance moves the simulated clock forward and returns the updated time.
func (c *Clock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	c.offset += d
	now := c.base.Add(c.offset)
	c.mu.Unlock()
	return now
}

// Base returns the initial base time for the clock.
func (c *Clock) Base() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.base
}
