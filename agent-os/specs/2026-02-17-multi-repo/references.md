# Multi-Repo Support — References

## Key Files
- `internal/app/app.go` — home struct, Run(), Init(), Update()
- `internal/app/messages.go` — all Bubble Tea message types
- `internal/app/commands.go` — tea.Cmd functions
- `internal/app/pr_screen.go` — Reviews screen handler
- `internal/app/issue_screen.go` — Issues screen handler
- `internal/app/watchlist_screen.go` — Watchlist screen handler
- `internal/app/reconcile.go` — state reconciliation
- `internal/app/helpers.go` — list manipulation, startReview, startTmux
- `internal/app/tracked.go` — tracked PR persistence
- `internal/app/tracked_issues.go` — tracked issue persistence
- `internal/app/reviews.go` — review result persistence
- `internal/config/config.go` — YAML config loading
- `internal/github/types.go` — PR, Issue types
- `internal/github/prs.go` — gh CLI fetch functions
- `internal/worktree/manager.go` — worktree CRUD
- `internal/ui/prlist.go` — PR list rendering
- `internal/ui/issuelist.go` — issue list rendering
- `internal/ui/styles.go` — lipgloss styles

## gh CLI Fields
- `gh pr list --json` returns: number, title, author, headRefName, baseRefName, url, reviewDecision, statusCheckRollup, labels, additions, deletions, updatedAt, reviewRequests, latestReviews, comments
- Adding `state` returns: "OPEN", "CLOSED", or "MERGED"

## Existing Patterns
- Single Bubble Tea model (`home` struct)
- Message-based updates via tea.Cmd/tea.Msg
- Shell out to `gh`/`git` (no SDK wrappers)
- State maps keyed by `int` (PR/issue number)
- Worktree paths: `{baseDir}/pr-{number}-{branch}/`
