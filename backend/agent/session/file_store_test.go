package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreSaveLoadPerSessionFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store, err := NewFileStore(filepath.Join(tmpDir, "sessions"))
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	sessionA := Session{
		ID: "session_a",
		Messages: []Message{
			{Role: "user", Content: "hello", CreatedAt: "2026-03-13T12:00:00Z"},
		},
		UpdatedAt: "2026-03-13T12:00:00Z",
	}
	sessionB := Session{
		ID: "session_b",
		Messages: []Message{
			{Role: "assistant", Content: "world", CreatedAt: "2026-03-13T12:00:05Z"},
		},
		UpdatedAt: "2026-03-13T12:00:05Z",
	}

	if err := store.Save(sessionA); err != nil {
		t.Fatalf("save session_a: %v", err)
	}
	if err := store.Save(sessionB); err != nil {
		t.Fatalf("save session_b: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "sessions", "session_a.json")); err != nil {
		t.Fatalf("stat session_a file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "sessions", "session_b.json")); err != nil {
		t.Fatalf("stat session_b file: %v", err)
	}

	gotA, ok, err := store.Load("session_a")
	if err != nil {
		t.Fatalf("load session_a: %v", err)
	}
	if !ok {
		t.Fatal("session_a should exist")
	}
	if gotA.ID != "session_a" || len(gotA.Messages) != 1 || gotA.Messages[0].Content != "hello" {
		t.Fatalf("unexpected session_a payload: %+v", gotA)
	}

	gotB, ok, err := store.Load("session_b")
	if err != nil {
		t.Fatalf("load session_b: %v", err)
	}
	if !ok {
		t.Fatal("session_b should exist")
	}
	if gotB.ID != "session_b" || len(gotB.Messages) != 1 || gotB.Messages[0].Content != "world" {
		t.Fatalf("unexpected session_b payload: %+v", gotB)
	}
}

func TestFileStoreMigratesLegacyAggregateFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	legacyPath := filepath.Join(tmpDir, "agent_sessions.json")

	legacy := fileDump{
		Sessions: []Session{
			{
				ID: "session_a",
				Messages: []Message{
					{Role: "user", Content: "hello", CreatedAt: "2026-03-13T12:00:00Z"},
				},
				UpdatedAt: "2026-03-13T12:00:00Z",
			},
			{
				ID: "session_b",
				Messages: []Message{
					{Role: "assistant", Content: "world", CreatedAt: "2026-03-13T12:00:05Z"},
				},
				UpdatedAt: "2026-03-13T12:00:05Z",
			},
		},
	}
	payload, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy payload: %v", err)
	}
	if err := os.WriteFile(legacyPath, payload, 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	store, err := NewFileStore(legacyPath)
	if err != nil {
		t.Fatalf("new file store from legacy path: %v", err)
	}

	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy file should be archived, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "agent_sessions.json.bak")); err != nil {
		t.Fatalf("legacy backup file not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "sessions", "session_a.json")); err != nil {
		t.Fatalf("migrated session_a file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "sessions", "session_b.json")); err != nil {
		t.Fatalf("migrated session_b file missing: %v", err)
	}

	got, ok, err := store.Load("session_b")
	if err != nil {
		t.Fatalf("load migrated session_b: %v", err)
	}
	if !ok {
		t.Fatal("migrated session_b should exist")
	}
	if got.ID != "session_b" || len(got.Messages) != 1 || got.Messages[0].Content != "world" {
		t.Fatalf("unexpected migrated session_b payload: %+v", got)
	}
}

func TestFileStoreMigratesSiblingLegacyFileWhenUsingSessionsDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	legacyPath := filepath.Join(tmpDir, "agent_sessions.json")
	sessionsDir := filepath.Join(tmpDir, "sessions")

	legacy := fileDump{
		Sessions: []Session{
			{
				ID: "session_x",
				Messages: []Message{
					{Role: "user", Content: "legacy", CreatedAt: "2026-03-13T12:00:00Z"},
				},
				UpdatedAt: "2026-03-13T12:00:00Z",
			},
		},
	}
	payload, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy payload: %v", err)
	}
	if err := os.WriteFile(legacyPath, payload, 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	store, err := NewFileStore(sessionsDir)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "agent_sessions.json.bak")); err != nil {
		t.Fatalf("legacy backup file not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sessionsDir, "session_x.json")); err != nil {
		t.Fatalf("migrated session_x file missing: %v", err)
	}

	got, ok, err := store.Load("session_x")
	if err != nil {
		t.Fatalf("load migrated session_x: %v", err)
	}
	if !ok {
		t.Fatal("migrated session_x should exist")
	}
	if got.ID != "session_x" || len(got.Messages) != 1 || got.Messages[0].Content != "legacy" {
		t.Fatalf("unexpected migrated session_x payload: %+v", got)
	}
}

func TestFileStoreList(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	store, err := NewFileStore(filepath.Join(tmpDir, "sessions"))
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	if err := store.Save(Session{ID: "session_a", UpdatedAt: "2026-03-13T12:00:00Z"}); err != nil {
		t.Fatalf("save session_a: %v", err)
	}
	if err := store.Save(Session{ID: "session_b", UpdatedAt: "2026-03-14T12:00:00Z"}); err != nil {
		t.Fatalf("save session_b: %v", err)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(items))
	}

	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	if !ids["session_a"] || !ids["session_b"] {
		t.Fatalf("unexpected list ids: %+v", ids)
	}
}
