package timeline

import (
	"testing"

	"trace2offer/backend/internal/model"
)

func TestServiceTracksStatusTransitions(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	service.OnLeadCreated(model.Lead{
		ID:        "lead_1",
		Status:    "new",
		CreatedAt: "2026-03-16T00:00:00Z",
	})

	service.OnLeadUpdated(
		model.Lead{ID: "lead_1", Status: "new", CreatedAt: "2026-03-16T00:00:00Z", UpdatedAt: "2026-03-16T00:00:00Z"},
		model.Lead{ID: "lead_1", Status: "preparing", UpdatedAt: "2026-03-17T00:00:00Z"},
	)

	timelineItem, ok := repo.Get("lead_1")
	if !ok {
		t.Fatal("expected timeline exists")
	}
	if len(timelineItem.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(timelineItem.Stages))
	}

	if timelineItem.Stages[0].Stage != "new" {
		t.Fatalf("expected first stage new, got %q", timelineItem.Stages[0].Stage)
	}
	if timelineItem.Stages[0].StartedAt != "2026-03-16T00:00:00Z" {
		t.Fatalf("expected first stage start retained, got %q", timelineItem.Stages[0].StartedAt)
	}
	if timelineItem.Stages[0].EndedAt != "2026-03-17T00:00:00Z" {
		t.Fatalf("expected first stage end set, got %q", timelineItem.Stages[0].EndedAt)
	}

	if timelineItem.Stages[1].Stage != "preparing" {
		t.Fatalf("expected second stage preparing, got %q", timelineItem.Stages[1].Stage)
	}
	if timelineItem.Stages[1].StartedAt != "2026-03-17T00:00:00Z" {
		t.Fatalf("expected second stage start set, got %q", timelineItem.Stages[1].StartedAt)
	}
	if timelineItem.Stages[1].EndedAt != "" {
		t.Fatalf("expected second stage open, got ended_at=%q", timelineItem.Stages[1].EndedAt)
	}
}

func TestServiceReenterStageCreatesNewSegment(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	service.OnLeadCreated(model.Lead{ID: "lead_1", Status: "new", CreatedAt: "2026-03-16T00:00:00Z"})
	service.OnLeadUpdated(
		model.Lead{ID: "lead_1", Status: "new", UpdatedAt: "2026-03-16T00:00:00Z"},
		model.Lead{ID: "lead_1", Status: "preparing", UpdatedAt: "2026-03-17T00:00:00Z"},
	)
	service.OnLeadUpdated(
		model.Lead{ID: "lead_1", Status: "preparing", UpdatedAt: "2026-03-17T00:00:00Z"},
		model.Lead{ID: "lead_1", Status: "new", UpdatedAt: "2026-03-18T00:00:00Z"},
	)

	timelineItem, _ := repo.Get("lead_1")
	if len(timelineItem.Stages) != 2 {
		t.Fatalf("expected 2 unique stages after re-enter, got %d", len(timelineItem.Stages))
	}
	if timelineItem.Stages[0].Stage != "new" {
		t.Fatalf("expected first stage new, got %q", timelineItem.Stages[0].Stage)
	}
	if timelineItem.Stages[0].StartedAt != "2026-03-16T00:00:00Z" {
		t.Fatalf("expected new stage keeps earliest start, got %q", timelineItem.Stages[0].StartedAt)
	}
	if timelineItem.Stages[0].EndedAt != "" {
		t.Fatalf("expected new stage reopened as open interval, got ended_at=%q", timelineItem.Stages[0].EndedAt)
	}
	if timelineItem.Stages[1].Stage != "preparing" {
		t.Fatalf("expected second stage preparing, got %q", timelineItem.Stages[1].Stage)
	}
	if timelineItem.Stages[1].EndedAt != "2026-03-18T00:00:00Z" {
		t.Fatalf("expected preparing stage closed at re-enter time, got %q", timelineItem.Stages[1].EndedAt)
	}
}

