package prep

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIngestionHashChangeDetection(t *testing.T) {
	t.Parallel()

	ingestion, prepDataDir := newIngestionServiceForTest(t)
	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "overview.md", "# RAG\n\nfirst")

	first, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("first rebuild index: %v", err)
	}
	if first == nil {
		t.Fatal("expected first summary")
	}
	if first.DocumentsScanned != 1 || first.DocumentsIndexed != 1 || first.DocumentsSkipped != 0 {
		t.Fatalf("unexpected first summary: %+v", first)
	}
	if first.ChunksCreated == 0 {
		t.Fatalf("expected chunks created on first run, got %+v", first)
	}

	second, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("second rebuild index: %v", err)
	}
	if second == nil {
		t.Fatal("expected second summary")
	}
	if second.DocumentsScanned != 1 || second.DocumentsIndexed != 0 || second.DocumentsSkipped != 1 {
		t.Fatalf("unexpected second summary: %+v", second)
	}

	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "overview.md", "# RAG\n\nsecond")
	third, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("third rebuild index: %v", err)
	}
	if third == nil {
		t.Fatal("expected third summary")
	}
	if third.DocumentsIndexed != 1 || third.DocumentsSkipped != 0 {
		t.Fatalf("expected re-index after hash change, got %+v", third)
	}
	if third.ChunksUpdated == 0 {
		t.Fatalf("expected chunks updated after hash change, got %+v", third)
	}
}

func TestIngestionPartialRebuildScopeFiltering(t *testing.T) {
	t.Parallel()

	ingestion, prepDataDir := newIngestionServiceForTest(t)
	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "topic.md", "topic")

	summary, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("partial rebuild: %v", err)
	}
	if summary == nil {
		t.Fatal("expected summary")
	}
	if summary.DocumentsScanned != 1 {
		t.Fatalf("expected scanned=1 for partial rebuild, got %+v", summary)
	}

	var topicCount int
	if err := ingestion.indexStore.db.QueryRow(`SELECT COUNT(*) FROM documents WHERE scope = ?`, string(ScopeTopics)).Scan(&topicCount); err != nil {
		t.Fatalf("count topic docs: %v", err)
	}
	if topicCount != 1 {
		t.Fatalf("expected topic docs=1, got %d", topicCount)
	}
}

func TestIndexRunSummarySerialization(t *testing.T) {
	t.Parallel()

	summary := IndexRunSummary{
		RunID:            "run_123",
		StartedAt:        "2026-03-27T10:00:00Z",
		CompletedAt:      "2026-03-27T10:00:05Z",
		Status:           IndexRunStatusCompleted,
		DocumentsScanned: 5,
		DocumentsIndexed: 4,
		DocumentsSkipped: 1,
		ChunksCreated:    42,
		ChunksUpdated:    8,
		Errors: []IndexRunError{
			{Source: "knowledge/topics/rag/bad.md", Message: "permission denied"},
		},
	}

	payload, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if decoded["run_id"] != "run_123" {
		t.Fatalf("expected run_id serialized, got %+v", decoded)
	}
	if decoded["documents_scanned"] != float64(5) {
		t.Fatalf("expected documents_scanned serialized, got %+v", decoded)
	}
	if _, ok := decoded["errors"]; !ok {
		t.Fatalf("expected errors field serialized, got %+v", decoded)
	}
}

func TestIngestionErrorHandlingForUnreadableOrBrokenFiles(t *testing.T) {
	t.Parallel()

	ingestion, prepDataDir := newIngestionServiceForTest(t)
	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "ok.md", "ok")

	unreadablePath := writeKnowledgeFile(t, prepDataDir, "topics", "rag", "noaccess.md", "secret")
	if err := os.Chmod(unreadablePath, 0); err != nil {
		t.Fatalf("chmod unreadable file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadablePath, 0o644)
	})

	brokenPath := filepath.Join(prepDataDir, "knowledge", "topics", "rag", "broken.md")
	if err := os.Symlink(filepath.Join(prepDataDir, "knowledge", "topics", "rag", "missing-target.md"), brokenPath); err != nil {
		t.Fatalf("create broken symlink: %v", err)
	}

	summary, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("rebuild with unreadable files should not fail hard: %v", err)
	}
	if summary == nil {
		t.Fatal("expected summary")
	}
	if summary.DocumentsScanned != 3 {
		t.Fatalf("expected scanned documents=3, got %+v", summary)
	}
	if len(summary.Errors) == 0 {
		t.Fatalf("expected at least one per-file error, got %+v", summary)
	}
	hasBroken := false
	for _, item := range summary.Errors {
		if strings.Contains(item.Source, "broken.md") || strings.Contains(item.Source, "noaccess.md") {
			hasBroken = true
			break
		}
	}
	if !hasBroken {
		t.Fatalf("expected broken/unreadable file errors, got %+v", summary.Errors)
	}
	if summary.DocumentsIndexed == 0 {
		t.Fatalf("expected readable docs still indexed, got %+v", summary)
	}
}

func TestIngestionRejectsNonTopicScope(t *testing.T) {
	t.Parallel()

	ingestion, prepDataDir := newIngestionServiceForTest(t)
	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "overview.md", "content")

	_, err := ingestion.Ingest("leads", "lead_1")
	if err == nil {
		t.Fatal("expected non-topic scope ingest to fail")
	}
}

