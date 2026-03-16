package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	agentruntime "trace2offer/backend/agent"
	"trace2offer/backend/internal/calendar"
	"trace2offer/backend/internal/heartbeat"
	"trace2offer/backend/internal/lead"
	"trace2offer/backend/internal/model"
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
	agentRuntime AgentRuntime
	stats        *stats.Service
	reminders    *reminder.Service
	heartbeat    *heartbeat.Service
	calendar     *calendar.Service
	timelines    *timeline.Service
}

func NewRouter(leads storage.LeadStore, leadTimelines storage.LeadTimelineStore, runtime AgentRuntime, statsService *stats.Service, reminderService *reminder.Service, heartbeatService *heartbeat.Service, calendarService *calendar.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware())

	timelineService := timeline.NewService(leadTimelines)
	h := &handler{
		leads:        lead.NewService(leads).WithStatusObserver(timelineService),
		agentRuntime: runtime,
		stats:        statsService,
		reminders:    reminderService,
		heartbeat:    heartbeatService,
		calendar:     calendarService,
		timelines:    timelineService,
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
		api.GET("/lead-timelines", h.listLeadTimelines)

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

func respondLeadError(c *gin.Context, message string, err error) {
	if lead.IsValidationError(err) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
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
