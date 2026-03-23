package model

// Candidate is a recommended opportunity waiting for manual review.
type Candidate struct {
	ID                  string   `json:"id"`
	Company             string   `json:"company"`
	Position            string   `json:"position"`
	Source              string   `json:"source"`
	Location            string   `json:"location"`
	JDURL               string   `json:"jd_url"`
	CompanyWebsiteURL   string   `json:"company_website_url"`
	Status              string   `json:"status"`
	MatchScore          int      `json:"match_score"`
	MatchReasons        []string `json:"match_reasons,omitempty"`
	RecommendationNotes string   `json:"recommendation_notes"`
	Notes               string   `json:"notes"`
	PromotedLeadID      string   `json:"promoted_lead_id,omitempty"`
	CreatedAt           string   `json:"created_at,omitempty"`
	UpdatedAt           string   `json:"updated_at,omitempty"`
}

// CandidateMutationInput is the writable subset for create/update endpoints.
type CandidateMutationInput struct {
	Company             string   `json:"company"`
	Position            string   `json:"position"`
	Source              string   `json:"source"`
	Location            string   `json:"location"`
	JDURL               string   `json:"jd_url"`
	CompanyWebsiteURL   string   `json:"company_website_url"`
	Status              string   `json:"status"`
	MatchScore          int      `json:"match_score"`
	MatchReasons        []string `json:"match_reasons"`
	RecommendationNotes string   `json:"recommendation_notes"`
	Notes               string   `json:"notes"`
	PromotedLeadID      string   `json:"promoted_lead_id"`
}

// CandidatePromoteInput controls how a candidate is converted to a lead.
type CandidatePromoteInput struct {
	Source          string   `json:"source"`
	Status          string   `json:"status"`
	Priority        int      `json:"priority"`
	NextAction      string   `json:"next_action"`
	NextActionAt    string   `json:"next_action_at"`
	InterviewAt     string   `json:"interview_at"`
	ReminderMethods []string `json:"reminder_methods"`
	Notes           string   `json:"notes"`
}
