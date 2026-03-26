package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	defaultResumeGetMaxChars = 30000
	minResumeGetMaxChars     = 1000
	maxResumeGetMaxChars     = 200000
)

// ResumeContent is the canonical resume source payload.
type ResumeContent struct {
	Content       string
	TotalChars    int
	ReturnedChars int
	Truncated     bool
	SourceName    string
	ImportedAt    string
}

// ResumeSourceProvider provides canonical resume content for agent tools.
type ResumeSourceProvider interface {
	Read(maxChars int) (ResumeContent, error)
}

// NewResumeTools builds resume-related tools.
func NewResumeTools(provider ResumeSourceProvider) []Tool {
	if provider == nil {
		return nil
	}
	return []Tool{
		&resumeGetTool{provider: provider},
	}
}

type resumeGetTool struct {
	provider ResumeSourceProvider
}

type resumeGetInput struct {
	MaxChars int `json:"max_chars"`
}

func (t *resumeGetTool) Definition() Definition {
	return Definition{
		Name:        "resume_get",
		Description: "读取用户完整简历事实源，默认返回前 30000 字符，可通过 max_chars 控制长度。",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"max_chars": map[string]any{
					"type":        "integer",
					"description": "返回字符上限，范围 1000-200000，默认 30000",
				},
			},
		},
	}
}

func (t *resumeGetTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if t == nil || t.provider == nil {
		return "", fmt.Errorf("resume source provider is unavailable")
	}

	var args resumeGetInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	maxChars := args.MaxChars
	if maxChars <= 0 {
		maxChars = defaultResumeGetMaxChars
	}
	if maxChars < minResumeGetMaxChars {
		maxChars = minResumeGetMaxChars
	}
	if maxChars > maxResumeGetMaxChars {
		maxChars = maxResumeGetMaxChars
	}

	content, err := t.provider.Read(maxChars)
	if err != nil {
		return "", err
	}

	return marshalOutput(map[string]any{
		"content":        content.Content,
		"total_chars":    content.TotalChars,
		"returned_chars": content.ReturnedChars,
		"truncated":      content.Truncated,
		"source_name":    content.SourceName,
		"imported_at":    content.ImportedAt,
	})
}
