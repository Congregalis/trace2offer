package agent

import "testing"

func TestParseResumeProfileOutput(t *testing.T) {
	t.Parallel()

	raw := "```json\n{\"name\":\"Carol\",\"total_years\":5,\"core_skills\":[\"Go\",\"System Design\"],\"preferred_roles\":\"Staff Engineer, Tech Lead\"}\n```"
	parsed, err := parseResumeProfileOutput(raw)
	if err != nil {
		t.Fatalf("parse resume output error: %v", err)
	}
	if parsed.Name != "Carol" {
		t.Fatalf("expected name Carol, got %q", parsed.Name)
	}
	if parsed.TotalYears != 5 {
		t.Fatalf("expected total years 5, got %v", parsed.TotalYears)
	}
	if len(parsed.CoreSkills) != 2 {
		t.Fatalf("expected 2 core skills, got %#v", parsed.CoreSkills)
	}
	if len(parsed.PreferredRoles) != 2 {
		t.Fatalf("expected split preferred roles, got %#v", parsed.PreferredRoles)
	}
}

func TestExtractResumeTextPlainText(t *testing.T) {
	t.Parallel()

	text, err := extractResumeText("resume.txt", "text/plain", []byte("Go Engineer\n\n5 years\tbackend"))
	if err != nil {
		t.Fatalf("extract plain text error: %v", err)
	}
	if text == "" {
		t.Fatal("expected extracted text")
	}
}
