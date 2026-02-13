# Product Mission

## Problem

PR reviews pile up. Developers lose track of which PRs need their attention, context-switch between GitHub tabs and their terminal, and lack efficient tooling to triage, review, and respond to review requests. Existing tools like Conductor.build are closed-source and not extensible, while Claude Squad focuses on development sessions rather than review workflows.

## Target Users

Developers who regularly review pull requests, especially those already using Claude Code in their workflow. The primary user is someone with a queue of review requests who wants to work through them efficiently without leaving the terminal.

## Solution

Approver is a terminal UI purpose-built for PR review management with Claude Code integration. It combines GitHub PR fetching, git worktree management, AI-powered code review, and review posting into a single keyboard-driven workflow. The key differentiators:

- **Review-first design** - Built around the reviewer's workflow, not the author's. Keeps reviews moving.
- **Claude Code integration** - AI-assisted review with structured output that maps directly to GitHub inline comments.
- **Lightweight and extensible** - Open-source Go TUI, single binary, stays in your dev flow.
- **Secondary: own-PR tracking** - Also helps you know which of your own PRs need action from you.

Open to building on Claude Squad's foundation if it provides real benefits, but the core value is a focused review management tool.
