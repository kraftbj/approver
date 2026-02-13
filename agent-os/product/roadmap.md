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

- ~~**Review panel hint misleading without worktree**~~ [DONE] - Shows "press w to create a worktree first" when no worktree exists

## Enhancements

- ~~**Auto-create worktree on demand**~~ [DONE] - `c` and `t` auto-create worktrees, then dispatch the pending action
- ~~**Worktree creation spinner**~~ [DONE] - `W` indicator in PR list, "creating..." in detail panel, spinner while creating
- ~~**Approve from the app**~~ [DONE] - `A` key approves with confirmation dialog via `gh pr review --approve`
- ~~**Per-repo setup commands**~~ [DONE] - `repos` config map with `setup_command`, runs after worktree creation
- ~~**Tmux detach hint**~~ [DONE] - tmux status bar shows "Ctrl+b d: back to Approver"
- ~~**Fun review spinner**~~ [DONE] - Rotates through playful messages every 4 seconds during review

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
