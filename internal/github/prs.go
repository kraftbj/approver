package github

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/kraft/approver/internal/debug"
)

var (
	repoEnvMu sync.RWMutex
	repoEnv   = make(map[string]map[string]string)
)

// SetRepoEnv registers extra environment variables for commands run in dir.
func SetRepoEnv(dir string, env map[string]string) {
	if len(env) == 0 {
		return
	}
	repoEnvMu.Lock()
	repoEnv[dir] = env
	repoEnvMu.Unlock()
}

// GHCommand creates an exec.Cmd for "gh" with the given args, dir, and any
// registered per-repo environment variables applied.
func GHCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("gh", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	repoEnvMu.RLock()
	extra := repoEnv[dir]
	repoEnvMu.RUnlock()
	if len(extra) > 0 {
		cmd.Env = os.Environ()
		for k, v := range extra {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd
}

// ghPRFields is the set of fields we request from gh pr list/view.
const ghPRFields = "number,title,author,headRefName,baseRefName,url,reviewDecision,statusCheckRollup,labels,additions,deletions,updatedAt,reviewRequests,latestReviews,comments,state"

// prSearchQueries are run separately and merged because GHE doesn't support OR.
var prSearchQueries = []string{
	"review-requested:@me",
	"author:@me",
	"assignee:@me",
}

// FetchPRs fetches all PRs where the current user's review is requested.
// repoDir is used as the working directory for the gh command (determines which repo).
func FetchPRs(repoDir string, limit int) ([]PR, error) {
	debug.Log("[FetchPRs] repoDir=%q limit=%d", repoDir, limit)
	if limit <= 0 {
		limit = 50
	}

	seen := make(map[int]bool)
	var prs []PR

	for _, query := range prSearchQueries {
		args := []string{"pr", "list", "--search", query, "--json", ghPRFields, "--limit", strconv.Itoa(limit)}
		debug.Log("[FetchPRs] running: gh %s (dir=%s)", strings.Join(args, " "), repoDir)
		cmd := GHCommand(repoDir, args...)

		out, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				debug.Log("[FetchPRs] ERROR query=%q stderr=%s", query, string(exitErr.Stderr))
				return nil, fmt.Errorf("gh pr list failed: %s", string(exitErr.Stderr))
			}
			debug.Log("[FetchPRs] ERROR query=%q err=%v", query, err)
			return nil, fmt.Errorf("gh pr list failed: %w", err)
		}

		var batch []PR
		if err := json.Unmarshal(out, &batch); err != nil {
			debug.Log("[FetchPRs] JSON parse error: %v (raw=%s)", err, string(out[:min(200, len(out))]))
			return nil, fmt.Errorf("parsing gh output: %w", err)
		}

		debug.Log("[FetchPRs] query=%q returned %d PRs", query, len(batch))
		for _, pr := range batch {
			if !seen[pr.Number] {
				seen[pr.Number] = true
				pr.Source = "auto"
				pr.CommentCount = len(pr.Comments)
				prs = append(prs, pr)
				debug.Log("[FetchPRs]   new PR #%d %q", pr.Number, pr.Title)
			} else {
				debug.Log("[FetchPRs]   dup PR #%d (skipped)", pr.Number)
			}
		}
	}

	debug.Log("[FetchPRs] total unique PRs: %d", len(prs))
	return prs, nil
}

// FetchPR fetches a single PR by number or URL.
func FetchPR(repoDir string, numberOrURL string) (*PR, error) {
	numberOrURL = strings.TrimSpace(numberOrURL)
	cmd := GHCommand(repoDir, "pr", "view", numberOrURL, "--json", ghPRFields)

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
	cmd := GHCommand(repoDir, "pr", "view", strconv.Itoa(prNumber), "--json", "comments")

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

// DefaultHost is the default GitHub hostname.
const DefaultHost = "github.com"

// CheckGHInstalled verifies that the gh CLI is on PATH.
func CheckGHInstalled() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found. Install it from https://cli.github.com/")
	}
	return nil
}

// CheckGHHosts verifies that the gh CLI is authenticated for each given host.
func CheckGHHosts(hosts []string) error {
	debug.Log("[CheckGHHosts] checking %d hosts: %v", len(hosts), hosts)
	for _, host := range hosts {
		cmd := exec.Command("gh", "auth", "token", "--hostname", host)
		cmd.Stdout = io.Discard
		if err := cmd.Run(); err != nil {
			debug.Log("[CheckGHHosts] FAIL host=%q err=%v", host, err)
			return fmt.Errorf("gh is not authenticated for %s. Run 'gh auth login --hostname %s'", host, host)
		}
		debug.Log("[CheckGHHosts] OK host=%q", host)
	}
	return nil
}

// ParseHostFromRemote extracts the hostname from a git remote URL.
// Supports SSH (git@host:org/repo.git) and HTTPS (https://host/org/repo.git) formats.
// Returns DefaultHost if the URL cannot be parsed.
func ParseHostFromRemote(remoteURL string) string {
	remoteURL = strings.TrimSpace(remoteURL)

	// SSH: git@host:org/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		if idx := strings.Index(remoteURL, ":"); idx > 4 {
			return remoteURL[4:idx]
		}
	}

	// HTTPS: https://host/org/repo.git
	if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		trimmed := remoteURL
		trimmed = strings.TrimPrefix(trimmed, "https://")
		trimmed = strings.TrimPrefix(trimmed, "http://")
		if idx := strings.Index(trimmed, "/"); idx > 0 {
			return trimmed[:idx]
		}
	}

	return DefaultHost
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
