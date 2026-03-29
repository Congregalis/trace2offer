package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agentruntime "trace2offer/backend/agent"
	"trace2offer/backend/internal/calendar"
	"trace2offer/backend/internal/candidate"
	"trace2offer/backend/internal/discovery"
	"trace2offer/backend/internal/heartbeat"
	"trace2offer/backend/internal/model"
	"trace2offer/backend/internal/prep"
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
	candidateStore, err := storage.NewFileCandidateStore(filepath.Join(tmpDir, "candidates.json"))
	if err != nil {
		t.Fatalf("init candidate store: %v", err)
	}
	discoveryRuleStore, err := storage.NewFileDiscoveryRuleStore(filepath.Join(tmpDir, "discovery_rules.json"))
	if err != nil {
		t.Fatalf("init discovery rule store: %v", err)
	}
	leadTimelineStore, err := storage.NewFileLeadTimelineStore(filepath.Join(tmpDir, "lead_timelines.json"))
	if err != nil {
		t.Fatalf("init lead timeline store: %v", err)
	}
	discoveryService := discovery.NewService(discoveryRuleStore, candidate.NewService(candidateStore, nil))
	statsService := stats.NewService(leadStore)
	reminderService := reminder.NewService(leadStore)
	heartbeatService, err := heartbeat.NewService(heartbeat.Config{
		DataDir:          tmpDir,
		ReminderService:  reminderService,
		StatsService:     statsService,
		DiscoveryService: discoveryService,
	})
	if err != nil {
		t.Fatalf("init heartbeat service: %v", err)
	}
	_ = heartbeatService.RunOnce(time.Now().UTC())
	prepConfig, err := prep.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("load prep config: %v", err)
	}
	prepIndexStore, err := prep.NewIndexStore(prepConfig.IndexDBPath)
	if err != nil {
		t.Fatalf("init prep index store: %v", err)
	}
	t.Cleanup(func() {
		_ = prepIndexStore.Close()
	})
	prepService, err := prep.NewService(
		prepConfig,
		prep.WithIndexStore(prepIndexStore),
		prep.WithEmbeddingProvider(&stubPrepEmbeddingProvider{}),
	)
	if err != nil {
		t.Fatalf("init prep service: %v", err)
	}

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Jobs</title>
    <item>
      <title>Backend Engineer at Example</title>
      <link>https://jobs.example.com/backend-1?utm_source=rss</link>
      <description>Go distributed systems cloud platform</description>
    </item>
  </channel>
