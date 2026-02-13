package worktree

import (
	"testing"
)

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fix/auth-token", "fix-auth-token"},
		{"feature/user-profile", "feature-user-profile"},
		{"simple", "simple"},
		{"a/b/c/d", "a-b-c-d"},
		{"branch with spaces", "branch-with-spaces"},
	}

	for _, tt := range tests {
		got := sanitizeBranch(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeBranch(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSanitizeBranchTruncation(t *testing.T) {
	long := "this-is-a-very-long-branch-name-that-exceeds-forty-characters-total"
	got := sanitizeBranch(long)
	if len(got) > 40 {
		t.Errorf("sanitizeBranch should truncate to 40 chars, got %d", len(got))
	}
}

func TestExtractPRNumber(t *testing.T) {
	tests := []struct {
		dirName  string
		expected int
	}{
		{"pr-142-fix-auth-token", 142},
		{"pr-1-simple", 1},
		{"pr-9999-long-branch-name", 9999},
		{"not-a-pr-dir", 0},
		{"pr--missing-number", 0},
	}

	for _, tt := range tests {
		got := extractPRNumber(tt.dirName)
		if got != tt.expected {
			t.Errorf("extractPRNumber(%q) = %d, want %d", tt.dirName, got, tt.expected)
		}
	}
}

func TestWorktreePath(t *testing.T) {
	m := &Manager{BaseDir: "/tmp/worktrees", RepoDir: "/tmp/repo"}
	path := m.WorktreePath(142, "fix/auth-token")
	expected := "/tmp/worktrees/pr-142-fix-auth-token"
	if path != expected {
		t.Errorf("WorktreePath = %q, want %q", path, expected)
	}
}
