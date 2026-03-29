package prep

import "trace2offer/backend/internal/model"

type Scope string

const (
	ScopeTopics Scope = "topics"
	ScopeAll    Scope = "*"
)

const (
	RebuildModeIncremental = "incremental"
	RebuildModeFull        = "full"
)

const (
	IndexRunStatusRunning   = "running"
	IndexRunStatusCompleted = "completed"
	IndexRunStatusFailed    = "failed"
)

const (
	PrepSessionStatusDraft     = "draft"
	PrepSessionStatusSubmitted = "submitted"
)

const (
	EvaluationStatusPending   = "pending"
	EvaluationStatusRunning   = "running"
	EvaluationStatusCompleted = "completed"
	EvaluationStatusFailed    = "failed"
)

type Meta struct {
	Enabled              bool     `json:"enabled"`
	DefaultQuestionCount int      `json:"default_question_count"`
	SupportedScopes      []string `json:"supported_scopes"`
}

type Question struct {
	QuestionID     int      `json:"question_id,omitempty"`
	Question       string   `json:"question,omitempty"`
	ID             int      `json:"id"`
	Type           string   `json:"type"`
	Content        string   `json:"content"`
	ExpectedPoints []string `json:"expected_points"`
	ContextSources []string `json:"context_sources"`
}

type Answer struct {
	QuestionID  int     `json:"question_id"`
	Answer      string  `json:"answer"`
	SubmittedAt *string `json:"submitted_at,omitempty"`
}

type Session struct {
	ID               string                     `json:"id"`
	LeadID           string                     `json:"lead_id"`
	Company          string                     `json:"company"`
	Position         string                     `json:"position"`
	Status           string                     `json:"status"`
	Config           SessionConfig              `json:"config"`
	Sources          []ContextSource            `json:"sources"`
	Questions        []Question                 `json:"questions"`
	Answers          []Answer                   `json:"answers"`
	Evaluation       *Evaluation                `json:"evaluation,omitempty"`
	ReferenceAnswers map[string]ReferenceAnswer `json:"reference_answers"`
	GenerationTrace  *GenerationTrace           `json:"generation_trace,omitempty"`
	CreatedAt        string                     `json:"created_at"`
	UpdatedAt        string                     `json:"updated_at"`
}

