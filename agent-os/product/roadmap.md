# Product Roadmap

## Phase 1: Core MVP [DONE]

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

## Phase 3: GitHub Feedback Loop [DONE]

- ~~**Comment Viewer**~~ [DONE] - Tab cycles info/review/comments; shows author, time, body for each comment
- ~~**Review Posting**~~ [DONE] - `X` key requests changes with single-line reason input
- ~~**Background Poller**~~ [DONE] - Polls every `poll_interval` seconds (default 300, configurable, 0 = off)
- **Inline Comment Drafting** - Needs file/line browser UX design (deferred)

## Phase 4: Polish [DONE]

- ~~**Setup Scripts**~~ [DONE] - Configurable per-repo setup command run in each worktree
- ~~**Notifications**~~ [DONE] - Yellow `!` badge when comment count, CI, or review decision changes between polls
- ~~**Persistence**~~ [DONE] - Review results saved to ~/.config/approver/reviews/ and restored on startup
- ~~**Update Branch to Trunk**~~ [DONE] - `u` key merges origin/{base} into worktree, offers to push
- **Multi-Repo** - Needs config model for multiple repo dirs (deferred)
