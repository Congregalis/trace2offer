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

type Topic struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type TopicCreateInput struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TopicPatchInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type KnowledgeDocument struct {
	Scope     Scope  `json:"scope"`
	ScopeID   string `json:"scope_id"`
	Filename  string `json:"filename"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

type KnowledgeDocumentCreateInput struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type KnowledgeDocumentUpdateInput struct {
	Content string `json:"content"`
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
