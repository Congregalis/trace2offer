package lead

import (
	"errors"
	"strconv"
	"testing"

	"trace2offer/backend/internal/model"
)

func TestNormalizeMutationInput(t *testing.T) {
	t.Parallel()

	input := model.LeadMutationInput{
		Company:           "  OpenAI ",
		Position:          " Backend Engineer  ",
		Source:            "  Referral ",
		Status:            " ",
		Priority:          -3,
		NextAction:        "  follow up ",
		NextActionAt:      "2026-03-20T09:30:00Z",
		InterviewAt:       "2026-03-21T02:00:00Z",
		ReminderMethods:   []string{"web_push", "in_app", "web_push"},
		Notes:             " note ",
		CompanyWebsiteURL: " https://openai.com ",
		JDURL:             " https://openai.com/careers/backend-engineer ",
		Location:          "  San Francisco  ",
	}

	normalized, err := NormalizeMutationInput(input)
	if err != nil {
		t.Fatalf("normalize input error: %v", err)
	}

	if normalized.Company != "OpenAI" {
		t.Fatalf("expected trimmed company, got %q", normalized.Company)
	}
	if normalized.Position != "Backend Engineer" {
		t.Fatalf("expected trimmed position, got %q", normalized.Position)
	}
	if normalized.Source != "Referral" {
		t.Fatalf("expected trimmed source, got %q", normalized.Source)
	}
	if normalized.Status != "new" {
		t.Fatalf("expected default status new, got %q", normalized.Status)
	}
	if normalized.Priority != 0 {
		t.Fatalf("expected non-negative priority, got %d", normalized.Priority)
	}
	if normalized.CompanyWebsiteURL != "https://openai.com" {
		t.Fatalf("expected trimmed company website url, got %q", normalized.CompanyWebsiteURL)
	}
	if normalized.JDURL != "https://openai.com/careers/backend-engineer" {
		t.Fatalf("expected trimmed jd url, got %q", normalized.JDURL)
	}
	if normalized.Location != "San Francisco" {
		t.Fatalf("expected trimmed location, got %q", normalized.Location)
	}
	if normalized.NextActionAt != "2026-03-20T09:30:00Z" {
		t.Fatalf("expected normalized next_action_at, got %q", normalized.NextActionAt)
	}
	if normalized.InterviewAt != "2026-03-21T02:00:00Z" {
		t.Fatalf("expected normalized interview_at, got %q", normalized.InterviewAt)
	}
	if len(normalized.ReminderMethods) != 2 {
		t.Fatalf("expected deduplicated reminder methods, got %+v", normalized.ReminderMethods)
	}
	if normalized.ReminderMethods[0] != "web_push" || normalized.ReminderMethods[1] != "in_app" {
		t.Fatalf("unexpected reminder methods order: %+v", normalized.ReminderMethods)
	}
}

func TestNormalizeMutationInputCanonicalStatus(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeMutationInput(model.LeadMutationInput{
		Company:  "OpenAI",
		Position: "Platform Engineer",
		Status:   StatusPreparing,
	})
	if err != nil {
		t.Fatalf("normalize input error: %v", err)
	}
	if normalized.Status != StatusPreparing {
		t.Fatalf("expected status %q, got %q", StatusPreparing, normalized.Status)
	}
}

