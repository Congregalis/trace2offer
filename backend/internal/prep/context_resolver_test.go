package prep

import (
	"os"
	"path/filepath"
	"testing"

	"trace2offer/backend/internal/model"
)

func TestContextResolverResolveAggregatesSources(t *testing.T) {
	t.Parallel()

	service, dataRoot := newPrepServiceForContextTest(t)

	if _, err := service.CreateTopic(TopicCreateInput{
		Key:         "rag",
		Name:        "RAG",
		Description: "检索增强生成",
	}); err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if _, err := service.CreateKnowledgeDocument("topics", "rag", KnowledgeDocumentCreateInput{
		Filename: "overview",
		Content:  "# RAG",
	}); err != nil {
		t.Fatalf("create topic doc: %v", err)
	}

	writeFileForContextTest(t, filepath.Join(dataRoot, "resume", "current.md"), "# Resume\n\nGo engineer")

	preview, err := service.GetLeadContextPreview(model.Lead{
		ID:       "lead_ctx",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		JDText:   "JD content",
	})
	if err != nil {
		t.Fatalf("get lead context preview: %v", err)
	}

	if !preview.HasResume {
		t.Fatalf("expected has_resume=true")
	}
	if !contextSourceExists(preview.Sources, "lead", "jd_text", "JD 原文") {
		t.Fatalf("expected jd source included, got %+v", preview.Sources)
	}
	if !contextSourceExists(preview.Sources, "topic", "markdown", "rag/overview.md") {
		t.Fatalf("expected topic doc source included, got %+v", preview.Sources)
	}
}

func TestContextResolverResolveWithoutJD(t *testing.T) {
	t.Parallel()

	service, _ := newPrepServiceForContextTest(t)

	preview, err := service.GetLeadContextPreview(model.Lead{
		ID:       "lead_no_jd",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		JDText:   "",
	})
	if err != nil {
		t.Fatalf("get lead context preview: %v", err)
	}
	if contextSourceExists(preview.Sources, "lead", "jd_text", "JD 原文") {
		t.Fatalf("expected no jd source when jd_text is empty, got %+v", preview.Sources)
	}
}

func TestContextResolverResolveWithoutResume(t *testing.T) {
	t.Parallel()

	service, _ := newPrepServiceForContextTest(t)
	preview, err := service.GetLeadContextPreview(model.Lead{
		ID:       "lead_no_resume",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		JDText:   "JD content",
	})
	if err != nil {
		t.Fatalf("get lead context preview: %v", err)
	}
	if preview.HasResume {
		t.Fatalf("expected has_resume=false when resume file missing")
	}
}

func TestContextResolverResolveWithEmptyTopics(t *testing.T) {
	t.Parallel()

	service, _ := newPrepServiceForContextTest(t)
	preview, err := service.GetLeadContextPreview(model.Lead{
		ID:       "lead_topic_empty",
		Company:  "OpenAI",
		Position: "Agent Engineer",
		JDText:   "JD content",
	})
	if err != nil {
		t.Fatalf("get lead context preview: %v", err)
	}
	if contextSourceExistsByScope(preview.Sources, "topic") {
		t.Fatalf("expected no topic sources when topics empty, got %+v", preview.Sources)
	}
}

func newPrepServiceForContextTest(t *testing.T) (*Service, string) {
	t.Helper()

	dataRoot := t.TempDir()
	service, err := NewService(Config{
		Enabled:              true,
		DataDir:              filepath.Join(dataRoot, "prep"),
		DefaultQuestionCount: defaultQuestionCount,
		SupportedScopes:      DefaultSupportedScopes(),
	})
	if err != nil {
		t.Fatalf("new prep service: %v", err)
	}
	return service, dataRoot
}

func writeFileForContextTest(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func contextSourceExists(sources []ContextSource, scope string, kind string, title string) bool {
	for _, source := range sources {
		if source.Scope == scope && source.Kind == kind && source.Title == title {
			return true
		}
	}
	return false
}

func contextSourceExistsByScope(sources []ContextSource, scope string) bool {
	for _, source := range sources {
		if source.Scope == scope {
			return true
		}
	}
	return false
}
