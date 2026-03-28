package prep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"trace2offer/backend/internal/model"
)

type Service struct {
	config          Config
	topicStore      *TopicStore
	knowledgeStore  *KnowledgeStore
	contextResolver *ContextResolver
	sessionStore    *SessionStore
	indexStore      *IndexStore
	embedProvider   EmbeddingProvider
	questionModel   QuestionModel
	questionGen     *QuestionGenerator
	retrievalEngine *RetrievalEngine
	ingestion       *IngestionService
}

type ServiceOption func(*Service)

func WithIndexStore(store *IndexStore) ServiceOption {
	return func(service *Service) {
		if service == nil {
			return
		}
		service.indexStore = store
	}
}

func WithEmbeddingProvider(provider EmbeddingProvider) ServiceOption {
	return func(service *Service) {
		if service == nil {
			return
		}
		service.embedProvider = provider
	}
}

func WithQuestionModel(model QuestionModel) ServiceOption {
	return func(service *Service) {
		if service == nil {
			return
		}
		service.questionModel = model
	}
}

func WithQuestionGenerator(generator *QuestionGenerator) ServiceOption {
	return func(service *Service) {
		if service == nil {
			return
		}
		service.questionGen = generator
	}
}

func NewService(config Config, options ...ServiceOption) (*Service, error) {
	normalized := config
	normalized.DataDir = filepath.Clean(strings.TrimSpace(normalized.DataDir))
	if normalized.DefaultQuestionCount <= 0 {
		normalized.DefaultQuestionCount = defaultQuestionCount
	}
	if strings.TrimSpace(normalized.EmbeddingProvider) == "" {
		normalized.EmbeddingProvider = "huggingface"
	}
	if strings.TrimSpace(normalized.EmbeddingModel) == "" {
		normalized.EmbeddingModel = defaultHFModel
	}
	if normalized.EmbeddingDimension <= 0 {
		normalized.EmbeddingDimension = 1024
	}
	if strings.TrimSpace(normalized.IndexDBPath) == "" {
		normalized.IndexDBPath = filepath.Join(normalized.DataDir, "prep_index.sqlite")
	}
	if len(normalized.SupportedScopes) == 0 {
		normalized.SupportedScopes = DefaultSupportedScopes()
	}
	if err := normalized.Validate(); err != nil {
		return nil, err
	}

	service := &Service{config: normalized}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	if !normalized.Enabled {
		return service, nil
	}
	if err := service.initializeStorage(); err != nil {
		return nil, err
	}
	if err := service.initializeIndexing(); err != nil {
		return nil, err
	}

	topicStore, err := NewTopicStore(filepath.Join(normalized.DataDir, "topic_catalog.json"))
	if err != nil {
		return nil, err
	}
	knowledgeStore, err := NewKnowledgeStore(filepath.Join(normalized.DataDir, "knowledge"))
	if err != nil {
		return nil, err
	}
	service.topicStore = topicStore
	service.knowledgeStore = knowledgeStore
	sessionStore, err := NewSessionStore(filepath.Join(normalized.DataDir, "sessions"))
	if err != nil {
		return nil, err
	}
	service.sessionStore = sessionStore
	service.contextResolver = NewContextResolver(normalized.DataDir, topicStore, knowledgeStore)
	service.retrievalEngine = NewRetrievalEngine(service.indexStore, service.embedProvider)
	if service.questionGen == nil {
		service.questionGen = NewQuestionGenerator(
			service.contextResolver,
			service.retrievalEngine,
			service.sessionStore,
			service.embedProvider,
			service.questionModel,
			normalized.DefaultQuestionCount,
		)
	}
	if service.indexStore != nil && service.embedProvider != nil {
		ingestion, err := NewIngestionService(normalized.DataDir, IngestionDependencies{
			IndexStore:        service.indexStore,
			EmbeddingProvider: service.embedProvider,
			ChunkConfig: ChunkConfig{
				ChunkSizeTokens: defaultChunkSizeTokens,
				OverlapTokens:   defaultChunkOverlap,
			},
		})
		if err != nil {
			return nil, err
		}
		service.ingestion = ingestion
	}
	return service, nil
}

func (s *Service) GetMeta() Meta {
	if s == nil {
		return Meta{
			Enabled:              false,
			DefaultQuestionCount: defaultQuestionCount,
			SupportedScopes:      scopeNames(DefaultSupportedScopes()),
		}
	}
	return Meta{
		Enabled:              s.config.Enabled,
		DefaultQuestionCount: s.config.DefaultQuestionCount,
		SupportedScopes:      scopeNames(s.config.SupportedScopes),
	}
}

