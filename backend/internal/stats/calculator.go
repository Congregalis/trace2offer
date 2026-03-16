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

// Status labels for display.
var statusLabels = map[string]string{
	leadpkg.StatusNew:          "新线索",
	leadpkg.StatusPreparing:    "准备中",
	leadpkg.StatusApplied:      "已投递",
	leadpkg.StatusInterviewing: "面试中",
	leadpkg.StatusOffered:      "已获 Offer",
	leadpkg.StatusDeclined:     "已拒绝",
	leadpkg.StatusRejected:     "被拒绝",
	leadpkg.StatusArchived:     "已归档",
}

var orderedStatuses = []string{
	leadpkg.StatusNew,
	leadpkg.StatusPreparing,
	leadpkg.StatusApplied,
	leadpkg.StatusInterviewing,
	leadpkg.StatusOffered,
	leadpkg.StatusDeclined,
	leadpkg.StatusRejected,
	leadpkg.StatusArchived,
}

var funnelStatuses = []string{
	leadpkg.StatusNew,
	leadpkg.StatusPreparing,
	leadpkg.StatusApplied,
	leadpkg.StatusInterviewing,
	leadpkg.StatusOffered,
}

var terminalStatusSet = map[string]struct{}{
	leadpkg.StatusOffered:  {},
	leadpkg.StatusDeclined: {},
	leadpkg.StatusRejected: {},
	leadpkg.StatusArchived: {},
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
	statusCountsMap := make(map[string]int, len(orderedStatuses))

	for _, status := range orderedStatuses {
		statusCountsMap[status] = 0
	}

	weekStart := startOfWeekUTC(c.now)

	for _, lead := range c.leads {
		status := strings.TrimSpace(lead.Status)
		if _, ok := statusCountsMap[status]; ok {
			statusCountsMap[status]++
		}

		if isActiveStatus(status) {
			active++
		}
		if status == leadpkg.StatusOffered {
			offered++
		}

		if createdAt, ok := parseRFC3339(lead.CreatedAt); ok {
			if !createdAt.Before(weekStart) {
				thisWeekNew++
			}
		}
	}

	successRate := 0.0
	successPopulation := statusCountsMap[leadpkg.StatusOffered] + statusCountsMap[leadpkg.StatusDeclined] + statusCountsMap[leadpkg.StatusRejected]
	if successPopulation > 0 {
		successRate = round1(float64(statusCountsMap[leadpkg.StatusOffered]) / float64(successPopulation) * 100)
	}

	statusCounts := make([]StatusCount, 0, len(orderedStatuses))
	for _, status := range orderedStatuses {
		count := statusCountsMap[status]
		percentage := 0.0
		if total > 0 {
			percentage = round1(float64(count) / float64(total) * 100)
		}
		statusCounts = append(statusCounts, StatusCount{
			Status:     status,
			Label:      statusLabels[status],
			Count:      count,
			Percentage: percentage,
		})
	}

	return OverviewStats{
		Total:        total,
		Active:       active,
		Offered:      offered,
		SuccessRate:  successRate,
		ThisWeekNew:  thisWeekNew,
		StatusCounts: statusCounts,
		LastUpdated:  c.now.Format(time.RFC3339),
	}
}

