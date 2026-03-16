package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentruntime "trace2offer/backend/agent"
	"trace2offer/backend/internal/calendar"
	"trace2offer/backend/internal/heartbeat"
	"trace2offer/backend/internal/model"
	"trace2offer/backend/internal/reminder"
	"trace2offer/backend/internal/stats"
	"trace2offer/backend/internal/storage"
)

func TestLeadAndChatAPI(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	leadStore, err := storage.NewFileLeadStore(filepath.Join(tmpDir, "leads.json"))
	if err != nil {
		t.Fatalf("init lead store: %v", err)
	}
	leadTimelineStore, err := storage.NewFileLeadTimelineStore(filepath.Join(tmpDir, "lead_timelines.json"))
	if err != nil {
		t.Fatalf("init lead timeline store: %v", err)
	}
	statsService := stats.NewService(leadStore)
	reminderService := reminder.NewService(leadStore)
	heartbeatService, err := heartbeat.NewService(heartbeat.Config{
		DataDir:         tmpDir,
		ReminderService: reminderService,
		StatsService:    statsService,
	})
	if err != nil {
		t.Fatalf("init heartbeat service: %v", err)
	}
	_ = heartbeatService.RunOnce(time.Now().UTC())

	router := NewRouter(leadStore, leadTimelineStore, &stubAgentRuntime{}, statsService, reminderService, heartbeatService, calendar.NewService(leadStore))

	resp := doJSONRequest(t, router, http.MethodGet, "/api/leads", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/leads status=%d body=%s", resp.Code, resp.Body.String())
	}

	var listPayload struct {
		Data []model.Lead `json:"data"`
	}
	decodeJSONBody(t, resp, &listPayload)
	if len(listPayload.Data) < 3 {
		t.Fatalf("expected seed leads >= 3, got %d", len(listPayload.Data))
	}

	createInput := model.LeadMutationInput{
		Company:           "OpenAI",
		Position:          "Go Backend Engineer",
		Source:            "Direct Apply",
		Status:            "new",
		Priority:          5,
		NextAction:        "send application",
		Notes:             "test lead",
		CompanyWebsiteURL: "https://openai.com",
		JDURL:             "https://openai.com/careers/backend",
		Location:          "San Francisco, CA",
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/leads", createInput)
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/leads status=%d body=%s", resp.Code, resp.Body.String())
	}

	var createPayload struct {
		Data model.Lead `json:"data"`
	}
	decodeJSONBody(t, resp, &createPayload)
	if createPayload.Data.ID == "" {
		t.Fatal("created lead ID should not be empty")
	}
	if createPayload.Data.CompanyWebsiteURL != "https://openai.com" {
		t.Fatalf("expected company website url persisted, got %q", createPayload.Data.CompanyWebsiteURL)
	}
	if createPayload.Data.JDURL != "https://openai.com/careers/backend" {
		t.Fatalf("expected jd url persisted, got %q", createPayload.Data.JDURL)
	}
	if createPayload.Data.Location != "San Francisco, CA" {
		t.Fatalf("expected location persisted, got %q", createPayload.Data.Location)
	}

	leadID := createPayload.Data.ID

	putInput := model.LeadMutationInput{
		Company:           "OpenAI",
		Position:          "Go Backend Engineer",
		Source:            "Direct Apply",
		Status:            "interviewing",
		Priority:          4,
		NextAction:        "prepare interview",
		Notes:             "round 1 scheduled",
		CompanyWebsiteURL: "https://openai.com",
		JDURL:             "https://openai.com/careers/backend",
		Location:          "San Francisco, CA",
	}

	resp = doJSONRequest(t, router, http.MethodPut, "/api/leads/"+leadID, putInput)
	if resp.Code != http.StatusOK {
		t.Fatalf("PUT /api/leads/:id status=%d body=%s", resp.Code, resp.Body.String())
	}

	var updatePayload struct {
		Data model.Lead `json:"data"`
	}
	decodeJSONBody(t, resp, &updatePayload)
	if updatePayload.Data.Status != "interviewing" {
		t.Fatalf("expected status interviewing, got %s", updatePayload.Data.Status)
	}
	if updatePayload.Data.JDURL != "https://openai.com/careers/backend" {
		t.Fatalf("expected jd url retained on put, got %q", updatePayload.Data.JDURL)
	}

	patchInput := model.LeadMutationInput{
		Company:           "OpenAI",
		Position:          "Go Backend Engineer",
		Source:            "Direct Apply",
		Status:            "interviewing",
		Priority:          4,
		NextAction:        "prepare system design",
		Notes:             "updated via patch",
		CompanyWebsiteURL: "https://openai.com",
		JDURL:             "https://openai.com/careers/backend-v2",
		Location:          "Remote (US)",
	}

	resp = doJSONRequest(t, router, http.MethodPatch, "/api/leads/"+leadID, patchInput)
	if resp.Code != http.StatusOK {
		t.Fatalf("PATCH /api/leads/:id status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &updatePayload)
	if updatePayload.Data.Notes != "updated via patch" {
		t.Fatalf("expected patch notes updated, got %s", updatePayload.Data.Notes)
	}
	if updatePayload.Data.JDURL != "https://openai.com/careers/backend-v2" {
		t.Fatalf("expected jd url updated via patch, got %q", updatePayload.Data.JDURL)
	}
	if updatePayload.Data.Location != "Remote (US)" {
		t.Fatalf("expected location updated via patch, got %q", updatePayload.Data.Location)
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/lead-timelines", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/lead-timelines status=%d body=%s", resp.Code, resp.Body.String())
	}

	var timelineListPayload struct {
		Data []model.LeadTimeline `json:"data"`
	}
	decodeJSONBody(t, resp, &timelineListPayload)
	if len(timelineListPayload.Data) == 0 {
		t.Fatal("expected at least one timeline item")
	}
	var tracked model.LeadTimeline
	foundTimeline := false
	for _, item := range timelineListPayload.Data {
		if item.LeadID == leadID {
			tracked = item
			foundTimeline = true
			break
		}
	}
	if !foundTimeline {
		t.Fatalf("expected timeline exists for lead %q", leadID)
	}
	if len(tracked.Stages) < 2 {
		t.Fatalf("expected at least 2 stages tracked, got %d", len(tracked.Stages))
	}
	if tracked.Stages[0].Stage != "new" {
		t.Fatalf("expected first stage new, got %q", tracked.Stages[0].Stage)
	}
	if tracked.Stages[0].EndedAt == "" {
		t.Fatalf("expected first stage ended_at recorded, got empty")
	}
	if tracked.Stages[len(tracked.Stages)-1].Stage != "interviewing" {
		t.Fatalf("expected latest stage interviewing, got %q", tracked.Stages[len(tracked.Stages)-1].Stage)
	}

	resp = doJSONRequest(t, router, http.MethodDelete, "/api/leads/"+leadID, nil)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/leads/:id status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/lead-timelines", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/lead-timelines after lead delete status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &timelineListPayload)
	for _, item := range timelineListPayload.Data {
		if item.LeadID == leadID {
			t.Fatalf("expected timeline deleted with lead, got lead_id=%q", item.LeadID)
		}
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/leads", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/leads after delete status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &listPayload)
	for _, lead := range listPayload.Data {
		if lead.ID == leadID {
			t.Fatalf("deleted lead still exists: %s", leadID)
		}
	}

	chatReq := map[string]any{
		"session_id": "",
		"message":    "hello agent",
		"history":    []map[string]string{},
	}
	resp = doJSONRequest(t, router, http.MethodPost, "/api/agent/chat", chatReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/agent/chat status=%d body=%s", resp.Code, resp.Body.String())
	}

	var chatPayload struct {
		Data struct {
			SessionID string `json:"session_id"`
			Reply     string `json:"reply"`
		} `json:"data"`
	}
	decodeJSONBody(t, resp, &chatPayload)
	if chatPayload.Data.SessionID == "" {
		t.Fatal("chat response session_id should not be empty")
	}
	if chatPayload.Data.Reply == "" {
		t.Fatal("chat response reply should not be empty")
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/agent/sessions", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/agent/sessions status=%d body=%s", resp.Code, resp.Body.String())
	}

	var sessionsPayload struct {
		Data []agentruntime.SessionSummaryView `json:"data"`
	}
	decodeJSONBody(t, resp, &sessionsPayload)
	if len(sessionsPayload.Data) == 0 {
		t.Fatal("expected at least one session summary")
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/agent/sessions/"+chatPayload.Data.SessionID, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/agent/sessions/:id status=%d body=%s", resp.Code, resp.Body.String())
	}

	var sessionPayload struct {
		Data agentruntime.SessionView `json:"data"`
	}
	decodeJSONBody(t, resp, &sessionPayload)
	if sessionPayload.Data.ID == "" {
		t.Fatal("session detail id should not be empty")
	}
	if len(sessionPayload.Data.Messages) < 2 {
		t.Fatalf("expected session detail contains user/assistant messages, got %d", len(sessionPayload.Data.Messages))
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/agent/sessions", nil)
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/agent/sessions status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/agent/settings", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/agent/settings status=%d body=%s", resp.Code, resp.Body.String())
	}

	var settingsPayload struct {
		Data struct {
			Model                string `json:"model"`
			MaxSteps             int    `json:"max_steps"`
			OpenAIBaseURL        string `json:"openai_base_url"`
			OpenAITimeoutSeconds int    `json:"openai_timeout_seconds"`
			HasOpenAIAPIKey      bool   `json:"has_openai_api_key"`
		} `json:"data"`
	}
	decodeJSONBody(t, resp, &settingsPayload)
	if settingsPayload.Data.Model == "" {
		t.Fatal("settings response model should not be empty")
	}

	patchReq := map[string]any{
		"model":                  "gpt-5-mini",
		"max_steps":              8,
		"openai_timeout_seconds": 45,
	}
	resp = doJSONRequest(t, router, http.MethodPatch, "/api/agent/settings", patchReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("PATCH /api/agent/settings status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/user/profile", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/user/profile status=%d body=%s", resp.Code, resp.Body.String())
	}

	profileReq := map[string]any{
		"name":                  "Alice",
		"current_title":         "Senior Backend Engineer",
		"total_years":           6,
		"core_skills":           []string{"Go", "Distributed Systems"},
		"project_evidence":      []string{"设计高可用消息系统"},
		"preferred_roles":       []string{"Staff Backend Engineer"},
		"preferred_locations":   []string{"Remote"},
		"job_search_priorities": []string{"成长空间"},
	}
	resp = doJSONRequest(t, router, http.MethodPut, "/api/user/profile", profileReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("PUT /api/user/profile status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/stats", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/stats status=%d body=%s", resp.Code, resp.Body.String())
	}

	var dashboardPayload struct {
		Data stats.DashboardStats `json:"data"`
	}
	decodeJSONBody(t, resp, &dashboardPayload)
	if dashboardPayload.Data.Overview.Total == 0 {
		t.Fatal("dashboard overview total should not be zero")
	}
	if len(dashboardPayload.Data.Overview.StatusCounts) == 0 {
		t.Fatal("dashboard overview should include status counts")
	}
	if len(dashboardPayload.Data.Funnel.Stages) != 5 {
		t.Fatalf("expected 5 funnel stages, got %d", len(dashboardPayload.Data.Funnel.Stages))
	}
	if dashboardPayload.Data.WeeklyTrend.Period != "week" {
		t.Fatalf("expected weekly trend period=week, got %q", dashboardPayload.Data.WeeklyTrend.Period)
	}
	if len(dashboardPayload.Data.WeeklyTrend.Points) != 7 {
		t.Fatalf("expected 7 weekly points, got %d", len(dashboardPayload.Data.WeeklyTrend.Points))
	}
	if dashboardPayload.Data.MonthlyTrend.Period != "month" {
		t.Fatalf("expected monthly trend period=month, got %q", dashboardPayload.Data.MonthlyTrend.Period)
	}
	if len(dashboardPayload.Data.MonthlyTrend.Points) != 30 {
		t.Fatalf("expected 30 monthly points, got %d", len(dashboardPayload.Data.MonthlyTrend.Points))
	}
	if len(dashboardPayload.Data.Duration.ByStatus) != 5 {
		t.Fatalf("expected 5 duration statuses, got %d", len(dashboardPayload.Data.Duration.ByStatus))
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/stats/", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/stats/ status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/reminders/due", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/reminders/due status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/heartbeat/status", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/heartbeat/status status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/heartbeat/reports/latest", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/heartbeat/reports/latest status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/calendar/interviews.ics", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/calendar/interviews.ics status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, "PROPFIND", "/api/caldav", nil)
	if resp.Code != http.StatusMultiStatus {
		t.Fatalf("PROPFIND /api/caldav status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, "REPORT", "/api/caldav/trace2offer", nil)
	if resp.Code != http.StatusMultiStatus {
		t.Fatalf("REPORT /api/caldav/trace2offer status=%d body=%s", resp.Code, resp.Body.String())
	}
}

type stubAgentRuntime struct {
	settings agentruntime.RuntimeSettings
	profile  agentruntime.UserProfile
	sessions map[string]agentruntime.SessionView
}

func (s *stubAgentRuntime) Run(_ context.Context, request agentruntime.Request) (agentruntime.Response, error) {
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = "session_test_" + time.Now().UTC().Format("150405")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	session := s.ensureSession(sessionID)
	session.Messages = append(session.Messages, agentruntime.SessionMessageView{
		Role:      "user",
		Content:   strings.TrimSpace(request.Message),
		CreatedAt: now,
	})
	session.Messages = append(session.Messages, agentruntime.SessionMessageView{
		Role:      "assistant",
		Content:   "stub reply",
		CreatedAt: now,
	})
	session.UpdatedAt = now
	s.saveSession(session)

	return agentruntime.Response{
		SessionID: sessionID,
		Reply:     "stub reply",
	}, nil
}

func (s *stubAgentRuntime) CreateSession(_ context.Context) (agentruntime.SessionView, error) {
	sessionID := "session_created_" + time.Now().UTC().Format("150405")
	now := time.Now().UTC().Format(time.RFC3339)
	session := agentruntime.SessionView{
		ID:        sessionID,
		UpdatedAt: now,
		Messages:  nil,
	}
	s.saveSession(session)
	return session, nil
}

func (s *stubAgentRuntime) GetSession(_ context.Context, sessionID string) (agentruntime.SessionView, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return agentruntime.SessionView{}, agentruntime.ErrSessionNotFound
	}
	if s.sessions == nil {
		return agentruntime.SessionView{}, agentruntime.ErrSessionNotFound
	}

	session, ok := s.sessions[sessionID]
	if !ok {
		return agentruntime.SessionView{}, agentruntime.ErrSessionNotFound
	}
	return session, nil
}

func (s *stubAgentRuntime) ListSessions(_ context.Context) ([]agentruntime.SessionSummaryView, error) {
	if len(s.sessions) == 0 {
		return nil, nil
	}

	summaries := make([]agentruntime.SessionSummaryView, 0, len(s.sessions))
	for _, item := range s.sessions {
		preview := ""
		if count := len(item.Messages); count > 0 {
			preview = item.Messages[count-1].Content
		}
		summaries = append(summaries, agentruntime.SessionSummaryView{
			ID:           item.ID,
			Title:        "stub",
			Preview:      preview,
			MessageCount: len(item.Messages),
			UpdatedAt:    item.UpdatedAt,
		})
	}
	return summaries, nil
}

func (s *stubAgentRuntime) GetSettings() agentruntime.RuntimeSettingsView {
	if s.settings.Model == "" {
		s.settings = agentruntime.RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			OpenAIBaseURL:        "https://api.openai.com/v1/responses",
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "stub_key",
		}
	}
	return s.settings.View()
}

func (s *stubAgentRuntime) UpdateSettings(_ context.Context, patch agentruntime.RuntimeSettingsPatch) (agentruntime.RuntimeSettingsView, error) {
	settings := s.settings
	if settings.Model == "" {
		settings = agentruntime.RuntimeSettings{
			Model:                "gpt-5-mini",
			MaxSteps:             6,
			OpenAIBaseURL:        "https://api.openai.com/v1/responses",
			OpenAITimeoutSeconds: 60,
			OpenAIAPIKey:         "stub_key",
		}
	}
	s.settings = patch.Apply(settings)
	return s.settings.View(), nil
}

func (s *stubAgentRuntime) GetUserProfile() (agentruntime.UserProfile, error) {
	return s.profile, nil
}

func (s *stubAgentRuntime) UpdateUserProfile(_ context.Context, profile agentruntime.UserProfile) (agentruntime.UserProfile, error) {
	s.profile = profile
	return s.profile, nil
}

func (s *stubAgentRuntime) ImportUserProfileFromResume(_ context.Context, sourceName string, contentType string, _ []byte) (agentruntime.UserProfileImportResult, error) {
	extracted := agentruntime.UserProfile{
		Name:               "stub",
		CurrentTitle:       "imported",
		CoreSkills:         []string{"Go"},
		ProjectEvidence:    []string{"简历导入"},
		PreferredRoles:     []string{"Backend Engineer"},
		PreferredLocations: []string{"Remote"},
	}
	s.profile = extracted
	return agentruntime.UserProfileImportResult{
		Profile:      s.profile,
		Extracted:    extracted,
		SourceName:   sourceName,
		ContentType:  contentType,
		TextLength:   10,
		Truncated:    false,
		ExtractModel: "gpt-5-mini",
	}, nil
}

func (s *stubAgentRuntime) ensureSession(sessionID string) agentruntime.SessionView {
	if s.sessions == nil {
		s.sessions = map[string]agentruntime.SessionView{}
	}

	if session, ok := s.sessions[sessionID]; ok {
		return session
	}
	return agentruntime.SessionView{ID: sessionID}
}

func (s *stubAgentRuntime) saveSession(session agentruntime.SessionView) {
	if s.sessions == nil {
		s.sessions = map[string]agentruntime.SessionView{}
	}
	s.sessions[session.ID] = session
}

func doJSONRequest(t *testing.T, router http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeJSONBody(t *testing.T, resp *httptest.ResponseRecorder, out any) {
	t.Helper()

	if err := json.Unmarshal(resp.Body.Bytes(), out); err != nil {
		t.Fatalf("decode json body: %v; body=%s", err, resp.Body.String())
	}
}
