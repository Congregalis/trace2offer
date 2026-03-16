package agent

import (
	"strings"
	"testing"
)

func TestMergeImportedUserProfile(t *testing.T) {
	t.Parallel()

	base := UserProfile{
		Name:               "Alice",
		CoreSkills:         []string{"Go", "Kubernetes"},
		PreferredLocations: []string{"Remote"},
		Notes:              "manual",
	}
	imported := UserProfile{
		CurrentTitle:       "Staff Engineer",
		CoreSkills:         []string{"Distributed Systems", "Go"},
		PreferredLocations: []string{"Taipei", "Remote"},
		Notes:              "resume",
	}

	merged := mergeImportedUserProfile(base, imported)
	if merged.Name != "Alice" {
		t.Fatalf("expected keep existing name, got %q", merged.Name)
	}
	if merged.CurrentTitle != "Staff Engineer" {
		t.Fatalf("expected imported title, got %q", merged.CurrentTitle)
	}
	if len(merged.CoreSkills) < 3 {
		t.Fatalf("expected merged skills, got %#v", merged.CoreSkills)
	}
	if !strings.Contains(merged.Notes, "resume") {
		t.Fatalf("expected imported notes first, got %q", merged.Notes)
	}
}

func TestFormatUserProfilePrompt(t *testing.T) {
	t.Parallel()

	prompt := formatUserProfilePrompt(UserProfile{
		Name:                "Bob",
		TotalYears:          7,
		CoreSkills:          []string{"Go", "Redis"},
		ProjectEvidence:     []string{"主导流量系统重构"},
		PreferredRoles:      []string{"Principal Backend Engineer"},
		StrengthSummary:     "复杂系统拆解",
		PortfolioLinks:      []string{"https://github.com/bob"},
		JobSearchPriorities: []string{"成长", "影响力"},
	})

	if !strings.Contains(prompt, "[USER]") {
		t.Fatalf("expected prompt to include [USER], got %q", prompt)
	}
	if !strings.Contains(prompt, "core_skills") {
		t.Fatalf("expected prompt to include core_skills, got %q", prompt)
	}
	if !strings.Contains(prompt, "Principal Backend Engineer") {
		t.Fatalf("expected prompt to include preferred role, got %q", prompt)
	}
}
