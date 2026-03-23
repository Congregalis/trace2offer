package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"trace2offer/backend/agent/tool"
	"trace2offer/backend/internal/candidate"
	"trace2offer/backend/internal/lead"
)

var ErrRuntimeUnavailable = errors.New("agent runtime is unavailable")

// ManagedRuntimeConfig wires static dependencies and settings persistence.
type ManagedRuntimeConfig struct {
	SettingsPath        string
	SessionDataPath     string
	MemoryDataPath      string
	UserProfileDataPath string
	LeadManager         lead.Manager
	CandidateManager    candidate.Manager
	StatsProvider       tool.StatsSummaryProvider
	Defaults            RuntimeSettings
}

// ManagedRuntime wraps runtime with hot-reloadable settings.
type ManagedRuntime struct {
	mu           sync.RWMutex
	store        *runtimeSettingsStore
	settings     RuntimeSettings
	runtime      *Runtime
	userProfiles *UserProfileManager

	sessionDataPath  string
	memoryDataPath   string
	leadManager      lead.Manager
	candidateManager candidate.Manager
	statsProvider    tool.StatsSummaryProvider
}

func NewManagedRuntime(config ManagedRuntimeConfig) (*ManagedRuntime, error) {
	if strings.TrimSpace(config.SettingsPath) == "" {
		return nil, &SettingsValidationError{Field: "settings_path", Message: "settings path is required"}
	}
	if strings.TrimSpace(config.SessionDataPath) == "" {
		return nil, &SettingsValidationError{Field: "session_data_path", Message: "session data path is required"}
	}
	if strings.TrimSpace(config.MemoryDataPath) == "" {
		return nil, &SettingsValidationError{Field: "memory_data_path", Message: "memory data path is required"}
	}
	if strings.TrimSpace(config.UserProfileDataPath) == "" {
		return nil, &SettingsValidationError{Field: "user_profile_data_path", Message: "user profile data path is required"}
	}
	if config.LeadManager == nil {
		return nil, &SettingsValidationError{Field: "lead_manager", Message: "lead manager is required"}
	}

	profileStore, err := NewFileUserProfileStore(config.UserProfileDataPath)
	if err != nil {
		return nil, err
	}
	profileManager := NewUserProfileManager(profileStore)

	defaults := mergeRuntimeSettings(defaultRuntimeSettings(), config.Defaults)
	store, err := newRuntimeSettingsStore(config.SettingsPath)
	if err != nil {
		return nil, err
	}
	settings, err := store.LoadOrCreate(defaults)
	if err != nil {
		return nil, err
	}
	if err := validateRuntimeSettings(settings); err != nil {
		return nil, err
	}

	manager := &ManagedRuntime{
		store:            store,
		settings:         settings,
		sessionDataPath:  config.SessionDataPath,
		memoryDataPath:   config.MemoryDataPath,
		leadManager:      config.LeadManager,
		candidateManager: config.CandidateManager,
		statsProvider:    config.StatsProvider,
		userProfiles:     profileManager,
	}

	runtime, err := manager.buildRuntime(settings)
	if err != nil {
		return nil, err
	}
	manager.runtime = runtime
	return manager, nil
}

func (m *ManagedRuntime) Run(ctx context.Context, request Request) (Response, error) {
	if m == nil {
		return Response{}, ErrRuntimeUnavailable
	}

	m.mu.RLock()
	runtime := m.runtime
	m.mu.RUnlock()
	if runtime == nil {
		return Response{}, ErrRuntimeUnavailable
	}

	return runtime.Run(ctx, request)
}

func (m *ManagedRuntime) CreateSession(_ context.Context) (SessionView, error) {
	if m == nil {
		return SessionView{}, ErrRuntimeUnavailable
	}

	m.mu.RLock()
	runtime := m.runtime
	m.mu.RUnlock()
	if runtime == nil {
		return SessionView{}, ErrRuntimeUnavailable
	}

	return runtime.CreateSession("")
}

func (m *ManagedRuntime) GetSession(_ context.Context, sessionID string) (SessionView, error) {
	if m == nil {
		return SessionView{}, ErrRuntimeUnavailable
	}

	m.mu.RLock()
	runtime := m.runtime
	m.mu.RUnlock()
	if runtime == nil {
		return SessionView{}, ErrRuntimeUnavailable
	}

	return runtime.GetSession(sessionID)
}

func (m *ManagedRuntime) ListSessions(_ context.Context) ([]SessionSummaryView, error) {
	if m == nil {
		return nil, ErrRuntimeUnavailable
	}

	m.mu.RLock()
	runtime := m.runtime
	m.mu.RUnlock()
	if runtime == nil {
		return nil, ErrRuntimeUnavailable
	}

	return runtime.ListSessions()
}

