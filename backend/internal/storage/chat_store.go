package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ChatStore persists chat sessions in local storage.
type ChatStore interface {
	Append(sessionID string, role string, content string) error
}

type ChatMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type ChatSession struct {
	ID        string        `json:"id"`
	Messages  []ChatMessage `json:"messages"`
	UpdatedAt string        `json:"updated_at"`
}

type chatDump struct {
	Sessions []ChatSession `json:"sessions"`
}

// FileChatStore persists chat sessions to a local JSON file.
type FileChatStore struct {
	path     string
	mu       sync.Mutex
	sessions map[string]ChatSession
}

func NewFileChatStore(path string) (*FileChatStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create chat data dir: %w", err)
	}

	store := &FileChatStore{
		path:     path,
		sessions: map[string]ChatSession{},
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat chat data file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileChatStore) Append(sessionID string, role string, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	session, ok := s.sessions[sessionID]
	if !ok {
		session = ChatSession{ID: sessionID}
	}

	session.Messages = append(session.Messages, ChatMessage{
		Role:      role,
		Content:   content,
		CreatedAt: now,
	})
	session.UpdatedAt = now
	s.sessions[sessionID] = session

	if err := s.saveLocked(); err != nil {
		return err
	}

	return nil
}

func (s *FileChatStore) loadFromDisk() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read chat data file: %w", err)
	}
	if len(data) == 0 {
		s.sessions = map[string]ChatSession{}
		return nil
	}

	var dump chatDump
	if err := json.Unmarshal(data, &dump); err != nil {
		return fmt.Errorf("decode chat data file: %w", err)
	}

	s.sessions = make(map[string]ChatSession, len(dump.Sessions))
	for _, session := range dump.Sessions {
		s.sessions[session.ID] = session
	}

	return nil
}

func (s *FileChatStore) saveLocked() error {
	dump := chatDump{Sessions: make([]ChatSession, 0, len(s.sessions))}
	for _, session := range s.sessions {
		dump.Sessions = append(dump.Sessions, session)
	}
	// Keep file diffs stable.
	sort.Slice(dump.Sessions, func(i, j int) bool {
		return dump.Sessions[i].ID < dump.Sessions[j].ID
	})

	payload, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return fmt.Errorf("encode chat data file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp chat data file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace chat data file: %w", err)
	}

	return nil
}
