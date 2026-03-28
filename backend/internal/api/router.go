package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	agentruntime "trace2offer/backend/agent"
	"trace2offer/backend/internal/calendar"
	"trace2offer/backend/internal/candidate"
	"trace2offer/backend/internal/discovery"
	"trace2offer/backend/internal/heartbeat"
	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
	"trace2offer/backend/internal/prep"
	"trace2offer/backend/internal/reminder"
	"trace2offer/backend/internal/stats"
	"trace2offer/backend/internal/storage"
	"trace2offer/backend/internal/timeline"
)

type AgentRuntime interface {
	Run(ctx context.Context, request agentruntime.Request) (agentruntime.Response, error)
	CreateSession(ctx context.Context) (agentruntime.SessionView, error)
	GetSession(ctx context.Context, sessionID string) (agentruntime.SessionView, error)
	ListSessions(ctx context.Context) ([]agentruntime.SessionSummaryView, error)
	GetSettings() agentruntime.RuntimeSettingsView
	UpdateSettings(ctx context.Context, patch agentruntime.RuntimeSettingsPatch) (agentruntime.RuntimeSettingsView, error)
	GetUserProfile() (agentruntime.UserProfile, error)
	UpdateUserProfile(ctx context.Context, profile agentruntime.UserProfile) (agentruntime.UserProfile, error)
	ImportUserProfileFromResume(ctx context.Context, sourceName string, contentType string, content []byte) (agentruntime.UserProfileImportResult, error)
}

type handler struct {
	leads        *lead.Service
	candidates   *candidate.Service
	discovery    *discovery.Service
	agentRuntime AgentRuntime
	stats        *stats.Service
	reminders    *reminder.Service
	heartbeat    *heartbeat.Service
	calendar     *calendar.Service
	timelines    *timeline.Service
	prep         *prep.Service
}

