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

const retrievalQueryPlannerSystemPrompt = `You are the retrieval-query planner for Trace2Offer interview prep.

Goal:
- Produce one high-recall, high-precision query for searching interview-prep knowledge chunks.

Hard rules:
1) Prioritize job-critical skills from JD and candidate-specific strengths from resume.
2) Include topic keys when they are relevant; avoid stuffing unrelated keywords.
3) Keep query concise (roughly <= 20 terms), information-dense, and search-friendly.
4) Output JSON only: {"query":"..."}`

type QuestionModel interface {
	Name() string
	Generate(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

type QuestionModelStreamer interface {
	GenerateStream(
		ctx context.Context,
		systemPrompt string,
		userPrompt string,
		onDelta func(string),
	) (string, error)
}

type QuestionModelStructuredGenerator interface {
	GenerateStructuredQuestions(
		ctx context.Context,
		systemPrompt string,
		userPrompt string,
		questionCount int,
		onDelta func(string),
	) (string, error)
}

type openAIProviderStreamer interface {
	GenerateStream(ctx context.Context, request agentprovider.Request, onDelta func(string)) (agentprovider.Response, error)
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

func (m *openAIQuestionModel) GenerateStream(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
	onDelta func(string),
) (string, error) {
	if m == nil || m.provider == nil {
		return "", fmt.Errorf("question model provider is nil")
	}
	streamer, ok := m.provider.(openAIProviderStreamer)
	if !ok {
		return m.Generate(ctx, systemPrompt, userPrompt)
	}

	response, err := streamer.GenerateStream(ctx, agentprovider.Request{
		Model: m.model,
		Messages: []agentprovider.Message{
			{Role: "system", Content: strings.TrimSpace(systemPrompt)},
			{Role: "user", Content: strings.TrimSpace(userPrompt)},
		},
	}, onDelta)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response.Content), nil
}

func (m *openAIQuestionModel) GenerateStructuredQuestions(
	ctx context.Context,
	systemPrompt string,
	userPrompt string,
	questionCount int,
	onDelta func(string),
) (string, error) {
	if m == nil || m.provider == nil {
		return "", fmt.Errorf("question model provider is nil")
	}
	if questionCount <= 0 {
		questionCount = defaultQuestionCount
	}

	request := agentprovider.Request{
		Model: m.model,
		Messages: []agentprovider.Message{
			{Role: "system", Content: strings.TrimSpace(systemPrompt)},
			{Role: "user", Content: strings.TrimSpace(userPrompt)},
		},
		Tools: []agentprovider.Tool{
			buildInterviewQuestionsTool(questionCount),
		},
		ToolChoice: &agentprovider.ToolChoice{
			Type: "function",
			Name: "emit_interview_questions",
		},
	}

	streamer, ok := m.provider.(openAIProviderStreamer)
	if !ok {
		response, err := m.provider.Generate(ctx, request)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(response.Content), nil
	}

	response, err := streamer.GenerateStream(ctx, request, onDelta)
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
	return g.GenerateWithProgress(ctx, config, nil)
}

func (g *QuestionGenerator) GenerateWithProgress(
	ctx context.Context,
	config GenerationConfig,
	reporter GenerationProgressReporter,
) (*GenerationResult, error) {
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
	trace := GenerationTrace{
		InputSnapshot: InputSnapshot{
			LeadID:        leadID,
			TopicKeys:     append([]string{}, topicKeys...),
			QuestionCount: questionCount,
		},
		QueryPlanning: QueryPlanningTrace{},
		RetrievalResults: RetrievalSummary{
			Sources: []string{},
		},
		PromptSections: []PromptSection{},
	}
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageInputSnapshot,
		Status:  GenerationProgressCompleted,
		Message: "输入配置已锁定",
		Trace:   cloneGenerationTrace(trace),
	})

	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageQueryPlanning,
		Status:  GenerationProgressStarted,
		Message: "开始规划检索 query",
		Trace:   cloneGenerationTrace(trace),
	})

	resumeForQuery := ""
	if g.resolver != nil {
		if resumeText, ok := g.resolver.readResumeText(); ok {
			resumeForQuery = resumeText
		}
	}
	jdForQuery := strings.TrimSpace(config.Lead.JDText)
	queryPlanningDeltaCount := 0
	queryPlanning := g.planRetrievalQuery(ctx, config.Lead, topicKeys, resumeForQuery, jdForQuery, func(delta string) {
		queryPlanningDeltaCount++
		g.emitProgress(reporter, GenerationProgressEvent{
			Stage:   GenerationStageQueryPlanning,
			Status:  GenerationProgressProgress,
			Message: "Agent 输出片段",
			Delta:   delta,
			Trace:   cloneGenerationTrace(trace),
		})
	})
	retrievalQuery := strings.TrimSpace(queryPlanning.FinalQuery)
	if retrievalQuery == "" {
		retrievalQuery = fallbackRetrievalQuery(config.Lead, topicKeys)
		queryPlanning.FinalQuery = retrievalQuery
		queryPlanning.Strategy = "fallback"
	}
	trace.QueryPlanning = queryPlanning
	trace.RetrievalQuery = retrievalQuery
	if queryPlanningDeltaCount == 0 {
		if err := g.emitTextDeltas(
			ctx,
			reporter,
			GenerationStageQueryPlanning,
			queryPlanning.RawOutput,
			"Agent 输出片段",
			trace,
		); err != nil {
			return nil, err
		}
	}
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageQueryPlanning,
		Status:  GenerationProgressCompleted,
		Message: "检索 query 规划完成",
		Trace:   cloneGenerationTrace(trace),
	})

	searchResult := SearchResult{}
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageRetrieval,
		Status:  GenerationProgressStarted,
		Message: "开始执行检索",
		Trace:   cloneGenerationTrace(trace),
	})
	if g.retrieval != nil {
		searchInput := SearchConfig{
			LeadID:          leadID,
			CompanySlug:     normalizeCompanySlug(config.Lead.Company),
			Query:           retrievalQuery,
			TopicKeys:       topicKeys,
			TopK:            questionCount,
			IncludeTrace:    true,
			IncludeResume:   config.IncludeResume,
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
	sourceTitles := uniqueSourceTitles(searchResult.RetrievedChunks)
	trace.RetrievalResults = RetrievalSummary{
		CandidatesFound: len(searchResult.CandidateChunks),
		FinalSelected:   len(searchResult.RetrievedChunks),
		Sources:         sourceTitles,
	}
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageRetrieval,
		Status:  GenerationProgressCompleted,
		Message: "检索完成",
		Trace:   cloneGenerationTrace(trace),
	})

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
	assembledPrompt := BuildQuestionGenerationPrompt(PromptConfig{
		Count:            questionCount,
		TopicKeys:        topicKeys,
		RetrievedChunks:  searchResult.RetrievedChunks,
		CandidateProfile: candidateProfile,
		JobDescription:   strings.TrimSpace(config.Lead.JDText),
	})
	trace.PromptSections = []PromptSection{
		{Title: "system", Content: promptSections.System},
		{Title: "context", Content: promptSections.Context},
		{Title: "candidate_context", Content: promptSections.CandidateProfile},
		{Title: "job_description", Content: promptSections.JobDescription},
		{Title: "task", Content: promptSections.Task},
		{Title: "requirements", Content: promptSections.Requirements},
		{Title: "output_format", Content: promptSections.OutputFormat},
	}
	trace.AssembledPrompt = assembledPrompt
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStagePromptAssembly,
		Status:  GenerationProgressCompleted,
		Message: "Prompt 组装完成",
		Trace:   cloneGenerationTrace(trace),
	})

	modelName := "fallback"
	modelOutput := ""
	generationDeltaCount := 0
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageGeneration,
		Status:  GenerationProgressStarted,
		Message: "开始生成问题",
		Trace:   cloneGenerationTrace(trace),
	})
	if g.model != nil {
		modelName = g.model.Name()
		if structured, ok := g.model.(QuestionModelStructuredGenerator); ok {
			output, err := structured.GenerateStructuredQuestions(ctx, promptSections.System, assembledPrompt, questionCount, func(delta string) {
				generationDeltaCount++
				g.emitProgress(reporter, GenerationProgressEvent{
					Stage:   GenerationStageGeneration,
					Status:  GenerationProgressProgress,
					Message: "模型输出片段",
					Delta:   delta,
					Trace:   cloneGenerationTrace(trace),
				})
			})
			if err != nil {
				return nil, err
			}
			modelOutput = output
		} else if streamer, ok := g.model.(QuestionModelStreamer); ok {
			var streamedOutput strings.Builder
			output, err := streamer.GenerateStream(ctx, promptSections.System, assembledPrompt, func(delta string) {
				generationDeltaCount++
				streamedOutput.WriteString(delta)
				g.emitProgress(reporter, GenerationProgressEvent{
					Stage:   GenerationStageGeneration,
					Status:  GenerationProgressProgress,
					Message: "模型输出片段",
					Delta:   delta,
					Trace:   cloneGenerationTrace(trace),
				})
			})
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(output) == "" {
				output = streamedOutput.String()
			}
			modelOutput = output
		} else {
			output, err := g.model.Generate(ctx, promptSections.System, assembledPrompt)
			if err != nil {
				return nil, err
			}
			modelOutput = output
		}
	}
	if generationDeltaCount == 0 {
		if err := g.emitTextDeltas(
			ctx,
			reporter,
			GenerationStageGeneration,
			modelOutput,
			"模型输出片段",
			trace,
		); err != nil {
			return nil, err
		}
	}
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
	trace.GenerationResult = GenerationSummary{
		QuestionsGenerated: len(questions),
		GenerationTimeMS:   generationMS,
		Model:              modelName,
	}
	g.emitProgress(reporter, GenerationProgressEvent{
		Stage:   GenerationStageGeneration,
		Status:  GenerationProgressCompleted,
		Message: "问题生成完成",
		Trace:   cloneGenerationTrace(trace),
	})
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
			IncludeLeadDocs: config.IncludeLeadDocs,
		},
		Sources:          sources,
		Questions:        questions,
		Answers:          []Answer{},
		ReferenceAnswers: map[string]ReferenceAnswer{},
		GenerationTrace:  cloneGenerationTrace(trace),
		CreatedAt:        now,
		UpdatedAt:        now,
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

