# Approver Phase 1 MVP - Implementation Plan

## Context

PR reviews pile up. Approver is a Go TUI for managing GitHub PR reviews - fetch PRs where your review is requested, create worktrees, display status, and navigate efficiently. Phase 1 is the core MVP: dashboard, worktrees, status, keyboard navigation. No Claude Code integration or review posting yet (Phase 2+).

## Task 1: Project Scaffold

1. `go mod init github.com/kraft/approver`
2. Dependencies: bubbletea, lipgloss, bubbles, cobra, yaml.v3
3. Directory structure: main.go, internal/{app,ui,github,worktree,config}
4. Minimal main.go with cobra root command calling app.Run()
5. Stub app.go with Bubble Tea model that shows "Approver" and quits on `q`

## Task 2: GitHub PR Fetching

- `PR` struct with JSON tags matching `gh --json` output
- `Source` field: "review-requested" (auto) vs "manual" (user-added)
- `FetchPRs(repoDir string)` - shells out to `gh pr list --search "review-requested:@me"`
- `FetchPR(repoDir string, numberOrURL string)` - fetch single PR
- Helper methods: CIStatus(), ReviewIcon(), SizeString()

## Task 3: UI Components

- PR List (left panel, 30% width) with two-line items
- PR Detail (right panel, 70% width) with full PR info
- Menu bar with context-sensitive key hints
- Error box with auto-clear
- Styles for CI status, review state, selection

## Task 4: Application Shell

Single-model Bubble Tea architecture:
- State machine: stateDefault, stateConfirm, stateHelp, stateLoading
- Init(): spinner tick + async fetchPRsCmd
- Update(): window sizing (30/70 split), key dispatch, message handling
- View(): lipgloss.JoinHorizontal(list, detail) + menu/errbox

## Task 5: Manual PR Tracking

- `a` key opens text input for PR number/URL
- FetchPR() fetches single PR, adds with Source: "manual"
- Persists to ~/.config/approver/tracked.json
- `d` removes manually-tracked PRs
- Merged with auto-fetched on refresh (deduped)

## Task 6: Worktree Management

- WorktreeManager with baseDir (~/.config/approver/worktrees)
- Create: git fetch + git worktree add
- Delete: git worktree remove with confirmation
- ScanExisting: reconcile on startup
- Path: {baseDir}/pr-{number}-{sanitized-branch}

## Task 7: Help Overlay + Config

- Help overlay (?): centered keybinding list
- Open in browser (o): gh pr view --web
- YAML config at ~/.config/approver/config.yaml

## Task 8: Polish

- Empty state message
- Missing gh check at startup
- Loading spinner
- .gitignore, Makefile

## Key Architecture Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Bubble Tea model | Single model | Proven by Claude Squad, simpler |
| Layout | Two-panel (30/70) | Three panels too cramped |
| GitHub/git | Shell out to gh/git | Handles auth, simpler than SDKs |
| Worktree creation | Explicit (w key) | User decides which PRs to review locally |
| Refresh | Manual only (R key) | Background polling deferred to Phase 3 |
| PR sources | Auto + manual | Track own PRs or ones you want to pick up |
| Worktree location | ~/.config/approver/worktrees/ | Keeps user's repo clean |