</rss>`))
	}))
	defer feedServer.Close()

	router := NewRouter(leadStore, candidateStore, leadTimelineStore, &stubAgentRuntime{}, statsService, reminderService, heartbeatService, calendar.NewService(leadStore), discoveryService, prepService)

	resp := doJSONRequest(t, router, http.MethodGet, "/api/prep/meta", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/meta status=%d body=%s", resp.Code, resp.Body.String())
	}

	var prepMetaPayload struct {
		Data prep.Meta `json:"data"`
	}
	decodeJSONBody(t, resp, &prepMetaPayload)
	if !prepMetaPayload.Data.Enabled {
		t.Fatal("expected prep meta enabled true")
	}
	if prepMetaPayload.Data.DefaultQuestionCount != 8 {
		t.Fatalf("expected default_question_count=8, got %d", prepMetaPayload.Data.DefaultQuestionCount)
	}
	if len(prepMetaPayload.Data.SupportedScopes) != 1 {
		t.Fatalf("expected 1 supported scope, got %d", len(prepMetaPayload.Data.SupportedScopes))
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/prep/documents", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/documents status=%d body=%s", resp.Code, resp.Body.String())
	}
	var documentListPayload struct {
		Data []prep.KnowledgeDocument `json:"data"`
	}
	decodeJSONBody(t, resp, &documentListPayload)
	if len(documentListPayload.Data) != 0 {
		t.Fatalf("expected empty prep documents initially, got %d", len(documentListPayload.Data))
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/documents", map[string]any{
		"filename": "overview",
		"content":  "# RAG\n\nv1",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/prep/documents status=%d body=%s", resp.Code, resp.Body.String())
	}
	var documentPayload struct {
		Data prep.KnowledgeDocument `json:"data"`
	}
	decodeJSONBody(t, resp, &documentPayload)
	if documentPayload.Data.Filename != "overview.md" {
		t.Fatalf("expected created filename overview.md, got %q", documentPayload.Data.Filename)
	}

	resp = doJSONRequest(t, router, http.MethodPut, "/api/prep/documents/overview.md", map[string]any{
		"content": "# RAG\n\nupdated",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("PUT /api/prep/documents/:filename status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &documentPayload)
	if documentPayload.Data.Content != "# RAG\n\nupdated" {
		t.Fatalf("expected updated document content, got %q", documentPayload.Data.Content)
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/prep/documents", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/documents status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &documentListPayload)
	if len(documentListPayload.Data) != 1 {
		t.Fatalf("expected one prep document, got %d", len(documentListPayload.Data))
	}
	if documentListPayload.Data[0].Content != "# RAG\n\nupdated" {
		t.Fatalf("expected listed document content updated, got %q", documentListPayload.Data[0].Content)
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/index/rebuild", map[string]any{
		"scope": "*",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/prep/index/rebuild status=%d body=%s", resp.Code, resp.Body.String())
	}
	var indexSummaryPayload struct {
		Data prep.IndexRunSummary `json:"data"`
	}
	decodeJSONBody(t, resp, &indexSummaryPayload)
	if indexSummaryPayload.Data.RunID == "" {
		t.Fatalf("expected index run id populated")
	}
	if indexSummaryPayload.Data.Mode != prep.RebuildModeIncremental {
		t.Fatalf("expected default rebuild mode incremental, got %+v", indexSummaryPayload.Data)
	}
	if indexSummaryPayload.Data.DocumentsScanned == 0 {
		t.Fatalf("expected index rebuild scans documents, got %+v", indexSummaryPayload.Data)
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/index/rebuild", map[string]any{
		"scope": "*",
		"mode":  "full",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/prep/index/rebuild full status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &indexSummaryPayload)
	if indexSummaryPayload.Data.Mode != prep.RebuildModeFull {
		t.Fatalf("expected full rebuild mode, got %+v", indexSummaryPayload.Data)
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/prep/index/documents", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/index/documents status=%d body=%s", resp.Code, resp.Body.String())
	}
	var indexDocumentListPayload struct {
		Data []prep.Document `json:"data"`
	}
	decodeJSONBody(t, resp, &indexDocumentListPayload)
	if len(indexDocumentListPayload.Data) == 0 {
		t.Fatal("expected indexed documents not empty")
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/prep/index/chunks?document_id="+url.QueryEscape(indexDocumentListPayload.Data[0].ID)+"&limit=20", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/index/chunks status=%d body=%s", resp.Code, resp.Body.String())
	}
	var indexChunkListPayload struct {
		Data []prep.IndexedChunk `json:"data"`
	}
	decodeJSONBody(t, resp, &indexChunkListPayload)
	if len(indexChunkListPayload.Data) == 0 {
		t.Fatal("expected indexed chunks not empty")
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/leads", nil)
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
		JDText:            "create jd text",
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
	if createPayload.Data.JDText != "create jd text" {
		t.Fatalf("expected jd text persisted, got %q", createPayload.Data.JDText)
	}
	if createPayload.Data.Location != "San Francisco, CA" {
		t.Fatalf("expected location persisted, got %q", createPayload.Data.Location)
	}

	leadID := createPayload.Data.ID

	if err := os.MkdirAll(filepath.Join(tmpDir, "resume"), 0o755); err != nil {
		t.Fatalf("mkdir resume dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "resume", "current.md"), []byte("# Resume\n\nGo backend engineer"), 0o644); err != nil {
		t.Fatalf("write resume current.md: %v", err)
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/prep/leads/"+leadID+"/context-preview", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/leads/:lead_id/context-preview status=%d body=%s", resp.Code, resp.Body.String())
	}
	var contextPreviewPayload struct {
		Data prep.LeadContextPreview `json:"data"`
	}
	decodeJSONBody(t, resp, &contextPreviewPayload)
	if contextPreviewPayload.Data.LeadID != leadID {
		t.Fatalf("expected context lead_id=%q, got %q", leadID, contextPreviewPayload.Data.LeadID)
	}
	if !contextPreviewPayload.Data.HasResume {
		t.Fatal("expected context has_resume=true")
	}
	if !hasPrepContextSource(contextPreviewPayload.Data.Sources, "lead", "jd_text", "JD 原文") {
		t.Fatalf("expected jd source in context preview, got %+v", contextPreviewPayload.Data.Sources)
	}
	if !hasPrepContextSource(contextPreviewPayload.Data.Sources, "library", "markdown", "overview.md") {
		t.Fatalf("expected library source in context preview, got %+v", contextPreviewPayload.Data.Sources)
	}
	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/retrieval/preview", map[string]any{
		"lead_id":       leadID,
		"query":         "RAG 常见面试问题",
		"top_k":         5,
		"include_trace": true,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/prep/retrieval/preview status=%d body=%s", resp.Code, resp.Body.String())
	}
	var retrievalPreviewPayload struct {
		Data prep.SearchResult `json:"data"`
	}
	decodeJSONBody(t, resp, &retrievalPreviewPayload)
	if retrievalPreviewPayload.Data.NormalizedQuery == "" {
		t.Fatalf("expected normalized_query in retrieval payload")
	}
	if retrievalPreviewPayload.Data.Trace == nil {
		t.Fatalf("expected retrieval trace in retrieval payload")
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/sessions", map[string]any{
		"lead_id":           leadID,
		"question_count":    2,
		"include_resume":    true,
		"include_lead_docs": true,
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/prep/sessions status=%d body=%s", resp.Code, resp.Body.String())
	}
	var createdPrepSessionPayload struct {
		Data prep.Session `json:"data"`
	}
	decodeJSONBody(t, resp, &createdPrepSessionPayload)
	if createdPrepSessionPayload.Data.ID == "" {
		t.Fatal("expected created prep session id")
	}
	if createdPrepSessionPayload.Data.GenerationTrace.RetrievalQuery == "" {
		t.Fatal("expected generation trace retrieval query")
	}

	resp = doJSONRequest(t, router, http.MethodGet, "/api/prep/sessions/"+createdPrepSessionPayload.Data.ID, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/prep/sessions/:session_id status=%d body=%s", resp.Code, resp.Body.String())
	}
	decodeJSONBody(t, resp, &createdPrepSessionPayload)
	if createdPrepSessionPayload.Data.ID == "" {
		t.Fatal("expected session detail id")
	}

	sessionFixture := prep.Session{
		ID:       "prep_01",
		LeadID:   leadID,
		Company:  "OpenAI",
		Position: "Go Backend Engineer",
		Status:   prep.PrepSessionStatusDraft,
		Questions: []prep.Question{{
			ID:      1,
			Type:    "open",
			Content: "What is RAG?",
		}, {
			ID:      2,
			Type:    "open",
			Content: "How does retrieval work?",
		}},
		CreatedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
	}
	sessionFixtureRaw, err := json.MarshalIndent(sessionFixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal prep session fixture: %v", err)
	}
	sessionFixturePath := filepath.Join(tmpDir, "prep", "sessions", "prep_01.json")
	if err := os.WriteFile(sessionFixturePath, sessionFixtureRaw, 0o644); err != nil {
		t.Fatalf("write prep session fixture: %v", err)
	}

	resp = doJSONRequest(t, router, http.MethodPut, "/api/prep/sessions/prep_01/draft-answers", map[string]any{
		"answers": []map[string]any{
			{"question_id": 1, "answer": "RAG stands for Retrieval-Augmented Generation"},
			{"question_id": 2, "answer": "Retriever + Generator"},
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("PUT /api/prep/sessions/:session_id/draft-answers status=%d body=%s", resp.Code, resp.Body.String())
	}
	var saveDraftPayload struct {
		Data prep.SaveDraftAnswersResult `json:"data"`
	}
	decodeJSONBody(t, resp, &saveDraftPayload)
	if saveDraftPayload.Data.SessionID != "prep_01" {
		t.Fatalf("expected session_id=prep_01, got %q", saveDraftPayload.Data.SessionID)
	}
	if saveDraftPayload.Data.AnswersCount != 2 {
		t.Fatalf("expected answers_count=2, got %d", saveDraftPayload.Data.AnswersCount)
	}
	if saveDraftPayload.Data.SavedAt == "" {
		t.Fatalf("expected non-empty saved_at")
	}

	storedSessionRaw, err := os.ReadFile(sessionFixturePath)
	if err != nil {
		t.Fatalf("read updated prep session fixture: %v", err)
	}
	var storedSession prep.Session
	if err := json.Unmarshal(storedSessionRaw, &storedSession); err != nil {
		t.Fatalf("decode updated prep session fixture: %v", err)
	}
	if len(storedSession.Answers) != 2 {
		t.Fatalf("expected 2 persisted answers, got %d", len(storedSession.Answers))
	}

	resp = doJSONRequest(t, router, http.MethodPut, "/api/prep/sessions/prep_01/draft-answers", map[string]any{
		"answers": []map[string]any{{"question_id": 99, "answer": "invalid question"}},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("PUT /api/prep/sessions/:session_id/draft-answers invalid question status=%d body=%s", resp.Code, resp.Body.String())
	}

	sessionFixture.Status = prep.PrepSessionStatusSubmitted
	sessionFixtureRaw, err = json.MarshalIndent(sessionFixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal submitted prep session fixture: %v", err)
	}
	if err := os.WriteFile(sessionFixturePath, sessionFixtureRaw, 0o644); err != nil {
		t.Fatalf("write submitted prep session fixture: %v", err)
	}

	resp = doJSONRequest(t, router, http.MethodPut, "/api/prep/sessions/prep_01/draft-answers", map[string]any{
		"answers": []map[string]any{{"question_id": 1, "answer": "should fail when submitted"}},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("PUT /api/prep/sessions/:session_id/draft-answers submitted status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodPut, "/api/prep/sessions/missing/draft-answers", map[string]any{
		"answers": []map[string]any{{"question_id": 1, "answer": "x"}},
	})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("PUT /api/prep/sessions/:session_id/draft-answers missing status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodDelete, "/api/prep/documents/overview.md", nil)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/prep/documents/:filename status=%d body=%s", resp.Code, resp.Body.String())
	}

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
		JDText:            "put jd text",
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
	if updatePayload.Data.JDText != "put jd text" {
		t.Fatalf("expected jd text updated via put, got %q", updatePayload.Data.JDText)
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
		JDText:            "patch jd text",
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
	if updatePayload.Data.JDText != "patch jd text" {
		t.Fatalf("expected jd text updated via patch, got %q", updatePayload.Data.JDText)
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

	candidateCreateReq := map[string]any{
		"company":              "Anthropic",
		"position":             "Software Engineer",
		"source":               "LinkedIn",
		"location":             "Remote",
		"jd_url":               "https://www.anthropic.com/careers/software-engineer",
		"company_website_url":  "https://www.anthropic.com",
		"match_score":          88,
		"match_reasons":        []string{"Go backend", "LLM infra"},
		"recommendation_notes": "符合画像",
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/candidates", candidateCreateReq)
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/candidates status=%d body=%s", resp.Code, resp.Body.String())
	}

	var candidateCreatePayload struct {
		Data model.Candidate `json:"data"`
	}
	decodeJSONBody(t, resp, &candidateCreatePayload)
	if candidateCreatePayload.Data.ID == "" {
		t.Fatal("created candidate ID should not be empty")
	}
	if candidateCreatePayload.Data.Status != "pending_review" {
		t.Fatalf("expected default candidate status pending_review, got %q", candidateCreatePayload.Data.Status)
	}

	candidateID := candidateCreatePayload.Data.ID

	resp = doJSONRequest(t, router, http.MethodGet, "/api/candidates", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/candidates status=%d body=%s", resp.Code, resp.Body.String())
	}

	var candidateListPayload struct {
		Data []model.Candidate `json:"data"`
	}
	decodeJSONBody(t, resp, &candidateListPayload)
	if len(candidateListPayload.Data) == 0 {
		t.Fatal("expected candidates list not empty")
	}

	candidateUpdateReq := map[string]any{
		"company":              "Anthropic",
		"position":             "Software Engineer",
		"source":               "LinkedIn",
		"location":             "Remote",
		"jd_url":               "https://www.anthropic.com/careers/software-engineer",
		"company_website_url":  "https://www.anthropic.com",
		"status":               "shortlisted",
		"match_score":          92,
		"match_reasons":        []string{"Go backend", "LLM infra", "系统设计"},
		"recommendation_notes": "更新后更匹配",
	}
	resp = doJSONRequest(t, router, http.MethodPatch, "/api/candidates/"+candidateID, candidateUpdateReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("PATCH /api/candidates/:id status=%d body=%s", resp.Code, resp.Body.String())
	}

	var candidateUpdatePayload struct {
		Data model.Candidate `json:"data"`
	}
	decodeJSONBody(t, resp, &candidateUpdatePayload)
	if candidateUpdatePayload.Data.Status != "shortlisted" {
		t.Fatalf("expected status shortlisted, got %q", candidateUpdatePayload.Data.Status)
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/candidates/"+candidateID+"/promote", map[string]any{
		"status":      "new",
		"priority":    5,
		"next_action": "完善简历并投递",
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/candidates/:id/promote status=%d body=%s", resp.Code, resp.Body.String())
	}

	var candidatePromotePayload struct {
		Data struct {
			Candidate model.Candidate `json:"candidate"`
			Lead      model.Lead      `json:"lead"`
		} `json:"data"`
	}
	decodeJSONBody(t, resp, &candidatePromotePayload)
	if candidatePromotePayload.Data.Lead.ID == "" {
		t.Fatal("promoted lead id should not be empty")
	}
	if candidatePromotePayload.Data.Candidate.Status != "promoted" {
		t.Fatalf("expected candidate status promoted, got %q", candidatePromotePayload.Data.Candidate.Status)
	}

	resp = doJSONRequest(t, router, http.MethodDelete, "/api/candidates/"+candidateID, nil)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/candidates/:id status=%d body=%s", resp.Code, resp.Body.String())
	}

	discoveryCreateReq := map[string]any{
		"name":             "example jobs",
		"feed_url":         feedServer.URL,
		"source":           "rss:example",
		"default_location": "Remote",
		"include_keywords": []string{"go", "backend"},
		"exclude_keywords": []string{"intern"},
		"enabled":          true,
	}
	resp = doJSONRequest(t, router, http.MethodPost, "/api/discovery/rules", discoveryCreateReq)
	if resp.Code != http.StatusCreated {
		t.Fatalf("POST /api/discovery/rules status=%d body=%s", resp.Code, resp.Body.String())
	}
	var discoveryCreatePayload struct {
		Data model.DiscoveryRule `json:"data"`
	}
	decodeJSONBody(t, resp, &discoveryCreatePayload)
	if discoveryCreatePayload.Data.ID == "" {
		t.Fatal("created discovery rule id should not be empty")
	}
	discoveryRuleID := discoveryCreatePayload.Data.ID

	resp = doJSONRequest(t, router, http.MethodGet, "/api/discovery/rules", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/discovery/rules status=%d body=%s", resp.Code, resp.Body.String())
	}
	var discoveryListPayload struct {
		Data []model.DiscoveryRule `json:"data"`
	}
	decodeJSONBody(t, resp, &discoveryListPayload)
	if len(discoveryListPayload.Data) == 0 {
		t.Fatal("expected discovery rules not empty")
	}

	resp = doJSONRequest(t, router, http.MethodPost, "/api/discovery/run", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/discovery/run status=%d body=%s", resp.Code, resp.Body.String())
	}
	var discoveryRunPayload struct {
		Data model.DiscoveryRunResult `json:"data"`
	}
	decodeJSONBody(t, resp, &discoveryRunPayload)
	if discoveryRunPayload.Data.RulesExecuted == 0 {
		t.Fatalf("expected discovery rules executed > 0, got %+v", discoveryRunPayload.Data)
	}

	discoveryUpdateReq := map[string]any{
		"name":             "example jobs disabled",
		"feed_url":         feedServer.URL,
		"source":           "rss:example",
		"default_location": "Remote",
		"include_keywords": []string{"go"},
		"enabled":          false,
	}
	resp = doJSONRequest(t, router, http.MethodPatch, "/api/discovery/rules/"+discoveryRuleID, discoveryUpdateReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("PATCH /api/discovery/rules/:id status=%d body=%s", resp.Code, resp.Body.String())
	}

	resp = doJSONRequest(t, router, http.MethodDelete, "/api/discovery/rules/"+discoveryRuleID, nil)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/discovery/rules/:id status=%d body=%s", resp.Code, resp.Body.String())
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

	resp = doMultipartFileRequest(
		t,
		router,
		http.MethodPost,
		"/api/user/profile/import",
		"resume",
		"resume.txt",
		"text/plain",
		[]byte("Go engineer\n\nplatform"),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("POST /api/user/profile/import status=%d body=%s", resp.Code, resp.Body.String())
	}

	var importPayload struct {
		Data struct {
			ResumePath       string `json:"resume_path"`
			ResumeTotalChars int    `json:"resume_total_chars"`
			ResumeTruncated  bool   `json:"resume_truncated"`
			Profile          struct {
				Name string `json:"name"`
			} `json:"profile"`
		} `json:"data"`
	}
	decodeJSONBody(t, resp, &importPayload)
	if importPayload.Data.ResumePath == "" {
		t.Fatal("expected import response resume_path not empty")
	}
	if importPayload.Data.ResumeTotalChars == 0 {
		t.Fatal("expected import response resume_total_chars > 0")
	}
	if importPayload.Data.ResumeTruncated {
		t.Fatal("expected import response resume_truncated=false")
	}
	if importPayload.Data.Profile.Name == "" {
		t.Fatal("expected import response profile name not empty")
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

func TestPrepEndpointsReturnConfigErrorWhenHFUnavailable(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	leadStore, err := storage.NewFileLeadStore(filepath.Join(tmpDir, "leads.json"))
	if err != nil {
		t.Fatalf("init lead store: %v", err)
	}
	candidateStore, err := storage.NewFileCandidateStore(filepath.Join(tmpDir, "candidates.json"))
	if err != nil {
		t.Fatalf("init candidate store: %v", err)
	}
	leadTimelineStore, err := storage.NewFileLeadTimelineStore(filepath.Join(tmpDir, "lead_timelines.json"))
	if err != nil {
		t.Fatalf("init lead timeline store: %v", err)
	}
	statsService := stats.NewService(leadStore)
	reminderService := reminder.NewService(leadStore)

	prepConfig, err := prep.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("load prep config: %v", err)
	}
	prepService, err := prep.NewService(prepConfig)
	if err != nil {
		t.Fatalf("init prep service: %v", err)
	}
	if _, err := prepService.CreateDocument(prep.KnowledgeDocumentCreateInput{Filename: "rag", Content: "retrieval"}); err != nil {
		t.Fatalf("create prep document: %v", err)
	}

	leadList := leadStore.List()
	if len(leadList) == 0 {
		t.Fatal("expected seeded leads")
	}
	leadID := strings.TrimSpace(leadList[0].ID)
	if leadID == "" {
		t.Fatal("expected non-empty lead id")
	}

	router := NewRouter(leadStore, candidateStore, leadTimelineStore, nil, statsService, reminderService, nil, nil, nil, prepService)

	assertConfigError := func(resp *httptest.ResponseRecorder, route string) {
		t.Helper()
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s expected 400, got %d body=%s", route, resp.Code, resp.Body.String())
		}
		var payload struct {
			Message string `json:"message"`
		}
		decodeJSONBody(t, resp, &payload)
		if !strings.Contains(payload.Message, "T2O_PREP_HF_API_KEY") {
			t.Fatalf("%s expected message contains T2O_PREP_HF_API_KEY, got %q", route, payload.Message)
		}
	}

	resp := doJSONRequest(t, router, http.MethodPost, "/api/prep/index/rebuild", map[string]any{
		"scope": "*",
	})
	assertConfigError(resp, "/api/prep/index/rebuild")

	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/retrieval/preview", map[string]any{
		"lead_id":           leadID,
		"query":             "RAG",
		"include_resume":    true,
		"include_lead_docs": true,
	})
	assertConfigError(resp, "/api/prep/retrieval/preview")

	resp = doJSONRequest(t, router, http.MethodPost, "/api/prep/sessions", map[string]any{
		"lead_id":           leadID,
		"question_count":    1,
		"include_resume":    true,
		"include_lead_docs": true,
	})
	assertConfigError(resp, "/api/prep/sessions")
}

type stubAgentRuntime struct {
	settings agentruntime.RuntimeSettings
	profile  agentruntime.UserProfile
	sessions map[string]agentruntime.SessionView
}

type stubPrepEmbeddingProvider struct{}

func (p *stubPrepEmbeddingProvider) Name() string {
	return "stub"
}

func (p *stubPrepEmbeddingProvider) Model() string {
	return "stub-v1"
}

func (p *stubPrepEmbeddingProvider) Dimension() int {
	return 2
}

func (p *stubPrepEmbeddingProvider) Embed(texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for range texts {
		vectors = append(vectors, []float32{0.5, 0.5})
	}
	return vectors, nil
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
		Profile:          s.profile,
		Extracted:        extracted,
		SourceName:       sourceName,
		ContentType:      contentType,
		TextLength:       10,
		Truncated:        false,
		ExtractModel:     "gpt-5-mini",
		ResumePath:       "/tmp/resume/current.md",
		ResumeTotalChars: 1024,
		ResumeTruncated:  false,
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

func doMultipartFileRequest(t *testing.T, router http.Handler, method string, path string, field string, filename string, contentType string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, path, payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())

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

func hasPrepContextSource(sources []prep.ContextSource, scope string, kind string, title string) bool {
	for _, source := range sources {
		if source.Scope == scope && source.Kind == kind && source.Title == title {
			return true
		}
	}
	return false
}
