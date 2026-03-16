package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"trace2offer/backend/internal/stats"
)

// StatsSummaryProvider supplies stats summary for insight tools.
type StatsSummaryProvider interface {
	GetSummary() stats.SummaryStats
}

// NewInsightTools builds data-insight tools used by the agent.
func NewInsightTools(provider StatsSummaryProvider) []Tool {
	if provider == nil {
		return nil
	}
	return []Tool{
		&statsSummaryTool{provider: provider},
		&jobStrategyTool{provider: provider},
	}
}

type statsSummaryTool struct {
	provider StatsSummaryProvider
}

func (t *statsSummaryTool) Definition() Definition {
	return Definition{
		Name:        "stats_summary",
		Description: "获取当前线索统计总览、转化漏斗、渠道表现、停留时长和洞察建议",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *statsSummaryTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if t == nil || t.provider == nil {
		return "", fmt.Errorf("stats provider is unavailable")
	}
	if err := decodeOptionalObject(input); err != nil {
		return "", err
	}

	summary := t.provider.GetSummary()
	return marshalOutput(map[string]any{
		"summary": summary,
	})
}

type jobStrategyTool struct {
	provider StatsSummaryProvider
}

type jobStrategyInput struct {
	FocusRoles      []string `json:"focus_roles"`
	FocusLocations  []string `json:"focus_locations"`
	FocusIndustries []string `json:"focus_industries"`
	HorizonDays     int      `json:"horizon_days"`
}

type strategyPriority struct {
	Category string   `json:"category"`
	Title    string   `json:"title"`
	Reason   string   `json:"reason"`
	Actions  []string `json:"actions"`
	Impact   string   `json:"impact"`
	Urgency  string   `json:"urgency"`
}

func (t *jobStrategyTool) Definition() Definition {
	return Definition{
		Name:        "job_search_strategy",
		Description: "基于当前求职数据生成阶段性投递策略与执行优先级",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"focus_roles": map[string]any{
					"type":        "array",
					"description": "当前重点投递岗位方向",
					"items":       map[string]any{"type": "string"},
				},
				"focus_locations": map[string]any{
					"type":        "array",
					"description": "当前重点地区",
					"items":       map[string]any{"type": "string"},
				},
				"focus_industries": map[string]any{
					"type":        "array",
					"description": "当前重点行业",
					"items":       map[string]any{"type": "string"},
				},
				"horizon_days": map[string]any{
					"type":        "integer",
					"description": "策略执行周期（天）",
				},
			},
		},
	}
}

func (t *jobStrategyTool) Run(_ context.Context, input json.RawMessage) (string, error) {
	if t == nil || t.provider == nil {
		return "", fmt.Errorf("stats provider is unavailable")
	}

	var args jobStrategyInput
	if err := decodeInput(input, &args); err != nil {
		return "", err
	}
	args.FocusRoles = normalizeStringList(args.FocusRoles)
	args.FocusLocations = normalizeStringList(args.FocusLocations)
	args.FocusIndustries = normalizeStringList(args.FocusIndustries)
	if args.HorizonDays <= 0 {
		args.HorizonDays = 14
	}

	summary := t.provider.GetSummary()
	priorities := buildStrategyPriorities(summary, args)

	return marshalOutput(map[string]any{
		"horizon_days": args.HorizonDays,
		"profile_focus": map[string]any{
			"roles":      args.FocusRoles,
			"locations":  args.FocusLocations,
			"industries": args.FocusIndustries,
		},
		"overview": map[string]any{
			"total":         summary.Overview.Total,
			"active":        summary.Overview.Active,
			"success_rate":  summary.Overview.SuccessRate,
			"this_week_new": summary.Overview.ThisWeekNew,
			"urgent_count":  summary.Insights.Urgent,
		},
		"priorities": priorities,
	})
}