func NewRouter(leads storage.LeadStore, candidates storage.CandidateStore, leadTimelines storage.LeadTimelineStore, runtime AgentRuntime, statsService *stats.Service, reminderService *reminder.Service, heartbeatService *heartbeat.Service, calendarService *calendar.Service, discoveryService *discovery.Service, prepService *prep.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	timelineService := timeline.NewService(leadTimelines)
	leadService := lead.NewService(leads).WithStatusObserver(timelineService)
	h := &handler{
		leads:        leadService,
		candidates:   candidate.NewService(candidates, leadService),
		discovery:    discoveryService,
		agentRuntime: runtime,
		stats:        statsService,
		reminders:    reminderService,
		heartbeat:    heartbeatService,
		calendar:     calendarService,
		timelines:    timelineService,
		prep:         prepService,
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.GET("/leads", h.listLeads)
		api.POST("/leads", h.createLead)
		api.PUT("/leads/:id", h.updateLead)
		api.PATCH("/leads/:id", h.updateLead)
		api.DELETE("/leads/:id", h.deleteLead)

		api.GET("/candidates", h.listCandidates)
		api.POST("/candidates", h.createCandidate)
		api.PUT("/candidates/:id", h.updateCandidate)
		api.PATCH("/candidates/:id", h.updateCandidate)
		api.DELETE("/candidates/:id", h.deleteCandidate)
		api.POST("/candidates/:id/promote", h.promoteCandidate)

		api.GET("/lead-timelines", h.listLeadTimelines)

		discoveryGroup := api.Group("/discovery")
		discoveryGroup.GET("/rules", h.listDiscoveryRules)
		discoveryGroup.POST("/rules", h.createDiscoveryRule)
		discoveryGroup.PATCH("/rules/:id", h.updateDiscoveryRule)
		discoveryGroup.DELETE("/rules/:id", h.deleteDiscoveryRule)
		discoveryGroup.POST("/run", h.runDiscoveryNow)

		agent := api.Group("/agent")
		agent.POST("/chat", h.chatWithAgent)
		agent.GET("/sessions", h.listAgentSessions)
		agent.POST("/sessions", h.createAgentSession)
		agent.GET("/sessions/:id", h.getAgentSession)
		agent.GET("/settings", h.getAgentSettings)
		agent.PATCH("/settings", h.updateAgentSettings)

		user := api.Group("/user")
		user.GET("/profile", h.getUserProfile)
		user.PUT("/profile", h.updateUserProfile)
		user.POST("/profile/import", h.importUserProfile)

		stats := api.Group("/stats")
		stats.GET("/overview", h.getStatsOverview)
		stats.GET("/funnel", h.getStatsFunnel)
		stats.GET("/sources", h.getStatsSources)
		stats.GET("/trends", h.getStatsTrends)
		stats.GET("/insights", h.getStatsInsights)
		stats.GET("/summary", h.getStatsSummary)
		stats.GET("", h.getStatsDashboard)
		stats.GET("/", h.getStatsDashboard)

		reminders := api.Group("/reminders")
		reminders.GET("/due", h.getDueReminders)

		heartbeat := api.Group("/heartbeat")
		heartbeat.GET("/status", h.getHeartbeatStatus)
		heartbeat.GET("/reports/latest", h.getHeartbeatReportsLatest)

		prep := api.Group("/prep")
		prep.GET("/meta", h.getPrepMeta)
		prep.GET("/index/status", h.getPrepIndexStatus)
		prep.GET("/index/documents", h.listPrepIndexDocuments)
		prep.GET("/index/chunks", h.listPrepIndexChunks)
		prep.POST("/index/rebuild", h.rebuildPrepIndex)
		prep.GET("/leads/:lead_id/context-preview", h.getPrepLeadContextPreview)
		prep.GET("/topics", h.listPrepTopics)
		prep.POST("/topics", h.createPrepTopic)
		prep.PATCH("/topics/:key", h.updatePrepTopic)
		prep.DELETE("/topics/:key", h.deletePrepTopic)
		prep.GET("/knowledge/:scope/:scope_id/documents", h.listPrepKnowledgeDocuments)
		prep.POST("/knowledge/:scope/:scope_id/documents", h.createPrepKnowledgeDocument)
		prep.PUT("/knowledge/:scope/:scope_id/documents/:filename", h.updatePrepKnowledgeDocument)
		prep.DELETE("/knowledge/:scope/:scope_id/documents/:filename", h.deletePrepKnowledgeDocument)
		prep.POST("/retrieval/preview", h.searchPrepRetrieval)
		prep.POST("/sessions", h.createPrepSession)
		prep.POST("/sessions/stream", h.streamPrepSession)
		prep.GET("/sessions/:session_id", h.getPrepSession)
		prep.PUT("/sessions/:session_id/draft-answers", h.updatePrepDraftAnswers)

		api.GET("/calendar/interviews.ics", h.exportInterviewICS)
		api.GET("/caldav/trace2offer", h.exportInterviewICS)
		api.Handle("OPTIONS", "/caldav", h.handleCalDAVOptions)
		api.Handle("OPTIONS", "/caldav/trace2offer", h.handleCalDAVOptions)
		api.Handle("PROPFIND", "/caldav", h.handleCalDAVPropfind)
		api.Handle("PROPFIND", "/caldav/trace2offer", h.handleCalDAVPropfind)
		api.Handle("REPORT", "/caldav/trace2offer", h.handleCalDAVReport)
	}

	return r
}

func (h *handler) listLeads(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.leads.List()})
}

