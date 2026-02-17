package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Manager handles git worktree operations for PR branches.
type Manager struct {
	// BaseDir is where worktrees are created (default: ~/.config/approver/worktrees/{repoName}).
	BaseDir string

	// RepoDir is the main git repository to create worktrees from.
	RepoDir string

	// RepoName is the short name of the repository.
	RepoName string
}

// NewManager creates a worktree manager with default paths.
// Worktrees are namespaced under {baseDir}/{repoName}/.
func NewManager(repoDir string, worktreeDir string, repoName string) (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	rootDir := filepath.Join(home, ".config", "approver", "worktrees")
	if worktreeDir != "" {
		rootDir = worktreeDir
	}
	baseDir := filepath.Join(rootDir, repoName)
	return &Manager{BaseDir: baseDir, RepoDir: repoDir, RepoName: repoName}, nil
}

// WorktreePath returns the path for a PR's worktree.
func (m *Manager) WorktreePath(prNumber int, branch string) string {
	sanitized := sanitizeBranch(branch)
	return filepath.Join(m.BaseDir, fmt.Sprintf("pr-%d-%s", prNumber, sanitized))
}

// Create sets up a worktree for a PR branch.
// It fetches the branch from origin and creates a worktree.
func (m *Manager) Create(prNumber int, branch string) (string, error) {
	path := m.WorktreePath(prNumber, branch)

	// Ensure base directory exists
	if err := os.MkdirAll(m.BaseDir, 0o700); err != nil {
		return "", fmt.Errorf("creating worktree directory: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Fetch the branch from origin
	fetchCmd := exec.Command("git", "fetch", "origin", "--", branch)
	fetchCmd.Dir = m.RepoDir
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %s", string(out))
	}

	// Create the worktree with a local branch tracking the remote
	wtCmd := exec.Command("git", "worktree", "add", "-b", branch, "--", path, fmt.Sprintf("origin/%s", branch))
	wtCmd.Dir = m.RepoDir
	if out, err := wtCmd.CombinedOutput(); err != nil {
		// Branch may already exist locally — try without -b
		wtCmd2 := exec.Command("git", "worktree", "add", "--", path, branch)
		wtCmd2.Dir = m.RepoDir
		if out2, err2 := wtCmd2.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("git worktree add failed: %s\n%s", string(out), string(out2))
		}
	}

	return path, nil
}

// Delete removes a worktree for a PR.
func (m *Manager) Delete(prNumber int) error {
	// Find the worktree path by scanning existing worktrees
	entries, err := m.ScanExisting()
	if err != nil {
		return err
	}

	path, exists := entries[prNumber]
	if !exists {
		return fmt.Errorf("no worktree found for PR #%d", prNumber)
	}

	cmd := exec.Command("git", "worktree", "remove", path, "--force")
	cmd.Dir = m.RepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed: %s", string(out))
	}

	return nil
}

// ScanExisting lists all approver-managed worktrees and maps PR numbers to paths.
func (m *Manager) ScanExisting() (map[int]string, error) {
	result := make(map[int]string)

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.RepoDir
	out, err := cmd.Output()
	if err != nil {
		return result, nil // Not fatal - just return empty
	}

	// Parse porcelain output - each worktree starts with "worktree <path>"
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		path := strings.TrimPrefix(line, "worktree ")

		// Check if this is an approver-managed worktree
		if !strings.HasPrefix(path, m.BaseDir) {
			continue
		}

		// Extract PR number from directory name (pr-{number}-{branch})
		base := filepath.Base(path)
		prNum := extractPRNumber(base)
		if prNum > 0 {
			result[prNum] = path
		}
	}

	return result, nil
}

// issueBranchName returns the branch name for an issue worktree.
func issueBranchName(issueNumber int, title string) string {
	slug := sanitizeBranch(strings.ToLower(title))
	return fmt.Sprintf("issue-%d-%s", issueNumber, slug)
}

// issueWorktreePath returns the path for an issue's worktree.
func (m *Manager) issueWorktreePath(issueNumber int, title string) string {
	branch := issueBranchName(issueNumber, title)
	return filepath.Join(m.BaseDir, branch)
}