func (g *QuestionGenerator) planRetrievalQuery(
	ctx context.Context,
	lead model.Lead,
	topicKeys []string,
	resumeText string,
	jdText string,
	onDelta func(string),
) QueryPlanningTrace {
	fallbackQuery := fallbackRetrievalQuery(lead, topicKeys)
	trace := QueryPlanningTrace{
		Strategy:      "fallback",
		Model:         "fallback",
		ResumeExcerpt: excerptForTrace(resumeText, 280),
		JDExcerpt:     excerptForTrace(jdText, 280),
		Prompt:        "",
		RawOutput:     "",
		FinalQuery:    fallbackQuery,
	}

	if g == nil || g.model == nil {
		return trace
	}

	prompt := buildRetrievalQueryPlannerPrompt(lead, topicKeys, resumeText, jdText)
	modelName := strings.TrimSpace(g.model.Name())
	if modelName == "" {
		modelName = "unknown"
	}
	trace.Model = modelName
	trace.Prompt = prompt

	var (
		rawOutput string
		err       error
	)
	if streamer, ok := g.model.(QuestionModelStreamer); ok {
		var streamedOutput strings.Builder
		rawOutput, err = streamer.GenerateStream(ctx, retrievalQueryPlannerSystemPrompt, prompt, func(delta string) {
			streamedOutput.WriteString(delta)
			if onDelta != nil {
				onDelta(delta)
			}
		})
		if strings.TrimSpace(rawOutput) == "" {
			rawOutput = streamedOutput.String()
		}
	} else {
		rawOutput, err = g.model.Generate(ctx, retrievalQueryPlannerSystemPrompt, prompt)
	}
	if err != nil {
		trace.RawOutput = strings.TrimSpace(err.Error())
		return trace
	}

	trimmedOutput := strings.TrimSpace(rawOutput)
	trace.RawOutput = trimmedOutput
	plannedQuery := parseRetrievalQueryPlannerOutput(trimmedOutput)
	if plannedQuery == "" {
		return trace
	}

	trace.Strategy = "agent"
	trace.FinalQuery = plannedQuery
	return trace
}

