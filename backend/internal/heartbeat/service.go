package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"trace2offer/backend/internal/reminder"
	"trace2offer/backend/internal/stats"
)

type Config struct {
	DataDir          string
	Interval         time.Duration
	ReminderService  *reminder.Service
	StatsService     *stats.Service
	StatusFileName   string
	ReportFolderName string
}

type Status struct {
	IntervalMinutes int    `json:"interval_minutes"`
	LastRunAt       string `json:"last_run_at,omitempty"`
	NextRunAt       string `json:"next_run_at,omitempty"`
	LastDueCount    int    `json:"last_due_count"`
	LastError       string `json:"last_error,omitempty"`
}

type Report struct {
	Type        string `json:"type"`
	Key         string `json:"key"`
	Path        string `json:"path"`
	GeneratedAt string `json:"generated_at"`
	Content     string `json:"content"`
}

// Service executes periodic checks and generates report suggestions.
type Service struct {
	reminders *reminder.Service
	stats     *stats.Service
	interval  time.Duration
	status    Status

	statusPath string
	reportDir  string

	mu sync.RWMutex
}

func NewService(config Config) (*Service, error) {
	dataDir := strings.TrimSpace(config.DataDir)
	if dataDir == "" {
		return nil, fmt.Errorf("heartbeat data dir is required")
	}
	if config.ReminderService == nil {
		return nil, fmt.Errorf("heartbeat reminder service is required")
	}
	if config.StatsService == nil {
		return nil, fmt.Errorf("heartbeat stats service is required")
	}

	interval := config.Interval
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	statusFileName := strings.TrimSpace(config.StatusFileName)
	if statusFileName == "" {
		statusFileName = "heartbeat_status.json"
	}
	reportFolderName := strings.TrimSpace(config.ReportFolderName)
	if reportFolderName == "" {
		reportFolderName = "heartbeat_reports"
	}

	reportDir := filepath.Join(dataDir, reportFolderName)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return nil, fmt.Errorf("create heartbeat report dir: %w", err)
	}

	statusPath := filepath.Join(dataDir, statusFileName)
	service := &Service{
		reminders:  config.ReminderService,
		stats:      config.StatsService,
		interval:   interval,
		statusPath: statusPath,
		reportDir:  reportDir,
		status: Status{
			IntervalMinutes: int(interval / time.Minute),
		},
	}
	_ = service.loadStatusFromDisk()
	return service, nil
}

func (s *Service) Start(ctx context.Context) {
	if s == nil {
		return
	}

	now := time.Now().UTC()
	_ = s.RunOnce(now)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case tick := <-ticker.C:
			_ = s.RunOnce(tick.UTC())
		}
	}
}

func (s *Service) RunOnce(now time.Time) error {
	if s == nil {
		return fmt.Errorf("heartbeat service is nil")
	}

	now = now.UTC()
	dueItems := s.reminders.GetDueAt(now)
	summary := s.stats.GetSummary()

	dailyReport := s.buildDailyReport(now, summary, dueItems)
	weeklyReport := s.buildWeeklyReport(now, summary, dueItems)

	if err := s.writeReport(dailyReport); err != nil {
		s.updateStatus(now, len(dueItems), err)
		return err
	}
	if err := s.writeReport(weeklyReport); err != nil {
		s.updateStatus(now, len(dueItems), err)
		return err
	}

	s.updateStatus(now, len(dueItems), nil)
	return nil
}