type SaveDraftAnswersResult struct {
	SessionID    string `json:"session_id"`
	SavedAt      string `json:"saved_at"`
	AnswersCount int    `json:"answers_count"`
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

type ContextSource struct {
	Scope string `json:"scope"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
}

type LeadContextPreview struct {
	LeadID    string          `json:"lead_id"`
	Company   string          `json:"company"`
	Position  string          `json:"position"`
	HasResume bool            `json:"has_resume"`
	Sources   []ContextSource `json:"sources"`
}

type SessionConfig struct {
	QuestionCount   int  `json:"question_count"`
	IncludeResume   bool `json:"include_resume"`
	IncludeLeadDocs bool `json:"include_lead_docs"`
}

type CreateSessionInput struct {
	LeadID          string `json:"lead_id"`
	QuestionCount   int    `json:"question_count"`
	IncludeResume   bool   `json:"include_resume"`
	IncludeLeadDocs bool   `json:"include_lead_docs"`
}

type GenerationConfig struct {
	Lead model.Lead

	LeadID          string `json:"lead_id"`
	QuestionCount   int    `json:"question_count"`
	IncludeResume   bool   `json:"include_resume"`
	IncludeLeadDocs bool   `json:"include_lead_docs"`
}

type GenerationResult struct {
	Session *Session `json:"session"`
}

type Evaluation struct {
	Status       string            `json:"status"`
	Error        string            `json:"error,omitempty"`
	Scores       []QuestionScore   `json:"scores"`
	Overall      OverallEvaluation `json:"overall"`
	StartedAt    string            `json:"started_at,omitempty"`
	CompletedAt  string            `json:"completed_at,omitempty"`
	OverallScore float64           `json:"overall_score,omitempty"`
	Summary      string            `json:"summary,omitempty"`
}

type QuestionScore struct {
	QuestionID   int                    `json:"question_id"`
	Score        float64                `json:"score"`
	Answered     bool                   `json:"answered"`
	Summary      string                 `json:"summary"`
	Strengths    []string               `json:"strengths"`
	Improvements []string               `json:"improvements"`
	WeakPoints   []string               `json:"weak_points"`
	Sources      []QuestionScoreSource  `json:"sources"`
	Trace        map[string]interface{} `json:"trace,omitempty"`
}

type QuestionScoreSource struct {
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

type OverallEvaluation struct {
	AverageScore   float64  `json:"average_score"`
	AnsweredCount  int      `json:"answered_count"`
	TotalQuestions int      `json:"total_questions"`
	Strengths      []string `json:"strengths"`
	WeakPoints     []string `json:"weak_points"`
	Summary        string   `json:"summary"`
}

type ReferenceAnswer struct {
	QuestionID int      `json:"question_id"`
	Summary    string   `json:"summary"`
	Points     []string `json:"points"`
	Source     string   `json:"source,omitempty"`
}

type GenerationTrace struct {
	InputSnapshot    InputSnapshot      `json:"input_snapshot"`
	QueryPlanning    QueryPlanningTrace `json:"query_planning"`
	RetrievalQuery   string             `json:"retrieval_query"`
	RetrievalResults RetrievalSummary   `json:"retrieval_results"`
	PromptSections   []PromptSection    `json:"prompt_sections"`
	AssembledPrompt  string             `json:"assembled_prompt"`
	GenerationResult GenerationSummary  `json:"generation_result"`
}

const (
	GenerationStageInputSnapshot  = "input_snapshot"
	GenerationStageQueryPlanning  = "query_planning"
	GenerationStageRetrieval      = "retrieval"
	GenerationStagePromptAssembly = "prompt_assembly"
	GenerationStageGeneration     = "generation"
)

const (
	GenerationProgressStarted   = "started"
	GenerationProgressProgress  = "progress"
	GenerationProgressCompleted = "completed"
)

type GenerationProgressEvent struct {
	Stage   string           `json:"stage"`
	Status  string           `json:"status"`
	Message string           `json:"message,omitempty"`
	Delta   string           `json:"delta,omitempty"`
	Trace   *GenerationTrace `json:"trace,omitempty"`
}

type GenerationProgressReporter func(event GenerationProgressEvent)

type QueryPlanningTrace struct {
	Strategy      string `json:"strategy"`
	Model         string `json:"model"`
	ResumeExcerpt string `json:"resume_excerpt"`
	JDExcerpt     string `json:"jd_excerpt"`
	Prompt        string `json:"prompt"`
	RawOutput     string `json:"raw_output"`
	FinalQuery    string `json:"final_query"`
}

type InputSnapshot struct {
	LeadID        string `json:"lead_id"`
	QuestionCount int    `json:"question_count"`
}

type RetrievalSummary struct {
	CandidatesFound int      `json:"candidates_found"`
	FinalSelected   int      `json:"final_selected"`
	Sources         []string `json:"sources"`
}

type PromptSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type GenerationSummary struct {
	QuestionsGenerated int    `json:"questions_generated"`
	GenerationTimeMS   int64  `json:"generation_time_ms"`
	Model              string `json:"model"`
}

type ChunkConfig struct {
	ChunkSize       int `json:"chunk_size"`
	Overlap         int `json:"overlap"`
	ChunkSizeTokens int `json:"chunk_size_tokens,omitempty"`
	OverlapTokens   int `json:"overlap_tokens,omitempty"`
}

type EmbeddingConfig struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key,omitempty"`
	Dimension int    `json:"dimension"`
}

type IndexRunSummary struct {
	RunID            string          `json:"run_id"`
	Mode             string          `json:"mode"`
	StartedAt        string          `json:"started_at"`
	CompletedAt      string          `json:"completed_at"`
	Status           string          `json:"status"`
	DocumentsScanned int             `json:"documents_scanned"`
	DocumentsIndexed int             `json:"documents_indexed"`
	DocumentsSkipped int             `json:"documents_skipped"`
	DocumentsDeleted int             `json:"documents_deleted"`
	ChunksCreated    int             `json:"chunks_created"`
	ChunksUpdated    int             `json:"chunks_updated"`
	Errors           []IndexRunError `json:"errors"`
}

type IndexRunError struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

type RebuildIndexInput struct {
	Scope   string `json:"scope"`
	ScopeID string `json:"scope_id"`
	Mode    string `json:"mode"`
}

type IndexStatus struct {
	EmbeddingProvider string `json:"embedding_provider"`
	EmbeddingModel    string `json:"embedding_model"`
	DocumentCount     int    `json:"document_count"`
	ChunkCount        int    `json:"chunk_count"`
	LastIndexedAt     string `json:"last_indexed_at"`
	LastIndexStatus   string `json:"last_index_status"`
}

type Document struct {
	ID          string `json:"id"`
	Scope       Scope  `json:"scope"`
	ScopeID     string `json:"scope_id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	SourcePath  string `json:"source_path"`
	ContentHash string `json:"content_hash"`
	UpdatedAt   string `json:"updated_at"`
}

type Chunk struct {
	Index      int       `json:"index"`
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"`
	Embedding  []float32 `json:"embedding"`
	UpdatedAt  string    `json:"updated_at"`
}

type IndexRun struct {
	ID                 string `json:"id"`
	StartedAt          string `json:"started_at"`
	CompletedAt        string `json:"completed_at"`
	Status             string `json:"status"`
	DocumentsProcessed int    `json:"documents_processed"`
	ChunksCreated      int    `json:"chunks_created"`
	Errors             string `json:"errors"`
}

type SearchConfig struct {
	LeadID          string `json:"lead_id"`
	CompanySlug     string `json:"company_slug,omitempty"`
	Query           string `json:"query"`
	TopK            int    `json:"top_k"`
	IncludeTrace    bool   `json:"include_trace"`
	IncludeResume   bool   `json:"include_resume"`
	IncludeLeadDocs bool   `json:"include_lead_docs"`
}

type SearchResult struct {
	Query           string           `json:"query"`
	NormalizedQuery string           `json:"normalized_query"`
	Filters         SearchFilters    `json:"filters"`
	Trace           *RetrievalTrace  `json:"trace,omitempty"`
	CandidateChunks []RetrievedChunk `json:"candidate_chunks"`
	RetrievedChunks []RetrievedChunk `json:"retrieved_chunks"`
	FinalContext    FinalContext     `json:"final_context"`
}

type SearchFilters struct {
	Scope []string `json:"scope"`
}

type RetrievalTrace struct {
	StageQueryNormalization TraceStage `json:"stage_query_normalization"`
	StageInitialRetrieval   TraceStage `json:"stage_initial_retrieval"`
	StageDeduplication      TraceStage `json:"stage_deduplication"`
	StageReranking          TraceStage `json:"stage_reranking"`
}

type TraceStage struct {
	Input       string                 `json:"input,omitempty"`
	Output      string                 `json:"output,omitempty"`
	Method      string                 `json:"method"`
	InputCount  int                    `json:"input_count,omitempty"`
	OutputCount int                    `json:"output_count,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type RetrievedChunk struct {
	ID          string      `json:"id"`
	Content     string      `json:"content"`
	Score       float64     `json:"score"`
	Source      ChunkSource `json:"source"`
	WhySelected string      `json:"why_selected"`
}

type ChunkSource struct {
	Scope         string `json:"scope"`
	ScopeID       string `json:"scope_id"`
	DocumentTitle string `json:"document_title"`
	ChunkIndex    int    `json:"chunk_index"`
}

type FinalContext struct {
	TotalTokens int    `json:"total_tokens"`
	ChunksUsed  int    `json:"chunks_used"`
	Context     string `json:"context"`
}

type IndexedChunk struct {
	ID            string `json:"id"`
	DocumentID    string `json:"document_id"`
	Scope         Scope  `json:"scope"`
	ScopeID       string `json:"scope_id"`
	DocumentTitle string `json:"document_title"`
	ChunkIndex    int    `json:"chunk_index"`
	Content       string `json:"content"`
	TokenCount    int    `json:"token_count"`
	UpdatedAt     string `json:"updated_at"`
}

func DefaultSupportedScopes() []Scope {
	return []Scope{
		ScopeTopics,
	}
}

func isSupportedScope(scope Scope) bool {
	switch scope {
	case ScopeTopics:
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
