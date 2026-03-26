package agent

import (
	"fmt"
	"strings"
	"time"
)

// UserProfile is the structured capability profile injected into agent prompt.
type UserProfile struct {
	Name                 string   `json:"name"`
	CurrentTitle         string   `json:"current_title"`
	TotalYears           float64  `json:"total_years"`
	CoreSkills           []string `json:"core_skills"`
	ProgrammingLanguages []string `json:"programming_languages"`
	ProjectEvidence      []string `json:"project_evidence"`
	PreferredRoles       []string `json:"preferred_roles"`
	PreferredLocations   []string `json:"preferred_locations"`
	JobSearchPriorities  []string `json:"job_search_priorities"`
	StrengthSummary      string   `json:"strength_summary"`
	UpdatedAt            string   `json:"updated_at,omitempty"`
}

func normalizeUserProfile(input UserProfile) UserProfile {
	profile := input

	profile.Name = strings.TrimSpace(profile.Name)
	profile.CurrentTitle = strings.TrimSpace(profile.CurrentTitle)
	profile.StrengthSummary = strings.TrimSpace(profile.StrengthSummary)

	if profile.TotalYears < 0 {
		profile.TotalYears = 0
	}

	profile.CoreSkills = normalizeStringList(profile.CoreSkills, 64)
	profile.ProgrammingLanguages = normalizeStringList(profile.ProgrammingLanguages, 32)
	profile.ProjectEvidence = normalizeStringList(profile.ProjectEvidence, 64)
	profile.PreferredRoles = normalizeStringList(profile.PreferredRoles, 24)
	profile.PreferredLocations = normalizeStringList(profile.PreferredLocations, 24)
	profile.JobSearchPriorities = normalizeStringList(profile.JobSearchPriorities, 32)

	if strings.TrimSpace(profile.UpdatedAt) == "" {
		profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return profile
}

func mergeImportedUserProfile(base UserProfile, imported UserProfile) UserProfile {
	base = normalizeUserProfile(base)
	imported = normalizeUserProfile(imported)

	next := base
	next.Name = chooseNonEmpty(imported.Name, next.Name)
	next.CurrentTitle = chooseNonEmpty(imported.CurrentTitle, next.CurrentTitle)
	if imported.TotalYears > 0 {
		next.TotalYears = imported.TotalYears
	}
	next.CoreSkills = mergeStringList(imported.CoreSkills, next.CoreSkills, 64)
	next.ProgrammingLanguages = mergeStringList(imported.ProgrammingLanguages, next.ProgrammingLanguages, 32)
	next.ProjectEvidence = mergeStringList(imported.ProjectEvidence, next.ProjectEvidence, 64)
	next.PreferredRoles = mergeStringList(imported.PreferredRoles, next.PreferredRoles, 24)
	next.PreferredLocations = mergeStringList(imported.PreferredLocations, next.PreferredLocations, 24)
	next.JobSearchPriorities = mergeStringList(imported.JobSearchPriorities, next.JobSearchPriorities, 32)
	next.StrengthSummary = chooseNonEmpty(imported.StrengthSummary, next.StrengthSummary)
	next.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return normalizeUserProfile(next)
}

func formatUserProfilePrompt(profile UserProfile) string {
	profile = normalizeUserProfile(profile)

	builder := strings.Builder{}
	builder.WriteString("[USER]")

	appendProfileLine(&builder, "name", profile.Name)
	appendProfileLine(&builder, "current_title", profile.CurrentTitle)
	if profile.TotalYears > 0 {
		appendProfileLine(&builder, "total_years", fmt.Sprintf("%.1f", profile.TotalYears))
	}
	appendProfileList(&builder, "core_skills", profile.CoreSkills)
	appendProfileList(&builder, "programming_languages", profile.ProgrammingLanguages)
	appendProfileList(&builder, "project_evidence", profile.ProjectEvidence)
	appendProfileList(&builder, "preferred_roles", profile.PreferredRoles)
	appendProfileList(&builder, "preferred_locations", profile.PreferredLocations)
	appendProfileList(&builder, "job_search_priorities", profile.JobSearchPriorities)
	appendProfileLine(&builder, "strength_summary", profile.StrengthSummary)

	if strings.TrimSpace(builder.String()) == "[USER]" {
		return "[USER]\\nprofile_status: 用户暂未维护能力画像"
	}
	return builder.String()
}

func appendProfileLine(builder *strings.Builder, key string, value string) {
	if builder == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	builder.WriteString("\n")
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(trimmed)
}

func appendProfileList(builder *strings.Builder, key string, values []string) {
	if builder == nil {
		return
	}
	values = normalizeStringList(values, len(values))
	if len(values) == 0 {
		return
	}
	builder.WriteString("\n")
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(strings.Join(values, ", "))
}

func chooseNonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func normalizeStringList(input []string, limit int) []string {
	if len(input) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = len(input)
	}
	result := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
		if len(result) >= limit {
			break
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func mergeStringList(primary []string, fallback []string, limit int) []string {
	if limit <= 0 {
		limit = len(primary) + len(fallback)
	}
	merged := make([]string, 0, len(primary)+len(fallback))
	seen := map[string]struct{}{}
	appendIfNew := func(list []string) {
		for _, item := range list {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, trimmed)
			if len(merged) >= limit {
				return
			}
		}
	}

	appendIfNew(primary)
	if len(merged) < limit {
		appendIfNew(fallback)
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
