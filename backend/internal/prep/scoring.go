package prep

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultQuestionScoringTopK = 4
)

const questionScoringSystemPrompt = `You are a strict interview evaluator for Trace2Offer.

Your task:
- Score one interview answer against the question and expected points.
- Use retrieved context only as supporting evidence, not as mandatory ground truth.

Hard rules:
1) Score must be a float in [0, 10].
2) Be concise and specific.
3) Return strict JSON only with schema:
{
  "score": 0,
  "answered": true,
  "summary": "...",
  "strengths": ["..."],
  "improvements": ["..."],
  "weak_points": ["..."]
}`

type retrievalSearcher interface {
	Search(query string, config SearchConfig) (*SearchResult, error)
}

type ScoringEngine struct {
	retrieval retrievalSearcher
	model     QuestionModel
	topK      int
	now       func() time.Time
}

func NewScoringEngine(retrieval retrievalSearcher, model QuestionModel) *ScoringEngine {
	return &ScoringEngine{
		retrieval: retrieval,
		model:     model,
		topK:      defaultQuestionScoringTopK,
		now:       time.Now,
	}
}

func (e *ScoringEngine) EvaluateSession(ctx context.Context, session Session) (*Evaluation, error) {
	if e == nil {
		return nil, &ValidationError{Field: "scoring", Message: "scoring engine is required"}
	}
	if e.model == nil {
		return nil, ErrScoringEngineUnavailable
	}

	answerByQuestionID := make(map[int]Answer, len(session.Answers))
	for _, answer := range session.Answers {
		if answer.QuestionID > 0 {
			answerByQuestionID[answer.QuestionID] = answer
		}
	}

	scores := make([]QuestionScore, 0, len(session.Questions))
	for index, question := range session.Questions {
		questionID := resolveQuestionID(question, index)
		answerText := strings.TrimSpace(resolveAnswer(answerByQuestionID, questionID, question.QuestionID))
		answered := answerText != ""

		retrievalQuery := buildQuestionRetrievalQuery(question, answerText)
		retrievedChunks, retrievalErr := e.retrieveQuestionChunks(retrievalQuery, session)
		if retrievalErr != nil {
			return nil, fmt.Errorf("question %d retrieval failed: %w", questionID, retrievalErr)
		}

		baseTrace := map[string]interface{}{
			"retrieval_query": retrievalQuery,
			"retrieved_count": len(retrievedChunks),
			"scored_at":       e.now().UTC().Format(time.RFC3339),
		}
		if !answered {
			scores = append(scores, QuestionScore{
				QuestionID: questionID,
				Score:      0,
				Answered:   false,
				Summary:    "该题未作答。",
				Strengths:  []string{"题目目标已识别"},
				Improvements: []string{
					"请补充完整答案后重新评分",
				},
				WeakPoints: []string{"当前为空答案"},
				Sources:    buildQuestionScoreSources(retrievedChunks),
				Trace:      baseTrace,
			})
			continue
		}

		userPrompt := buildQuestionScoringPrompt(question, answerText, retrievedChunks)
		rawOutput, err := e.model.Generate(ctx, questionScoringSystemPrompt, userPrompt)
		if err != nil {
			return nil, fmt.Errorf("question %d scoring model request failed: %w", questionID, err)
		}
		parsed, err := parseQuestionScoreOutput(rawOutput, questionID, true)
		if err != nil {
			return nil, fmt.Errorf("question %d scoring output invalid: %w", questionID, err)
		}
		parsed.Sources = buildQuestionScoreSources(retrievedChunks)
		parsed.Trace = baseTrace
		scores = append(scores, parsed)
	}

	overall := buildOverallEvaluation(scores)
	return &Evaluation{
		Status:       EvaluationStatusCompleted,
		Scores:       scores,
		Overall:      overall,
		OverallScore: overall.AverageScore,
		Summary:      overall.Summary,
	}, nil
}

func (e *ScoringEngine) retrieveQuestionChunks(query string, session Session) ([]RetrievedChunk, error) {
	if e == nil || e.retrieval == nil {
		return []RetrievedChunk{}, nil
	}
	result, err := e.retrieval.Search(query, SearchConfig{
		LeadID:          strings.TrimSpace(session.LeadID),
		CompanySlug:     normalizeCompanySlug(session.Company),
		Query:           query,
		TopK:            e.topK,
		IncludeTrace:    false,
		IncludeResume:   session.Config.IncludeResume,
		IncludeLeadDocs: session.Config.IncludeLeadDocs,
	})
	if err != nil {
		return []RetrievedChunk{}, err
	}
	if result == nil {
		return []RetrievedChunk{}, nil
	}
	return append([]RetrievedChunk{}, result.RetrievedChunks...), nil
}

