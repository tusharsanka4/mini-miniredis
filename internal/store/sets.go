package store

import "sort"

func (s *Store) SetAdd(key string, members []string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "set", true)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, member := range members {
		if !value.set[member] {
			value.set[member] = true
			added++
		}
	}
	return added, nil
}

func (s *Store) SetRemove(key string, members []string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "set", false)
	if err != nil || value == nil {
		return 0, err
	}
	removed := 0
	for _, member := range members {
		if value.set[member] {
			delete(value.set, member)
			removed++
		}
	}
	if len(value.set) == 0 {
		delete(s.values, key)
	}
	return removed, nil
}

func (s *Store) SetMembers(key string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "set", false)
	if err != nil || value == nil {
		return nil, err
	}
	members := make([]string, 0, len(value.set))
	for member := range value.set {
		members = append(members, member)
	}
	sort.Strings(members) // Stable output makes inspecting and testing easier.
	return members, nil
}

func (s *Store) SetContains(key, member string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "set", false)
	if err != nil || value == nil {
		return 0, err
	}
	if value.set[member] {
		return 1, nil
	}
	return 0, nil
}

func (s *Store) SetLen(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "set", false)
	if err != nil || value == nil {
		return 0, err
	}
	return len(value.set), nil
}
