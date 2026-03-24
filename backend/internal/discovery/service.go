package discovery

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"trace2offer/backend/internal/candidate"
	"trace2offer/backend/internal/model"
	"trace2offer/backend/internal/storage"
)

var ErrRuleStoreUnavailable = fmt.Errorf("discovery rule store is unavailable")
var ErrCandidateManagerUnavailable = fmt.Errorf("candidate manager is unavailable")

// Service drives periodic candidate discovery from RSS/Atom feeds.
type Service struct {
	rules       storage.DiscoveryRuleStore
	candidates  candidate.Manager
	httpClient  *http.Client
	requestUser string
}

func NewService(ruleStore storage.DiscoveryRuleStore, candidateManager candidate.Manager) *Service {
	return &Service{
		rules:      ruleStore,
		candidates: candidateManager,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *Service) ListRules() []model.DiscoveryRule {
	if s == nil || s.rules == nil {
		return nil
	}
	return s.rules.List()
}

func (s *Service) CreateRule(input model.DiscoveryRuleMutationInput) (model.DiscoveryRule, error) {
	if s == nil || s.rules == nil {
		return model.DiscoveryRule{}, ErrRuleStoreUnavailable
	}
	normalized, err := normalizeRuleInput(input)
	if err != nil {
		return model.DiscoveryRule{}, err
	}
	return s.rules.Create(normalized)
}

func (s *Service) UpdateRule(id string, input model.DiscoveryRuleMutationInput) (model.DiscoveryRule, bool, error) {
	if s == nil || s.rules == nil {
		return model.DiscoveryRule{}, false, ErrRuleStoreUnavailable
	}
	normalizedID := strings.TrimSpace(id)
	if normalizedID == "" {
		return model.DiscoveryRule{}, false, &ValidationError{Field: "id", Message: "rule id is required"}
	}
	normalizedInput, err := normalizeRuleInput(input)
	if err != nil {
		return model.DiscoveryRule{}, false, err
	}
	return s.rules.Update(normalizedID, normalizedInput)
}

func (s *Service) DeleteRule(id string) (bool, error) {
	if s == nil || s.rules == nil {
		return false, ErrRuleStoreUnavailable
	}
	normalizedID := strings.TrimSpace(id)
	if normalizedID == "" {
		return false, &ValidationError{Field: "id", Message: "rule id is required"}
	}
	return s.rules.Delete(normalizedID)
}

func (s *Service) RunOnce(ctx context.Context, now time.Time) (model.DiscoveryRunResult, error) {
	result := model.DiscoveryRunResult{
		RanAt: now.UTC().Format(time.RFC3339),
	}
	if s == nil || s.rules == nil {
		return result, ErrRuleStoreUnavailable
	}
	if s.candidates == nil {
		return result, ErrCandidateManagerUnavailable
	}

	rules := s.rules.List()
	result.RulesTotal = len(rules)
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		result.RulesExecuted++

		entries, err := s.fetchFeedEntries(ctx, rule.FeedURL)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("rule=%s fetch failed: %v", rule.ID, err))
			continue
		}
		result.EntriesFetched += len(entries)

		for _, entry := range entries {
			if !shouldIncludeEntry(rule, entry) {
				continue
			}

			candidateInput := buildCandidateMutation(rule, entry)
			_, created, err := s.candidates.UpsertByJDURL(candidateInput)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("rule=%s entry=%s upsert failed: %v", rule.ID, entry.Link, err))
				continue
			}
			if created {
				result.CandidatesCreated++
			} else {
				result.CandidatesUpdated++
			}
		}
	}

	return result, nil
}

type feedEntry struct {
	Title       string
	Link        string
	Description string
}

func (s *Service) fetchFeedEntries(ctx context.Context, feedURL string) ([]feedEntry, error) {
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.requestUser) != "" {
		req.Header.Set("User-Agent", s.requestUser)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status=%d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if entries, ok := parseRSS(body); ok {
		return entries, nil
	}
	if entries, ok := parseAtom(body); ok {
		return entries, nil
	}
	return nil, fmt.Errorf("unsupported feed format")
}

func parseRSS(raw []byte) ([]feedEntry, bool) {
	var rss struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(raw, &rss); err != nil {
		return nil, false
	}
	if len(rss.Channel.Items) == 0 {
		return nil, false
	}
	entries := make([]feedEntry, 0, len(rss.Channel.Items))
	for _, item := range rss.Channel.Items {
		link := strings.TrimSpace(item.Link)
		if link == "" {
			continue
		}
		entries = append(entries, feedEntry{
			Title:       strings.TrimSpace(item.Title),
			Link:        link,
			Description: compactWhitespace(item.Description),
		})
	}
	return entries, true
}

