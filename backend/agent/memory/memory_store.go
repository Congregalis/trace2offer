package memory

import "sync"

// InMemoryStore stores memory snapshots in memory.
type InMemoryStore struct {
	mu        sync.RWMutex
	snapshots map[string]Snapshot
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{snapshots: map[string]Snapshot{}}
}

func (s *InMemoryStore) Load(sessionID string) (Snapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.snapshots[sessionID]
	if !ok {
		return Snapshot{}, false, nil
	}
	return cloneSnapshot(item), true, nil
}

func (s *InMemoryStore) Save(snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshots[snapshot.SessionID] = cloneSnapshot(snapshot)
	return nil
}
