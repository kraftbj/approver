# Reference Analysis

## Claude Squad (github.com/smtg-ai/claude-squad)

Go TUI for managing Claude Code sessions with git worktrees. AGPL-3.0, ~6k stars.

### Relevant patterns for Approver:
- **Single Bubble Tea model** (`app/app.go`): Proven architecture. All state in one struct, message-based updates. Approver adopts this.
- **Worktree management** (`session/git/worktree.go`): Create/delete patterns with git CLI. Path conventions and cleanup logic.
- **List component** (`ui/list.go`): Scrollable list with selection, j/k navigation. Two-line item rendering.
- **Overlay compositing** (`ui/overlay/`): Modal dialogs rendered on top of main view. Used for confirmation dialogs, help screens.
- **Terminal resize handling**: Responsive layout that adjusts to window size changes.

### Not applicable to Phase 1:
- Tmux session management (Claude Code integration is Phase 2)
- Tabbed panel views (single detail panel is sufficient)
- Session persistence (worktree state is filesystem-based)

## Conductor.build

Closed-source commercial tool for PR review management.

### Features observed:
- PR queue with status indicators (CI, review state, labels)
- Worktree creation per PR
- Integrated diff viewer
- Review posting from within the tool
- Multi-repo support

### Approver differentiators:
- Open source and extensible
- Claude Code integration (Phase 2)
- Terminal-native (no Electron/web overhead)
- Focused on the review workflow specifically

### What we borrow conceptually:
- Two-panel layout (list + detail)
- PR status display format (CI icons, review state, size)
- Worktree-per-PR model
- Keyboard-first navigation
