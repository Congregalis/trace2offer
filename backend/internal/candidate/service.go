package candidate

import (
	"errors"
	"fmt"
	"strings"

	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
)

var ErrRepositoryUnavailable = errors.New("candidate repository is unavailable")
var ErrLeadManagerUnavailable = errors.New("lead manager is unavailable")
var ErrCandidateNotFound = errors.New("candidate not found")

const (
	StatusPendingReview = "pending_review"
	StatusShortlisted   = "shortlisted"
	StatusDismissed     = "dismissed"
	StatusPromoted      = "promoted"
)

const DefaultCandidateStatus = StatusPendingReview

var canonicalCandidateStatuses = []string{
	StatusPendingReview,
	StatusShortlisted,
	StatusDismissed,
	StatusPromoted,
}

var canonicalCandidateStatusSet = map[string]struct{}{
	StatusPendingReview: {},
	StatusShortlisted:   {},
	StatusDismissed:     {},
	StatusPromoted:      {},
}

// Repository defines persistence behavior for candidate operations.
type Repository interface {
	List() []model.Candidate
	Create(input model.CandidateMutationInput) (model.Candidate, error)
	Update(id string, input model.CandidateMutationInput) (model.Candidate, bool, error)
	Delete(id string) (bool, error)
}

// Service holds candidate business rules and normalization.
type Service struct {
	repo        Repository
	leadManager lead.Manager
}

func NewService(repo Repository, leadManager lead.Manager) *Service {
	return &Service{
		repo:        repo,
		leadManager: leadManager,
	}
}

// ValidationError means user input is invalid.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "invalid candidate payload"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Field) != "" {
		return e.Field + " is invalid"
	}
	return "invalid candidate payload"
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

func (s *Service) List() []model.Candidate {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.List()
}

func (s *Service) Create(input model.CandidateMutationInput) (model.Candidate, error) {
	if s == nil || s.repo == nil {
		return model.Candidate{}, ErrRepositoryUnavailable
	}
	normalized, err := normalizeMutationInput(input, false)
	if err != nil {
		return model.Candidate{}, err
	}
	return s.repo.Create(normalized)
}

func (s *Service) Update(id string, input model.CandidateMutationInput) (model.Candidate, bool, error) {
	if s == nil || s.repo == nil {
		return model.Candidate{}, false, ErrRepositoryUnavailable
	}
	normalizedID, err := normalizeCandidateID(id)
	if err != nil {
		return model.Candidate{}, false, err
	}
	normalizedInput, err := normalizeMutationInput(input, false)
	if err != nil {
		return model.Candidate{}, false, err
	}
	return s.repo.Update(normalizedID, normalizedInput)
}

func (s *Service) Delete(id string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrRepositoryUnavailable
	}
	normalizedID, err := normalizeCandidateID(id)
	if err != nil {
		return false, err
	}
	return s.repo.Delete(normalizedID)
}

func (s *Service) Promote(id string, input model.CandidatePromoteInput) (model.Candidate, model.Lead, error) {
	if s == nil || s.repo == nil {
		return model.Candidate{}, model.Lead{}, ErrRepositoryUnavailable
	}
	if s.leadManager == nil {
		return model.Candidate{}, model.Lead{}, ErrLeadManagerUnavailable
	}

	normalizedID, err := normalizeCandidateID(id)
	if err != nil {
		return model.Candidate{}, model.Lead{}, err
	}

	current, found := s.findByID(normalizedID)
	if !found {
		return model.Candidate{}, model.Lead{}, ErrCandidateNotFound
	}
	if current.Status == StatusPromoted && strings.TrimSpace(current.PromotedLeadID) != "" {
		return model.Candidate{}, model.Lead{}, &ValidationError{
			Field:   "status",
			Message: "candidate already promoted",
		}
	}

	leadInput := model.LeadMutationInput{
		Company:           current.Company,
		Position:          current.Position,
		Source:            chooseNonEmpty(input.Source, current.Source),
		Status:            strings.TrimSpace(input.Status),
		Priority:          input.Priority,
		NextAction:        strings.TrimSpace(input.NextAction),
		NextActionAt:      strings.TrimSpace(input.NextActionAt),
		InterviewAt:       strings.TrimSpace(input.InterviewAt),
		ReminderMethods:   append([]string(nil), input.ReminderMethods...),
		Notes:             composeLeadNotes(current, strings.TrimSpace(input.Notes)),
		CompanyWebsiteURL: current.CompanyWebsiteURL,
		JDURL:             current.JDURL,
		Location:          current.Location,
	}
	if leadInput.Priority <= 0 {
		leadInput.Priority = 3
	}
	if leadInput.NextAction == "" {
		leadInput.NextAction = "review candidate and apply"
	}

	createdLead, err := s.leadManager.Create(leadInput)
	if err != nil {
		return model.Candidate{}, model.Lead{}, err
	}

	promotedInput := model.CandidateMutationInput{
		Company:             current.Company,
		Position:            current.Position,
		Source:              current.Source,
		Location:            current.Location,
		JDURL:               current.JDURL,
		CompanyWebsiteURL:   current.CompanyWebsiteURL,
		Status:              StatusPromoted,
		MatchScore:          current.MatchScore,
		MatchReasons:        append([]string(nil), current.MatchReasons...),
		RecommendationNotes: current.RecommendationNotes,
		Notes:               current.Notes,
		PromotedLeadID:      createdLead.ID,
	}
	promotedInput, err = normalizeMutationInput(promotedInput, true)
	if err != nil {
		return model.Candidate{}, model.Lead{}, err
	}
	updatedCandidate, found, err := s.repo.Update(normalizedID, promotedInput)
	if err != nil {
		return model.Candidate{}, model.Lead{}, err
	}
	if !found {
		return model.Candidate{}, model.Lead{}, ErrCandidateNotFound
	}

	return updatedCandidate, createdLead, nil
}

