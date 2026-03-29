package prep

import (
	"context"
	"testing"
)

type stubScoringRetrieval struct {
	calls   int
	queries []string
}

func (s *stubScoringRetrieval) Search(query string, _ SearchConfig) (*SearchResult, error) {
	s.calls++
	s.queries = append(s.queries, query)
	return &SearchResult{
		Query: query,
		RetrievedChunks: []RetrievedChunk{
			{
				ID:      "chunk_1",
				Content: "retrieved scoring context",
				Score:   0.81,
				Source: ChunkSource{
					Scope:         "topics",
					ScopeID:       "rag",
					DocumentTitle: "rag.md",
					ChunkIndex:    0,
				},
			},
		},
	}, nil
}

type stubScoringModel struct {
	outputs []string
	calls   int
}

func (m *stubScoringModel) Name() string {
	return "stub-scoring-model"
}

func (m *stubScoringModel) Generate(_ context.Context, _, _ string) (string, error) {
	if len(m.outputs) == 0 {
		m.calls++
		return "{}", nil
	}
	index := m.calls
	if index >= len(m.outputs) {
		index = len(m.outputs) - 1
	}
	m.calls++
	return m.outputs[index], nil
}

func TestScoringEngineEvaluateSessionPartialAnswers(t *testing.T) {
	t.Parallel()

	retrieval := &stubScoringRetrieval{}
	model := &stubScoringModel{
		outputs: []string{
			`{"score": 8.1, "answered": true, "summary":"结构清晰", "strengths":["覆盖核心点"], "improvements":["补充指标"], "weak_points":["缺少量化结果"]}`,
		},
	}
	engine := NewScoringEngine(retrieval, model)

	evaluation, err := engine.EvaluateSession(context.Background(), Session{
		ID:      "prep_score_1",
		LeadID:  "lead_1",
		Company: "OpenAI",
		Config: SessionConfig{
			IncludeResume:   true,
			IncludeLeadDocs: true,
		},
		Questions: []Question{
			{ID: 1, Content: "解释 RAG 架构", ExpectedPoints: []string{"retrieval", "generation"}},
			{ID: 2, Content: "如何做召回评估", ExpectedPoints: []string{"recall", "precision"}},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "RAG 需要先检索，再生成。"},
		},
	})
	if err != nil {
		t.Fatalf("evaluate session: %v", err)
	}
	if evaluation == nil {
		t.Fatal("expected evaluation")
	}
	if len(evaluation.Scores) != 2 {
		t.Fatalf("expected 2 question scores, got %d", len(evaluation.Scores))
	}
	if retrieval.calls != 2 {
		t.Fatalf("expected retrieval per question (2), got %d", retrieval.calls)
	}
	if model.calls != 1 {
		t.Fatalf("expected model call only for answered question, got %d", model.calls)
	}
	if evaluation.Scores[1].Answered {
		t.Fatalf("expected question 2 unanswered, got %+v", evaluation.Scores[1])
	}
	if evaluation.Scores[1].Score != 0 {
		t.Fatalf("expected unanswered score 0, got %.1f", evaluation.Scores[1].Score)
	}
	if evaluation.Overall.AnsweredCount != 1 || evaluation.Overall.TotalQuestions != 2 {
		t.Fatalf("unexpected overall counts: %+v", evaluation.Overall)
	}
}

func TestScoringEngineEvaluateSessionFallsBackWhenJSONInvalid(t *testing.T) {
	t.Parallel()

	engine := NewScoringEngine(&stubScoringRetrieval{}, &stubScoringModel{
		outputs: []string{"invalid-json-output"},
	})
	_, err := engine.EvaluateSession(context.Background(), Session{
		ID:      "prep_score_2",
		LeadID:  "lead_2",
		Company: "OpenAI",
		Questions: []Question{
			{ID: 1, Content: "解释上下文窗口", ExpectedPoints: []string{"限制", "策略"}},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "需要控制 token，避免截断。"},
		},
	})
	if err == nil {
		t.Fatalf("expected parse error on invalid model output, got nil")
	}
}

func TestScoringEngineEvaluateSessionFallsBackMissingFields(t *testing.T) {
	t.Parallel()

	engine := NewScoringEngine(&stubScoringRetrieval{}, &stubScoringModel{
		outputs: []string{`{"score": 7.8}`},
	})
	_, err := engine.EvaluateSession(context.Background(), Session{
		ID:      "prep_score_3",
		LeadID:  "lead_3",
		Company: "OpenAI",
		Questions: []Question{
			{ID: 1, Content: "讲讲索引重建策略", ExpectedPoints: []string{"增量", "全量"}},
		},
		Answers: []Answer{
			{QuestionID: 1, Answer: "优先增量，必要时全量重建。"},
		},
	})
	if err == nil {
		t.Fatalf("expected validation error on missing fields, got nil")
	}
}
