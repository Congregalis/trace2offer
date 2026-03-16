package stats

import (
	"time"
)

// OverviewStats represents high-level statistics about all leads.
type OverviewStats struct {
	Total        int           `json:"total"`
	Active       int           `json:"active"`
	Offered      int           `json:"offered"`
	SuccessRate  float64       `json:"success_rate"`
	ThisWeekNew  int           `json:"this_week_new"`
	StatusCounts []StatusCount `json:"status_counts"`
	LastUpdated  string        `json:"last_updated"`
}

// StatusCount represents lead count in a specific status.
type StatusCount struct {
	Status     string  `json:"status"`
	Label      string  `json:"label"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// FunnelStage represents a single stage in the conversion funnel.
type FunnelStage struct {
	Status             string  `json:"status"`
	Label              string  `json:"label"`
	Count              int     `json:"count"`
	CumulativeCount    int     `json:"cumulative_count"`
	Percentage         float64 `json:"percentage"`
	ConversionFromPrev float64 `json:"conversion_from_prev"`
	AvgDays            float64 `json:"avg_days"`
}

// FunnelStats represents the conversion funnel from new to offered.
type FunnelStats struct {
	Stages     []FunnelStage `json:"stages"`
	Conversion float64       `json:"conversion"`
	TotalTime  float64       `json:"total_time_avg"`
}

// SourceStats represents statistics for a single source/channel.
type SourceStats struct {
	Source      string  `json:"source"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	Applied     int     `json:"applied"`
	Interviewed int     `json:"interviewed"`
	Offered     int     `json:"offered"`
	SuccessRate float64 `json:"success_rate"`
}

// SourceAnalysis represents the complete source/channel analysis.
type SourceAnalysis struct {
	Sources    []SourceStats `json:"sources"`
	TopSource  string        `json:"top_source"`
	BestSource string        `json:"best_source"`
}

// TimePoint represents a single data point in a time series.
type TimePoint struct {
	Date  string `json:"date"`
	Label string `json:"label"`
	New   int    `json:"new"`
	Moved int    `json:"moved"`
	Total int    `json:"total"`
}

// TrendStats represents time-series trends over a period.
type TrendStats struct {
	Period    string      `json:"period"`
	Points    []TimePoint `json:"points"`
	Growth    float64     `json:"growth_rate"`
	IsGrowing bool        `json:"is_growing"`
}

// DurationStatus represents average dwell time at a status.
type DurationStatus struct {
	Status  string  `json:"status"`
	Label   string  `json:"label"`
	Count   int     `json:"count"`
	AvgDays float64 `json:"avg_days"`
}

// DurationStats represents average dwell-time analysis.
type DurationStats struct {
	AverageCycleDays  float64          `json:"average_cycle_days"`
	AverageActiveDays float64          `json:"average_active_days"`
	SlowestStatus     string           `json:"slowest_status"`
	SlowestLabel      string           `json:"slowest_label"`
	ByStatus          []DurationStatus `json:"by_status"`
}

// InsightItem represents a single insight or suggestion.
type InsightItem struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	Action      string `json:"action"`
	RelatedLead string `json:"related_lead,omitempty"`
}

// InsightStats represents AI-generated insights.
type InsightStats struct {
	Total     int           `json:"total"`
	Urgent    int           `json:"urgent"`
	Tips      int           `json:"tips"`
	Insights  []InsightItem `json:"insights"`
	Generated string        `json:"generated_at"`
}

// SummaryStats aggregates all statistics for the dashboard.
type SummaryStats struct {
	Overview  OverviewStats  `json:"overview"`
	Funnel    FunnelStats    `json:"funnel"`
	Sources   SourceAnalysis `json:"sources"`
	Trends    TrendStats     `json:"trends"`
	Duration  DurationStats  `json:"duration"`
	Insights  InsightStats   `json:"insights"`
	Generated time.Time      `json:"generated_at"`
}

// DashboardStats is the complete payload for the core stats dashboard.
type DashboardStats struct {
	Overview     OverviewStats `json:"overview"`
	Funnel       FunnelStats   `json:"funnel"`
	WeeklyTrend  TrendStats    `json:"weekly_trend"`
	MonthlyTrend TrendStats    `json:"monthly_trend"`
	Duration     DurationStats `json:"duration"`
	Generated    time.Time     `json:"generated_at"`
}
