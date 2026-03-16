package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"

	"trace2offer/backend/agent/provider"
)

const (
	maxResumeFileSizeBytes = 8 * 1024 * 1024
	maxResumeTextRunes     = 24000
)

var (
	errResumeFileTooLarge   = &ResumeImportError{Message: "简历文件太大，限制 8MB"}
	errResumeFileEmpty      = &ResumeImportError{Message: "简历文件为空"}
	errResumeTextEmpty      = &ResumeImportError{Message: "简历内容为空，无法导入能力画像"}
	errResumeFormatNotFound = &ResumeImportError{Message: "暂不支持该简历格式，请上传 pdf / docx / txt / md"}

	resumeJSONObjectPattern = regexp.MustCompile(`(?s)\{.*\}`)
	xmlTagPattern           = regexp.MustCompile(`(?s)<[^>]+>`)
	multiWhitespacePattern  = regexp.MustCompile(`[\t\f\v ]+`)
)

// ResumeImportError means client input is invalid for resume import.
type ResumeImportError struct {
	Message string
}

func (e *ResumeImportError) Error() string {
	if e == nil || strings.TrimSpace(e.Message) == "" {
		return "resume import failed"
	}
	return strings.TrimSpace(e.Message)
}

func IsResumeImportError(err error) bool {
	var resumeErr *ResumeImportError
	return errors.As(err, &resumeErr)
}

// UserProfileImportResult returns both merged and extracted profile.
type UserProfileImportResult struct {
	Profile      UserProfile `json:"profile"`
	Extracted    UserProfile `json:"extracted"`
	SourceName   string      `json:"source_name"`
	ContentType  string      `json:"content_type"`
	TextLength   int         `json:"text_length"`
	Truncated    bool        `json:"truncated"`
	ExtractModel string      `json:"extract_model"`
}

type resumeImporter struct {
	modelProvider provider.Provider
	model         string
}

func newResumeImporter(modelProvider provider.Provider, model string) *resumeImporter {
	if modelProvider == nil {
		return nil
	}
	return &resumeImporter{
		modelProvider: modelProvider,
		model:         strings.TrimSpace(model),
	}
}

const resumeProfileExtractionPrompt = `你是“简历能力画像抽取器”。
任务：从简历文本中提取候选人的能力画像。
规则：
1) 只输出一个 JSON 对象，不要 markdown，不要解释。
2) 只允许以下字段：
name,current_title,total_years,primary_industry,industries,core_skills,tooling,programming_languages,domain_knowledge,project_evidence,achievements,education,certifications,preferred_roles,preferred_industries,preferred_locations,remote_preference,employment_types,salary_expectation,work_authorization,visa_needs,preferred_company_stages,excluded_companies,job_search_priorities,strength_summary,portfolio_links,notes。
3) 列表字段必须是字符串数组，total_years 必须是数字，其他字段是字符串。
4) 缺失信息返回空字符串、0 或空数组。
5) 不得编造简历中不存在的信息。`

func (i *resumeImporter) Import(ctx context.Context, resumeText string) (UserProfile, bool, error) {
	if i == nil || i.modelProvider == nil {
		return UserProfile{}, false, errors.New("resume importer is unavailable")
	}

	cleaned := normalizeResumeText(resumeText)
	if cleaned == "" {
		return UserProfile{}, false, errResumeTextEmpty
	}

	truncated := false
	if utf8.RuneCountInString(cleaned) > maxResumeTextRunes {
		cleaned = truncateByRunes(cleaned, maxResumeTextRunes)
		truncated = true
	}

	response, err := i.modelProvider.Generate(ctx, provider.Request{
		Model: i.model,
		Messages: []provider.Message{
			{Role: "system", Content: resumeProfileExtractionPrompt},
			{Role: "user", Content: "resume_text:\n" + cleaned},
		},
	})
	if err != nil {
		return UserProfile{}, false, fmt.Errorf("extract resume profile with llm: %w", err)
	}

	parsed, err := parseResumeProfileOutput(response.Content)
	if err != nil {
		return UserProfile{}, false, err
	}
	parsed.UpdatedAt = ""
	return normalizeUserProfile(parsed), truncated, nil
}

