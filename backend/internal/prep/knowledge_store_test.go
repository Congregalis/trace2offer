package prep

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestKnowledgeStoreCRUDRoundtrip(t *testing.T) {
	t.Parallel()

	store, err := NewKnowledgeStore(filepath.Join(t.TempDir(), "knowledge"))
	if err != nil {
		t.Fatalf("new knowledge store: %v", err)
	}

	documents, err := store.List("topics", "rag")
	if err != nil {
		t.Fatalf("list empty docs: %v", err)
	}
	if len(documents) != 0 {
		t.Fatalf("expected no docs on fresh scope")
	}

	created, err := store.Create("topics", "rag", KnowledgeDocumentCreateInput{
		Filename: "overview",
		Content:  "# RAG\n\nchunking",
	})
	if err != nil {
		t.Fatalf("create knowledge doc: %v", err)
	}
	if created.Filename != "overview.md" {
		t.Fatalf("expected filename overview.md, got %q", created.Filename)
	}
	if created.Scope != ScopeTopics || created.ScopeID != "rag" {
		t.Fatalf("unexpected scope in created document: scope=%q scope_id=%q", created.Scope, created.ScopeID)
	}

	_, err = store.Create("topics", "rag", KnowledgeDocumentCreateInput{
		Filename: "overview.md",
		Content:  "duplicate",
	})
	if !errors.Is(err, ErrDocumentAlreadyExists) {
		t.Fatalf("expected ErrDocumentAlreadyExists, got %v", err)
	}

	listed, err := store.List("topics", "rag")
	if err != nil {
		t.Fatalf("list docs after create: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(listed))
	}
	if listed[0].Content != "# RAG\n\nchunking" {
		t.Fatalf("expected content kept, got %q", listed[0].Content)
	}

	updated, found, err := store.Update("topics", "rag", "overview.md", KnowledgeDocumentUpdateInput{
		Content: "updated",
	})
	if err != nil {
		t.Fatalf("update doc: %v", err)
	}
	if !found {
		t.Fatalf("expected document found for update")
	}
	if updated.Content != "updated" {
		t.Fatalf("expected updated content, got %q", updated.Content)
	}

	_, found, err = store.Update("topics", "rag", "missing.md", KnowledgeDocumentUpdateInput{Content: "x"})
	if err != nil {
		t.Fatalf("update missing doc should not error, got %v", err)
	}
	if found {
		t.Fatalf("expected missing doc update found=false")
	}

	deleted, err := store.Delete("topics", "rag", "overview.md")
	if err != nil {
		t.Fatalf("delete doc: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete success")
	}

	listed, err = store.List("topics", "rag")
	if err != nil {
		t.Fatalf("list docs after delete: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected empty docs after delete, got %d", len(listed))
	}
}

func TestKnowledgeStoreValidation(t *testing.T) {
	t.Parallel()

	store, err := NewKnowledgeStore(filepath.Join(t.TempDir(), "knowledge"))
	if err != nil {
		t.Fatalf("new knowledge store: %v", err)
	}

	_, err = store.List("invalid_scope", "x")
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid scope, got %v", err)
	}

	_, err = store.Create("topics", "bad/scope", KnowledgeDocumentCreateInput{
		Filename: "doc",
		Content:  "",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid scope_id, got %v", err)
	}

	_, err = store.Create("topics", "rag", KnowledgeDocumentCreateInput{
		Filename: "../escape",
		Content:  "",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid filename, got %v", err)
	}
}
