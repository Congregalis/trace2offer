package stats

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	leadpkg "trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
)

// Status labels for display
var statusLabels = map[string]string{
	leadpkg.StatusNew:          "New",
	leadpkg.StatusPreparing:    "Preparing",
	leadpkg.StatusApplied:      "Applied",
	leadpkg.StatusInterviewing: "Interviewing",
	leadpkg.StatusOffered:      "Offered",
	leadpkg.StatusDeclined:     "Declined",
	leadpkg.StatusRejected:     "Rejected",
	leadpkg.StatusArchived:     "Archived",
}

// Calculator computes statistics from leads.
type Calculator struct {
	leads []model.Lead
	now   time.Time
}

// NewCalculator creates a new statistics calculator.
func NewCalculator(leads []model.Lead) *Calculator {
	return &Calculator{
		leads: leads,
		now:   time.Now().UTC(),
	}
}

// CalculateOverview computes high-level overview statistics.
func (c *Calculator) CalculateOverview() OverviewStats {
	total := len(c.leads)
	active := 0
	offered := 0
	thisWeekNew := 0

	weekStart := c.now.AddDate(0, 0, -int(c.now.Weekday()))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.UTC)

	for _, lead := range c.leads {
		// Count active (not terminal states)
		if lead.Status != leadpkg.StatusOffered &&
			lead.Status != leadpkg.StatusDeclined &&
			lead.Status != leadpkg.StatusRejected &&
			lead.Status != leadpkg.StatusArchived {
			active++
		}

		// Count offered
		if lead.Status == leadpkg.StatusOffered {
			offered++
		}

		// Count this week's new leads
		if createdAt, err := time.Parse(time.RFC3339, lead.CreatedAt); err == nil {
			if createdAt.After(weekStart) || createdAt.Equal(weekStart) {
				thisWeekNew++
			}
		}
	}

	successRate := 0.0
	if total > 0 {
		// Success rate = offered / (offered + rejected + declined)
		terminalCount := offered
		for _, lead := range c.leads {
			if lead.Status == leadpkg.StatusRejected || lead.Status == leadpkg.StatusDeclined {
				terminalCount++
			}
		}
		if terminalCount > 0 {
			successRate = math.Round(float64(offered)/float64(terminalCount)*1000) / 10
		}
	}

	return OverviewStats{
		Total:       total,
		Active:      active,
		Offered:     offered,
		SuccessRate: successRate,
		ThisWeekNew: thisWeekNew,
		LastUpdated: c.now.Format(time.RFC3339),
	}
}

// CalculateFunnel computes the conversion funnel statistics.
func (c *Calculator) CalculateFunnel() FunnelStats {
	// Define funnel stages in order
	stages := []struct {
		status string
		label  string
	}{
		{leadpkg.StatusNew, statusLabels[leadpkg.StatusNew]},
		{leadpkg.StatusPreparing, statusLabels[leadpkg.StatusPreparing]},
		{leadpkg.StatusApplied, statusLabels[leadpkg.StatusApplied]},
		{leadpkg.StatusInterviewing, statusLabels[leadpkg.StatusInterviewing]},
		{leadpkg.StatusOffered, statusLabels[leadpkg.StatusOffered]},
	}

	// Count leads in each stage
	stageCounts := make(map[string]int)
	for _, lead := range c.leads {
		stageCounts[lead.Status]++
	}

	// Calculate funnel stages
	funnelStages := make([]FunnelStage, 0, len(stages))
	total := len(c.leads)

	for _, s := range stages {
		count := stageCounts[s.status]
		percentage := 0.0
		if total > 0 {
			percentage = math.Round(float64(count)/float64(total)*1000) / 10
		}

		// Calculate average days in this stage (simplified - would need transition tracking for accuracy)
		avgDays := c.calculateAvgDaysInStatus(s.status)

		funnelStages = append(funnelStages, FunnelStage{
			Status:     s.status,
			Label:      s.label,
			Count:      count,
			Percentage: percentage,
			AvgDays:    avgDays,
		})
	}

	// Calculate overall conversion rate (new -> offered)
	conversion := 0.0
	if total > 0 && stageCounts[leadpkg.StatusNew] > 0 {
		conversion = math.Round(float64(stageCounts[leadpkg.StatusOffered])/float64(total)*1000) / 10
	}

	return FunnelStats{
		Stages:     funnelStages,
		Conversion: conversion,
		TotalTime:  c.calculateTotalCycleTime(),
	}
}

// calculateAvgDaysInStatus estimates average days leads stay in a status.
// This is a simplified calculation - in production, you'd track status transitions.
func (c *Calculator) calculateAvgDaysInStatus(status string) float64 {
	var totalDays float64
	var count int

	for _, lead := range c.leads {
		if lead.Status != status {
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, lead.CreatedAt)
		if err != nil {
			continue
		}

		updatedAt, err := time.Parse(time.RFC3339, lead.UpdatedAt)
		if err != nil {
			updatedAt = c.now
		}

		days := updatedAt.Sub(createdAt).Hours() / 24
		totalDays += days
		count++
	}

	if count == 0 {
		return 0
	}
	return math.Round(totalDays/float64(count)*10) / 10
}

