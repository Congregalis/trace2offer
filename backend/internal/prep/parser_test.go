package prep

import "testing"

func TestParseQuestionScoreOutputInvalidJSONReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseQuestionScoreOutput("this is not json", 1, true)
	if err == nil {
		t.Fatalf("expected parse error on invalid json output")
	}
}

func TestParseQuestionScoreOutputMissingFieldsReturnsError(t *testing.T) {
	t.Parallel()

	_, err := parseQuestionScoreOutput(`{"score": 8.2}`, 2, true)
	if err == nil {
		t.Fatalf("expected parse error on missing required fields")
	}
}
