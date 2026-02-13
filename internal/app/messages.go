package app

import (
	gh "github.com/kraft/approver/internal/github"
)

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
	err error
}

// prAddedMsg is sent when a single PR is fetched for manual tracking.
type prAddedMsg struct {
	pr gh.PR
}

// prAddErrorMsg is sent when adding a manual PR fails.
type prAddErrorMsg struct {
	err error
}