func parseResumeProfileOutput(raw string) (UserProfile, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return UserProfile{}, &ResumeImportError{Message: "模型没有返回能力画像"}
	}
	if strings.HasPrefix(text, "```") {
		text = extractCodeBlock(text)
	}
	text = strings.TrimSpace(text)

	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		match := resumeJSONObjectPattern.FindString(text)
		if strings.TrimSpace(match) == "" {
			return UserProfile{}, &ResumeImportError{Message: "模型返回格式不是合法 JSON"}
		}
		if err := json.Unmarshal([]byte(match), &payload); err != nil {
			return UserProfile{}, &ResumeImportError{Message: "模型返回 JSON 解析失败"}
		}
	}

	profile := UserProfile{
		Name:                   readAnyString(payload["name"]),
		CurrentTitle:           readAnyString(payload["current_title"]),
		TotalYears:             readAnyNumber(payload["total_years"]),
		PrimaryIndustry:        readAnyString(payload["primary_industry"]),
		Industries:             readAnyStringList(payload["industries"]),
		CoreSkills:             readAnyStringList(payload["core_skills"]),
		Tooling:                readAnyStringList(payload["tooling"]),
		ProgrammingLanguages:   readAnyStringList(payload["programming_languages"]),
		DomainKnowledge:        readAnyStringList(payload["domain_knowledge"]),
		ProjectEvidence:        readAnyStringList(payload["project_evidence"]),
		Achievements:           readAnyStringList(payload["achievements"]),
		Education:              readAnyStringList(payload["education"]),
		Certifications:         readAnyStringList(payload["certifications"]),
		PreferredRoles:         readAnyStringList(payload["preferred_roles"]),
		PreferredIndustries:    readAnyStringList(payload["preferred_industries"]),
		PreferredLocations:     readAnyStringList(payload["preferred_locations"]),
		RemotePreference:       readAnyString(payload["remote_preference"]),
		EmploymentTypes:        readAnyStringList(payload["employment_types"]),
		SalaryExpectation:      readAnyString(payload["salary_expectation"]),
		WorkAuthorization:      readAnyString(payload["work_authorization"]),
		VisaNeeds:              readAnyString(payload["visa_needs"]),
		PreferredCompanyStages: readAnyStringList(payload["preferred_company_stages"]),
		ExcludedCompanies:      readAnyStringList(payload["excluded_companies"]),
		JobSearchPriorities:    readAnyStringList(payload["job_search_priorities"]),
		StrengthSummary:        readAnyString(payload["strength_summary"]),
		PortfolioLinks:         readAnyStringList(payload["portfolio_links"]),
		Notes:                  readAnyString(payload["notes"]),
	}

	return normalizeUserProfile(profile), nil
}

func extractResumeText(filename string, contentType string, fileBytes []byte) (string, error) {
	if len(fileBytes) == 0 {
		return "", errResumeFileEmpty
	}
	if len(fileBytes) > maxResumeFileSizeBytes {
		return "", errResumeFileTooLarge
	}

	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	switch {
	case ext == ".pdf" || strings.Contains(contentType, "application/pdf"):
		return extractPDFText(fileBytes)
	case ext == ".docx" || strings.Contains(contentType, "wordprocessingml.document"):
		return extractDOCXText(fileBytes)
	case ext == ".txt", ext == ".md", ext == ".markdown", ext == ".json", ext == ".yaml", ext == ".yml", ext == ".csv":
		return extractPlainText(fileBytes)
	case strings.HasPrefix(contentType, "text/"):
		return extractPlainText(fileBytes)
	default:
		return "", errResumeFormatNotFound
	}
}

