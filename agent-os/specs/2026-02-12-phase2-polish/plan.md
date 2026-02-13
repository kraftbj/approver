# Phase 2 Polish: Bug Fixes and Enhancements

Addresses rough edges found during Phase 2 testing: misleading hints, missing spinners, extra keystrokes for worktree creation, and no way to approve PRs from the TUI.

## Changes

1. Fix review panel hint when no worktree exists
2. Add worktree creation spinner with `W` indicator in PR list
3. Auto-create worktree when pressing `c` or `t` without one
4. Fun rotating messages during review pipeline
5. Tmux status bar hint showing detach shortcut
6. `A` key to approve PRs with confirmation dialog
7. Per-repo setup commands after worktree creation
8. Update roadmap to mark items done