func (m *ManagedRuntime) GetSettings() RuntimeSettingsView {
	if m == nil {
		return RuntimeSettingsView{}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings.View()
}

func (m *ManagedRuntime) UpdateSettings(_ context.Context, patch RuntimeSettingsPatch) (RuntimeSettingsView, error) {
	if m == nil {
		return RuntimeSettingsView{}, ErrRuntimeUnavailable
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	next := patch.Apply(m.settings)
	if err := validateRuntimeSettings(next); err != nil {
		return RuntimeSettingsView{}, err
	}

	nextRuntime, err := m.buildRuntime(next)
	if err != nil {
		return RuntimeSettingsView{}, err
	}
	if err := m.store.Save(next); err != nil {
		return RuntimeSettingsView{}, err
	}

	m.runtime = nextRuntime
	m.settings = next
	return next.View(), nil
}

func (m *ManagedRuntime) GetUserProfile() (UserProfile, error) {
	if m == nil {
		return UserProfile{}, ErrRuntimeUnavailable
	}
	if m.userProfiles == nil {
		return UserProfile{}, ErrUserProfileStoreUnavailable
	}
	return m.userProfiles.Get()
}

func (m *ManagedRuntime) UpdateUserProfile(_ context.Context, profile UserProfile) (UserProfile, error) {
	if m == nil {
		return UserProfile{}, ErrRuntimeUnavailable
	}
	if m.userProfiles == nil {
		return UserProfile{}, ErrUserProfileStoreUnavailable
	}
	return m.userProfiles.Update(profile)
}

func (m *ManagedRuntime) ImportUserProfileFromResume(ctx context.Context, sourceName string, contentType string, content []byte) (UserProfileImportResult, error) {
	if m == nil {
		return UserProfileImportResult{}, ErrRuntimeUnavailable
	}
	if m.userProfiles == nil {
		return UserProfileImportResult{}, ErrUserProfileStoreUnavailable
	}

	resumeText, err := extractResumeText(sourceName, contentType, content)
	if err != nil {
		return UserProfileImportResult{}, err
	}

	m.mu.RLock()
	runtime := m.runtime
	model := strings.TrimSpace(m.settings.Model)
	m.mu.RUnlock()
	if runtime == nil || runtime.provider == nil {
		return UserProfileImportResult{}, ErrRuntimeUnavailable
	}

	importer := newResumeImporter(runtime.provider, model)
	extracted, truncated, err := importer.Import(ctx, resumeText)
	if err != nil {
		return UserProfileImportResult{}, err
	}

	merged, imported, err := m.userProfiles.MergeImported(extracted)
	if err != nil {
		return UserProfileImportResult{}, err
	}

	return UserProfileImportResult{
		Profile:      merged,
		Extracted:    imported,
		SourceName:   strings.TrimSpace(sourceName),
		ContentType:  strings.TrimSpace(contentType),
		TextLength:   utf8.RuneCountInString(resumeText),
		Truncated:    truncated,
		ExtractModel: model,
	}, nil
}

func (m *ManagedRuntime) buildRuntime(settings RuntimeSettings) (*Runtime, error) {
	return NewDefaultRuntime(BootstrapConfig{
		SessionDataPath:  m.sessionDataPath,
		MemoryDataPath:   m.memoryDataPath,
		LeadManager:      m.leadManager,
		CandidateManager: m.candidateManager,
		StatsProvider:    m.statsProvider,
		AgentConfig: Config{
			Model:        settings.Model,
			MaxSteps:     settings.MaxSteps,
			SystemPrompt: settings.SystemPrompt,
		},
		UserProfiles:  m.userProfiles,
		OpenAIAPIKey:  settings.OpenAIAPIKey,
		OpenAIModel:   settings.Model,
		OpenAIBaseURL: settings.OpenAIBaseURL,
		OpenAITimeout: time.Duration(settings.OpenAITimeoutSeconds) * time.Second,
	})
}

func (m *ManagedRuntime) ReloadFromDisk() (RuntimeSettingsView, error) {
	if m == nil {
		return RuntimeSettingsView{}, ErrRuntimeUnavailable
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	settings, err := m.store.LoadOrCreate(m.settings)
	if err != nil {
		return RuntimeSettingsView{}, err
	}
	if err := validateRuntimeSettings(settings); err != nil {
		return RuntimeSettingsView{}, err
	}

	runtime, err := m.buildRuntime(settings)
	if err != nil {
		return RuntimeSettingsView{}, fmt.Errorf("rebuild runtime from disk settings: %w", err)
	}

	m.runtime = runtime
	m.settings = settings
	return settings.View(), nil
}
