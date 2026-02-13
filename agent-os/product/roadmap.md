# Product Roadmap

## Phase 1: Core MVP

- **PR Dashboard** - Fetch and display PRs requesting your review via `gh pr list --search "review-requested:@me"`
- **Worktree Management** - Create/delete git worktrees for each PR branch
- **Status Display** - Show PR title, author, branch, CI status, review state, labels
- **Keyboard Navigation** - j/k to navigate, Enter to open detail, q to quit

## Phase 2: Claude Code Integration [DONE]

- **3-Agent Review Pipeline** - `c` key triggers headless review (Agent 1: code review, Agent 2: validate existing reviews, Agent 3: produce confirmed checklist)
- **Review Output** - Display Claude's analysis in detail panel with Tab toggle between PR info and review results
- **Interactive Claude Session** - `t` key opens tmux session with Claude in the PR worktree
- **Configurable** - Review prompt, budget, and allowed tools via config.yaml
- **Fun review spinner** - Cycle through playful status messages while review runs (since there's no streaming output)

## Bugs

- **Review panel hint misleading without worktree** - "Press c to start a review" shows even when no worktree exists, but `c` then errors asking to press `w` first. Should say "Press w to create a worktree, then c to review."

## Enhancements

- **Auto-create worktree on demand** - When pressing `c` or `t` without a worktree, automatically create one instead of requiring `w` first
- **Worktree creation spinner** - Show a progress indicator while a worktree is being created (fetch + checkout can be slow)
- **Approve from the app** - Submit PR approval (and possibly request-changes/comment) directly from the TUI without opening a browser
- **Per-repo setup commands** - Config option to specify a command to run in each worktree after creation (e.g. `npm install`, `docker compose up`, `make dev`) to spin up local dev environments
- **Tmux detach hint** - Show a brief "Ctrl+b d to return" message before entering tmux, or display it in the tmux status bar

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