func buildRetrievalQueryPlannerPrompt(lead model.Lead, topicKeys []string, resumeText string, jdText string) string {
	return strings.Join([]string{
		"<query_task>",
		"Create one retrieval query for interview question generation.",
		"Return JSON only: {\"query\":\"...\"}.",
		"</query_task>",
		"",
		"<lead>",
		fmt.Sprintf("company: %s", strings.TrimSpace(lead.Company)),
		fmt.Sprintf("position: %s", strings.TrimSpace(lead.Position)),
		fmt.Sprintf("topic_keys: %s", strings.Join(topicKeys, ", ")),
		"</lead>",
		"",
		"<resume>",
		limitPromptInput(resumeText, 3200),
		"</resume>",
		"",
		"<job_description>",
		limitPromptInput(jdText, 3200),
		"</job_description>",
		"",
		"<query_quality_checklist>",
		"- should include role/domain signal (e.g. backend, distributed systems, llm, rag).",
		"- should include at least one JD-critical keyword if JD exists.",
		"- should include at least one candidate-skill keyword if resume exists.",
		"- avoid punctuation-heavy natural language sentences; prefer keyword-style query.",
		"</query_quality_checklist>",
	}, "\n")
}

func buildInterviewQuestionsTool(questionCount int) agentprovider.Tool {
	if questionCount <= 0 {
		questionCount = defaultQuestionCount
	}
	return agentprovider.Tool{
		Type:        "function",
		Name:        "emit_interview_questions",
		Description: "Return interview questions in strict JSON arguments. Do not include markdown.",
		Strict:      true,
		Parameters: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"questions": map[string]any{
					"type":     "array",
					"minItems": questionCount,
					"maxItems": questionCount,
					"items": map[string]any{
						"type":                 "object",
						"additionalProperties": false,
						"properties": map[string]any{
							"id": map[string]any{
								"type": "integer",
							},
							"type": map[string]any{
								"type": "string",
							},
							"content": map[string]any{
								"type": "string",
							},
							"expected_points": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
							},
							"context_sources": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
							},
						},
						"required": []string{"id", "type", "content", "expected_points", "context_sources"},
					},
				},
			},
			"required": []string{"questions"},
		},
	}
}

