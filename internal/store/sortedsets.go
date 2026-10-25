package store

import "sort"

type ScoredMember struct {
	Member string
	Score  float64
}

func (s *Store) SortedSetAdd(key string, members []ScoredMember) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "zset", true)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, member := range members {
		if _, exists := value.zset[member.Member]; !exists {
			added++
		}
		value.zset[member.Member] = member.Score
	}
	return added, nil
}

func (s *Store) SortedSetRange(key string, start, stop int) ([]ScoredMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "zset", false)
	if err != nil || value == nil {
		return nil, err
	}
	members := make([]ScoredMember, 0, len(value.zset))
	for member, score := range value.zset {
		members = append(members, ScoredMember{Member: member, Score: score})
	}
	// Redis orders equal scores by the member's bytes.
	sort.Slice(members, func(i, j int) bool {
		if members[i].Score == members[j].Score {
			return members[i].Member < members[j].Member
		}
		return members[i].Score < members[j].Score
	})
	start, end := sliceRange(len(members), start, stop)
	return members[start:end], nil
}

func (s *Store) SortedSetScore(key, member string) (float64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "zset", false)
	if err != nil || value == nil {
		return 0, false, err
	}
	score, exists := value.zset[member]
	return score, exists, nil
}

func (s *Store) SortedSetRemove(key string, members []string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "zset", false)
	if err != nil || value == nil {
		return 0, err
	}
	removed := 0
	for _, member := range members {
		if _, exists := value.zset[member]; exists {
			delete(value.zset, member)
			removed++
		}
	}
	if len(value.zset) == 0 {
		delete(s.values, key)
	}
	return removed, nil
}

func (s *Store) SortedSetLen(key string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, err := s.lookup(key, "zset", false)
	if err != nil || value == nil {
		return 0, err
	}
	return len(value.zset), nil
}
