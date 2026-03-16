package timeline

import (
	"strings"
	"time"

	"trace2offer/backend/internal/model"
)

var terminalStages = map[string]struct{}{
	"declined": {},
	"rejected": {},
	"archived": {},
}

// Repository defines persistence behavior needed by timeline tracking.
type Repository interface {
	List() []model.LeadTimeline
	Get(leadID string) (model.LeadTimeline, bool)
	Save(timeline model.LeadTimeline) (model.LeadTimeline, error)
	Delete(leadID string) (bool, error)
}

// Service tracks lead status transitions and materializes stage intervals.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() []model.LeadTimeline {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.List()
}

// OnLeadCreated records the initial stage start.
func (s *Service) OnLeadCreated(lead model.Lead) {
	if s == nil || s.repo == nil {
		return
	}

	leadID := strings.TrimSpace(lead.ID)
	status := strings.TrimSpace(lead.Status)
	if leadID == "" || status == "" {
		return
	}

	eventAt := firstValidTimestamp(lead.CreatedAt, lead.UpdatedAt)
	s.recordTransition(leadID, "", status, eventAt, eventAt)
}

// OnLeadUpdated records stage end/start when status changes.
func (s *Service) OnLeadUpdated(before model.Lead, after model.Lead) {
	if s == nil || s.repo == nil {
		return
	}

	leadID := strings.TrimSpace(after.ID)
	currentStatus := strings.TrimSpace(after.Status)
	if leadID == "" || currentStatus == "" {
		return
	}

	previousStatus := strings.TrimSpace(before.Status)
	eventAt := firstValidTimestamp(after.UpdatedAt, after.CreatedAt, before.UpdatedAt)
	fromStartedAt := firstValidTimestamp(before.CreatedAt, before.UpdatedAt, eventAt)

	if previousStatus == currentStatus {
		s.ensureOpenStage(leadID, currentStatus, fromStartedAt, eventAt)
		return
	}

	s.recordTransition(leadID, previousStatus, currentStatus, fromStartedAt, eventAt)
}

// OnLeadDeleted removes timeline data for the lead.
func (s *Service) OnLeadDeleted(leadID string) {
	if s == nil || s.repo == nil {
		return
	}
	_, _ = s.repo.Delete(strings.TrimSpace(leadID))
}

func (s *Service) recordTransition(leadID string, fromStage string, toStage string, fromStartedAt string, eventAt string) {
	normalizedLeadID := strings.TrimSpace(leadID)
	normalizedTo := strings.TrimSpace(toStage)
	if normalizedLeadID == "" || normalizedTo == "" {
		return
	}

	normalizedFrom := strings.TrimSpace(fromStage)
	normalizedEventAt := normalizeTimestamp(eventAt)
	normalizedFromStartedAt := normalizeTimestamp(fromStartedAt)
	toTerminal := isTerminalStage(normalizedTo)

	timelineItem, ok := s.repo.Get(normalizedLeadID)
	if !ok {
		timelineItem = model.LeadTimeline{LeadID: normalizedLeadID}
	}
	stages := append([]model.LeadTimelineStage(nil), timelineItem.Stages...)

	if normalizedFrom != "" && normalizedFrom != normalizedTo {
		fromIndex := findStageIndex(stages, normalizedFrom)
		if fromIndex >= 0 {
			if strings.TrimSpace(stages[fromIndex].StartedAt) == "" {
				stages[fromIndex].StartedAt = normalizedFromStartedAt
			}
			stages[fromIndex].EndedAt = normalizedEventAt
		} else {
			stages = append(stages, model.LeadTimelineStage{
				Stage:     normalizedFrom,
				StartedAt: normalizedFromStartedAt,
				EndedAt:   normalizedEventAt,
			})
		}
	}

	for index := range stages {
		if strings.TrimSpace(stages[index].EndedAt) != "" {
			continue
		}
		if strings.TrimSpace(stages[index].Stage) == normalizedTo && !toTerminal {
			continue
		}
		stages[index].EndedAt = normalizedEventAt
	}

	toIndex := findStageIndex(stages, normalizedTo)
	if toIndex >= 0 {
		if strings.TrimSpace(stages[toIndex].StartedAt) == "" ||
			compareTimestamps(normalizedEventAt, stages[toIndex].StartedAt) < 0 {
			stages[toIndex].StartedAt = normalizedEventAt
		}
		if toTerminal {
			stages[toIndex].EndedAt = maxTimestamp(stages[toIndex].EndedAt, normalizedEventAt)
		} else {
			stages[toIndex].EndedAt = ""
		}
	} else {
		endedAt := ""
		if toTerminal {
			endedAt = normalizedEventAt
		}
		stages = append(stages, model.LeadTimelineStage{
			Stage:     normalizedTo,
			StartedAt: normalizedEventAt,
			EndedAt:   endedAt,
		})
	}

	timelineItem.Stages = stages
	timelineItem.UpdatedAt = normalizedEventAt
	_, _ = s.repo.Save(timelineItem)
}

