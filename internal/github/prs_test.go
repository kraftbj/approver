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
    "baseRefName": "main",
    "url": "https://github.com/org/repo/pull/142",
    "reviewDecision": "APPROVED",
    "statusCheckRollup": [
      {"name": "CI", "status": "COMPLETED", "conclusion": "SUCCESS", "__typename": "CheckRun"},
      {"name": "lint", "status": "COMPLETED", "conclusion": "SUCCESS", "__typename": "CheckRun"},
      {"context": "Code coverage", "state": "SUCCESS", "__typename": "StatusContext"}
    ],
    "labels": [{"name": "bug"}, {"name": "auth"}],
    "additions": 32,
    "deletions": 5,
    "updatedAt": "2026-02-12T10:00:00Z",
    "reviewRequests": [
      {"__typename": "User", "login": "bob", "name": "Bob"},
      {"__typename": "Team", "login": "", "name": "security-team"}
    ],
    "latestReviews": [
      {"author": {"login": "bob"}, "state": "APPROVED"},
      {"author": {"login": "carol"}, "state": "CHANGES_REQUESTED"}
    ]
  },
  {
    "number": 143,
    "title": "Add user profile page",
    "author": {"login": "bob"},
    "headRefName": "feature/user-profile",
    "baseRefName": "main",
    "url": "https://github.com/org/repo/pull/143",
    "reviewDecision": "REVIEW_REQUIRED",
    "statusCheckRollup": [
      {"name": "CI", "status": "IN_PROGRESS", "conclusion": "", "__typename": "CheckRun"},
      {"context": "License check", "state": "PENDING", "__typename": "StatusContext"}
    ],
    "labels": [{"name": "feature"}],
    "additions": 245,
    "deletions": 12,
    "updatedAt": "2026-02-11T08:30:00Z",
    "reviewRequests": [],
    "latestReviews": []
  },
  {
    "number": 144,
    "title": "Refactor database layer",
    "author": {"login": "charlie"},
    "headRefName": "refactor/db",
    "baseRefName": "develop",
    "url": "https://github.com/org/repo/pull/144",
    "reviewDecision": "CHANGES_REQUESTED",
    "statusCheckRollup": [
      {"name": "CI", "status": "COMPLETED", "conclusion": "FAILURE", "__typename": "CheckRun"},
      {"context": "Code coverage", "state": "FAILURE", "__typename": "StatusContext"}
    ],
    "labels": [],
    "additions": 500,
    "deletions": 300,
    "updatedAt": "2026-02-10T15:00:00Z",
    "reviewRequests": [{"__typename": "User", "login": "alice", "name": "Alice"}],
    "latestReviews": [{"author": {"login": "alice"}, "state": "CHANGES_REQUESTED"}]
  },
  {
    "number": 145,
    "title": "Update README",
    "author": {"login": "dave"},
    "headRefName": "docs/readme",
    "baseRefName": "main",
    "url": "https://github.com/org/repo/pull/145",
    "reviewDecision": "",
    "statusCheckRollup": [],
    "labels": [{"name": "docs"}],
    "additions": 10,
    "deletions": 2,
    "updatedAt": "2026-02-12T12:00:00Z",
    "reviewRequests": [],
    "latestReviews": []
  },
  {
    "number": 146,
    "title": "Mixed checks with skipped",
    "author": {"login": "eve"},
    "headRefName": "feature/mixed",
    "baseRefName": "main",
    "url": "https://github.com/org/repo/pull/146",
    "reviewDecision": "",
    "statusCheckRollup": [
      {"name": "CI", "status": "COMPLETED", "conclusion": "SUCCESS", "__typename": "CheckRun"},
      {"name": "optional-lint", "status": "COMPLETED", "conclusion": "SKIPPED", "__typename": "CheckRun"},
      {"context": "Coverage gate", "state": "SUCCESS", "__typename": "StatusContext"}
    ],
    "labels": [],
    "additions": 5,
    "deletions": 1,
    "updatedAt": "2026-02-12T14:00:00Z",
    "reviewRequests": [],
    "latestReviews": []
  }
]`

func TestParsePRJSON(t *testing.T) {
	var prs []PR
	if err := json.Unmarshal([]byte(samplePRJSON), &prs); err != nil {
		t.Fatalf("failed to parse sample JSON: %v", err)
	}

	if len(prs) != 5 {
		t.Fatalf("expected 5 PRs, got %d", len(prs))
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
	if pr.BaseRefName != "main" {
		t.Errorf("expected base branch main, got %q", pr.BaseRefName)
	}
	if pr.ReviewDecision != "APPROVED" {
		t.Errorf("expected APPROVED, got %q", pr.ReviewDecision)
	}
	if len(pr.StatusChecks) != 3 {
		t.Errorf("expected 3 status checks, got %d", len(pr.StatusChecks))
	}
	if len(pr.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(pr.Labels))
	}
	if pr.Additions != 32 || pr.Deletions != 5 {
		t.Errorf("expected +32/-5, got +%d/-%d", pr.Additions, pr.Deletions)
	}
	if len(pr.ReviewRequests) != 2 {
		t.Errorf("expected 2 review requests, got %d", len(pr.ReviewRequests))
	}
	if len(pr.LatestReviews) != 2 {
		t.Errorf("expected 2 latest reviews, got %d", len(pr.LatestReviews))
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
		{0, "pass"},    // All checks succeeded (including StatusContext)
		{1, "pending"}, // CI in progress + pending StatusContext
		{2, "fail"},    // CI failed + failed StatusContext
		{3, "none"},    // No checks
		{4, "pass"},    // Pass + skipped + StatusContext pass = overall pass
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

func TestEffectiveState(t *testing.T) {
	tests := []struct {
		name     string
		check    StatusCheck
		expected string
	}{
		{"CheckRun SUCCESS", StatusCheck{TypeName: "CheckRun", Conclusion: "SUCCESS"}, "pass"},
		{"CheckRun NEUTRAL", StatusCheck{TypeName: "CheckRun", Conclusion: "NEUTRAL"}, "pass"},
		{"CheckRun FAILURE", StatusCheck{TypeName: "CheckRun", Conclusion: "FAILURE"}, "fail"},
		{"CheckRun TIMED_OUT", StatusCheck{TypeName: "CheckRun", Conclusion: "TIMED_OUT"}, "fail"},
		{"CheckRun SKIPPED", StatusCheck{TypeName: "CheckRun", Conclusion: "SKIPPED"}, "skipped"},
		{"CheckRun in progress", StatusCheck{TypeName: "CheckRun", Status: "IN_PROGRESS", Conclusion: ""}, "pending"},
		{"StatusContext SUCCESS", StatusCheck{TypeName: "StatusContext", State: "SUCCESS"}, "pass"},
		{"StatusContext FAILURE", StatusCheck{TypeName: "StatusContext", State: "FAILURE"}, "fail"},
		{"StatusContext ERROR", StatusCheck{TypeName: "StatusContext", State: "ERROR"}, "fail"},
		{"StatusContext PENDING", StatusCheck{TypeName: "StatusContext", State: "PENDING"}, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.check.EffectiveState()
			if got != tt.expected {
				t.Errorf("EffectiveState() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	checkRun := StatusCheck{Name: "CI", TypeName: "CheckRun"}
	if got := checkRun.DisplayName(); got != "CI" {
		t.Errorf("DisplayName() = %q, want %q", got, "CI")
	}

	statusCtx := StatusCheck{Context: "Code coverage", TypeName: "StatusContext"}
	if got := statusCtx.DisplayName(); got != "Code coverage" {
		t.Errorf("DisplayName() = %q, want %q", got, "Code coverage")
	}
}

func TestReviewerSummary(t *testing.T) {
	var prs []PR
	json.Unmarshal([]byte(samplePRJSON), &prs)

	// PR #142 has: bob (requested, approved), security-team (requested, team), carol (reviewed, changes)
	reviewers := prs[0].ReviewerSummary()
	if len(reviewers) != 3 {
		t.Fatalf("expected 3 reviewers, got %d", len(reviewers))
	}

	// bob: requested + approved
	if reviewers[0].Name != "bob" || reviewers[0].State != "approved" {
		t.Errorf("reviewer[0]: got %+v, want bob/approved", reviewers[0])
	}
	// security-team: requested, no review (team)
	if reviewers[1].Name != "security-team" || reviewers[1].State != "pending" || !reviewers[1].Team {
		t.Errorf("reviewer[1]: got %+v, want security-team/pending/team", reviewers[1])
	}
	// carol: not requested but reviewed with changes
	if reviewers[2].Name != "carol" || reviewers[2].State != "changes" {
		t.Errorf("reviewer[2]: got %+v, want carol/changes", reviewers[2])
	}

	// PR #144 has: alice (requested, changes_requested)
	reviewers = prs[2].ReviewerSummary()
	if len(reviewers) != 1 {
		t.Fatalf("expected 1 reviewer, got %d", len(reviewers))
	}
	if reviewers[0].Name != "alice" || reviewers[0].State != "changes" {
		t.Errorf("reviewer[0]: got %+v, want alice/changes", reviewers[0])
	}

	// PR #145 has no reviewers
	reviewers = prs[3].ReviewerSummary()
	if len(reviewers) != 0 {
		t.Errorf("expected 0 reviewers, got %d", len(reviewers))
	}
}

const sampleCommentsJSON = `{
  "comments": [
    {
      "author": {"login": "alice"},
      "body": "Looks good, just a few nits.",
      "createdAt": "2026-02-12T10:30:00Z",
      "url": "https://github.com/org/repo/pull/142#issuecomment-1"
    },
    {
      "author": {"login": "bob"},
      "body": "Fixed the nits, PTAL.",
      "createdAt": "2026-02-12T11:00:00Z",
      "url": "https://github.com/org/repo/pull/142#issuecomment-2"
    }
  ]
}`

func TestFetchComments(t *testing.T) {
	var result struct {
		Comments []Comment `json:"comments"`
	}
	if err := json.Unmarshal([]byte(sampleCommentsJSON), &result); err != nil {
		t.Fatalf("failed to parse sample comments JSON: %v", err)
	}

	comments := result.Comments
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}

	if comments[0].Author.Login != "alice" {
		t.Errorf("expected author alice, got %q", comments[0].Author.Login)
	}
	if comments[0].Body != "Looks good, just a few nits." {
		t.Errorf("unexpected body: %q", comments[0].Body)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2026-02-12T10:30:00Z")
	if !comments[0].CreatedAt.Equal(expectedTime) {
		t.Errorf("expected createdAt %v, got %v", expectedTime, comments[0].CreatedAt)
	}

	if comments[1].Author.Login != "bob" {
		t.Errorf("expected author bob, got %q", comments[1].Author.Login)
	}
	if comments[1].URL != "https://github.com/org/repo/pull/142#issuecomment-2" {
		t.Errorf("unexpected URL: %q", comments[1].URL)
	}
}

func TestParseHostFromRemote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"SSH github.com", "git@github.com:org/repo.git", "github.com"},
		{"SSH GHE", "git@github.example.com:org/repo.git", "github.example.com"},
		{"HTTPS github.com", "https://github.com/org/repo.git", "github.com"},
		{"HTTPS GHE", "https://github.example.com/org/repo.git", "github.example.com"},
		{"HTTPS without .git", "https://github.com/org/repo", "github.com"},
		{"empty string", "", DefaultHost},
		{"garbage", "not-a-url", DefaultHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHostFromRemote(tt.input)
			if got != tt.expected {
				t.Errorf("ParseHostFromRemote(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
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
