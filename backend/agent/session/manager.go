package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var ErrStoreUnavailable = errors.New("session store is unavailable")

// Store persists sessions.
type Store interface {
	Load(id string) (Session, bool, error)
	Save(data Session) error
	List() ([]Session, error)
}

// Manager manages session lifecycle and message history.
type Manager struct {
	store Store
}

func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

func (m *Manager) Ensure(sessionID string) (Session, error) {
	if m == nil || m.store == nil {
		return Session{}, ErrStoreUnavailable
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = newSessionID()
	}

	session, ok, err := m.store.Load(sessionID)
	if err != nil {
		return Session{}, err
	}
	if ok {
		return cloneSession(session), nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session = Session{ID: sessionID, UpdatedAt: now}
	if err := m.store.Save(session); err != nil {
		return Session{}, err
	}
	return cloneSession(session), nil
}

func (m *Manager) Load(sessionID string) (Session, bool, error) {
	if m == nil || m.store == nil {
		return Session{}, false, ErrStoreUnavailable
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Session{}, false, nil
	}

	session, ok, err := m.store.Load(sessionID)
	if err != nil {
		return Session{}, false, err
	}
	if !ok {
		return Session{}, false, nil
	}
	return cloneSession(session), true, nil
}

func (m *Manager) Append(sessionID string, role string, content string) (Session, error) {
	if m == nil || m.store == nil {
		return Session{}, ErrStoreUnavailable
	}

	role = strings.TrimSpace(role)
	if role == "" {
		return Session{}, errors.New("message role is required")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return Session{}, errors.New("message content is required")
	}

	session, err := m.Ensure(sessionID)
	if err != nil {
		return Session{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session.Messages = append(session.Messages, Message{
		Role:      role,
		Content:   content,
		CreatedAt: now,
	})
	session.UpdatedAt = now

	if err := m.store.Save(session); err != nil {
		return Session{}, err
	}

	return cloneSession(session), nil
}

func (m *Manager) Messages(sessionID string) ([]Message, error) {
	session, err := m.Ensure(sessionID)
	if err != nil {
		return nil, err
	}
	return cloneMessages(session.Messages), nil
}

func (m *Manager) List() ([]Session, error) {
	if m == nil || m.store == nil {
		return nil, ErrStoreUnavailable
	}

	items, err := m.store.List()
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	cloned := make([]Session, len(items))
	for index := range items {
		cloned[index] = cloneSession(items[index])
	}

	sort.Slice(cloned, func(i, j int) bool {
		left := parseTimestamp(cloned[i].UpdatedAt)
		right := parseTimestamp(cloned[j].UpdatedAt)
		if left != right {
			return left.After(right)
		}
		return cloned[i].ID < cloned[j].ID
	})
	return cloned, nil
}

func cloneSession(data Session) Session {
	return Session{
		ID:        data.ID,
		Messages:  cloneMessages(data.Messages),
		UpdatedAt: data.UpdatedAt,
	}
}

func cloneMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}
	copied := make([]Message, len(messages))
	copy(copied, messages)
	return copied
}

func newSessionID() string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return "session_" + hex.EncodeToString(random[:])
}

func parseTimestamp(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}