func (h *handler) createLead(c *gin.Context) {
	input, ok := bindLeadInput(c)
	if !ok {
		return
	}

	created, err := h.leads.Create(input)
	if err != nil {
		respondLeadError(c, "create lead failed", err)
		return
	}

	// Invalidate stats cache when leads change
	if h.stats != nil {
		h.stats.InvalidateCache()
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *handler) updateLead(c *gin.Context) {
	input, ok := bindLeadInput(c)
	if !ok {
		return
	}

	updated, found, err := h.leads.Update(c.Param("id"), input)
	if err != nil {
		respondLeadError(c, "update lead failed", err)
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "lead not found"})
		return
	}

	// Invalidate stats cache when leads change
	if h.stats != nil {
		h.stats.InvalidateCache()
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *handler) deleteLead(c *gin.Context) {
	deleted, err := h.leads.Delete(c.Param("id"))
	if err != nil {
		respondLeadError(c, "delete lead failed", err)
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"message": "lead not found"})
		return
	}

	// Invalidate stats cache when leads change
	if h.stats != nil {
		h.stats.InvalidateCache()
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) listLeadTimelines(c *gin.Context) {
	if h.timelines == nil {
		c.JSON(http.StatusOK, gin.H{"data": []model.LeadTimeline{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.timelines.List()})
}

func (h *handler) listCandidates(c *gin.Context) {
	if h.candidates == nil {
		c.JSON(http.StatusOK, gin.H{"data": []model.Candidate{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.candidates.List()})
}

func (h *handler) createCandidate(c *gin.Context) {
	input, ok := bindCandidateInput(c)
	if !ok {
		return
	}
	if h.candidates == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "candidate service is not configured"})
		return
	}

	created, err := h.candidates.Create(input)
	if err != nil {
		respondCandidateError(c, "create candidate failed", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *handler) updateCandidate(c *gin.Context) {
	input, ok := bindCandidateInput(c)
	if !ok {
		return
	}
	if h.candidates == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "candidate service is not configured"})
		return
	}

	updated, found, err := h.candidates.Update(c.Param("id"), input)
	if err != nil {
		respondCandidateError(c, "update candidate failed", err)
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "candidate not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *handler) deleteCandidate(c *gin.Context) {
	if h.candidates == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "candidate service is not configured"})
		return
	}

	deleted, err := h.candidates.Delete(c.Param("id"))
	if err != nil {
		respondCandidateError(c, "delete candidate failed", err)
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"message": "candidate not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) promoteCandidate(c *gin.Context) {
	if h.candidates == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "candidate service is not configured"})
		return
	}

	var input model.CandidatePromoteInput
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid candidate promote payload", "error": err.Error()})
		return
	}

	updatedCandidate, createdLead, err := h.candidates.Promote(c.Param("id"), input)
	if err != nil {
		if errors.Is(err, candidate.ErrCandidateNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "candidate not found"})
			return
		}
		respondCandidateError(c, "promote candidate failed", err)
		return
	}

	if h.stats != nil {
		h.stats.InvalidateCache()
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"candidate": updatedCandidate,
			"lead":      createdLead,
		},
	})
}

func (h *handler) listDiscoveryRules(c *gin.Context) {
	if h.discovery == nil {
		c.JSON(http.StatusOK, gin.H{"data": []model.DiscoveryRule{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.discovery.ListRules()})
}

func (h *handler) createDiscoveryRule(c *gin.Context) {
	if h.discovery == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "discovery service is not configured"})
		return
	}

	input, ok := bindDiscoveryRuleInput(c)
	if !ok {
		return
	}
	created, err := h.discovery.CreateRule(input)
	if err != nil {
		respondDiscoveryError(c, "create discovery rule failed", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *handler) updateDiscoveryRule(c *gin.Context) {
	if h.discovery == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "discovery service is not configured"})
		return
	}

	input, ok := bindDiscoveryRuleInput(c)
	if !ok {
		return
	}
	updated, found, err := h.discovery.UpdateRule(c.Param("id"), input)
	if err != nil {
		respondDiscoveryError(c, "update discovery rule failed", err)
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "discovery rule not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *handler) deleteDiscoveryRule(c *gin.Context) {
	if h.discovery == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "discovery service is not configured"})
		return
	}

	deleted, err := h.discovery.DeleteRule(c.Param("id"))
	if err != nil {
		respondDiscoveryError(c, "delete discovery rule failed", err)
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"message": "discovery rule not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handler) runDiscoveryNow(c *gin.Context) {
	if h.discovery == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "discovery service is not configured"})
		return
	}

	result, err := h.discovery.RunOnce(c.Request.Context(), time.Now().UTC())
	if err != nil {
		respondDiscoveryError(c, "run discovery failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *handler) chatWithAgent(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
		History   []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"history"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid chat request", "error": err.Error()})
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "message is required"})
		return
	}

	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	response, err := h.agentRuntime.Run(c.Request.Context(), agentruntime.Request{
		SessionID: strings.TrimSpace(req.SessionID),
		Message:   message,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent run failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"session_id": response.SessionID,
			"reply":      response.Reply,
		},
	})
}

func (h *handler) listAgentSessions(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	sessions, err := h.agentRuntime.ListSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "list sessions failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

func (h *handler) createAgentSession(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	session, err := h.agentRuntime.CreateSession(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "create session failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": session})
}

func (h *handler) getAgentSession(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "session id is required"})
		return
	}

	session, err := h.agentRuntime.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, agentruntime.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "get session failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (h *handler) getAgentSettings(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": h.agentRuntime.GetSettings(),
	})
}