func (s *Service) GetStatus() Status {
	if s == nil {
		return Status{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Service) GetLatestReports() ([]Report, error) {
	if s == nil {
		return nil, fmt.Errorf("heartbeat service is nil")
	}

	reportTypes := []string{"daily", "weekly"}
	reports := make([]Report, 0, len(reportTypes))
	for _, reportType := range reportTypes {
		reportPath, key, err := s.findLatestReport(reportType)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(reportPath)
		if err != nil {
			continue
		}
		info, err := os.Stat(reportPath)
		if err != nil {
			continue
		}
		reports = append(reports, Report{
			Type:        reportType,
			Key:         key,
			Path:        reportPath,
			GeneratedAt: info.ModTime().UTC().Format(time.RFC3339),
			Content:     string(content),
		})
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Type < reports[j].Type
	})
	return reports, nil
}

func (s *Service) buildDailyReport(now time.Time, summary stats.SummaryStats, due []reminder.Item) Report {
	key := now.Format("2006-01-02")
	path := filepath.Join(s.reportDir, fmt.Sprintf("daily_%s.md", key))
	content := strings.Builder{}
	content.WriteString("# Daily Job Search Suggestion\n\n")
	content.WriteString(fmt.Sprintf("- GeneratedAt: %s\n", now.Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("- DueReminders: %d\n", len(due)))
	content.WriteString(fmt.Sprintf("- ActiveLeads: %d\n", summary.Overview.Active))
	content.WriteString(fmt.Sprintf("- FunnelConversion: %.1f%%\n", summary.Funnel.Conversion))
	content.WriteString("\n## Action Suggestions\n")

	if len(due) > 0 {
		content.WriteString("- 今天优先处理到期提醒，按紧急程度依次推进。\n")
	}
	if summary.Insights.Urgent > 0 {
		content.WriteString(fmt.Sprintf("- 当前有 %d 条紧急洞察，建议先清理高风险线索。\n", summary.Insights.Urgent))
	}
	if summary.Overview.ThisWeekNew == 0 {
		content.WriteString("- 本周尚无新增线索，建议今天补充 2-3 条高质量机会。\n")
	}
	if summary.Funnel.Conversion < 15 && summary.Overview.Total >= 5 {
		content.WriteString("- 转化率偏低，今天至少复盘 1 次被拒或卡点案例。\n")
	}
	if summary.Sources.BestSource != "" {
		content.WriteString(fmt.Sprintf("- 优先从 %s 渠道继续拓展线索。\n", summary.Sources.BestSource))
	}

	return Report{
		Type:        "daily",
		Key:         key,
		Path:        path,
		GeneratedAt: now.Format(time.RFC3339),
		Content:     content.String(),
	}
}

func (s *Service) buildWeeklyReport(now time.Time, summary stats.SummaryStats, due []reminder.Item) Report {
	year, week := now.ISOWeek()
	key := fmt.Sprintf("%d-W%02d", year, week)
	path := filepath.Join(s.reportDir, fmt.Sprintf("weekly_%s.md", key))
	content := strings.Builder{}
	content.WriteString("# Weekly Job Search Suggestion\n\n")
	content.WriteString(fmt.Sprintf("- GeneratedAt: %s\n", now.Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("- WeekKey: %s\n", key))
	content.WriteString(fmt.Sprintf("- TotalLeads: %d\n", summary.Overview.Total))
	content.WriteString(fmt.Sprintf("- SuccessRate: %.1f%%\n", summary.Overview.SuccessRate))
	content.WriteString(fmt.Sprintf("- OpenReminders: %d\n", len(due)))
	content.WriteString("\n## Strategy Suggestions\n")
	content.WriteString("- 复盘本周每个阶段的停留时长，找出最慢环节并设目标缩短 20%。\n")
	content.WriteString("- 统计渠道反馈质量，下周将高回报渠道占比提升到 50% 以上。\n")
	content.WriteString("- 为重点岗位准备一版定制化简历模板，减少重复劳动。\n")
	if summary.Insights.Urgent > 0 {
		content.WriteString("- 下周一优先处理当前紧急提醒，避免机会过期。\n")
	}

	return Report{
		Type:        "weekly",
		Key:         key,
		Path:        path,
		GeneratedAt: now.Format(time.RFC3339),
		Content:     content.String(),
	}
}

func (s *Service) writeReport(report Report) error {
	if strings.TrimSpace(report.Path) == "" {
		return fmt.Errorf("report path is empty")
	}
	if err := os.WriteFile(report.Path, []byte(report.Content), 0o644); err != nil {
		return fmt.Errorf("write heartbeat report %s: %w", report.Path, err)
	}
	return nil
}

func (s *Service) updateStatus(now time.Time, dueCount int, runErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.IntervalMinutes = int(s.interval / time.Minute)
	s.status.LastRunAt = now.Format(time.RFC3339)
	s.status.NextRunAt = now.Add(s.interval).Format(time.RFC3339)
	s.status.LastDueCount = dueCount
	if runErr != nil {
		s.status.LastError = runErr.Error()
	} else {
		s.status.LastError = ""
	}
	_ = s.saveStatusToDisk()
}

func (s *Service) saveStatusToDisk() error {
	payload, err := json.MarshalIndent(s.status, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.statusPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.statusPath)
}

func (s *Service) loadStatusFromDisk() error {
	payload, err := os.ReadFile(s.statusPath)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}

	var parsed Status
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return err
	}
	if parsed.IntervalMinutes <= 0 {
		parsed.IntervalMinutes = int(s.interval / time.Minute)
	}

	s.mu.Lock()
	s.status = parsed
	s.mu.Unlock()
	return nil
}

func (s *Service) findLatestReport(reportType string) (string, string, error) {
	entries, err := os.ReadDir(s.reportDir)
	if err != nil {
		return "", "", err
	}
	prefix := reportType + "_"
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".md") {
			continue
		}
		paths = append(paths, name)
	}
	if len(paths) == 0 {
		return "", "", fmt.Errorf("report not found")
	}
	sort.Strings(paths)
	latest := paths[len(paths)-1]
	key := strings.TrimSuffix(strings.TrimPrefix(latest, prefix), ".md")
	return filepath.Join(s.reportDir, latest), key, nil
}
