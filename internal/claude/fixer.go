package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/kraft/approver/internal/github"
)

// DefaultFixAllowedTools is the set of tools the fix agent can use.
const DefaultFixAllowedTools = "Read,Write,Edit,Glob,Grep"

// fixPromptTemplate is the prompt for the fix agent.
const fixPromptTemplate = `Fix the following issues in this codebase. Make the minimal change needed for each. Do not refactor unrelated code.

Issues to fix:
%s`

// FixItem represents a single finding from either the Claude review or GitHub PR comments.
type FixItem struct {
	Source  string // "review" or "comment"
	Summary string // One-line description for the selection overlay
	Detail  string // Full text to include in the fix prompt
}

// FixItemsFromReview converts review issues into FixItems.
func FixItemsFromReview(result ReviewResult) []FixItem {
	if len(result.Issues) == 0 && result.Agent3Out == "" {
		return nil
	}

	// If we have parsed issues, use them
	if len(result.Issues) > 0 {
		items := make([]FixItem, 0, len(result.Issues))
		for _, issue := range result.Issues {
			summary := issue.Problem
			if issue.File != "" {
				summary = fmt.Sprintf("%s:%d — %s", issue.File, issue.Line, issue.Problem)
			}
			if issue.Severity != "" {
				summary = fmt.Sprintf("[%s] %s", issue.Severity, summary)
			}

			detail := issue.Problem
			if issue.File != "" {
				detail = fmt.Sprintf("%s:%d — %s", issue.File, issue.Line, issue.Problem)
			}
			if issue.Fix != "" {
				detail += "\nSuggested fix: " + issue.Fix
			}

			items = append(items, FixItem{
				Source:  "review",
				Summary: summary,
				Detail:  detail,
			})
		}
		return items
	}

	// Fallback: use the raw agent3 output as a single item
	lines := strings.Split(strings.TrimSpace(result.Agent3Out), "\n")
	summary := "Review findings"
	if len(lines) > 0 {
		summary = truncate(lines[0], 80)
	}
	return []FixItem{{
		Source:  "review",
		Summary: summary,
		Detail:  result.Agent3Out,
	}}
}

// FixItemsFromComments converts GitHub PR comments into FixItems.
// Filters out bot comments and comments that don't appear actionable.
func FixItemsFromComments(comments []github.Comment) []FixItem {
	var items []FixItem
	for _, c := range comments {
		// Skip bot comments
		if strings.HasSuffix(c.Author.Login, "[bot]") || strings.HasSuffix(c.Author.Login, "-bot") {
			continue
		}
		// Skip very short comments (likely reactions or acknowledgements)
		body := strings.TrimSpace(c.Body)
		if len(body) < 10 {
			continue
		}

		summary := fmt.Sprintf("@%s: %s", c.Author.Login, truncate(body, 70))

		items = append(items, FixItem{
			Source:  "comment",
			Summary: summary,
			Detail:  fmt.Sprintf("Comment by @%s:\n%s", c.Author.Login, body),
		})
	}
	return items
}

// RunFixAgent runs a Claude agent to fix the selected issues in the worktree.
func RunFixAgent(ctx context.Context, worktreePath string, items []FixItem, allowedTools string, onProgress ProgressFunc) (string, error) {
	if allowedTools == "" {
		allowedTools = DefaultFixAllowedTools
	}

	// Build the numbered issue list for the prompt
	var parts []string
	for i, item := range items {
		parts = append(parts, fmt.Sprintf("%d. [%s] %s", i+1, item.Source, item.Detail))
	}
	prompt := fmt.Sprintf(fixPromptTemplate, strings.Join(parts, "\n\n"))

	if onProgress != nil {
		onProgress("Fixing issues...")
	}

	output, err := RunAgent(ctx, worktreePath, prompt, allowedTools)
	if err != nil {
		return "", fmt.Errorf("fix agent failed: %w", err)
	}

	return output, nil
}

// truncate shortens a string to maxLen, adding "..." if truncated.
// Also collapses newlines to spaces for single-line display.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
