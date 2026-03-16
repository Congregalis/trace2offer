package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"trace2offer/backend/agent"
	"trace2offer/backend/internal/api"
	appconfig "trace2offer/backend/internal/config"
	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/reminder"
	"trace2offer/backend/internal/stats"
	"trace2offer/backend/internal/storage"
)

func main() {
	envFile := strings.TrimSpace(os.Getenv("T2O_ENV_FILE"))
	if envFile != "" {
		if err := appconfig.LoadDotEnv(envFile); err != nil {
			log.Fatalf("load env file failed: %v", err)
		}
	} else {
		if err := loadDefaultEnvFiles(); err != nil {
			log.Fatalf("load default env files failed: %v", err)
		}
	}

	port := getenv("PORT", "8080")
	dataDir := getenv("T2O_DATA_DIR", "./data")
	model := getenv("T2O_AGENT_MODEL", "gpt-5-mini")

	leadStore, err := storage.NewFileLeadStore(filepath.Join(dataDir, "leads.json"))
	if err != nil {
		log.Fatalf("init lead store failed: %v", err)
	}
	statsService := stats.NewService(leadStore)
	reminderService := reminder.NewService(leadStore)

	maxSteps, err := getenvInt("T2O_AGENT_MAX_STEPS", 6)
	if err != nil {
		log.Fatalf("invalid T2O_AGENT_MAX_STEPS: %v", err)
	}
	openAITimeoutSeconds, err := getenvInt("T2O_OPENAI_TIMEOUT_SECONDS", 60)
	if err != nil {
		log.Fatalf("invalid T2O_OPENAI_TIMEOUT_SECONDS: %v", err)
	}

	runtime, err := agent.NewManagedRuntime(agent.ManagedRuntimeConfig{
		SettingsPath:        getenv("T2O_AGENT_SETTINGS_DATA", filepath.Join(dataDir, "agent_runtime_config.json")),
		SessionDataPath:     getenv("T2O_AGENT_SESSION_DATA", filepath.Join(dataDir, "sessions")),
		MemoryDataPath:      getenv("T2O_AGENT_MEMORY_DATA", filepath.Join(dataDir, "agent_memory.json")),
		UserProfileDataPath: getenv("T2O_AGENT_USER_PROFILE_DATA", filepath.Join(dataDir, "user_profile.json")),
		LeadManager:         lead.NewService(leadStore),
		StatsProvider:       statsService,
		Defaults: agent.RuntimeSettings{
			Model:                model,
			MaxSteps:             maxSteps,
			SystemPrompt:         getenv("T2O_AGENT_SYSTEM_PROMPT", ""),
			OpenAIBaseURL:        getenv("T2O_OPENAI_BASE_URL", ""),
			OpenAITimeoutSeconds: openAITimeoutSeconds,
			OpenAIAPIKey:         getenv("OPENAI_API_KEY", ""),
		},
	})
	if err != nil {
		log.Fatalf("init agent runtime failed: %v", err)
	}

	router := api.NewRouter(leadStore, runtime, statsService, reminderService)
	addr := ":" + port
	log.Printf("trace2offer backend listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func loadDefaultEnvFiles() error {
	for _, path := range []string{".env", "backend/.env"} {
		if err := appconfig.LoadDotEnv(path); err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s=%q: %w", key, raw, err)
	}
	return value, nil
}
