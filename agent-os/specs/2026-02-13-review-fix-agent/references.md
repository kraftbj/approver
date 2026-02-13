# Review Fix Agent — References

## Key Files

- `internal/claude/runner.go` — RunAgent function reused by fix agent
- `internal/claude/review.go` — ReviewResult and Issue types consumed by FixItemsFromReview
- `internal/app/app.go` — Main Bubble Tea model, state machine, Update loop
- `internal/app/helpers.go` — Overlay renderers and helper methods
- `internal/app/pr_screen.go` — PR screen key handlers and view dispatch
- `internal/ui/prdetail.go` — Detail panel view methods

## Patterns

- Review pipeline: `startReview()` → `runClaudeReviewCmd()` → `claudeReviewDoneMsg`
- Fix pipeline: `startFix()` → `runFixCmd()` → `fixDoneMsg`
- Overlay pattern: `stateConfirm` / `stateInput` → `viewConfirmOverlay()` / `viewInputOverlay()`
- Fix overlay: `stateFixSelect` → `viewFixSelectOverlay()`
