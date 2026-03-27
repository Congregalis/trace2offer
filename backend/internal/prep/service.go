package prep

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Service struct {
	config Config
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
	if err := os.MkdirAll(filepath.Join(rootDir, "sessions"), 0o755); err != nil {
		return fmt.Errorf("create prep sessions dir: %w", err)
	}

	topicCatalogPath := filepath.Join(rootDir, "topic_catalog.json")
	if err := ensureTopicCatalogFile(topicCatalogPath); err != nil {
		return err
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
