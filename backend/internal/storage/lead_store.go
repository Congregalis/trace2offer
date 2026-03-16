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

// LeadStore defines the behavior needed by API handlers.
type LeadStore interface {
	List() []model.Lead
	Create(input model.LeadMutationInput) (model.Lead, error)
	Update(id string, input model.LeadMutationInput) (model.Lead, bool, error)
	Delete(id string) (bool, error)
}

// FileLeadStore persists leads in a local JSON file.
type FileLeadStore struct {
	path  string
	mu    sync.RWMutex
	leads []model.Lead
}

func NewFileLeadStore(path string) (*FileLeadStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create lead data dir: %w", err)
	}

	store := &FileLeadStore{path: path}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		store.leads = seedLeads()
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat lead data file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}

	if len(store.leads) == 0 {
		store.leads = seedLeads()
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (s *FileLeadStore) List() []model.Lead {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copied := make([]model.Lead, len(s.leads))
	copy(copied, s.leads)
	return copied
}

func (s *FileLeadStore) Create(input model.LeadMutationInput) (model.Lead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	lead := model.Lead{
		ID:                newID("lead"),
		Company:           input.Company,
		Position:          input.Position,
		Source:            input.Source,
		Status:            input.Status,
		Priority:          input.Priority,
		NextAction:        input.NextAction,
		Notes:             input.Notes,
		CompanyWebsiteURL: input.CompanyWebsiteURL,
		JDURL:             input.JDURL,
		Location:          input.Location,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.leads = append(s.leads, lead)
	if err := s.saveLocked(); err != nil {
		return model.Lead{}, err
	}

	return lead, nil
}

func (s *FileLeadStore) Update(id string, input model.LeadMutationInput) (model.Lead, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.leads {
		if s.leads[i].ID != id {
			continue
		}

		updated := s.leads[i]
		updated.Company = input.Company
		updated.Position = input.Position
		updated.Source = input.Source
		updated.Status = input.Status
		updated.Priority = input.Priority
		updated.NextAction = input.NextAction
		updated.Notes = input.Notes
		updated.CompanyWebsiteURL = input.CompanyWebsiteURL
		updated.JDURL = input.JDURL
		updated.Location = input.Location
		if updated.CreatedAt == "" {
			updated.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		updated.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		s.leads[i] = updated
		if err := s.saveLocked(); err != nil {
			return model.Lead{}, true, err
		}

		return updated, true, nil
	}

	return model.Lead{}, false, nil
}

func (s *FileLeadStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.leads {
		if s.leads[i].ID != id {
			continue
		}

		s.leads = append(s.leads[:i], s.leads[i+1:]...)
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (s *FileLeadStore) loadFromDisk() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read lead data file: %w", err)
	}

	if len(data) == 0 {
		s.leads = nil
		return nil
	}

	var leads []model.Lead
	if err := json.Unmarshal(data, &leads); err != nil {
		return fmt.Errorf("decode lead data file: %w", err)
	}

	s.leads = leads
	return nil
}

func (s *FileLeadStore) saveLocked() error {
	payload, err := json.MarshalIndent(s.leads, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lead data file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp lead data file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace lead data file: %w", err)
	}

	return nil
}

func seedLeads() []model.Lead {
	now := time.Now().UTC()

	return []model.Lead{
		{
			ID:                newID("lead"),
			Company:           "Stripe",
			Position:          "Backend Engineer",
			Source:            "LinkedIn",
			Status:            "new",
			Priority:          5,
			NextAction:        "完善简历并投递",
			Notes:             "偏好支付基础设施方向",
			CompanyWebsiteURL: "https://stripe.com",
			JDURL:             "https://stripe.com/jobs/listing/backend-engineer/0000000",
			Location:          "San Francisco, CA",
			CreatedAt:         now.Add(-72 * time.Hour).Format(time.RFC3339),
			UpdatedAt:         now.Add(-72 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:                newID("lead"),
			Company:           "Figma",
			Position:          "Platform Engineer",
			Source:            "Referral",
			Status:            "preparing",
			Priority:          4,
			NextAction:        "下周二前跟进内推进度",
			Notes:             "朋友在团队内，可以走 referral",
			CompanyWebsiteURL: "https://www.figma.com",
			JDURL:             "https://boards.greenhouse.io/figma/jobs/0000000",
			Location:          "New York, NY",
			CreatedAt:         now.Add(-48 * time.Hour).Format(time.RFC3339),
			UpdatedAt:         now.Add(-24 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:                newID("lead"),
			Company:           "Datadog",
			Position:          "Software Engineer",
			Source:            "Company Site",
			Status:            "interviewing",
			Priority:          5,
			NextAction:        "准备系统设计面试",
			Notes:             "JD 强调分布式系统经验",
			CompanyWebsiteURL: "https://www.datadoghq.com",
			JDURL:             "https://careers.datadoghq.com/detail/0000000",
			Location:          "Boston, MA",
			CreatedAt:         now.Add(-120 * time.Hour).Format(time.RFC3339),
			UpdatedAt:         now.Add(-6 * time.Hour).Format(time.RFC3339),
		},
	}
}
