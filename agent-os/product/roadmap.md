# Product Roadmap

## Phase 1: Core MVP

- **PR Dashboard** - Fetch and display PRs requesting your review via `gh pr list --search "review-requested:@me"`
- **Worktree Management** - Create/delete git worktrees for each PR branch
- **Status Display** - Show PR title, author, branch, CI status, review state, labels
- **Keyboard Navigation** - j/k to navigate, Enter to open detail, q to quit

## Phase 2: Claude Code Integration

- **Manual Claude Review** - Trigger `claude -p` in a PR's worktree with a structured review prompt
- **Review Output** - Display Claude's analysis in a panel (security, performance, quality, architecture)
- **Embedded Terminal** - Attach to a Claude Code session for interactive review via tmux

## Phase 3: GitHub Feedback Loop

- **Background Poller** - Poll GitHub every N minutes for new comments, CI status changes, mergeability
- **Inline Comment Drafting** - Create draft inline comments on specific files/lines
- **Review Posting** - Submit draft reviews (approve/request changes/comment) via `gh api`
- **Comment Viewer** - Show existing PR comments and conversation threads

## Phase 4: Polish

- **Setup Scripts** - Configurable per-repo setup command run in each worktree
- **Notifications** - Visual indicators when PR state changes (new comment, CI passed/failed)
- **Multi-Repo** - Support reviewing PRs across multiple repos
- **Persistence** - Remember reviewed PRs, preserve draft comments across sessions
- **Update Branch to Trunk** - One-key merge of base branch into PR worktree and push (`u` key)