func resolveQuestionID(question Question, index int) int {
	if question.ID > 0 {
		return question.ID
	}
	if question.QuestionID > 0 {
		return question.QuestionID
	}
	return index + 1
}

func resolveAnswer(answerByQuestionID map[int]Answer, primaryID int, secondaryID int) string {
	if item, ok := answerByQuestionID[primaryID]; ok {
		return item.Answer
	}
	if secondaryID > 0 {
		if item, ok := answerByQuestionID[secondaryID]; ok {
			return item.Answer
		}
	}
	return ""
}

func buildQuestionRetrievalQuery(question Question, answer string) string {
	parts := []string{
		strings.TrimSpace(question.Content),
		strings.Join(normalizeStringList(question.ExpectedPoints), " "),
	}
	trimmedAnswer := strings.TrimSpace(answer)
	if trimmedAnswer != "" {
		parts = append(parts, limitPromptInput(trimmedAnswer, 140))
	}
	return strings.Join(normalizeStringList(parts), " ")
}

func buildQuestionScoringPrompt(question Question, answer string, chunks []RetrievedChunk) string {
	contextSection := buildContextSection(chunks)
	expected := normalizeStringList(question.ExpectedPoints)
	if len(expected) == 0 {
		expected = []string{"回答结构清晰", "结合真实实践", "说明取舍与结果"}
	}
	return strings.Join([]string{
		"<question>",
		strings.TrimSpace(question.Content),
		"</question>",
		"",
		"<expected_points>",
		strings.Join(expected, "\n"),
		"</expected_points>",
		"",
		"<candidate_answer>",
		strings.TrimSpace(answer),
		"</candidate_answer>",
		"",
		contextSection,
	}, "\n")
}

func buildQuestionScoreSources(chunks []RetrievedChunk) []QuestionScoreSource {
	if len(chunks) == 0 {
		return []QuestionScoreSource{}
	}
	out := make([]QuestionScoreSource, 0, len(chunks))
	seen := map[string]struct{}{}
	for _, chunk := range chunks {
		title := strings.TrimSpace(chunk.Source.DocumentTitle)
		if title == "" {
			continue
		}
		if _, exists := seen[title]; exists {
			continue
		}
		seen[title] = struct{}{}
		out = append(out, QuestionScoreSource{
			Title: title,
			Score: clampScore(chunk.Score * 10),
		})
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func buildOverallEvaluation(scores []QuestionScore) OverallEvaluation {
	total := len(scores)
	if total == 0 {
		return OverallEvaluation{
			AverageScore:   0,
			AnsweredCount:  0,
			TotalQuestions: 0,
			Strengths:      []string{},
			WeakPoints:     []string{},
			Summary:        "暂无可评分题目。",
		}
	}

	sum := 0.0
	answeredCount := 0
	strengthsPool := make([]string, 0, total*2)
	weakPool := make([]string, 0, total*2)
	for _, item := range scores {
		sum += item.Score
		if item.Answered {
			answeredCount++
		}
		strengthsPool = append(strengthsPool, item.Strengths...)
		weakPool = append(weakPool, item.WeakPoints...)
		if len(item.WeakPoints) == 0 {
			weakPool = append(weakPool, item.Improvements...)
		}
	}

	avg := clampScore(sum / float64(total))
	summary := fmt.Sprintf("共 %d 题，已作答 %d 题，平均分 %.1f/10。", total, answeredCount, avg)
	if answeredCount < total {
		summary += " 未作答题按 0 分计入。"
	}

	return OverallEvaluation{
		AverageScore:   avg,
		AnsweredCount:  answeredCount,
		TotalQuestions: total,
		Strengths:      takeFirst(normalizeStringList(strengthsPool), 6),
		WeakPoints:     takeFirst(normalizeStringList(weakPool), 6),
		Summary:        summary,
	}
}

func takeFirst(values []string, max int) []string {
	if max <= 0 || len(values) == 0 {
		return []string{}
	}
	if len(values) <= max {
		return append([]string{}, values...)
	}
	return append([]string{}, values[:max]...)
}