// calculateTotalCycleTime estimates average total time from new to terminal status.
func (c *Calculator) calculateTotalCycleTime() float64 {
	var totalDays float64
	var count int

	terminalStatuses := map[string]bool{
		leadpkg.StatusOffered:  true,
		leadpkg.StatusDeclined: true,
		leadpkg.StatusRejected: true,
		leadpkg.StatusArchived: true,
	}

	for _, lead := range c.leads {
		if !terminalStatuses[lead.Status] {
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, lead.CreatedAt)
		if err != nil {
			continue
		}

		days := c.now.Sub(createdAt).Hours() / 24
		totalDays += days
		count++
	}

	if count == 0 {
		return 0
	}
	return math.Round(totalDays/float64(count)*10) / 10
}

// CalculateSources computes source/channel analysis.
func (c *Calculator) CalculateSources() SourceAnalysis {
	sourceMap := make(map[string]*SourceStats)

	for _, lead := range c.leads {
		source := strings.TrimSpace(lead.Source)
		if source == "" {
			source = "Unknown"
		}

		if _, exists := sourceMap[source]; !exists {
			sourceMap[source] = &SourceStats{
				Source: source,
			}
		}

		stats := sourceMap[source]
		stats.Count++

		// Track progression
		if lead.Status == leadpkg.StatusApplied {
			stats.Applied++
		}
		if lead.Status == leadpkg.StatusInterviewing {
			stats.Interviewed++
		}
		if lead.Status == leadpkg.StatusOffered {
			stats.Offered++
		}
	}

	// Convert to slice and calculate percentages
	total := len(c.leads)
	sources := make([]SourceStats, 0, len(sourceMap))

	for _, stats := range sourceMap {
		if total > 0 {
			stats.Percentage = math.Round(float64(stats.Count)/float64(total)*1000) / 10
		}
		if stats.Count > 0 {
			stats.SuccessRate = math.Round(float64(stats.Offered)/float64(stats.Count)*1000) / 10
		}
		sources = append(sources, *stats)
	}

	// Sort by count descending
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Count > sources[j].Count
	})

	// Identify top and best source
	topSource := ""
	bestSource := ""

	if len(sources) > 0 {
		topSource = sources[0].Source

		// Best source has at least 2 leads and highest success rate
		bestRate := -1.0
		for _, s := range sources {
			if s.Count >= 2 && s.SuccessRate > bestRate {
				bestRate = s.SuccessRate
				bestSource = s.Source
			}
		}
	}

	return SourceAnalysis{
		Sources:    sources,
		TopSource:  topSource,
		BestSource: bestSource,
	}
}

// CalculateTrends computes time-series trends for leads.
func (c *Calculator) CalculateTrends(period string) TrendStats {
	// Default to 4 weeks
	days := 28
	if period == "month" {
		days = 30
	} else if period == "quarter" {
		days = 90
	}

	// Generate time points
	points := make([]TimePoint, 0)
	now := c.now

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		// Count new leads on this date
		newCount := 0
		movedCount := 0

		for _, lead := range c.leads {
			// Check if created on this date
			createdAt, err := time.Parse(time.RFC3339, lead.CreatedAt)
			if err == nil {
				if createdAt.Format("2006-01-02") == dateStr {
					newCount++
				}
			}

			// Check if status changed on this date (simplified - check updated)
			updatedAt, err := time.Parse(time.RFC3339, lead.UpdatedAt)
			if err == nil {
				if updatedAt.Format("2006-01-02") == dateStr {
					// Only count as "moved" if not new
					createdAt, _ := time.Parse(time.RFC3339, lead.CreatedAt)
					if createdAt.Format("2006-01-02") != dateStr {
						movedCount++
					}
				}
			}
		}

		// Calculate total active leads up to this date
		totalActive := 0
		for _, lead := range c.leads {
			createdAt, err := time.Parse(time.RFC3339, lead.CreatedAt)
			if err != nil {
				continue
			}
			// If lead was created on or before this date
			if createdAt.Before(date) || createdAt.Equal(date) {
				// And hasn't been archived/rejected/declined before this date
				if lead.Status != leadpkg.StatusArchived &&
					lead.Status != leadpkg.StatusRejected &&
					lead.Status != leadpkg.StatusDeclined {
					totalActive++
				}
			}
		}

		label := date.Format("1/2")
		if i == days-1 {
			label = "Start"
		} else if i == 0 {
			label = "Today"
		}

		points = append(points, TimePoint{
			Date:  dateStr,
			Label: label,
			New:   newCount,
			Moved: movedCount,
			Total: totalActive,
		})
	}

	// Calculate growth rate
	growth := 0.0
	isGrowing := false

	if len(points) >= 7 {
		firstWeek := 0
		lastWeek := 0
		for i, p := range points {
			if i < 7 {
				firstWeek += p.New
			}
			if i >= len(points)-7 {
				lastWeek += p.New
			}
		}
		if firstWeek > 0 {
			growth = math.Round(float64(lastWeek-firstWeek)/float64(firstWeek)*1000) / 10
		}
		isGrowing = lastWeek >= firstWeek
	}

	return TrendStats{
		Period:    period,
		Points:    points,
		Growth:    growth,
		IsGrowing: isGrowing,
	}
}

