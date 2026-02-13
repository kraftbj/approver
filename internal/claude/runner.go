package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// DefaultPrompt is the default Agent 1 review prompt.
const DefaultPrompt = "Review the code changes in this repository. Analyze for: security vulnerabilities, correctness and logic bugs, edge cases, performance issues, and best practices. Report findings with severity ratings (high/medium/low) and file:line references."

// DefaultAllowedTools is the set of tools review agents can use.
const DefaultAllowedTools = "Read,Glob,Grep"

// DefaultBudget is the default max USD per review.
const DefaultBudget = 1.00

// agent2PromptTemplate is the prompt for Agent 2 (existing review validation).
const agent2PromptTemplate = `Here are the existing review comments on this PR:

%s

Validate each concern against the actual code in this repository. For each comment, determine if the concern is valid, a false positive, or a style preference. Summarize confirmed concerns with file:line references.`

// agent3PromptTemplate is the prompt for Agent 3 (comparator).
const agent3PromptTemplate = `You have two sets of review findings:

## Independent Code Review
%s

## Existing Review Validation
%s

Independently verify each reported issue against the actual code. Dismiss false positives, style nits, and over-engineering suggestions. Produce a NUMBERED CHECKLIST of confirmed issues. Each item: number, file path, line number, what's wrong, severity (high/medium/low), and what the fix should be. Only real bugs and significant issues make the list.`

// CheckClaude verifies that the claude CLI is available on PATH.
func CheckClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// RunAgent executes a single claude -p call in the given worktree directory.
func RunAgent(ctx context.Context, worktreePath, prompt, allowedTools string) (string, error) {
	args := []string{"-p", prompt, "--allowedTools", allowedTools}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = worktreePath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("claude failed: %s\n%s", err, stderr.String())
	}

	return stdout.String(), nil
}

// ghComment represents a single comment from gh pr view --json output.
type ghComment struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type ghCommentsResponse struct {
	Comments []ghComment `json:"comments"`
}

// FetchPRComments fetches existing review comments for a PR.
func FetchPRComments(repoDir string, prNumber int) (string, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", prNumber),
		"--json", "comments",
	)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh pr view comments failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("gh pr view comments failed: %w", err)
	}

	var resp ghCommentsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("parsing comments: %w", err)
	}

	if len(resp.Comments) == 0 {
		return "(no existing review comments)", nil
	}

	var parts []string
	for _, c := range resp.Comments {
		parts = append(parts, fmt.Sprintf("@%s:\n%s", c.Author.Login, c.Body))
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

// ProgressFunc is called with pipeline step updates.
type ProgressFunc func(step string)

// RunReviewPipeline orchestrates the full 3-agent review pipeline.
func RunReviewPipeline(ctx context.Context, worktreePath, repoDir string, prNumber int, prompt, allowedTools string, onProgress ProgressFunc) (*ReviewResult, error) {
	if prompt == "" {
		prompt = DefaultPrompt
	}
	if allowedTools == "" {
		allowedTools = DefaultAllowedTools
	}

	result := &ReviewResult{}

	// Step 1: Fetch existing PR comments
	if onProgress != nil {
		onProgress("Fetching PR comments...")
	}
	comments, err := FetchPRComments(repoDir, prNumber)
	if err != nil {
		// Non-fatal: continue without comments
		comments = "(failed to fetch comments)"
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Step 2: Run Agent 1 and Agent 2 concurrently
	var agent1Out, agent2Out string
	var agent1Err, agent2Err error
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		if onProgress != nil {
			onProgress("Agent 1: reviewing code...")
		}
		agent1Out, agent1Err = RunAgent(ctx, worktreePath, prompt, allowedTools)
	}()

	go func() {
		defer wg.Done()
		if onProgress != nil {
			onProgress("Agent 2: validating reviews...")
		}
		agent2Prompt := fmt.Sprintf(agent2PromptTemplate, comments)
		agent2Out, agent2Err = RunAgent(ctx, worktreePath, agent2Prompt, allowedTools)
	}()

	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if agent1Err != nil {
		return nil, fmt.Errorf("agent 1 failed: %w", agent1Err)
	}
	if agent2Err != nil {
		return nil, fmt.Errorf("agent 2 failed: %w", agent2Err)
	}

	result.Agent1Out = agent1Out
	result.Agent2Out = agent2Out

	// Step 3: Run Agent 3 (comparator)
	if onProgress != nil {
		onProgress("Agent 3: comparing findings...")
	}
	agent3Prompt := fmt.Sprintf(agent3PromptTemplate, agent1Out, agent2Out)
	agent3Out, err := RunAgent(ctx, worktreePath, agent3Prompt, allowedTools)
	if err != nil {
		return nil, fmt.Errorf("agent 3 failed: %w", err)
	}

	result.Agent3Out = agent3Out
	result.RawOutput = agent3Out
	result.Issues = parseChecklist(agent3Out)

	// Build summary
	if len(result.Issues) > 0 {
		result.Summary = fmt.Sprintf("%d issues found (%d high severity)", len(result.Issues), result.HighSeverityCount())
	} else {
		result.Summary = "Review complete (see raw output)"
	}

	return result, nil
}
