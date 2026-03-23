package storage

import (
	"path/filepath"
	"testing"

	"trace2offer/backend/internal/model"
)

func TestFileDiscoveryRuleStoreCRUD(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "discovery_rules.json")
	store, err := NewFileDiscoveryRuleStore(path)
	if err != nil {
		t.Fatalf("new discovery rule store error: %v", err)
	}

	if got := store.List(); len(got) != 0 {
		t.Fatalf("expected empty rule list, got %d", len(got))
	}

	enabled := true
	created, err := store.Create(model.DiscoveryRuleMutationInput{
		Name:            "remote golang",
		FeedURL:         "https://example.com/jobs.rss",
		Source:          "RSS",
		DefaultLocation: "Remote",
		IncludeKeywords: []string{"golang", "backend"},
		ExcludeKeywords: []string{"intern"},
		Enabled:         &enabled,
	})
	if err != nil {
		t.Fatalf("create discovery rule error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created rule id")
	}

	disabled := false
	updated, found, err := store.Update(created.ID, model.DiscoveryRuleMutationInput{
		Name:            "remote golang updated",
		FeedURL:         "https://example.com/jobs-updated.rss",
		Source:          "RSS",
		DefaultLocation: "US Remote",
		IncludeKeywords: []string{"golang"},
		ExcludeKeywords: []string{"contract"},
		Enabled:         &disabled,
	})
	if err != nil {
		t.Fatalf("update discovery rule error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true on update")
	}
	if updated.Name != "remote golang updated" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if updated.Enabled {
		t.Fatal("expected updated enabled=false")
	}

	deleted, err := store.Delete(created.ID)
	if err != nil {
		t.Fatalf("delete discovery rule error: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}
}
