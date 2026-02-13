# AGENTS.md

Instructions for AI coding agents working on the Approver codebase.

## Project Overview

Approver is a terminal UI for managing GitHub PR reviews. Built with Go and Bubble Tea, it fetches PRs where your review is requested, creates git worktrees for local review, and provides keyboard-driven navigation.

## Setup Commands

| Command | Description |
|---------|-------------|
| `go build -o approver .` | Build the binary |
| `go test ./...` | Run all tests |
| `go test ./internal/github/...` | Run tests for a specific package |
| `make build` | Build via Makefile |
| `make run` | Build and run |

## Code Style

- Use `gofmt` / `goimports` for formatting
- Follow effective Go conventions
- Keep packages focused: one responsibility per package
- Error handling: return errors, don't panic
- Use `internal/` for non-exported packages

## Architecture

### Package Structure

- **internal/app**: Application state, Bubble Tea model, keybindings, message types
- **internal/ui**: View components (PR list, detail panel, menu, overlays, styles)
- **internal/github**: GitHub integration via `gh` CLI (PR fetching, types)
- **internal/worktree**: Git worktree management (create, delete, scan)
- **internal/config**: YAML configuration loading and defaults

### Key Patterns

- Single Bubble Tea model architecture (all state in `home` struct)
- Message-based updates via tea.Cmd/tea.Msg
- Shell out to `gh` and `git` for external operations (no SDK wrappers)
- Lip Gloss for declarative styling
- Two-panel layout: list (30%) + detail (70%)

## Testing

- Use real JSON fixtures for gh output parsing tests
- Test command construction, not execution (avoid requiring gh/git in CI)
- Run `go test ./...` before submitting

## PR Guidelines

- Submit PRs against the `main` branch
- Run `go test ./...` before pushing
- Keep PRs focused: one logical change per PR
