package prep

import (
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
}

func NewService(config Config) (*Service, error) {
	normalized := config
	normalized.DataDir = filepath.Clean(strings.TrimSpace(normalized.DataDir))
	if normalized.DefaultQuestionCount <= 0 {
		normalized.DefaultQuestionCount = defaultQuestionCount
	}
	if len(normalized.SupportedScopes) == 0 {
		normalized.SupportedScopes = DefaultSupportedScopes()
	}
	if err := normalized.Validate(); err != nil {
		return nil, err
	}

	service := &Service{config: normalized}
	if !normalized.Enabled {
		return service, nil
	}
	if err := service.initializeStorage(); err != nil {
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
	service.contextResolver = NewContextResolver(normalized.DataDir, topicStore, knowledgeStore)
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
