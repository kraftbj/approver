package github

import (
	"encoding/json"
	"testing"
	"time"
)

const samplePRJSON = `[
  {
    "number": 142,
    "title": "Fix auth token refresh",
    "author": {"login": "alice"},
    "headRefName": "fix/auth-token",
    "url": "https://github.com/org/repo/pull/142",
    "reviewDecision": "APPROVED",
    "statusCheckRollup": [
      {"name": "CI", "status": "COMPLETED", "conclusion": "SUCCESS", "__typename": "CheckRun"},
      {"name": "lint", "status": "COMPLETED", "conclusion": "SUCCESS", "__typename": "CheckRun"}
    ],
    "labels": [{"name": "bug"}, {"name": "auth"}],
    "additions": 32,
    "deletions": 5,
    "updatedAt": "2026-02-12T10:00:00Z"
  },
  {
    "number": 143,
    "title": "Add user profile page",
    "author": {"login": "bob"},
    "headRefName": "feature/user-profile",
    "url": "https://github.com/org/repo/pull/143",
    "reviewDecision": "REVIEW_REQUIRED",
    "statusCheckRollup": [
      {"name": "CI", "status": "IN_PROGRESS", "conclusion": "", "__typename": "CheckRun"}
    ],
    "labels": [{"name": "feature"}],
    "additions": 245,
    "deletions": 12,
    "updatedAt": "2026-02-11T08:30:00Z"
  },
  {
    "number": 144,
    "title": "Refactor database layer",
    "author": {"login": "charlie"},
    "headRefName": "refactor/db",
    "url": "https://github.com/org/repo/pull/144",
    "reviewDecision": "CHANGES_REQUESTED",
    "statusCheckRollup": [
      {"name": "CI", "status": "COMPLETED", "conclusion": "FAILURE", "__typename": "CheckRun"}
    ],
    "labels": [],
    "additions": 500,
    "deletions": 300,
    "updatedAt": "2026-02-10T15:00:00Z"
  },
  {
    "number": 145,
    "title": "Update README",
    "author": {"login": "dave"},
    "headRefName": "docs/readme",
    "url": "https://github.com/org/repo/pull/145",
    "reviewDecision": "",
    "statusCheckRollup": [],
    "labels": [{"name": "docs"}],
    "additions": 10,
    "deletions": 2,
    "updatedAt": "2026-02-12T12:00:00Z"
  }
]`

func TestParsePRJSON(t *testing.T) {
	var prs []PR
	if err := json.Unmarshal([]byte(samplePRJSON), &prs); err != nil {
		t.Fatalf("failed to parse sample JSON: %v", err)
	}

	if len(prs) != 4 {
		t.Fatalf("expected 4 PRs, got %d", len(prs))
	}

	// Verify first PR fields
	pr := prs[0]
	if pr.Number != 142 {
		t.Errorf("expected number 142, got %d", pr.Number)
	}
	if pr.Title != "Fix auth token refresh" {
		t.Errorf("expected title 'Fix auth token refresh', got %q", pr.Title)
	}
	if pr.Author.Login != "alice" {
		t.Errorf("expected author alice, got %q", pr.Author.Login)
	}
	if pr.HeadRefName != "fix/auth-token" {
		t.Errorf("expected branch fix/auth-token, got %q", pr.HeadRefName)
	}
	if pr.ReviewDecision != "APPROVED" {
		t.Errorf("expected APPROVED, got %q", pr.ReviewDecision)
	}
	if len(pr.StatusChecks) != 2 {
		t.Errorf("expected 2 status checks, got %d", len(pr.StatusChecks))
	}
	if len(pr.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(pr.Labels))
	}
	if pr.Additions != 32 || pr.Deletions != 5 {
		t.Errorf("expected +32/-5, got +%d/-%d", pr.Additions, pr.Deletions)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2026-02-12T10:00:00Z")
	if !pr.UpdatedAt.Equal(expectedTime) {
		t.Errorf("expected updatedAt %v, got %v", expectedTime, pr.UpdatedAt)
	}
}

func TestCIStatus(t *testing.T) {
	var prs []PR
	json.Unmarshal([]byte(samplePRJSON), &prs)

	tests := []struct {
		prIndex  int
		expected string
	}{
		{0, "pass"},    // All checks succeeded
		{1, "pending"}, // CI in progress
		{2, "fail"},    // CI failed
		{3, "none"},    // No checks
	}

	for _, tt := range tests {
		got := prs[tt.prIndex].CIStatus()
		if got != tt.expected {
			t.Errorf("PR #%d CIStatus(): expected %q, got %q", prs[tt.prIndex].Number, tt.expected, got)
		}
	}
}

func TestReviewIcon(t *testing.T) {
	var prs []PR
	json.Unmarshal([]byte(samplePRJSON), &prs)

	tests := []struct {
		prIndex  int
		expected string
	}{
		{0, "APPROVED"},
		{1, "REVIEW"},
		{2, "CHANGES"},
		{3, "PENDING"},
	}

	for _, tt := range tests {
		got := prs[tt.prIndex].ReviewIcon()
		if got != tt.expected {
			t.Errorf("PR #%d ReviewIcon(): expected %q, got %q", prs[tt.prIndex].Number, tt.expected, got)
		}
	}
}

func TestSizeString(t *testing.T) {
	pr := PR{Additions: 32, Deletions: 5}
	if got := pr.SizeString(); got != "+32/-5" {
		t.Errorf("expected +32/-5, got %q", got)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		updatedAt time.Time
		contains  string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-48 * time.Hour), "2d ago"},
	}

	for _, tt := range tests {
		pr := PR{UpdatedAt: tt.updatedAt}
		got := pr.RelativeTime()
		if got != tt.contains {
			t.Errorf("RelativeTime() for %v: expected %q, got %q", tt.updatedAt, tt.contains, got)
		}
	}
}
