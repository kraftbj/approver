package claude

import (
	"fmt"
	"os/exec"
)

// TmuxSession represents a tmux session for interactive Claude Code.
type TmuxSession struct {
	Name         string
	WorktreePath string
}

// CheckTmux verifies that tmux is available on PATH.
func CheckTmux() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// NewTmuxSession creates a TmuxSession for the given PR.
func NewTmuxSession(prNumber int, worktreePath string) *TmuxSession {
	return &TmuxSession{
		Name:         fmt.Sprintf("approver-pr-%d", prNumber),
		WorktreePath: worktreePath,
	}
}

// NewIssueTmuxSession creates a TmuxSession for the given issue.
func NewIssueTmuxSession(issueNumber int, worktreePath string) *TmuxSession {
	return &TmuxSession{
		Name:         fmt.Sprintf("approver-issue-%d", issueNumber),
		WorktreePath: worktreePath,
	}
}

// Create starts a new detached tmux session running claude in the worktree.
func (s *TmuxSession) Create() error {
	cmd := exec.Command("tmux", "new-session", "-d", "-s", s.Name, "-c", s.WorktreePath, "claude")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux new-session failed: %s", string(out))
	}
	// Best-effort: set status bar hint for returning to Approver
	exec.Command("tmux", "set-option", "-t", s.Name, "status-right", " Ctrl+b d: back to Approver ").Run()
	return nil
}

// AttachCmd returns an exec.Cmd that attaches to the tmux session.
// Use with tea.ExecProcess to hand off the terminal.
func (s *TmuxSession) AttachCmd() *exec.Cmd {
	return exec.Command("tmux", "attach-session", "-t", s.Name)
}

// Exists checks whether the tmux session is still running.
func (s *TmuxSession) Exists() bool {
	cmd := exec.Command("tmux", "has-session", "-t", s.Name)
	return cmd.Run() == nil
}

// Kill terminates the tmux session.
func (s *TmuxSession) Kill() error {
	cmd := exec.Command("tmux", "kill-session", "-t", s.Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux kill-session failed: %s", string(out))
	}
	return nil
}
