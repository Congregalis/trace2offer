package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"trace2offer/backend/agent"
	"trace2offer/backend/internal/api"
	"trace2offer/backend/internal/calendar"
	"trace2offer/backend/internal/candidate"
	appconfig "trace2offer/backend/internal/config"
	"trace2offer/backend/internal/discovery"
	"trace2offer/backend/internal/heartbeat"
	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/prep"
	"trace2offer/backend/internal/reminder"
	"trace2offer/backend/internal/stats"
	"trace2offer/backend/internal/storage"
	"trace2offer/backend/internal/timeline"
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
	openAIAPIFormat := getenv("T2O_OPENAI_API_FORMAT", "responses")
	openAITimeoutSeconds, err := getenvInt("T2O_OPENAI_TIMEOUT_SECONDS", 60)
	if err != nil {
		log.Fatalf("invalid T2O_OPENAI_TIMEOUT_SECONDS: %v", err)
	}
	prepOpenAITimeoutSeconds, err := getenvInt("T2O_PREP_OPENAI_TIMEOUT_SECONDS", 180)
	if err != nil {
		log.Fatalf("invalid T2O_PREP_OPENAI_TIMEOUT_SECONDS: %v", err)
	}
	doclingTimeoutSeconds, err := getenvInt("T2O_DOCLING_TIMEOUT_SECONDS", 120)
	if err != nil {
		log.Fatalf("invalid T2O_DOCLING_TIMEOUT_SECONDS: %v", err)
	}
	maxSteps, err := getenvInt("T2O_AGENT_MAX_STEPS", 6)
	if err != nil {
		log.Fatalf("invalid T2O_AGENT_MAX_STEPS: %v", err)
	}

	leadStore, err := storage.NewFileLeadStore(filepath.Join(dataDir, "leads.json"))
	if err != nil {
		log.Fatalf("init lead store failed: %v", err)
	}
	candidateStore, err := storage.NewFileCandidateStore(filepath.Join(dataDir, "candidates.json"))
	if err != nil {
		log.Fatalf("init candidate store failed: %v", err)
	}
	discoveryRuleStore, err := storage.NewFileDiscoveryRuleStore(filepath.Join(dataDir, "discovery_rules.json"))
	if err != nil {
		log.Fatalf("init discovery rule store failed: %v", err)
	}
	leadTimelineStore, err := storage.NewFileLeadTimelineStore(filepath.Join(dataDir, "lead_timelines.json"))
	if err != nil {
		log.Fatalf("init lead timeline store failed: %v", err)
	}
	timelineService := timeline.NewService(leadTimelineStore)
	leadManager := lead.NewService(leadStore).WithStatusObserver(timelineService)
	candidateManager := candidate.NewService(candidateStore, leadManager)
	discoveryService := discovery.NewService(discoveryRuleStore, candidateManager)
	statsService := stats.NewService(leadStore)
	reminderService := reminder.NewService(leadStore)
	calendarService := calendar.NewService(leadStore)
	heartbeatService, err := heartbeat.NewService(heartbeat.Config{
		DataDir:          dataDir,
		Interval:         30 * time.Minute,
		ReminderService:  reminderService,
		StatsService:     statsService,
		DiscoveryService: discoveryService,
	})
	if err != nil {
		log.Fatalf("init heartbeat service failed: %v", err)
	}
	prepConfig, err := prep.LoadConfig(dataDir)
	if err != nil {
		log.Fatalf("load prep config failed: %v", err)
	}

	var prepOptions []prep.ServiceOption
	if strings.TrimSpace(getenv("OPENAI_API_KEY", "")) != "" {
		questionModel, modelErr := prep.NewOpenAIQuestionModel(
			getenv("OPENAI_API_KEY", ""),
			getenv("T2O_OPENAI_BASE_URL", ""),
			openAIAPIFormat,
			model,
			time.Duration(prepOpenAITimeoutSeconds)*time.Second,
		)
		if modelErr != nil {
			log.Printf("init prep question model skipped: %v", modelErr)
		} else {
			prepOptions = append(prepOptions, prep.WithQuestionModel(questionModel))
		}
	}

	prepService, err := prep.NewService(prepConfig, prepOptions...)
	if err != nil {
		log.Fatalf("init prep service failed: %v", err)
	}

	runtime, err := agent.NewManagedRuntime(agent.ManagedRuntimeConfig{
		SettingsPath:          getenv("T2O_AGENT_SETTINGS_DATA", filepath.Join(dataDir, "agent_runtime_config.json")),
		SessionDataPath:       getenv("T2O_AGENT_SESSION_DATA", filepath.Join(dataDir, "sessions")),
		MemoryDataPath:        getenv("T2O_AGENT_MEMORY_DATA", filepath.Join(dataDir, "agent_memory.json")),
		UserProfileDataPath:   getenv("T2O_AGENT_USER_PROFILE_DATA", filepath.Join(dataDir, "user_profile.json")),
		ResumeDataDir:         getenv("T2O_AGENT_RESUME_DATA_DIR", filepath.Join(dataDir, "resume")),
		ResumePDFExtractor:    getenv("T2O_RESUME_PDF_EXTRACTOR", "legacy"),
		DoclingPythonBin:      getenv("T2O_DOCLING_PYTHON_BIN", "python3"),
		DoclingTimeoutSeconds: doclingTimeoutSeconds,
		LeadManager:           leadManager,
		CandidateManager:      candidateManager,
		StatsProvider:         statsService,
		Defaults: agent.RuntimeSettings{
			Model:                model,
			MaxSteps:             maxSteps,
			SystemPrompt:         getenv("T2O_AGENT_SYSTEM_PROMPT", ""),
			OpenAIAPIFormat:      openAIAPIFormat,
			OpenAIBaseURL:        getenv("T2O_OPENAI_BASE_URL", ""),
			OpenAITimeoutSeconds: openAITimeoutSeconds,
			OpenAIAPIKey:         getenv("OPENAI_API_KEY", ""),
		},
	})
	if err != nil {
		log.Fatalf("init agent runtime failed: %v", err)
	}

	go heartbeatService.Start(context.Background())

	router := api.NewRouter(leadStore, candidateStore, leadTimelineStore, runtime, statsService, reminderService, heartbeatService, calendarService, discoveryService, prepService)
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