func (h *handler) updateAgentSettings(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	var patch agentruntime.RuntimeSettingsPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid settings payload", "error": err.Error()})
		return
	}

	settings, err := h.agentRuntime.UpdateSettings(c.Request.Context(), patch)
	if err != nil {
		if agentruntime.IsSettingsValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		if errors.Is(err, agentruntime.ErrRuntimeUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "update agent settings failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": settings})
}

func (h *handler) getUserProfile(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	profile, err := h.agentRuntime.GetUserProfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "get user profile failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *handler) updateUserProfile(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	var input agentruntime.UserProfile
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid user profile payload", "error": err.Error()})
		return
	}

	profile, err := h.agentRuntime.UpdateUserProfile(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "update user profile failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *handler) getStatsDashboard(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetDashboard()})
}

func (h *handler) getStatsOverview(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetOverview()})
}

func (h *handler) getStatsFunnel(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetFunnel()})
}

func (h *handler) getStatsSources(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetSources()})
}

func (h *handler) getStatsTrends(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	period := c.DefaultQuery("period", "month")
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetTrends(period)})
}

func (h *handler) getStatsInsights(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetInsights()})
}

func (h *handler) getStatsSummary(c *gin.Context) {
	if h.stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "stats service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.stats.GetSummary()})
}

func (h *handler) getDueReminders(c *gin.Context) {
	if h.reminders == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "reminder service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.reminders.GetDue()})
}

func (h *handler) getHeartbeatStatus(c *gin.Context) {
	if h.heartbeat == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "heartbeat service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.heartbeat.GetStatus()})
}

func (h *handler) getHeartbeatReportsLatest(c *gin.Context) {
	if h.heartbeat == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "heartbeat service is not configured"})
		return
	}

	reports, err := h.heartbeat.GetLatestReports()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load heartbeat reports failed", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reports})
}

func (h *handler) getPrepMeta(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": h.prep.GetMeta()})
}

func (h *handler) getPrepIndexStatus(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	status, err := h.prep.GetIndexStatus()
	if err != nil {
		respondPrepError(c, "get prep index status failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": status})
}

func (h *handler) listPrepIndexDocuments(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	documents, err := h.prep.ListIndexDocuments()
	if err != nil {
		respondPrepError(c, "list prep index documents failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": documents})
}

func (h *handler) listPrepIndexChunks(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	documentID := strings.TrimSpace(c.Query("document_id"))
	limitRaw := strings.TrimSpace(c.Query("limit"))
	limit := 200
	if limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "limit must be an integer"})
			return
		}
		if parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "limit must be greater than 0"})
			return
		}
		limit = parsed
	}

	chunks, err := h.prep.ListIndexChunks(documentID, limit)
	if err != nil {
		respondPrepError(c, "list prep index chunks failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": chunks})
}

func (h *handler) rebuildPrepIndex(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepIndexRebuildInput(c)
	if !ok {
		return
	}

	summary, err := h.prep.RebuildIndexWithMode(input.Scope, input.ScopeID, input.Mode)
	if err != nil {
		respondPrepError(c, "rebuild prep index failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary})
}

func (h *handler) getPrepLeadContextPreview(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}
	if h.leads == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "lead service is not configured"})
		return
	}

	leadID := strings.TrimSpace(c.Param("lead_id"))
	if leadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "lead_id is required"})
		return
	}

	var target model.Lead
	found := false
	for _, item := range h.leads.List() {
		if item.ID == leadID {
			target = item
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "lead not found"})
		return
	}

	preview, err := h.prep.GetLeadContextPreview(target)
	if err != nil {
		respondPrepError(c, "get prep context preview failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": preview})
}

func (h *handler) listPrepTopics(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	topics, err := h.prep.ListTopics()
	if err != nil {
		respondPrepError(c, "list prep topics failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": topics})
}