func buildStrategyPriorities(summary stats.SummaryStats, args jobStrategyInput) []strategyPriority {
	priorities := make([]strategyPriority, 0, 5)

	if summary.Insights.Urgent > 0 {
		priorities = append(priorities, strategyPriority{
			Category: "follow_up",
			Title:    "优先清理高风险跟进项",
			Reason:   fmt.Sprintf("当前有 %d 条紧急洞察，先把会错过窗口期的线索处理掉。", summary.Insights.Urgent),
			Actions: []string{
				"优先处理高优先级且停滞的线索，24 小时内完成一次主动跟进",
				"对面试中线索设置明确截止时间和下一步动作",
			},
			Impact:  "降低机会流失率",
			Urgency: "high",
		})
	}

	if summary.Funnel.Conversion > 0 && summary.Funnel.Conversion < 15 {
		priorities = append(priorities, strategyPriority{
			Category: "conversion",
			Title:    "先提转化质量，再扩线索数量",
			Reason:   fmt.Sprintf("当前漏斗转化率 %.1f%% 偏低，盲目加投递只会放大低效。", summary.Funnel.Conversion),
			Actions: []string{
				"把目标岗位拆成 1-2 个主赛道，重写对应版本简历",
				"每周复盘被拒/卡住原因，沉淀 3 条可执行改进点",
			},
			Impact:  "提升投递到面试的转化",
			Urgency: "high",
		})
	}

	if summary.Overview.ThisWeekNew == 0 && summary.Overview.Active < 8 {
		priorities = append(priorities, strategyPriority{
			Category: "pipeline",
			Title:    "补充新线索，避免管道断流",
			Reason:   "本周新增线索为 0，且活跃线索池偏小，后续容易出现空窗期。",
			Actions: []string{
				"按当前重点方向每天新增 2-3 条高质量线索",
				"优先投递与你最近成功阶段相似的岗位画像",
			},
			Impact:  "维持稳定面试节奏",
			Urgency: "medium",
		})
	}

	if summary.Sources.BestSource != "" {
		priorities = append(priorities, strategyPriority{
			Category: "channel",
			Title:    "提高高回报渠道占比",
			Reason:   fmt.Sprintf("当前最优渠道是 %s，应增加同类渠道投递配比。", summary.Sources.BestSource),
			Actions: []string{
				"把高回报渠道作为第一优先级，至少占新增线索的 50%",
				"对低回报渠道设定止损阈值，连续两周无反馈就降权",
			},
			Impact:  "在相同投入下提高回报",
			Urgency: "medium",
		})
	}

	if summary.Duration.SlowestStatus != "" && summary.Duration.AverageActiveDays > 7 {
		priorities = append(priorities, strategyPriority{
			Category: "cadence",
			Title:    "缩短线索停留周期",
			Reason:   fmt.Sprintf("活跃线索平均停留 %.1f 天，流程节奏偏慢。", summary.Duration.AverageActiveDays),
			Actions: []string{
				"为每条线索设定明确 next_action 和截止时间",
				"连续两次无反馈的线索切换策略（换联系人或换岗位）",
			},
			Impact:  "提升整体推进速度",
			Urgency: "medium",
		})
	}

	if len(priorities) == 0 {
		priorities = append(priorities, strategyPriority{
			Category: "maintain",
			Title:    "保持当前节奏并持续小步优化",
			Reason:   "当前数据没有明显风险点，适合稳定执行并滚动优化。",
			Actions: []string{
				"继续按既定节奏推进投递和跟进",
				"每周固定复盘一次渠道与阶段转化",
			},
			Impact:  "稳定产出、控制波动",
			Urgency: "low",
		})
	}

	if len(args.FocusRoles) > 0 || len(args.FocusLocations) > 0 || len(args.FocusIndustries) > 0 {
		focusText := buildFocusText(args)
		priorities[0].Reason = priorities[0].Reason + " 当前偏好聚焦为：" + focusText + "。"
	}

	return priorities
}

func buildFocusText(args jobStrategyInput) string {
	parts := make([]string, 0, 3)
	if len(args.FocusRoles) > 0 {
		parts = append(parts, "岗位="+strings.Join(args.FocusRoles, "/"))
	}
	if len(args.FocusLocations) > 0 {
		parts = append(parts, "地点="+strings.Join(args.FocusLocations, "/"))
	}
	if len(args.FocusIndustries) > 0 {
		parts = append(parts, "行业="+strings.Join(args.FocusIndustries, "/"))
	}
	if len(parts) == 0 {
		return "未设置"
	}
	return strings.Join(parts, "，")
}

func decodeOptionalObject(raw json.RawMessage) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "{}" {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(trimmed), &args); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