func (s *Service) findByID(id string) (model.Candidate, bool) {
	if s == nil || s.repo == nil {
		return model.Candidate{}, false
	}
	for _, item := range s.repo.List() {
		if item.ID == id {
			return item, true
		}
	}
	return model.Candidate{}, false
}

func normalizeCandidateID(id string) (string, error) {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return "", &ValidationError{Field: "id", Message: "candidate id is required"}
	}
	return normalized, nil
}

func normalizeCandidateStatus(raw string, allowPromoted bool) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return DefaultCandidateStatus, true
	}
	if value == StatusPromoted && !allowPromoted {
		return "", false
	}
	if _, ok := canonicalCandidateStatusSet[value]; !ok {
		return "", false
	}
	return value, true
}

func normalizeMutationInput(input model.CandidateMutationInput, allowPromoted bool) (model.CandidateMutationInput, error) {
	input.Company = strings.TrimSpace(input.Company)
	input.Position = strings.TrimSpace(input.Position)
	input.Source = strings.TrimSpace(input.Source)
	input.Location = strings.TrimSpace(input.Location)
	input.JDURL = strings.TrimSpace(input.JDURL)
	input.CompanyWebsiteURL = strings.TrimSpace(input.CompanyWebsiteURL)
	input.Status = strings.TrimSpace(input.Status)
	input.RecommendationNotes = strings.TrimSpace(input.RecommendationNotes)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.Company == "" {
		return model.CandidateMutationInput{}, &ValidationError{Field: "company", Message: "company is required"}
	}
	if input.Position == "" {
		return model.CandidateMutationInput{}, &ValidationError{Field: "position", Message: "position is required"}
	}
	normalizedStatus, ok := normalizeCandidateStatus(input.Status, allowPromoted)
	if !ok {
		return model.CandidateMutationInput{}, &ValidationError{
			Field:   "status",
			Message: "status is invalid, allowed: " + strings.Join(canonicalCandidateStatuses, ", "),
		}
	}
	input.Status = normalizedStatus
	if input.MatchScore < 0 {
		input.MatchScore = 0
	}
	if input.MatchScore > 100 {
		input.MatchScore = 100
	}
	input.MatchReasons = normalizeReasonList(input.MatchReasons)

	if !allowPromoted {
		input.PromotedLeadID = ""
	} else {
		input.PromotedLeadID = strings.TrimSpace(input.PromotedLeadID)
		if input.Status == StatusPromoted && input.PromotedLeadID == "" {
			return model.CandidateMutationInput{}, &ValidationError{
				Field:   "promoted_lead_id",
				Message: "promoted_lead_id is required when status is promoted",
			}
		}
	}

	return input, nil
}

func normalizeReasonList(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	result := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		reason := strings.TrimSpace(item)
		if reason == "" {
			continue
		}
		key := strings.ToLower(reason)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, reason)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func composeLeadNotes(candidate model.Candidate, extra string) string {
	parts := []string{}
	if candidate.RecommendationNotes != "" {
		parts = append(parts, fmt.Sprintf("[candidate_recommendation] %s", candidate.RecommendationNotes))
	}
	if len(candidate.MatchReasons) > 0 {
		parts = append(parts, "[candidate_match_reasons] "+strings.Join(candidate.MatchReasons, ", "))
	}
	if candidate.Notes != "" {
		parts = append(parts, "[candidate_notes] "+candidate.Notes)
	}
	if extra != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, "\n")
}

func chooseNonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}