func (h *handler) createPrepTopic(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepTopicCreateInput(c)
	if !ok {
		return
	}

	created, err := h.prep.CreateTopic(input)
	if err != nil {
		respondPrepError(c, "create prep topic failed", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *handler) updatePrepTopic(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepTopicPatchInput(c)
	if !ok {
		return
	}

	updated, found, err := h.prep.UpdateTopic(c.Param("key"), input)
	if err != nil {
		respondPrepError(c, "update prep topic failed", err)
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "prep topic not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *handler) deletePrepTopic(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	deleted, err := h.prep.DeleteTopic(c.Param("key"))
	if err != nil {
		respondPrepError(c, "delete prep topic failed", err)
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"message": "prep topic not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handler) listPrepKnowledgeDocuments(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	documents, err := h.prep.ListKnowledgeDocuments(c.Param("scope"), c.Param("scope_id"))
	if err != nil {
		respondPrepError(c, "list prep knowledge documents failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": documents})
}

func (h *handler) createPrepKnowledgeDocument(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepKnowledgeCreateInput(c)
	if !ok {
		return
	}

	created, err := h.prep.CreateKnowledgeDocument(c.Param("scope"), c.Param("scope_id"), input)
	if err != nil {
		respondPrepError(c, "create prep knowledge document failed", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *handler) updatePrepKnowledgeDocument(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepKnowledgeUpdateInput(c)
	if !ok {
		return
	}

	updated, found, err := h.prep.UpdateKnowledgeDocument(c.Param("scope"), c.Param("scope_id"), c.Param("filename"), input)
	if err != nil {
		respondPrepError(c, "update prep knowledge document failed", err)
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "prep knowledge document not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *handler) deletePrepKnowledgeDocument(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	deleted, err := h.prep.DeleteKnowledgeDocument(c.Param("scope"), c.Param("scope_id"), c.Param("filename"))
	if err != nil {
		respondPrepError(c, "delete prep knowledge document failed", err)
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"message": "prep knowledge document not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *handler) searchPrepRetrieval(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepSearchInput(c)
	if !ok {
		return
	}

	if strings.TrimSpace(input.LeadID) != "" {
		if h.leads == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"message": "lead service is not configured"})
			return
		}
		lead, found := findLeadByID(h.leads.List(), input.LeadID)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"message": "lead not found"})
			return
		}
		input.CompanySlug = strings.TrimSpace(prep.NormalizeCompanySlug(lead.Company))
	}

	result, err := h.prep.Search(input)
	if err != nil {
		respondPrepError(c, "search prep retrieval failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *handler) updatePrepDraftAnswers(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	input, ok := bindPrepDraftAnswersInput(c)
	if !ok {
		return
	}

	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "session_id is required"})
		return
	}

	err := h.prep.SaveDraftAnswers(sessionID, input.Answers)
	if err != nil {
		if errors.Is(err, prep.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "prep session not found"})
			return
		}
		respondPrepError(c, "save prep draft answers failed", err)
		return
	}

	session, err := h.prep.GetSession(sessionID)
	if err != nil {
		if errors.Is(err, prep.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "prep session not found"})
			return
		}
		respondPrepError(c, "load prep session after saving draft answers failed", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": prep.SaveDraftAnswersResult{
		SessionID:    strings.TrimSpace(session.ID),
		SavedAt:      session.UpdatedAt,
		AnswersCount: len(session.Answers),
	}})
}

func (h *handler) createPrepSession(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}
	if h.leads == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "lead service is not configured"})
		return
	}

	input, ok := bindPrepCreateSessionInput(c)
	if !ok {
		return
	}

	lead, found := findLeadByID(h.leads.List(), input.LeadID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "lead not found"})
		return
	}

	created, err := h.prep.CreateSession(c.Request.Context(), lead, input)
	if err != nil {
		respondPrepError(c, "create prep session failed", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *handler) streamPrepSession(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}
	if h.leads == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "lead service is not configured"})
		return
	}

	input, ok := bindPrepCreateSessionInput(c)
	if !ok {
		return
	}

	lead, found := findLeadByID(h.leads.List(), input.LeadID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"message": "lead not found"})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "streaming is not supported"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	sendEvent := func(name string, payload any) {
		if c.Request.Context().Err() != nil {
			return
		}
		c.SSEvent(name, payload)
		flusher.Flush()
	}

	sendEvent("started", gin.H{
		"lead_id": input.LeadID,
	})

	created, err := h.prep.CreateSessionWithProgress(c.Request.Context(), lead, input, func(event prep.GenerationProgressEvent) {
		sendEvent("stage", event)
	})
	if err != nil {
		sendEvent("error", gin.H{
			"message": err.Error(),
		})
		return
	}

	sendEvent("completed", gin.H{
		"session": created,
	})
}

