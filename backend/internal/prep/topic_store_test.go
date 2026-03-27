package prep

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestTopicStoreCRUDRoundtrip(t *testing.T) {
	t.Parallel()

	store, err := NewTopicStore(filepath.Join(t.TempDir(), "topic_catalog.json"))
	if err != nil {
		t.Fatalf("new topic store: %v", err)
	}

	if len(store.List()) != 0 {
		t.Fatalf("expected empty topic list on fresh store")
	}

	created, err := store.Create(TopicCreateInput{
		Key:         "RAG",
		Name:        "RAG",
		Description: "检索增强生成",
	})
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if created.Key != "rag" {
		t.Fatalf("expected normalized key rag, got %q", created.Key)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatalf("expected created_at/updated_at populated")
	}

	_, err = store.Create(TopicCreateInput{
		Key:  "rag",
		Name: "Duplicate",
	})
	if !errors.Is(err, ErrTopicAlreadyExists) {
		t.Fatalf("expected ErrTopicAlreadyExists, got %v", err)
	}

	updated, found, err := store.Update("rag", TopicPatchInput{
		Name:        stringPtr("RAG Core"),
		Description: stringPtr("RAG 核心原理"),
	})
	if err != nil {
		t.Fatalf("update topic: %v", err)
	}
	if !found {
		t.Fatalf("expected topic found for update")
	}
	if updated.Name != "RAG Core" {
		t.Fatalf("expected name updated, got %q", updated.Name)
	}
	if updated.Description != "RAG 核心原理" {
		t.Fatalf("expected description updated, got %q", updated.Description)
	}

	listed := store.List()
	if len(listed) != 1 {
		t.Fatalf("expected 1 topic after update, got %d", len(listed))
	}
	if listed[0].Key != "rag" {
		t.Fatalf("expected topic key rag in list, got %q", listed[0].Key)
	}

	deleted, err := store.Delete("rag")
	if err != nil {
		t.Fatalf("delete topic: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete success")
	}
	if len(store.List()) != 0 {
		t.Fatalf("expected empty list after delete")
	}
}

func TestTopicStoreValidation(t *testing.T) {
	t.Parallel()

	store, err := NewTopicStore(filepath.Join(t.TempDir(), "topic_catalog.json"))
	if err != nil {
		t.Fatalf("new topic store: %v", err)
	}

	_, err = store.Create(TopicCreateInput{
		Key:  "bad key",
		Name: "Invalid",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for invalid key, got %v", err)
	}

	_, _, err = store.Update("missing", TopicPatchInput{})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for empty patch, got %v", err)
	}
}

func stringPtr(value string) *string {
	v := value
	return &v
}
