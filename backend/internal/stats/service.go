package stats

import (
	"sync"
	"time"

	"trace2offer/backend/internal/lead"
)

// Service provides statistics calculation with caching.
type Service struct {
	repo    lead.Repository
	cache   *statsCache
	cacheMu sync.RWMutex
	ttl     time.Duration
}

// statsCache holds cached statistics with expiration.
type statsCache struct {
	summary   SummaryStats
	dashboard DashboardStats
	generated time.Time
}

// NewService creates a new statistics service.
func NewService(repo lead.Repository) *Service {
	return &Service{
		repo: repo,
		ttl:  5 * time.Minute, // Cache for 5 minutes
	}
}

// SetCacheTTL allows customizing the cache TTL.
func (s *Service) SetCacheTTL(ttl time.Duration) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.ttl = ttl
	s.cache = nil // Invalidate current cache
}

// invalidateCache clears the cache (call when leads change).
func (s *Service) invalidateCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = nil
}

// getOrCompute returns cached stats or computes new ones.
func (s *Service) getOrCompute() statsCache {
	s.cacheMu.RLock()
	if s.cache != nil && time.Since(s.cache.generated) < s.ttl {
		defer s.cacheMu.RUnlock()
		return *s.cache
	}
	s.cacheMu.RUnlock()

	// Need to compute
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if s.cache != nil && time.Since(s.cache.generated) < s.ttl {
		return *s.cache
	}

	// Compute fresh stats
	leads := s.repo.List()
	calculator := NewCalculator(leads)
	summary := calculator.CalculateSummary()
	dashboard := calculator.CalculateDashboard()

	s.cache = &statsCache{
		summary:   summary,
		dashboard: dashboard,
		generated: time.Now(),
	}

	return *s.cache
}

// GetOverview returns high-level overview statistics.
func (s *Service) GetOverview() OverviewStats {
	return s.getOrCompute().summary.Overview
}

// GetFunnel returns conversion funnel statistics.
func (s *Service) GetFunnel() FunnelStats {
	return s.getOrCompute().summary.Funnel
}

// GetSources returns source/channel analysis.
func (s *Service) GetSources() SourceAnalysis {
	return s.getOrCompute().summary.Sources
}

// GetTrends returns time-series trends.
func (s *Service) GetTrends(period string) TrendStats {
	leads := s.repo.List()
	calculator := NewCalculator(leads)
	return calculator.CalculateTrends(period)
}

// GetInsights returns AI-style generated insights.
func (s *Service) GetInsights() InsightStats {
	return s.getOrCompute().summary.Insights
}

// GetSummary returns all statistics combined.
func (s *Service) GetSummary() SummaryStats {
	return s.getOrCompute().summary
}

// GetDashboard returns the core dashboard payload.
func (s *Service) GetDashboard() DashboardStats {
	return s.getOrCompute().dashboard
}

// InvalidateCache manually invalidates the cache.
// Call this when leads are created, updated, or deleted.
func (s *Service) InvalidateCache() {
	s.invalidateCache()
}
