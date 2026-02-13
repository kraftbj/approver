# Tech Stack

## Language

**Go** - Battle-tested for TUIs (Claude Squad, lazygit, k9s). Fast compilation, single binary distribution.

## TUI Framework

**Bubble Tea** - The standard Go TUI framework. Rich component ecosystem with Lip Gloss for styling and Bubbles for common components.

## GitHub Integration

**`gh` CLI** - Already authenticated, handles tokens, supports all operations including inline review comments. No need to manage OAuth tokens or API keys separately.

## AI Integration

**Claude Code (`claude -p`)** - Headless mode with structured JSON output, tool approval via `--allowedTools`. Runs in PR worktrees with access to full repo context.

## Git Operations

**`git` CLI** - Direct git commands for worktree creation, deletion, and management.

## Configuration

**YAML** - User preferences stored at `~/.config/approver/config.yaml`. Covers poll interval, review prompt customization, per-repo setup scripts.
