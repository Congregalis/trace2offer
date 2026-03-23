package model

// DiscoveryRule defines one periodic discovery source.
type DiscoveryRule struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	FeedURL         string   `json:"feed_url"`
	Source          string   `json:"source"`
	DefaultLocation string   `json:"default_location"`
	IncludeKeywords []string `json:"include_keywords,omitempty"`
	ExcludeKeywords []string `json:"exclude_keywords,omitempty"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       string   `json:"created_at,omitempty"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
}

// DiscoveryRuleMutationInput is the writable subset for discovery rules.
type DiscoveryRuleMutationInput struct {
	Name            string   `json:"name"`
	FeedURL         string   `json:"feed_url"`
	Source          string   `json:"source"`
	DefaultLocation string   `json:"default_location"`
	IncludeKeywords []string `json:"include_keywords"`
	ExcludeKeywords []string `json:"exclude_keywords"`
	Enabled         *bool    `json:"enabled"`
}

// DiscoveryRunResult is one discovery execution summary.
type DiscoveryRunResult struct {
	RanAt             string   `json:"ran_at"`
	RulesTotal        int      `json:"rules_total"`
	RulesExecuted     int      `json:"rules_executed"`
	EntriesFetched    int      `json:"entries_fetched"`
	CandidatesCreated int      `json:"candidates_created"`
	CandidatesUpdated int      `json:"candidates_updated"`
	Errors            []string `json:"errors,omitempty"`
}
