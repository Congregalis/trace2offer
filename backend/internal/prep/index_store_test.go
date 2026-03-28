package prep

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestIndexStoreInitializesSchema(t *testing.T) {
	t.Parallel()

	store := newTestIndexStore(t)

	for _, table := range []string{"documents", "chunks", "index_runs", "index_meta"} {
		var found string
		err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found)
		if err != nil {
			t.Fatalf("table %s not found: %v", table, err)
		}
		if found != table {
			t.Fatalf("expected table %s, got %s", table, found)
		}
	}
}

func TestIndexStoreMetaRoundTrip(t *testing.T) {
	t.Parallel()

	store := newTestIndexStore(t)

	if err := store.UpsertMeta("index_fingerprint", "v1"); err != nil {
		t.Fatalf("upsert meta: %v", err)
	}
	value, found, err := store.GetMeta("index_fingerprint")
	if err != nil {
		t.Fatalf("get meta: %v", err)
	}
	if !found {
		t.Fatal("expected meta found")
	}
	if value != "v1" {
		t.Fatalf("expected meta value v1, got %q", value)
	}

	if err := store.UpsertMeta("index_fingerprint", "v2"); err != nil {
		t.Fatalf("upsert meta update: %v", err)
	}
	value, found, err = store.GetMeta("index_fingerprint")
	if err != nil {
		t.Fatalf("get meta after update: %v", err)
	}
	if !found || value != "v2" {
		t.Fatalf("expected meta value v2, got found=%v value=%q", found, value)
	}
}

func TestIndexStoreStatusCountsAndLastRun(t *testing.T) {
	t.Parallel()

	store := newTestIndexStore(t)
	now := time.Now().UTC().Format(time.RFC3339)

	if err := store.UpsertDocument(Document{
		ID:          "doc_1",
		Scope:       ScopeTopics,
		ScopeID:     "system-design",
		Kind:        "knowledge",
		Title:       "CAP theorem",
		SourcePath:  "topics/system-design/cap.md",
		ContentHash: "hash-1",
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("upsert document: %v", err)
	}

	if err := store.ReplaceDocumentChunks("doc_1", []Chunk{
		{
			ID:         "chunk_1",
			DocumentID: "doc_1",
			ChunkIndex: 0,
			Content:    "CAP theorem says consistency availability partition tolerance tradeoffs.",
			TokenCount: 12,
			Embedding:  []float32{0.1, 0.2, 0.3},
			UpdatedAt:  now,
		},
		{
			ID:         "chunk_2",
			DocumentID: "doc_1",
			ChunkIndex: 1,
			Content:    "In distributed systems two of three properties are typically achieved.",
			TokenCount: 13,
			Embedding:  []float32{0.4, 0.5, 0.6},
			UpdatedAt:  now,
		},
	}); err != nil {
		t.Fatalf("replace chunks: %v", err)
	}

	if err := store.UpsertIndexRun(IndexRun{
		ID:                 "run_1",
		StartedAt:          now,
		CompletedAt:        now,
		Status:             IndexRunStatusCompleted,
		DocumentsProcessed: 1,
		ChunksCreated:      2,
	}); err != nil {
		t.Fatalf("upsert index run: %v", err)
	}

	status, err := store.GetStatus("huggingface", "BAAI/bge-m3")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status.DocumentCount != 1 {
		t.Fatalf("expected document count 1, got %d", status.DocumentCount)
	}
	if status.ChunkCount != 2 {
		t.Fatalf("expected chunk count 2, got %d", status.ChunkCount)
	}
	if status.LastIndexedAt == "" {
		t.Fatalf("expected last indexed at")
	}
	if status.LastIndexStatus != IndexRunStatusCompleted {
		t.Fatalf("expected status %q, got %q", IndexRunStatusCompleted, status.LastIndexStatus)
	}
}

func TestIndexStoreCreateGetAndSearchSimilar(t *testing.T) {
	t.Parallel()

	store := newTestIndexStore(t)
	now := time.Now().UTC().Format(time.RFC3339)

	err := store.CreateDocument(Document{
		ID:          "doc_search",
		Scope:       ScopeTopics,
		ScopeID:     "search",
		Kind:        "knowledge",
		Title:       "vector.md",
		SourcePath:  "knowledge/topics/search/vector.md",
		ContentHash: "hash-search",
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	document, found, err := store.GetDocument("doc_search")
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if !found {
		t.Fatal("expected document found")
	}
	if document.Title != "vector.md" {
		t.Fatalf("unexpected document title: %q", document.Title)
	}

	if err := store.ReplaceDocumentChunks("doc_search", []Chunk{
		{
			ID:         "chunk_a",
			DocumentID: "doc_search",
			ChunkIndex: 0,
			Content:    "alpha",
			TokenCount: 1,
			Embedding:  []float32{1, 0},
			UpdatedAt:  now,
		},
		{
			ID:         "chunk_b",
			DocumentID: "doc_search",
			ChunkIndex: 1,
			Content:    "beta",
			TokenCount: 1,
			Embedding:  []float32{0, 1},
			UpdatedAt:  now,
		},
	}); err != nil {
		t.Fatalf("replace chunks: %v", err)
	}

	results, err := store.SearchSimilar([]float32{1, 0}, 1)
	if err != nil {
		t.Fatalf("search similar: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected single result, got %d", len(results))
	}
	if results[0].Chunk.ID != "chunk_a" {
		t.Fatalf("expected top chunk_a, got %s", results[0].Chunk.ID)
	}
}

func TestIndexStoreReplaceDocumentChunksRollsBackOnError(t *testing.T) {
	t.Parallel()

	store := newTestIndexStore(t)
	now := time.Now().UTC().Format(time.RFC3339)

	if err := store.UpsertDocument(Document{
		ID:          "doc_tx",
		Scope:       ScopeTopics,
		ScopeID:     "tx",
		Kind:        "markdown",
		Title:       "tx.md",
		SourcePath:  "knowledge/topics/tx/tx.md",
		ContentHash: "hash-tx",
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("upsert document: %v", err)
	}

	if err := store.ReplaceDocumentChunks("doc_tx", []Chunk{{
		ID:         "chunk_existing",
		DocumentID: "doc_tx",
		ChunkIndex: 0,
		Content:    "existing",
		TokenCount: 1,
		Embedding:  []float32{1, 0},
		UpdatedAt:  now,
	}}); err != nil {
		t.Fatalf("replace chunks initial: %v", err)
	}

	err := store.ReplaceDocumentChunks("doc_tx", []Chunk{
		{
			ID:         "chunk_new_ok",
			DocumentID: "doc_tx",
			ChunkIndex: 0,
			Content:    "new",
			TokenCount: 1,
			Embedding:  []float32{1, 0},
			UpdatedAt:  now,
		},
		{
			ID:         "chunk_new_bad",
			DocumentID: "doc_tx",
			ChunkIndex: 1,
			Content:    "bad",
			TokenCount: 1,
			Embedding:  nil,
			UpdatedAt:  now,
		},
	})
	if err == nil {
		t.Fatal("expected replace chunks to fail")
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE document_id = ?`, "doc_tx").Scan(&count); err != nil {
		t.Fatalf("count chunks after rollback: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected previous chunk preserved after rollback, got %d", count)
	}
}

func newTestIndexStore(t *testing.T) *IndexStore {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	store, err := NewIndexStore(dsn)
	if err != nil {
		t.Fatalf("new index store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if store == nil || store.db == nil {
		t.Fatalf("expected non-nil index store db")
	}
	if err := pingDB(store.db); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	return store
}

func pingDB(db *sql.DB) error {
	return db.Ping()
}
