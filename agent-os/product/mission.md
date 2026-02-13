# Product Mission

## Problem

GitHub workflows are fragmented. Developers context-switch between review queues, issue trackers, and their terminal — losing focus and momentum. PR reviews pile up, assigned issues lack workspace tooling, and there's no unified view of what needs attention. Existing tools are either closed-source (Conductor.build), focused on development sessions rather than workflow management (Claude Squad), or require leaving the terminal (GitHub web UI).

## Target Users

Developers who live in the terminal and want a single hub for their GitHub work — reviewing PRs, working on issues, and tracking important activity — with AI assistance built into every workflow. The primary user has a queue of review requests, a backlog of assigned issues, and PRs they want to keep an eye on.

## Solution

Approver is a terminal UI that serves as a GitHub AI workflow hub with three screens:

1. **Reviews** — PRs requesting your review, with AI-powered code review and one-key approve/request-changes
2. **Issues** — Assigned issues with workspace creation, AI-assisted coding (collaborative and autonomous modes)
3. **Watchlist** — Manually-tracked PRs with AI summaries of activity

Key differentiators:

- **Three-screen workflow hub** — Reviews, Issues, and Watchlist in one keyboard-driven interface
- **AI-powered across all workflows** — Claude Code integration for review, coding, and summarization
- **Workspace management** — Git worktrees for both PR review and issue development
- **Lightweight and extensible** — Open-source Go TUI, single binary, stays in your dev flow
