package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type fileDump struct {
	Snapshots []Snapshot `json:"snapshots"`
}

// FileStore persists memories in local JSON.
type FileStore struct {
	path      string
	mu        sync.RWMutex
	snapshots map[string]Snapshot
}

func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create memory data dir: %w", err)
	}

	store := &FileStore{
		path:      path,
		snapshots: map[string]Snapshot{},
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat memory data file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileStore) Load(sessionID string) (Snapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.snapshots[sessionID]
	if !ok {
		return Snapshot{}, false, nil
	}
	return cloneSnapshot(item), true, nil
}

func (s *FileStore) Save(snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshots[snapshot.SessionID] = cloneSnapshot(snapshot)
	return s.saveLocked()
}

func (s *FileStore) loadFromDisk() error {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read memory data file: %w", err)
	}
	if len(payload) == 0 {
		s.snapshots = map[string]Snapshot{}
		return nil
	}

	var dump fileDump
	if err := json.Unmarshal(payload, &dump); err != nil {
		return fmt.Errorf("decode memory data file: %w", err)
	}

	s.snapshots = make(map[string]Snapshot, len(dump.Snapshots))
	for _, item := range dump.Snapshots {
		s.snapshots[item.SessionID] = cloneSnapshot(item)
	}
	return nil
}

func (s *FileStore) saveLocked() error {
	dump := fileDump{Snapshots: make([]Snapshot, 0, len(s.snapshots))}
	for _, item := range s.snapshots {
		dump.Snapshots = append(dump.Snapshots, cloneSnapshot(item))
	}
	sort.Slice(dump.Snapshots, func(i, j int) bool {
		return dump.Snapshots[i].SessionID < dump.Snapshots[j].SessionID
	})

	payload, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return fmt.Errorf("encode memory data file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp memory data file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace memory data file: %w", err)
	}

	return nil
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	return Snapshot{
		SessionID: snapshot.SessionID,
		Facts:     cloneFacts(snapshot.Facts),
		UpdatedAt: snapshot.UpdatedAt,
	}
}
