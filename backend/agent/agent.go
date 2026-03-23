package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"trace2offer/backend/agent/memory"
	"trace2offer/backend/agent/provider"
	"trace2offer/backend/agent/session"
	"trace2offer/backend/agent/tool"
)

// Config controls loop behavior.
type Config struct {
	Model        string
	MaxSteps     int
	SystemPrompt string
}

func DefaultConfig() Config {
	return Config{
		Model:    "gpt-5-mini",
		MaxSteps: 6,
		SystemPrompt: `你是 Trace2Offer 的求职 Agent。
目标：高质量完成用户请求，必要时调用工具，别编造。

【Think Tool 使用指南】
当你需要复杂推理时，可以使用 think 工具：
- 分析多个线索的优先级，制定投递策略
- 评估收到的 Offer，对比各机会
- 整理从工具获取的信息，做出决策
- 决定下一步行动计划

注意：如果你已经有足够的信息直接回答用户，就不需要调用 think。

【工具选择建议】
- 搜索线索 → 用 lead_search（支持状态、公司、优先级筛选）
- 创建线索 → 用 lead_create 或 lead_create_from_jd_url
- 管理候选池 → 用 candidate_list / candidate_create / candidate_update / candidate_delete
- 候选转线索 → 用 candidate_promote
- 查看统计 → 用 stats_summary 或 job_search_strategy
- 复杂推理 → 用 think（分析、对比、制定策略时使用）

每一步只能做一个动作。`,
	}
}

// Dependencies are required to build Runtime.
type Dependencies struct {
	Config         Config
	SessionManager *session.Manager
	MemoryManager  *memory.Manager
	Tools          *tool.Registry
	Provider       provider.Provider
	UserProfiles   *UserProfileManager
}

// Runtime is the minimal agent loop runtime.
type Runtime struct {
	config       Config
	sessions     *session.Manager
	memory       *memory.Manager
	tools        *tool.Registry
	provider     provider.Provider
	userProfiles *UserProfileManager
}

// Request is one user message.
type Request struct {
	SessionID string
	Message   string
}

// Response is one agent reply.
type Response struct {
	SessionID string
	Reply     string
}

func NewRuntime(deps Dependencies) (*Runtime, error) {
	if deps.SessionManager == nil {
		return nil, errors.New("session manager is required")
	}
	if deps.MemoryManager == nil {
		return nil, errors.New("memory manager is required")
	}
	if deps.Tools == nil {
		return nil, errors.New("tool registry is required")
	}
	if deps.Provider == nil {
		return nil, errors.New("model provider is required")
	}

	config := deps.Config
	defaults := DefaultConfig()
	if strings.TrimSpace(config.Model) == "" {
		config.Model = defaults.Model
	}
	if config.MaxSteps <= 0 {
		config.MaxSteps = defaults.MaxSteps
	}
	if strings.TrimSpace(config.SystemPrompt) == "" {
		config.SystemPrompt = defaults.SystemPrompt
	}

	return &Runtime{
		config:       config,
		sessions:     deps.SessionManager,
		memory:       deps.MemoryManager,
		tools:        deps.Tools,
		provider:     deps.Provider,
		userProfiles: deps.UserProfiles,
	}, nil
}

func (r *Runtime) Run(ctx context.Context, request Request) (Response, error) {
	if r == nil {
		return Response{}, errors.New("agent runtime is nil")
	}

	message := strings.TrimSpace(request.Message)
	if message == "" {
		return Response{}, errors.New("user message is required")
	}

	sess, err := r.sessions.Ensure(strings.TrimSpace(request.SessionID))
	if err != nil {
		return Response{}, fmt.Errorf("ensure session: %w", err)
	}

	sess, err = r.sessions.Append(sess.ID, "user", message)
	if err != nil {
		return Response{}, fmt.Errorf("append user message: %w", err)
	}

	userPrompt := ""
	if r.userProfiles != nil {
		userPrompt = strings.TrimSpace(r.userProfiles.PromptBlock())
	}

	for step := 0; step < r.config.MaxSteps; step++ {
		recall, err := r.memory.Recall(sess.ID)
		if err != nil {
			return Response{}, fmt.Errorf("recall memory: %w", err)
		}

		modelMessages := r.buildModelMessages(sess.Messages, recall, userPrompt)
		modelResp, err := r.provider.Generate(ctx, provider.Request{
			Model:    r.config.Model,
			Messages: modelMessages,
		})
		if err != nil {
			return Response{}, fmt.Errorf("call model provider: %w", err)
		}

		decision := parseModelDecision(modelResp.Content)
		if decision.Type == "tool" {
			observation, callErr := r.executeTool(ctx, decision)
			sess, err = r.sessions.Append(sess.ID, "tool", observation)
			if err != nil {
				return Response{}, fmt.Errorf("append tool observation: %w", err)
			}
			if callErr != nil {
				continue
			}
			continue
		}

		reply := strings.TrimSpace(decision.Message)
		if reply == "" {
			reply = strings.TrimSpace(modelResp.Content)
		}
		if reply == "" {
			reply = "我这轮没拿到有效输出，麻烦你重试一次。"
		}

		if _, err := r.sessions.Append(sess.ID, "assistant", reply); err != nil {
			return Response{}, fmt.Errorf("append assistant message: %w", err)
		}
		_ = r.memory.Remember(sess.ID, compactFact(message))

		return Response{SessionID: sess.ID, Reply: reply}, nil
	}

	fallback := "工具调用超过上限了，我先停下，建议你换个更具体的问题再试。"
	if _, err := r.sessions.Append(sess.ID, "assistant", fallback); err != nil {
		return Response{}, fmt.Errorf("append fallback message: %w", err)
	}
	return Response{SessionID: sess.ID, Reply: fallback}, nil
}