func (s *Service) initializeStorage() error {
	if s == nil {
		return fmt.Errorf("prep service is nil")
	}

	rootDir := strings.TrimSpace(s.config.DataDir)
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return fmt.Errorf("create prep data dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "knowledge"), 0o755); err != nil {
		return fmt.Errorf("create prep knowledge dir: %w", err)
	}
	for _, scope := range DefaultSupportedScopes() {
		if err := os.MkdirAll(filepath.Join(rootDir, "knowledge", string(scope)), 0o755); err != nil {
			return fmt.Errorf("create prep knowledge %s dir: %w", scope, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "sessions"), 0o755); err != nil {
		return fmt.Errorf("create prep sessions dir: %w", err)
	}

	topicCatalogPath := filepath.Join(rootDir, "topic_catalog.json")
	if err := ensureTopicCatalogFile(topicCatalogPath); err != nil {
		return err
	}
	return nil
}

func (s *Service) ListTopics() ([]Topic, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if s.topicStore == nil {
		return nil, ErrTopicStoreUnavailable
	}
	return s.topicStore.List(), nil
}

func (s *Service) CreateTopic(input TopicCreateInput) (Topic, error) {
	if err := s.ensureEnabled(); err != nil {
		return Topic{}, err
	}
	if s.topicStore == nil {
		return Topic{}, ErrTopicStoreUnavailable
	}
	return s.topicStore.Create(input)
}

func (s *Service) UpdateTopic(key string, patch TopicPatchInput) (Topic, bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return Topic{}, false, err
	}
	if s.topicStore == nil {
		return Topic{}, false, ErrTopicStoreUnavailable
	}
	return s.topicStore.Update(key, patch)
}

func (s *Service) DeleteTopic(key string) (bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return false, err
	}
	if s.topicStore == nil {
		return false, ErrTopicStoreUnavailable
	}
	return s.topicStore.Delete(key)
}

func (s *Service) ListKnowledgeDocuments(scope string, scopeID string) ([]KnowledgeDocument, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if s.knowledgeStore == nil {
		return nil, ErrKnowledgeStoreUnavailable
	}
	return s.knowledgeStore.List(scope, scopeID)
}

func (s *Service) CreateKnowledgeDocument(scope string, scopeID string, input KnowledgeDocumentCreateInput) (KnowledgeDocument, error) {
	if err := s.ensureEnabled(); err != nil {
		return KnowledgeDocument{}, err
	}
	if s.knowledgeStore == nil {
		return KnowledgeDocument{}, ErrKnowledgeStoreUnavailable
	}
	return s.knowledgeStore.Create(scope, scopeID, input)
}

func (s *Service) UpdateKnowledgeDocument(scope string, scopeID string, filename string, input KnowledgeDocumentUpdateInput) (KnowledgeDocument, bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return KnowledgeDocument{}, false, err
	}
	if s.knowledgeStore == nil {
		return KnowledgeDocument{}, false, ErrKnowledgeStoreUnavailable
	}
	return s.knowledgeStore.Update(scope, scopeID, filename, input)
}

func (s *Service) DeleteKnowledgeDocument(scope string, scopeID string, filename string) (bool, error) {
	if err := s.ensureEnabled(); err != nil {
		return false, err
	}
	if s.knowledgeStore == nil {
		return false, ErrKnowledgeStoreUnavailable
	}
	return s.knowledgeStore.Delete(scope, scopeID, filename)
}

func (s *Service) GetLeadContextPreview(lead model.Lead) (LeadContextPreview, error) {
	if err := s.ensureEnabled(); err != nil {
		return LeadContextPreview{}, err
	}
	if s.contextResolver == nil {
		return LeadContextPreview{}, ErrContextResolverUnavailable
	}
	return s.contextResolver.Resolve(lead)
}

func (s *Service) GetIndexStatus() (IndexStatus, error) {
	if err := s.ensureEnabled(); err != nil {
		return IndexStatus{}, err
	}
	if s.indexStore == nil {
		return IndexStatus{}, ErrIndexStoreUnavailable
	}

	embeddingProvider := s.config.EmbeddingProvider
	embeddingModel := s.config.EmbeddingModel
	if infoProvider, ok := s.embedProvider.(EmbeddingProviderInfo); ok {
		embeddingProvider = infoProvider.Name()
		embeddingModel = infoProvider.Model()
	}
	return s.indexStore.GetStatus(embeddingProvider, embeddingModel)
}

