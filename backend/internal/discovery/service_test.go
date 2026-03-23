package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trace2offer/backend/internal/candidate"
	"trace2offer/backend/internal/model"
	"trace2offer/backend/internal/storage"
)

func TestRunOnceIngestsRSSIntoCandidates(t *testing.T) {
	t.Parallel()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Jobs</title>
    <item>
      <title>Backend Engineer at Example</title>
      <link>https://jobs.example.com/backend-1?utm_source=rss</link>
      <description>Go distributed systems cloud platform</description>
    </item>
    <item>
      <title>Marketing Specialist at Example</title>
      <link>https://jobs.example.com/marketing-1</link>
      <description>marketing campaign</description>
    </item>
  </channel>
</rss>`))
	}))
	defer feedServer.Close()

	ruleStore, err := storage.NewFileDiscoveryRuleStore(t.TempDir() + "/rules.json")
	if err != nil {
		t.Fatalf("new discovery rule store error: %v", err)
	}
	candidateStore, err := storage.NewFileCandidateStore(t.TempDir() + "/candidates.json")
	if err != nil {
		t.Fatalf("new candidate store error: %v", err)
	}
	candidateService := candidate.NewService(candidateStore, nil)
	service := NewService(ruleStore, candidateService)

	enabled := true
	_, err = service.CreateRule(model.DiscoveryRuleMutationInput{
		Name:            "Example Jobs",
		FeedURL:         feedServer.URL,
		Source:          "rss:example",
		DefaultLocation: "Remote",
		IncludeKeywords: []string{"go", "backend"},
		ExcludeKeywords: []string{"marketing"},
		Enabled:         &enabled,
	})
	if err != nil {
		t.Fatalf("create rule error: %v", err)
	}

	result, err := service.RunOnce(context.Background(), time.Date(2026, 3, 23, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("run discovery once error: %v", err)
	}
	if result.RulesExecuted != 1 {
		t.Fatalf("expected rules executed=1, got %d", result.RulesExecuted)
	}
	if result.CandidatesCreated != 1 {
		t.Fatalf("expected created=1, got %d", result.CandidatesCreated)
	}
	if result.CandidatesUpdated != 0 {
		t.Fatalf("expected updated=0, got %d", result.CandidatesUpdated)
	}

	candidates := candidateService.List()
	if len(candidates) != 1 {
		t.Fatalf("expected candidates=1, got %d", len(candidates))
	}
	if candidates[0].Position != "Backend Engineer" {
		t.Fatalf("expected normalized position Backend Engineer, got %q", candidates[0].Position)
	}
	if candidates[0].Company != "Example" {
		t.Fatalf("expected normalized company Example, got %q", candidates[0].Company)
	}
}

func TestRunOnceUpsertsByJDURL(t *testing.T) {
	t.Parallel()

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Jobs</title>
    <item>
      <title>Backend Engineer at Example</title>
      <link>https://jobs.example.com/backend-1?utm_source=rss</link>
      <description>Go distributed systems cloud platform</description>
    </item>
  </channel>
</rss>`))
	}))
	defer feedServer.Close()

	ruleStore, err := storage.NewFileDiscoveryRuleStore(t.TempDir() + "/rules.json")
	if err != nil {
		t.Fatalf("new discovery rule store error: %v", err)
	}
	candidateStore, err := storage.NewFileCandidateStore(t.TempDir() + "/candidates.json")
	if err != nil {
		t.Fatalf("new candidate store error: %v", err)
	}
	candidateService := candidate.NewService(candidateStore, nil)
	service := NewService(ruleStore, candidateService)

	enabled := true
	_, err = service.CreateRule(model.DiscoveryRuleMutationInput{
		Name:            "Example Jobs",
		FeedURL:         feedServer.URL,
		Source:          "rss:example",
		DefaultLocation: "Remote",
		IncludeKeywords: []string{"go", "backend"},
		Enabled:         &enabled,
	})
	if err != nil {
		t.Fatalf("create rule error: %v", err)
	}

	_, err = service.RunOnce(context.Background(), time.Date(2026, 3, 23, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}
	result, err := service.RunOnce(context.Background(), time.Date(2026, 3, 23, 8, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}
	if result.CandidatesCreated != 0 {
		t.Fatalf("expected second run created=0, got %d", result.CandidatesCreated)
	}
	if result.CandidatesUpdated != 1 {
		t.Fatalf("expected second run updated=1, got %d", result.CandidatesUpdated)
	}
}
