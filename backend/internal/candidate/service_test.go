package candidate

import (
	"errors"
	"testing"

	"trace2offer/backend/internal/model"
)

func TestCreateNormalizesCandidateInput(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo, &stubLeadManager{})

	created, err := service.Create(model.CandidateMutationInput{
		Company:             "  Anthropic ",
		Position:            " Software Engineer ",
		Source:              " LinkedIn ",
		Status:              "",
		MatchScore:          130,
		MatchReasons:        []string{"Go backend", "Go backend", " LLM infra "},
		RecommendationNotes: "  fit ",
	})
	if err != nil {
		t.Fatalf("create candidate error: %v", err)
	}
	if created.Company != "Anthropic" {
		t.Fatalf("expected trimmed company, got %q", created.Company)
	}
	if created.Position != "Software Engineer" {
		t.Fatalf("expected trimmed position, got %q", created.Position)
	}
	if created.Status != StatusPendingReview {
		t.Fatalf("expected default status %q, got %q", StatusPendingReview, created.Status)
	}
	if created.MatchScore != 100 {
		t.Fatalf("expected score clamped to 100, got %d", created.MatchScore)
	}
	if len(created.MatchReasons) != 2 {
		t.Fatalf("expected deduplicated reasons, got %+v", created.MatchReasons)
	}
}

func TestPromoteConvertsCandidateToLead(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	leadManager := &stubLeadManager{}
	service := NewService(repo, leadManager)

	created, err := service.Create(model.CandidateMutationInput{
		Company:             "Anthropic",
		Position:            "Software Engineer",
		Source:              "LinkedIn",
		Location:            "Remote",
		JDURL:               "https://www.anthropic.com/careers/software-engineer",
		CompanyWebsiteURL:   "https://www.anthropic.com",
		MatchScore:          90,
		MatchReasons:        []string{"Go backend", "LLM infra"},
		RecommendationNotes: "很好",
		Notes:               "待确认",
	})
	if err != nil {
		t.Fatalf("create candidate error: %v", err)
	}

	updated, promoted, err := service.Promote(created.ID, model.CandidatePromoteInput{
		Status:      "new",
		Priority:    5,
		NextAction:  "投递并跟进",
		Source:      "Auto Discovery",
		Notes:       "手动确认后转入 lead",
		InterviewAt: "",
	})
	if err != nil {
		t.Fatalf("promote candidate error: %v", err)
	}

	if promoted.ID == "" {
		t.Fatal("expected promoted lead id not empty")
	}
	if promoted.Company != created.Company {
		t.Fatalf("expected lead company=%q, got %q", created.Company, promoted.Company)
	}
	if promoted.Source != "Auto Discovery" {
		t.Fatalf("expected overridden lead source, got %q", promoted.Source)
	}
	if updated.Status != StatusPromoted {
		t.Fatalf("expected candidate status promoted, got %q", updated.Status)
	}
	if updated.PromotedLeadID != promoted.ID {
		t.Fatalf("expected candidate promoted_lead_id=%q, got %q", promoted.ID, updated.PromotedLeadID)
	}

	if leadManager.createdCount != 1 {
		t.Fatalf("expected one lead created, got %d", leadManager.createdCount)
	}
}

func TestPromoteCandidateNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&stubRepository{}, &stubLeadManager{})
	_, _, err := service.Promote("missing", model.CandidatePromoteInput{})
	if !errors.Is(err, ErrCandidateNotFound) {
		t.Fatalf("expected ErrCandidateNotFound, got %v", err)
	}
}

