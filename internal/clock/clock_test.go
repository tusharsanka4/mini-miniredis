package clock

import (
	"sync"
	"testing"
	"time"
)

func TestManualClock(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewManual(start)
	for i := 0; i < 10; i++ {
		if got := c.Now(); !got.Equal(start) {
			t.Fatalf("clock moved without Advance or Set: %v", got)
		}
	}

	c.Advance(5 * time.Second)
	if want := start.Add(5 * time.Second); !c.Now().Equal(want) {
		t.Fatalf("after Advance: got %v, want %v", c.Now(), want)
	}

	// Tests may move time backward too.
	c.Advance(-time.Second)
	if want := start.Add(4 * time.Second); !c.Now().Equal(want) {
		t.Fatalf("after negative Advance: got %v, want %v", c.Now(), want)
	}

	c.Set(start)
	if !c.Now().Equal(start) {
		t.Fatalf("after Set: got %v, want %v", c.Now(), start)
	}
}

func TestConcurrentAdvance(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewManual(start)
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				c.Advance(time.Second)
				c.Now()
			}
		}()
	}
	workers.Wait()
	if want := start.Add(800 * time.Second); !c.Now().Equal(want) {
		t.Fatalf("concurrent advances lost time: got %v, want %v", c.Now(), want)
	}
}
