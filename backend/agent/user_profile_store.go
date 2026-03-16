package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var ErrUserProfileStoreUnavailable = errors.New("user profile store is unavailable")

// UserProfileStore persists user profile data.
type UserProfileStore interface {
	Load() (UserProfile, bool, error)
	Save(profile UserProfile) error
}

// UserProfileManager wraps persistence and prompt rendering.
type UserProfileManager struct {
	store UserProfileStore
}

func NewUserProfileManager(store UserProfileStore) *UserProfileManager {
	return &UserProfileManager{store: store}
}

func (m *UserProfileManager) Get() (UserProfile, error) {
	if m == nil || m.store == nil {
		return UserProfile{}, ErrUserProfileStoreUnavailable
	}

	profile, ok, err := m.store.Load()
	if err != nil {
		return UserProfile{}, err
	}
	if !ok {
		return normalizeUserProfile(UserProfile{}), nil
	}
	return normalizeUserProfile(profile), nil
}

func (m *UserProfileManager) Update(profile UserProfile) (UserProfile, error) {
	if m == nil || m.store == nil {
		return UserProfile{}, ErrUserProfileStoreUnavailable
	}

	normalized := normalizeUserProfile(profile)
	if err := m.store.Save(normalized); err != nil {
		return UserProfile{}, err
	}
	return normalized, nil
}

func (m *UserProfileManager) MergeImported(profile UserProfile) (UserProfile, UserProfile, error) {
	if m == nil || m.store == nil {
		return UserProfile{}, UserProfile{}, ErrUserProfileStoreUnavailable
	}

	current, err := m.Get()
	if err != nil {
		return UserProfile{}, UserProfile{}, err
	}
	imported := normalizeUserProfile(profile)
	merged := mergeImportedUserProfile(current, imported)
	if err := m.store.Save(merged); err != nil {
		return UserProfile{}, UserProfile{}, err
	}
	return merged, imported, nil
}

func (m *UserProfileManager) PromptBlock() string {
	if m == nil || m.store == nil {
		return "[USER]\\nprofile_status: 用户能力画像存储不可用"
	}

	profile, err := m.Get()
	if err != nil {
		return "[USER]\\nprofile_status: 用户能力画像读取失败"
	}
	return formatUserProfilePrompt(profile)
}

// FileUserProfileStore persists user profile in one local JSON file.
type FileUserProfileStore struct {
	path    string
	mu      sync.RWMutex
	profile UserProfile
	loaded  bool
}

func NewFileUserProfileStore(path string) (*FileUserProfileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create user profile data dir: %w", err)
	}

	store := &FileUserProfileStore{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := store.saveLocked(normalizeUserProfile(UserProfile{})); err != nil {
			return nil, err
		}
		store.loaded = true
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat user profile data file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileUserProfileStore) Load() (UserProfile, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.loaded {
		return UserProfile{}, false, nil
	}
	return normalizeUserProfile(s.profile), true, nil
}

func (s *FileUserProfileStore) Save(profile UserProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(profile)
}

func (s *FileUserProfileStore) loadFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read user profile data file: %w", err)
	}
	if len(payload) == 0 {
		s.profile = normalizeUserProfile(UserProfile{})
		s.loaded = true
		return nil
	}

	var parsed UserProfile
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return fmt.Errorf("decode user profile data file: %w", err)
	}
	s.profile = normalizeUserProfile(parsed)
	s.loaded = true
	return nil
}

func (s *FileUserProfileStore) saveLocked(profile UserProfile) error {
	normalized := normalizeUserProfile(profile)
	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode user profile data file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp user profile data file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace user profile data file: %w", err)
	}

	s.profile = normalized
	s.loaded = true
	return nil
}
