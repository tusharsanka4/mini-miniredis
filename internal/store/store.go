package store

import (
	"errors"
	"sync"

	"github.com/tusharsanka4/mini-miniredis/internal/clock"
)

var ErrWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

// A key holds exactly one kind of value. Only its matching field is used.
type entry struct {
	kind string
	text string
	list []string
	set  map[string]bool
	hash map[string]string
	zset map[string]float64
}

// Store holds the values shared by all client connections.
type Store struct {
	commandMu sync.Mutex
	mu        sync.Mutex
	values    map[string]*entry
	clock     clock.Clock
}

// The command lock is shared by TCP and HTTP. EXEC holds it for the whole queue.
// The separate map mutex still protects each individual store operation.
func (s *Store) LockCommands() {
	s.commandMu.Lock()
}

func (s *Store) UnlockCommands() {
	s.commandMu.Unlock()
}

func New() *Store {
	return NewWithClock(clock.Real{})
}

// NewWithClock lets tests control time. Passing nil uses the real clock.
func NewWithClock(c clock.Clock) *Store {
	if c == nil {
		c = clock.Real{}
	}
	return &Store{values: make(map[string]*entry), clock: c}
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// SET replaces any previous value, including other data types.
	s.values[key] = &entry{kind: "string", text: value}
}

func (s *Store) Get(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "string", false)
	if err != nil || value == nil {
		return "", false, err
	}
	return value.text, true, nil
}

func (s *Store) Type(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if value := s.values[key]; value != nil {
		return value.kind
	}
	return "none"
}

func (s *Store) Exists(keys ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, key := range keys {
		if _, exists := s.values[key]; exists {
			count++
		}
	}
	return count
}

// Call lookup only while holding s.mu.
func (s *Store) lookup(key, kind string, create bool) (*entry, error) {
	value := s.values[key]
	if value != nil {
		if value.kind != kind {
			return nil, ErrWrongType
		}
		return value, nil
	}
	if !create {
		return nil, nil
	}
	value = &entry{kind: kind}
	switch kind {
	case "set":
		value.set = make(map[string]bool)
	case "hash":
		value.hash = make(map[string]string)
	case "zset":
		value.zset = make(map[string]float64)
	}
	s.values[key] = value
	return value, nil
}

// Redis ranges use inclusive endpoints and allow negative indexes.
// Convert them to a Go slice range with an exclusive end.
func sliceRange(length, start, stop int) (int, int) {
	if start < 0 {
		start += length
	}
	if stop < 0 {
		stop += length
	}
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start >= length || stop < start {
		return 0, 0
	}
	return start, stop + 1
}

func (s *Store) Delete(keys ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := 0
	for _, key := range keys {
		if _, exists := s.values[key]; exists {
			delete(s.values, key)
			deleted++
		}
	}
	return deleted
}
