# Phase 2: Standards

## Error Handling

- Shell commands (`claude -p`, `gh`, `tmux`) may fail — always capture stderr and return meaningful errors.
- Context cancellation must be respected: check `ctx.Err()` before starting each pipeline step.
- Never panic on parse failures — fall back to raw text display.

## Coding Style

- Follow existing patterns: tea.Cmd for async work, tea.Msg for results.
- New package `internal/claude/` follows same conventions as `internal/worktree/`.
- Shell out to CLIs directly (`exec.Command`), no SDK wrappers.
- Keep prompts as constants with `fmt.Sprintf` for variable substitution.

## Testing

- Test parsing logic (checklist extraction) with fixture strings.
- Test command construction (correct args, working directory), not execution.
- Skip tests requiring `claude`/`tmux` in CI environments.
