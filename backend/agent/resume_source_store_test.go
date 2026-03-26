package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileResumeSourceStoreSaveAndRead(t *testing.T) {
	t.Parallel()

	store, err := NewFileResumeSourceStore(t.TempDir())
	if err != nil {
		t.Fatalf("new resume source store error: %v", err)
	}

	result, err := store.Save("resume.pdf", "application/pdf", " Go Engineer \n\n  Built platform  ")
	if err != nil {
		t.Fatalf("save resume source error: %v", err)
	}
	if result.Path == "" {
		t.Fatal("expected resume path not empty")
	}
	if result.TotalChars == 0 {
		t.Fatal("expected total chars > 0")
	}

	markdownBytes, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read markdown file error: %v", err)
	}
	markdown := string(markdownBytes)
	if !strings.Contains(markdown, "# Resume") || !strings.Contains(markdown, "## Content") {
		t.Fatalf("unexpected markdown format: %q", markdown)
	}
	if !strings.HasSuffix(markdown, "\n") {
		t.Fatalf("expected markdown with trailing newline, got %q", markdown)
	}

	metaPath := filepath.Join(filepath.Dir(result.Path), resumeMetaFileName)
	if _, err := os.Stat(metaPath); err != nil {
		t.Fatalf("expected metadata file exists: %v", err)
	}

	read, err := store.Read(12000)
	if err != nil {
		t.Fatalf("read resume source error: %v", err)
	}
	if read.Content != "Go Engineer\n\nBuilt platform" {
		t.Fatalf("unexpected resume content: %q", read.Content)
	}
	if read.TotalChars != result.TotalChars {
		t.Fatalf("expected total chars=%d, got %d", result.TotalChars, read.TotalChars)
	}
}

func TestFileResumeSourceStoreReadNotFound(t *testing.T) {
	t.Parallel()

	store, err := NewFileResumeSourceStore(t.TempDir())
	if err != nil {
		t.Fatalf("new resume source store error: %v", err)
	}

	_, err = store.Read(12000)
	if !errors.Is(err, ErrResumeSourceNotFound) {
		t.Fatalf("expected ErrResumeSourceNotFound, got %v", err)
	}
}

func TestFileResumeSourceStoreReadTruncation(t *testing.T) {
	t.Parallel()

	store, err := NewFileResumeSourceStore(t.TempDir())
	if err != nil {
		t.Fatalf("new resume source store error: %v", err)
	}

	if _, err := store.Save("resume.txt", "text/plain", "1234567890"); err != nil {
		t.Fatalf("save resume source error: %v", err)
	}

	read, err := store.Read(4)
	if err != nil {
		t.Fatalf("read resume source error: %v", err)
	}
	if !read.Truncated {
		t.Fatalf("expected truncated=true, got false")
	}
	if read.Content != "1234" {
		t.Fatalf("expected truncated content 1234, got %q", read.Content)
	}
	if read.ReturnedChars != 4 {
		t.Fatalf("expected returned chars=4, got %d", read.ReturnedChars)
	}
}
