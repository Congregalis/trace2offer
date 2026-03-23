package tool

import (
	"context"
	"encoding/json"
	"testing"

	"trace2offer/backend/internal/model"
)

func TestCandidateToolsCreateAndPromote(t *testing.T) {
	t.Parallel()

	manager := &stubCandidateManager{}
	tools := NewCandidateCRUDTools(manager)
	registry, err := NewRegistry(tools...)
	if err != nil {
		t.Fatalf("new candidate tool registry error: %v", err)
	}

	createOutput, err := registry.Call(context.Background(), "candidate_create", json.RawMessage(`{
		"company":"Anthropic",
		"position":"Software Engineer",
		"source":"LinkedIn",
		"status":"pending_review",
		"match_score":89,
		"match_reasons":["Go backend","LLM infra"],
		"recommendation_notes":"fit"
	}`))
	if err != nil {
		t.Fatalf("candidate_create failed: %v", err)
	}
	var createPayload struct {
		Candidate model.Candidate `json:"candidate"`
	}
	if err := json.Unmarshal([]byte(createOutput), &createPayload); err != nil {
		t.Fatalf("decode create output failed: %v", err)
	}
	if createPayload.Candidate.ID == "" {
		t.Fatal("expected created candidate id not empty")
	}

	promoteOutput, err := registry.Call(context.Background(), "candidate_promote", json.RawMessage(`{
		"id":"cand_1",
		"status":"new",
		"priority":5
	}`))
	if err != nil {
		t.Fatalf("candidate_promote failed: %v", err)
	}
	var promotePayload struct {
		Candidate model.Candidate `json:"candidate"`
		Lead      model.Lead      `json:"lead"`
	}
	if err := json.Unmarshal([]byte(promoteOutput), &promotePayload); err != nil {
		t.Fatalf("decode promote output failed: %v", err)
	}
	if promotePayload.Lead.ID == "" {
		t.Fatal("expected promoted lead id not empty")
	}
	if promotePayload.Candidate.Status != "promoted" {
		t.Fatalf("expected candidate status promoted, got %q", promotePayload.Candidate.Status)
	}
}

type stubCandidateManager struct {
	candidates []model.Candidate
}

func (s *stubCandidateManager) List() []model.Candidate {
	copied := make([]model.Candidate, len(s.candidates))
	copy(copied, s.candidates)
	return copied
}

func (s *stubCandidateManager) Create(input model.CandidateMutationInput) (model.Candidate, error) {
	created := model.Candidate{
		ID:                  "cand_1",
		Company:             input.Company,
		Position:            input.Position,
		Source:              input.Source,
		Status:              input.Status,
		MatchScore:          input.MatchScore,
		MatchReasons:        append([]string(nil), input.MatchReasons...),
		RecommendationNotes: input.RecommendationNotes,
	}
	s.candidates = append(s.candidates, created)
	return created, nil
}

func (s *stubCandidateManager) Update(id string, input model.CandidateMutationInput) (model.Candidate, bool, error) {
	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}
		updated := s.candidates[i]
		updated.Company = input.Company
		updated.Position = input.Position
		updated.Source = input.Source
		updated.Status = input.Status
		updated.MatchScore = input.MatchScore
		updated.MatchReasons = append([]string(nil), input.MatchReasons...)
		updated.RecommendationNotes = input.RecommendationNotes
		updated.PromotedLeadID = input.PromotedLeadID
		s.candidates[i] = updated
		return updated, true, nil
	}
	return model.Candidate{}, false, nil
}

func (s *stubCandidateManager) Delete(id string) (bool, error) {
	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}
		s.candidates = append(s.candidates[:i], s.candidates[i+1:]...)
		return true, nil
	}
	return false, nil
}

func (s *stubCandidateManager) UpsertByJDURL(input model.CandidateMutationInput) (model.Candidate, bool, error) {
	for i := range s.candidates {
		if s.candidates[i].JDURL != input.JDURL {
			continue
		}
		updated := s.candidates[i]
		updated.Company = input.Company
		updated.Position = input.Position
		updated.Source = input.Source
		updated.Status = input.Status
		updated.MatchScore = input.MatchScore
		updated.MatchReasons = append([]string(nil), input.MatchReasons...)
		updated.RecommendationNotes = input.RecommendationNotes
		s.candidates[i] = updated
		return updated, false, nil
	}
	created, err := s.Create(input)
	return created, true, err
}

func (s *stubCandidateManager) Promote(id string, _ model.CandidatePromoteInput) (model.Candidate, model.Lead, error) {
	for i := range s.candidates {
		if s.candidates[i].ID != id {
			continue
		}
		s.candidates[i].Status = "promoted"
		s.candidates[i].PromotedLeadID = "lead_1"
		return s.candidates[i], model.Lead{
			ID:       "lead_1",
			Company:  s.candidates[i].Company,
			Position: s.candidates[i].Position,
			Status:   "new",
		}, nil
	}
	return model.Candidate{}, model.Lead{}, nil
}