func (h *handler) getPrepSession(c *gin.Context) {
	if h.prep == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "prep service is not configured"})
		return
	}

	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "session_id is required"})
		return
	}

	session, err := h.prep.GetSession(sessionID)
	if err != nil {
		if errors.Is(err, prep.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "prep session not found"})
			return
		}
		respondPrepError(c, "get prep session failed", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": session})
}

func (h *handler) exportInterviewICS(c *gin.Context) {
	if h.calendar == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "calendar service is not configured"})
		return
	}
	ics, err := h.calendar.BuildInterviewICS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "generate calendar export failed", "error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="trace2offer-interviews.ics"`)
	c.String(http.StatusOK, ics)
}

func (h *handler) handleCalDAVOptions(c *gin.Context) {
	c.Header("DAV", "1, 2, calendar-access")
	c.Header("Allow", "OPTIONS, PROPFIND, REPORT, GET")
	c.Header("Content-Length", "0")
	c.Status(http.StatusNoContent)
}

func (h *handler) handleCalDAVPropfind(c *gin.Context) {
	requestPath := strings.TrimSpace(c.Request.URL.Path)
	rootPath := "/api/caldav"
	calendarPath := "/api/caldav/trace2offer"

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(http.StatusMultiStatus)
	if requestPath == rootPath {
		c.String(http.StatusMultiStatus, buildCalDAVRootMultiStatus(rootPath, calendarPath))
		return
	}
	c.String(http.StatusMultiStatus, buildCalDAVCalendarMultiStatus(calendarPath))
}

func (h *handler) handleCalDAVReport(c *gin.Context) {
	if h.calendar == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "calendar service is not configured"})
		return
	}

	ics, err := h.calendar.BuildInterviewICS()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "generate caldav report failed", "error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusMultiStatus, buildCalDAVReportResponse("/api/caldav/trace2offer", ics))
}

func (h *handler) importUserProfile(c *gin.Context) {
	if h.agentRuntime == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "agent runtime is not configured"})
		return
	}

	file, err := c.FormFile("resume")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "resume file is required", "error": err.Error()})
		return
	}

	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "open resume file failed", "error": err.Error()})
		return
	}
	defer opened.Close()

	content, err := io.ReadAll(opened)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "read resume file failed", "error": err.Error()})
		return
	}

	imported, err := h.agentRuntime.ImportUserProfileFromResume(
		c.Request.Context(),
		strings.TrimSpace(file.Filename),
		strings.TrimSpace(file.Header.Get("Content-Type")),
		content,
	)
	if err != nil {
		if agentruntime.IsResumeImportError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "import user profile failed", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": imported})
}

func bindLeadInput(c *gin.Context) (model.LeadMutationInput, bool) {
	var input model.LeadMutationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid lead payload", "error": err.Error()})
		return model.LeadMutationInput{}, false
	}

	return input, true
}

func bindCandidateInput(c *gin.Context) (model.CandidateMutationInput, bool) {
	var input model.CandidateMutationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid candidate payload", "error": err.Error()})
		return model.CandidateMutationInput{}, false
	}
	return input, true
}

func bindDiscoveryRuleInput(c *gin.Context) (model.DiscoveryRuleMutationInput, bool) {
	var input model.DiscoveryRuleMutationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid discovery rule payload", "error": err.Error()})
		return model.DiscoveryRuleMutationInput{}, false
	}
	return input, true
}

func bindPrepTopicCreateInput(c *gin.Context) (prep.TopicCreateInput, bool) {
	var input prep.TopicCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep topic payload", "error": err.Error()})
		return prep.TopicCreateInput{}, false
	}
	return input, true
}

func bindPrepTopicPatchInput(c *gin.Context) (prep.TopicPatchInput, bool) {
	var input prep.TopicPatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep topic patch payload", "error": err.Error()})
		return prep.TopicPatchInput{}, false
	}
	return input, true
}

func bindPrepKnowledgeCreateInput(c *gin.Context) (prep.KnowledgeDocumentCreateInput, bool) {
	var input prep.KnowledgeDocumentCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep knowledge payload", "error": err.Error()})
		return prep.KnowledgeDocumentCreateInput{}, false
	}
	return input, true
}

func bindPrepKnowledgeUpdateInput(c *gin.Context) (prep.KnowledgeDocumentUpdateInput, bool) {
	var input prep.KnowledgeDocumentUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep knowledge patch payload", "error": err.Error()})
		return prep.KnowledgeDocumentUpdateInput{}, false
	}
	return input, true
}

