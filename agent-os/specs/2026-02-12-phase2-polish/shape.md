# Shape

## Review Panel Hint Fix
- `ViewReview()` accepts `hasWorktree` param
- Shows "press w first" when no worktree and no review data

## Worktree Spinner
- `creatingWorktrees map[int]bool` on home struct
- `IsCreatingWorktree` runtime field on PR struct
- `W` indicator in PR list, "creating..." in detail panel
- Spinner ticks while any worktree is being created

## Auto-Create Worktree
- `pendingAction` enum: none, review, tmux
- `pendingPR` tracks which PR the action is for
- `c`/`t` without worktree sets pending + fires createWorktreeCmd
- `worktreeCreatedMsg` checks pending and dispatches

## Fun Review Spinner
- `reviewSpinnerTickMsg` fires every 4s
- Rotates through ~10 playful messages
- Updates `reviewStep` for all reviewing PRs

## Approve from App
- `A` key shows "Approve PR #N? (y/n)" confirm dialog
- `gh pr review <N> --approve` via command
- Success refreshes PR to update review decision

## Per-Repo Setup
- `Repos map[string]RepoConfig` in config with `setup_command`
- Substring match on remote URL
- Runs after worktree creation, before pending action dispatch
