package prep

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrPrepDisabled              = errors.New("prep module is disabled")
	ErrTopicStoreUnavailable     = errors.New("prep topic store is unavailable")
	ErrKnowledgeStoreUnavailable = errors.New("prep knowledge store is unavailable")
	ErrTopicAlreadyExists        = errors.New("prep topic already exists")
	ErrDocumentAlreadyExists     = errors.New("prep knowledge document already exists")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if strings.TrimSpace(e.Field) == "" {
		return strings.TrimSpace(e.Message)
	}
	return fmt.Sprintf("%s: %s", strings.TrimSpace(e.Field), strings.TrimSpace(e.Message))
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

var topicKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type topicCatalog struct {
	Topics []Topic `json:"topics"`
}

type TopicStore struct {
	path   string
	mu     sync.RWMutex
	topics []Topic
}

func NewTopicStore(path string) (*TopicStore, error) {
	normalizedPath := filepath.Clean(strings.TrimSpace(path))
	if normalizedPath == "" || normalizedPath == "." {
		return nil, fmt.Errorf("prep topic catalog path is required")
	}

	if err := os.MkdirAll(filepath.Dir(normalizedPath), 0o755); err != nil {
		return nil, fmt.Errorf("create prep topic catalog dir: %w", err)
	}
	if err := ensureTopicCatalogFile(normalizedPath); err != nil {
		return nil, err
	}

	store := &TopicStore{path: normalizedPath}
	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TopicStore) List() []Topic {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	copied := make([]Topic, len(s.topics))
	copy(copied, s.topics)
	return copied
}

func (s *TopicStore) Create(input TopicCreateInput) (Topic, error) {
	if s == nil {
		return Topic{}, ErrTopicStoreUnavailable
	}

	key, name, description, err := normalizeTopicCreateInput(input)
	if err != nil {
		return Topic{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findTopicIndexLocked(key) >= 0 {
		return Topic{}, fmt.Errorf("%w: key=%s", ErrTopicAlreadyExists, key)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	created := Topic{
		Key:         key,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.topics = append(s.topics, created)
	sortTopicsByKey(s.topics)

	if err := s.saveLocked(); err != nil {
		return Topic{}, err
	}
	return created, nil
}

func (s *TopicStore) Update(key string, patch TopicPatchInput) (Topic, bool, error) {
	if s == nil {
		return Topic{}, false, ErrTopicStoreUnavailable
	}

	normalizedKey, err := normalizeTopicKey(key)
	if err != nil {
		return Topic{}, false, err
	}

	nextName, hasName, err := normalizeTopicPatchName(patch.Name)
	if err != nil {
		return Topic{}, false, err
	}
	nextDescription, hasDescription := normalizeTopicPatchDescription(patch.Description)
	if !hasName && !hasDescription {
		return Topic{}, false, &ValidationError{Field: "patch", Message: "at least one field is required"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findTopicIndexLocked(normalizedKey)
	if index < 0 {
		return Topic{}, false, nil
	}

	updated := s.topics[index]
	if hasName {
		updated.Name = nextName
	}
	if hasDescription {
		updated.Description = nextDescription
	}
	updated.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	s.topics[index] = updated

	if err := s.saveLocked(); err != nil {
		return Topic{}, true, err
	}
	return updated, true, nil
}

func (s *TopicStore) Delete(key string) (bool, error) {
	if s == nil {
		return false, ErrTopicStoreUnavailable
	}

	normalizedKey, err := normalizeTopicKey(key)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findTopicIndexLocked(normalizedKey)
	if index < 0 {
		return false, nil
	}
	s.topics = append(s.topics[:index], s.topics[index+1:]...)
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *TopicStore) loadFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read prep topic catalog file: %w", err)
	}
	if len(data) == 0 {
		s.topics = []Topic{}
		return nil
	}

	var catalog topicCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("decode prep topic catalog file: %w", err)
	}

	normalized := make([]Topic, 0, len(catalog.Topics))
	seen := map[string]struct{}{}
	for _, item := range catalog.Topics {
		key, err := normalizeTopicKey(item.Key)
		if err != nil {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = key
		}
		normalized = append(normalized, Topic{
			Key:         key,
			Name:        name,
			Description: strings.TrimSpace(item.Description),
			CreatedAt:   strings.TrimSpace(item.CreatedAt),
			UpdatedAt:   strings.TrimSpace(item.UpdatedAt),
		})
	}
	sortTopicsByKey(normalized)
	s.topics = normalized
	return nil
}

func (s *TopicStore) saveLocked() error {
	catalog := topicCatalog{Topics: s.topics}
	payload, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode prep topic catalog file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp prep topic catalog file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace prep topic catalog file: %w", err)
	}
	return nil
}

func (s *TopicStore) findTopicIndexLocked(key string) int {
	for i := range s.topics {
		if s.topics[i].Key == key {
			return i
		}
	}
	return -1
}

func sortTopicsByKey(items []Topic) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
}

func normalizeTopicCreateInput(input TopicCreateInput) (string, string, string, error) {
	key, err := normalizeTopicKey(input.Key)
	if err != nil {
		return "", "", "", err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return "", "", "", &ValidationError{Field: "name", Message: "name is required"}
	}
	return key, name, strings.TrimSpace(input.Description), nil
}

func normalizeTopicPatchName(raw *string) (string, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return "", false, &ValidationError{Field: "name", Message: "name cannot be empty"}
	}
	return value, true, nil
}

func normalizeTopicPatchDescription(raw *string) (string, bool) {
	if raw == nil {
		return "", false
	}
	return strings.TrimSpace(*raw), true
}

func normalizeTopicKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return "", &ValidationError{Field: "key", Message: "key is required"}
	}
	if !topicKeyPattern.MatchString(key) {
		return "", &ValidationError{Field: "key", Message: "key must match ^[a-z0-9][a-z0-9_-]{0,63}$"}
	}
	return key, nil
}