func TestServiceUpdateWithoutStatusChangeDoesNotSplitStage(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	service.OnLeadCreated(model.Lead{ID: "lead_1", Status: "new", CreatedAt: "2026-03-16T00:00:00Z"})
	service.OnLeadUpdated(
		model.Lead{ID: "lead_1", Status: "new", UpdatedAt: "2026-03-16T05:00:00Z"},
		model.Lead{ID: "lead_1", Status: "new", UpdatedAt: "2026-03-17T05:00:00Z"},
	)

	timelineItem, _ := repo.Get("lead_1")
	if len(timelineItem.Stages) != 1 {
		t.Fatalf("expected still 1 stage when status unchanged, got %d", len(timelineItem.Stages))
	}
	if timelineItem.Stages[0].EndedAt != "" {
		t.Fatalf("expected stage remains open, got ended_at=%q", timelineItem.Stages[0].EndedAt)
	}
}

func TestServiceDelete(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)
	service.OnLeadCreated(model.Lead{ID: "lead_1", Status: "new", CreatedAt: "2026-03-16T00:00:00Z"})
	service.OnLeadDeleted("lead_1")
	if _, ok := repo.Get("lead_1"); ok {
		t.Fatal("expected timeline removed on lead delete")
	}
}

func TestServiceTerminalCurrentStageHasEndedAt(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	service.OnLeadCreated(model.Lead{ID: "lead_1", Status: "new", CreatedAt: "2026-03-16T00:00:00Z"})
	service.OnLeadUpdated(
		model.Lead{ID: "lead_1", Status: "new", UpdatedAt: "2026-03-16T00:00:00Z"},
		model.Lead{ID: "lead_1", Status: "rejected", UpdatedAt: "2026-03-17T00:00:00Z"},
	)

	timelineItem, _ := repo.Get("lead_1")
	if len(timelineItem.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(timelineItem.Stages))
	}
	if timelineItem.Stages[1].Stage != "rejected" {
		t.Fatalf("expected second stage rejected, got %q", timelineItem.Stages[1].Stage)
	}
	if timelineItem.Stages[1].EndedAt == "" {
		t.Fatal("expected terminal stage rejected has ended_at")
	}
	if timelineItem.Stages[1].EndedAt != "2026-03-17T00:00:00Z" {
		t.Fatalf("expected terminal ended_at at transition time, got %q", timelineItem.Stages[1].EndedAt)
	}

	// Same-status update should not clear or push ended_at.
	service.OnLeadUpdated(
		model.Lead{ID: "lead_1", Status: "rejected", UpdatedAt: "2026-03-17T00:00:00Z"},
		model.Lead{ID: "lead_1", Status: "rejected", UpdatedAt: "2026-03-19T00:00:00Z"},
	)
	timelineItem, _ = repo.Get("lead_1")
	if timelineItem.Stages[1].EndedAt != "2026-03-17T00:00:00Z" {
		t.Fatalf("expected terminal ended_at unchanged on same status update, got %q", timelineItem.Stages[1].EndedAt)
	}
}

type stubRepository struct {
	items map[string]model.LeadTimeline
}

func (s *stubRepository) List() []model.LeadTimeline {
	list := make([]model.LeadTimeline, 0, len(s.items))
	for _, item := range s.items {
		copied := item
		copied.Stages = append([]model.LeadTimelineStage(nil), item.Stages...)
		list = append(list, copied)
	}
	return list
}

func (s *stubRepository) Get(leadID string) (model.LeadTimeline, bool) {
	if s.items == nil {
		return model.LeadTimeline{}, false
	}
	item, ok := s.items[leadID]
	if !ok {
		return model.LeadTimeline{}, false
	}
	copied := item
	copied.Stages = append([]model.LeadTimelineStage(nil), item.Stages...)
	return copied, true
}

func (s *stubRepository) Save(timeline model.LeadTimeline) (model.LeadTimeline, error) {
	if s.items == nil {
		s.items = map[string]model.LeadTimeline{}
	}
	copied := timeline
	copied.Stages = append([]model.LeadTimelineStage(nil), timeline.Stages...)
	s.items[timeline.LeadID] = copied
	return copied, nil
}

func (s *stubRepository) Delete(leadID string) (bool, error) {
	if s.items == nil {
		return false, nil
	}
	if _, ok := s.items[leadID]; !ok {
		return false, nil
	}
	delete(s.items, leadID)
	return true, nil
}
