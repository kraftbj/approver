package github

import (
	"fmt"
	"time"
)

// PR represents a GitHub pull request, matching the gh --json output format.
type PR struct {
	Number         int            `json:"number"`
	Title          string         `json:"title"`
	Author         Author         `json:"author"`
	HeadRefName    string         `json:"headRefName"`
	URL            string         `json:"url"`
	ReviewDecision string         `json:"reviewDecision"`
	StatusChecks   StatusChecks   `json:"statusCheckRollup"`
	Labels         []Label        `json:"labels"`
	Additions      int            `json:"additions"`
	Deletions      int            `json:"deletions"`
	UpdatedAt      time.Time      `json:"updatedAt"`

	// Source indicates how this PR was added to the list.
	// "review-requested" for auto-fetched, "manual" for user-added.
	Source string `json:"-"`

	// HasWorktree is set at runtime by reconciling with worktree state.
	HasWorktree bool `json:"-"`
}

type Author struct {
	Login string `json:"login"`
}

type Label struct {
	Name string `json:"name"`
}

type StatusChecks []StatusCheck

type StatusCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	// __typename distinguishes CheckRun vs StatusContext
	TypeName string `json:"__typename"`
}

// CIStatus returns a summary of CI check status: "pass", "fail", "pending", or "none".
func (pr *PR) CIStatus() string {
	if len(pr.StatusChecks) == 0 {
		return "none"
	}

	hasPending := false
	for _, check := range pr.StatusChecks {
		switch {
		case check.Conclusion == "FAILURE" || check.Conclusion == "ERROR" ||
			check.Conclusion == "TIMED_OUT" || check.Conclusion == "CANCELLED":
			return "fail"
		case check.Status == "IN_PROGRESS" || check.Status == "QUEUED" ||
			check.Status == "PENDING" || check.Conclusion == "":
			hasPending = true
		}
	}

	if hasPending {
		return "pending"
	}
	return "pass"
}

// ReviewIcon returns a display string for the review decision.
func (pr *PR) ReviewIcon() string {
	switch pr.ReviewDecision {
	case "APPROVED":
		return "APPROVED"
	case "CHANGES_REQUESTED":
		return "CHANGES"
	case "REVIEW_REQUIRED":
		return "REVIEW"
	default:
		return "PENDING"
	}
}

// SizeString returns a human-readable size like "+32/-5".
func (pr *PR) SizeString() string {
	return fmt.Sprintf("+%d/-%d", pr.Additions, pr.Deletions)
}

// RelativeTime returns a human-readable relative time like "2h ago" or "3d ago".
func (pr *PR) RelativeTime() string {
	d := time.Since(pr.UpdatedAt)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
