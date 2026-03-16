package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"trace2offer/backend/agent/memory"
	"trace2offer/backend/agent/provider"
	"trace2offer/backend/agent/session"
	"trace2offer/backend/agent/tool"
	"trace2offer/backend/internal/model"
)

func TestRuntimeToolLoop(t *testing.T) {
	t.Parallel()

	leadManager := &stubLeadManager{
		leads: []model.Lead{{
			ID:       "lead_1",
			Company:  "OpenAI",
			Position: "Backend Engineer",
			Status:   "new",
		}},
	}

	registry, err := tool.NewRegistry(tool.NewLeadCRUDTools(leadManager)...)
	if err != nil {
		t.Fatalf("new tool registry error: %v", err)
	}

	model := &scriptedProvider{responses: []string{
		`{"type":"tool","tool":"lead_list","arguments":{}}`,
		`{"type":"final","message":"当前共 1 条 lead。"}`,
	}}

	runtime, err := NewRuntime(Dependencies{
		SessionManager: session.NewManager(session.NewInMemoryStore()),
		MemoryManager:  memory.NewManager(memory.NewInMemoryStore(), 10),
		Tools:          registry,
		Provider:       model,
		Config: Config{
			Model:    "test-model",
			MaxSteps: 4,
		},
	})
	if err != nil {
		t.Fatalf("new runtime error: %v", err)
	}

	response, err := runtime.Run(context.Background(), Request{Message: "帮我看看 lead"})
	if err != nil {
		t.Fatalf("run runtime error: %v", err)
	}
	if response.SessionID == "" {
		t.Fatal("expected session id")
	}
	if response.Reply != "当前共 1 条 lead。" {
		t.Fatalf("unexpected reply: %s", response.Reply)
	}

	if len(model.requests) != 2 {
		t.Fatalf("expected 2 model requests, got %d", len(model.requests))
	}

	messages, err := runtime.sessions.Messages(response.SessionID)
	if err != nil {
		t.Fatalf("load session messages error: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages (user/tool/assistant), got %d", len(messages))
	}
	if messages[1].Role != "tool" {
		t.Fatalf("expected second message role=tool, got %s", messages[1].Role)
	}
}

func TestRuntimeRequiresMessage(t *testing.T) {
	t.Parallel()

	registry, err := tool.NewRegistry(&noopTool{})
	if err != nil {
		t.Fatalf("new tool registry error: %v", err)
	}

	runtime, err := NewRuntime(Dependencies{
		SessionManager: session.NewManager(session.NewInMemoryStore()),
		MemoryManager:  memory.NewManager(memory.NewInMemoryStore(), 10),
		Tools:          registry,
		Provider:       &scriptedProvider{responses: []string{`{"type":"final","message":"ok"}`}},
	})
	if err != nil {
		t.Fatalf("new runtime error: %v", err)
	}

	_, err = runtime.Run(context.Background(), Request{Message: " "})
	if err == nil {
		t.Fatal("expected message validation error")
	}
}

type scriptedProvider struct {
	responses []string
	requests  []provider.Request
}

func (p *scriptedProvider) Name() string {
	return "scripted"
}

func (p *scriptedProvider) Generate(_ context.Context, request provider.Request) (provider.Response, error) {
	p.requests = append(p.requests, request)
	if len(p.responses) == 0 {
		return provider.Response{}, errors.New("no scripted response")
	}
	response := p.responses[0]
	p.responses = p.responses[1:]
	return provider.Response{Content: response}, nil
}

type stubLeadManager struct {
	leads []model.Lead
}

func (s *stubLeadManager) List() []model.Lead {
	copied := make([]model.Lead, len(s.leads))
	copy(copied, s.leads)
	return copied
}

func (s *stubLeadManager) Create(input model.LeadMutationInput) (model.Lead, error) {
	created := model.Lead{
		ID:         "lead_created",
		Company:    input.Company,
		Position:   input.Position,
		Source:     input.Source,
		Status:     input.Status,
		Priority:   input.Priority,
		NextAction: input.NextAction,
		Notes:      input.Notes,
	}
	s.leads = append(s.leads, created)
	return created, nil
}

func (s *stubLeadManager) Update(id string, input model.LeadMutationInput) (model.Lead, bool, error) {
	for index := range s.leads {
		if s.leads[index].ID != id {
			continue
		}
		s.leads[index].Company = input.Company
		s.leads[index].Position = input.Position
		s.leads[index].Source = input.Source
		s.leads[index].Status = input.Status
		s.leads[index].Priority = input.Priority
		s.leads[index].NextAction = input.NextAction
		s.leads[index].Notes = input.Notes
		return s.leads[index], true, nil
	}
	return model.Lead{}, false, nil
}

func (s *stubLeadManager) Delete(id string) (bool, error) {
	for index := range s.leads {
		if s.leads[index].ID != id {
			continue
		}
		s.leads = append(s.leads[:index], s.leads[index+1:]...)
		return true, nil
	}
	return false, nil
}

type noopTool struct{}

func (n *noopTool) Definition() tool.Definition {
	return tool.Definition{Name: "noop", Description: "noop", InputSchema: map[string]any{"type": "object"}}
}

func (n *noopTool) Run(_ context.Context, _ json.RawMessage) (string, error) {
	return `{}`, nil
}
