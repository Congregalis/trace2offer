package agent

import (
	"fmt"
	"strings"
	"time"

	"trace2offer/backend/agent/memory"
	"trace2offer/backend/agent/provider"
	"trace2offer/backend/agent/provider/openai"
	"trace2offer/backend/agent/session"
	"trace2offer/backend/agent/tool"
	"trace2offer/backend/internal/lead"
)

// BootstrapConfig wires minimal runtime dependencies.
type BootstrapConfig struct {
	SessionDataPath string
	MemoryDataPath  string
	LeadManager     lead.Manager
	UserProfiles    *UserProfileManager
	AgentConfig     Config
	OpenAIAPIKey    string
	OpenAIModel     string
	OpenAIBaseURL   string
	OpenAITimeout   time.Duration
}

// NewDefaultRuntime builds a minimal runtime with OpenAI Responses provider.
func NewDefaultRuntime(config BootstrapConfig) (*Runtime, error) {
	if config.LeadManager == nil {
		return nil, fmt.Errorf("lead manager is required")
	}
	if strings.TrimSpace(config.SessionDataPath) == "" {
		return nil, fmt.Errorf("session data path is required")
	}
	if strings.TrimSpace(config.MemoryDataPath) == "" {
		return nil, fmt.Errorf("memory data path is required")
	}

	sessionStore, err := session.NewFileStore(config.SessionDataPath)
	if err != nil {
		return nil, err
	}
	memoryStore, err := memory.NewFileStore(config.MemoryDataPath)
	if err != nil {
		return nil, err
	}

	openAIProvider, err := openai.New(openai.Config{
		APIKey:  config.OpenAIAPIKey,
		Model:   config.OpenAIModel,
		BaseURL: config.OpenAIBaseURL,
		Timeout: config.OpenAITimeout,
	})
	if err != nil {
		return nil, err
	}

	providerManager, err := provider.NewManager(openAIProvider.Name(), openAIProvider)
	if err != nil {
		return nil, err
	}
	modelProvider, err := providerManager.Default()
	if err != nil {
		return nil, err
	}

	extractor := tool.NewLLMJDExtractor(modelProvider, strings.TrimSpace(config.OpenAIModel))
	leadTools := tool.NewLeadCRUDTools(config.LeadManager, tool.WithJDExtractor(extractor))
	registry, err := tool.NewRegistry(leadTools...)
	if err != nil {
		return nil, err
	}

	runtimeConfig := config.AgentConfig
	if strings.TrimSpace(runtimeConfig.Model) == "" {
		runtimeConfig.Model = strings.TrimSpace(config.OpenAIModel)
	}

	return NewRuntime(Dependencies{
		Config:         runtimeConfig,
		SessionManager: session.NewManager(sessionStore),
		MemoryManager:  memory.NewManager(memoryStore, 20),
		Tools:          registry,
		Provider:       modelProvider,
		UserProfiles:   config.UserProfiles,
	})
}
