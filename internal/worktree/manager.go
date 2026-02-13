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
	// BaseDir is where worktrees are created (default: ~/.config/approver/worktrees).
	BaseDir string

	// RepoDir is the main git repository to create worktrees from.
	RepoDir string
}

// NewManager creates a worktree manager with default paths.
func NewManager(repoDir string) *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{
		BaseDir: filepath.Join(home, ".config", "approver", "worktrees"),
		RepoDir: repoDir,
	}
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
	if err := os.MkdirAll(m.BaseDir, 0o755); err != nil {
		return "", fmt.Errorf("creating worktree directory: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	// Fetch the branch from origin
	fetchCmd := exec.Command("git", "fetch", "origin", branch)
	fetchCmd.Dir = m.RepoDir
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch failed: %s", string(out))
	}

	// Create the worktree
	wtCmd := exec.Command("git", "worktree", "add", path, fmt.Sprintf("origin/%s", branch))
	wtCmd.Dir = m.RepoDir
	if out, err := wtCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add failed: %s", string(out))
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

// UpdateBranch merges the base branch into a PR's worktree and pushes.
func (m *Manager) UpdateBranch(prNumber int, baseBranch string) error {
	entries, err := m.ScanExisting()
	if err != nil {
		return err
	}

	path, exists := entries[prNumber]
	if !exists {
		return fmt.Errorf("no worktree found for PR #%d", prNumber)
	}

	// Fetch the base branch
	fetchCmd := exec.Command("git", "fetch", "origin", baseBranch)
	fetchCmd.Dir = path
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s", string(out))
	}

	// Merge
	mergeCmd := exec.Command("git", "merge", fmt.Sprintf("origin/%s", baseBranch), "--no-edit")
	mergeCmd.Dir = path
	if out, err := mergeCmd.CombinedOutput(); err != nil {
		// Abort the failed merge
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = path
		abortCmd.Run()
		return fmt.Errorf("merge conflict with %s: %s", baseBranch, string(out))
	}

	// Push
	pushCmd := exec.Command("git", "push")
	pushCmd.Dir = path
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s", string(out))
	}

	return nil
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
