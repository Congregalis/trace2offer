package prep

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseQuestionScoreOutput(raw string, questionID int, answered bool) (QuestionScore, error) {
	candidate := extractJSONCandidate(raw)
	if strings.TrimSpace(candidate) == "" {
		return QuestionScore{}, fmt.Errorf("model output does not contain valid json object")
	}

	var parsed struct {
		Score        *float64 `json:"score"`
		Answered     *bool    `json:"answered"`
		Summary      string   `json:"summary"`
		Strengths    []string `json:"strengths"`
		Improvements []string `json:"improvements"`
		WeakPoints   []string `json:"weak_points"`
	}
	if err := json.Unmarshal([]byte(candidate), &parsed); err != nil {
		return QuestionScore{}, fmt.Errorf("decode scoring output: %w", err)
	}

	if parsed.Score == nil {
		return QuestionScore{}, fmt.Errorf("missing required field: score")
	}
	score := clampScore(*parsed.Score)
	finalAnswered := answered
	if parsed.Answered != nil {
		finalAnswered = *parsed.Answered
	}
	summary := strings.TrimSpace(parsed.Summary)
	if summary == "" {
		return QuestionScore{}, fmt.Errorf("missing required field: summary")
	}
	strengths := normalizeStringList(parsed.Strengths)
	if len(strengths) == 0 {
		strengths = []string{"无"}
	}
	improvements := normalizeStringList(parsed.Improvements)
	if len(improvements) == 0 {
		improvements = []string{"无"}
	}
	weakPoints := normalizeStringList(parsed.WeakPoints)
	if len(weakPoints) == 0 {
		weakPoints = []string{"无"}
	}

	return QuestionScore{
		QuestionID:   questionID,
		Score:        score,
		Answered:     finalAnswered,
		Summary:      summary,
		Strengths:    strengths,
		Improvements: improvements,
		WeakPoints:   weakPoints,
		Sources:      []QuestionScoreSource{},
	}, nil
}

func extractJSONCandidate(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "```") {
		lines := strings.Split(trimmed, "\n")
		if len(lines) >= 3 {
			trimmed = strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return trimmed
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return strings.TrimSpace(trimmed[start : end+1])
	}
	return ""
}

func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 10 {
		return 10
	}
	return roundScore(score)
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, item := range values {
		normalized := strings.Join(strings.Fields(strings.TrimSpace(item)), " ")
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
