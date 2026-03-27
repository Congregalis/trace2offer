package prep

type Scope string

const (
	ScopeTopics    Scope = "topics"
	ScopeCompanies Scope = "companies"
	ScopeLeads     Scope = "leads"
)

type Meta struct {
	Enabled              bool     `json:"enabled"`
	DefaultQuestionCount int      `json:"default_question_count"`
	SupportedScopes      []string `json:"supported_scopes"`
}

func DefaultSupportedScopes() []Scope {
	return []Scope{
		ScopeTopics,
		ScopeCompanies,
		ScopeLeads,
	}
}

func isSupportedScope(scope Scope) bool {
	switch scope {
	case ScopeTopics, ScopeCompanies, ScopeLeads:
		return true
	default:
		return false
	}
}

func scopeNames(scopes []Scope) []string {
	names := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		names = append(names, string(scope))
	}
	return names
}
