package prep

import (
	"path/filepath"
	"testing"
)

func TestServiceSubmitSessionRoundtrip(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	if err := store.Create(&Session{
		ID:       "prep_service_submit",
		LeadID:   "lead_001",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "answer"},
		},
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	service := &Service{
		config:       Config{Enabled: true},
		sessionStore: store,
	}

	submitted, err := service.SubmitSession("prep_service_submit")
	if err != nil {
		t.Fatalf("submit session: %v", err)
	}
	if submitted.Status != PrepSessionStatusSubmitted {
		t.Fatalf("expected submitted status, got %q", submitted.Status)
	}
	if len(submitted.Answers) != 1 || submitted.Answers[0].SubmittedAt == nil || *submitted.Answers[0].SubmittedAt == "" {
		t.Fatalf("expected submitted_at in answers, got %+v", submitted.Answers)
	}
}

func TestServiceSubmitSessionValidation(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	if err := store.Create(&Session{
		ID:       "prep_service_empty",
		LeadID:   "lead_002",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "  "},
		},
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	service := &Service{
		config:       Config{Enabled: true},
		sessionStore: store,
	}

	if _, err := service.SubmitSession("prep_service_empty"); !IsValidationError(err) {
		t.Fatalf("expected validation error for empty answers, got %v", err)
	}

	if err := store.Create(&Session{
		ID:       "prep_service_twice",
		LeadID:   "lead_003",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "ok"},
		},
	}); err != nil {
		t.Fatalf("create second session: %v", err)
	}

	if _, err := service.SubmitSession("prep_service_twice"); err != nil {
		t.Fatalf("first submit should pass, got %v", err)
	}
	if _, err := service.SubmitSession("prep_service_twice"); !IsValidationError(err) {
		t.Fatalf("second submit should be validation error, got %v", err)
	}
}
