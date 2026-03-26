package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"trace2offer/backend/agent/tool"
)

var ErrResumeSourceUnavailable = errors.New("resume source store is unavailable")
var ErrResumeSourceNotFound = errors.New("resume source not found")

const (
	resumeCurrentFileName = "current.md"
	resumeMetaFileName    = "current.meta.json"
)

type resumeSourceMetadata struct {
	SourceName  string `json:"source_name"`
	ContentType string `json:"content_type"`
	ImportedAt  string `json:"imported_at"`
	TextLength  int    `json:"text_length"`
}

// ResumeSourceWriteResult contains persisted resume source details.
type ResumeSourceWriteResult struct {
	Path       string
	SourceName string
	ImportedAt string
	TotalChars int
	Truncated  bool
	Content    string
}

// FileResumeSourceStore persists canonical resume markdown and metadata.
type FileResumeSourceStore struct {
	dir      string
	mdPath   string
	metaPath string
	mu       sync.RWMutex
}

func NewFileResumeSourceStore(dir string) (*FileResumeSourceStore, error) {
	cleaned := strings.TrimSpace(dir)
	if cleaned == "" {
		return nil, fmt.Errorf("resume data dir is required")
	}
	if err := os.MkdirAll(cleaned, 0o755); err != nil {
		return nil, fmt.Errorf("create resume data dir: %w", err)
	}
	return &FileResumeSourceStore{
		dir:      cleaned,
		mdPath:   filepath.Join(cleaned, resumeCurrentFileName),
		metaPath: filepath.Join(cleaned, resumeMetaFileName),
	}, nil
}

func (s *FileResumeSourceStore) CurrentPath() string {
	if s == nil {
		return ""
	}
	return s.mdPath
}

func (s *FileResumeSourceStore) Save(sourceName string, contentType string, rawText string) (ResumeSourceWriteResult, error) {
	if s == nil {
		return ResumeSourceWriteResult{}, ErrResumeSourceUnavailable
	}

	cleaned := normalizeResumeText(rawText)
	if cleaned == "" {
		return ResumeSourceWriteResult{}, errResumeTextEmpty
	}

	sourceName = strings.TrimSpace(sourceName)
	if sourceName == "" {
		sourceName = "unknown"
	}

	meta := resumeSourceMetadata{
		SourceName:  sourceName,
		ContentType: strings.TrimSpace(contentType),
		ImportedAt:  time.Now().UTC().Format(time.RFC3339),
		TextLength:  utf8.RuneCountInString(cleaned),
	}

	markdown := buildCanonicalResumeMarkdown(meta, cleaned)
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return ResumeSourceWriteResult{}, fmt.Errorf("encode resume metadata: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := writeFileAtomically(s.mdPath, []byte(markdown)); err != nil {
		return ResumeSourceWriteResult{}, err
	}
	if err := writeFileAtomically(s.metaPath, metaBytes); err != nil {
		return ResumeSourceWriteResult{}, err
	}

	return ResumeSourceWriteResult{
		Path:       s.mdPath,
		SourceName: meta.SourceName,
		ImportedAt: meta.ImportedAt,
		TotalChars: meta.TextLength,
		Truncated:  false,
		Content:    cleaned,
	}, nil
}

func (s *FileResumeSourceStore) Read(maxChars int) (tool.ResumeContent, error) {
	if s == nil {
		return tool.ResumeContent{}, ErrResumeSourceUnavailable
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	rawMarkdown, err := os.ReadFile(s.mdPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return tool.ResumeContent{}, ErrResumeSourceNotFound
		}
		return tool.ResumeContent{}, fmt.Errorf("read resume markdown: %w", err)
	}

	content := extractCanonicalResumeContent(string(rawMarkdown))
	if strings.TrimSpace(content) == "" {
		return tool.ResumeContent{}, ErrResumeSourceNotFound
	}

	meta, err := s.loadMetadata()
	if err != nil {
		return tool.ResumeContent{}, err
	}

	totalChars := utf8.RuneCountInString(content)
	if meta.TextLength > 0 {
		totalChars = meta.TextLength
	}

	limit := maxChars
	if limit <= 0 {
		limit = totalChars
	}
	truncated := false
	returned := content
	if utf8.RuneCountInString(content) > limit {
		returned = truncateByRunes(content, limit)
		truncated = true
	}

	return tool.ResumeContent{
		Content:       returned,
		TotalChars:    totalChars,
		ReturnedChars: utf8.RuneCountInString(returned),
		Truncated:     truncated,
		SourceName:    meta.SourceName,
		ImportedAt:    meta.ImportedAt,
	}, nil
}

func (s *FileResumeSourceStore) loadMetadata() (resumeSourceMetadata, error) {
	payload, err := os.ReadFile(s.metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return resumeSourceMetadata{}, nil
		}
		return resumeSourceMetadata{}, fmt.Errorf("read resume metadata: %w", err)
	}
	if len(payload) == 0 {
		return resumeSourceMetadata{}, nil
	}

	var meta resumeSourceMetadata
	if err := json.Unmarshal(payload, &meta); err != nil {
		return resumeSourceMetadata{}, fmt.Errorf("decode resume metadata: %w", err)
	}
	meta.SourceName = strings.TrimSpace(meta.SourceName)
	meta.ContentType = strings.TrimSpace(meta.ContentType)
	meta.ImportedAt = strings.TrimSpace(meta.ImportedAt)
	return meta, nil
}

func buildCanonicalResumeMarkdown(meta resumeSourceMetadata, content string) string {
	builder := strings.Builder{}
	builder.WriteString("# Resume\n\n")
	builder.WriteString("- source_name: ")
	builder.WriteString(meta.SourceName)
	builder.WriteString("\n")
	builder.WriteString("- content_type: ")
	builder.WriteString(meta.ContentType)
	builder.WriteString("\n")
	builder.WriteString("- imported_at: ")
	builder.WriteString(meta.ImportedAt)
	builder.WriteString("\n")
	builder.WriteString("- text_length: ")
	builder.WriteString(fmt.Sprintf("%d", meta.TextLength))
	builder.WriteString("\n\n")
	builder.WriteString("## Content\n\n")
	builder.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	return builder.String()
}

func extractCanonicalResumeContent(markdown string) string {
	const marker = "\n## Content\n\n"
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	idx := strings.Index(normalized, marker)
	if idx < 0 {
		return strings.TrimSpace(normalized)
	}
	content := normalized[idx+len(marker):]
	return strings.TrimRight(content, "\n")
}

func writeFileAtomically(path string, payload []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp file %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace file %s: %w", filepath.Base(path), err)
	}
	return nil
}
