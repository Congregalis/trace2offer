package model

// Lead is the canonical data shape used by API and file persistence.
type Lead struct {
	ID                string   `json:"id"`
	Company           string   `json:"company"`
	Position          string   `json:"position"`
	Source            string   `json:"source"`
	Status            string   `json:"status"`
	Priority          int      `json:"priority"`
	NextAction        string   `json:"next_action"`
	NextActionAt      string   `json:"next_action_at,omitempty"`
	ReminderMethods   []string `json:"reminder_methods,omitempty"`
	Notes             string   `json:"notes"`
	CompanyWebsiteURL string   `json:"company_website_url"`
	JDURL             string   `json:"jd_url"`
	Location          string   `json:"location"`
	CreatedAt         string   `json:"created_at,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
}

// LeadMutationInput is the writable subset for create/update endpoints.
type LeadMutationInput struct {
	Company           string   `json:"company"`
	Position          string   `json:"position"`
	Source            string   `json:"source"`
	Status            string   `json:"status"`
	Priority          int      `json:"priority"`
	NextAction        string   `json:"next_action"`
	NextActionAt      string   `json:"next_action_at"`
	ReminderMethods   []string `json:"reminder_methods"`
	Notes             string   `json:"notes"`
	CompanyWebsiteURL string   `json:"company_website_url"`
	JDURL             string   `json:"jd_url"`
	Location          string   `json:"location"`
}
