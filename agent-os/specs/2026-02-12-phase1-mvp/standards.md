# Applicable Standards

## 1. Go Coding Style

- Use `gofmt` / `goimports` for formatting
- Follow effective Go conventions
- Keep packages focused: one responsibility per package
- Error handling: return errors, don't panic
- Use internal/ for non-exported packages

## 2. CLI/TUI Conventions

- Bubble Tea: single model architecture with message passing
- Lip Gloss: declarative styling, no ANSI escape codes
- Cobra: command parsing with subcommands
- Exit cleanly on q/ctrl+c
- Handle terminal resize (WindowSizeMsg)

## 3. External Process Execution

- Shell out to `gh` and `git` for GitHub/git operations
- Capture stdout and stderr separately
- Parse JSON output from gh (not text scraping)
- Handle command-not-found gracefully
- Respect user's existing gh authentication

## 4. Configuration and Persistence

- XDG-style paths: ~/.config/approver/
- YAML for user-editable config
- JSON for machine-managed state (tracked PRs)
- Sensible defaults for all config values
- Config file is optional - app works without it