func parseAtom(raw []byte) ([]feedEntry, bool) {
	var atom struct {
		Entries []struct {
			Title   string `xml:"title"`
			Summary string `xml:"summary"`
			Content string `xml:"content"`
			Links   []struct {
				Href string `xml:"href,attr"`
				Rel  string `xml:"rel,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(raw, &atom); err != nil {
		return nil, false
	}
	if len(atom.Entries) == 0 {
		return nil, false
	}
	entries := make([]feedEntry, 0, len(atom.Entries))
	for _, item := range atom.Entries {
		link := ""
		for _, candidate := range item.Links {
			if strings.TrimSpace(candidate.Href) == "" {
				continue
			}
			if candidate.Rel == "alternate" || candidate.Rel == "" {
				link = candidate.Href
				break
			}
		}
		if strings.TrimSpace(link) == "" && len(item.Links) > 0 {
			link = item.Links[0].Href
		}
		if strings.TrimSpace(link) == "" {
			continue
		}
		desc := strings.TrimSpace(item.Summary)
		if desc == "" {
			desc = strings.TrimSpace(item.Content)
		}
		entries = append(entries, feedEntry{
			Title:       strings.TrimSpace(item.Title),
			Link:        link,
			Description: compactWhitespace(desc),
		})
	}
	return entries, true
}

func shouldIncludeEntry(rule model.DiscoveryRule, entry feedEntry) bool {
	text := strings.ToLower(strings.TrimSpace(entry.Title + " " + entry.Description))
	if text == "" {
		return false
	}
	for _, item := range rule.ExcludeKeywords {
		keyword := strings.ToLower(strings.TrimSpace(item))
		if keyword == "" {
			continue
		}
		if strings.Contains(text, keyword) {
			return false
		}
	}
	if len(rule.IncludeKeywords) == 0 {
		return true
	}
	for _, item := range rule.IncludeKeywords {
		keyword := strings.ToLower(strings.TrimSpace(item))
		if keyword == "" {
			continue
		}
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func buildCandidateMutation(rule model.DiscoveryRule, entry feedEntry) model.CandidateMutationInput {
	link := strings.TrimSpace(entry.Link)
	source := strings.TrimSpace(rule.Source)
	if source == "" {
		source = "discovery:" + strings.TrimSpace(rule.Name)
	}

	includeHits := collectIncludeHits(rule.IncludeKeywords, entry.Title+" "+entry.Description)
	matchScore := 60
	if len(includeHits) > 0 {
		matchScore = 60 + len(includeHits)*10
		if matchScore > 95 {
			matchScore = 95
		}
	}

	recommendation := strings.TrimSpace(entry.Description)
	if recommendation == "" {
		recommendation = "Discovered from feed: " + rule.Name
	}
	if len(recommendation) > 280 {
		recommendation = recommendation[:280]
	}

	company, position := deriveCompanyAndPosition(rule, entry.Title)

	return model.CandidateMutationInput{
		Company:             company,
		Position:            position,
		Source:              source,
		Location:            strings.TrimSpace(rule.DefaultLocation),
		JDURL:               link,
		CompanyWebsiteURL:   "",
		Status:              candidate.StatusPendingReview,
		MatchScore:          matchScore,
		MatchReasons:        includeHits,
		RecommendationNotes: recommendation,
		Notes:               "",
	}
}

func collectIncludeHits(include []string, text string) []string {
	if len(include) == 0 {
		return nil
	}
	lowerText := strings.ToLower(text)
	seen := map[string]struct{}{}
	result := make([]string, 0, len(include))
	for _, item := range include {
		keyword := strings.TrimSpace(item)
		if keyword == "" {
			continue
		}
		lower := strings.ToLower(keyword)
		if strings.Contains(lowerText, lower) {
			if _, ok := seen[lower]; ok {
				continue
			}
			seen[lower] = struct{}{}
			result = append(result, keyword)
		}
	}
	return result
}

func deriveCompanyAndPosition(rule model.DiscoveryRule, title string) (string, string) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		trimmed = "Unknown Position"
	}

	if strings.Contains(trimmed, " at ") {
		parts := strings.SplitN(trimmed, " at ", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		if left != "" && right != "" {
			return right, left
		}
	}
	if strings.Contains(trimmed, " - ") {
		parts := strings.SplitN(trimmed, " - ", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		if left != "" && right != "" {
			return left, right
		}
	}

	company := strings.TrimSpace(rule.Name)
	if company == "" {
		company = "Discovered Company"
	}
	return company, trimmed
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "invalid discovery rule payload"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Field) != "" {
		return e.Field + " is invalid"
	}
	return "invalid discovery rule payload"
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

func normalizeRuleInput(input model.DiscoveryRuleMutationInput) (model.DiscoveryRuleMutationInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.FeedURL = strings.TrimSpace(input.FeedURL)
	input.Source = strings.TrimSpace(input.Source)
	input.DefaultLocation = strings.TrimSpace(input.DefaultLocation)
	input.IncludeKeywords = normalizeKeywordList(input.IncludeKeywords)
	input.ExcludeKeywords = normalizeKeywordList(input.ExcludeKeywords)

	if input.Name == "" {
		return model.DiscoveryRuleMutationInput{}, &ValidationError{
			Field:   "name",
			Message: "name is required",
		}
	}
	if input.FeedURL == "" {
		return model.DiscoveryRuleMutationInput{}, &ValidationError{
			Field:   "feed_url",
			Message: "feed_url is required",
		}
	}
	parsed, err := url.Parse(input.FeedURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return model.DiscoveryRuleMutationInput{}, &ValidationError{
			Field:   "feed_url",
			Message: "feed_url is invalid",
		}
	}
	input.FeedURL = parsed.String()

	return input, nil
}

func normalizeKeywordList(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	result := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func compactWhitespace(raw string) string {
	parts := strings.Fields(strings.TrimSpace(raw))
	return strings.Join(parts, " ")
}