func (s *Service) ListIndexDocuments() ([]Document, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if s.indexStore == nil {
		return nil, ErrIndexStoreUnavailable
	}
	return s.indexStore.ListDocuments()
}

func (s *Service) ListIndexChunks(documentID string, limit int) ([]IndexedChunk, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if s.indexStore == nil {
		return nil, ErrIndexStoreUnavailable
	}
	return s.indexStore.ListChunks(documentID, limit)
}

func (s *Service) initializeIndexing() error {
	if s == nil {
		return fmt.Errorf("prep service is nil")
	}

	if s.indexStore == nil {
		store, err := NewIndexStore(s.config.IndexDBPath)
		if err != nil {
			return err
		}
		s.indexStore = store
	}

	if s.embedProvider != nil {
		return nil
	}

	providerName := strings.ToLower(strings.TrimSpace(s.config.EmbeddingProvider))
	switch providerName {
	case "", "huggingface":
		provider, err := NewHuggingFaceEmbeddingProvider(HuggingFaceEmbeddingProviderConfig{
			APIKey:    s.config.HuggingFaceAPIKey,
			Model:     s.config.EmbeddingModel,
			BaseURL:   s.config.HuggingFaceBaseURL,
			Dimension: s.config.EmbeddingDimension,
		})
		if err != nil {
			return err
		}
		s.embedProvider = provider
		return nil
	default:
		return &ValidationError{Field: "embedding_provider", Message: fmt.Sprintf("unsupported embedding provider: %s", providerName)}
	}
}

func (s *Service) CreateSession(ctx context.Context, lead model.Lead, input CreateSessionInput) (Session, error) {
	if err := s.ensureEnabled(); err != nil {
		return Session{}, err
	}
	if s.sessionStore == nil {
		return Session{}, ErrSessionStoreUnavailable
	}
	if s.questionGen == nil {
		return Session{}, ErrQuestionGeneratorUnavailable
	}
	if err := s.ensureEmbeddingReady(); err != nil {
		return Session{}, err
	}

	leadID := strings.TrimSpace(lead.ID)
	if leadID == "" {
		return Session{}, &ValidationError{Field: "lead_id", Message: "lead_id is required"}
	}
	inputLeadID := strings.TrimSpace(input.LeadID)
	if inputLeadID == "" {
		input.LeadID = leadID
	} else if inputLeadID != leadID {
		return Session{}, &ValidationError{Field: "lead_id", Message: "lead_id does not match selected lead"}
	}
	generated, err := s.questionGen.GenerateWithContext(ctx, GenerationConfig{
		Lead:            lead,
		LeadID:          input.LeadID,
		QuestionCount:   input.QuestionCount,
		IncludeResume:   input.IncludeResume,
		IncludeLeadDocs: input.IncludeLeadDocs,
	})
	if err != nil {
		return Session{}, err
	}
	if generated == nil || generated.Session == nil {
		return Session{}, fmt.Errorf("question generation returned empty session")
	}
	return *generated.Session, nil
}

func (s *Service) CreateSessionWithProgress(
	ctx context.Context,
	lead model.Lead,
	input CreateSessionInput,
	reporter GenerationProgressReporter,
) (Session, error) {
	if err := s.ensureEnabled(); err != nil {
		return Session{}, err
	}
	if s.sessionStore == nil {
		return Session{}, ErrSessionStoreUnavailable
	}
	if s.questionGen == nil {
		return Session{}, ErrQuestionGeneratorUnavailable
	}
	if err := s.ensureEmbeddingReady(); err != nil {
		return Session{}, err
	}

	leadID := strings.TrimSpace(lead.ID)
	if leadID == "" {
		return Session{}, &ValidationError{Field: "lead_id", Message: "lead_id is required"}
	}
	inputLeadID := strings.TrimSpace(input.LeadID)
	if inputLeadID == "" {
		input.LeadID = leadID
	} else if inputLeadID != leadID {
		return Session{}, &ValidationError{Field: "lead_id", Message: "lead_id does not match selected lead"}
	}
	generated, err := s.questionGen.GenerateWithProgress(ctx, GenerationConfig{
		Lead:            lead,
		LeadID:          input.LeadID,
		QuestionCount:   input.QuestionCount,
		IncludeResume:   input.IncludeResume,
		IncludeLeadDocs: input.IncludeLeadDocs,
	}, reporter)
	if err != nil {
		return Session{}, err
	}
	if generated == nil || generated.Session == nil {
		return Session{}, fmt.Errorf("question generation returned empty session")
	}
	return *generated.Session, nil
}

