package heartbeat

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"trace2offer/backend/internal/model"
	"trace2offer/backend/internal/reminder"
	"trace2offer/backend/internal/stats"
)

func TestServiceRunOnceGeneratesReports(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	repo := &stubLeadRepo{
		leads: []model.Lead{
			{
				ID:              "lead_1",
				Company:         "OpenAI",
				Position:        "Backend Engineer",
				Status:          "new",
				Priority:        5,
				NextAction:      "发送跟进邮件",
				NextActionAt:    now.Add(-time.Hour).Format(time.RFC3339),
				ReminderMethods: []string{"in_app"},
				CreatedAt:       now.Add(-72 * time.Hour).Format(time.RFC3339),
				UpdatedAt:       now.Add(-72 * time.Hour).Format(time.RFC3339),
			},
		},
	}
	reminderService := reminder.NewService(repo)
	statsService := stats.NewService(repo)

	service, err := NewService(Config{
		DataDir:         t.TempDir(),
		Interval:        30 * time.Minute,
		ReminderService: reminderService,
		StatsService:    statsService,
	})
	if err != nil {
		t.Fatalf("new heartbeat service failed: %v", err)
	}

	if err := service.RunOnce(now); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	status := service.GetStatus()
	if status.LastRunAt == "" || status.NextRunAt == "" {
		t.Fatalf("expected status timestamps, got %+v", status)
	}

	reports, err := service.GetLatestReports()
	if err != nil {
		t.Fatalf("get latest reports failed: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}

	if _, err := filepath.Glob(filepath.Join(service.reportDir, "daily_*.md")); err != nil {
		t.Fatalf("glob daily report failed: %v", err)
	}
}

func TestServiceStartRunsWithTicker(t *testing.T) {
	t.Parallel()

	repo := &stubLeadRepo{}
	reminderService := reminder.NewService(repo)
	statsService := stats.NewService(repo)

	service, err := NewService(Config{
		DataDir:         t.TempDir(),
		Interval:        10 * time.Millisecond,
		ReminderService: reminderService,
		StatsService:    statsService,
	})
	if err != nil {
		t.Fatalf("new heartbeat service failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
	defer cancel()
	service.Start(ctx)

	status := service.GetStatus()
	if status.LastRunAt == "" {
		t.Fatal("expected heartbeat start to trigger at least one run")
	}
}

func TestServiceRunOnceWithDiscovery(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC)
	repo := &stubLeadRepo{}
	reminderService := reminder.NewService(repo)
	statsService := stats.NewService(repo)
	discoveryRunner := &stubDiscoveryRunner{
		result: model.DiscoveryRunResult{
			RanAt:             now.Format(time.RFC3339),
			CandidatesCreated: 2,
			CandidatesUpdated: 1,
			Errors:            []string{"one failed"},
		},
	}

	service, err := NewService(Config{
		DataDir:          t.TempDir(),
		Interval:         30 * time.Minute,
		ReminderService:  reminderService,
		StatsService:     statsService,
		DiscoveryService: discoveryRunner,
	})
	if err != nil {
		t.Fatalf("new heartbeat service failed: %v", err)
	}

	if err := service.RunOnce(now); err != nil {
		t.Fatalf("run once failed: %v", err)
	}

	if discoveryRunner.callCount != 1 {
		t.Fatalf("expected discovery runner called once, got %d", discoveryRunner.callCount)
	}
	status := service.GetStatus()
	if status.LastDiscoveryCreated != 2 || status.LastDiscoveryUpdated != 1 {
		t.Fatalf("unexpected discovery status: %+v", status)
	}
	if status.LastDiscoveryErrors != 1 {
		t.Fatalf("expected discovery errors=1, got %d", status.LastDiscoveryErrors)
	}
}

type stubLeadRepo struct {
	leads []model.Lead
}

type stubDiscoveryRunner struct {
	result    model.DiscoveryRunResult
	err       error
	callCount int
}

func (s *stubDiscoveryRunner) RunOnce(_ context.Context, _ time.Time) (model.DiscoveryRunResult, error) {
	s.callCount++
	return s.result, s.err
}

func (s *stubLeadRepo) List() []model.Lead {
	copied := make([]model.Lead, len(s.leads))
	copy(copied, s.leads)
	return copied
}

func (s *stubLeadRepo) Create(input model.LeadMutationInput) (model.Lead, error) {
	lead := model.Lead{
		ID:         "lead_created",
		Company:    input.Company,
		Position:   input.Position,
		Source:     input.Source,
		Status:     input.Status,
		Priority:   input.Priority,
		NextAction: input.NextAction,
	}
	s.leads = append(s.leads, lead)
	return lead, nil
}

func (s *stubLeadRepo) Update(id string, input model.LeadMutationInput) (model.Lead, bool, error) {
	for i := range s.leads {
		if s.leads[i].ID != id {
			continue
		}
		s.leads[i].Company = input.Company
		s.leads[i].Position = input.Position
		s.leads[i].Source = input.Source
		s.leads[i].Status = input.Status
		s.leads[i].Priority = input.Priority
		s.leads[i].NextAction = input.NextAction
		return s.leads[i], true, nil
	}
	return model.Lead{}, false, nil
}

func (s *stubLeadRepo) Delete(id string) (bool, error) {
	for i := range s.leads {
		if s.leads[i].ID != id {
			continue
		}
		s.leads = append(s.leads[:i], s.leads[i+1:]...)
		return true, nil
	}
	return false, nil
}
