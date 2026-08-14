package app

import (
	"github.com/kraft/approver/internal/claude"
	"github.com/kraft/approver/internal/conductor"
	gh "github.com/kraft/approver/internal/github"
)

// worktreesScanMsg carries the result of scanning existing worktrees.
type worktreesScanMsg struct {
	repo      string
	worktrees map[ItemKey]string
}

// issueWorktreesScanMsg carries the result of scanning existing issue worktrees.
type issueWorktreesScanMsg struct {
	repo      string
	worktrees map[ItemKey]string
}

// trackedIssuesLoadedMsg carries the fetched tracked issues.
type trackedIssuesLoadedMsg struct {
	issues []gh.Issue
}

// reviewsLoadedMsg carries persisted review results loaded from disk.
type reviewsLoadedMsg struct {
	reviews map[ItemKey]claude.ReviewResult
}

// browserErrorMsg is sent when opening a URL in the browser fails.
type browserErrorMsg struct {
	err error
}

// trackedRemoveErrorMsg is sent when removing a tracked PR or issue fails.
type trackedRemoveErrorMsg struct {
	err error
}

// commentsLoadedMsg is sent when PR comments have been fetched.
type commentsLoadedMsg struct {
	key      ItemKey
	comments []gh.Comment
}

// commentsErrorMsg is sent when fetching comments fails.
type commentsErrorMsg struct {
	key ItemKey
	err error
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
	key  ItemKey
	path string
}

// worktreeDeletedMsg is sent after a worktree is deleted.
type worktreeDeletedMsg struct {
	key ItemKey
}

// worktreeErrorMsg is sent when a worktree operation fails.
type worktreeErrorMsg struct {
	key ItemKey
	err error
}

// conductorWorkspacesLoadedMsg carries ready workspaces from Conductor.
type conductorWorkspacesLoadedMsg struct {
	workspaces []conductor.Workspace
}

// conductorWorkspacesErrorMsg is sent when optional Conductor scanning fails.
type conductorWorkspacesErrorMsg struct {
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

// claudeReviewDoneMsg is sent when the review pipeline completes.
type claudeReviewDoneMsg struct {
	key    ItemKey
	review claude.ReviewResult
}

// claudeReviewErrorMsg is sent when the review pipeline fails.
type claudeReviewErrorMsg struct {
	key ItemKey
	err error
}

// claudeReviewProgressMsg carries pipeline step updates for display.
type claudeReviewProgressMsg struct {
	key  ItemKey
	step string
}

// tmuxSessionErrorMsg is sent when a tmux operation fails.
type tmuxSessionErrorMsg struct {
	err error
}

// reviewSpinnerTickMsg rotates the fun review spinner messages.
type reviewSpinnerTickMsg struct{}

// prApprovedMsg is sent when a PR is successfully approved.
type prApprovedMsg struct {
	key ItemKey
}

// prApproveErrorMsg is sent when approving a PR fails.
type prApproveErrorMsg struct {
	key ItemKey
	err error
}

// prChangesRequestedMsg is sent when changes are successfully requested on a PR.
type prChangesRequestedMsg struct {
	key ItemKey
}

// prChangesRequestErrorMsg is sent when requesting changes fails.
type prChangesRequestErrorMsg struct {
	key ItemKey
	err error
}

// branchUpdatedMsg is sent when a worktree branch is updated from trunk.
type branchUpdatedMsg struct {
	key ItemKey
}

// branchUpdateErrorMsg is sent when updating a branch fails.
type branchUpdateErrorMsg struct {
	key ItemKey
	err error
}

// branchPushedMsg is sent when a worktree branch is pushed to origin.
type branchPushedMsg struct {
	key ItemKey
}

// branchPushErrorMsg is sent when pushing a branch fails.
type branchPushErrorMsg struct {
	key ItemKey
	err error
}

// pollTickMsg signals that a background poll interval has elapsed.
type pollTickMsg struct{}

// pollPRsLoadedMsg carries silently-refreshed PRs from a background poll.
type pollPRsLoadedMsg struct {
	prs []gh.PR
}

// setupDoneMsg is sent when a per-repo setup command completes.
type setupDoneMsg struct {
	key    ItemKey
	wtPath string
}

// setupErrorMsg is sent when a per-repo setup command fails.
type setupErrorMsg struct {
	key ItemKey
	err error
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
	key  ItemKey
	path string
}

// issueWorktreeDeletedMsg is sent after an issue worktree is deleted.
type issueWorktreeDeletedMsg struct {
	key ItemKey
}

// issueWorktreeErrorMsg is sent when an issue worktree operation fails.
type issueWorktreeErrorMsg struct {
	key ItemKey
	err error
}

// itemDetectedMsg is sent when auto-detect resolves a PR or issue from user input.
type itemDetectedMsg struct {
	pr    *gh.PR
	issue *gh.Issue
}

// itemDetectErrorMsg is sent when auto-detect fails for both PR and issue.
type itemDetectErrorMsg struct {
	err error
}

// Fix agent messages

// fixDoneMsg is sent when the fix agent completes successfully.
type fixDoneMsg struct {
	key    ItemKey
	output string
}

// fixErrorMsg is sent when the fix agent fails.
type fixErrorMsg struct {
	key ItemKey
	err error
}

// fixProgressMsg carries fix agent progress updates.
type fixProgressMsg struct {
	key  ItemKey
	step string
}

// fixCommitPushDoneMsg is sent when fix commit+push completes.
type fixCommitPushDoneMsg struct {
	key ItemKey
}

// fixCommitPushErrorMsg is sent when fix commit+push fails.
type fixCommitPushErrorMsg struct {
	key ItemKey
	err error
}

// fixSpinnerTickMsg rotates the fun fix spinner messages.
type fixSpinnerTickMsg struct{}

// reviewSavedMsg is sent when a review has been saved to a file.
type reviewSavedMsg struct {
	path string
}

// reviewSaveErrorMsg is sent when saving a review to a file fails.
type reviewSaveErrorMsg struct {
	err error
}

// configSavedMsg is sent when the config has been saved to disk.
type configSavedMsg struct{}

// configSaveErrorMsg is sent when saving the config fails.
type configSaveErrorMsg struct {
	err error
}
