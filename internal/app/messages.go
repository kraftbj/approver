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

// prChangesRequestedMsg is sent when changes are successfully requested on a PR.
type prChangesRequestedMsg struct {
	prNumber int
}

// prChangesRequestErrorMsg is sent when requesting changes fails.
type prChangesRequestErrorMsg struct {
	prNumber int
	err      error
}

// branchUpdatedMsg is sent when a worktree branch is updated from trunk.
type branchUpdatedMsg struct {
	prNumber int
}

// branchUpdateErrorMsg is sent when updating a branch fails.
type branchUpdateErrorMsg struct {
	prNumber int
	err      error
}

// branchPushedMsg is sent when a worktree branch is pushed to origin.
type branchPushedMsg struct {
	prNumber int
}

// branchPushErrorMsg is sent when pushing a branch fails.
type branchPushErrorMsg struct {
	prNumber int
	err      error
}

// pollTickMsg signals that a background poll interval has elapsed.
type pollTickMsg struct{}

// pollPRsLoadedMsg carries silently-refreshed PRs from a background poll.
type pollPRsLoadedMsg struct {
	prs []gh.PR
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

// Issue screen messages

// issuesLoadedMsg is sent when issues have been fetched from GitHub.
type issuesLoadedMsg struct {
	issues []gh.Issue
}

// issuesErrorMsg is sent when fetching issues fails.
type issuesErrorMsg struct {
	err error
}

// issueAddedMsg is sent when a single issue is fetched for manual tracking.
type issueAddedMsg struct {
	issue gh.Issue
}

// issueAddErrorMsg is sent when adding a manual issue fails.
type issueAddErrorMsg struct {
	err error
}

// issueWorktreeCreatedMsg is sent after an issue worktree is created.
type issueWorktreeCreatedMsg struct {
	issueNumber int
	path        string
}

// issueWorktreeDeletedMsg is sent after an issue worktree is deleted.
type issueWorktreeDeletedMsg struct {
	issueNumber int
}

// issueWorktreeErrorMsg is sent when an issue worktree operation fails.
type issueWorktreeErrorMsg struct {
	issueNumber int
	err         error
}

// Watchlist screen messages

// watchlistLoadedMsg is sent when watchlist PRs have been fetched.
type watchlistLoadedMsg struct {
	prs []gh.PR
}

// watchlistErrorMsg is sent when fetching watchlist PRs fails.
type watchlistErrorMsg struct {
	err error
}

// watchlistPRAddedMsg is sent when a PR is added to the watchlist.
type watchlistPRAddedMsg struct {
	pr gh.PR
}

// watchlistAddErrorMsg is sent when adding a watchlist PR fails.
type watchlistAddErrorMsg struct {
	err error
}

