package calendar

import (
	"strings"
	"testing"
	"time"

	"trace2offer/backend/internal/model"
)

func TestBuildInterviewICS(t *testing.T) {
	t.Parallel()

	repo := &stubLeadRepo{
		leads: []model.Lead{
			{
				ID:          "lead_1",
				Company:     "OpenAI",
				Position:    "Backend Engineer",
				InterviewAt: "2026-03-18T06:00:00Z",
				NextAction:  "准备系统设计",
				Notes:       "准备项目案例",
			},
			{
				ID:       "lead_2",
				Company:  "NoInterview",
				Position: "N/A",
			},
		},
	}

	service := NewService(repo)
	service.now = func() time.Time { return time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC) }

	ics, err := service.BuildInterviewICS()
	if err != nil {
		t.Fatalf("build ics failed: %v", err)
	}
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Fatal("expected VCALENDAR header")
	}
	if !strings.Contains(ics, "SUMMARY:Interview - OpenAI / Backend Engineer") {
		t.Fatal("expected interview summary in ICS")
	}
	if strings.Contains(ics, "NoInterview") {
		t.Fatal("lead without interview_at should not be exported")
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
