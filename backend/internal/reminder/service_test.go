package reminder

import (
	"testing"
	"time"

	"trace2offer/backend/internal/model"
)

func TestServiceGetDueByNextActionAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	service := NewService(&stubLeadRepo{
		leads: []model.Lead{
			{
				ID:              "lead_due",
				Company:         "OpenAI",
				Position:        "Backend Engineer",
				Status:          "new",
				NextAction:      "跟进招聘经理",
				NextActionAt:    now.Add(-time.Hour).Format(time.RFC3339),
				ReminderMethods: []string{"web_push", "in_app"},
			},
			{
				ID:              "lead_future",
				Company:         "Figma",
				Position:        "Platform Engineer",
				Status:          "preparing",
				NextActionAt:    now.Add(time.Hour).Format(time.RFC3339),
				ReminderMethods: []string{"in_app"},
			},
		},
	})
	service.now = func() time.Time { return now }

	items := service.GetDue()
	if len(items) != 1 {
		t.Fatalf("expected 1 due reminder, got %d", len(items))
	}
	if items[0].LeadID != "lead_due" {
		t.Fatalf("expected lead_due, got %q", items[0].LeadID)
	}
	if items[0].Type != "next_action_due" {
		t.Fatalf("expected type next_action_due, got %q", items[0].Type)
	}
	if len(items[0].Methods) != 2 {
		t.Fatalf("expected 2 methods, got %+v", items[0].Methods)
	}
}

func TestServiceHighPriorityOverdueReminder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	service := NewService(&stubLeadRepo{
		leads: []model.Lead{
			{
				ID:              "lead_overdue",
				Company:         "Datadog",
				Position:        "Platform Engineer",
				Status:          "interviewing",
				Priority:        5,
				UpdatedAt:       now.Add(-96 * time.Hour).Format(time.RFC3339),
				ReminderMethods: []string{"in_app"},
			},
		},
	})
	service.now = func() time.Time { return now }

	items := service.GetDue()
	if len(items) != 1 {
		t.Fatalf("expected 1 overdue reminder, got %d", len(items))
	}
	if items[0].Type != "high_priority_overdue" {
		t.Fatalf("expected high_priority_overdue, got %q", items[0].Type)
	}
}

func TestServiceInterview24HourReminder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)
	service := NewService(&stubLeadRepo{
		leads: []model.Lead{
			{
				ID:              "lead_interview",
				Company:         "Stripe",
				Position:        "Backend Engineer",
				Status:          "interviewing",
				InterviewAt:     now.Add(23 * time.Hour).Format(time.RFC3339),
				ReminderMethods: []string{"web_push", "in_app"},
			},
		},
	})
	service.now = func() time.Time { return now }

	items := service.GetDue()
	if len(items) != 1 {
		t.Fatalf("expected 1 interview reminder, got %d", len(items))
	}
	if items[0].Type != "interview_24h" {
		t.Fatalf("expected interview_24h, got %q", items[0].Type)
	}
}

type stubLeadRepo struct {
	leads []model.Lead
}

func (s *stubLeadRepo) List() []model.Lead {
	copied := make([]model.Lead, len(s.leads))
	copy(copied, s.leads)
	return copied
}
