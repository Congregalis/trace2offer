package agent

import (
	"fmt"
	"strings"
	"time"
)

// UserProfile is the structured capability profile injected into agent prompt.
type UserProfile struct {
	Name                   string   `json:"name"`
	CurrentTitle           string   `json:"current_title"`
	TotalYears             float64  `json:"total_years"`
	PrimaryIndustry        string   `json:"primary_industry"`
	Industries             []string `json:"industries"`
	CoreSkills             []string `json:"core_skills"`
	Tooling                []string `json:"tooling"`
	ProgrammingLanguages   []string `json:"programming_languages"`
	DomainKnowledge        []string `json:"domain_knowledge"`
	ProjectEvidence        []string `json:"project_evidence"`
	Achievements           []string `json:"achievements"`
	Education              []string `json:"education"`
	Certifications         []string `json:"certifications"`
	PreferredRoles         []string `json:"preferred_roles"`
	PreferredIndustries    []string `json:"preferred_industries"`
	PreferredLocations     []string `json:"preferred_locations"`
	RemotePreference       string   `json:"remote_preference"`
	EmploymentTypes        []string `json:"employment_types"`
	SalaryExpectation      string   `json:"salary_expectation"`
	WorkAuthorization      string   `json:"work_authorization"`
	VisaNeeds              string   `json:"visa_needs"`
	PreferredCompanyStages []string `json:"preferred_company_stages"`
	ExcludedCompanies      []string `json:"excluded_companies"`
	JobSearchPriorities    []string `json:"job_search_priorities"`
	StrengthSummary        string   `json:"strength_summary"`
	PortfolioLinks         []string `json:"portfolio_links"`
	Notes                  string   `json:"notes"`
	UpdatedAt              string   `json:"updated_at,omitempty"`
}

