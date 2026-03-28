package prep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trace2offer/backend/internal/model"
)

type stubQuestionModel struct {
	output           string
	lastSystemPrompt string
	lastUserPrompt   string
}

func (m *stubQuestionModel) Name() string {
	return "stub-model"
}

func (m *stubQuestionModel) Generate(_ context.Context, systemPrompt string, userPrompt string) (string, error) {
	m.lastSystemPrompt = systemPrompt
	m.lastUserPrompt = userPrompt
	return m.output, nil
}

func TestParseGeneratedQuestionsFallbackOnMalformedJSON(t *testing.T) {
	t.Parallel()

	questions := parseGeneratedQuestions("not-a-json-payload", 3, []string{"rag-fundamentals.md"}, []string{"rag"})
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}
	if questions[0].ID != 1 {
		t.Fatalf("expected first fallback question id=1, got %d", questions[0].ID)
	}
	if questions[0].Content == "" {
		t.Fatal("expected fallback question content not empty")
	}
}

func TestQuestionGeneratorGenerateBuildsTraceAndSession(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	prepDir := filepath.Join(tmpDir, "prep")
	if err := os.MkdirAll(prepDir, 0o755); err != nil {
		t.Fatalf("mkdir prep dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "resume"), 0o755); err != nil {
		t.Fatalf("mkdir resume dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "resume", "current.md"), []byte("Go backend engineer"), 0o644); err != nil {
		t.Fatalf("write resume: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "user_profile.json"), []byte(`{"core_skills":["Go","RAG"]}`), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	topicStore, err := NewTopicStore(filepath.Join(prepDir, "topic_catalog.json"))
	if err != nil {
		t.Fatalf("new topic store: %v", err)
	}
	if _, err := topicStore.Create(TopicCreateInput{Key: "rag", Name: "RAG", Description: "retrieval"}); err != nil {
		t.Fatalf("create topic: %v", err)
	}

	knowledgeStore, err := NewKnowledgeStore(filepath.Join(prepDir, "knowledge"))
	if err != nil {
		t.Fatalf("new knowledge store: %v", err)
	}
	if _, err := knowledgeStore.Create("topics", "rag", KnowledgeDocumentCreateInput{Filename: "rag-fundamentals", Content: "RAG architecture"}); err != nil {
		t.Fatalf("create topic doc: %v", err)
	}

	resolver := NewContextResolver(prepDir, topicStore, knowledgeStore)
	indexStore, err := NewIndexStore(filepath.Join(prepDir, "prep_index.sqlite"))
	if err != nil {
		t.Fatalf("new index store: %v", err)
	}
	t.Cleanup(func() { _ = indexStore.Close() })
	retrievalEngine := NewRetrievalEngine(indexStore, nil)
	modelStub := &stubQuestionModel{output: `{"questions":[{"id":1,"type":"technical","content":"Explain RAG architecture","expected_points":["retrieval","generation"],"context_sources":["rag-fundamentals.md"]}]}`}
	generator := NewQuestionGenerator(resolver, retrievalEngine, nil, nil, modelStub, 8)

	generated, err := generator.GenerateWithContext(context.Background(), GenerationConfig{
		Lead: model.Lead{
			ID:       "lead_123",
			Company:  "OpenAI",
			Position: "Agent Engineer",
			JDText:   "Need strong RAG and system design skills",
		},
		LeadID:          "lead_123",
		TopicKeys:       []string{"rag"},
		QuestionCount:   2,
		IncludeResume:   true,
		IncludeProfile:  true,
		IncludeLeadDocs: true,
	})
	if err != nil {
		t.Fatalf("generate session: %v", err)
	}
	if generated == nil || generated.Session == nil {
		t.Fatal("expected generated session payload")
	}
	session := generated.Session

	if session.ID == "" {
		t.Fatal("expected generated session id")
	}
	if len(session.Questions) != 2 {
		t.Fatalf("expected 2 questions (including fallback fill), got %d", len(session.Questions))
	}
	if session.GenerationTrace.InputSnapshot.LeadID != "lead_123" {
		t.Fatalf("unexpected trace lead id: %q", session.GenerationTrace.InputSnapshot.LeadID)
	}
	if session.GenerationTrace.RetrievalQuery == "" {
		t.Fatal("expected retrieval query in trace")
	}
	if session.GenerationTrace.GenerationResult.Model != "stub-model" {
		t.Fatalf("expected model trace stub-model, got %q", session.GenerationTrace.GenerationResult.Model)
	}
	if !strings.Contains(modelStub.lastUserPrompt, "Resume:") {
		t.Fatalf("expected prompt contains Resume section, got: %s", modelStub.lastUserPrompt)
	}
	if !strings.Contains(modelStub.lastUserPrompt, "User Profile:") {
		t.Fatalf("expected prompt contains User Profile section, got: %s", modelStub.lastUserPrompt)
	}
	if !strings.Contains(modelStub.lastUserPrompt, "Lead Summary:") {
		t.Fatalf("expected prompt contains Lead Summary section, got: %s", modelStub.lastUserPrompt)
	}
}
