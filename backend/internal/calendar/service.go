package calendar

import (
	"fmt"
	"strings"
	"time"

	"trace2offer/backend/internal/model"
)

type LeadRepository interface {
	List() []model.Lead
}

// Service generates ICS content from interview schedules.
type Service struct {
	repo LeadRepository
	now  func() time.Time
}

func NewService(repo LeadRepository) *Service {
	return &Service{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) BuildInterviewICS() (string, error) {
	if s == nil || s.repo == nil {
		return "", fmt.Errorf("calendar service is unavailable")
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}

	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Trace2Offer//Interview Calendar//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"X-WR-CALNAME:Trace2Offer Interviews",
	}

	for _, lead := range s.repo.List() {
		interviewAt, ok := parseRFC3339(strings.TrimSpace(lead.InterviewAt))
		if !ok {
			continue
		}
		endAt := interviewAt.Add(time.Hour)

		descriptionParts := make([]string, 0, 3)
		if strings.TrimSpace(lead.NextAction) != "" {
			descriptionParts = append(descriptionParts, "Next Action: "+strings.TrimSpace(lead.NextAction))
		}
		if strings.TrimSpace(lead.Notes) != "" {
			descriptionParts = append(descriptionParts, "Notes: "+strings.TrimSpace(lead.Notes))
		}
		if strings.TrimSpace(lead.JDURL) != "" {
			descriptionParts = append(descriptionParts, "JD: "+strings.TrimSpace(lead.JDURL))
		}
		description := escapeICSText(strings.Join(descriptionParts, "\n"))

		lines = append(lines,
			"BEGIN:VEVENT",
			"UID:"+escapeICSText(strings.TrimSpace(lead.ID))+"@trace2offer",
			"DTSTAMP:"+formatICSDateTime(now),
			"DTSTART:"+formatICSDateTime(interviewAt),
			"DTEND:"+formatICSDateTime(endAt),
			"SUMMARY:"+escapeICSText(fmt.Sprintf("Interview - %s / %s", strings.TrimSpace(lead.Company), strings.TrimSpace(lead.Position))),
			"DESCRIPTION:"+description,
			"END:VEVENT",
		)
	}

	lines = append(lines, "END:VCALENDAR")
	return strings.Join(lines, "\r\n") + "\r\n", nil
}

func parseRFC3339(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func formatICSDateTime(ts time.Time) string {
	return ts.UTC().Format("20060102T150405Z")
}

func escapeICSText(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, ";", "\\;")
	value = strings.ReplaceAll(value, ",", "\\,")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}
