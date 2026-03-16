package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"trace2offer/backend/internal/model"
)

func TestLeadCreateFromJDURLTool_CreateFromJSONLD(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<!doctype html><html><head>
<title>Senior Go Engineer - Example</title>
<script type="application/ld+json">
{
  "@context":"https://schema.org",
  "@type":"JobPosting",
  "title":"Senior Go Engineer",
  "description":"Build distributed backend systems.",
  "qualifications":"3+ years with Go.",
  "hiringOrganization":{"@type":"Organization","name":"Example Labs"},
  "jobLocation":{
    "@type":"Place",
    "address":{
      "@type":"PostalAddress",
      "addressLocality":"San Francisco",
      "addressRegion":"CA",
      "addressCountry":"US"
    }
  }
}
</script>
</head><body>ok</body></html>`)
	}))
	defer server.Close()

	manager := &testLeadManager{}
	tool := &leadCreateFromJDURLTool{
		manager:      manager,
		httpClient:   server.Client(),
		rJinaBaseURL: defaultRJinaBaseURL,
	}

	rawURL := server.URL + "/jobs/go-engineer?utm_source=newsletter"
	output, err := tool.Run(context.Background(), mustJSON(t, map[string]any{
		"jd_url": rawURL,
	}))
	if err != nil {
		t.Fatalf("run tool failed: %v", err)
	}

	var payload struct {
		Action string     `json:"action"`
		Lead   model.Lead `json:"lead"`
		Parsed struct {
			Parser string `json:"parser"`
		} `json:"parsed"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output failed: %v", err)
	}

	if payload.Action != "created" {
		t.Fatalf("expected action=created, got %q", payload.Action)
	}
	if payload.Parsed.Parser != "json_ld" {
		t.Fatalf("expected parser=json_ld, got %q", payload.Parsed.Parser)
	}
	if payload.Lead.Company != "Example Labs" {
		t.Fatalf("expected company from json-ld, got %q", payload.Lead.Company)
	}
	if payload.Lead.Position != "Senior Go Engineer" {
		t.Fatalf("expected position from json-ld, got %q", payload.Lead.Position)
	}
	if payload.Lead.Location != "San Francisco, CA, US" {
		t.Fatalf("unexpected location: %q", payload.Lead.Location)
	}
	if strings.Contains(payload.Lead.JDURL, "utm_source") {
		t.Fatalf("expected canonical jd_url without tracking params, got %q", payload.Lead.JDURL)
	}
}

func TestLeadCreateFromJDURLTool_UpsertByCanonicalURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<!doctype html><html><head>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"JobPosting","title":"Platform Engineer","hiringOrganization":{"name":"Demo Corp"},"description":"Build platform services."}</script>
</head><body>ok</body></html>`)
	}))
	defer server.Close()

	manager := &testLeadManager{}
	tool := &leadCreateFromJDURLTool{
		manager:      manager,
		httpClient:   server.Client(),
		rJinaBaseURL: defaultRJinaBaseURL,
	}

	firstURL := server.URL + "/jobs/platform?utm_source=a"
	secondURL := server.URL + "/jobs/platform?utm_source=b"

	firstOutput, err := tool.Run(context.Background(), mustJSON(t, map[string]any{
		"jd_url": firstURL,
	}))
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if !strings.Contains(firstOutput, `"action":"created"`) {
		t.Fatalf("expected first action created, got %s", firstOutput)
	}

	secondOutput, err := tool.Run(context.Background(), mustJSON(t, map[string]any{
		"jd_url": secondURL,
	}))
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if !strings.Contains(secondOutput, `"action":"updated"`) {
		t.Fatalf("expected second action updated, got %s", secondOutput)
	}

	leads := manager.List()
	if len(leads) != 1 {
		t.Fatalf("expected one lead after upsert, got %d", len(leads))
	}
}

func TestLeadCreateFromJDURLTool_UsesRJinaForByteDanceStylePage(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Host {
			case "jobs.bytedance.com":
				return newStringResponse(http.StatusOK, "<html><head><title>字节跳动招聘官网｜字节跳动社招</title></head><body></body></html>", req), nil
			case "r.jina.ai":
				return newStringResponse(http.StatusOK, `Title: Agent技术研发工程师-豆包 - 字节跳动

URL Source: http://jobs.bytedance.com/experienced/position/7523153596519794951/detail

Agent技术研发工程师-豆包
上海 正式 研发 职位 ID：A51790A
职位描述
负责豆包创意Agent技术研发，提升大模型在创意场景的应用能力。
职位要求
熟练掌握Python/Java/Go等至少一门语言。`, req), nil
			default:
				return nil, fmt.Errorf("unexpected host: %s", req.URL.Host)
			}
		}),
	}

	manager := &testLeadManager{}
	tool := &leadCreateFromJDURLTool{
		manager:      manager,
		httpClient:   client,
		rJinaBaseURL: defaultRJinaBaseURL,
	}

	output, err := tool.Run(context.Background(), mustJSON(t, map[string]any{
		"jd_url": "https://jobs.bytedance.com/experienced/position/7523153596519794951/detail?recomId=abc",
	}))
	if err != nil {
		t.Fatalf("run tool failed: %v", err)
	}

	var payload struct {
		Action string     `json:"action"`
		Lead   model.Lead `json:"lead"`
		Parsed struct {
			Parser string `json:"parser"`
		} `json:"parsed"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output failed: %v", err)
	}

	if payload.Action != "created" {
		t.Fatalf("expected action=created, got %q", payload.Action)
	}
	if payload.Parsed.Parser != "bytedance_rjina" {
		t.Fatalf("expected parser bytedance_rjina, got %q", payload.Parsed.Parser)
	}
	if payload.Lead.Company != "ByteDance" {
		t.Fatalf("expected company ByteDance, got %q", payload.Lead.Company)
	}
	if payload.Lead.Position != "Agent技术研发工程师-豆包" {
		t.Fatalf("expected position parsed from markdown, got %q", payload.Lead.Position)
	}
	if payload.Lead.Location != "上海" {
		t.Fatalf("expected location 上海, got %q", payload.Lead.Location)
	}
	if strings.Contains(payload.Lead.JDURL, "recomId") {
		t.Fatalf("expected bytedance jd_url without recomId, got %q", payload.Lead.JDURL)
	}
}