func bindPrepIndexRebuildInput(c *gin.Context) (prep.RebuildIndexInput, bool) {
	var input prep.RebuildIndexInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep index rebuild payload", "error": err.Error()})
		return prep.RebuildIndexInput{}, false
	}
	return input, true
}

func bindPrepSearchInput(c *gin.Context) (prep.SearchConfig, bool) {
	var input prep.SearchConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep retrieval payload", "error": err.Error()})
		return prep.SearchConfig{}, false
	}
	return input, true
}

func bindPrepCreateSessionInput(c *gin.Context) (prep.CreateSessionInput, bool) {
	var input prep.CreateSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep session payload", "error": err.Error()})
		return prep.CreateSessionInput{}, false
	}
	if strings.TrimSpace(input.LeadID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "lead_id is required"})
		return prep.CreateSessionInput{}, false
	}
	return input, true
}

func findLeadByID(leads []model.Lead, leadID string) (model.Lead, bool) {
	normalizedLeadID := strings.TrimSpace(leadID)
	for _, item := range leads {
		if item.ID == normalizedLeadID {
			return item, true
		}
	}
	return model.Lead{}, false
}

func bindPrepDraftAnswersInput(c *gin.Context) (struct {
	Answers []prep.Answer `json:"answers"`
}, bool) {
	var input struct {
		Answers []prep.Answer `json:"answers"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid prep draft answers payload", "error": err.Error()})
		return struct {
			Answers []prep.Answer `json:"answers"`
		}{}, false
	}
	return input, true
}

func respondLeadError(c *gin.Context, message string, err error) {
	if lead.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"message": message, "error": err.Error()})
}

func respondCandidateError(c *gin.Context, message string, err error) {
	if candidate.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if errors.Is(err, candidate.ErrRepositoryUnavailable) || errors.Is(err, candidate.ErrLeadManagerUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	if lead.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"message": message, "error": err.Error()})
}

func respondDiscoveryError(c *gin.Context, message string, err error) {
	if discovery.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if errors.Is(err, discovery.ErrRuleStoreUnavailable) || errors.Is(err, discovery.ErrCandidateManagerUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": message, "error": err.Error()})
}

func respondPrepError(c *gin.Context, message string, err error) {
	if prep.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if errors.Is(err, prep.ErrPrepDisabled) || errors.Is(err, prep.ErrTopicStoreUnavailable) || errors.Is(err, prep.ErrKnowledgeStoreUnavailable) || errors.Is(err, prep.ErrIndexStoreUnavailable) || errors.Is(err, prep.ErrSessionStoreUnavailable) || errors.Is(err, prep.ErrIngestionUnavailable) || errors.Is(err, prep.ErrQuestionGeneratorUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	if errors.Is(err, prep.ErrContextResolverUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
		return
	}
	if errors.Is(err, prep.ErrTopicAlreadyExists) || errors.Is(err, prep.ErrDocumentAlreadyExists) {
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"message": message, "error": err.Error()})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, PROPFIND, REPORT")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func buildCalDAVRootMultiStatus(rootPath string, calendarPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>%s</d:href>
    <d:propstat>
      <d:prop>
        <d:displayname>Trace2Offer CalDAV</d:displayname>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>%s</d:href>
    <d:propstat>
      <d:prop>
        <d:displayname>Trace2Offer Interviews</d:displayname>
        <d:resourcetype>
          <d:collection/>
          <cal:calendar/>
        </d:resourcetype>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`, rootPath, calendarPath)
}

func buildCalDAVCalendarMultiStatus(calendarPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>%s</d:href>
    <d:propstat>
      <d:prop>
        <d:displayname>Trace2Offer Interviews</d:displayname>
        <d:resourcetype>
          <d:collection/>
          <cal:calendar/>
        </d:resourcetype>
        <cal:supported-calendar-component-set>
          <cal:comp name="VEVENT"/>
        </cal:supported-calendar-component-set>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`, calendarPath)
}

func buildCalDAVReportResponse(calendarPath string, ics string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>%s</d:href>
    <d:propstat>
      <d:prop>
        <cal:calendar-data>%s</cal:calendar-data>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`, calendarPath, xmlEscape(ics))
}

func xmlEscape(raw string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(raw))
	return buffer.String()
}
