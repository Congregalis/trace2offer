package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"trace2offer/backend/internal/candidate"
	"trace2offer/backend/internal/model"
)

// NewCandidateCRUDTools returns candidate CRUD tools used by agent runtime.
func NewCandidateCRUDTools(manager candidate.Manager) []Tool {
	return []Tool{
		&candidateListTool{manager: manager},
		&candidateCreateTool{manager: manager},
		&candidateUpdateTool{manager: manager},
		&candidateDeleteTool{manager: manager},
		&candidatePromoteTool{manager: manager},
	}
}

type candidateListTool struct {
	manager candidate.Manager
}

func (t *candidateListTool) Definition() Definition {
	return Definition{
		Name:        "candidate_list",
		Description: "列出所有候选职位",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *candidateListTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if err := ensureCandidateManager(t.manager); err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(input))) > 0 && string(input) != "{}" {
		var ignored map[string]any
		if err := json.Unmarshal(input, &ignored); err != nil {
			return "", fmt.Errorf("invalid candidate_list arguments: %w", err)
		}
	}

	items := t.manager.List()
	return marshalOutput(map[string]any{
		"count":      len(items),
		"candidates": items,
	})
}

type candidateCreateTool struct {
	manager candidate.Manager
}

func (t *candidateCreateTool) Definition() Definition {
	return Definition{
		Name:        "candidate_create",
		Description: "创建一个候选职位",
		InputSchema: candidateMutationInputSchema([]string{"company", "position"}),
	}
}

func (t *candidateCreateTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if err := ensureCandidateManager(t.manager); err != nil {
		return "", err
	}

	var args model.CandidateMutationInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	created, err := t.manager.Create(args)
	if err != nil {
		return "", err
	}
	return marshalOutput(map[string]any{"candidate": created})
}

type candidateUpdateTool struct {
	manager candidate.Manager
}

func (t *candidateUpdateTool) Definition() Definition {
	schema := candidateMutationInputSchema([]string{"id", "company", "position"})
	properties := schema["properties"].(map[string]any)
	properties["id"] = map[string]any{
		"type":        "string",
		"description": "candidate id",
	}
	return Definition{
		Name:        "candidate_update",
		Description: "更新一个候选职位",
		InputSchema: schema,
	}
}

type candidateUpdateInput struct {
	ID string `json:"id"`
	model.CandidateMutationInput
}

func (t *candidateUpdateTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if err := ensureCandidateManager(t.manager); err != nil {
		return "", err
	}

	var args candidateUpdateInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	updated, found, err := t.manager.Update(args.ID, args.CandidateMutationInput)
	if err != nil {
		return "", err
	}

	return marshalOutput(map[string]any{
		"found":     found,
		"candidate": updated,
	})
}

type candidateDeleteTool struct {
	manager candidate.Manager
}

func (t *candidateDeleteTool) Definition() Definition {
	return Definition{
		Name:        "candidate_delete",
		Description: "删除一个候选职位",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "candidate id",
				},
			},
			"required": []string{"id"},
		},
	}
}

func (t *candidateDeleteTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if err := ensureCandidateManager(t.manager); err != nil {
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

type candidatePromoteTool struct {
	manager candidate.Manager
}

func (t *candidatePromoteTool) Definition() Definition {
	return Definition{
		Name:        "candidate_promote",
		Description: "将候选职位转为正式 lead",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{
					"type":        "string",
					"description": "candidate id",
				},
				"source": map[string]any{
					"type":        "string",
					"description": "可选，覆盖 lead source",
				},
				"status": map[string]any{
					"type":        "string",
					"description": "可选，lead status，默认 new",
				},
				"priority": map[string]any{
					"type":        "integer",
					"description": "可选，lead priority",
				},
				"next_action": map[string]any{
					"type":        "string",
					"description": "可选，lead next action",
				},
				"next_action_at": map[string]any{
					"type":        "string",
					"description": "可选，lead next action time RFC3339",
				},
				"interview_at": map[string]any{
					"type":        "string",
					"description": "可选，lead interview time RFC3339",
				},
				"reminder_methods": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
				"notes": map[string]any{
					"type":        "string",
					"description": "可选，附加 lead notes",
				},
			},
			"required": []string{"id"},
		},
	}
}

type candidatePromoteToolInput struct {
	ID string `json:"id"`
	model.CandidatePromoteInput
}

func (t *candidatePromoteTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if err := ensureCandidateManager(t.manager); err != nil {
		return "", err
	}

	var args candidatePromoteToolInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}

	updatedCandidate, createdLead, err := t.manager.Promote(args.ID, args.CandidatePromoteInput)
	if err != nil {
		return "", err
	}

	return marshalOutput(map[string]any{
		"candidate": updatedCandidate,
		"lead":      createdLead,
	})
}

func ensureCandidateManager(manager candidate.Manager) error {
	if manager == nil {
		return fmt.Errorf("candidate manager is unavailable")
	}
	return nil
}

func candidateMutationInputSchema(required []string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"company": map[string]any{
				"type": "string",
			},
			"position": map[string]any{
				"type": "string",
			},
			"source": map[string]any{
				"type": "string",
			},
			"location": map[string]any{
				"type": "string",
			},
			"jd_url": map[string]any{
				"type": "string",
			},
			"company_website_url": map[string]any{
				"type": "string",
			},
			"status": map[string]any{
				"type": "string",
			},
			"match_score": map[string]any{
				"type": "integer",
			},
			"match_reasons": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"recommendation_notes": map[string]any{
				"type": "string",
			},
			"notes": map[string]any{
				"type": "string",
			},
		},
		"required": required,
	}
}