func parseRetrievalQueryPlannerOutput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	tryDecode := func(payload string) string {
		type queryPayload struct {
			Query       string `json:"query"`
			SearchQuery string `json:"search_query"`
		}
		var parsed queryPayload
		if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
			return ""
		}
		if query := normalizeRetrievalQuery(parsed.Query); query != "" {
			return query
		}
		return normalizeRetrievalQuery(parsed.SearchQuery)
	}

	if parsed := tryDecode(trimmed); parsed != "" {
		return parsed
	}

	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			body := strings.Join(lines[1:len(lines)-1], "\n")
			if parsed := tryDecode(strings.TrimSpace(body)); parsed != "" {
				return parsed
			}
		}
	}

	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		query := normalizeRetrievalQuery(line)
		if query == "" {
			continue
		}
		lower := strings.ToLower(query)
		if strings.HasPrefix(lower, "query:") {
			query = strings.TrimSpace(query[len("query:"):])
			query = normalizeRetrievalQuery(query)
		}
		if query != "" {
			return query
		}
	}

	return ""
}

func normalizeRetrievalQuery(query string) string {
	normalized := strings.TrimSpace(query)
	normalized = strings.Trim(normalized, "\"`")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func fallbackRetrievalQuery(lead model.Lead, topicKeys []string) string {
	parts := cleanStrings([]string{
		strings.TrimSpace(lead.Position),
		strings.TrimSpace(lead.Company),
		strings.Join(topicKeys, " "),
		"interview preparation",
	})
	query := strings.TrimSpace(strings.Join(parts, " "))
	if query == "" {
		return "interview preparation"
	}
	return query
}

func limitPromptInput(content string, maxRunes int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "(not provided)"
	}
	if maxRunes <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "\n...(truncated)"
}

func excerptForTrace(content string, maxRunes int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	if maxRunes <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func (g *QuestionGenerator) emitProgress(reporter GenerationProgressReporter, event GenerationProgressEvent) {
	if reporter == nil {
		return
	}
	reporter(event)
}

func (g *QuestionGenerator) emitTextDeltas(
	ctx context.Context,
	reporter GenerationProgressReporter,
	stage string,
	content string,
	message string,
	trace GenerationTrace,
) error {
	if reporter == nil {
		return nil
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	chunks := splitTextIntoChunks(trimmed, 72)
	for _, chunk := range chunks {
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		reporter(GenerationProgressEvent{
			Stage:   stage,
			Status:  GenerationProgressProgress,
			Message: message,
			Delta:   chunk,
			Trace:   cloneGenerationTrace(trace),
		})
	}
	return nil
}

func splitTextIntoChunks(text string, chunkSize int) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return []string{}
	}
	if chunkSize <= 0 {
		chunkSize = 72
	}
	runes := []rune(trimmed)
	out := make([]string, 0, len(runes)/chunkSize+1)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

func cloneGenerationTrace(trace GenerationTrace) *GenerationTrace {
	clone := trace
	clone.InputSnapshot.TopicKeys = append([]string{}, trace.InputSnapshot.TopicKeys...)
	clone.RetrievalResults.Sources = append([]string{}, trace.RetrievalResults.Sources...)
	clone.PromptSections = append([]PromptSection{}, trace.PromptSections...)
	return &clone
}
