package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"trace2offer/backend/agent/provider/openai"
)

func TestManagedRuntimeUpdateSettings(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	manager, err := NewManagedRuntime(ManagedRuntimeConfig{
		SettingsPath:        filepath.Join(tmpDir, "agent_runtime_config.json"),
		SessionDataPath:     filepath.Join(tmpDir, "sessions"),
		MemoryDataPath:      filepath.Join(tmpDir, "agent_memory.json"),
		UserProfileDataPath: filepath.Join(tmpDir, "user_profile.json"),
		ResumeDataDir:       filepath.Join(tmpDir, "resume"),
		LeadManager:         &stubLeadManager{},
		Defaults: RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			SystemPrompt:         "default prompt",
			OpenAIBaseURL:        "https://api.openai.com/v1/responses",
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "test_api_key",
		},
	})
	if err != nil {
		t.Fatalf("new managed runtime error: %v", err)
	}

	initial := manager.GetSettings()
	if !initial.HasOpenAIAPIKey {
		t.Fatal("expected api key to be configured")
	}
	if initial.MaxSteps != 6 {
		t.Fatalf("expected max steps 6, got %d", initial.MaxSteps)
	}

	nextModel := "gpt-5"
	nextSteps := 9
	updated, err := manager.UpdateSettings(context.Background(), RuntimeSettingsPatch{
		Model:    &nextModel,
		MaxSteps: &nextSteps,
	})
	if err != nil {
		t.Fatalf("update settings error: %v", err)
	}

	if updated.Model != "gpt-5" {
		t.Fatalf("expected updated model gpt-5, got %s", updated.Model)
	}
	if updated.MaxSteps != 9 {
		t.Fatalf("expected updated max steps 9, got %d", updated.MaxSteps)
	}

	payload, err := os.ReadFile(filepath.Join(tmpDir, "agent_runtime_config.json"))
	if err != nil {
		t.Fatalf("read settings file error: %v", err)
	}

	var persisted RuntimeSettings
	if err := json.Unmarshal(payload, &persisted); err != nil {
		t.Fatalf("decode settings file error: %v", err)
	}
	if persisted.Model != "gpt-5" {
		t.Fatalf("expected persisted model gpt-5, got %s", persisted.Model)
	}
	if persisted.MaxSteps != 9 {
		t.Fatalf("expected persisted max steps 9, got %d", persisted.MaxSteps)
	}
}

func TestManagedRuntimeUpdateSettingsValidation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manager, err := NewManagedRuntime(ManagedRuntimeConfig{
		SettingsPath:        filepath.Join(tmpDir, "agent_runtime_config.json"),
		SessionDataPath:     filepath.Join(tmpDir, "sessions"),
		MemoryDataPath:      filepath.Join(tmpDir, "agent_memory.json"),
		UserProfileDataPath: filepath.Join(tmpDir, "user_profile.json"),
		ResumeDataDir:       filepath.Join(tmpDir, "resume"),
		LeadManager:         &stubLeadManager{},
		Defaults: RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			SystemPrompt:         "default prompt",
			OpenAIBaseURL:        "https://api.openai.com/v1/responses",
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "test_api_key",
		},
	})
	if err != nil {
		t.Fatalf("new managed runtime error: %v", err)
	}

	zero := 0
	_, err = manager.UpdateSettings(context.Background(), RuntimeSettingsPatch{MaxSteps: &zero})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !IsSettingsValidationError(err) {
		t.Fatalf("expected settings validation error, got %T %v", err, err)
	}
}

func TestManagedRuntimeInvalidResumeExtractor(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	_, err := NewManagedRuntime(ManagedRuntimeConfig{
		SettingsPath:          filepath.Join(tmpDir, "agent_runtime_config.json"),
		SessionDataPath:       filepath.Join(tmpDir, "sessions"),
		MemoryDataPath:        filepath.Join(tmpDir, "agent_memory.json"),
		UserProfileDataPath:   filepath.Join(tmpDir, "user_profile.json"),
		ResumeDataDir:         filepath.Join(tmpDir, "resume"),
		ResumePDFExtractor:    "invalid_mode",
		DoclingPythonBin:      "python3",
		DoclingTimeoutSeconds: 120,
		LeadManager:           &stubLeadManager{},
		Defaults: RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			SystemPrompt:         "default prompt",
			OpenAIBaseURL:        "https://api.openai.com/v1/responses",
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "test_api_key",
		},
	})
	if err == nil {
		t.Fatal("expected invalid resume extractor error")
	}
}

func TestManagedRuntimeUpdateSettingsSwitchesDefaultBaseURLForAPIFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manager, err := NewManagedRuntime(ManagedRuntimeConfig{
		SettingsPath:        filepath.Join(tmpDir, "agent_runtime_config.json"),
		SessionDataPath:     filepath.Join(tmpDir, "sessions"),
		MemoryDataPath:      filepath.Join(tmpDir, "agent_memory.json"),
		UserProfileDataPath: filepath.Join(tmpDir, "user_profile.json"),
		ResumeDataDir:       filepath.Join(tmpDir, "resume"),
		LeadManager:         &stubLeadManager{},
		Defaults: RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			SystemPrompt:         "default prompt",
			OpenAIAPIFormat:      openai.APIFormatResponses,
			OpenAIBaseURL:        openai.DefaultResponsesBaseURL,
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "test_api_key",
		},
	})
	if err != nil {
		t.Fatalf("new managed runtime error: %v", err)
	}

	nextFormat := openai.APIFormatChatCompletions
	updated, err := manager.UpdateSettings(context.Background(), RuntimeSettingsPatch{
		OpenAIAPIFormat: &nextFormat,
	})
	if err != nil {
		t.Fatalf("update settings error: %v", err)
	}

	if updated.OpenAIAPIFormat != openai.APIFormatChatCompletions {
		t.Fatalf("expected api format %q, got %q", openai.APIFormatChatCompletions, updated.OpenAIAPIFormat)
	}
	if updated.OpenAIBaseURL != openai.DefaultChatCompletionsBaseURL {
		t.Fatalf("expected base url %q, got %q", openai.DefaultChatCompletionsBaseURL, updated.OpenAIBaseURL)
	}
}

func TestManagedRuntimeUpdateSettingsRejectsInvalidAPIFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	manager, err := NewManagedRuntime(ManagedRuntimeConfig{
		SettingsPath:        filepath.Join(tmpDir, "agent_runtime_config.json"),
		SessionDataPath:     filepath.Join(tmpDir, "sessions"),
		MemoryDataPath:      filepath.Join(tmpDir, "agent_memory.json"),
		UserProfileDataPath: filepath.Join(tmpDir, "user_profile.json"),
		ResumeDataDir:       filepath.Join(tmpDir, "resume"),
		LeadManager:         &stubLeadManager{},
		Defaults: RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			SystemPrompt:         "default prompt",
			OpenAIAPIFormat:      openai.APIFormatResponses,
			OpenAIBaseURL:        openai.DefaultResponsesBaseURL,
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "test_api_key",
		},
	})
	if err != nil {
		t.Fatalf("new managed runtime error: %v", err)
	}

	invalidFormat := "not_supported"
	_, err = manager.UpdateSettings(context.Background(), RuntimeSettingsPatch{
		OpenAIAPIFormat: &invalidFormat,
	})
	if err == nil {
		t.Fatal("expected validation error for invalid api format")
	}
	if !IsSettingsValidationError(err) {
		t.Fatalf("expected settings validation error, got %T %v", err, err)
	}
}
