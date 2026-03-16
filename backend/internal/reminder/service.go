package reminder

import (
	"fmt"
	"sort"
	"strings"
	"time"

	leadpkg "trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
)

// LeadRepository is the read-only dependency reminder service needs.
type LeadRepository interface {
	List() []model.Lead
}

// Item is one due reminder payload for frontend notifications.
type Item struct {
	ID         string   `json:"id"`
	LeadID     string   `json:"lead_id"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Message    string   `json:"message"`
	DueAt      string   `json:"due_at"`
	Severity   string   `json:"severity"`
	Methods    []string `json:"methods"`
	Company    string   `json:"company"`
	Position   string   `json:"position"`
	NextAction string   `json:"next_action"`
}

// Service calculates reminder items from lead data.
type Service struct {
	repo              LeadRepository
	now               func() time.Time
	overdueDays       int
	priorityThreshold int
}

func NewService(repo LeadRepository) *Service {
	return &Service{
		repo: repo,
		now: func() time.Time {
			return time.Now().UTC()
		},
		overdueDays:       3,
		priorityThreshold: 4,
	}
}

func (s *Service) GetDue() []Item {
	if s == nil {
		return nil
	}
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	return s.GetDueAt(now)
}

func (s *Service) GetDueAt(now time.Time) []Item {
	if s == nil || s.repo == nil {
		return nil
	}

	leads := s.repo.List()
	items := make([]Item, 0, len(leads)*2)
	for _, lead := range leads {
		if !isActiveLeadStatus(lead.Status) {
			continue
		}
		methods := normalizeReminderMethods(lead.ReminderMethods)
		items = append(items, s.buildNextActionReminder(lead, now, methods)...)
		items = append(items, s.buildHighPriorityOverdueReminder(lead, now, methods)...)
		items = append(items, s.buildInterviewReminder(lead, now, methods)...)
	}

	sort.Slice(items, func(i, j int) bool {
		left, _ := parseRFC3339(items[i].DueAt)
		right, _ := parseRFC3339(items[j].DueAt)
		if left.Equal(right) {
			return items[i].LeadID < items[j].LeadID
		}
		return left.Before(right)
	})

	return items
}

func (s *Service) buildNextActionReminder(lead model.Lead, now time.Time, methods []string) []Item {
	if strings.TrimSpace(lead.NextActionAt) == "" {
		return nil
	}
	dueAt, ok := parseRFC3339(lead.NextActionAt)
	if !ok || dueAt.After(now) {
		return nil
	}

	return []Item{{
		ID:         buildReminderID("next_action", lead.ID, dueAt),
		LeadID:     strings.TrimSpace(lead.ID),
		Type:       "next_action_due",
		Title:      fmt.Sprintf("%s - %s 到期提醒", strings.TrimSpace(lead.Company), strings.TrimSpace(lead.Position)),
		Message:    buildNextActionMessage(lead),
		DueAt:      dueAt.Format(time.RFC3339),
		Severity:   "warning",
		Methods:    methods,
		Company:    strings.TrimSpace(lead.Company),
		Position:   strings.TrimSpace(lead.Position),
		NextAction: strings.TrimSpace(lead.NextAction),
	}}
}

func (s *Service) buildHighPriorityOverdueReminder(lead model.Lead, now time.Time, methods []string) []Item {
	if lead.Priority < s.priorityThreshold || s.overdueDays <= 0 {
		return nil
	}

	updatedAt, ok := parseRFC3339(lead.UpdatedAt)
	if !ok {
		updatedAt, ok = parseRFC3339(lead.CreatedAt)
	}
	if !ok {
		return nil
	}

	threshold := updatedAt.Add(time.Duration(s.overdueDays) * 24 * time.Hour)
	if threshold.After(now) {
		return nil
	}

	return []Item{{
		ID:         buildReminderID("high_priority_overdue", lead.ID, threshold),
		LeadID:     strings.TrimSpace(lead.ID),
		Type:       "high_priority_overdue",
		Title:      fmt.Sprintf("%s - %s 逾期预警", strings.TrimSpace(lead.Company), strings.TrimSpace(lead.Position)),
		Message:    fmt.Sprintf("高优先级线索已超过 %d 天未更新，建议立即跟进。", s.overdueDays),
		DueAt:      threshold.Format(time.RFC3339),
		Severity:   "critical",
		Methods:    methods,
		Company:    strings.TrimSpace(lead.Company),
		Position:   strings.TrimSpace(lead.Position),
		NextAction: strings.TrimSpace(lead.NextAction),
	}}
}

func (s *Service) buildInterviewReminder(lead model.Lead, now time.Time, methods []string) []Item {
	if strings.TrimSpace(lead.InterviewAt) == "" {
		return nil
	}
	interviewAt, ok := parseRFC3339(lead.InterviewAt)
	if !ok {
		return nil
	}

	alertAt := interviewAt.Add(-24 * time.Hour)
	if alertAt.After(now) {
		return nil
	}
	if now.After(interviewAt.Add(2 * time.Hour)) {
		return nil
	}

	return []Item{{
		ID:         buildReminderID("interview_24h", lead.ID, alertAt),
		LeadID:     strings.TrimSpace(lead.ID),
		Type:       "interview_24h",
		Title:      fmt.Sprintf("%s - %s 面试前 24 小时提醒", strings.TrimSpace(lead.Company), strings.TrimSpace(lead.Position)),
		Message:    fmt.Sprintf("面试时间：%s，请提前确认材料与时间安排。", interviewAt.Format("2006-01-02 15:04 MST")),
		DueAt:      alertAt.Format(time.RFC3339),
		Severity:   "warning",
		Methods:    methods,
		Company:    strings.TrimSpace(lead.Company),
		Position:   strings.TrimSpace(lead.Position),
		NextAction: strings.TrimSpace(lead.NextAction),
	}}
}

func buildNextActionMessage(lead model.Lead) string {
	nextAction := strings.TrimSpace(lead.NextAction)
	if nextAction == "" {
		return "已到达下一步动作时间，请及时跟进。"
	}
	return "下一步动作：" + nextAction
}

func buildReminderID(prefix string, leadID string, dueAt time.Time) string {
	return fmt.Sprintf("%s:%s:%s", strings.TrimSpace(prefix), strings.TrimSpace(leadID), dueAt.UTC().Format(time.RFC3339))
}

func isActiveLeadStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case leadpkg.StatusOffered, leadpkg.StatusDeclined, leadpkg.StatusRejected, leadpkg.StatusArchived:
		return false
	default:
		return true
	}
}

func normalizeReminderMethods(raw []string) []string {
	if len(raw) == 0 {
		return []string{leadpkg.ReminderMethodInApp}
	}

	methods := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		method := strings.TrimSpace(strings.ToLower(item))
		if method == "" {
			continue
		}
		switch method {
		case leadpkg.ReminderMethodInApp, leadpkg.ReminderMethodEmail, leadpkg.ReminderMethodWebPush:
		default:
			continue
		}
		if _, ok := seen[method]; ok {
			continue
		}
		seen[method] = struct{}{}
		methods = append(methods, method)
	}
	if len(methods) == 0 {
		return []string{leadpkg.ReminderMethodInApp}
	}
	return methods
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
