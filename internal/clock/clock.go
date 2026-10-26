package clock

import (
	"sync"
	"time"
)

// Clock is the only source of time the store will need for expiration.
type Clock interface {
	Now() time.Time
}

// Real follows the system clock.
type Real struct{}

func (Real) Now() time.Time {
	return time.Now()
}

// Manual stays at its current time until Set or Advance is called.
// The mutex lets a test advance time while client goroutines read it.
type Manual struct {
	mu  sync.Mutex
	now time.Time
}

func NewManual(start time.Time) *Manual {
	return &Manual{now: start}
}

func (c *Manual) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Manual) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

func (c *Manual) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}
