package github

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// ghPRFields is the set of fields we request from gh pr list/view.
const ghPRFields = "number,title,author,headRefName,baseRefName,url,reviewDecision,statusCheckRollup,labels,additions,deletions,updatedAt,reviewRequests,latestReviews,comments"

// FetchPRs fetches all PRs where the current user's review is requested.
// repoDir is used as the working directory for the gh command (determines which repo).
func FetchPRs(repoDir string, limit int) ([]PR, error) {
	if limit <= 0 {
		limit = 50
	}
	cmd := exec.Command("gh", "pr", "list",
		"--search", "review-requested:@me",
		"--json", ghPRFields,
		"--limit", strconv.Itoa(limit),
	)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}

	var prs []PR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	for i := range prs {
		prs[i].Source = "review-requested"
		prs[i].CommentCount = len(prs[i].Comments)
	}

	return prs, nil
}

// FetchPR fetches a single PR by number or URL.
func FetchPR(repoDir string, numberOrURL string) (*PR, error) {
	cmd := exec.Command("gh", "pr", "view", "--", numberOrURL,
		"--json", ghPRFields,
	)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr view failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh pr view failed: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(out, &pr); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	pr.Source = "manual"
	pr.CommentCount = len(pr.Comments)
	return &pr, nil
}

// FetchComments fetches comments for a PR by number.
func FetchComments(repoDir string, prNumber int) ([]Comment, error) {
	cmd := exec.Command("gh", "pr", "view", strconv.Itoa(prNumber),
		"--json", "comments",
	)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh pr view comments failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh pr view comments failed: %w", err)
	}

	var result struct {
		Comments []Comment `json:"comments"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing comments output: %w", err)
	}

	return result.Comments, nil
}

// CheckGH verifies that the gh CLI is available and authenticated.
func CheckGH() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found. Install it from https://cli.github.com/")
	}

	// Use "gh auth token" - exits 0 and prints token if authenticated,
	// exits 1 if not. Simpler and more reliable than "gh auth status".
	cmd := exec.Command("gh", "auth", "token")
	cmd.Stdout = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh is not authenticated. Run 'gh auth login' first")
	}

	return nil
}

// CheckGitRepo verifies the given directory is inside a git repository.
func CheckGitRepo(dir string) error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository. Run approver from within a git repo")
	}
	return nil
}

// ParseRepoFromDir extracts the repo owner/name from a git directory.
func ParseRepoFromDir(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository or no remote 'origin' configured")
	}

	url := strings.TrimSpace(string(out))
	return url, nil
}
