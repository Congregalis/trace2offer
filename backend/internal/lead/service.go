package lead

import (
	"errors"
	"strings"
	"time"

	"trace2offer/backend/internal/model"
)

var ErrRepositoryUnavailable = errors.New("lead repository is unavailable")

const (
	StatusNew          = "new"
	StatusPreparing    = "preparing"
	StatusApplied      = "applied"
	StatusInterviewing = "interviewing"
	StatusOffered      = "offered"
	StatusDeclined     = "declined"
	StatusRejected     = "rejected"
	StatusArchived     = "archived"
)

const (
	ReminderMethodInApp   = "in_app"
	ReminderMethodEmail   = "email"
	ReminderMethodWebPush = "web_push"
)

const DefaultLeadStatus = StatusNew
const DefaultReminderMethod = ReminderMethodInApp

var canonicalLeadStatuses = []string{
	StatusNew,
	StatusPreparing,
	StatusApplied,
	StatusInterviewing,
	StatusOffered,
	StatusDeclined,
	StatusRejected,
	StatusArchived,
}

var canonicalReminderMethods = []string{
	ReminderMethodInApp,
	ReminderMethodEmail,
	ReminderMethodWebPush,
}

var canonicalLeadStatusSet = map[string]struct{}{
	StatusNew:          {},
	StatusPreparing:    {},
	StatusApplied:      {},
	StatusInterviewing: {},
	StatusOffered:      {},
	StatusDeclined:     {},
	StatusRejected:     {},
	StatusArchived:     {},
}

var canonicalReminderMethodSet = map[string]struct{}{
	ReminderMethodInApp:   {},
	ReminderMethodEmail:   {},
	ReminderMethodWebPush: {},
}

// Repository defines the persistence behavior needed by lead operations.
type Repository interface {
	List() []model.Lead
	Create(input model.LeadMutationInput) (model.Lead, error)
	Update(id string, input model.LeadMutationInput) (model.Lead, bool, error)
	Delete(id string) (bool, error)
}

// StatusObserver receives lead lifecycle updates, typically for timeline tracking.
type StatusObserver interface {
	OnLeadCreated(lead model.Lead)
	OnLeadUpdated(before model.Lead, after model.Lead)
	OnLeadDeleted(leadID string)
}

// Manager is the reusable lead CRUD contract shared by API and Agent tools.
type Manager interface {
	List() []model.Lead
	Create(input model.LeadMutationInput) (model.Lead, error)
	Update(id string, input model.LeadMutationInput) (model.Lead, bool, error)
	Delete(id string) (bool, error)
}

// ValidationError means user input is invalid.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "invalid lead payload"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Field) != "" {
		return e.Field + " is invalid"
	}
	return "invalid lead payload"
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// Service holds lead business rules and input normalization.
type Service struct {
	repo     Repository
	observer StatusObserver
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithStatusObserver(observer StatusObserver) *Service {
	if s == nil {
		return s
	}
	s.observer = observer
	return s
}

func (s *Service) List() []model.Lead {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.List()
}

func (s *Service) Create(input model.LeadMutationInput) (model.Lead, error) {
	if s == nil || s.repo == nil {
		return model.Lead{}, ErrRepositoryUnavailable
	}

	normalized, err := NormalizeMutationInput(input)
	if err != nil {
		return model.Lead{}, err
	}

	created, err := s.repo.Create(normalized)
	if err != nil {
		return model.Lead{}, err
	}
	if s.observer != nil {
		s.observer.OnLeadCreated(created)
	}
	return created, nil
}

func (s *Service) Update(id string, input model.LeadMutationInput) (model.Lead, bool, error) {
	if s == nil || s.repo == nil {
		return model.Lead{}, false, ErrRepositoryUnavailable
	}

	normalizedID, err := NormalizeLeadID(id)
	if err != nil {
		return model.Lead{}, false, err
	}

	normalizedInput, err := NormalizeMutationInput(input)
	if err != nil {
		return model.Lead{}, false, err
	}

	before, _ := s.findByID(normalizedID)
	updated, found, err := s.repo.Update(normalizedID, normalizedInput)
	if err != nil {
		return model.Lead{}, false, err
	}
	if found && s.observer != nil {
		s.observer.OnLeadUpdated(before, updated)
	}
	return updated, found, nil
}

func (s *Service) Delete(id string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrRepositoryUnavailable
	}

	normalizedID, err := NormalizeLeadID(id)
	if err != nil {
		return false, err
	}

	deleted, err := s.repo.Delete(normalizedID)
	if err != nil {
		return false, err
	}
	if deleted && s.observer != nil {
		s.observer.OnLeadDeleted(normalizedID)
	}
	return deleted, nil
}

