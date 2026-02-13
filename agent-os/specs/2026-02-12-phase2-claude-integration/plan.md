# Phase 2: Claude Code Integration — Plan

## Overview

Add Claude Code integration to Approver: a 3-agent headless review pipeline (`c` key), review output display in the detail panel, and tmux-based interactive Claude sessions (`t` key).

## Review Pipeline

```
Pre-step (Go): Fetch existing PR reviews via gh api
         |
    +---------+---------+
    v                   v
Agent 1 (claude -p)  Agent 2 (claude -p)
Code review          Validate existing reviews
in worktree          in worktree
    |                   |
    +---------+---------+
              v
      Agent 3 (claude -p)
      Compare both inputs,
      produce NUMBERED CHECKLIST
```

- Agents 1 and 2 run concurrently
- Agent 3 runs after both, receives their outputs
- Final output: numbered checklist in detail panel

## Key Bindings

| Key | Action |
|-----|--------|
| `c` | Start/cancel headless review pipeline |
| `t` | Open interactive Claude tmux session |
| `Tab` | Toggle detail panel between PR info and review |

## New Packages

- `internal/claude/` — review pipeline runner, tmux session management

## Modified Packages

- `internal/app/` — new messages, state fields, key handlers
- `internal/config/` — review prompt, budget, allowed tools
- `internal/ui/` — review display, menu updates, PR indicators
- `internal/github/` — runtime state fields on PR struct
