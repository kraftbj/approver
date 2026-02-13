# Reference Implementations

## `internal/app/app.go` — Monolithic Model (becomes dispatcher)

The current `home` struct is the single tea.Model. It holds all state (PR list, worktrees, reviews, config) and handles all messages in `Update()`. The multi-screen refactor keeps `home` as the dispatcher but extracts screen-specific key handling and rendering into `Screen` implementations.

Key patterns to preserve:
- `newHome()` initializes all maps and UI components
- `Init()` fires initial fetch commands in parallel
- `Update()` handles all message types centrally
- `View()` composes menu + content + error bar

## `internal/app/app.go` handleDefaultKey — Key Handler Pattern

Each screen gets its own key handler following this pattern:
- Switch on key string
- Access shared state via `*home` pointer
- Return `tea.Cmd` for async operations, `nil` for sync-only updates
- Compose multiple commands with `tea.Batch()`

## `internal/ui/prlist.go` / `prdetail.go` — List + Detail Pattern

The two-panel layout used by the PR screen:
- List component: `SetSize()`, `SetItems()`, `MoveUp()`, `MoveDown()`, `Selected()`, `View()`
- Detail component: `SetSize()`, `View(item)` with multiple view modes
- 30/70 width split via `lipgloss.JoinHorizontal()`
- Scroll offset tracking with `clampScroll()`

The Issues screen follows this exact pattern with `IssueList` and `IssueDetail`.

## `internal/app/tracked.go` — File Persistence Pattern

JSON file persistence for manually-tracked items:
- `trackedFilePath()` returns `~/.config/approver/tracked.json`
- `loadTrackedPRs()` / `saveTrackedPRs()` for read/write
- `addTrackedPR()` / `removeTrackedPR()` for mutations with dedup
- `fetchTrackedPRsCmd()` as tea.Cmd that fetches full data for each tracked item

Watchlist follows this pattern with `tracked.json`. Issues follow it with `tracked-issues.json`.

## `internal/github/prs.go` — gh CLI Fetch Pattern

Shell out to `gh` CLI for GitHub API access:
- `exec.Command("gh", ...)` with `cmd.Dir = repoDir`
- `--json` flag with field list for structured output
- `json.Unmarshal` into typed structs
- Error handling: check `*exec.ExitError` for stderr

`issues.go` follows this pattern using `gh issue list` and `gh issue view`.

## `internal/github/types.go` — Type + Runtime Field Pattern

GitHub types with both JSON-mapped and runtime fields:
- JSON fields use struct tags: `json:"number"`
- Runtime fields use `json:"-"`: `HasWorktree`, `IsReviewing`, etc.
- Runtime fields are reconciled by `home` after state changes
- Helper methods on the type: `CIStatus()`, `ReviewIcon()`, `SizeString()`

The `Issue` type follows this pattern with `HasWorkspace`, `IsWorking`, `HasTmux`.
