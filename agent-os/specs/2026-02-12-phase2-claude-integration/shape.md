# Phase 2: Claude Code Integration — Shape

## Scope

Review-only half of the 6-agent swarm (agents 1-3). No fixing, no validation, no auto-approval. The numbered checklist from Agent 3 is the final deliverable.

## Decisions

- **3 agents, not 6**: Approver is a review tool, not a fix tool. Agents 4-6 (fixer, validator, final reviewer) are out of scope.
- **Plain text output**: `claude -p` default output. No JSON parsing of Claude responses — just display the text.
- **Best-effort parsing**: Parse issues from Agent 3 output when possible, but always fall back to raw text display.
- **Worktree required**: Both `c` and `t` require a worktree. The worktree gives Claude full repo context.
- **tmux for interactive only**: `c` (headless) works without tmux. `t` requires tmux.
- **Read-only tools**: Review agents get `Read,Glob,Grep` only — no writes to the codebase.
- **Cancellable**: Pressing `c` while a review is running cancels it.
- **One review at a time per PR**: No stacking reviews.

## What's NOT in scope

- Auto-posting review comments to GitHub
- Fix suggestions or auto-fix
- Custom agent prompts per-PR (only global config)
- Review history/persistence across sessions