func TestNormalizeMutationInputInvalidStatus(t *testing.T) {
	t.Parallel()

	_, err := NormalizeMutationInput(model.LeadMutationInput{
		Company:  "OpenAI",
		Position: "Backend Engineer",
		Status:   "some_invalid_status",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func TestNormalizeMutationInputInvalidReminderMethod(t *testing.T) {
	t.Parallel()

	_, err := NormalizeMutationInput(model.LeadMutationInput{
		Company:         "OpenAI",
		Position:        "Backend Engineer",
		ReminderMethods: []string{"sms"},
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func TestNormalizeMutationInputInvalidInterviewAt(t *testing.T) {
	t.Parallel()

	_, err := NormalizeMutationInput(model.LeadMutationInput{
		Company:     "OpenAI",
		Position:    "Backend Engineer",
		InterviewAt: "invalid",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func TestServiceCreateValidationError(t *testing.T) {
	t.Parallel()

	service := NewService(&stubRepository{})
	_, err := service.Create(model.LeadMutationInput{Company: "", Position: "Backend"})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T %v", err, err)
	}
}

func TestServiceUpdateDeleteFlow(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo)

	created, err := service.Create(model.LeadMutationInput{
		Company:  "OpenAI",
		Position: "Platform Engineer",
		Status:   "new",
	})
	if err != nil {
		t.Fatalf("create lead error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created lead id should not be empty")
	}

	updated, found, err := service.Update("  "+created.ID+" ", model.LeadMutationInput{
		Company:  "OpenAI",
		Position: "Platform Engineer",
		Status:   "interviewing",
	})
	if err != nil {
		t.Fatalf("update lead error: %v", err)
	}
	if !found {
		t.Fatal("expected lead found on update")
	}
	if updated.Status != "interviewing" {
		t.Fatalf("expected status interviewing, got %s", updated.Status)
	}

	deleted, err := service.Delete(created.ID)
	if err != nil {
		t.Fatalf("delete lead error: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete success")
	}

	deleted, err = service.Delete(created.ID)
	if err != nil {
		t.Fatalf("delete missing lead error: %v", err)
	}
	if deleted {
		t.Fatal("expected delete=false for missing lead")
	}
}

func TestServiceRepositoryUnavailable(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	_, err := service.Create(model.LeadMutationInput{Company: "A", Position: "B"})
	if !errors.Is(err, ErrRepositoryUnavailable) {
		t.Fatalf("expected ErrRepositoryUnavailable, got %v", err)
	}
}

func TestServiceStatusObserver(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	observer := &stubStatusObserver{}
	service := NewService(repo).WithStatusObserver(observer)

	created, err := service.Create(model.LeadMutationInput{
		Company:  "OpenAI",
		Position: "Backend Engineer",
		Status:   "new",
	})
	if err != nil {
		t.Fatalf("create lead error: %v", err)
	}

	if observer.createdCount != 1 {
		t.Fatalf("expected created callback once, got %d", observer.createdCount)
	}
	if observer.lastCreated.ID != created.ID {
		t.Fatalf("expected created callback lead id %q, got %q", created.ID, observer.lastCreated.ID)
	}

	updated, found, err := service.Update(created.ID, model.LeadMutationInput{
		Company:  "OpenAI",
		Position: "Backend Engineer",
		Status:   "interviewing",
	})
	if err != nil {
		t.Fatalf("update lead error: %v", err)
	}
	if !found {
		t.Fatal("expected lead found on update")
	}
	if observer.updatedCount != 1 {
		t.Fatalf("expected updated callback once, got %d", observer.updatedCount)
	}
	if observer.lastAfter.ID != updated.ID {
		t.Fatalf("expected updated callback after id %q, got %q", updated.ID, observer.lastAfter.ID)
	}

	deleted, err := service.Delete(created.ID)
	if err != nil {
		t.Fatalf("delete lead error: %v", err)
	}
	if !deleted {
		t.Fatal("expected delete success")
	}
	if observer.deletedCount != 1 {
		t.Fatalf("expected deleted callback once, got %d", observer.deletedCount)
	}
	if observer.lastDeletedID != created.ID {
		t.Fatalf("expected deleted callback id %q, got %q", created.ID, observer.lastDeletedID)
	}
}

type stubRepository struct {
	items []model.Lead
	next  int
}

type stubStatusObserver struct {
	createdCount  int
	updatedCount  int
	deletedCount  int
	lastCreated   model.Lead
	lastBefore    model.Lead
	lastAfter     model.Lead
	lastDeletedID string
}

func (s *stubStatusObserver) OnLeadCreated(lead model.Lead) {
	s.createdCount++
	s.lastCreated = lead
}

func (s *stubStatusObserver) OnLeadUpdated(before model.Lead, after model.Lead) {
	s.updatedCount++
	s.lastBefore = before
	s.lastAfter = after
}

func (s *stubStatusObserver) OnLeadDeleted(leadID string) {
	s.deletedCount++
	s.lastDeletedID = leadID
}

func (s *stubRepository) List() []model.Lead {
	copied := make([]model.Lead, len(s.items))
	copy(copied, s.items)
	return copied
}

func (s *stubRepository) Create(input model.LeadMutationInput) (model.Lead, error) {
	s.next++
	created := model.Lead{
		ID:       "lead_" + strconv.Itoa(s.next),
		Company:  input.Company,
		Position: input.Position,
		Status:   input.Status,
	}
	s.items = append(s.items, created)
	return created, nil
}

func (s *stubRepository) Update(id string, input model.LeadMutationInput) (model.Lead, bool, error) {
	for index := range s.items {
		if s.items[index].ID != id {
			continue
		}
		s.items[index].Company = input.Company
		s.items[index].Position = input.Position
		s.items[index].Status = input.Status
		return s.items[index], true, nil
	}
	return model.Lead{}, false, nil
}

func (s *stubRepository) Delete(id string) (bool, error) {
	for index := range s.items {
		if s.items[index].ID != id {
			continue
		}
		s.items = append(s.items[:index], s.items[index+1:]...)
		return true, nil
	}
	return false, nil
}
