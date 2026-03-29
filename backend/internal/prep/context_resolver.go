package prep

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"trace2offer/backend/internal/model"
)

var ErrContextResolverUnavailable = errors.New("prep context resolver is unavailable")

var companySlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type ContextResolver struct {
	knowledgeStore *KnowledgeStore
	resumePath     string
}

func NewContextResolver(prepDataDir string, knowledgeStore *KnowledgeStore) *ContextResolver {
	rootDir := filepath.Dir(filepath.Clean(strings.TrimSpace(prepDataDir)))
	return &ContextResolver{
		knowledgeStore: knowledgeStore,
		resumePath:     filepath.Join(rootDir, "resume", "current.md"),
	}
}

func (r *ContextResolver) Resolve(lead model.Lead) (LeadContextPreview, error) {
	if r == nil || r.knowledgeStore == nil {
		return LeadContextPreview{}, ErrContextResolverUnavailable
	}

	leadID := strings.TrimSpace(lead.ID)
	if leadID == "" {
		return LeadContextPreview{}, &ValidationError{Field: "lead_id", Message: "lead_id is required"}
	}

	preview := LeadContextPreview{
		LeadID:   leadID,
		Company:  strings.TrimSpace(lead.Company),
		Position: strings.TrimSpace(lead.Position),
		Sources:  []ContextSource{},
	}

	hasResume, err := r.hasResume()
	if err != nil {
		return LeadContextPreview{}, err
	}
	preview.HasResume = hasResume

	if strings.TrimSpace(lead.JDText) != "" {
		preview.Sources = append(preview.Sources, ContextSource{
			Scope: "lead",
			Kind:  "jd_text",
			Title: "JD 原文",
		})
	}
	if preview.HasResume {
		preview.Sources = append(preview.Sources, ContextSource{
			Scope: "resume",
			Kind:  "markdown",
			Title: "resume/current.md",
		})
	}

	librarySources, err := r.collectLibrarySources()
	if err != nil {
		return LeadContextPreview{}, err
	}
	preview.Sources = append(preview.Sources, librarySources...)

	return preview, nil
}

func (r *ContextResolver) collectLibrarySources() ([]ContextSource, error) {
	documents, err := r.knowledgeStore.List(string(KnowledgeLibraryScope), KnowledgeLibraryScopeID)
	if err != nil {
		return nil, err
	}
	sources := make([]ContextSource, 0, len(documents))
	for _, document := range documents {
		sources = append(sources, ContextSource{
			Scope: "library",
			Kind:  "markdown",
			Title: document.Filename,
		})
	}
	return sources, nil
}

func (r *ContextResolver) hasResume() (bool, error) {
	if strings.TrimSpace(r.resumePath) == "" {
		return false, nil
	}
	content, err := os.ReadFile(r.resumePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read prep resume source: %w", err)
	}
	return strings.TrimSpace(string(content)) != "", nil
}

func normalizeCompanySlug(company string) string {
	slug := strings.ToLower(strings.TrimSpace(company))
	slug = companySlugPattern.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

func NormalizeCompanySlug(company string) string {
	return normalizeCompanySlug(company)
}

func (r *ContextResolver) BuildPromptCandidateProfile(lead model.Lead) string {
	if r == nil {
		return buildLeadSummaryForPrompt(lead)
	}

	sections := []string{buildLeadSummaryForPrompt(lead)}
	if resumeText, ok := r.readResumeText(); ok {
		sections = append(sections, "Resume:\n"+resumeText)
	}
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func (r *ContextResolver) readResumeText() (string, bool) {
	if r == nil {
		return "", false
	}
	content, err := os.ReadFile(r.resumePath)
	if err != nil {
		return "", false
	}
	trimmed := strings.TrimSpace(string(content))
	return trimmed, trimmed != ""
}

func buildLeadSummaryForPrompt(lead model.Lead) string {
	lines := []string{
		"Lead Summary:",
		fmt.Sprintf("- Company: %s", strings.TrimSpace(lead.Company)),
		fmt.Sprintf("- Position: %s", strings.TrimSpace(lead.Position)),
	}
	if source := strings.TrimSpace(lead.Source); source != "" {
		lines = append(lines, fmt.Sprintf("- Source: %s", source))
	}
	if location := strings.TrimSpace(lead.Location); location != "" {
		lines = append(lines, fmt.Sprintf("- Location: %s", location))
	}
	if jdURL := strings.TrimSpace(lead.JDURL); jdURL != "" {
		lines = append(lines, fmt.Sprintf("- JD URL: %s", jdURL))
	}
	if notes := strings.TrimSpace(lead.Notes); notes != "" {
		lines = append(lines, fmt.Sprintf("- Notes: %s", notes))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