func TestLeadCreateFromJDURLTool_PrefersLLMExtraction(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html><head><title>请稍候</title></head><body>loading</body></html>")
	}))
	defer server.Close()

	manager := &testLeadManager{}
	tool := &leadCreateFromJDURLTool{
		manager:      manager,
		httpClient:   server.Client(),
		rJinaBaseURL: defaultRJinaBaseURL,
		extractor: &stubJDExtractor{
			result: jdExtraction{
				Company:      "TestCorp",
				Position:     "Staff Backend Engineer",
				Location:     "Shanghai",
				Description:  "Build infra",
				Requirements: "Go, distributed systems",
				Parser:       "llm_jina",
				Confidence:   0.91,
			},
		},
	}

	output, err := tool.Run(context.Background(), mustJSON(t, map[string]any{
		"jd_url": server.URL + "/job/abc?utm_source=foo",
	}))
	if err != nil {
		t.Fatalf("run tool failed: %v", err)
	}

	var payload struct {
		Lead struct {
			Company  string `json:"company"`
			Position string `json:"position"`
			Location string `json:"location"`
		} `json:"lead"`
		Parsed struct {
			Parser string `json:"parser"`
		} `json:"parsed"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output failed: %v", err)
	}
	if payload.Lead.Company != "TestCorp" {
		t.Fatalf("expected llm company, got %q", payload.Lead.Company)
	}
	if payload.Lead.Position != "Staff Backend Engineer" {
		t.Fatalf("expected llm position, got %q", payload.Lead.Position)
	}
	if payload.Lead.Location != "Shanghai" {
		t.Fatalf("expected llm location, got %q", payload.Lead.Location)
	}
	if payload.Parsed.Parser != "llm_jina" {
		t.Fatalf("expected parser llm_jina, got %q", payload.Parsed.Parser)
	}
}

func TestParseJDLLMOutput_WithCodeFence(t *testing.T) {
	t.Parallel()

	raw := "```json\n{\"company\":\"A\",\"position\":\"B\",\"location\":\"C\",\"description\":\"D\",\"requirements\":\"E\",\"confidence\":0.83}\n```"
	parsed, err := parseJDLLMOutput(raw)
	if err != nil {
		t.Fatalf("parse output failed: %v", err)
	}
	if parsed.Company != "A" || parsed.Position != "B" || parsed.Confidence != 0.83 {
		t.Fatalf("unexpected parsed content: %+v", parsed)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json failed: %v", err)
	}
	return payload
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newStringResponse(statusCode int, body string, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

type testLeadManager struct {
	leads []model.Lead
	seq   int
}

type stubJDExtractor struct {
	result jdExtraction
	err    error
}

func (s *stubJDExtractor) Extract(_ context.Context, _ string, _ string) (jdExtraction, error) {
	if s == nil {
		return jdExtraction{}, nil
	}
	return s.result, s.err
}

func (m *testLeadManager) List() []model.Lead {
	copied := make([]model.Lead, len(m.leads))
	copy(copied, m.leads)
	return copied
}

func (m *testLeadManager) Create(input model.LeadMutationInput) (model.Lead, error) {
	m.seq++
	created := model.Lead{
		ID:                fmt.Sprintf("lead_%d", m.seq),
		Company:           input.Company,
		Position:          input.Position,
		Source:            input.Source,
		Status:            input.Status,
		Priority:          input.Priority,
		NextAction:        input.NextAction,
		Notes:             input.Notes,
		CompanyWebsiteURL: input.CompanyWebsiteURL,
		JDURL:             input.JDURL,
		Location:          input.Location,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	m.leads = append(m.leads, created)
	return created, nil
}

func (m *testLeadManager) Update(id string, input model.LeadMutationInput) (model.Lead, bool, error) {
	for i := range m.leads {
		if m.leads[i].ID != id {
			continue
		}
		m.leads[i].Company = input.Company
		m.leads[i].Position = input.Position
		m.leads[i].Source = input.Source
		m.leads[i].Status = input.Status
		m.leads[i].Priority = input.Priority
		m.leads[i].NextAction = input.NextAction
		m.leads[i].Notes = input.Notes
		m.leads[i].CompanyWebsiteURL = input.CompanyWebsiteURL
		m.leads[i].JDURL = input.JDURL
		m.leads[i].Location = input.Location
		m.leads[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return m.leads[i], true, nil
	}
	return model.Lead{}, false, nil
}

func (m *testLeadManager) Delete(id string) (bool, error) {
	for i := range m.leads {
		if m.leads[i].ID != id {
			continue
		}
		m.leads = append(m.leads[:i], m.leads[i+1:]...)
		return true, nil
	}
	return false, nil
}
