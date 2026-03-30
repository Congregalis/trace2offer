package prep

import (
	"context"
	"path/filepath"
	"testing"
)

type stubReferenceRetrieval struct {
	calls int
	empty bool
}

func (s *stubReferenceRetrieval) Search(query string, _ SearchConfig) (*SearchResult, error) {
	s.calls++
	if s.empty {
		return &SearchResult{
			Query:           query,
			CandidateChunks: []RetrievedChunk{},
			RetrievedChunks: []RetrievedChunk{},
		}, nil
	}
	return &SearchResult{
		Query: query,
		RetrievedChunks: []RetrievedChunk{
			{
				ID:      "ref_chunk_1",
				Content: "agent runtime should separate planner and executor",
				Score:   0.84,
				Source: ChunkSource{
					Scope:         "topics",
					ScopeID:       "rag",
					DocumentTitle: "runtime.md",
					ChunkIndex:    0,
				},
			},
		},
	}, nil
}

type stubReferenceModel struct {
	output string
	calls  int
}

func (m *stubReferenceModel) Name() string {
	return "stub-reference-model"
}

func (m *stubReferenceModel) Generate(_ context.Context, _, _ string) (string, error) {
	m.calls++
	return m.output, nil
}

func TestGenerateReferenceAnswerCacheHit(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	session := &Session{
		ID:       "prep_ref_cache",
		LeadID:   "lead_1",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		Status:   PrepSessionStatusSubmitted,
		Config: SessionConfig{
			IncludeResume:   true,
			IncludeLeadDocs: true,
		},
		Questions: []Question{
			{ID: 1, Content: "如何设计 Agent Runtime？", ExpectedPoints: []string{"状态管理", "工具规范"}},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "先拆分 planner/executor，再定义工具协议。"},
		},
		ReferenceAnswers: map[string]ReferenceAnswer{},
	}
	if err := store.Create(session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	retrieval := &stubReferenceRetrieval{}
	model := &stubReferenceModel{output: `{"reference_answer":"建议从状态机、工具协议、回放评测三层设计。"} `}
	service := &Service{
		config: Config{
			Enabled: true,
		},
		sessionStore:       store,
		referenceAnswerGen: NewReferenceAnswerGenerator(retrieval, model),
	}

	first, err := service.GenerateReferenceAnswer("prep_ref_cache", 1)
	if err != nil {
		t.Fatalf("first generate reference answer: %v", err)
	}
	second, err := service.GenerateReferenceAnswer("prep_ref_cache", 1)
	if err != nil {
		t.Fatalf("second generate reference answer: %v", err)
	}

	if first.ReferenceAnswer == "" || second.ReferenceAnswer == "" {
		t.Fatalf("expected non-empty reference answer, got first=%+v second=%+v", first, second)
	}
	if model.calls != 1 {
		t.Fatalf("expected model called once due cache hit, got %d", model.calls)
	}
	if retrieval.calls != 1 {
		t.Fatalf("expected retrieval called once due cache hit, got %d", retrieval.calls)
	}

	latest, err := store.Get("prep_ref_cache")
	if err != nil {
		t.Fatalf("load cached session: %v", err)
	}
	if latest.ReferenceAnswers == nil || latest.ReferenceAnswers["1"].ReferenceAnswer == "" {
		t.Fatalf("expected cached reference answer persisted, got %+v", latest.ReferenceAnswers)
	}
}

func TestGenerateReferenceAnswerInvalidQuestionID(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	if err := store.Create(&Session{
		ID:               "prep_ref_invalid_q",
		LeadID:           "lead_2",
		Company:          "OpenAI",
		Position:         "Agent Engineer",
		Status:           PrepSessionStatusSubmitted,
		Questions:        []Question{{ID: 1, Content: "Q1"}},
		ReferenceAnswers: map[string]ReferenceAnswer{},
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	service := &Service{
		config:       Config{Enabled: true},
		sessionStore: store,
		referenceAnswerGen: NewReferenceAnswerGenerator(
			&stubReferenceRetrieval{},
			&stubReferenceModel{output: `{"reference_answer":"ok"}`},
		),
	}

	_, err = service.GenerateReferenceAnswer("prep_ref_invalid_q", 99)
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid question id, got %v", err)
	}
}

func TestGenerateReferenceAnswerWithEmptyRetrievalResults(t *testing.T) {
	t.Parallel()

	store, err := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	if err := store.Create(&Session{
		ID:       "prep_ref_empty_retrieval",
		LeadID:   "lead_3",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		Status:   PrepSessionStatusSubmitted,
		Questions: []Question{
			{ID: 1, Content: "怎么做评测体系？"},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "通过离线数据集回归。"},
		},
		ReferenceAnswers: map[string]ReferenceAnswer{},
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}

	service := &Service{
		config:       Config{Enabled: true},
		sessionStore: store,
		referenceAnswerGen: NewReferenceAnswerGenerator(
			&stubReferenceRetrieval{empty: true},
			&stubReferenceModel{output: `{"reference_answer":"先定义评分维度，再做回归集与线上监控联动。"} `},
		),
	}

	result, err := service.GenerateReferenceAnswer("prep_ref_empty_retrieval", 1)
	if err != nil {
		t.Fatalf("generate reference answer with empty retrieval: %v", err)
	}
	if result.ReferenceAnswer == "" {
		t.Fatalf("expected non-empty reference answer, got %+v", result)
	}
	if len(result.Sources) != 0 {
		t.Fatalf("expected no sources when retrieval empty, got %+v", result.Sources)
	}
}
