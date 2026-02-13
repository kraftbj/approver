# Phase 2: References

## PR Review Swarm Prompt

Source: `~/code/kraftbj/pr-review-swarm-prompt.md`

The 6-agent pipeline that inspired this design. Phase 2 implements agents 1-3 (review half). Key pattern preserved: the "checklist as contract" — Agent 3's numbered checklist is the authoritative output.

## Claude Code CLI

- `claude -p "<prompt>"` — headless mode, prints response to stdout
- `--allowedTools "Read,Glob,Grep"` — restrict to read-only tools
- `--cwd <path>` — set working directory (worktree path)

## tmux

- `tmux new-session -d -s <name> -c <path> claude` — create detached session running Claude
- `tmux attach-session -t <name>` — attach to session (hand off terminal)
- `tmux has-session -t <name>` — check if session exists
- `tmux kill-session -t <name>` — cleanup