// CreateForIssue creates a worktree for an issue, branching from the default branch.
func (m *Manager) CreateForIssue(issueNumber int, title string) (string, error) {
	branch := issueBranchName(issueNumber, title)
	path := filepath.Join(m.BaseDir, branch)

	if err := os.MkdirAll(m.BaseDir, 0o700); err != nil {
		return "", fmt.Errorf("creating worktree directory: %w", err)
	}

	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Determine default branch
	defaultBranch, err := m.defaultBranch()
	if err != nil {
		return "", err
	}

	// Fetch latest default branch
	fetchCmd := exec.Command("git", "fetch", "origin", "--", defaultBranch)
	fetchCmd.Dir = m.RepoDir
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %s", string(out))
	}

	// Create worktree with a new branch from origin/default
	wtCmd := exec.Command("git", "worktree", "add", "-b", branch, "--", path, fmt.Sprintf("origin/%s", defaultBranch))
	wtCmd.Dir = m.RepoDir
	if out, err := wtCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s", string(out))
	}

	return path, nil
}

// DeleteIssueWorktree removes a worktree for an issue.
func (m *Manager) DeleteIssueWorktree(issueNumber int) error {
	entries, err := m.ScanIssueWorktrees()
	if err != nil {
		return err
	}

	path, exists := entries[issueNumber]
	if !exists {
		return fmt.Errorf("no worktree found for issue #%d", issueNumber)
	}

	cmd := exec.Command("git", "worktree", "remove", path, "--force")
	cmd.Dir = m.RepoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed: %s", string(out))
	}

	return nil
}

// ScanIssueWorktrees lists all approver-managed issue worktrees.
func (m *Manager) ScanIssueWorktrees() (map[int]string, error) {
	result := make(map[int]string)

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.RepoDir
	out, err := cmd.Output()
	if err != nil {
		return result, nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		path := strings.TrimPrefix(line, "worktree ")

		if !strings.HasPrefix(path, m.BaseDir) {
			continue
		}

		base := filepath.Base(path)
		issueNum := extractIssueNumber(base)
		if issueNum > 0 {
			result[issueNum] = path
		}
	}

	return result, nil
}

// defaultBranch returns the default branch name (e.g., "main" or "master").
func (m *Manager) defaultBranch() (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = m.RepoDir
	out, err := cmd.Output()
	if err != nil {
		// Fallback: try "main", then "master"
		for _, branch := range []string{"main", "master"} {
			check := exec.Command("git", "rev-parse", "--verify", fmt.Sprintf("origin/%s", branch))
			check.Dir = m.RepoDir
			if check.Run() == nil {
				return branch, nil
			}
		}
		return "", fmt.Errorf("could not determine default branch")
	}
	// Output is like "origin/main" — strip the "origin/" prefix
	branch := strings.TrimSpace(string(out))
	branch = strings.TrimPrefix(branch, "origin/")
	return branch, nil
}

// issueNumberRegex matches "issue-{number}-" prefix.
var issueNumberRegex = regexp.MustCompile(`^issue-(\d+)-`)

// extractIssueNumber extracts the issue number from a worktree directory name.
func extractIssueNumber(dirName string) int {
	matches := issueNumberRegex.FindStringSubmatch(dirName)
	if len(matches) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(matches[1])
	return n
}

// prNumberRegex matches "pr-{number}-" prefix.
var prNumberRegex = regexp.MustCompile(`^pr-(\d+)-`)

// extractPRNumber extracts the PR number from a worktree directory name.
func extractPRNumber(dirName string) int {
	matches := prNumberRegex.FindStringSubmatch(dirName)
	if len(matches) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(matches[1])
	return n
}

// sanitizeBranch converts a branch name to a filesystem-safe directory name.
func sanitizeBranch(branch string) string {
	// Replace common path separators and special chars
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		" ", "-",
		":", "-",
		"~", "-",
	)
	result := replacer.Replace(branch)

	// Truncate to reasonable length
	if len(result) > 40 {
		result = result[:40]
	}

	return result
}
