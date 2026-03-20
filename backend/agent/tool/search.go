package tool

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
)

// SearchLeadsTool provides semantic search over leads.
// Reference: Anthropic Engineering - "Writing effective tools for agents"
//
// Why this design:
// 1. Targeted over generic: "search_leads" is better than "list_all_leads"
//    - Agents with limited context can't efficiently process all leads at once
//    - Listing all contacts and searching manually is wasteful (like reading entire address book)
//
// 2. Rich filtering: Supports status, company, position, priority
//    - Agents can express intent naturally ("找出所有面试中的")
//
// 3. Returns meaningful context: Goes beyond raw database records
//    - Adds computed fields like days_since_update, next_action_urgent
//    - Prioritizes fields that inform downstream actions (name, status, next_action)
//
// 4. Response size control: Default limit prevents context overflow
//    - Anthropic recommends pagination, range selection, truncation
type SearchLeadsTool struct {
	Manager lead.Manager
}

// SearchLeadsInput represents search parameters.
type SearchLeadsInput struct {
	Status    string `json:"status,omitempty"`    // Filter by status: new, preparing, applied, interviewing, offered, rejected
	Company   string `json:"company,omitempty"`   // Partial match on company name
	Position  string `json:"position,omitempty"`  // Partial match on position title
	Priority  *int   `json:"priority,omitempty"`  // Filter by priority (1-5)
	Limit     int    `json:"limit,omitempty"`     // Max results (default 10)
	SortBy    string `json:"sort_by,omitempty"`   // Sort: priority, updated_at, created_at
	Interview bool   `json:"interview,omitempty"` // Only leads with upcoming interviews
	Stale     bool   `json:"stale,omitempty"`     // Only leads not updated in N days
}

func (t *SearchLeadsTool) Definition() Definition {
	return Definition{
		Name: "lead_search",
		Description: `搜索 lead 线索。

支持多种筛选条件（可以组合使用）：
- status: 状态筛选 (new, preparing, applied, interviewing, offered, rejected)
- company: 公司名模糊匹配
- position: 职位名模糊匹配  
- priority: 优先级筛选 (1=最高, 5=最低)
- interview: 是否即将有面试 (true=只返回24小时内有面试的)
- stale: 是否长期未更新 (true=返回7天以上未更新的)
- limit: 返回数量限制 (默认10)
- sort_by: 排序方式 (priority, updated_at, created_at)

返回结果会包含：
- 线索基本信息 (公司、职位、状态)
- 关键时间点 (下次行动时间、面试时间)
- urgency 标记 (是否需要紧急处理)

示例用法：
- "搜索所有面试中的线索" → status: "interviewing"
- "查找字节跳动的岗位" → company: "字节"
- "我有哪些面试要准备" → interview: true
- "哪些线索很久没更新了" → stale: true`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status":    map[string]any{"type": "string", "description": "状态筛选"},
				"company":   map[string]any{"type": "string", "description": "公司名模糊匹配"},
				"position":  map[string]any{"type": "string", "description": "职位名模糊匹配"},
				"priority":  map[string]any{"type": "integer", "description": "优先级 1-5"},
				"limit":     map[string]any{"type": "integer", "description": "返回数量，默认10"},
				"sort_by":   map[string]any{"type": "string", "description": "排序: priority, updated_at, created_at"},
				"interview": map[string]any{"type": "boolean", "description": "只返回24小时内有面试的"},
				"stale":     map[string]any{"type": "boolean", "description": "只返回7天以上未更新的"},
			},
		},
	}
}

