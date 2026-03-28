package prep

import (
	"fmt"
	"sort"
	"strings"
)

type PromptConfig struct {
	Count            int
	RetrievedChunks  []RetrievedChunk
	CandidateProfile string
	JobDescription   string
}

type PromptBuildInput struct {
	Count            int
	RetrievedChunks  []RetrievedChunk
	CandidateProfile string
	JobDescription   string
}

const questionGenerationSystemPrompt = `You are Trace2Offer's interview question architect.

Your mission:
- Generate role-specific interview questions grounded in the provided context.
- Maximize signal quality: each question should reveal candidate depth, judgment, and practical execution.

Hard rules:
1) Use only provided inputs (retrieved context, candidate context, and job description). Do not fabricate facts.
2) Keep each question concise, concrete, and interview-ready.
3) Ensure coverage balance and avoid near-duplicate questions.
4) For each question, produce actionable expected_points that an interviewer can score against.
5) Output must be strict JSON only, no markdown, no prose outside JSON.`

func BuildQuestionGenerationPrompt(config PromptConfig) string {
	sections := BuildQuestionGenerationPromptSections(PromptBuildInput(config))
	return strings.Join([]string{
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

	return QuestionPromptSections{
		System:           questionGenerationSystemPrompt,
		Context:          buildContextSection(input.RetrievedChunks),
		CandidateProfile: buildCandidateProfileSection(input.CandidateProfile),
		JobDescription:   buildJobDescriptionSection(input.JobDescription),
		Task: strings.Join([]string{
			"<task>",
			fmt.Sprintf("- Generate exactly %d interview questions.", count),
			"- Ensure question-set coverage across core competency dimensions in the supplied context.",
			"- Each question must be answerable in 3-8 minutes of spoken response.",
			"- Prioritize high-signal questions that expose trade-offs, decision quality, and execution details.",
			"</task>",
		}, "\n"),
		Requirements: strings.Join([]string{
			"<requirements>",
			"- Mix conceptual depth and practical execution scenarios.",
			"- Align with the company/role context whenever JD or lead signals are present.",
			"- Avoid generic textbook wording; favor concrete constraints and trade-offs.",
			"- expected_points should be 3-6 concise scoring bullets per question.",
			"- context_sources must reference titles from provided retrieved context when applicable.",
			"- If context is weak, keep context_sources as an empty array rather than inventing sources.",
			"</requirements>",
		}, "\n"),
		OutputFormat: strings.Join([]string{
			"<output_format>",
			"Return JSON only with this schema:",
			"{",
			"  \"questions\": [",
			"    {",
			"      \"id\": 1,",
			"      \"type\": \"technical|system_design|behavioral|coding\",",
			"      \"content\": \"...\",",
			"      \"expected_points\": [\"...\", \"...\", \"...\"],",
			"      \"context_sources\": [\"source title\", \"source title\"]",
			"    }",
			"  ]",
			"}",
			"Validation:",
			"- questions.length must be exactly the required count.",
			"- id must start from 1 and be unique.",
			"- No markdown fences.",
			"</output_format>",
		}, "\n"),
	}
}

func buildContextSection(chunks []RetrievedChunk) string {
	if len(chunks) == 0 {
		return strings.Join([]string{
			"<retrieved_context>",
			"- (no retrieved chunks)",
			"</retrieved_context>",
		}, "\n")
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
		for idx, item := range items {
			itemLines = append(itemLines, strings.Join([]string{
				fmt.Sprintf("<chunk rank=\"%d\" score=\"%.3f\">", idx+1, item.Score),
				strings.TrimSpace(item.Content),
				"</chunk>",
			}, "\n"))
		}
		parts = append(parts, strings.Join([]string{
			fmt.Sprintf("<source title=\"%s\">", source),
			strings.Join(itemLines, "\n"),
			"</source>",
		}, "\n"))
	}
	return strings.Join([]string{
		"<retrieved_context>",
		strings.Join(parts, "\n\n"),
		"</retrieved_context>",
	}, "\n")
}

func buildCandidateProfileSection(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return strings.Join([]string{
			"<candidate_context>",
			"- (not provided)",
			"</candidate_context>",
		}, "\n")
	}
	return strings.Join([]string{
		"<candidate_context>",
		trimmed,
		"</candidate_context>",
	}, "\n")
}

func buildJobDescriptionSection(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return strings.Join([]string{
			"<job_description>",
			"- (not provided)",
			"</job_description>",
		}, "\n")
	}
	return strings.Join([]string{
		"<job_description>",
		trimmed,
		"</job_description>",
	}, "\n")
}