func extractPlainText(fileBytes []byte) (string, error) {
	text := normalizeResumeText(string(fileBytes))
	if text == "" {
		return "", errResumeTextEmpty
	}
	return text, nil
}

func extractPDFText(fileBytes []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", &ResumeImportError{Message: "PDF 读取失败，请确认文件未损坏"}
	}
	plainReader, err := reader.GetPlainText()
	if err != nil {
		return "", &ResumeImportError{Message: "PDF 文本提取失败，请尝试 docx 或 txt"}
	}
	payload, err := io.ReadAll(plainReader)
	if err != nil {
		return "", &ResumeImportError{Message: "PDF 内容读取失败"}
	}
	text := normalizeResumeText(string(payload))
	if text == "" {
		return "", &ResumeImportError{Message: "PDF 中未提取到可用文本，请尝试 docx 或 txt"}
	}
	return text, nil
}

func extractDOCXText(fileBytes []byte) (string, error) {
	archive, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return "", &ResumeImportError{Message: "DOCX 读取失败，请确认文件未损坏"}
	}

	targets := make([]*zip.File, 0, len(archive.File))
	for _, item := range archive.File {
		name := strings.ToLower(item.Name)
		if name == "word/document.xml" || (strings.HasPrefix(name, "word/header") && strings.HasSuffix(name, ".xml")) || (strings.HasPrefix(name, "word/footer") && strings.HasSuffix(name, ".xml")) {
			targets = append(targets, item)
		}
	}
	if len(targets) == 0 {
		return "", &ResumeImportError{Message: "DOCX 中未找到正文内容"}
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name < targets[j].Name
	})

	parts := make([]string, 0, len(targets))
	for _, item := range targets {
		reader, err := item.Open()
		if err != nil {
			continue
		}
		payload, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			continue
		}
		parsed := extractTextFromDocxXML(string(payload))
		if parsed != "" {
			parts = append(parts, parsed)
		}
	}
	text := normalizeResumeText(strings.Join(parts, "\n"))
	if text == "" {
		return "", &ResumeImportError{Message: "DOCX 文本提取失败，请尝试 txt 或 pdf"}
	}
	return text, nil
}

func extractTextFromDocxXML(raw string) string {
	replacer := strings.NewReplacer(
		"</w:p>", "\n",
		"<w:tab/>", "\t",
		"<w:tab />", "\t",
		"<w:br/>", "\n",
		"<w:br />", "\n",
		"<w:cr/>", "\n",
		"<w:cr />", "\n",
	)
	cleaned := replacer.Replace(raw)
	cleaned = xmlTagPattern.ReplaceAllString(cleaned, " ")
	cleaned = html.UnescapeString(cleaned)
	return normalizeResumeText(cleaned)
}

func normalizeResumeText(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = html.UnescapeString(line)
		line = multiWhitespacePattern.ReplaceAllString(line, " ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func truncateByRunes(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes])
}

func readAnyString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
	case int:
		return strings.TrimSpace(strconv.Itoa(typed))
	case int64:
		return strings.TrimSpace(strconv.FormatInt(typed, 10))
	case json.Number:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func readAnyNumber(value any) float64 {
	switch typed := value.(type) {
	case float64:
		if typed < 0 {
			return 0
		}
		return typed
	case int:
		if typed < 0 {
			return 0
		}
		return float64(typed)
	case int64:
		if typed < 0 {
			return 0
		}
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil || parsed < 0 {
			return 0
		}
		return parsed
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil || parsed < 0 {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func readAnyStringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return normalizeStringList(typed, len(typed))
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			candidate := readAnyString(item)
			if candidate == "" {
				continue
			}
			items = append(items, candidate)
		}
		return normalizeStringList(items, len(items))
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		parts := strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n'
		})
		return normalizeStringList(parts, len(parts))
	default:
		return nil
	}
}
