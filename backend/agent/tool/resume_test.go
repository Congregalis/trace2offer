package tool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestResumeGetToolDefaultAndClamp(t *testing.T) {
	t.Parallel()

	provider := &stubResumeSourceProvider{
		content: ResumeContent{
			Content:       "resume content",
			TotalChars:    50000,
			ReturnedChars: 12000,
			Truncated:     true,
			SourceName:    "resume.pdf",
			ImportedAt:    "2026-03-26T00:00:00Z",
		},
	}
	tool := &resumeGetTool{provider: provider}

	output, err := tool.Run(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("run resume_get default error: %v", err)
	}
	if provider.lastMaxChars != defaultResumeGetMaxChars {
		t.Fatalf("expected default max chars=%d, got %d", defaultResumeGetMaxChars, provider.lastMaxChars)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output error: %v", err)
	}
	if payload["source_name"] != "resume.pdf" {
		t.Fatalf("expected source_name resume.pdf, got %#v", payload["source_name"])
	}

	if _, err := tool.Run(context.Background(), json.RawMessage(`{"max_chars":12}`)); err != nil {
		t.Fatalf("run resume_get with small max_chars error: %v", err)
	}
	if provider.lastMaxChars != minResumeGetMaxChars {
		t.Fatalf("expected clamped min max chars=%d, got %d", minResumeGetMaxChars, provider.lastMaxChars)
	}

	if _, err := tool.Run(context.Background(), json.RawMessage(`{"max_chars":700000}`)); err != nil {
		t.Fatalf("run resume_get with large max_chars error: %v", err)
	}
	if provider.lastMaxChars != maxResumeGetMaxChars {
		t.Fatalf("expected clamped max max chars=%d, got %d", maxResumeGetMaxChars, provider.lastMaxChars)
	}
}

func TestResumeGetToolProviderError(t *testing.T) {
	t.Parallel()

	provider := &stubResumeSourceProvider{err: errors.New("resume source not found")}
	tool := &resumeGetTool{provider: provider}
	if _, err := tool.Run(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected provider error, got nil")
	}
}

type stubResumeSourceProvider struct {
	content      ResumeContent
	err          error
	lastMaxChars int
}

func (s *stubResumeSourceProvider) Read(maxChars int) (ResumeContent, error) {
	s.lastMaxChars = maxChars
	if s.err != nil {
		return ResumeContent{}, s.err
	}
	return s.content, nil
}
