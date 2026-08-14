package claude

import (
	"fmt"
	"strings"
)

// InteractiveReviewPrompt returns a prompt suitable for a visible Claude session.
func InteractiveReviewPrompt(prNumber int, baseRef, headRef, customPrompt string) string {
	prompt := customPrompt
	if prompt == "" {
		prompt = DefaultPrompt
	}

	var parts []string
	if prNumber > 0 {
		parts = append(parts, fmt.Sprintf("Review PR #%d in this worktree.", prNumber))
	} else {
		parts = append(parts, "Review the PR in this worktree.")
	}
	parts = append(parts, "If /ce-code-review is available, use it for the review. If it is not available, perform an equivalent read-only code review yourself.")
	parts = append(parts, "Before judging the diff, inspect the PR history: read existing PR comments and reviews, then inspect the commit list and commit messages so you understand how the PR evolved.")
	if baseRef != "" {
		parts = append(parts, fmt.Sprintf("Base branch: %s.", baseRef))
	}
	if headRef != "" {
		parts = append(parts, fmt.Sprintf("Head branch: %s.", headRef))
	}
	if prNumber > 0 {
		historyHint := fmt.Sprintf("Useful starting points: `gh pr view %d --comments` for PR discussion and `gh pr view %d --json commits` for commit history.", prNumber, prNumber)
		if baseRef != "" {
			historyHint += fmt.Sprintf(" Also inspect `git log --oneline --decorate origin/%s..HEAD` when that base ref is available locally.", baseRef)
		}
		parts = append(parts, historyHint)
	}
	parts = append(parts, prompt)
	return strings.Join(parts, " ")
}
