# Multi-Repo Support — Shape

## Problem

Approver operates on a single repo (`os.Getwd()` at startup). Users who work across multiple repos must restart Approver in each directory. Separately, merged PRs on the watchlist have no visual indicator and never disappear.

## Appetite

Two features, can be done as separate PRs:
1. Merged PR detection + cleanup (small, independent)
2. Multi-repo support (big)

## Solution

### Merged PR Detection
- Add `state` field to PR (gh returns OPEN/CLOSED/MERGED)
- Show purple MERGED badge in watchlist
- Auto-remove merged PRs on second poll cycle (first cycle shows badge)
- Clean up worktree when removing merged PR

### Multi-Repo Support
- New `repo_sources` config: explicit paths + scan directories
- `RepoEntry` type: name (dir basename) + absolute path
- `ItemKey{Repo, Number}` replaces bare `int` in all state maps
- Per-repo worktree managers with namespaced paths
- Grouped list rendering with repo headers (hidden when single repo)
- Parallel fetch/poll across all configured repos
- Namespaced persistence (reviews, tracked PRs/issues)

## Rabbit Holes
- No repo discovery beyond config — users explicitly list repos
- No cross-repo PR relationships
- No per-repo config overrides beyond setup_command (already exists)

## No-Gos
- Repo auto-detection from GitHub org membership
- Nested scan_dir recursion (1 level only)
