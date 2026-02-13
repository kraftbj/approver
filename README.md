# Approver

A terminal UI for managing GitHub PR reviews, assigned issues, and tracked PRs — with Claude AI integration for automated code review and interactive coding sessions.

## Requirements

- **Go 1.21+** (uses builtin `min`/`max`)
- **[GitHub CLI (`gh`)](https://cli.github.com/)** — authenticated with `gh auth login`
- **git** — for worktree management
- **[Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code)** — optional, for AI review and tmux sessions
- **tmux** — optional, for interactive Claude sessions

Works on **macOS**, **Linux**, and **Windows**.

## Install

```
go install github.com/kraft/approver@latest
```

Or build from source:

```
git clone https://github.com/kraftbj/approver.git
cd approver
go build -o approver .
```

## Usage

Run `approver` from any git repository with a GitHub remote:

```
cd your-repo
approver
```

### Screens

Switch between screens with `1`, `2`, `3`:

| Screen | Description |
|--------|-------------|
| **1: Reviews** | PRs requesting your review |
| **2: Issues** | Your assigned issues |
| **3: Watchlist** | Manually-tracked PRs |

### Key Bindings

**Navigation:**
| Key | Action |
|-----|--------|
| `j`/`k`, up/down | Navigate list |
| `Tab` | Cycle detail panel view |
| `o` | Open in browser |
| `R` | Refresh |
| `?` | Help |
| `q` | Quit |

**Reviews:**
| Key | Action |
|-----|--------|
| `w` / `W` | Create / delete worktree |
| `c` | Start Claude AI review (auto-creates worktree) |
| `t` | Open Claude tmux session (auto-creates worktree) |
| `A` | Approve PR (with confirmation) |
| `X` | Request changes (with reason) |
| `u` | Update branch (merge base into worktree) |

**Issues:**
| Key | Action |
|-----|--------|
| `w` / `W` | Create / delete worktree |
| `c` | Start Claude tmux session (auto-creates worktree) |
| `p` | Open linked PR in browser |
| `a` / `d` | Add / remove manually-tracked issue |

## Configuration

Optional config file at `~/.config/approver/config.yaml`:

```yaml
# Max PRs to fetch (default: 50)
pr_limit: 100

# Background refresh interval in seconds (default: 300, 0 to disable)
poll_interval: 300

# Override worktree directory (default: ../approver-worktrees/)
worktree_dir: ~/worktrees

# Custom prompt for AI review
review_prompt: "Focus on security and error handling"

# Tools the AI review agents can use (default: Read,Glob,Grep)
allowed_tools: "Read,Glob,Grep,Bash"

# Per-repo setup commands (matched by remote URL substring)
repos:
  my-org/my-repo:
    setup_command: "npm install"
```

## License

GPLv2 or later. See [LICENSE](LICENSE).
