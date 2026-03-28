package prep

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentprovider "trace2offer/backend/agent/provider"
	openaiprovider "trace2offer/backend/agent/provider/openai"
	"trace2offer/backend/internal/model"
)

var ErrQuestionGeneratorUnavailable = fmt.Errorf("prep question generator is unavailable")

type QuestionModel interface {
	Name() string
	Generate(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

type openAIQuestionModel struct {
	model    string
	provider agentprovider.Provider
}

func NewOpenAIQuestionModel(apiKey string, baseURL string, modelName string, timeout time.Duration) (QuestionModel, error) {
	provider, err := openaiprovider.New(openaiprovider.Config{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: strings.TrimSpace(baseURL),
		Model:   strings.TrimSpace(modelName),
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return &openAIQuestionModel{model: strings.TrimSpace(modelName), provider: provider}, nil
}

func (m *openAIQuestionModel) Name() string {
	if m == nil || strings.TrimSpace(m.model) == "" {
		return "gpt-5-mini"
	}
	return m.model
}

func (m *openAIQuestionModel) Generate(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	if m == nil || m.provider == nil {
		return "", fmt.Errorf("question model provider is nil")
	}
	response, err := m.provider.Generate(ctx, agentprovider.Request{
		Model: m.model,
		Messages: []agentprovider.Message{
			{Role: "system", Content: strings.TrimSpace(systemPrompt)},
			{Role: "user", Content: strings.TrimSpace(userPrompt)},
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response.Content), nil
}

type QuestionGenerator struct {
	resolver             *ContextResolver
	retrieval            *RetrievalEngine
	sessionStore         *SessionStore
	embeddingProvider    EmbeddingProvider
	model                QuestionModel
	defaultQuestionCount int
	now                  func() time.Time
}

func NewQuestionGenerator(
	resolver *ContextResolver,
	retrieval *RetrievalEngine,
	sessionStore *SessionStore,
	embeddingProvider EmbeddingProvider,
	model QuestionModel,
	defaultQuestions int,
) *QuestionGenerator {
	if defaultQuestions <= 0 {
		defaultQuestions = defaultQuestionCount
	}
	return &QuestionGenerator{
		resolver:             resolver,
		retrieval:            retrieval,
		sessionStore:         sessionStore,
		embeddingProvider:    embeddingProvider,
		model:                model,
		defaultQuestionCount: defaultQuestions,
		now:                  time.Now,
	}
}

func (g *QuestionGenerator) Generate(config GenerationConfig) (*GenerationResult, error) {
	return g.GenerateWithContext(context.Background(), config)
}

func (g *QuestionGenerator) GenerateFromInput(ctx context.Context, lead model.Lead, input CreateSessionInput) (Session, error) {
	result, err := g.GenerateWithContext(ctx, GenerationConfig{
		Lead:            lead,
		LeadID:          input.LeadID,
		TopicKeys:       input.TopicKeys,
		QuestionCount:   input.QuestionCount,
		IncludeResume:   input.IncludeResume,
		IncludeProfile:  input.IncludeProfile,
		IncludeLeadDocs: input.IncludeLeadDocs,
	})
	if err != nil {
		return Session{}, err
	}
	if result == nil || result.Session == nil {
		return Session{}, fmt.Errorf("generated session is nil")
	}
	return *result.Session, nil
}

func (g *QuestionGenerator) GenerateWithContext(ctx context.Context, config GenerationConfig) (*GenerationResult, error) {
	if g == nil {
		return nil, ErrQuestionGeneratorUnavailable
	}

	leadID := strings.TrimSpace(config.LeadID)
	if leadID == "" {
		leadID = strings.TrimSpace(config.Lead.ID)
	}
	if leadID == "" {
		return nil, &ValidationError{Field: "lead_id", Message: "lead_id is required"}
	}
	topicKeys := cleanStrings(config.TopicKeys)
	if len(topicKeys) == 0 {
		return nil, &ValidationError{Field: "topic_keys", Message: "topic_keys is required"}
	}
	questionCount := config.QuestionCount
	if questionCount <= 0 {
		questionCount = g.defaultQuestionCount
		if questionCount <= 0 {
			questionCount = defaultQuestionCount
		}
	}
	startedAt := g.now()

	retrievalQuery := strings.TrimSpace(config.Lead.Position + " " + config.Lead.Company + " " + config.Lead.JDText)
	if retrievalQuery == "" {
		retrievalQuery = "interview preparation"
	}

	searchResult := SearchResult{}
	if g.retrieval != nil {
		searchInput := SearchConfig{
			LeadID:          leadID,
			CompanySlug:     normalizeCompanySlug(config.Lead.Company),
			Query:           retrievalQuery,
			TopicKeys:       topicKeys,
			TopK:            questionCount,
			IncludeTrace:    true,
			IncludeResume:   config.IncludeResume,
			IncludeProfile:  config.IncludeProfile,
			IncludeLeadDocs: config.IncludeLeadDocs,
		}
		result, err := g.retrieval.Search(retrievalQuery, searchInput)
		if err != nil {
			return nil, err
		}
		if result != nil {
			searchResult = *result
		}
	}

	candidateProfile := buildLeadSummaryForPrompt(config.Lead)
	if g.resolver != nil {
		candidateProfile = g.resolver.BuildPromptCandidateProfile(config.Lead)
	}

	promptSections := BuildQuestionGenerationPromptSections(PromptBuildInput{
		Count:            questionCount,
		TopicKeys:        topicKeys,
		RetrievedChunks:  searchResult.RetrievedChunks,
		CandidateProfile: candidateProfile,
		JobDescription:   strings.TrimSpace(config.Lead.JDText),
	})

	modelName := "fallback"
	modelOutput := ""
	if g.model != nil {
		modelName = g.model.Name()
			output, err := g.model.Generate(ctx, promptSections.System, BuildQuestionGenerationPrompt(PromptConfig{
				Count:            questionCount,
				TopicKeys:        topicKeys,
				RetrievedChunks:  searchResult.RetrievedChunks,
				CandidateProfile: candidateProfile,
				JobDescription:   strings.TrimSpace(config.Lead.JDText),
			}))
		if err != nil {
			return nil, err
		}
		modelOutput = output
	}

	sourceTitles := uniqueSourceTitles(searchResult.RetrievedChunks)
	questions := parseGeneratedQuestions(modelOutput, questionCount, sourceTitles, topicKeys)

	sources := []ContextSource{}
	if g.resolver != nil {
		preview, err := g.resolver.Resolve(config.Lead)
		if err == nil {
			sources = preview.Sources
		}
	}

	now := g.now().UTC().Format(time.RFC3339)
	generationMS := time.Since(startedAt).Milliseconds()
	session := &Session{
		ID:       fmt.Sprintf("prep_%d", g.now().UTC().UnixNano()),
		LeadID:   leadID,
		Company:  strings.TrimSpace(config.Lead.Company),
		Position: strings.TrimSpace(config.Lead.Position),
		Status:   PrepSessionStatusDraft,
		Config: SessionConfig{
			TopicKeys:       topicKeys,
			QuestionCount:   questionCount,
			IncludeResume:   config.IncludeResume,
			IncludeProfile:  config.IncludeProfile,
			IncludeLeadDocs: config.IncludeLeadDocs,
		},
		Sources:          sources,
		Questions:        questions,
		Answers:          []Answer{},
		ReferenceAnswers: map[string]ReferenceAnswer{},
		GenerationTrace: &GenerationTrace{
			InputSnapshot: InputSnapshot{
				LeadID:        leadID,
				TopicKeys:     topicKeys,
				QuestionCount: questionCount,
			},
			RetrievalQuery: retrievalQuery,
			RetrievalResults: RetrievalSummary{
				CandidatesFound: len(searchResult.CandidateChunks),
				FinalSelected:   len(searchResult.RetrievedChunks),
				Sources:         sourceTitles,
			},
			PromptSections: []PromptSection{
				{Title: "system", Content: promptSections.System},
				{Title: "context", Content: promptSections.Context},
				{Title: "task", Content: promptSections.Task},
			},
			GenerationResult: GenerationSummary{
				QuestionsGenerated: len(questions),
				GenerationTimeMS:   generationMS,
				Model:              modelName,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if g.sessionStore != nil {
		if err := g.sessionStore.Create(session); err != nil {
			return nil, err
		}
	}

	return &GenerationResult{Session: session}, nil
}

func parseGeneratedQuestions(raw string, questionCount int, defaultSources []string, topicKeys []string) []Question {
	if questionCount <= 0 {
		questionCount = defaultQuestionCount
	}

	type generatedQuestion struct {
		ID             int      `json:"id"`
		Type           string   `json:"type"`
		Content        string   `json:"content"`
		ExpectedPoints []string `json:"expected_points"`
		ContextSources []string `json:"context_sources"`
	}
	var payload struct {
		Questions []generatedQuestion `json:"questions"`
	}

	questions := make([]Question, 0, questionCount)
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		for _, item := range payload.Questions {
			content := strings.TrimSpace(item.Content)
			if content == "" {
				continue
			}
			qid := item.ID
			if qid <= 0 {
				qid = len(questions) + 1
			}
			qType := strings.TrimSpace(item.Type)
			if qType == "" {
				qType = "technical"
			}
			sources := cleanStrings(item.ContextSources)
			if len(sources) == 0 {
				sources = append(sources, defaultSources...)
			}
			expected := cleanStrings(item.ExpectedPoints)
			if len(expected) == 0 {
				expected = []string{"结构化回答", "结合实际经验"}
			}
			questions = append(questions, Question{
				ID:             qid,
				Type:           qType,
				Content:        content,
				ExpectedPoints: expected,
				ContextSources: sources,
			})
			if len(questions) >= questionCount {
				break
			}
		}
	}

	for len(questions) < questionCount {
		nextID := len(questions) + 1
		topic := "通用"
		if len(topicKeys) > 0 {
			topic = strings.TrimSpace(topicKeys[nextID%len(topicKeys)])
			if topic == "" {
				topic = "通用"
			}
		}
		questions = append(questions, Question{
			ID:      nextID,
			Type:    "technical",
			Content: fmt.Sprintf("请说明你在 %s 相关项目中的关键决策和权衡。", topic),
			ExpectedPoints: []string{
				"背景与目标",
				"技术方案与取舍",
				"结果与复盘",
			},
			ContextSources: append([]string{}, defaultSources...),
		})
	}

	return questions
}

func uniqueSourceTitles(chunks []RetrievedChunk) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(chunks))
	for _, item := range chunks {
		title := strings.TrimSpace(item.Source.DocumentTitle)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		out = append(out, title)
	}
	return out
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
