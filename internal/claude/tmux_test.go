package claude

import (
	"testing"
)

func TestNewTmuxSession_Name(t *testing.T) {
	s := NewTmuxSession(42, "/tmp/worktree")
	if s.Name != "approver-pr-42" {
		t.Errorf("expected name approver-pr-42, got %q", s.Name)
	}
	if s.WorktreePath != "/tmp/worktree" {
		t.Errorf("expected path /tmp/worktree, got %q", s.WorktreePath)
	}
}

func TestNewReviewTmuxSession_Name(t *testing.T) {
	s := NewReviewTmuxSession(42, "/tmp/worktree")
	if s.Name != "approver-pr-42-review" {
		t.Errorf("expected name approver-pr-42-review, got %q", s.Name)
	}
	if s.WorktreePath != "/tmp/worktree" {
		t.Errorf("expected path /tmp/worktree, got %q", s.WorktreePath)
	}
}

func TestAttachCmd_Args(t *testing.T) {
	s := NewTmuxSession(99, "/tmp/wt")
	cmd := s.AttachCmd()

	args := cmd.Args
	expected := []string{"tmux", "attach-session", "-t", "approver-pr-99"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, exp := range expected {
		if args[i] != exp {
			t.Errorf("arg %d: expected %q, got %q", i, exp, args[i])
		}
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote("/tmp/a repo/branch's")
	want := "'/tmp/a repo/branch'\\''s'"
	if got != want {
		t.Fatalf("shellQuote = %q, want %q", got, want)
	}
}
