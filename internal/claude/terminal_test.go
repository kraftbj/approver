package claude

import (
	"strings"
	"testing"
)

func TestInteractiveReviewPrompt(t *testing.T) {
	got := InteractiveReviewPrompt(42, "trunk", "feature/review", "")
	for _, want := range []string{
		"Review PR #42 in this worktree.",
		"If /ce-code-review is available, use it for the review.",
		"read existing PR comments and reviews",
		"inspect the commit list and commit messages",
		"Base branch: trunk.",
		"Head branch: feature/review.",
		"gh pr view 42 --comments",
		"gh pr view 42 --json commits",
		"git log --oneline --decorate origin/trunk..HEAD",
		DefaultPrompt,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt %q does not contain %q", got, want)
		}
	}
}
