package store

func (s *Store) Push(key string, values []string, left bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "list", true)
	if err != nil {
		return 0, err
	}
	if left {
		// LPUSH key a b puts b before a.
		items := make([]string, 0, len(values)+len(value.list))
		for i := len(values) - 1; i >= 0; i-- {
			items = append(items, values[i])
		}
		value.list = append(items, value.list...)
	} else {
		value.list = append(value.list, values...)
	}
	return len(value.list), nil
}

func (s *Store) Pop(key string, left bool) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "list", false)
	if err != nil || value == nil {
		return "", false, err
	}
	index := len(value.list) - 1
	if left {
		index = 0
	}
	item := value.list[index]
	value.list[index] = "" // Release the removed string's backing memory.
	if left {
		value.list = value.list[1:]
	} else {
		value.list = value.list[:index]
	}
	if len(value.list) == 0 {
		delete(s.values, key)
	}
	return item, true, nil
}

func (s *Store) ListRange(key string, start, stop int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "list", false)
	if err != nil || value == nil {
		return nil, err
	}
	start, end := sliceRange(len(value.list), start, stop)
	// Copy before releasing the lock so another client cannot change the result.
	return append([]string(nil), value.list[start:end]...), nil
}

func (s *Store) ListLen(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "list", false)
	if err != nil || value == nil {
		return 0, err
	}
	return len(value.list), nil
}