func (r *Runtime) executeTool(ctx context.Context, decision modelDecision) (string, error) {
	output, err := r.tools.Call(ctx, decision.Tool, decision.Arguments)
	payload := map[string]any{
		"tool":      decision.Tool,
		"arguments": json.RawMessage(decision.Arguments),
		"ok":        err == nil,
	}
	if err != nil {
		payload["error"] = err.Error()
	} else {
		payload["result"] = json.RawMessage(output)
	}

	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		if err != nil {
			return fmt.Sprintf("tool=%s ok=false error=%s", decision.Tool, err.Error()), err
		}
		return fmt.Sprintf("tool=%s ok=true result=%s", decision.Tool, output), nil
	}
	return string(encoded), err
}

func (r *Runtime) buildModelMessages(history []session.Message, recall []string, userPrompt string) []provider.Message {
	messages := make([]provider.Message, 0, len(history)+1)
	messages = append(messages, provider.Message{
		Role:    "system",
		Content: r.buildSystemPrompt(recall, userPrompt),
	})

	for _, item := range history {
		role := strings.TrimSpace(item.Role)
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		switch role {
		case "user", "assistant", "system":
			messages = append(messages, provider.Message{Role: role, Content: content})
		case "tool":
			messages = append(messages, provider.Message{Role: "user", Content: "工具执行结果: " + content})
		default:
			messages = append(messages, provider.Message{Role: "user", Content: content})
		}
	}

	return messages
}

func (r *Runtime) buildSystemPrompt(recall []string, userPrompt string) string {
	builder := strings.Builder{}
	builder.WriteString(strings.TrimSpace(r.config.SystemPrompt))

	if strings.TrimSpace(userPrompt) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(strings.TrimSpace(userPrompt))
	}

	if len(recall) > 0 {
		builder.WriteString("\n\n当前记忆：")
		for _, fact := range recall {
			trimmed := strings.TrimSpace(fact)
			if trimmed == "" {
				continue
			}
			builder.WriteString("\n- ")
			builder.WriteString(trimmed)
		}
	}

	defs := r.tools.Definitions()
	if len(defs) > 0 {
		builder.WriteString("\n\n可用工具：")
		for _, item := range defs {
			schema, _ := json.Marshal(item.InputSchema)
			builder.WriteString("\n- ")
			builder.WriteString(item.Name)
			if item.Description != "" {
				builder.WriteString(": ")
				builder.WriteString(item.Description)
			}
			builder.WriteString(" | schema=")
			builder.WriteString(string(schema))
		}
	}

	builder.WriteString("\n\n输出要求（必须严格遵守）：")
	builder.WriteString("\n1) 只输出一个 JSON 对象，不要 markdown，不要额外解释。")
	builder.WriteString("\n2) 需要调用工具时输出：{\"type\":\"tool\",\"tool\":\"tool_name\",\"arguments\":{...}}")
	builder.WriteString("\n3) 不需要工具时输出：{\"type\":\"final\",\"message\":\"给用户的回复\"}")
	builder.WriteString("\n4) 一轮最多调用一个工具。")

	return builder.String()
}

type modelDecision struct {
	Type      string          `json:"type"`
	Message   string          `json:"message"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

func parseModelDecision(raw string) modelDecision {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return modelDecision{Type: "final", Message: ""}
	}

	candidate := trimmed
	if strings.HasPrefix(candidate, "```") {
		candidate = extractCodeBlock(candidate)
	}

	var decision modelDecision
	if err := json.Unmarshal([]byte(candidate), &decision); err != nil {
		return modelDecision{Type: "final", Message: trimmed}
	}

	decision.Type = strings.ToLower(strings.TrimSpace(decision.Type))
	decision.Tool = strings.TrimSpace(decision.Tool)
	decision.Message = strings.TrimSpace(decision.Message)
	if len(strings.TrimSpace(string(decision.Arguments))) == 0 {
		decision.Arguments = json.RawMessage("{}")
	}

	if decision.Type == "tool" && decision.Tool != "" {
		return decision
	}
	if decision.Type == "final" {
		return decision
	}
	if decision.Message != "" {
		return modelDecision{Type: "final", Message: decision.Message}
	}
	return modelDecision{Type: "final", Message: trimmed}
}

func extractCodeBlock(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) <= 2 {
		return trimmed
	}
	body := lines[1 : len(lines)-1]
	return strings.TrimSpace(strings.Join(body, "\n"))
}

func compactFact(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) > 120 {
		return string(runes[:120])
	}
	return text
}