// CalculateFunnel computes the conversion funnel statistics.
func (c *Calculator) CalculateFunnel() FunnelStats {
	stageCounts := make(map[string]int, len(funnelStatuses))
	for _, status := range funnelStatuses {
		stageCounts[status] = 0
	}

	for _, lead := range c.leads {
		status := strings.TrimSpace(lead.Status)
		if _, ok := stageCounts[status]; ok {
			stageCounts[status]++
		}
	}

	cumulative := make([]int, len(funnelStatuses))
	running := 0
	for i := len(funnelStatuses) - 1; i >= 0; i-- {
		running += stageCounts[funnelStatuses[i]]
		cumulative[i] = running
	}

	entryCount := 0
	if len(cumulative) > 0 {
		entryCount = cumulative[0]
	}

	funnelStages := make([]FunnelStage, 0, len(funnelStatuses))
	for i, status := range funnelStatuses {
		stageCount := stageCounts[status]
		stageCumulative := cumulative[i]

		percentage := 0.0
		if entryCount > 0 {
			percentage = round1(float64(stageCumulative) / float64(entryCount) * 100)
		}

		conversionFromPrev := 0.0
		if i == 0 {
			if entryCount > 0 {
				conversionFromPrev = 100
			}
		} else if cumulative[i-1] > 0 {
			conversionFromPrev = round1(float64(stageCumulative) / float64(cumulative[i-1]) * 100)
		}

		funnelStages = append(funnelStages, FunnelStage{
			Status:             status,
			Label:              statusLabels[status],
			Count:              stageCount,
			CumulativeCount:    stageCumulative,
			Percentage:         percentage,
			ConversionFromPrev: conversionFromPrev,
			AvgDays:            c.calculateAvgDaysInStatus(status),
		})
	}

	conversion := 0.0
	if entryCount > 0 {
		conversion = round1(float64(stageCounts[leadpkg.StatusOffered]) / float64(entryCount) * 100)
	}

	return FunnelStats{
		Stages:     funnelStages,
		Conversion: conversion,
		TotalTime:  c.calculateTotalCycleTime(),
	}
}

// CalculateDuration computes average dwell-time analysis by status.
func (c *Calculator) CalculateDuration() DurationStats {
	byStatus := make([]DurationStatus, 0, len(funnelStatuses))
	slowestStatus := ""
	slowestLabel := ""
	slowestDays := -1.0

	for _, status := range funnelStatuses {
		count := 0
		for _, lead := range c.leads {
			if strings.TrimSpace(lead.Status) == status {
				count++
			}
		}

		avgDays := c.calculateAvgDaysInStatus(status)
		byStatus = append(byStatus, DurationStatus{
			Status:  status,
			Label:   statusLabels[status],
			Count:   count,
			AvgDays: avgDays,
		})

		if count > 0 && avgDays > slowestDays {
			slowestDays = avgDays
			slowestStatus = status
			slowestLabel = statusLabels[status]
		}
	}

	activeDaysTotal := 0.0
	activeCount := 0
	for _, lead := range c.leads {
		status := strings.TrimSpace(lead.Status)
		if !isActiveStatus(status) {
			continue
		}

		updatedAt := c.resolveLeadUpdatedAt(lead)
		activeDaysTotal += durationDays(updatedAt, c.now)
		activeCount++
	}

	averageActiveDays := 0.0
	if activeCount > 0 {
		averageActiveDays = round1(activeDaysTotal / float64(activeCount))
	}

	return DurationStats{
		AverageCycleDays:  c.calculateTotalCycleTime(),
		AverageActiveDays: averageActiveDays,
		SlowestStatus:     slowestStatus,
		SlowestLabel:      slowestLabel,
		ByStatus:          byStatus,
	}
}

// calculateAvgDaysInStatus estimates average dwell days in current status.
func (c *Calculator) calculateAvgDaysInStatus(status string) float64 {
	totalDays := 0.0
	count := 0

	for _, lead := range c.leads {
		if strings.TrimSpace(lead.Status) != status {
			continue
		}

		updatedAt := c.resolveLeadUpdatedAt(lead)
		totalDays += durationDays(updatedAt, c.now)
		count++
	}

	if count == 0 {
		return 0
	}

	return round1(totalDays / float64(count))
}

