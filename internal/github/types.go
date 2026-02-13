package github

import (
	"fmt"
	"time"
)

// Comment represents a PR comment from GitHub.
type Comment struct {
	Author    Author    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
}

// PR represents a GitHub pull request, matching the gh --json output format.
type PR struct {
	Number         int            `json:"number"`
	Title          string         `json:"title"`
	Author         Author         `json:"author"`
	HeadRefName    string         `json:"headRefName"`
	BaseRefName    string         `json:"baseRefName"`
	Mergeable      string         `json:"mergeable"`
	URL            string         `json:"url"`
	ReviewDecision string         `json:"reviewDecision"`
	StatusChecks   StatusChecks   `json:"statusCheckRollup"`
	Labels         []Label        `json:"labels"`
	Additions      int            `json:"additions"`
	Deletions      int            `json:"deletions"`
	UpdatedAt      time.Time      `json:"updatedAt"`

	// Review data
	ReviewRequests []ReviewRequest `json:"reviewRequests"`
	LatestReviews  []Review        `json:"latestReviews"`

	// Source indicates how this PR was added to the list.
	// "review-requested" for auto-fetched, "manual" for user-added.
	Source string `json:"-"`

	// HasWorktree is set at runtime by reconciling with worktree state.
	HasWorktree bool `json:"-"`

	// HasReview is set at runtime when a completed review exists.
	HasReview bool `json:"-"`

	// IsReviewing is set at runtime when a review is in progress.
	IsReviewing bool `json:"-"`

	// HasTmux is set at runtime when a tmux Claude session exists.
	HasTmux bool `json:"-"`

	// IsCreatingWorktree is set at runtime while a worktree is being created.
	IsCreatingWorktree bool `json:"-"`
}

type Author struct {
	Login string `json:"login"`
}

type Label struct {
	Name string `json:"name"`
}

type StatusChecks []StatusCheck

type StatusCheck struct {
	// CheckRun fields
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	// StatusContext fields
	Context string `json:"context"`
	State   string `json:"state"`
	// __typename distinguishes CheckRun vs StatusContext
	TypeName string `json:"__typename"`
}

// DisplayName returns the check name, handling both CheckRun and StatusContext types.
func (c StatusCheck) DisplayName() string {
	if c.Context != "" {
		return c.Context
	}
	return c.Name
}

// EffectiveState normalizes the check result to "pass", "fail", "pending", or "skipped".
func (c StatusCheck) EffectiveState() string {
	if c.TypeName == "StatusContext" {
		switch c.State {
		case "SUCCESS":
			return "pass"
		case "FAILURE", "ERROR":
			return "fail"
		default:
			return "pending"
		}
	}
	// CheckRun
	if c.Conclusion == "" {
		return "pending"
	}
	switch c.Conclusion {
	case "SUCCESS", "NEUTRAL":
		return "pass"
	case "SKIPPED":
		return "skipped"
	case "FAILURE", "ERROR", "TIMED_OUT", "CANCELLED":
		return "fail"
	default:
		return "pending"
	}
}

// Review types

type ReviewRequest struct {
	TypeName string `json:"__typename"`
	Login    string `json:"login"`
	Name     string `json:"name"`
}

type Review struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State string `json:"state"`
}

type ReviewerStatus struct {
	Name  string
	State string // "pending", "approved", "changes", "commented"
	Team  bool
}

// ReviewerSummary builds a list of reviewers with their current state.
func (pr *PR) ReviewerSummary() []ReviewerStatus {
	// Build map of latest review state by login
	reviewed := make(map[string]string)
	for _, r := range pr.LatestReviews {
		var state string
		switch r.State {
		case "APPROVED":
			state = "approved"
		case "CHANGES_REQUESTED":
			state = "changes"
		case "COMMENTED":
			state = "commented"
		case "DISMISSED":
			state = "pending"
		default:
			continue
		}
		reviewed[r.Author.Login] = state
	}

	var result []ReviewerStatus
	seen := make(map[string]bool)

	// Start with requested reviewers
	for _, req := range pr.ReviewRequests {
		name := req.Login
		isTeam := req.TypeName == "Team"
		if isTeam {
			name = req.Name
			if name == "" {
				name = req.Login
			}
		}
		if name == "" {
			continue
		}

		key := name
		if seen[key] {
			continue
		}
		seen[key] = true

		state := "pending"
		if !isTeam {
			if s, ok := reviewed[req.Login]; ok {
				state = s
			}
		}
		result = append(result, ReviewerStatus{Name: name, State: state, Team: isTeam})
	}

	// Add reviewers who submitted reviews but aren't in the request list
	for _, r := range pr.LatestReviews {
		login := r.Author.Login
		if login == "" || seen[login] {
			continue
		}
		seen[login] = true
		state := reviewed[login]
		result = append(result, ReviewerStatus{Name: login, State: state})
	}

	return result
}

// CIStatus returns a summary of CI check status: "pass", "fail", "pending", or "none".
func (pr *PR) CIStatus() string {
	if len(pr.StatusChecks) == 0 {
		return "none"
	}

	hasPending := false
	for _, check := range pr.StatusChecks {
		switch check.EffectiveState() {
		case "fail":
			return "fail"
		case "pending":
			hasPending = true
		}
		// "pass" and "skipped" don't affect overall status
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