func TestIngestionIncrementalRemovesStaleDocuments(t *testing.T) {
	t.Parallel()

	ingestion, prepDataDir := newIngestionServiceForTest(t)
	targetPath := writeKnowledgeFile(t, prepDataDir, "topics", "rag", "overview.md", "content")

	first, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	if first.DocumentsIndexed != 1 {
		t.Fatalf("expected first indexed=1, got %+v", first)
	}

	if err := os.Remove(targetPath); err != nil {
		t.Fatalf("remove source file: %v", err)
	}
	second, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if second.DocumentsDeleted != 1 {
		t.Fatalf("expected stale delete count 1, got %+v", second)
	}

	var count int
	if err := ingestion.indexStore.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&count); err != nil {
		t.Fatalf("count docs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero docs after stale cleanup, got %d", count)
	}
}

func TestIngestionFullModeAlwaysRebuilds(t *testing.T) {
	t.Parallel()

	ingestion, prepDataDir := newIngestionServiceForTest(t)
	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "overview.md", "content")

	first, err := ingestion.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	if first.DocumentsIndexed != 1 {
		t.Fatalf("expected first indexed=1, got %+v", first)
	}

	second, err := ingestion.IngestWithMode("topics", "rag", RebuildModeFull)
	if err != nil {
		t.Fatalf("full ingest: %v", err)
	}
	if second.Mode != RebuildModeFull {
		t.Fatalf("expected mode full, got %+v", second)
	}
	if second.DocumentsDeleted == 0 {
		t.Fatalf("expected full mode deletes existing docs first, got %+v", second)
	}
	if second.DocumentsIndexed != 1 || second.DocumentsSkipped != 0 {
		t.Fatalf("expected full mode reindex same doc, got %+v", second)
	}
}

func TestIngestionFingerprintMismatchForcesFullRebuild(t *testing.T) {
	t.Parallel()

	prepDataDir := filepath.Join(t.TempDir(), "prep")
	indexStore, err := NewIndexStore(filepath.Join(prepDataDir, "prep_index.sqlite"))
	if err != nil {
		t.Fatalf("new index store: %v", err)
	}
	t.Cleanup(func() {
		_ = indexStore.Close()
	})
	writeKnowledgeFile(t, prepDataDir, "topics", "rag", "overview.md", "content")

	providerV1 := &stubInfoEmbeddingProvider{name: "stub", model: "v1", dimension: 2}
	ingestionV1, err := NewIngestionService(prepDataDir, IngestionDependencies{
		IndexStore:        indexStore,
		EmbeddingProvider: providerV1,
		ChunkConfig:       ChunkConfig{ChunkSize: 16, Overlap: 4},
	})
	if err != nil {
		t.Fatalf("new ingestion v1: %v", err)
	}
	if _, err := ingestionV1.Ingest("topics", "rag"); err != nil {
		t.Fatalf("ingest v1: %v", err)
	}

	providerV2 := &stubInfoEmbeddingProvider{name: "stub", model: "v2", dimension: 2}
	ingestionV2, err := NewIngestionService(prepDataDir, IngestionDependencies{
		IndexStore:        indexStore,
		EmbeddingProvider: providerV2,
		ChunkConfig:       ChunkConfig{ChunkSize: 16, Overlap: 4},
	})
	if err != nil {
		t.Fatalf("new ingestion v2: %v", err)
	}
	summary, err := ingestionV2.Ingest("topics", "rag")
	if err != nil {
		t.Fatalf("ingest v2: %v", err)
	}
	if summary.Mode != RebuildModeFull {
		t.Fatalf("expected fingerprint mismatch forces full mode, got %+v", summary)
	}
	if summary.DocumentsDeleted == 0 {
		t.Fatalf("expected full mode deletes existing docs, got %+v", summary)
	}
	if summary.DocumentsIndexed != 1 || summary.DocumentsSkipped != 0 {
		t.Fatalf("expected full rebuild indexes doc again, got %+v", summary)
	}
}

func newIngestionServiceForTest(t *testing.T) (*IngestionService, string) {
	t.Helper()

	prepDataDir := filepath.Join(t.TempDir(), "prep")
	indexStore, err := NewIndexStore(filepath.Join(prepDataDir, "prep_index.sqlite"))
	if err != nil {
		t.Fatalf("new index store: %v", err)
	}
	t.Cleanup(func() {
		_ = indexStore.Close()
	})

	ingestion, err := NewIngestionService(prepDataDir, IngestionDependencies{
		IndexStore: indexStore,
		EmbeddingProvider: &stubEmbeddingProvider{
			vectors: map[string][]float32{},
		},
		ChunkConfig: ChunkConfig{ChunkSize: 16, Overlap: 4},
	})
	if err != nil {
		t.Fatalf("new ingestion service: %v", err)
	}
	return ingestion, prepDataDir
}

func writeKnowledgeFile(t *testing.T, prepDataDir string, scope string, scopeID string, filename string, content string) string {
	t.Helper()

	path := filepath.Join(prepDataDir, "knowledge", scope, scopeID, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir knowledge dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write knowledge file: %v", err)
	}
	return path
}

type stubInfoEmbeddingProvider struct {
	name      string
	model     string
	dimension int
}

func (p *stubInfoEmbeddingProvider) Name() string {
	return p.name
}

func (p *stubInfoEmbeddingProvider) Model() string {
	return p.model
}

func (p *stubInfoEmbeddingProvider) Dimension() int {
	return p.dimension
}

func (p *stubInfoEmbeddingProvider) Embed(texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for range texts {
		vectors = append(vectors, []float32{0.5, 0.5})
	}
	return vectors, nil
}
