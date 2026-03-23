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

// CandidateStore defines persistence behavior for candidate records.
type CandidateStore interface {
	List() []model.Candidate
	Create(input model.CandidateMutationInput) (model.Candidate, error)
	Update(id string, input model.CandidateMutationInput) (model.Candidate, bool, error)
	Delete(id string) (bool, error)
}

// FileCandidateStore persists candidates in a local JSON file.
type FileCandidateStore struct {
	path       string
	mu         sync.RWMutex
	candidates []model.Candidate
}

func NewFileCandidateStore(path string) (*FileCandidateStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create candidate data dir: %w", err)
	}

	store := &FileCandidateStore{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		store.candidates = []model.Candidate{}
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat candidate data file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}
	if store.candidates == nil {
		store.candidates = []model.Candidate{}
	}

	return store, nil
}

func (s *FileCandidateStore) List() []model.Candidate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copied := make([]model.Candidate, len(s.candidates))
	copy(copied, s.candidates)
	return copied
}

func (s *FileCandidateStore) Create(input model.CandidateMutationInput) (model.Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	candidate := model.Candidate{
		ID:                  newID("cand"),
		Company:             input.Company,
		Position:            input.Position,
		Source:              input.Source,
		Location:            input.Location,
		JDURL:               input.JDURL,
		CompanyWebsiteURL:   input.CompanyWebsiteURL,
		Status:              input.Status,
		MatchScore:          input.MatchScore,
		MatchReasons:        append([]string(nil), input.MatchReasons...),
		RecommendationNotes: input.RecommendationNotes,
		Notes:               input.Notes,
		PromotedLeadID:      input.PromotedLeadID,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	s.candidates = append(s.candidates, candidate)
	if err := s.saveLocked(); err != nil {
		return model.Candidate{}, err
	}
	return candidate, nil
}

func (s *FileCandidateStore) Update(id string, input model.CandidateMutationInput) (model.Candidate, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}

		updated := s.candidates[i]
		updated.Company = input.Company
		updated.Position = input.Position
		updated.Source = input.Source
		updated.Location = input.Location
		updated.JDURL = input.JDURL
		updated.CompanyWebsiteURL = input.CompanyWebsiteURL
		updated.Status = input.Status
		updated.MatchScore = input.MatchScore
		updated.MatchReasons = append([]string(nil), input.MatchReasons...)
		updated.RecommendationNotes = input.RecommendationNotes
		updated.Notes = input.Notes
		updated.PromotedLeadID = input.PromotedLeadID
		if updated.CreatedAt == "" {
			updated.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		updated.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		s.candidates[i] = updated

		if err := s.saveLocked(); err != nil {
			return model.Candidate{}, true, err
		}
		return updated, true, nil
	}

	return model.Candidate{}, false, nil
}

func (s *FileCandidateStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}
		s.candidates = append(s.candidates[:i], s.candidates[i+1:]...)
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *FileCandidateStore) loadFromDisk() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read candidate data file: %w", err)
	}
	if len(data) == 0 {
		s.candidates = []model.Candidate{}
		return nil
	}

	var candidates []model.Candidate
	if err := json.Unmarshal(data, &candidates); err != nil {
		return fmt.Errorf("decode candidate data file: %w", err)
	}
	if candidates == nil {
		candidates = []model.Candidate{}
	}
	s.candidates = candidates
	return nil
}

func (s *FileCandidateStore) saveLocked() error {
	payload, err := json.MarshalIndent(s.candidates, "", "  ")
	if err != nil {
		return fmt.Errorf("encode candidate data file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp candidate data file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace candidate data file: %w", err)
	}
	return nil
}
