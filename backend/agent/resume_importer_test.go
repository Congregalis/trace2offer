package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseResumeProfileOutput(t *testing.T) {
	t.Parallel()

	raw := "```json\n{\"name\":\"Carol\",\"total_years\":5,\"core_skills\":[\"Go\",\"System Design\"],\"preferred_roles\":\"Staff Engineer, Tech Lead\"}\n```"
	parsed, err := parseResumeProfileOutput(raw)
	if err != nil {
		t.Fatalf("parse resume output error: %v", err)
	}
	if parsed.Name != "Carol" {
		t.Fatalf("expected name Carol, got %q", parsed.Name)
	}
	if parsed.TotalYears != 5 {
		t.Fatalf("expected total years 5, got %v", parsed.TotalYears)
	}
	if len(parsed.CoreSkills) != 2 {
		t.Fatalf("expected 2 core skills, got %#v", parsed.CoreSkills)
	}
	if len(parsed.PreferredRoles) != 2 {
		t.Fatalf("expected split preferred roles, got %#v", parsed.PreferredRoles)
	}
}

func TestExtractResumeTextPlainText(t *testing.T) {
	t.Parallel()

	text, err := extractResumeText("resume.txt", "text/plain", []byte("Go Engineer\n\n5 years\tbackend"))
	if err != nil {
		t.Fatalf("extract plain text error: %v", err)
	}
	if text == "" {
		t.Fatal("expected extracted text")
	}
}

func TestNewResumeExtractConfig(t *testing.T) {
	t.Parallel()

	config, err := newResumeExtractConfig("docling", "", 30)
	if err != nil {
		t.Fatalf("new resume extract config error: %v", err)
	}
	if config.PDFExtractor != resumePDFExtractorDocling {
		t.Fatalf("expected docling extractor, got %q", config.PDFExtractor)
	}
	if config.DoclingPythonBin != defaultDoclingPythonBin {
		t.Fatalf("expected default python bin, got %q", config.DoclingPythonBin)
	}
	if config.DoclingTimeout != 30*time.Second {
		t.Fatalf("expected timeout 30s, got %s", config.DoclingTimeout)
	}

	if _, err := newResumeExtractConfig("unknown", "", 30); err == nil {
		t.Fatal("expected invalid extractor error")
	}
}

func TestExtractResumeTextWithDoclingMissingPython(t *testing.T) {
	t.Parallel()

	_, err := extractResumeTextWithConfig(
		context.Background(),
		"resume.pdf",
		"application/pdf",
		[]byte("%PDF-1.4 dummy"),
		ResumeExtractConfig{
			PDFExtractor:     resumePDFExtractorDocling,
			DoclingPythonBin: "/path/not-exists-python",
			DoclingTimeout:   5 * time.Second,
		},
	)
	if err == nil {
		t.Fatal("expected docling extraction error")
	}
	if !IsResumeImportError(err) {
		t.Fatalf("expected resume import error, got %T %v", err, err)
	}
	if !strings.Contains(err.Error(), "Python") {
		t.Fatalf("expected python-not-found hint, got %q", err.Error())
	}
}
