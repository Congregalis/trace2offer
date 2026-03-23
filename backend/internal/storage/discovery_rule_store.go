package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"trace2offer/backend/internal/model"
)

// DiscoveryRuleStore persists discovery rules.
type DiscoveryRuleStore interface {
	List() []model.DiscoveryRule
	Create(input model.DiscoveryRuleMutationInput) (model.DiscoveryRule, error)
	Update(id string, input model.DiscoveryRuleMutationInput) (model.DiscoveryRule, bool, error)
	Delete(id string) (bool, error)
}

// FileDiscoveryRuleStore persists discovery rules in local JSON.
type FileDiscoveryRuleStore struct {
	path  string
	mu    sync.RWMutex
	rules []model.DiscoveryRule
}

func NewFileDiscoveryRuleStore(path string) (*FileDiscoveryRuleStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create discovery rule dir: %w", err)
	}

	store := &FileDiscoveryRuleStore{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		store.rules = []model.DiscoveryRule{}
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat discovery rule file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}
	if store.rules == nil {
		store.rules = []model.DiscoveryRule{}
	}

	return store, nil
}

func (s *FileDiscoveryRuleStore) List() []model.DiscoveryRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copied := make([]model.DiscoveryRule, len(s.rules))
	copy(copied, s.rules)
	return copied
}

func (s *FileDiscoveryRuleStore) Create(input model.DiscoveryRuleMutationInput) (model.DiscoveryRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	rule := model.DiscoveryRule{
		ID:              newID("drule"),
		Name:            input.Name,
		FeedURL:         input.FeedURL,
		Source:          input.Source,
		DefaultLocation: input.DefaultLocation,
		IncludeKeywords: append([]string(nil), input.IncludeKeywords...),
		ExcludeKeywords: append([]string(nil), input.ExcludeKeywords...),
		Enabled:         enabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.rules = append(s.rules, rule)
	if err := s.saveLocked(); err != nil {
		return model.DiscoveryRule{}, err
	}
	return rule, nil
}

func (s *FileDiscoveryRuleStore) Update(id string, input model.DiscoveryRuleMutationInput) (model.DiscoveryRule, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.rules {
		if s.rules[i].ID != id {
			continue
		}

		updated := s.rules[i]
		updated.Name = input.Name
		updated.FeedURL = input.FeedURL
		updated.Source = input.Source
		updated.DefaultLocation = input.DefaultLocation
		updated.IncludeKeywords = append([]string(nil), input.IncludeKeywords...)
		updated.ExcludeKeywords = append([]string(nil), input.ExcludeKeywords...)
		if input.Enabled != nil {
			updated.Enabled = *input.Enabled
		}
		if updated.CreatedAt == "" {
			updated.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		updated.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		s.rules[i] = updated

		if err := s.saveLocked(); err != nil {
			return model.DiscoveryRule{}, true, err
		}
		return updated, true, nil
	}

	return model.DiscoveryRule{}, false, nil
}

func (s *FileDiscoveryRuleStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.rules {
		if s.rules[i].ID != id {
			continue
		}
		s.rules = append(s.rules[:i], s.rules[i+1:]...)
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *FileDiscoveryRuleStore) loadFromDisk() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read discovery rule file: %w", err)
	}
	if len(data) == 0 {
		s.rules = []model.DiscoveryRule{}
		return nil
	}

	var rules []model.DiscoveryRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("decode discovery rule file: %w", err)
	}
	if rules == nil {
		rules = []model.DiscoveryRule{}
	}
	s.rules = rules
	return nil
}

func (s *FileDiscoveryRuleStore) saveLocked() error {
	payload, err := json.MarshalIndent(s.rules, "", "  ")
	if err != nil {
		return fmt.Errorf("encode discovery rule file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp discovery rule file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace discovery rule file: %w", err)
	}
	return nil
}
