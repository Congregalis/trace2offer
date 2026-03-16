package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
)

type leadToolsConfig struct {
	jdExtractor JDExtractor
}

type LeadToolsOption func(*leadToolsConfig)

func WithJDExtractor(extractor JDExtractor) LeadToolsOption {
	return func(config *leadToolsConfig) {
		if config == nil {
			return
		}
		config.jdExtractor = extractor
	}
}

// NewLeadCRUDTools returns lead CRUD tools that share the same lead service with API.
func NewLeadCRUDTools(manager lead.Manager, options ...LeadToolsOption) []Tool {
	config := leadToolsConfig{}
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&config)
	}

	return []Tool{
		&leadListTool{manager: manager},
		&leadCreateTool{manager: manager},
		&leadUpdateTool{manager: manager},
		&leadDeleteTool{manager: manager},
		newLeadCreateFromJDURLTool(manager, config.jdExtractor),
	}
}

type leadListTool struct {
	manager lead.Manager
}

func (t *leadListTool) Definition() Definition {
	return Definition{
		Name:        "lead_list",
		Description: "列出所有 lead",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *leadListTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	if err := ensureManager(t.manager); err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(input))) > 0 && string(input) != "{}" {
		var ignored map[string]any
		if err := json.Unmarshal(input, &ignored); err != nil {
			return "", fmt.Errorf("invalid lead_list arguments: %w", err)
		}
	}

	leads := t.manager.List()
	return marshalOutput(map[string]any{
		"count": len(leads),
		"leads": leads,
	})
}

type leadCreateTool struct {
	manager lead.Manager
}

func (t *leadCreateTool) Definition() Definition {
	return Definition{
		Name:        "lead_create",
		Description: "创建一个 lead",
		InputSchema: mutationInputSchema([]string{"company", "position"}),
	}
}

func (t *leadCreateTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	if err := ensureManager(t.manager); err != nil {
		return "", err
	}

	var args model.LeadMutationInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	created, err := t.manager.Create(args)
	if err != nil {
		return "", err
	}

	return marshalOutput(map[string]any{"lead": created})
}

type leadUpdateTool struct {
	manager lead.Manager
}

func (t *leadUpdateTool) Definition() Definition {
	schema := mutationInputSchema([]string{"id", "company", "position"})
	properties := schema["properties"].(map[string]any)
	properties["id"] = map[string]any{
		"type":        "string",
		"description": "lead id",
	}
	return Definition{
		Name:        "lead_update",
		Description: "更新一个 lead",
		InputSchema: schema,
	}
}

type leadUpdateInput struct {
	ID string `json:"id"`
	model.LeadMutationInput
}

func (t *leadUpdateTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	if err := ensureManager(t.manager); err != nil {
		return "", err
	}

	var args leadUpdateInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	updated, found, err := t.manager.Update(args.ID, args.LeadMutationInput)
	if err != nil {
		return "", err
	}

	return marshalOutput(map[string]any{
		"found": found,
		"lead":  updated,
	})
}

type leadDeleteTool struct {
	manager lead.Manager
}

func (t *leadDeleteTool) Definition() Definition {
	return Definition{
		Name:        "lead_delete",
		Description: "删除一个 lead",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "lead id",
				},
			},
			"required": []string{"id"},
		},
	}
}

func (t *leadDeleteTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	if err := ensureManager(t.manager); err != nil {
		return "", err
	}

	var args struct {
		ID string `json:"id"`
	}
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	deleted, err := t.manager.Delete(args.ID)
	if err != nil {
		return "", err
	}

	return marshalOutput(map[string]any{"deleted": deleted})
}

func ensureManager(manager lead.Manager) error {
	if manager == nil {
		return fmt.Errorf("lead manager is unavailable")
	}
	return nil
}

func decodeInput(raw json.RawMessage, out any) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		trimmed = "{}"
	}
	if err := json.Unmarshal([]byte(trimmed), out); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}

func mutationInputSchema(required []string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"company":             map[string]any{"type": "string"},
			"position":            map[string]any{"type": "string"},
			"source":              map[string]any{"type": "string"},
			"status":              map[string]any{"type": "string"},
			"priority":            map[string]any{"type": "integer"},
			"next_action":         map[string]any{"type": "string"},
			"notes":               map[string]any{"type": "string"},
			"company_website_url": map[string]any{"type": "string"},
			"jd_url":              map[string]any{"type": "string"},
			"location":            map[string]any{"type": "string"},
		},
		"required": required,
	}
}

func marshalOutput(payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode tool output: %w", err)
	}
	return string(encoded), nil
}
