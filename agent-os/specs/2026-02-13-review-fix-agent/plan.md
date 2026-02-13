# Review Fix Agent — Plan

## Implementation Tasks

1. **FixItem type + RunFixAgent** — `internal/claude/fixer.go` with FixItem, conversion helpers, and agent runner
2. **Messages + commands** — New Bubble Tea message types and tea.Cmd functions
3. **State management** — New state (stateFixSelect), fields on home struct, fix selection key handler
4. **F key handler** — Build fix items from reviews + comments, enter selection overlay
5. **Detail panel** — ViewFixResult, ViewFixing, ViewFixEmpty methods; detailFix mode
6. **Config** — fix_allowed_tools option with default "Read,Write,Edit,Glob,Grep"
7. **Commit+push flow** — fixCommitPushCmd with git add, commit, push sequence
8. **Help text + hints** — F key in help overlay and bottom menu hints

## Files Changed

- New: `internal/claude/fixer.go`
- Modified: `internal/app/app.go`, `messages.go`, `commands.go`, `pr_screen.go`, `helpers.go`
- Modified: `internal/ui/prdetail.go`, `menu.go`
- Modified: `internal/config/config.go`
- Modified: `agent-os/product/roadmap.md`
