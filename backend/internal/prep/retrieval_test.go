package prep

import (
	"testing"
	"time"
)

func TestRetrievalTopKOrdering(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_a", ChunkID: "chunk_a", Scope: ScopeTopics, ScopeID: "rag", Title: "a.md", Content: "alpha", Embedding: []float32{1, 0}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_b", ChunkID: "chunk_b", Scope: ScopeTopics, ScopeID: "rag", Title: "b.md", Content: "beta", Embedding: []float32{0.9, 0.1}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_c", ChunkID: "chunk_c", Scope: ScopeTopics, ScopeID: "rag", Title: "c.md", Content: "gamma", Embedding: []float32{0.2, 0.8}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.01

	result, err := engine.Search("", SearchConfig{Query: "query", TopK: 2, IncludeTrace: true, TopicKeys: []string{"rag"}})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}

	if len(result.RetrievedChunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(result.RetrievedChunks))
	}
	if result.RetrievedChunks[0].Content != "alpha" {
		t.Fatalf("expected top-1 alpha, got %q", result.RetrievedChunks[0].Content)
	}
	if result.RetrievedChunks[1].Content != "beta" {
		t.Fatalf("expected top-2 beta, got %q", result.RetrievedChunks[1].Content)
	}
}

func TestRetrievalDeduplication(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	sharedHash := hashContent("same content")
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_1", ChunkID: "chunk_1", Scope: ScopeTopics, ScopeID: "rag", Title: "doc1.md", Content: "same content", ContentHash: sharedHash, Embedding: []float32{1, 0}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_2", ChunkID: "chunk_2", Scope: ScopeTopics, ScopeID: "rag", Title: "doc2.md", Content: "same content", ContentHash: sharedHash, Embedding: []float32{1, 0}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.01

	result, err := engine.Search("", SearchConfig{Query: "query", TopK: 5, IncludeTrace: true, LeadID: "lead_1", TopicKeys: []string{"rag"}})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}

	if len(result.RetrievedChunks) != 1 {
		t.Fatalf("expected deduped chunk count 1, got %d", len(result.RetrievedChunks))
	}
	if result.Trace == nil {
		t.Fatalf("expected trace present")
	}
	if result.Trace.StageDeduplication.InputCount != 2 || result.Trace.StageDeduplication.OutputCount != 1 {
		t.Fatalf("unexpected dedup trace: %+v", result.Trace.StageDeduplication)
	}
}

func TestRetrievalScopeAndTopicFilter(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_topic_rag", ChunkID: "chunk_topic_rag", Scope: ScopeTopics, ScopeID: "rag", Title: "rag.md", Content: "topic rag", Embedding: []float32{1, 0}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_topic_system", ChunkID: "chunk_topic_system", Scope: ScopeTopics, ScopeID: "system", Title: "system.md", Content: "topic system", Embedding: []float32{1, 0}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.01

	result, err := engine.Search("", SearchConfig{LeadID: "lead_1", Query: "query", TopK: 10, IncludeTrace: true, TopicKeys: []string{"rag"}})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}

	if len(result.RetrievedChunks) != 1 {
		t.Fatalf("expected 1 filtered topic chunk, got %d", len(result.RetrievedChunks))
	}
	for _, item := range result.RetrievedChunks {
		if item.Source.Scope != string(ScopeTopics) || item.Source.ScopeID != "rag" {
			t.Fatalf("unexpected scope after topic-only filtering: %+v", item.Source)
		}
	}
}

func TestRetrievalReturnsCandidateAndFinalChunks(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_1", ChunkID: "chunk_1", Scope: ScopeTopics, ScopeID: "rag", Title: "1.md", Content: "one", Embedding: []float32{1, 0}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_2", ChunkID: "chunk_2", Scope: ScopeTopics, ScopeID: "rag", Title: "2.md", Content: "two", Embedding: []float32{0.7, 0.3}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_3", ChunkID: "chunk_3", Scope: ScopeTopics, ScopeID: "rag", Title: "3.md", Content: "three", Embedding: []float32{0.2, 0.8}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.95

	result, err := engine.Search("", SearchConfig{
		Query:        "query",
		TopK:         5,
		IncludeTrace: true,
		TopicKeys:    []string{"rag"},
	})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}
	if len(result.CandidateChunks) != 3 {
		t.Fatalf("expected 3 candidate chunks, got %d", len(result.CandidateChunks))
	}
	if len(result.RetrievedChunks) != 1 {
		t.Fatalf("expected 1 final chunk, got %d", len(result.RetrievedChunks))
	}
	if result.Trace == nil {
		t.Fatalf("expected retrieval trace")
	}
	if result.Trace.StageInitialRetrieval.OutputCount != 3 {
		t.Fatalf("expected stage initial output 3, got %+v", result.Trace.StageInitialRetrieval)
	}
	if result.Trace.StageReranking.OutputCount != 1 {
		t.Fatalf("expected stage reranking output 1, got %+v", result.Trace.StageReranking)
	}
}

