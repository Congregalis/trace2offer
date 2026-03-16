package memory

import (
	"errors"
	"strings"
	"time"
)

var ErrStoreUnavailable = errors.New("memory store is unavailable")

// Snapshot keeps distilled memory for one session.
type Snapshot struct {
	SessionID string   `json:"session_id"`
	Facts     []string `json:"facts"`
	UpdatedAt string   `json:"updated_at"`
}

// Store persists memory snapshots.
type Store interface {
	Load(sessionID string) (Snapshot, bool, error)
	Save(snapshot Snapshot) error
}

// Manager manages recall/update of long-term memory.
type Manager struct {
	store    Store
	maxFacts int
}

func NewManager(store Store, maxFacts int) *Manager {
	if maxFacts <= 0 {
		maxFacts = 20
	}
	return &Manager{store: store, maxFacts: maxFacts}
}

func (m *Manager) Recall(sessionID string) ([]string, error) {
	if m == nil || m.store == nil {
		return nil, ErrStoreUnavailable
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, nil
	}

	snapshot, ok, err := m.store.Load(sessionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return cloneFacts(snapshot.Facts), nil
}

func (m *Manager) Remember(sessionID string, fact string) error {
	if m == nil || m.store == nil {
		return ErrStoreUnavailable
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session id is required")
	}

	fact = strings.TrimSpace(fact)
	if fact == "" {
		return nil
	}

	snapshot, ok, err := m.store.Load(sessionID)
	if err != nil {
		return err
	}
	if !ok {
		snapshot = Snapshot{SessionID: sessionID}
	}

	// De-duplicate while keeping recency.
	nextFacts := make([]string, 0, len(snapshot.Facts)+1)
	for _, existing := range snapshot.Facts {
		if existing == fact {
			continue
		}
		nextFacts = append(nextFacts, existing)
	}
	nextFacts = append(nextFacts, fact)
	if len(nextFacts) > m.maxFacts {
		nextFacts = nextFacts[len(nextFacts)-m.maxFacts:]
	}

	snapshot.Facts = nextFacts
	snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return m.store.Save(snapshot)
}

func cloneFacts(facts []string) []string {
	if len(facts) == 0 {
		return nil
	}
	copied := make([]string, len(facts))
	copy(copied, facts)
	return copied
}
