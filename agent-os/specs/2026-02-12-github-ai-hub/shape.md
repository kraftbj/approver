# Shaping Decisions: GitHub AI Hub

## Screen Switching

**Decision**: `1`/`2`/`3` number keys switch between Reviews, Issues, Watchlist.

**Alternatives considered**:
- Tab cycling through screens — rejected because Tab already cycles detail modes within a screen
- Arrow left/right — rejected because horizontal arrows could be useful for other navigation
- Named shortcuts (r/i/w) — rejected because these conflict with existing keybindings

## Architecture: Screen Interface vs. Separate Models

**Decision**: `Screen` interface with `HandleKey(h *home, key string) tea.Cmd` and `View(h *home, height int) string`. The `home` struct stays as the single tea.Model dispatcher. All domain state (maps, config, etc.) stays in `home`.

**Rationale**: Screens need access to shared state (worktrees, config, spinner). Passing `*home` to screen methods is simpler than duplicating state or using a context object. Message handlers stay in `home.Update()` because messages are screen-agnostic (e.g., worktreeCreatedMsg could come from any screen).

**Alternatives considered**:
- Separate tea.Model per screen — rejected because too much state duplication and message routing complexity
- State context object passed to screens — over-engineered for three screens

## PR-Specific State Extraction

**Decision**: `detailMode`, `reviewMsgIdx`, `prList`, and `detail` move into `prScreen` struct. Global state (worktrees map, reviews map, config, errBox, menu, spinner, state machine) stays in `home`.

**Rationale**: Each screen has its own list/detail UI state, but worktrees, reviews, and tmux sessions are shared resources.

## Issue Branches

**Decision**: Auto-named `issue-{number}-{sanitized-title}` created from the repo's default branch. Extend `worktree.Manager` with a `CreateFromBase` method.

**Alternatives considered**:
- Prompt for branch name — too much friction for the common case
- Use issue number only — not descriptive enough in `git branch` output

## Issue Source

**Decision**: Auto-fetch assigned issues via `gh issue list --assignee @me` plus manual additions via `a` key.

**Rationale**: Matches the PR pattern (auto-fetch review-requested + manual add).

## Watchlist: Merge Manual Tracking

**Decision**: Move manually-tracked PRs from Reviews screen to Watchlist. Reviews becomes review-requests only. Watchlist owns `tracked.json`.

**Rationale**: The Reviews screen's `a`/`d` keys for manual tracking always felt bolted-on. Watchlist is the natural home for "PRs I want to keep an eye on." This also simplifies the Reviews screen.

## Watchlist Scope

**Decision**: PRs only for now. Cross-repo PRs and issues deferred.

**Rationale**: Cross-repo requires a different fetch strategy and config model. Ship the simpler version first.

## AI Modes for Issues

**Decision**: Two modes matching the PR screen:
- `c` — collaborative (tmux session with Claude in the worktree)
- `C` — autonomous (headless Claude session)

**Rationale**: Reuses the existing tmux and Claude infrastructure. Collaborative mode is primary for issues since you're usually writing code, not reviewing.

## Naming

**Decision**: Keep "Approver" for now, defer rename.

**Rationale**: Renaming is a distraction from shipping. The name can change later without affecting architecture.
