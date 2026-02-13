package claude

import (
	"testing"
)

func TestParseChecklist_NumberedItems(t *testing.T) {
	output := `Here is the confirmed checklist:

1. **path/to/auth.go:42** - SQL injection vulnerability in login handler [high]
   The user input is concatenated directly into the query string.
   Fix: Use parameterized queries.

2. path/to/utils.go:15 - Nil pointer dereference when config is empty [medium]
   Fix: Add nil check before accessing config.Database.

3. api/handler.go:88 - Race condition in concurrent map access [high]
   Fix: Use sync.RWMutex to protect the shared map.
`

	issues := parseChecklist(output)

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// Issue 1
	if issues[0].Number != 1 {
		t.Errorf("issue 1 number: got %d", issues[0].Number)
	}
	if issues[0].File != "path/to/auth.go" {
		t.Errorf("issue 1 file: got %q", issues[0].File)
	}
	if issues[0].Line != 42 {
		t.Errorf("issue 1 line: got %d", issues[0].Line)
	}
	if issues[0].Severity != "high" {
		t.Errorf("issue 1 severity: got %q", issues[0].Severity)
	}

	// Issue 2
	if issues[1].Number != 2 {
		t.Errorf("issue 2 number: got %d", issues[1].Number)
	}
	if issues[1].Severity != "medium" {
		t.Errorf("issue 2 severity: got %q", issues[1].Severity)
	}

	// Issue 3
	if issues[2].File != "api/handler.go" {
		t.Errorf("issue 3 file: got %q", issues[2].File)
	}
	if issues[2].Line != 88 {
		t.Errorf("issue 3 line: got %d", issues[2].Line)
	}
}

func TestParseChecklist_ParenthesisFormat(t *testing.T) {
	output := `1) server.go:10 - Missing error check [low]
2) client.go:20 - Buffer overflow potential [high]
`
	issues := parseChecklist(output)
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].File != "server.go" {
		t.Errorf("issue 1 file: got %q", issues[0].File)
	}
	if issues[1].Severity != "high" {
		t.Errorf("issue 2 severity: got %q", issues[1].Severity)
	}
}

func TestParseChecklist_Empty(t *testing.T) {
	issues := parseChecklist("No issues found. The code looks clean.")
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestParseChecklist_NoSeverity(t *testing.T) {
	output := "1. main.go:5 - Unused import\n"
	issues := parseChecklist(output)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != "" {
		t.Errorf("expected empty severity, got %q", issues[0].Severity)
	}
}

func TestReviewResult_IssueCount(t *testing.T) {
	r := &ReviewResult{
		Issues: []Issue{
			{Number: 1, Severity: "high"},
			{Number: 2, Severity: "low"},
			{Number: 3, Severity: "high"},
		},
	}
	if r.IssueCount() != 3 {
		t.Errorf("expected 3, got %d", r.IssueCount())
	}
	if r.HighSeverityCount() != 2 {
		t.Errorf("expected 2 high, got %d", r.HighSeverityCount())
	}
}