// calculateTotalCycleTime estimates average total time from creation to terminal status.
func (c *Calculator) calculateTotalCycleTime() float64 {
	totalDays := 0.0
	count := 0

	for _, lead := range c.leads {
		status := strings.TrimSpace(lead.Status)
		if _, terminal := terminalStatusSet[status]; !terminal {
			continue
		}

		createdAt, ok := parseRFC3339(lead.CreatedAt)
		if !ok {
			continue
		}

		endedAt := c.resolveLeadUpdatedAt(lead)
		if endedAt.Before(createdAt) {
			continue
		}

		totalDays += durationDays(createdAt, endedAt)
		count++
	}

	if count == 0 {
		return 0
	}

	return round1(totalDays / float64(count))
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

	total := len(c.leads)
	sources := make([]SourceStats, 0, len(sourceMap))

	for _, stats := range sourceMap {
		if total > 0 {
			stats.Percentage = round1(float64(stats.Count) / float64(total) * 100)
		}
		if stats.Count > 0 {
			stats.SuccessRate = round1(float64(stats.Offered) / float64(stats.Count) * 100)
		}
		sources = append(sources, *stats)
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Count > sources[j].Count
	})

	topSource := ""
	bestSource := ""

	if len(sources) > 0 {
		topSource = sources[0].Source

		bestRate := -1.0
		for _, item := range sources {
			if item.Count >= 2 && item.SuccessRate > bestRate {
				bestRate = item.SuccessRate
				bestSource = item.Source
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
	normalizedPeriod := normalizePeriod(period)
	days := periodDays(normalizedPeriod)

	points := make([]TimePoint, 0, days)
	endOfToday := startOfDayUTC(c.now)

	for i := days - 1; i >= 0; i-- {
		dayStart := endOfToday.AddDate(0, 0, -i)
		dayEnd := dayStart.Add(24 * time.Hour)
		dateKey := dayStart.Format("2006-01-02")

		newCount := 0
		movedCount := 0
		totalActive := 0

		for _, lead := range c.leads {
			createdAt, createdOK := parseRFC3339(lead.CreatedAt)
			if createdOK && !createdAt.Before(dayStart) && createdAt.Before(dayEnd) {
				newCount++
			}

			updatedAt, updatedOK := parseRFC3339(lead.UpdatedAt)
			if updatedOK && !updatedAt.Before(dayStart) && updatedAt.Before(dayEnd) {
				createdSameDay := createdOK && createdAt.Format("2006-01-02") == dateKey
				if !createdSameDay {
					movedCount++
				}
			}

			if createdOK && createdAt.Before(dayEnd) && isActiveStatus(lead.Status) {
				totalActive++
			}
		}

		label := dayStart.Format("1/2")
		if normalizedPeriod == "week" {
			label = dayStart.Format("Mon")
		}

		points = append(points, TimePoint{
			Date:  dateKey,
			Label: label,
			New:   newCount,
			Moved: movedCount,
			Total: totalActive,
		})
	}

	growth, isGrowing := calculateGrowth(points)

	return TrendStats{
		Period:    normalizedPeriod,
		Points:    points,
		Growth:    growth,
		IsGrowing: isGrowing,
	}
}

// CalculateInsights generates AI-style insights based on statistics.
func (c *Calculator) CalculateInsights() InsightStats {
	insights := make([]InsightItem, 0)

	overview := c.CalculateOverview()
	sources := c.CalculateSources()
	funnel := c.CalculateFunnel()
	duration := c.CalculateDuration()

	highPriorityStagnant := 0
	for _, lead := range c.leads {
		status := strings.TrimSpace(lead.Status)
		if lead.Priority < 4 || !isActiveStatus(status) {
			continue
		}

		updatedAt := c.resolveLeadUpdatedAt(lead)
		daysSince := durationDays(updatedAt, c.now)
		if daysSince > 3 {
			highPriorityStagnant++
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
				Message:  fmt.Sprintf("你的转化率达到了 %.1f%%，表现非常出色。", funnel.Conversion),
				Action:   "继续保持",
			})
		}
	}

	for _, item := range duration.ByStatus {
		if item.Status == leadpkg.StatusInterviewing && item.Count > 0 && item.AvgDays >= 14 {
			insights = append(insights, InsightItem{
				Type:     "duration",
				Severity: "warning",
				Title:    "面试阶段停留偏久",
				Message:  fmt.Sprintf("面试阶段平均停留 %.1f 天，建议主动跟进面试反馈。", item.AvgDays),
				Action:   "查看面试中线索",
			})
			break
		}
	}

	if sources.BestSource != "" && sources.BestSource != sources.TopSource {
		bestRate := 0.0
		for _, item := range sources.Sources {
			if item.Source == sources.BestSource {
				bestRate = item.SuccessRate
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

	if overview.ThisWeekNew == 0 && overview.Total > 0 {
		insights = append(insights, InsightItem{
			Type:     "activity",
			Severity: "warning",
			Title:    "本周无新增线索",
			Message:  "你本周还没有添加任何新线索，建议花些时间寻找新机会。",
			Action:   "添加新线索",
		})
	}

	if overview.Offered > 0 {
		recentOffer := false
		for _, lead := range c.leads {
			if strings.TrimSpace(lead.Status) != leadpkg.StatusOffered {
				continue
			}
			updatedAt := c.resolveLeadUpdatedAt(lead)
			if durationDays(updatedAt, c.now) <= 7 {
				recentOffer = true
				break
			}
		}

		if recentOffer {
			insights = append(insights, InsightItem{
				Type:     "milestone",
				Severity: "success",
				Title:    "恭喜获得 Offer",
				Message:  "你最近收到了一个 Offer，这是对你努力的认可。",
				Action:   "查看 Offer 详情",
			})
		}
	}

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
		Duration:  c.CalculateDuration(),
		Insights:  c.CalculateInsights(),
		Generated: c.now,
	}
}

// CalculateDashboard computes the core dashboard payload.
func (c *Calculator) CalculateDashboard() DashboardStats {
	return DashboardStats{
		Overview:     c.CalculateOverview(),
		Funnel:       c.CalculateFunnel(),
		WeeklyTrend:  c.CalculateTrends("week"),
		MonthlyTrend: c.CalculateTrends("month"),
		Duration:     c.CalculateDuration(),
		Generated:    c.now,
	}
}

func (c *Calculator) resolveLeadUpdatedAt(lead model.Lead) time.Time {
	if updatedAt, ok := parseRFC3339(lead.UpdatedAt); ok {
		return updatedAt
	}
	if createdAt, ok := parseRFC3339(lead.CreatedAt); ok {
		return createdAt
	}
	return c.now
}

func isActiveStatus(status string) bool {
	_, terminal := terminalStatusSet[strings.TrimSpace(status)]
	return !terminal
}

func normalizePeriod(period string) string {
	switch strings.TrimSpace(strings.ToLower(period)) {
	case "week":
		return "week"
	case "quarter":
		return "quarter"
	case "month", "monthly", "":
		return "month"
	default:
		return "month"
	}
}

func periodDays(period string) int {
	switch period {
	case "week":
		return 7
	case "quarter":
		return 90
	case "month":
		fallthrough
	default:
		return 30
	}
}

func calculateGrowth(points []TimePoint) (float64, bool) {
	if len(points) < 2 {
		return 0, false
	}

	window := 7
	if len(points) < 14 {
		window = len(points) / 2
	}
	if window == 0 {
		return 0, false
	}

	firstWindow := 0
	lastWindow := 0
	for i := 0; i < window; i++ {
		firstWindow += points[i].New
		lastWindow += points[len(points)-window+i].New
	}

	growth := 0.0
	if firstWindow > 0 {
		growth = round1(float64(lastWindow-firstWindow) / float64(firstWindow) * 100)
	} else if lastWindow > 0 {
		growth = 100
	}

	return growth, lastWindow >= firstWindow
}

func parseRFC3339(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func startOfWeekUTC(ts time.Time) time.Time {
	dayStart := startOfDayUTC(ts)
	weekday := int(dayStart.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return dayStart.AddDate(0, 0, -(weekday - 1))
}

func startOfDayUTC(ts time.Time) time.Time {
	return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
}

func durationDays(start time.Time, end time.Time) float64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Hours() / 24
}
