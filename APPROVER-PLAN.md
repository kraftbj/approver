# Approver - PR Review TUI

## Context

**Approver** is a terminal UI for managing GitHub PR reviews with Claude Code. The name emphasizes the positive outcome - you sit down, review efficiently, and approve. The tool should:
- Fetch all PRs where your review is requested
- Create git worktrees for each PR
- Run Claude Code reviews on demand per-PR
- Poll GitHub periodically for status updates (new comments, mergeability changes, CI status)
- Draft inline review comments you can approve/edit before posting
- Post reviews back to GitHub via the API

**Conductor.build** does the workspace/worktree/UI part well but is closed-source, not extensible for this use case, and lacks PR review automation. **Claude Squad** (Go, AGPL-3.0, 6k stars) is the closest open-source project - it manages Claude Code sessions with worktrees in a TUI - but it's focused on development, not PR review management.

**Decision: Build a new TUI from scratch**, starting simple and growing incrementally. A TUI is the right starting point - lightweight, stays in your dev flow, and can be evolved into a native macOS app later.

## Architecture

```
approver/
├── main.go                     # Entry point
├── go.mod
├── internal/
│   ├── app/                    # Application state, keybindings, main model
│   │   └── app.go
│   ├── ui/                     # Bubble Tea views
│   │   ├── dashboard.go        # PR list view (main screen)
│   │   ├── review.go           # Single PR review detail view
│   │   ├── diff.go             # Diff viewer with inline comments
│   │   ├── terminal.go         # Embedded terminal (Claude Code session)
│   │   ├── status.go           # Status bar (polling status, counts)
│   │   └── comments.go         # Draft comment editor
│   ├── github/                 # GitHub integration (via gh CLI)
│   │   ├── prs.go              # Fetch PRs, reviews, comments
│   │   ├── reviews.go          # Post reviews, inline comments
│   │   └── poller.go           # Background polling for updates
│   ├── worktree/               # Git worktree management
│   │   ├── manager.go          # Create/list/delete worktrees
│   │   └── setup.go            # Run setup scripts per-worktree
│   └── claude/                 # Claude Code integration
│       ├── session.go          # Start/manage claude -p sessions
│       └── review.go           # PR review prompt templates
└── config/
    └── config.go               # User preferences, poll interval, etc.
```

## Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | **Go** | Battle-tested for TUIs (Claude Squad, lazygit, k9s all use it). Fast compilation, single binary distribution. |
| TUI framework | **Bubble Tea** | The standard Go TUI framework. Rich component ecosystem (Lip Gloss for styling, Bubbles for common components). |
| GitHub API | **`gh` CLI** | Already authenticated, handles tokens, supports all operations including inline review comments. |
| Claude integration | **`claude -p`** | Headless mode, structured JSON output, tool approval via `--allowedTools`. |
| Git operations | **`git` CLI** | Direct git commands for worktree management. |
| Config | **YAML file** | `~/.config/approver/config.yaml` for user preferences. |

## Feature Breakdown (Incremental)

### Phase 1: Core MVP
1. **PR Dashboard** - Fetch and display PRs requesting your review (`gh pr list --search "review-requested:@me"`)
2. **Worktree Management** - Create/delete worktrees for each PR branch
3. **Status Display** - Show PR title, author, branch, CI status, review state, labels
4. **Keyboard Navigation** - j/k to navigate, Enter to open detail, q to quit

### Phase 2: Claude Code Integration
5. **Manual Claude Review** - Trigger `claude -p` in a PR's worktree with a review prompt
6. **Review Output** - Display Claude's analysis in a panel (security, performance, quality, architecture)
7. **Embedded Terminal** - Attach to a Claude Code session for interactive review (via tmux)

### Phase 3: GitHub Feedback Loop
8. **Background Poller** - Poll GitHub every N minutes for new comments, CI status changes, mergeability
9. **Inline Comment Drafting** - Create draft inline comments on specific files/lines
10. **Review Posting** - Submit draft reviews (approve/request changes/comment) via `gh api`
11. **Comment Viewer** - Show existing PR comments and conversation threads

### Phase 4: Polish
12. **Setup Scripts** - Configurable per-repo setup command (like Conductor's setup script)
13. **Notifications** - Visual indicators when PR state changes (new comment, CI passed/failed)
14. **Multi-Repo** - Support reviewing PRs across multiple repos
15. **Persistence** - Remember which PRs you've already reviewed, draft comments across sessions

## Key Implementation Details

### Fetching PRs with review requests
```bash
gh pr list --search "review-requested:@me" \
  --json number,title,author,headRefName,url,reviewDecision,statusCheckRollup,labels,additions,deletions,updatedAt \
  --limit 50
```

### Inline review comments via GitHub API
```bash
# Create a review with inline comments
gh api repos/{owner}/{repo}/pulls/{pr}/reviews \
  --method POST \
  -f event="COMMENT" \
  -f body="Overall review summary" \
  --input comments.json
```

Where `comments.json` contains line-level comments:
```json
{
  "comments": [
    {
      "path": "src/auth.py",
      "line": 42,
      "body": "Potential SQL injection here - use parameterized queries"
    }
  ]
}
```

### Claude Code review prompt
```bash
claude -p "Review PR #${PR_NUM}. The diff is:
$(gh pr diff ${PR_NUM})

Analyze for:
1. Security vulnerabilities (with file:line references)
2. Performance issues
3. Code quality and maintainability
4. Missing tests or edge cases
5. Architectural concerns

Output as JSON with this schema: {summary, severity, comments: [{file, line, category, message}]}" \
  --output-format json \
  --json-schema '...' \
  --allowedTools "Read,Glob,Grep" \
  --cwd "${WORKTREE_PATH}"
```

The structured JSON output lets us directly map Claude's inline comments to GitHub API review comments.

### Background polling
A goroutine runs on a configurable interval (default: 5 min) and fetches:
- New comments on active PRs
- CI status changes
- Mergeability updates
- New review requests

Updates are pushed to the TUI via Bubble Tea's `tea.Cmd` message system.

## Project Setup

1. Clear `/Users/kraft/code/conductor/` (rm contents + .git, per user request)
2. Run `~/agent-os/scripts/project-install.sh` to initialize the project scaffold
3. `go mod init github.com/kraft/approver`
4. `go get github.com/charmbracelet/bubbletea`
5. `go get github.com/charmbracelet/lipgloss`
6. `go get github.com/charmbracelet/bubbles`

Binary name: `approver`

## Verification

- `go build -o approver && ./approver` launches the TUI
- Dashboard shows PRs from `gh pr list --search "review-requested:@me"`
- Selecting a PR and pressing `w` creates a worktree, `r` triggers a Claude review
- Review results display inline with file/line references
- `c` opens comment drafter, `S` submits the review to GitHub
- Background polling indicator shows last poll time and update count
