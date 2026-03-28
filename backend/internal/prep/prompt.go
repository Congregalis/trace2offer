package prep

import (
	"fmt"
	"sort"
	"strings"
)

type PromptConfig struct {
	Count            int
	TopicKeys        []string
	RetrievedChunks  []RetrievedChunk
	CandidateProfile string
	JobDescription   string
}

type PromptBuildInput struct {
	Count            int
	TopicKeys        []string
	RetrievedChunks  []RetrievedChunk
	CandidateProfile string
	JobDescription   string
}

func BuildQuestionGenerationPrompt(config PromptConfig) string {
	sections := BuildQuestionGenerationPromptSections(PromptBuildInput(config))
	return strings.Join([]string{
		sections.System,
		"",
		sections.Context,
		"",
		sections.CandidateProfile,
		"",
		sections.JobDescription,
		"",
		sections.Task,
		"",
		sections.Requirements,
		"",
		sections.OutputFormat,
	}, "\n")
}

type QuestionPromptSections struct {
	System           string
	Context          string
	CandidateProfile string
	JobDescription   string
	Task             string
	Requirements     string
	OutputFormat     string
}

func BuildQuestionGenerationPromptSections(input PromptBuildInput) QuestionPromptSections {
	count := input.Count
	if count <= 0 {
		count = defaultQuestionCount
	}
	topicKeys := normalizeTopicKeysForPrompt(input.TopicKeys)

	return QuestionPromptSections{
		System:           "System: You are an expert technical interviewer preparing questions for a candidate.",
		Context:          "Context:\n" + buildContextSection(input.RetrievedChunks),
		CandidateProfile: "Candidate Profile:\n" + buildCandidateProfileSection(input.CandidateProfile),
		JobDescription:   "Job Description:\n" + buildJobDescriptionSection(input.JobDescription),
		Task:             fmt.Sprintf("Task:\nGenerate %d interview questions covering: %s", count, strings.Join(topicKeys, ", ")),
		Requirements: strings.Join([]string{
			"Requirements:",
			"- Mix of conceptual and practical questions",
			"- Questions should be specific to the company/role",
			"- Include expected answer points",
		}, "\n"),
		OutputFormat: strings.Join([]string{
			"Output Format:",
			"```json",
			"{",
			"  \"questions\": [",
			"    {",
			"      \"id\": 1,",
			"      \"type\": \"technical\",",
			"      \"content\": \"...\",",
			"      \"expected_points\": [\"...\"],",
			"      \"context_sources\": [\"...\"]",
			"    }",
			"  ]",
			"}",
			"```",
		}, "\n"),
	}
}

func normalizeTopicKeysForPrompt(topicKeys []string) []string {
	normalized := make([]string, 0, len(topicKeys))
	seen := map[string]struct{}{}
	for _, item := range topicKeys {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	if len(normalized) == 0 {
		return []string{"general"}
	}
	return normalized
}

func buildContextSection(chunks []RetrievedChunk) string {
	if len(chunks) == 0 {
		return "- (no retrieved chunks)"
	}
	grouped := map[string][]RetrievedChunk{}
	for _, chunk := range chunks {
		source := strings.TrimSpace(chunk.Source.DocumentTitle)
		if source == "" {
			source = "unknown-source"
		}
		grouped[source] = append(grouped[source], chunk)
	}
	sources := make([]string, 0, len(grouped))
	for source := range grouped {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	parts := make([]string, 0, len(sources))
	for _, source := range sources {
		items := grouped[source]
		itemLines := make([]string, 0, len(items))
		for _, item := range items {
			itemLines = append(itemLines, fmt.Sprintf("- [score=%.3f] %s", item.Score, strings.TrimSpace(item.Content)))
		}
		parts = append(parts, fmt.Sprintf("[%s]\n%s", source, strings.Join(itemLines, "\n")))
	}
	return strings.Join(parts, "\n\n")
}

func buildCandidateProfileSection(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "- (not provided)"
	}
	return trimmed
}

func buildJobDescriptionSection(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "- (not provided)"
	}
	return trimmed
}
