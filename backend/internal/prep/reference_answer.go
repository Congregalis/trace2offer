package prep

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const referenceAnswerSystemPrompt = `You are a senior interview coach.

Task:
- Generate one high-quality reference answer for the given interview question.
- Keep it practical, structured, and directly usable for candidate review.

Hard rules:
1) Use provided context only; do not fabricate facts.
2) Answer should be concise but specific (around 150-300 Chinese characters when possible).
3) Return strict JSON only:
{
  "reference_answer": "..."
}`

type ReferenceAnswerInput struct {
	Session  Session
	Question Question
	Answer   string
}

type ReferenceAnswerGenerator struct {
	retrieval retrievalSearcher
	model     QuestionModel
	now       func() time.Time
}

func NewReferenceAnswerGenerator(retrieval retrievalSearcher, model QuestionModel) *ReferenceAnswerGenerator {
	return &ReferenceAnswerGenerator{
		retrieval: retrieval,
		model:     model,
		now:       time.Now,
	}
}

func (g *ReferenceAnswerGenerator) Generate(ctx context.Context, input ReferenceAnswerInput) (ReferenceAnswer, error) {
	if g == nil || g.model == nil {
		return ReferenceAnswer{}, ErrReferenceAnswerUnavailable
	}

	questionID := resolveQuestionID(input.Question, 0)
	retrievalQuery := buildQuestionRetrievalQuery(input.Question, input.Answer)
	retrievedChunks, err := g.retrieveChunks(input.Session, retrievalQuery)
	if err != nil {
		return ReferenceAnswer{}, err
	}

	prompt := buildReferenceAnswerPrompt(input.Question, input.Answer, retrievedChunks)
	output, err := g.model.Generate(ctx, referenceAnswerSystemPrompt, prompt)
	if err != nil {
		return ReferenceAnswer{}, err
	}
	referenceText, err := parseReferenceAnswerText(output)
	if err != nil {
		return ReferenceAnswer{}, err
	}

	return ReferenceAnswer{
		QuestionID:      questionID,
		ReferenceAnswer: referenceText,
		Sources:         buildQuestionScoreSources(retrievedChunks),
		GeneratedAt:     g.now().UTC().Format(time.RFC3339),
	}, nil
}

func (g *ReferenceAnswerGenerator) retrieveChunks(session Session, query string) ([]RetrievedChunk, error) {
	if g == nil || g.retrieval == nil {
		return []RetrievedChunk{}, nil
	}
	result, err := g.retrieval.Search(query, SearchConfig{
		LeadID:          strings.TrimSpace(session.LeadID),
		CompanySlug:     normalizeCompanySlug(session.Company),
		Query:           query,
		TopK:            defaultQuestionScoringTopK,
		IncludeTrace:    false,
		IncludeResume:   session.Config.IncludeResume,
		IncludeLeadDocs: session.Config.IncludeLeadDocs,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []RetrievedChunk{}, nil
	}
	return append([]RetrievedChunk{}, result.RetrievedChunks...), nil
}

func buildReferenceAnswerPrompt(question Question, answer string, chunks []RetrievedChunk) string {
	expectedPoints := normalizeStringList(question.ExpectedPoints)
	if len(expectedPoints) == 0 {
		expectedPoints = []string{"结构清晰", "关键取舍", "落地实践"}
	}
	return strings.Join([]string{
		"<reference_answer_task>",
		"请基于问题与上下文生成单题参考答案。",
		"</reference_answer_task>",
		"",
		"<question>",
		strings.TrimSpace(question.Content),
		"</question>",
		"",
		"<expected_points>",
		strings.Join(expectedPoints, "\n"),
		"</expected_points>",
		"",
		"<candidate_answer>",
		strings.TrimSpace(answer),
		"</candidate_answer>",
		"",
		buildContextSection(chunks),
	}, "\n")
}

func parseReferenceAnswerText(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("reference answer output is empty")
	}
	candidate := extractJSONCandidate(trimmed)
	if candidate != "" {
		var parsed struct {
			ReferenceAnswer string `json:"reference_answer"`
			Answer          string `json:"answer"`
		}
		if err := json.Unmarshal([]byte(candidate), &parsed); err != nil {
			return "", fmt.Errorf("decode reference answer output: %w", err)
		}
		value := strings.TrimSpace(parsed.ReferenceAnswer)
		if value == "" {
			value = strings.TrimSpace(parsed.Answer)
		}
		if value == "" {
			return "", fmt.Errorf("reference_answer is required")
		}
		return value, nil
	}

	return trimmed, nil
}

func findQuestionAndAnswer(session Session, questionID int) (Question, string, bool) {
	if questionID <= 0 {
		return Question{}, "", false
	}
	answerByQuestionID := map[int]string{}
	for _, answer := range session.Answers {
		if answer.QuestionID > 0 {
			answerByQuestionID[answer.QuestionID] = answer.Answer
		}
	}

	for index, question := range session.Questions {
		candidateID := resolveQuestionID(question, index)
		if candidateID != questionID {
			continue
		}
		answerText := strings.TrimSpace(answerByQuestionID[candidateID])
		if answerText == "" && question.QuestionID > 0 {
			answerText = strings.TrimSpace(answerByQuestionID[question.QuestionID])
		}
		return question, answerText, true
	}
	return Question{}, "", false
}

func referenceAnswerMapKey(questionID int) string {
	return strconv.Itoa(questionID)
}