func TestRetrievalEmptyResult(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_a", ChunkID: "chunk_a", Scope: ScopeTopics, ScopeID: "rag", Title: "a.md", Content: "alpha", Embedding: []float32{0.1, 0.9}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.95

	result, err := engine.Search("", SearchConfig{Query: "query", TopK: 5, IncludeTrace: true, TopicKeys: []string{"rag"}})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}
	if len(result.RetrievedChunks) != 0 {
		t.Fatalf("expected no chunks, got %d", len(result.RetrievedChunks))
	}
	if result.FinalContext.Context != "" {
		t.Fatalf("expected empty context, got %q", result.FinalContext.Context)
	}
}

func TestRetrievalTraceContract(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_a", ChunkID: "chunk_a", Scope: ScopeTopics, ScopeID: "rag", Title: "a.md", Content: "alpha", Embedding: []float32{1, 0}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.01

	result, err := engine.Search("", SearchConfig{Query: "query", TopK: 1, IncludeTrace: true, TopicKeys: []string{"rag"}})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}
	if result.Trace == nil {
		t.Fatal("expected trace in result")
	}
	if result.Trace.StageQueryNormalization.Method == "" {
		t.Fatal("expected query normalization method populated")
	}
	if result.Trace.StageInitialRetrieval.Method != "cosine_similarity" {
		t.Fatalf("unexpected retrieval method: %q", result.Trace.StageInitialRetrieval.Method)
	}
	if result.Trace.StageReranking.Method == "" {
		t.Fatal("expected reranking method populated")
	}
}

func TestRetrievalScoreThresholdFilter(t *testing.T) {
	t.Parallel()

	store := newIndexStoreForRetrievalTest(t)
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_a", ChunkID: "chunk_a", Scope: ScopeTopics, ScopeID: "rag", Title: "a.md", Content: "alpha", Embedding: []float32{1, 0}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_b", ChunkID: "chunk_b", Scope: ScopeTopics, ScopeID: "rag", Title: "b.md", Content: "beta", Embedding: []float32{0.8, 0.2}})
	addIndexedChunkForRetrievalTest(t, store, retrievalChunkFixture{DocID: "doc_c", ChunkID: "chunk_c", Scope: ScopeTopics, ScopeID: "rag", Title: "c.md", Content: "gamma", Embedding: []float32{0.4, 0.6}})

	engine := NewRetrievalEngine(store, &stubEmbeddingProvider{vectors: map[string][]float32{"query": {1, 0}}})
	engine.scoreThreshold = 0.75

	result, err := engine.Search("", SearchConfig{Query: "query", TopK: 5, IncludeTrace: true, TopicKeys: []string{"rag"}})
	if err != nil {
		t.Fatalf("search retrieval: %v", err)
	}

	if len(result.RetrievedChunks) != 2 {
		t.Fatalf("expected 2 chunks above threshold, got %d", len(result.RetrievedChunks))
	}
}

type retrievalChunkFixture struct {
	DocID       string
	ChunkID     string
	Scope       Scope
	ScopeID     string
	Kind        string
	Title       string
	Content     string
	ContentHash string
	Embedding   []float32
}

func newIndexStoreForRetrievalTest(t *testing.T) *IndexStore {
	t.Helper()

	store, err := NewIndexStore(":memory:")
	if err != nil {
		t.Fatalf("new index store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func addIndexedChunkForRetrievalTest(t *testing.T, store *IndexStore, fixture retrievalChunkFixture) {
	t.Helper()

	contentHash := fixture.ContentHash
	if contentHash == "" {
		contentHash = hashContent(fixture.Content)
	}
	kind := fixture.Kind
	if kind == "" {
		kind = "markdown"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.UpsertDocument(Document{
		ID:          fixture.DocID,
		Scope:       fixture.Scope,
		ScopeID:     fixture.ScopeID,
		Kind:        kind,
		Title:       fixture.Title,
		SourcePath:  "knowledge/" + string(fixture.Scope) + "/" + fixture.ScopeID + "/" + fixture.Title,
		ContentHash: contentHash,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("upsert doc: %v", err)
	}

	if err := store.ReplaceDocumentChunks(fixture.DocID, []Chunk{{
		ID:         fixture.ChunkID,
		DocumentID: fixture.DocID,
		ChunkIndex: 0,
		Content:    fixture.Content,
		TokenCount: 1,
		Embedding:  fixture.Embedding,
		UpdatedAt:  now,
	}}); err != nil {
		t.Fatalf("replace chunks: %v", err)
	}
}

type stubEmbeddingProvider struct {
	vectors map[string][]float32
}

func (p *stubEmbeddingProvider) Embed(texts []string) ([][]float32, error) {
	result := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vector, ok := p.vectors[text]
		if !ok {
			result = append(result, []float32{0, 0})
			continue
		}
		copied := make([]float32, len(vector))
		copy(copied, vector)
		result = append(result, copied)
	}
	return result, nil
}
