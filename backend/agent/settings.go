package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"trace2offer/backend/agent/provider/openai"
)

var ErrSettingsStoreUnavailable = errors.New("settings store is unavailable")

// SettingsValidationError means runtime settings are invalid.
type SettingsValidationError struct {
	Field   string
	Message string
}

func (e *SettingsValidationError) Error() string {
	if e == nil {
		return "invalid runtime settings"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Field) != "" {
		return e.Field + " is invalid"
	}
	return "invalid runtime settings"
}

func IsSettingsValidationError(err error) bool {
	var validationErr *SettingsValidationError
	return errors.As(err, &validationErr)
}

// RuntimeSettings holds editable runtime config.
type RuntimeSettings struct {
	Model                string `json:"model"`
	MaxSteps             int    `json:"max_steps"`
	SystemPrompt         string `json:"system_prompt"`
	OpenAIAPIFormat      string `json:"openai_api_format"`
	OpenAIBaseURL        string `json:"openai_base_url"`
	OpenAITimeoutSeconds int    `json:"openai_timeout_seconds"`
	OpenAIAPIKey         string `json:"openai_api_key"`
}

// RuntimeSettingsView is safe payload returned to frontend.
type RuntimeSettingsView struct {
	Model                string `json:"model"`
	MaxSteps             int    `json:"max_steps"`
	SystemPrompt         string `json:"system_prompt"`
	OpenAIAPIFormat      string `json:"openai_api_format"`
	OpenAIBaseURL        string `json:"openai_base_url"`
	OpenAITimeoutSeconds int    `json:"openai_timeout_seconds"`
	HasOpenAIAPIKey      bool   `json:"has_openai_api_key"`
}

// RuntimeSettingsPatch is partial update payload from API.
type RuntimeSettingsPatch struct {
	Model                *string `json:"model"`
	MaxSteps             *int    `json:"max_steps"`
	SystemPrompt         *string `json:"system_prompt"`
	OpenAIAPIFormat      *string `json:"openai_api_format"`
	OpenAIBaseURL        *string `json:"openai_base_url"`
	OpenAITimeoutSeconds *int    `json:"openai_timeout_seconds"`
	OpenAIAPIKey         *string `json:"openai_api_key"`
}

func (s RuntimeSettings) View() RuntimeSettingsView {
	format := openai.NormalizeAPIFormat(s.OpenAIAPIFormat)
	if format == "" {
		format = openai.DefaultAPIFormat
	}
	return RuntimeSettingsView{
		Model:                strings.TrimSpace(s.Model),
		MaxSteps:             s.MaxSteps,
		SystemPrompt:         s.SystemPrompt,
		OpenAIAPIFormat:      format,
		OpenAIBaseURL:        strings.TrimSpace(s.OpenAIBaseURL),
		OpenAITimeoutSeconds: s.OpenAITimeoutSeconds,
		HasOpenAIAPIKey:      strings.TrimSpace(s.OpenAIAPIKey) != "",
	}
}

func (p RuntimeSettingsPatch) Apply(base RuntimeSettings) RuntimeSettings {
	next := base
	prevFormat := openai.NormalizeAPIFormat(base.OpenAIAPIFormat)
	if prevFormat == "" {
		prevFormat = openai.DefaultAPIFormat
	}
	if p.Model != nil {
		next.Model = strings.TrimSpace(*p.Model)
	}
	if p.MaxSteps != nil {
		next.MaxSteps = *p.MaxSteps
	}
	if p.SystemPrompt != nil {
		next.SystemPrompt = *p.SystemPrompt
	}
	if p.OpenAIAPIFormat != nil {
		nextFormat := openai.NormalizeAPIFormat(*p.OpenAIAPIFormat)
		if nextFormat == "" && strings.TrimSpace(*p.OpenAIAPIFormat) == "" {
			nextFormat = openai.DefaultAPIFormat
		}
		next.OpenAIAPIFormat = nextFormat

		if p.OpenAIBaseURL == nil {
			currentBaseURL := strings.TrimSpace(next.OpenAIBaseURL)
			prevDefaultBaseURL := openai.DefaultBaseURLForFormat(prevFormat)
			if currentBaseURL == "" || currentBaseURL == prevDefaultBaseURL {
				next.OpenAIBaseURL = openai.DefaultBaseURLForFormat(nextFormat)
			}
		}
	}
	if p.OpenAIBaseURL != nil {
		next.OpenAIBaseURL = strings.TrimSpace(*p.OpenAIBaseURL)
	}
	if p.OpenAITimeoutSeconds != nil {
		next.OpenAITimeoutSeconds = *p.OpenAITimeoutSeconds
	}
	if p.OpenAIAPIKey != nil {
		next.OpenAIAPIKey = strings.TrimSpace(*p.OpenAIAPIKey)
	}
	return next
}

