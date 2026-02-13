# GitHub AI Hub: Multi-Screen Expansion

## Context

Approver is evolving from a PR review tool into a GitHub AI workflow hub with three screens. Phases 1-4 (review-focused) are done. This spec covers the full expansion: product doc updates, multi-screen infrastructure, Issues screen, and Watchlist screen.

## Decisions

- **Screen switching**: `1`/`2`/`3` keys switch between Reviews, Issues, Watchlist
- **Architecture**: `Screen` interface with `HandleKey()`/`View()`, `home` stays as tea.Model dispatcher, all domain state stays in `home`
- **Issue branches**: auto-named `issue-{number}-{slug}` by default, with option to customize
- **Issue source**: auto-fetch assigned issues + manual additions
- **Watchlist merges manually-tracked PRs**: Reviews screen becomes review-requests only, Watchlist gets the manual tracking + AI summaries
- **Watchlist scope**: PRs only for now (cross-repo and issues deferred)
- **AI modes for Issues**: collaborative (tmux session) or autonomous (headless review-style)
- **Naming**: defer rename decision, keep "Approver" for now

## Standards Applied

- `global/coding-style` — gofmt, small focused functions, DRY, no dead code
- `global/error-handling` — user-friendly messages, fail fast, graceful degradation
- `testing/test-writing` — test core flows, mock externals, fast tests
- `global/conventions` — consistent structure, clear commit messages

## Reference Implementations

- `internal/app/app.go` — current monolithic model (becomes dispatcher)
- `internal/app/app.go` handleDefaultKey — key handler pattern (each screen gets its own)
- `internal/ui/prlist.go` / `prdetail.go` — list+detail pattern (Issues screen follows this)
- `internal/app/tracked.go` — file persistence pattern (Watchlist follows this)
- `internal/github/prs.go` — gh CLI fetch pattern (issues.go follows this)
- `internal/github/types.go` — type + runtime field pattern (Issue type follows this)

---

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-02-12-github-ai-hub/` with:
- `plan.md` — this full plan
- `shape.md` — shaping decisions from the conversation
- `standards.md` — the four applicable standards
- `references.md` — reference implementations

## Task 2: Update Product Docs

**Files:** `agent-os/product/mission.md`, `agent-os/product/roadmap.md`

- **mission.md**: broaden from "PR review management" to "GitHub AI workflow hub" — three screens, AI-powered across all workflows
- **roadmap.md**: keep Phases 1-4 as historical record, add:
  - Phase 5: Multi-Screen Infrastructure (screen interface, 1/2/3 switching, shared state)
  - Phase 6: Issues Screen (fetch assigned, workspaces, AI modes)
  - Phase 7: Watchlist Screen (merge manual tracking, AI summaries)
  - Move deferred items (Inline Comment Drafting, Multi-Repo) to Future
- **tech-stack.md**: no changes needed

## Task 3: Multi-Screen Infrastructure

**Files:** new `internal/app/screen.go`, `internal/app/app.go`, new `internal/app/pr_screen.go`, `internal/ui/menu.go`

- Define `Screen` interface: `HandleKey(h *home, key string) tea.Cmd`, `View(h *home, height int) string`
- Add `activeScreen` enum to home: `screenReviews`, `screenIssues`, `screenWatchlist`
- Extract PR-specific UI logic from `handleDefaultKey` into `prScreen.HandleKey()`
- Extract `viewDashboard` into `prScreen.View()`
- PR-specific UI state moves to `prScreen`: `detailMode`, `reviewMsgIdx`, `prList`, `detail`
- Global state stays in `home`: all maps, config, errBox, menu, spinner, state machine
- `1`/`2`/`3` keys switch `activeScreen` (handled before dispatch to screen)
- Screen indicator in menu bar: `[1:Reviews] 2:Issues 3:Watchlist`
- Message handlers stay in `home.Update()` (messages are screen-agnostic)

## Task 4: Issues Screen — Types & Fetching

**Files:** new `internal/github/issues.go`, `internal/github/types.go`, new `internal/github/issues_test.go`

- `Issue` struct: Number, Title, Author, Labels, Assignees, State, Body, URL, UpdatedAt + runtime fields (HasWorkspace, IsWorking, HasTmux)
- `FetchIssues(repoDir)` — `gh issue list --assignee @me --json ... --limit 50`
- `FetchIssue(repoDir, numberOrURL)` — `gh issue view <ref> --json ...`
- Test with JSON fixture

## Task 5: Issues Screen — UI & Workspace

**Files:** new `internal/app/issue_screen.go`, new `internal/ui/issuelist.go`, new `internal/ui/issuedetail.go`, `internal/app/messages.go`, `internal/worktree/manager.go`

- `issueScreen` implements `Screen` interface
- Two-panel layout: issue list (30%) + detail (70%)
- Detail modes: info (metadata + body) and AI (session output)
- Keys: `j/k` navigate, `w`/`W` workspace, `a` add, `d` remove, `c` collaborative Claude, `C` autonomous Claude, `o` open, `R` refresh, `Tab` toggle
- Workspace = worktree + new branch from default branch
  - Auto-name: `issue-{number}-{sanitized-title}`
  - Extend `worktree.Manager` to support creating from base branch
- Messages: `issuesLoadedMsg`, `issuesErrorMsg`, `issueAddedMsg`, etc.
- Persistence: `tracked-issues.json` (same pattern as `tracked.json`)

## Task 6: Watchlist Screen — Merge Manual Tracking

**Files:** new `internal/app/watchlist_screen.go`, new `internal/ui/watchlist.go`, `internal/app/app.go`, `internal/app/tracked.go`

- Migrate manually-tracked PRs from Reviews to Watchlist
- Remove `a`/`d` keys from Reviews screen (review-requests only now)
- Watchlist owns `tracked.json`
- Keys: `j/k` navigate, `a` add, `d` remove, `s` AI summary, `o` open, `R` refresh, `Tab` toggle
- AI summary: headless Claude call to summarize PR diff + comments
- Summaries stored in memory (not persisted)

## Task 7: Roadmap Update

Mark Phase 5/6/7 items as done in `agent-os/product/roadmap.md`.

---

## Execution Order

Tasks 1-2 (docs) → Task 3 (infra, must come first) → Tasks 4-5 (Issues) → Task 6 (Watchlist) → Task 7 (cleanup)

One commit per task. Build + test after each.

## Verification (per task)

1. `go build -o approver .` succeeds
2. `go test ./...` passes
3. Feature works as described
4. Existing Reviews screen functionality unchanged
