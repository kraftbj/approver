package github

import (
	"encoding/json"
	"testing"
	"time"
)

const sampleIssueJSON = `[
  {
    "number": 42,
    "title": "Add dark mode support",
    "author": {"login": "alice"},
    "labels": [{"name": "enhancement"}, {"name": "ui"}],
    "assignees": [{"login": "bob"}, {"login": "alice"}],
    "state": "OPEN",
    "body": "We should add dark mode to the app.\n\nAcceptance criteria:\n- Toggle in settings\n- System preference detection",
    "url": "https://github.com/org/repo/issues/42",
    "updatedAt": "2026-02-12T10:00:00Z"
  },
  {
    "number": 99,
    "title": "Fix login redirect loop",
    "author": {"login": "charlie"},
    "labels": [{"name": "bug"}],
    "assignees": [{"login": "bob"}],
    "state": "OPEN",
    "body": "Users get stuck in a redirect loop after login.",
    "url": "https://github.com/org/repo/issues/99",
    "updatedAt": "2026-02-11T08:30:00Z"
  },
  {
    "number": 100,
    "title": "Update dependencies",
    "author": {"login": "dave"},
    "labels": [],
    "assignees": [{"login": "bob"}],
    "state": "CLOSED",
    "body": "",
    "url": "https://github.com/org/repo/issues/100",
    "updatedAt": "2026-02-10T15:00:00Z"
  }
]`

func TestParseIssueJSON(t *testing.T) {
	var issues []Issue
	if err := json.Unmarshal([]byte(sampleIssueJSON), &issues); err != nil {
		t.Fatalf("failed to parse sample JSON: %v", err)
	}

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	issue := issues[0]
	if issue.Number != 42 {
		t.Errorf("expected number 42, got %d", issue.Number)
	}
	if issue.Title != "Add dark mode support" {
		t.Errorf("expected title 'Add dark mode support', got %q", issue.Title)
	}
	if issue.Author.Login != "alice" {
		t.Errorf("expected author alice, got %q", issue.Author.Login)
	}
	if issue.State != "OPEN" {
		t.Errorf("expected state OPEN, got %q", issue.State)
	}
	if len(issue.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issue.Labels))
	}
	if len(issue.Assignees) != 2 {
		t.Errorf("expected 2 assignees, got %d", len(issue.Assignees))
	}
	if issue.URL != "https://github.com/org/repo/issues/42" {
		t.Errorf("unexpected URL: %q", issue.URL)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2026-02-12T10:00:00Z")
	if !issue.UpdatedAt.Equal(expectedTime) {
		t.Errorf("expected updatedAt %v, got %v", expectedTime, issue.UpdatedAt)
	}
}

func TestIssueLabelNames(t *testing.T) {
	var issues []Issue
	json.Unmarshal([]byte(sampleIssueJSON), &issues)

	labels := issues[0].LabelNames()
	if len(labels) != 2 || labels[0] != "enhancement" || labels[1] != "ui" {
		t.Errorf("expected [enhancement, ui], got %v", labels)
	}

	labels = issues[2].LabelNames()
	if len(labels) != 0 {
		t.Errorf("expected empty labels, got %v", labels)
	}
}

func TestIssueAssigneeLogins(t *testing.T) {
	var issues []Issue
	json.Unmarshal([]byte(sampleIssueJSON), &issues)

	logins := issues[0].AssigneeLogins()
	if len(logins) != 2 || logins[0] != "bob" || logins[1] != "alice" {
		t.Errorf("expected [bob, alice], got %v", logins)
	}
}