func (s *Service) ensureOpenStage(leadID string, stage string, startedAt string, eventAt string) {
	normalizedLeadID := strings.TrimSpace(leadID)
	normalizedStage := strings.TrimSpace(stage)
	if normalizedLeadID == "" || normalizedStage == "" {
		return
	}

	timelineItem, ok := s.repo.Get(normalizedLeadID)
	if !ok {
		timelineItem = model.LeadTimeline{LeadID: normalizedLeadID}
	}
	stages := append([]model.LeadTimelineStage(nil), timelineItem.Stages...)
	normalizedStartedAt := normalizeTimestamp(startedAt)
	normalizedEventAt := normalizeTimestamp(eventAt)
	stageTerminal := isTerminalStage(normalizedStage)
	for index := range stages {
		if strings.TrimSpace(stages[index].EndedAt) != "" {
			continue
		}
		if strings.TrimSpace(stages[index].Stage) == normalizedStage && !stageTerminal {
			continue
		}
		stages[index].EndedAt = normalizedEventAt
	}

	stageIndex := findStageIndex(stages, normalizedStage)
	if stageIndex >= 0 {
		if strings.TrimSpace(stages[stageIndex].StartedAt) == "" ||
			compareTimestamps(normalizedStartedAt, stages[stageIndex].StartedAt) < 0 {
			stages[stageIndex].StartedAt = normalizedStartedAt
		}
		if stageTerminal {
			if strings.TrimSpace(stages[stageIndex].EndedAt) == "" {
				stages[stageIndex].EndedAt = maxTimestamp(normalizedStartedAt, normalizedEventAt)
			}
		} else {
			stages[stageIndex].EndedAt = ""
		}
	} else {
		endedAt := ""
		if stageTerminal {
			endedAt = maxTimestamp(normalizedStartedAt, normalizedEventAt)
		}
		stages = append(stages, model.LeadTimelineStage{
			Stage:     normalizedStage,
			StartedAt: normalizedStartedAt,
			EndedAt:   endedAt,
		})
	}

	timelineItem.Stages = stages
	timelineItem.UpdatedAt = normalizedEventAt
	_, _ = s.repo.Save(timelineItem)
}

func findStageIndex(stages []model.LeadTimelineStage, stage string) int {
	target := strings.TrimSpace(stage)
	if target == "" {
		return -1
	}

	for index := range stages {
		if strings.TrimSpace(stages[index].Stage) != target {
			continue
		}
		return index
	}
	return -1
}

func firstValidTimestamp(raw ...string) string {
	for _, candidate := range raw {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		parsed, ok := parseDateTime(candidate)
		if ok {
			return parsed
		}
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func normalizeTimestamp(raw string) string {
	if parsed, ok := parseDateTime(raw); ok {
		return parsed
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func parseDateTime(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC().Format(time.RFC3339), true
	}
	if parsed, err := time.ParseInLocation("2006-01-02T15:04", value, time.Local); err == nil {
		return parsed.UTC().Format(time.RFC3339), true
	}
	if parsed, err := time.ParseInLocation("2006-01-02 15:04", value, time.Local); err == nil {
		return parsed.UTC().Format(time.RFC3339), true
	}
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return parsed.UTC().Format(time.RFC3339), true
	}
	return "", false
}

func compareTimestamps(left string, right string) int {
	leftParsed, leftOK := parseDateTime(left)
	rightParsed, rightOK := parseDateTime(right)
	if leftOK && rightOK {
		if leftParsed == rightParsed {
			return 0
		}
		if leftParsed < rightParsed {
			return -1
		}
		return 1
	}

	normalizedLeft := strings.TrimSpace(left)
	normalizedRight := strings.TrimSpace(right)
	if normalizedLeft == normalizedRight {
		return 0
	}
	if normalizedLeft < normalizedRight {
		return -1
	}
	return 1
}

func maxTimestamp(left string, right string) string {
	if compareTimestamps(left, right) >= 0 {
		return normalizeTimestamp(left)
	}
	return normalizeTimestamp(right)
}

func isTerminalStage(stage string) bool {
	normalized := strings.TrimSpace(stage)
	if normalized == "" {
		return false
	}
	_, ok := terminalStages[normalized]
	return ok
}