func normalizeUserProfile(input UserProfile) UserProfile {
	profile := input

	profile.Name = strings.TrimSpace(profile.Name)
	profile.CurrentTitle = strings.TrimSpace(profile.CurrentTitle)
	profile.PrimaryIndustry = strings.TrimSpace(profile.PrimaryIndustry)
	profile.RemotePreference = strings.TrimSpace(profile.RemotePreference)
	profile.SalaryExpectation = strings.TrimSpace(profile.SalaryExpectation)
	profile.WorkAuthorization = strings.TrimSpace(profile.WorkAuthorization)
	profile.VisaNeeds = strings.TrimSpace(profile.VisaNeeds)
	profile.StrengthSummary = strings.TrimSpace(profile.StrengthSummary)
	profile.Notes = strings.TrimSpace(profile.Notes)

	if profile.TotalYears < 0 {
		profile.TotalYears = 0
	}

	profile.Industries = normalizeStringList(profile.Industries, 32)
	profile.CoreSkills = normalizeStringList(profile.CoreSkills, 64)
	profile.Tooling = normalizeStringList(profile.Tooling, 64)
	profile.ProgrammingLanguages = normalizeStringList(profile.ProgrammingLanguages, 32)
	profile.DomainKnowledge = normalizeStringList(profile.DomainKnowledge, 48)
	profile.ProjectEvidence = normalizeStringList(profile.ProjectEvidence, 64)
	profile.Achievements = normalizeStringList(profile.Achievements, 48)
	profile.Education = normalizeStringList(profile.Education, 24)
	profile.Certifications = normalizeStringList(profile.Certifications, 24)
	profile.PreferredRoles = normalizeStringList(profile.PreferredRoles, 24)
	profile.PreferredIndustries = normalizeStringList(profile.PreferredIndustries, 24)
	profile.PreferredLocations = normalizeStringList(profile.PreferredLocations, 24)
	profile.EmploymentTypes = normalizeStringList(profile.EmploymentTypes, 16)
	profile.PreferredCompanyStages = normalizeStringList(profile.PreferredCompanyStages, 16)
	profile.ExcludedCompanies = normalizeStringList(profile.ExcludedCompanies, 64)
	profile.JobSearchPriorities = normalizeStringList(profile.JobSearchPriorities, 32)
	profile.PortfolioLinks = normalizeStringList(profile.PortfolioLinks, 16)

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
	next.PrimaryIndustry = chooseNonEmpty(imported.PrimaryIndustry, next.PrimaryIndustry)
	next.Industries = mergeStringList(imported.Industries, next.Industries, 32)
	next.CoreSkills = mergeStringList(imported.CoreSkills, next.CoreSkills, 64)
	next.Tooling = mergeStringList(imported.Tooling, next.Tooling, 64)
	next.ProgrammingLanguages = mergeStringList(imported.ProgrammingLanguages, next.ProgrammingLanguages, 32)
	next.DomainKnowledge = mergeStringList(imported.DomainKnowledge, next.DomainKnowledge, 48)
	next.ProjectEvidence = mergeStringList(imported.ProjectEvidence, next.ProjectEvidence, 64)
	next.Achievements = mergeStringList(imported.Achievements, next.Achievements, 48)
	next.Education = mergeStringList(imported.Education, next.Education, 24)
	next.Certifications = mergeStringList(imported.Certifications, next.Certifications, 24)
	next.PreferredRoles = mergeStringList(imported.PreferredRoles, next.PreferredRoles, 24)
	next.PreferredIndustries = mergeStringList(imported.PreferredIndustries, next.PreferredIndustries, 24)
	next.PreferredLocations = mergeStringList(imported.PreferredLocations, next.PreferredLocations, 24)
	next.RemotePreference = chooseNonEmpty(imported.RemotePreference, next.RemotePreference)
	next.EmploymentTypes = mergeStringList(imported.EmploymentTypes, next.EmploymentTypes, 16)
	next.SalaryExpectation = chooseNonEmpty(imported.SalaryExpectation, next.SalaryExpectation)
	next.WorkAuthorization = chooseNonEmpty(imported.WorkAuthorization, next.WorkAuthorization)
	next.VisaNeeds = chooseNonEmpty(imported.VisaNeeds, next.VisaNeeds)
	next.PreferredCompanyStages = mergeStringList(imported.PreferredCompanyStages, next.PreferredCompanyStages, 16)
	next.ExcludedCompanies = mergeStringList(imported.ExcludedCompanies, next.ExcludedCompanies, 64)
	next.JobSearchPriorities = mergeStringList(imported.JobSearchPriorities, next.JobSearchPriorities, 32)
	next.StrengthSummary = chooseNonEmpty(imported.StrengthSummary, next.StrengthSummary)
	next.PortfolioLinks = mergeStringList(imported.PortfolioLinks, next.PortfolioLinks, 16)
	next.Notes = mergeNotes(imported.Notes, next.Notes)
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
	appendProfileLine(&builder, "primary_industry", profile.PrimaryIndustry)
	appendProfileList(&builder, "industries", profile.Industries)
	appendProfileList(&builder, "core_skills", profile.CoreSkills)
	appendProfileList(&builder, "tooling", profile.Tooling)
	appendProfileList(&builder, "programming_languages", profile.ProgrammingLanguages)
	appendProfileList(&builder, "domain_knowledge", profile.DomainKnowledge)
	appendProfileList(&builder, "project_evidence", profile.ProjectEvidence)
	appendProfileList(&builder, "achievements", profile.Achievements)
	appendProfileList(&builder, "education", profile.Education)
	appendProfileList(&builder, "certifications", profile.Certifications)
	appendProfileList(&builder, "preferred_roles", profile.PreferredRoles)
	appendProfileList(&builder, "preferred_industries", profile.PreferredIndustries)
	appendProfileList(&builder, "preferred_locations", profile.PreferredLocations)
	appendProfileLine(&builder, "remote_preference", profile.RemotePreference)
	appendProfileList(&builder, "employment_types", profile.EmploymentTypes)
	appendProfileLine(&builder, "salary_expectation", profile.SalaryExpectation)
	appendProfileLine(&builder, "work_authorization", profile.WorkAuthorization)
	appendProfileLine(&builder, "visa_needs", profile.VisaNeeds)
	appendProfileList(&builder, "preferred_company_stages", profile.PreferredCompanyStages)
	appendProfileList(&builder, "excluded_companies", profile.ExcludedCompanies)
	appendProfileList(&builder, "job_search_priorities", profile.JobSearchPriorities)
	appendProfileLine(&builder, "strength_summary", profile.StrengthSummary)
	appendProfileList(&builder, "portfolio_links", profile.PortfolioLinks)
	appendProfileLine(&builder, "notes", profile.Notes)

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

func mergeNotes(imported string, existing string) string {
	imported = strings.TrimSpace(imported)
	existing = strings.TrimSpace(existing)
	if imported == "" {
		return existing
	}
	if existing == "" || existing == imported {
		return imported
	}
	return imported + "\n\n" + existing
}
