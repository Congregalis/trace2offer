package stats

import (
	"testing"
	"time"

	leadpkg "trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
)

func TestCalculatorCoreDashboardMetrics(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	leads := []model.Lead{
		testLead(leadpkg.StatusNew, "2026-03-16T01:00:00Z", "2026-03-15T12:00:00Z"),
		testLead(leadpkg.StatusPreparing, "2026-03-01T09:00:00Z", "2026-03-10T09:00:00Z"),
		testLead(leadpkg.StatusApplied, "2026-02-20T09:00:00Z", "2026-03-14T09:00:00Z"),
		testLead(leadpkg.StatusInterviewing, "2026-02-01T09:00:00Z", "2026-03-01T09:00:00Z"),
		testLead(leadpkg.StatusOffered, "2026-01-15T09:00:00Z", "2026-02-15T09:00:00Z"),
		testLead(leadpkg.StatusRejected, "2026-01-20T09:00:00Z", "2026-02-10T09:00:00Z"),
		testLead(leadpkg.StatusDeclined, "2026-02-01T09:00:00Z", "2026-03-05T09:00:00Z"),
		testLead(leadpkg.StatusArchived, "2026-02-02T09:00:00Z", "2026-03-06T09:00:00Z"),
	}

	calculator := NewCalculator(leads)
	calculator.now = now

	overview := calculator.CalculateOverview()
	if overview.Total != 8 {
		t.Fatalf("expected total=8, got %d", overview.Total)
	}
	if overview.Active != 4 {
		t.Fatalf("expected active=4, got %d", overview.Active)
	}
	if overview.Offered != 1 {
		t.Fatalf("expected offered=1, got %d", overview.Offered)
	}
	if overview.ThisWeekNew != 1 {
		t.Fatalf("expected this_week_new=1, got %d", overview.ThisWeekNew)
	}
	if overview.SuccessRate != 33.3 {
		t.Fatalf("expected success_rate=33.3, got %.1f", overview.SuccessRate)
	}
	if len(overview.StatusCounts) != 8 {
		t.Fatalf("expected 8 status cards, got %d", len(overview.StatusCounts))
	}

	funnel := calculator.CalculateFunnel()
	if len(funnel.Stages) != 5 {
		t.Fatalf("expected 5 funnel stages, got %d", len(funnel.Stages))
	}
	if funnel.Conversion != 20 {
		t.Fatalf("expected conversion=20, got %.1f", funnel.Conversion)
	}
	if funnel.Stages[0].CumulativeCount != 5 {
		t.Fatalf("expected first cumulative=5, got %d", funnel.Stages[0].CumulativeCount)
	}
	if funnel.Stages[4].CumulativeCount != 1 {
		t.Fatalf("expected offered cumulative=1, got %d", funnel.Stages[4].CumulativeCount)
	}

	duration := calculator.CalculateDuration()
	if len(duration.ByStatus) != 5 {
		t.Fatalf("expected 5 duration statuses, got %d", len(duration.ByStatus))
	}
	if duration.SlowestStatus == "" {
		t.Fatal("expected slowest status to be populated")
	}
}

func TestCalculatorTrendPeriodShape(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	leads := []model.Lead{
		testLead(leadpkg.StatusNew, "2026-03-15T01:00:00Z", "2026-03-15T01:00:00Z"),
		testLead(leadpkg.StatusApplied, "2026-03-10T01:00:00Z", "2026-03-12T01:00:00Z"),
	}

	calculator := NewCalculator(leads)
	calculator.now = now

	weekly := calculator.CalculateTrends("week")
	if weekly.Period != "week" {
		t.Fatalf("expected weekly period=week, got %q", weekly.Period)
	}
	if len(weekly.Points) != 7 {
		t.Fatalf("expected 7 weekly points, got %d", len(weekly.Points))
	}

	monthly := calculator.CalculateTrends("month")
	if monthly.Period != "month" {
		t.Fatalf("expected monthly period=month, got %q", monthly.Period)
	}
	if len(monthly.Points) != 30 {
		t.Fatalf("expected 30 monthly points, got %d", len(monthly.Points))
	}

	fallback := calculator.CalculateTrends("unknown")
	if fallback.Period != "month" {
		t.Fatalf("expected fallback period=month, got %q", fallback.Period)
	}
}

func testLead(status string, createdAt string, updatedAt string) model.Lead {
	return model.Lead{
		ID:        "lead_" + status + "_" + createdAt,
		Company:   "TestCo",
		Position:  "Backend Engineer",
		Source:    "LinkedIn",
		Status:    status,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
