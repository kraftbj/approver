package app

import (
	"testing"

	"github.com/kraft/approver/internal/conductor"
	"github.com/kraft/approver/internal/config"
	gh "github.com/kraft/approver/internal/github"
)

func TestReconcileConductorWorktreesMatchesRepoAndBranch(t *testing.T) {
	h := newHome()
	h.pr.prList.PRs = []gh.PR{
		{
			Number:      42,
			Repo:        "jetpack",
			RepoDir:     "/Users/kraft/code/jetpack",
			HeadRefName: "feature/test",
		},
		{
			Number:      43,
			Repo:        "jetpack",
			RepoDir:     "/Users/kraft/code/jetpack",
			HeadRefName: "feature/other",
		},
	}
	h.conductorWorkspaces = []conductor.Workspace{
		{
			RepoRoot: "/Users/kraft/code/jetpack",
			Branch:   "feature/test",
			Path:     "/Users/kraft/conductor/workspaces/jetpack/kigali",
		},
	}

	h.reconcileWorktrees()

	key := ItemKey{Repo: "jetpack", Number: 42}
	path, source, ok := h.worktreePathForKey(key)
	if !ok {
		t.Fatal("expected matched Conductor worktree")
	}
	if source != "conductor" {
		t.Fatalf("source: got %q", source)
	}
	if path != "/Users/kraft/conductor/workspaces/jetpack/kigali" {
		t.Fatalf("path: got %q", path)
	}
	if !h.pr.prList.PRs[0].HasWorktree {
		t.Fatal("expected first PR to have worktree")
	}
	if h.pr.prList.PRs[0].WorktreeSource != "conductor" {
		t.Fatalf("WorktreeSource: got %q", h.pr.prList.PRs[0].WorktreeSource)
	}
	if h.pr.prList.PRs[1].HasWorktree {
		t.Fatal("did not expect branch mismatch to have worktree")
	}
}

func TestWorktreePathForKeyPrefersApproverWorktree(t *testing.T) {
	h := newHome()
	key := ItemKey{Repo: "jetpack", Number: 42}
	h.worktrees[key] = "/tmp/approver"
	h.conductorWorktrees[key] = "/tmp/conductor"

	path, source, ok := h.worktreePathForKey(key)
	if !ok {
		t.Fatal("expected worktree")
	}
	if source != "approver" {
		t.Fatalf("source: got %q", source)
	}
	if path != "/tmp/approver" {
		t.Fatalf("path: got %q", path)
	}
}

func TestCountInitialFetches(t *testing.T) {
	repos := []config.RepoEntry{
		{Name: "all", Features: []string{"prs", "issues"}},
		{Name: "prs", Features: []string{"prs"}},
		{Name: "none", Features: []string{"unknown"}},
	}

	if got := countInitialFetches(repos); got != 3 {
		t.Fatalf("expected 3 fetches, got %d", got)
	}
}
