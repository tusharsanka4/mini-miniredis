package store

import "sync"

// Store holds the values shared by all client connections.
type Store struct {
	mu     sync.Mutex
	values map[string]string
}

func New() *Store {
	return &Store{values: make(map[string]string)}
}

func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.values[key]
	return value, exists
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