func (s *Service) findByID(id string) (model.Lead, bool) {
	if s == nil || s.repo == nil {
		return model.Lead{}, false
	}

	for _, item := range s.repo.List() {
		if item.ID == id {
			return item, true
		}
	}
	return model.Lead{}, false
}

func NormalizeLeadID(id string) (string, error) {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return "", &ValidationError{Field: "id", Message: "lead id is required"}
	}
	return normalized, nil
}

func CanonicalLeadStatuses() []string {
	copied := make([]string, len(canonicalLeadStatuses))
	copy(copied, canonicalLeadStatuses)
	return copied
}

func CanonicalReminderMethods() []string {
	copied := make([]string, len(canonicalReminderMethods))
	copy(copied, canonicalReminderMethods)
	return copied
}

func NormalizeLeadStatus(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return DefaultLeadStatus, true
	}
	if _, ok := canonicalLeadStatusSet[value]; ok {
		return value, true
	}
	return "", false
}

func NormalizeMutationInput(input model.LeadMutationInput) (model.LeadMutationInput, error) {
	input.Company = strings.TrimSpace(input.Company)
	input.Position = strings.TrimSpace(input.Position)
	input.Source = strings.TrimSpace(input.Source)
	input.Status = strings.TrimSpace(input.Status)
	input.NextAction = strings.TrimSpace(input.NextAction)
	input.NextActionAt = strings.TrimSpace(input.NextActionAt)
	input.InterviewAt = strings.TrimSpace(input.InterviewAt)
	input.Notes = strings.TrimSpace(input.Notes)
	input.CompanyWebsiteURL = strings.TrimSpace(input.CompanyWebsiteURL)
	input.JDURL = strings.TrimSpace(input.JDURL)
	input.JDText = strings.TrimSpace(input.JDText)
	input.Location = strings.TrimSpace(input.Location)

	if input.Company == "" {
		return model.LeadMutationInput{}, &ValidationError{Field: "company", Message: "company is required"}
	}
	if input.Position == "" {
		return model.LeadMutationInput{}, &ValidationError{Field: "position", Message: "position is required"}
	}
	normalizedStatus, ok := NormalizeLeadStatus(input.Status)
	if !ok {
		return model.LeadMutationInput{}, &ValidationError{
			Field:   "status",
			Message: "status is invalid, allowed: " + strings.Join(canonicalLeadStatuses, ", "),
		}
	}
	input.Status = normalizedStatus
	if input.Priority < 0 {
		input.Priority = 0
	}
	if input.NextActionAt != "" {
		normalizedAt, ok := normalizeRFC3339Time(input.NextActionAt)
		if !ok {
			return model.LeadMutationInput{}, &ValidationError{
				Field:   "next_action_at",
				Message: "next_action_at is invalid, expected RFC3339 datetime",
			}
		}
		input.NextActionAt = normalizedAt
	}
	if input.InterviewAt != "" {
		normalizedAt, ok := normalizeRFC3339Time(input.InterviewAt)
		if !ok {
			return model.LeadMutationInput{}, &ValidationError{
				Field:   "interview_at",
				Message: "interview_at is invalid, expected RFC3339 datetime",
			}
		}
		input.InterviewAt = normalizedAt
	}

	methods, err := normalizeReminderMethods(input.ReminderMethods)
	if err != nil {
		return model.LeadMutationInput{}, err
	}
	input.ReminderMethods = methods

	return input, nil
}

func normalizeReminderMethods(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return []string{DefaultReminderMethod}, nil
	}

	normalized := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		method := strings.TrimSpace(strings.ToLower(item))
		if method == "" {
			continue
		}
		if _, ok := canonicalReminderMethodSet[method]; !ok {
			return nil, &ValidationError{
				Field:   "reminder_methods",
				Message: "reminder_methods is invalid, allowed: " + strings.Join(canonicalReminderMethods, ", "),
			}
		}
		if _, exists := seen[method]; exists {
			continue
		}
		seen[method] = struct{}{}
		normalized = append(normalized, method)
	}

	if len(normalized) == 0 {
		return []string{DefaultReminderMethod}, nil
	}
	return normalized, nil
}

func normalizeRFC3339Time(raw string) (string, bool) {
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
	return "", false
}
