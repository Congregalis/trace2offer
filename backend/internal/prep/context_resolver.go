package prep

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"trace2offer/backend/internal/model"
)

var ErrContextResolverUnavailable = errors.New("prep context resolver is unavailable")

var companySlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

type ContextResolver struct {
	topicStore     *TopicStore
	knowledgeStore *KnowledgeStore
	resumePath     string
}

func NewContextResolver(prepDataDir string, topicStore *TopicStore, knowledgeStore *KnowledgeStore) *ContextResolver {
	rootDir := filepath.Dir(filepath.Clean(strings.TrimSpace(prepDataDir)))
	return &ContextResolver{
		topicStore:     topicStore,
		knowledgeStore: knowledgeStore,
		resumePath:     filepath.Join(rootDir, "resume", "current.md"),
	}
}

func (r *ContextResolver) Resolve(lead model.Lead) (LeadContextPreview, error) {
	if r == nil || r.topicStore == nil || r.knowledgeStore == nil {
		return LeadContextPreview{}, ErrContextResolverUnavailable
	}

	leadID := strings.TrimSpace(lead.ID)
	if leadID == "" {
		return LeadContextPreview{}, &ValidationError{Field: "lead_id", Message: "lead_id is required"}
	}

	preview := LeadContextPreview{
		LeadID:    leadID,
		Company:   strings.TrimSpace(lead.Company),
		Position:  strings.TrimSpace(lead.Position),
		TopicKeys: []string{},
		Sources:   []ContextSource{},
	}

	hasResume, err := r.hasResume()
	if err != nil {
		return LeadContextPreview{}, err
	}
	preview.HasResume = hasResume

	topics := r.topicStore.List()
	preview.TopicKeys = topicKeysFromTopics(topics)

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

	topicSources, err := r.collectTopicSources(preview.TopicKeys)
	if err != nil {
		return LeadContextPreview{}, err
	}
	preview.Sources = append(preview.Sources, topicSources...)

	return preview, nil
}

func (r *ContextResolver) collectTopicSources(topicKeys []string) ([]ContextSource, error) {
	sources := make([]ContextSource, 0)
	for _, key := range topicKeys {
		items, err := r.collectMarkdownSources(ScopeTopics, key, "topic")
		if err != nil {
			return nil, err
		}
		sources = append(sources, items...)
	}
	return sources, nil
}

func (r *ContextResolver) collectMarkdownSources(scope Scope, scopeID string, sourceScope string) ([]ContextSource, error) {
	scopeID = strings.TrimSpace(scopeID)
	if scopeID == "" {
		return []ContextSource{}, nil
	}

	documents, err := r.knowledgeStore.List(string(scope), scopeID)
	if err != nil {
		return nil, err
	}
	sources := make([]ContextSource, 0, len(documents))
	for _, document := range documents {
		title := fmt.Sprintf("%s/%s", scopeID, document.Filename)
		sources = append(sources, ContextSource{
			Scope: sourceScope,
			Kind:  "markdown",
			Title: title,
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

func topicKeysFromTopics(topics []Topic) []string {
	if len(topics) == 0 {
		return []string{}
	}

	keys := make([]string, 0, len(topics))
	for _, topic := range topics {
		key := strings.TrimSpace(topic.Key)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