func (t *SearchLeadsTool) Run(_ context.Context, input json.RawMessage) (string, error) {

	var args SearchLeadsInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	// Apply defaults
	if args.Limit <= 0 {
		args.Limit = 10
	}
	if args.SortBy == "" {
		args.SortBy = "updated_at"
	}

	allLeads := t.Manager.List()
	filtered := make([]model.Lead, 0)

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)
	oneDayFromNow := now.AddDate(0, 0, 1)

	for _, l := range allLeads {
		// Status filter
		if args.Status != "" && l.Status != args.Status {
			continue
		}

		// Company filter
		if args.Company != "" && !strings.Contains(strings.ToLower(l.Company), strings.ToLower(args.Company)) {
			continue
		}

		// Position filter
		if args.Position != "" && !strings.Contains(strings.ToLower(l.Position), strings.ToLower(args.Position)) {
			continue
		}

		// Priority filter
		if args.Priority != nil && l.Priority != *args.Priority {
			continue
		}

		// Interview filter (24小时内)
		if args.Interview {
			if l.InterviewAt == "" {
				continue
			}
			interviewTime, err := time.Parse(time.RFC3339, l.InterviewAt)
			if err == nil {
				if interviewTime.Before(now) || interviewTime.After(oneDayFromNow) {
					continue
				}
			}
		}

		// Stale filter (7天以上未更新)
		if args.Stale {
			if l.UpdatedAt == "" {
				// 没有更新时间记录，视为 stale
				filtered = append(filtered, l)
				continue
			}
			updatedTime, err := time.Parse(time.RFC3339, l.UpdatedAt)
			if err == nil {
				if updatedTime.After(sevenDaysAgo) {
					continue
				}
			}
		}

		filtered = append(filtered, l)
	}

	// Sort
	switch args.SortBy {
	case "priority":
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Priority < filtered[j].Priority // 1 = highest priority
		})
	case "created_at":
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].CreatedAt == "" || filtered[j].CreatedAt == "" {
				return filtered[i].CreatedAt != ""
			}
			return filtered[i].CreatedAt > filtered[j].CreatedAt
		})
	default: // updated_at
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].UpdatedAt == "" || filtered[j].UpdatedAt == "" {
				return filtered[i].UpdatedAt != ""
			}
			return filtered[i].UpdatedAt > filtered[j].UpdatedAt
		})
	}

	// Record matched count before limit truncation
	totalMatched := len(filtered)

	// Limit
	if len(filtered) > args.Limit {
		filtered = filtered[:args.Limit]
	}

	// Transform to enriched format (not raw DB records)
	enriched := make([]map[string]any, 0, len(filtered))
	for _, l := range filtered {
		enriched = append(enriched, enrichLead(l, now))
	}

	result := map[string]any{
		"query":         args,
		"total_all":     len(allLeads),
		"total_matched": totalMatched,
		"returned":      len(enriched),
		"leads":         enriched,
		"suggestion":    generateSearchSuggestion(args, len(enriched)),
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// enrichLead transforms a lead into agent-friendly format
func enrichLead(l model.Lead, now time.Time) map[string]any {
	m := map[string]any{
		"id":          l.ID,
		"company":     l.Company,
		"position":    l.Position,
		"status":      l.Status,
		"priority":    l.Priority,
		"created_at":  l.CreatedAt,
		"updated_at":  l.UpdatedAt,
		"source":      l.Source,
		"location":    l.Location,
	}

	// Add next_action with human-readable timing
	if l.NextAction != "" {
		m["next_action"] = l.NextAction
		if l.NextActionAt != "" {
			m["next_action_at"] = l.NextActionAt
			// Compute urgency
			actionTime, err := time.Parse(time.RFC3339, l.NextActionAt)
			if err == nil {
				hoursUntil := actionTime.Sub(now).Hours()
				if hoursUntil < 0 {
					m["next_action_urgent"] = "overdue"
				} else if hoursUntil < 24 {
					m["next_action_urgent"] = "today"
				} else if hoursUntil < 72 {
					m["next_action_urgent"] = "soon"
				}
			}
		}
	}

	// Add interview info
	if l.InterviewAt != "" {
		m["interview_at"] = l.InterviewAt
		interviewTime, err := time.Parse(time.RFC3339, l.InterviewAt)
		if err == nil {
			hoursUntil := interviewTime.Sub(now).Hours()
			if hoursUntil > 0 && hoursUntil < 24 {
				m["interview_imminent"] = true
			}
		}
	}

	// Add summary (for quick overview)
	if l.Notes != "" {
		notes := l.Notes
		if len(notes) > 100 {
			notes = notes[:100] + "..."
		}
		m["notes_preview"] = notes
	}

	// Compute days since last update
	if l.UpdatedAt != "" {
		updatedTime, _ := time.Parse(time.RFC3339, l.UpdatedAt)
		daysSince := int(now.Sub(updatedTime).Hours() / 24)
		m["days_since_update"] = daysSince
		if daysSince > 7 {
			m["stale"] = true
		}
	}

	return m
}

func generateSearchSuggestion(args SearchLeadsInput, count int) string {
	if count == 0 {
		if args.Stale {
			return "没有发现长期未更新的线索，这说明你跟进得很及时！👍"
		}
		if args.Interview {
			return "24小时内没有面试安排，当前是积累阶段，继续加油！"
		}
		return "没有找到匹配的线索，尝试放宽筛选条件？"
	}

	if args.Status != "" {
		switch args.Status {
		case "interviewing":
			return "这些都是面试流程中的机会，重点跟进！"
		case "applied":
			return "已投递，等待回复中。可以适当催一下进度。"
		case "preparing":
			return "准备中的岗位，尽早投递可以抢占先机。"
		}
	}

	if args.Stale {
		return "这些线索需要你关注一下，可能错过了最佳跟进时机。"
	}

	if args.Interview {
		return "好好准备！面试是求职的关键环节。"
	}

	return ""
}