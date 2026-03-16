package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type fileDump struct {
	Sessions []Session `json:"sessions"`
}

// FileStore persists sessions as one JSON file per session under a directory.
type FileStore struct {
	dirPath string
	mu      sync.RWMutex
}

func NewFileStore(path string) (*FileStore, error) {
	dirPath, legacyPath, err := resolveStorePaths(path)
	if err != nil {
		return nil, err
	}
	store := &FileStore{
		dirPath: dirPath,
	}
	if err := os.MkdirAll(store.dirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create session data dir: %w", err)
	}
	if legacyPath != "" {
		if err := store.migrateLegacyFile(legacyPath); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *FileStore) Load(id string) (Session, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID := strings.TrimSpace(id)
	if sessionID == "" {
		return Session{}, false, nil
	}
	payload, err := os.ReadFile(s.sessionFilePath(sessionID))
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, fmt.Errorf("read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return Session{}, false, fmt.Errorf("decode session file: %w", err)
	}
	if strings.TrimSpace(session.ID) == "" {
		session.ID = sessionID
	}
	return cloneSession(session), true, nil
}

func (s *FileStore) Save(data Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := strings.TrimSpace(data.ID)
	if sessionID == "" {
		return errors.New("session id is required")
	}

	session := cloneSession(data)
	session.ID = sessionID
	return s.saveSessionLocked(session)
}

func (s *FileStore) List() ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dirPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read session data dir: %w", err)
	}

	sessions := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if !strings.EqualFold(filepath.Ext(name), ".json") {
			continue
		}

		path := filepath.Join(s.dirPath, name)
		payload, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read session file %q: %w", name, readErr)
		}
		if len(strings.TrimSpace(string(payload))) == 0 {
			continue
		}

		var item Session
		if unmarshalErr := json.Unmarshal(payload, &item); unmarshalErr != nil {
			return nil, fmt.Errorf("decode session file %q: %w", name, unmarshalErr)
		}
		if strings.TrimSpace(item.ID) == "" {
			item.ID = decodeSessionID(name)
		}
		sessions = append(sessions, cloneSession(item))
	}

	return sessions, nil
}

func resolveStorePaths(path string) (string, string, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return "", "", errors.New("session data path is required")
	}
	cleaned = filepath.Clean(cleaned)
	if strings.EqualFold(filepath.Ext(cleaned), ".json") {
		return filepath.Join(filepath.Dir(cleaned), "sessions"), cleaned, nil
	}

	legacyPath := ""
	if strings.EqualFold(filepath.Base(cleaned), "sessions") {
		legacyPath = filepath.Join(filepath.Dir(cleaned), "agent_sessions.json")
	}
	return cleaned, legacyPath, nil
}

func (s *FileStore) migrateLegacyFile(legacyPath string) error {
	payload, err := os.ReadFile(legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read legacy session data file: %w", err)
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return s.archiveLegacyFile(legacyPath)
	}

	var dump fileDump
	if err := json.Unmarshal(payload, &dump); err != nil {
		return fmt.Errorf("decode legacy session data file: %w", err)
	}
	for _, item := range dump.Sessions {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		if err := s.Save(item); err != nil {
			return fmt.Errorf("migrate legacy session %q: %w", item.ID, err)
		}
	}

	return s.archiveLegacyFile(legacyPath)
}

func (s *FileStore) archiveLegacyFile(legacyPath string) error {
	backupPath := legacyPath + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		backupPath = fmt.Sprintf("%s.%d.bak", legacyPath, time.Now().UnixNano())
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat legacy session backup: %w", err)
	}
	if err := os.Rename(legacyPath, backupPath); err != nil {
		return fmt.Errorf("archive legacy session file: %w", err)
	}
	return nil
}

func (s *FileStore) saveSessionLocked(session Session) error {
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session file: %w", err)
	}

	path := s.sessionFilePath(session.ID)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp session file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace session file: %w", err)
	}

	return nil
}

func (s *FileStore) sessionFilePath(sessionID string) string {
	encoded := url.PathEscape(strings.TrimSpace(sessionID))
	if encoded == "" {
		encoded = "unknown"
	}
	return filepath.Join(s.dirPath, encoded+".json")
}

func decodeSessionID(fileName string) string {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	decoded, err := url.PathUnescape(base)
	if err != nil {
		return base
	}
	return decoded
}
