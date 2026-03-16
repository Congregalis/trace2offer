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
	items := make([]Item, 0, len(leads))
	for _, lead := range leads {
		if strings.TrimSpace(lead.NextActionAt) == "" {
			continue
		}
		if !isActiveLeadStatus(lead.Status) {
			continue
		}

		dueAt, ok := parseRFC3339(lead.NextActionAt)
		if !ok {
			continue
		}
		if dueAt.After(now) {
			continue
		}

		methods := normalizeReminderMethods(lead.ReminderMethods)
		items = append(items, Item{
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
		})
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