func (s *Service) GetSession(sessionID string) (Session, error) {
	if err := s.ensureEnabled(); err != nil {
		return Session{}, err
	}
	if s.sessionStore == nil {
		return Session{}, ErrSessionStoreUnavailable
	}

	session, err := s.sessionStore.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, err
	}
	return *session, nil
}

func (s *Service) GenerateQuestions(ctx context.Context, lead model.Lead, input CreateSessionInput) ([]Question, GenerationTrace, []ContextSource, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, GenerationTrace{}, nil, err
	}
	if s.questionGen == nil {
		return nil, GenerationTrace{}, nil, ErrQuestionGeneratorUnavailable
	}
	if err := s.ensureEmbeddingReady(); err != nil {
		return nil, GenerationTrace{}, nil, err
	}

	generated, err := s.questionGen.GenerateWithContext(ctx, GenerationConfig{
		Lead:            lead,
		LeadID:          input.LeadID,
		QuestionCount:   input.QuestionCount,
		IncludeResume:   input.IncludeResume,
		IncludeLeadDocs: input.IncludeLeadDocs,
	})
	if err != nil {
		return nil, GenerationTrace{}, nil, err
	}
	if generated == nil || generated.Session == nil {
		return nil, GenerationTrace{}, nil, fmt.Errorf("question generation returned empty session")
	}
	trace := GenerationTrace{}
	if generated.Session.GenerationTrace != nil {
		trace = *generated.Session.GenerationTrace
	}
	return generated.Session.Questions, trace, generated.Session.Sources, nil
}

func (s *Service) RebuildIndex(scope string, scopeID string) (*IndexRunSummary, error) {
	return s.RebuildIndexWithMode(scope, scopeID, RebuildModeIncremental)
}

func (s *Service) RebuildIndexWithMode(scope string, scopeID string, mode string) (*IndexRunSummary, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if s.ingestion == nil {
		return nil, ErrIngestionUnavailable
	}
	if err := s.ensureEmbeddingReady(); err != nil {
		return nil, err
	}
	return s.ingestion.IngestWithMode(scope, scopeID, mode)
}

func (s *Service) SaveDraftAnswers(sessionID string, answers []Answer) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if s.sessionStore == nil {
		return ErrSessionStoreUnavailable
	}

	session, err := s.sessionStore.Get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrSessionNotFound
		}
		return err
	}
	if strings.TrimSpace(session.Status) != PrepSessionStatusDraft {
		return &ValidationError{Field: "status", Message: "only draft session can save answers"}
	}

	questionIDs := make(map[int]struct{}, len(session.Questions))
	for _, question := range session.Questions {
		if question.ID > 0 {
			questionIDs[question.ID] = struct{}{}
		}
		if question.QuestionID > 0 {
			questionIDs[question.QuestionID] = struct{}{}
		}
	}
	for _, answer := range answers {
		if answer.QuestionID <= 0 {
			return &ValidationError{Field: "answers.question_id", Message: "question_id must be greater than 0"}
		}
		if _, ok := questionIDs[answer.QuestionID]; !ok {
			return &ValidationError{Field: "answers.question_id", Message: fmt.Sprintf("question_id %d not found in session", answer.QuestionID)}
		}
	}
	if err := s.sessionStore.UpdateAnswers(sessionID, answers); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrSessionNotFound
		}
		return err
	}
	return nil
}

func (s *Service) Search(input SearchConfig) (*SearchResult, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if s.retrievalEngine == nil {
		return nil, ErrIndexStoreUnavailable
	}
	if err := s.ensureEmbeddingReady(); err != nil {
		return nil, err
	}
	return s.retrievalEngine.Search(input.Query, input)
}

func (s *Service) ensureEmbeddingReady() error {
	if s == nil || s.embedProvider == nil {
		return ErrEmbedProviderUnavailable
	}
	if validator, ok := s.embedProvider.(EmbeddingProviderValidator); ok {
		return validator.Validate()
	}
	return nil
}

func (s *Service) ensureEnabled() error {
	if s == nil {
		return ErrPrepDisabled
	}
	if !s.config.Enabled {
		return ErrPrepDisabled
	}
	return nil
}

func ensureTopicCatalogFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat prep topic catalog file: %w", err)
	}

	payload, err := json.MarshalIndent(struct {
		Topics []struct{} `json:"topics"`
	}{
		Topics: []struct{}{},
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode prep topic catalog: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write prep topic catalog file: %w", err)
	}
	return nil
}
