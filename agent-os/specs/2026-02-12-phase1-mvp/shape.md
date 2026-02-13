# Approver Phase 1 - Shaping Notes

## Problem Statement

PR reviews pile up. Developers lose track of which PRs need their attention, context-switch between GitHub tabs and their terminal, and lack efficient tooling to triage and review. The tool that does this best (Conductor.build) is closed-source and not extensible.

## Appetite

Phase 1 is a focused MVP: dashboard + worktrees + navigation. No Claude Code integration, no review posting, no background polling. Those are Phase 2+.

## Solution Shape

A two-panel terminal UI:
- Left panel (30%): scrollable list of PRs needing review
- Right panel (70%): detail view of selected PR
- Bottom bar: context-sensitive keybindings

Key interactions:
- j/k: navigate PR list
- w: create worktree for selected PR
- W: delete worktree (with confirmation)
- o: open PR in browser
- a: add a PR manually by number/URL
- d: remove manually-tracked PR
- R: refresh PR list
- ?: help overlay
- q: quit

## Rabbit Holes to Avoid

- No background polling in Phase 1 (complexity, rate limits)
- No diff viewer (use browser or editor)
- No review posting (Phase 3)
- No multi-repo support (Phase 4)
- Don't build an SDK wrapper around gh - just shell out

## Key Trade-offs

**Single model vs component models**: Single model (like Claude Squad) is simpler and proven. Component models add indirection without benefit at this scale.

**Two panels vs three**: Three panels (list/detail/terminal) is too cramped. Two panels with overlays for modals is the right balance.

**Worktree location**: Using ~/.config/approver/worktrees/ keeps the user's repo clean. The alternative (sibling directories) pollutes the workspace.

**Manual PR tracking**: Essential for tracking your own PRs or ones you want to pick up before they're formally assigned. Simple JSON persistence is sufficient.

## No-gos

- No auto-approve or auto-merge
- No credential storage (rely on gh auth)
- No modification of user's git config
- No daemon/background process
