package prep

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStoreCreateGetUpdate(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	session := Session{
		ID:       "prep_01",
		LeadID:   "lead_123",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "technical", Content: "Explain RAG", ExpectedPoints: []string{"retrieval"}},
		},
		Answers: []Answer{},
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	loaded, err := store.Get("prep_01")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if loaded.LeadID != "lead_123" {
		t.Fatalf("unexpected lead id: %q", loaded.LeadID)
	}

	loaded.Status = "submitted"
	if err := store.Update(loaded); err != nil {
		t.Fatalf("update session: %v", err)
	}

	updated, err := store.Get("prep_01")
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if updated.Status != "submitted" {
		t.Fatalf("expected status submitted, got %q", updated.Status)
	}
}

func TestSessionStoreGetMissingReturnsNotFoundFalse(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	_, err = store.Get("prep_missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionStoreUpdateAnswersRequiresQuestionInSession(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	session := Session{
		ID:       "prep_02",
		LeadID:   "lead_456",
		Company:  "Anthropic",
		Position: "Backend Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "technical", Content: "Explain context windows"},
		},
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	err = store.UpdateAnswers("prep_02", []Answer{{QuestionID: 99, Answer: "unknown"}})
	if err == nil {
		t.Fatal("expected update answers error for missing question id")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	err = store.UpdateAnswers("prep_not_exists", []Answer{{QuestionID: 1, Answer: "answer"}})
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionStoreUpdateAnswersRoundtripMerge(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	createdAt := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	updatedAt := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	session := Session{
		ID:       "prep_roundtrip",
		LeadID:   "lead_789",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
			{ID: 2, Type: "open", Content: "Q2"},
		},
		Answers:   []Answer{{QuestionID: 1, Answer: "old answer"}},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := store.UpdateAnswers("prep_roundtrip", []Answer{{QuestionID: 1, Answer: "new answer"}, {QuestionID: 2, Answer: "second answer"}}); err != nil {
		t.Fatalf("update answers: %v", err)
	}

	loaded, err := store.Get("prep_roundtrip")
	if err != nil {
		t.Fatalf("get session after update answers: %v", err)
	}
	if len(loaded.Answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(loaded.Answers))
	}
	if loaded.Answers[0].QuestionID != 1 || loaded.Answers[0].Answer != "new answer" {
		t.Fatalf("unexpected answer[0]: %+v", loaded.Answers[0])
	}
	if loaded.Answers[1].QuestionID != 2 || loaded.Answers[1].Answer != "second answer" {
		t.Fatalf("unexpected answer[1]: %+v", loaded.Answers[1])
	}
	if loaded.UpdatedAt == updatedAt {
		t.Fatalf("expected updated_at changed, got %q", loaded.UpdatedAt)
	}
}

func TestSessionStoreSubmitRoundtrip(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	session := Session{
		ID:       "prep_submit",
		LeadID:   "lead_001",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "my answer"},
		},
		CreatedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	submitted, err := store.Submit("prep_submit")
	if err != nil {
		t.Fatalf("submit session: %v", err)
	}
	if submitted.Status != PrepSessionStatusSubmitted {
		t.Fatalf("expected submitted status, got %q", submitted.Status)
	}
	if len(submitted.Answers) != 1 {
		t.Fatalf("expected one answer, got %d", len(submitted.Answers))
	}
	if submitted.Answers[0].SubmittedAt == nil || *submitted.Answers[0].SubmittedAt == "" {
		t.Fatalf("expected submitted_at set, got %+v", submitted.Answers[0])
	}

	loaded, err := store.Get("prep_submit")
	if err != nil {
		t.Fatalf("get submitted session: %v", err)
	}
	if loaded.Status != PrepSessionStatusSubmitted {
		t.Fatalf("expected submitted status persisted, got %q", loaded.Status)
	}
	if len(loaded.Answers) != 1 || loaded.Answers[0].SubmittedAt == nil || *loaded.Answers[0].SubmittedAt == "" {
		t.Fatalf("expected submitted_at persisted, got %+v", loaded.Answers)
	}
}

func TestSessionStoreSubmitRequiresNonEmptyAnswer(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	session := Session{
		ID:       "prep_submit_empty",
		LeadID:   "lead_002",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "   "},
		},
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	_, err = store.Submit("prep_submit_empty")
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSessionStoreSubmitTwiceReturnsValidationError(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	session := Session{
		ID:       "prep_submit_twice",
		LeadID:   "lead_003",
		Company:  "OpenAI",
		Position: "Engineer",
		Status:   PrepSessionStatusDraft,
		Questions: []Question{
			{ID: 1, Type: "open", Content: "Q1"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "answer"},
		},
	}
	if err := store.Create(&session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	if _, err := store.Submit("prep_submit_twice"); err != nil {
		t.Fatalf("first submit should pass, got %v", err)
	}
	if _, err := store.Submit("prep_submit_twice"); !IsValidationError(err) {
		t.Fatalf("second submit should be validation error, got %v", err)
	}
}
