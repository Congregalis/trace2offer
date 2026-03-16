package storage

import (
	"path/filepath"
	"testing"

	"trace2offer/backend/internal/model"
)

func TestFileLeadTimelineStoreSaveListGetDelete(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "lead_timelines.json")
	store, err := NewFileLeadTimelineStore(path)
	if err != nil {
		t.Fatalf("init timeline store: %v", err)
	}

	saved, err := store.Save(model.LeadTimeline{
		LeadID: "lead_1",
		Stages: []model.LeadTimelineStage{
			{Stage: "new", StartedAt: "2026-04-01T09:00:00Z", EndedAt: "2026-04-05T09:00:00Z"},
			{Stage: "preparing", StartedAt: "2026-04-05T09:00:00Z"},
		},
		UpdatedAt: "2026-04-05T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("save timeline: %v", err)
	}
	if saved.LeadID != "lead_1" {
		t.Fatalf("expected lead_1, got %q", saved.LeadID)
	}
	if len(saved.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(saved.Stages))
	}

	item, ok := store.Get("lead_1")
	if !ok {
		t.Fatal("expected get success")
	}
	if len(item.Stages) != 2 {
		t.Fatalf("expected get stages=2, got %d", len(item.Stages))
	}

	list := store.List()
	if len(list) != 1 {
		t.Fatalf("expected list size 1, got %d", len(list))
	}

	reloaded, err := NewFileLeadTimelineStore(path)
	if err != nil {
		t.Fatalf("reload timeline store: %v", err)
	}
	item, ok = reloaded.Get("lead_1")
	if !ok {
		t.Fatal("expected persisted timeline exists")
	}
	if len(item.Stages) != 2 {
		t.Fatalf("expected persisted stages=2, got %d", len(item.Stages))
	}

	deleted, err := reloaded.Delete("lead_1")
	if err != nil {
		t.Fatalf("delete timeline: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete success")
	}

	if _, ok := reloaded.Get("lead_1"); ok {
		t.Fatal("expected timeline not found after delete")
	}
}

func TestFileLeadTimelineStoreMergesDuplicateStages(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "lead_timelines.json")
	store, err := NewFileLeadTimelineStore(path)
	if err != nil {
		t.Fatalf("init timeline store: %v", err)
	}

	_, err = store.Save(model.LeadTimeline{
		LeadID: "lead_1",
		Stages: []model.LeadTimelineStage{
			{Stage: "new", StartedAt: "2026-04-01T09:00:00Z", EndedAt: "2026-04-03T09:00:00Z"},
			{Stage: "new", StartedAt: "2026-04-05T09:00:00Z", EndedAt: ""},
		},
		UpdatedAt: "2026-04-05T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("save timeline: %v", err)
	}

	item, ok := store.Get("lead_1")
	if !ok {
		t.Fatal("expected timeline exists")
	}
	if len(item.Stages) != 1 {
		t.Fatalf("expected duplicate stages merged, got %d", len(item.Stages))
	}
	if item.Stages[0].StartedAt != "2026-04-01T09:00:00Z" {
		t.Fatalf("expected earliest started_at kept, got %q", item.Stages[0].StartedAt)
	}
	if item.Stages[0].EndedAt != "" {
		t.Fatalf("expected merged stage reopened, got ended_at=%q", item.Stages[0].EndedAt)
	}
}