// CalculateInsights generates AI-style insights based on statistics.
func (c *Calculator) CalculateInsights() InsightStats {
	insights := make([]InsightItem, 0)

	// Get basic stats for analysis
	overview := c.CalculateOverview()
	sources := c.CalculateSources()
	funnel := c.CalculateFunnel()

	// 1. Check for stagnant high-priority leads
	highPriorityStagnant := 0
	for _, lead := range c.leads {
		if lead.Priority >= 4 && lead.Status != leadpkg.StatusOffered &&
			lead.Status != leadpkg.StatusRejected &&
			lead.Status != leadpkg.StatusDeclined &&
			lead.Status != leadpkg.StatusArchived {
			// Check if updated recently
			updatedAt, err := time.Parse(time.RFC3339, lead.UpdatedAt)
			if err == nil {
				daysSince := c.now.Sub(updatedAt).Hours() / 24
				if daysSince > 3 {
					highPriorityStagnant++
				}
			}
		}
	}

	if highPriorityStagnant > 0 {
		severity := "warning"
		if highPriorityStagnant >= 3 {
			severity = "critical"
		}
		insights = append(insights, InsightItem{
			Type:     "stagnation",
			Severity: severity,
			Title:    "高优先级线索需要跟进",
			Message:  fmt.Sprintf("你有 %d 个高优先级线索已经超过 3 天没有更新，建议立即跟进。", highPriorityStagnant),
			Action:   "查看高优先级线索",
		})
	}

	// 2. Analyze conversion rate
	if overview.Total > 5 {
		if funnel.Conversion < 10 {
			insights = append(insights, InsightItem{
				Type:     "performance",
				Severity: "warning",
				Title:    "转化率偏低",
				Message:  fmt.Sprintf("你的整体转化率只有 %.1f%%，建议优化简历和求职策略。", funnel.Conversion),
				Action:   "查看转化率分析",
			})
		} else if funnel.Conversion > 30 {
			insights = append(insights, InsightItem{
				Type:     "performance",
				Severity: "info",
				Title:    "转化率优秀",
				Message:  fmt.Sprintf("你的转化率达到了 %.1f%%，表现非常出色！", funnel.Conversion),
				Action:   "继续保持",
			})
		}
	}

	// 3. Source analysis
	if sources.BestSource != "" && sources.BestSource != sources.TopSource {
		bestRate := 0.0
		for _, s := range sources.Sources {
			if s.Source == sources.BestSource {
				bestRate = s.SuccessRate
				break
			}
		}

		insights = append(insights, InsightItem{
			Type:     "source",
			Severity: "info",
			Title:    "高效渠道发现",
			Message:  fmt.Sprintf("%s 渠道的转化率最高 (%.1f%%)，建议优先使用此渠道。", sources.BestSource, bestRate),
			Action:   "查看渠道分析",
		})
	}

	// 4. Activity check
	if overview.ThisWeekNew == 0 && overview.Total > 0 {
		insights = append(insights, InsightItem{
			Type:     "activity",
			Severity: "warning",
			Title:    "本周无新增线索",
			Message:  "你本周还没有添加任何新线索，建议花些时间寻找新机会。",
			Action:   "添加新线索",
		})
	}

	// 5. Success celebration
	if overview.Offered > 0 {
		// Check if there's a recent offer (within last 7 days)
		recentOffer := false
		for _, lead := range c.leads {
			if lead.Status == leadpkg.StatusOffered {
				updatedAt, err := time.Parse(time.RFC3339, lead.UpdatedAt)
				if err == nil {
					daysSince := c.now.Sub(updatedAt).Hours() / 24
					if daysSince <= 7 {
						recentOffer = true
						break
					}
				}
			}
		}

		if recentOffer {
			insights = append(insights, InsightItem{
				Type:     "milestone",
				Severity: "success",
				Title:    "🎉 恭喜获得 Offer！",
				Message:  "你最近收到了一个 Offer，这是对你努力的认可！",
				Action:   "查看 Offer 详情",
			})
		}
	}

	// Count by severity
	urgent := 0
	for _, insight := range insights {
		if insight.Severity == "critical" || insight.Severity == "warning" {
			urgent++
		}
	}

	return InsightStats{
		Total:     len(insights),
		Urgent:    urgent,
		Tips:      len(insights) - urgent,
		Insights:  insights,
		Generated: c.now.Format(time.RFC3339),
	}
}

// CalculateSummary computes all statistics and returns a complete summary.
func (c *Calculator) CalculateSummary() SummaryStats {
	return SummaryStats{
		Overview:  c.CalculateOverview(),
		Funnel:    c.CalculateFunnel(),
		Sources:   c.CalculateSources(),
		Trends:    c.CalculateTrends("month"),
		Insights:  c.CalculateInsights(),
		Generated: c.now,
	}
}
