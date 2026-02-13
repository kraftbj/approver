package app

import (
	"github.com/kraft/approver/internal/claude"
	gh "github.com/kraft/approver/internal/github"
)

// commentsLoadedMsg is sent when PR comments have been fetched.
type commentsLoadedMsg struct {
	prNumber int
	comments []gh.Comment
}

// commentsErrorMsg is sent when fetching comments fails.
type commentsErrorMsg struct {
	prNumber int
	err      error
}

// Messages used by the Bubble Tea update loop.

// prsLoadedMsg is sent when PRs have been fetched from GitHub.
type prsLoadedMsg struct {
	prs []gh.PR
}

// prsErrorMsg is sent when fetching PRs fails.
type prsErrorMsg struct {
	err error
}

// hideErrMsg is sent after a timer to clear the error display.
type hideErrMsg struct{}

// worktreeCreatedMsg is sent after a worktree is successfully created.
type worktreeCreatedMsg struct {
	prNumber int
	path     string
}

// worktreeDeletedMsg is sent after a worktree is deleted.
type worktreeDeletedMsg struct {
	prNumber int
}

// worktreeErrorMsg is sent when a worktree operation fails.
type worktreeErrorMsg struct {
	prNumber int
	err      error
}

// prAddedMsg is sent when a single PR is fetched for manual tracking.
type prAddedMsg struct {
	pr gh.PR
}

// prAddErrorMsg is sent when adding a manual PR fails.
type prAddErrorMsg struct {
	err error
}

// claudeReviewDoneMsg is sent when the review pipeline completes.
type claudeReviewDoneMsg struct {
	prNumber int
	review   claude.ReviewResult
}

// claudeReviewErrorMsg is sent when the review pipeline fails.
type claudeReviewErrorMsg struct {
	prNumber int
	err      error
}

// claudeReviewProgressMsg carries pipeline step updates for display.
type claudeReviewProgressMsg struct {
	prNumber int
	step     string
}

// tmuxSessionErrorMsg is sent when a tmux operation fails.
type tmuxSessionErrorMsg struct {
	err error
}

// reviewSpinnerTickMsg rotates the fun review spinner messages.
type reviewSpinnerTickMsg struct{}

// prApprovedMsg is sent when a PR is successfully approved.
type prApprovedMsg struct {
	prNumber int
}

// prApproveErrorMsg is sent when approving a PR fails.
type prApproveErrorMsg struct {
	prNumber int
	err      error
}

// setupDoneMsg is sent when a per-repo setup command completes.
type setupDoneMsg struct {
	prNumber int
	wtPath   string
}

// setupErrorMsg is sent when a per-repo setup command fails.
type setupErrorMsg struct {
	prNumber int
	err      error
}