func TestUpsertByJDURLCreatesThenUpdates(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{}
	service := NewService(repo, &stubLeadManager{})

	created, createdFlag, err := service.UpsertByJDURL(model.CandidateMutationInput{
		Company:             "Anthropic",
		Position:            "Software Engineer",
		Source:              "RSS",
		JDURL:               "https://jobs.example.com/positions/123?utm_source=feed",
		Status:              "",
		MatchScore:          80,
		MatchReasons:        []string{"Go"},
		RecommendationNotes: "first",
	})
	if err != nil {
		t.Fatalf("upsert create failed: %v", err)
	}
	if !createdFlag {
		t.Fatal("expected first upsert to create candidate")
	}
	if created.ID == "" {
		t.Fatal("expected created candidate id not empty")
	}

	updated, createdFlag, err := service.UpsertByJDURL(model.CandidateMutationInput{
		Company:             "Anthropic",
		Position:            "Senior Software Engineer",
		Source:              "RSS",
		JDURL:               "https://jobs.example.com/positions/123?utm_source=other",
		Status:              "",
		MatchScore:          92,
		MatchReasons:        []string{"Go", "Distributed Systems"},
		RecommendationNotes: "second",
	})
	if err != nil {
		t.Fatalf("upsert update failed: %v", err)
	}
	if createdFlag {
		t.Fatal("expected second upsert to update existing candidate")
	}
	if updated.ID != created.ID {
		t.Fatalf("expected same candidate id, got created=%q updated=%q", created.ID, updated.ID)
	}
	if updated.Position != "Senior Software Engineer" {
		t.Fatalf("expected position updated, got %q", updated.Position)
	}
	if updated.MatchScore != 92 {
		t.Fatalf("expected score updated, got %d", updated.MatchScore)
	}
}

type stubRepository struct {
	candidates []model.Candidate
}

func (s *stubRepository) List() []model.Candidate {
	out := make([]model.Candidate, len(s.candidates))
	copy(out, s.candidates)
	return out
}

func (s *stubRepository) Create(input model.CandidateMutationInput) (model.Candidate, error) {
	candidate := model.Candidate{
		ID:                  "cand_test",
		Company:             input.Company,
		Position:            input.Position,
		Source:              input.Source,
		Location:            input.Location,
		JDURL:               input.JDURL,
		CompanyWebsiteURL:   input.CompanyWebsiteURL,
		Status:              input.Status,
		MatchScore:          input.MatchScore,
		MatchReasons:        append([]string(nil), input.MatchReasons...),
		RecommendationNotes: input.RecommendationNotes,
		Notes:               input.Notes,
		PromotedLeadID:      input.PromotedLeadID,
	}
	s.candidates = append(s.candidates, candidate)
	return candidate, nil
}

func (s *stubRepository) Update(id string, input model.CandidateMutationInput) (model.Candidate, bool, error) {
	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}
		updated := s.candidates[i]
		updated.Company = input.Company
		updated.Position = input.Position
		updated.Source = input.Source
		updated.Location = input.Location
		updated.JDURL = input.JDURL
		updated.CompanyWebsiteURL = input.CompanyWebsiteURL
		updated.Status = input.Status
		updated.MatchScore = input.MatchScore
		updated.MatchReasons = append([]string(nil), input.MatchReasons...)
		updated.RecommendationNotes = input.RecommendationNotes
		updated.Notes = input.Notes
		updated.PromotedLeadID = input.PromotedLeadID
		s.candidates[i] = updated
		return updated, true, nil
	}
	return model.Candidate{}, false, nil
}

func (s *stubRepository) Delete(id string) (bool, error) {
	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}
		s.candidates = append(s.candidates[:i], s.candidates[i+1:]...)
		return true, nil
	}
	return false, nil
}

type stubLeadManager struct {
	createdCount int
}

func (s *stubLeadManager) List() []model.Lead {
	return nil
}

func (s *stubLeadManager) Create(input model.LeadMutationInput) (model.Lead, error) {
	s.createdCount++
	return model.Lead{
		ID:                "lead_created",
		Company:           input.Company,
		Position:          input.Position,
		Source:            input.Source,
		Status:            input.Status,
		Priority:          input.Priority,
		NextAction:        input.NextAction,
		NextActionAt:      input.NextActionAt,
		InterviewAt:       input.InterviewAt,
		ReminderMethods:   append([]string(nil), input.ReminderMethods...),
		Notes:             input.Notes,
		CompanyWebsiteURL: input.CompanyWebsiteURL,
		JDURL:             input.JDURL,
		Location:          input.Location,
	}, nil
}

func (s *stubLeadManager) Update(_ string, _ model.LeadMutationInput) (model.Lead, bool, error) {
	return model.Lead{}, false, nil
}

func (s *stubLeadManager) Delete(_ string) (bool, error) {
	return false, nil
}