func mergeRuntimeSettings(defaults RuntimeSettings, loaded RuntimeSettings) RuntimeSettings {
	merged := RuntimeSettings{
		Model:                strings.TrimSpace(defaults.Model),
		MaxSteps:             defaults.MaxSteps,
		SystemPrompt:         defaults.SystemPrompt,
		OpenAIAPIFormat:      openai.NormalizeAPIFormat(defaults.OpenAIAPIFormat),
		OpenAIBaseURL:        strings.TrimSpace(defaults.OpenAIBaseURL),
		OpenAITimeoutSeconds: defaults.OpenAITimeoutSeconds,
		OpenAIAPIKey:         strings.TrimSpace(defaults.OpenAIAPIKey),
	}

	if model := strings.TrimSpace(loaded.Model); model != "" {
		merged.Model = model
	}
	if loaded.MaxSteps > 0 {
		merged.MaxSteps = loaded.MaxSteps
	}
	// System prompt allows empty string.
	merged.SystemPrompt = loaded.SystemPrompt

	if format := openai.NormalizeAPIFormat(loaded.OpenAIAPIFormat); format != "" {
		merged.OpenAIAPIFormat = format
	}
	if baseURL := strings.TrimSpace(loaded.OpenAIBaseURL); baseURL != "" {
		merged.OpenAIBaseURL = baseURL
	}
	if loaded.OpenAITimeoutSeconds > 0 {
		merged.OpenAITimeoutSeconds = loaded.OpenAITimeoutSeconds
	}
	if apiKey := strings.TrimSpace(loaded.OpenAIAPIKey); apiKey != "" {
		merged.OpenAIAPIKey = apiKey
	}

	if merged.OpenAIAPIFormat == "" {
		merged.OpenAIAPIFormat = openai.DefaultAPIFormat
	}
	if merged.OpenAIBaseURL == "" {
		merged.OpenAIBaseURL = openai.DefaultBaseURLForFormat(merged.OpenAIAPIFormat)
	}

	return merged
}

func validateRuntimeSettings(settings RuntimeSettings) error {
	if strings.TrimSpace(settings.Model) == "" {
		return &SettingsValidationError{Field: "model", Message: "model is required"}
	}
	if settings.MaxSteps <= 0 {
		return &SettingsValidationError{Field: "max_steps", Message: "max_steps must be greater than 0"}
	}
	if !openai.IsSupportedAPIFormat(settings.OpenAIAPIFormat) {
		return &SettingsValidationError{Field: "openai_api_format", Message: "openai_api_format must be one of: responses, chat_completions"}
	}
	if strings.TrimSpace(settings.OpenAIBaseURL) == "" {
		return &SettingsValidationError{Field: "openai_base_url", Message: "openai_base_url is required"}
	}
	if settings.OpenAITimeoutSeconds <= 0 {
		return &SettingsValidationError{Field: "openai_timeout_seconds", Message: "openai_timeout_seconds must be greater than 0"}
	}
	if strings.TrimSpace(settings.OpenAIAPIKey) == "" {
		return &SettingsValidationError{Field: "openai_api_key", Message: "openai_api_key is required"}
	}
	return nil
}

type runtimeSettingsStore struct {
	path string
	mu   sync.Mutex
}

func newRuntimeSettingsStore(path string) (*runtimeSettingsStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, &SettingsValidationError{Field: "settings_path", Message: "settings path is required"}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create runtime settings dir: %w", err)
	}

	return &runtimeSettingsStore{path: path}, nil
}

func (s *runtimeSettingsStore) LoadOrCreate(defaults RuntimeSettings) (RuntimeSettings, error) {
	if s == nil {
		return RuntimeSettings{}, ErrSettingsStoreUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	loaded, err := s.loadLocked()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			merged := mergeRuntimeSettings(defaults, RuntimeSettings{})
			if err := s.saveLocked(merged); err != nil {
				return RuntimeSettings{}, err
			}
			return merged, nil
		}
		return RuntimeSettings{}, err
	}

	merged := mergeRuntimeSettings(defaults, loaded)
	if !runtimeSettingsEqual(merged, loaded) {
		if err := s.saveLocked(merged); err != nil {
			return RuntimeSettings{}, err
		}
	}

	return merged, nil
}

func (s *runtimeSettingsStore) Save(settings RuntimeSettings) error {
	if s == nil {
		return ErrSettingsStoreUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(settings)
}

func (s *runtimeSettingsStore) loadLocked() (RuntimeSettings, error) {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		return RuntimeSettings{}, err
	}
	if len(payload) == 0 {
		return RuntimeSettings{}, nil
	}

	var parsed RuntimeSettings
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return RuntimeSettings{}, fmt.Errorf("decode runtime settings file: %w", err)
	}
	return parsed, nil
}

func (s *runtimeSettingsStore) saveLocked(settings RuntimeSettings) error {
	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode runtime settings file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return fmt.Errorf("write temp runtime settings file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace runtime settings file: %w", err)
	}

	return nil
}

func runtimeSettingsEqual(a RuntimeSettings, b RuntimeSettings) bool {
	return strings.TrimSpace(a.Model) == strings.TrimSpace(b.Model) &&
		a.MaxSteps == b.MaxSteps &&
		a.SystemPrompt == b.SystemPrompt &&
		openai.NormalizeAPIFormat(a.OpenAIAPIFormat) == openai.NormalizeAPIFormat(b.OpenAIAPIFormat) &&
		strings.TrimSpace(a.OpenAIBaseURL) == strings.TrimSpace(b.OpenAIBaseURL) &&
		a.OpenAITimeoutSeconds == b.OpenAITimeoutSeconds &&
		strings.TrimSpace(a.OpenAIAPIKey) == strings.TrimSpace(b.OpenAIAPIKey)
}

func defaultRuntimeSettings() RuntimeSettings {
	defaults := DefaultConfig()
	return RuntimeSettings{
		Model:                defaults.Model,
		MaxSteps:             defaults.MaxSteps,
		SystemPrompt:         defaults.SystemPrompt,
		OpenAIAPIFormat:      openai.DefaultAPIFormat,
		OpenAIBaseURL:        openai.DefaultBaseURLForFormat(openai.DefaultAPIFormat),
		OpenAITimeoutSeconds: int((60 * time.Second) / time.Second),
		OpenAIAPIKey:         "",
	}
}
