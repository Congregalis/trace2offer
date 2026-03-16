package tool

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"trace2offer/backend/internal/stats"
)

func TestStatsSummaryTool(t *testing.T) {
	t.Parallel()

	tool := &statsSummaryTool{
		provider: &stubStatsSummaryProvider{
			summary: stats.SummaryStats{
				Overview: stats.OverviewStats{Total: 5, Active: 4},
				Insights: stats.InsightStats{Urgent: 2},
				Generated: time.Date(
					2026, 3, 16, 12, 0, 0, 0, time.UTC,
				),
			},
		},
	}

	output, err := tool.Run(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("run summary tool failed: %v", err)
	}

	var payload struct {
		Summary struct {
			Overview struct {
				Total int `json:"total"`
			} `json:"overview"`
			Insights struct {
				Urgent int `json:"urgent"`
			} `json:"insights"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output failed: %v", err)
	}
	if payload.Summary.Overview.Total != 5 {
		t.Fatalf("expected total=5, got %d", payload.Summary.Overview.Total)
	}
	if payload.Summary.Insights.Urgent != 2 {
		t.Fatalf("expected urgent=2, got %d", payload.Summary.Insights.Urgent)
	}
}

func TestJobSearchStrategyTool(t *testing.T) {
	t.Parallel()

	tool := &jobStrategyTool{
		provider: &stubStatsSummaryProvider{
			summary: stats.SummaryStats{
				Overview: stats.OverviewStats{
					Total:       10,
					Active:      6,
					ThisWeekNew: 0,
				},
				Funnel: stats.FunnelStats{
					Conversion: 8.3,
				},
				Sources: stats.SourceAnalysis{
					BestSource: "Referral",
				},
				Duration: stats.DurationStats{
					AverageActiveDays: 9.2,
					SlowestStatus:     "interviewing",
				},
				Insights: stats.InsightStats{
					Urgent: 1,
				},
			},
		},
	}

	output, err := tool.Run(context.Background(), json.RawMessage(`{
		"focus_roles": ["后端工程师", "后端工程师", "平台工程师"],
		"focus_locations": ["上海"],
		"horizon_days": 21
	}`))
	if err != nil {
		t.Fatalf("run strategy tool failed: %v", err)
	}

	var payload struct {
		HorizonDays int `json:"horizon_days"`
		Priorities  []struct {
			Category string `json:"category"`
			Reason   string `json:"reason"`
		} `json:"priorities"`
		ProfileFocus struct {
			Roles []string `json:"roles"`
		} `json:"profile_focus"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output failed: %v", err)
	}
	if payload.HorizonDays != 21 {
		t.Fatalf("expected horizon_days=21, got %d", payload.HorizonDays)
	}
	if len(payload.Priorities) == 0 {
		t.Fatal("expected strategy priorities")
	}
	if payload.Priorities[0].Category != "follow_up" {
		t.Fatalf("expected first category follow_up, got %q", payload.Priorities[0].Category)
	}
	if len(payload.ProfileFocus.Roles) != 2 {
		t.Fatalf("expected deduplicated roles len=2, got %d", len(payload.ProfileFocus.Roles))
	}
}

type stubStatsSummaryProvider struct {
	summary stats.SummaryStats
}

func (s *stubStatsSummaryProvider) GetSummary() stats.SummaryStats {
	if s == nil {
		return stats.SummaryStats{}
	}
	return s.summary
}
