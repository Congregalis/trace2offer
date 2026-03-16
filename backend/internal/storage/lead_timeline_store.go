package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"trace2offer/backend/internal/model"
)

// LeadTimelineStore persists stage history for each lead.
type LeadTimelineStore interface {
	List() []model.LeadTimeline
	Get(leadID string) (model.LeadTimeline, bool)
	Save(timeline model.LeadTimeline) (model.LeadTimeline, error)
	Delete(leadID string) (bool, error)
}

type leadTimelineDump struct {
	Timelines []model.LeadTimeline `json:"timelines"`
}

// FileLeadTimelineStore persists lead timelines in a local JSON file.
type FileLeadTimelineStore struct {
	path      string
	mu        sync.RWMutex
	timelines map[string]model.LeadTimeline
}

func NewFileLeadTimelineStore(path string) (*FileLeadTimelineStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create lead timeline data dir: %w", err)
	}

	store := &FileLeadTimelineStore{
		path:      path,
		timelines: map[string]model.LeadTimeline{},
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat lead timeline data file: %w", err)
	}

	if err := store.loadFromDisk(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileLeadTimelineStore) List() []model.LeadTimeline {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]model.LeadTimeline, 0, len(s.timelines))
	for _, timeline := range s.timelines {
		list = append(list, cloneTimeline(timeline))
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].UpdatedAt == list[j].UpdatedAt {
			return list[i].LeadID < list[j].LeadID
		}
		return list[i].UpdatedAt > list[j].UpdatedAt
	})

	return list
}

func (s *FileLeadTimelineStore) Get(leadID string) (model.LeadTimeline, bool) {
	normalizedLeadID := strings.TrimSpace(leadID)
	if normalizedLeadID == "" {
		return model.LeadTimeline{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	timeline, ok := s.timelines[normalizedLeadID]
	if !ok {
		return model.LeadTimeline{}, false
	}

	return cloneTimeline(timeline), true
}

func (s *FileLeadTimelineStore) Save(timeline model.LeadTimeline) (model.LeadTimeline, error) {
	normalizedLeadID := strings.TrimSpace(timeline.LeadID)
	if normalizedLeadID == "" {
		return model.LeadTimeline{}, fmt.Errorf("lead_id is required")
	}

	normalized := sanitizeTimeline(model.LeadTimeline{
		LeadID:    normalizedLeadID,
		Stages:    append([]model.LeadTimelineStage(nil), timeline.Stages...),
		UpdatedAt: strings.TrimSpace(timeline.UpdatedAt),
	})

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(normalized.Stages) == 0 {
		delete(s.timelines, normalizedLeadID)
		if err := s.saveLocked(); err != nil {
			return model.LeadTimeline{}, err
		}
		return model.LeadTimeline{}, nil
	}

	s.timelines[normalizedLeadID] = normalized
	if err := s.saveLocked(); err != nil {
		return model.LeadTimeline{}, err
	}

	return cloneTimeline(normalized), nil
}

func (s *FileLeadTimelineStore) Delete(leadID string) (bool, error) {
	normalizedLeadID := strings.TrimSpace(leadID)
	if normalizedLeadID == "" {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.timelines[normalizedLeadID]; !ok {
		return false, nil
	}

	delete(s.timelines, normalizedLeadID)
	if err := s.saveLocked(); err != nil {
		return false, err
	}

	return true, nil
}

func (s *FileLeadTimelineStore) loadFromDisk() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("read lead timeline data file: %w", err)
	}

	if len(data) == 0 {
		s.timelines = map[string]model.LeadTimeline{}
		return nil
	}

	var dump leadTimelineDump
	if err := json.Unmarshal(data, &dump); err != nil {
		return fmt.Errorf("decode lead timeline data file: %w", err)
	}

	timelines := make(map[string]model.LeadTimeline, len(dump.Timelines))
	for _, item := range dump.Timelines {
		normalizedLeadID := strings.TrimSpace(item.LeadID)
		if normalizedLeadID == "" {
			continue
		}
		item.LeadID = normalizedLeadID
		normalized := sanitizeTimeline(item)
		if len(normalized.Stages) == 0 {
			continue
		}
		timelines[normalizedLeadID] = normalized
	}

	s.timelines = timelines
	return nil
}

func (s *FileLeadTimelineStore) saveLocked() error {
	list := make([]model.LeadTimeline, 0, len(s.timelines))
	for _, timeline := range s.timelines {
		normalized := sanitizeTimeline(cloneTimeline(timeline))
		if len(normalized.Stages) == 0 {
			continue
		}
		list = append(list, normalized)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].LeadID < list[j].LeadID
	})

	payload, err := json.MarshalIndent(leadTimelineDump{Timelines: list}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lead timeline data file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp lead timeline data file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace lead timeline data file: %w", err)
	}

	return nil
}

func cloneTimeline(timeline model.LeadTimeline) model.LeadTimeline {
	copied := timeline
	copied.Stages = append([]model.LeadTimelineStage(nil), timeline.Stages...)
	return copied
}

func sanitizeTimeline(timeline model.LeadTimeline) model.LeadTimeline {
	timeline.LeadID = strings.TrimSpace(timeline.LeadID)
	timeline.UpdatedAt = strings.TrimSpace(timeline.UpdatedAt)

	stages := make([]model.LeadTimelineStage, 0, len(timeline.Stages))
	stageIndexMap := map[string]int{}
	for _, stage := range timeline.Stages {
		normalizedStage := model.LeadTimelineStage{
			Stage:     strings.TrimSpace(stage.Stage),
			StartedAt: strings.TrimSpace(stage.StartedAt),
			EndedAt:   strings.TrimSpace(stage.EndedAt),
		}
		if normalizedStage.Stage == "" || normalizedStage.StartedAt == "" {
			continue
		}

		index, exists := stageIndexMap[normalizedStage.Stage]
		if !exists {
			stageIndexMap[normalizedStage.Stage] = len(stages)
			stages = append(stages, normalizedStage)
			continue
		}

		existing := stages[index]
		existing.StartedAt = minTimestamp(existing.StartedAt, normalizedStage.StartedAt)
		existing.EndedAt = mergeEndedAt(existing.EndedAt, normalizedStage.EndedAt)
		stages[index] = existing
	}
	timeline.Stages = stages

	return timeline
}

func minTimestamp(left string, right string) string {
	leftNorm := normalizeRFC3339OrKeep(left)
	rightNorm := normalizeRFC3339OrKeep(right)
	if leftNorm == "" {
		return rightNorm
	}
	if rightNorm == "" {
		return leftNorm
	}
	if leftNorm <= rightNorm {
		return leftNorm
	}
	return rightNorm
}

func mergeEndedAt(left string, right string) string {
	leftNorm := normalizeRFC3339OrKeep(left)
	rightNorm := normalizeRFC3339OrKeep(right)
	if leftNorm == "" || rightNorm == "" {
		return ""
	}
	if leftNorm >= rightNorm {
		return leftNorm
	}
	return rightNorm
}

func normalizeRFC3339OrKeep(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC().Format(time.RFC3339)
	}
	return trimmed
}
