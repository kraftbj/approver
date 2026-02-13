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

## Phase 3: GitHub Feedback Loop [DONE]

- **Comment Viewer** - Tab cycles info/review/comments; shows author, time, body for each comment
- **Review Posting** - `X` key requests changes with single-line reason input
- **Background Poller** - Polls every `poll_interval` seconds (default 300, configurable, 0 = off)

## Phase 4: Polish [DONE]

- **Setup Scripts** - Configurable per-repo setup command run in each worktree
- **Notifications** - Yellow `!` badge when comment count, CI, or review decision changes between polls
- **Persistence** - Review results saved to ~/.config/approver/reviews/ and restored on startup
- **Update Branch to Trunk** - `u` key merges origin/{base} into worktree, offers to push

## Phase 5: Multi-Screen Infrastructure [DONE]

- **Screen interface** - `Screen` with `HandleKey()`/`View()`, `home` stays as tea.Model dispatcher
- **Screen switching** - `1`/`2`/`3` keys switch between Reviews, Issues, Watchlist
- **Screen indicator** - Menu bar shows `[1:Reviews] 2:Issues 3:Watchlist` with active screen highlighted
- **PR screen extraction** - PR-specific UI logic extracted from `handleDefaultKey` into `prScreen`
- **Shared state** - All domain state (worktrees, reviews, config) stays in `home`

## Phase 6: Issues Screen [DONE]

- **Fetch assigned issues** - `gh issue list --assignee @me` with JSON output
- **Issue list + detail** - Two-panel layout matching PR screen pattern
- **Issue workspaces** - Worktree + new branch from default branch, auto-named `issue-{number}-{slug}`
- **AI modes** - Collaborative (tmux session) and autonomous (headless Claude)
- **Manual additions** - `a` key to add issues by number/URL
- **Persistence** - `tracked-issues.json` for manually-added issues

## Phase 7: Watchlist Screen [DONE]

- **Merge manual tracking** - Migrate manually-tracked PRs from Reviews to Watchlist
- **Reviews cleanup** - Remove `a`/`d` keys from Reviews (review-requests only)
- **AI summaries** - Headless Claude call to summarize PR diff + comments
- **Watchlist keys** - `a` add, `d` remove, `s` AI summary, `o` open, `R` refresh

## Future

- **Review Fix Agent** - Pass review findings to a new Claude agent that fixes the issues in the worktree, then seeks reviewer approval before committing and pushing
- ~~**Review Word Wrap**~~ - Done. Review output, comments, and issue descriptions now word-wrap to fit the detail panel width
- **Inline Comment Drafting** - File/line browser UX for posting inline review comments
- **Multi-Repo** - Config model for multiple repo dirs, cross-repo watchlist
- **Watchlist: Issues** - Track issues alongside PRs in the watchlist
- **Cross-Repo Watchlist** - Track PRs/issues from repos other than the current one
