package store

import "sort"

// pairs contains alternating field names and values, checked by the dispatcher.
func (s *Store) HashSet(key string, pairs []string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "hash", true)
	if err != nil {
		return 0, err
	}
	added := 0
	for i := 0; i < len(pairs); i += 2 {
		if _, exists := value.hash[pairs[i]]; !exists {
			added++
		}
		value.hash[pairs[i]] = pairs[i+1]
	}
	return added, nil
}

func (s *Store) HashGet(key, field string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "hash", false)
	if err != nil || value == nil {
		return "", false, err
	}
	text, exists := value.hash[field]
	return text, exists, nil
}

func (s *Store) HashDelete(key string, fields []string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "hash", false)
	if err != nil || value == nil {
		return 0, err
	}
	removed := 0
	for _, field := range fields {
		if _, exists := value.hash[field]; exists {
			delete(value.hash, field)
			removed++
		}
	}
	if len(value.hash) == 0 {
		delete(s.values, key)
	}
	return removed, nil
}

func (s *Store) HashGetAll(key string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "hash", false)
	if err != nil || value == nil {
		return nil, err
	}
	fields := make([]string, 0, len(value.hash))
	for field := range value.hash {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	pairs := make([]string, 0, 2*len(fields))
	for _, field := range fields {
		pairs = append(pairs, field, value.hash[field])
	}
	return pairs, nil
}

func (s *Store) HashLen(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "hash", false)
	if err != nil || value == nil {
		return 0, err
	}
	return len(value.hash), nil
}
