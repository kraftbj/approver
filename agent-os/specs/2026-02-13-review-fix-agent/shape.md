# Review Fix Agent — Shape

## Problem

After running the 3-agent Claude review pipeline or reading GitHub PR comments, acting on findings is manual — the user must open a tmux session or editor and fix things by hand. This breaks the review-fix-commit loop.

## Solution

A fix agent that takes selected review findings, applies fixes in the worktree using Claude, and offers to commit and push — all without leaving the TUI.

## Flow

1. User views review results (Tab to review tab) or comments (Tab to comments tab)
2. User presses `F` — selection overlay lists findings from both sources
3. User selects which findings to fix (j/k, space, enter)
4. Fix agent runs headlessly in the worktree (spinner shown)
5. Agent completes — output shown in fix detail tab — confirm overlay: commit and push? (y/n)
6. y → auto git add + commit + push in worktree
7. n → changes left in worktree for manual inspection

## Non-Goals

- Streaming agent output (same as review pipeline — no streaming)
- Auto-fixing without user selection
- Fixing issues on the Issues or Watchlist screens
