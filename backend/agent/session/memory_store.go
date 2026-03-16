package session

import "sync"

// InMemoryStore stores sessions in memory, useful for tests.
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{sessions: map[string]Session{}}
}

func (s *InMemoryStore) Load(id string) (Session, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.sessions[id]
	if !ok {
		return Session{}, false, nil
	}
	return cloneSession(item), true, nil
}

func (s *InMemoryStore) Save(data Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[data.ID] = cloneSession(data)
	return nil
}

func (s *InMemoryStore) List() ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.sessions) == 0 {
		return nil, nil
	}

	items := make([]Session, 0, len(s.sessions))
	for _, item := range s.sessions {
		items = append(items, cloneSession(item))
	}
	return items, nil
}
