package lead

import (
	"errors"
	"strings"

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

const DefaultLeadStatus = StatusNew

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

// Repository defines the persistence behavior needed by lead operations.
type Repository interface {
	List() []model.Lead
	Create(input model.LeadMutationInput) (model.Lead, error)
	Update(id string, input model.LeadMutationInput) (model.Lead, bool, error)
	Delete(id string) (bool, error)
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
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	return s.repo.Create(normalized)
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

	return s.repo.Update(normalizedID, normalizedInput)
}

func (s *Service) Delete(id string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrRepositoryUnavailable
	}

	normalizedID, err := NormalizeLeadID(id)
	if err != nil {
		return false, err
	}

	return s.repo.Delete(normalizedID)
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
	input.Notes = strings.TrimSpace(input.Notes)
	input.CompanyWebsiteURL = strings.TrimSpace(input.CompanyWebsiteURL)
	input.JDURL = strings.TrimSpace(input.JDURL)
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

	return input, nil
}
