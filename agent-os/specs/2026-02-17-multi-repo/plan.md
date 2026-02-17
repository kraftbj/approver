# Multi-Repo Support — Implementation Plan

## Phase 1: Merged PR Detection (independent)
1. Add `state` to `ghPRFields` and `PR` struct
2. MERGED badge in watchlist list rendering
3. `mergedSeen` map + auto-removal logic in watchlist poll handler
4. Worktree cleanup on merged PR removal

## Phase 2: Config + Types
1. New `internal/config/repos.go` — RepoSource, RepoEntry, ResolveRepoDirs
2. Extend Config with `RepoSources` field
3. Add `Repo`/`RepoDir` fields to PR and Issue types

## Phase 3: ItemKey Refactor (largest change)
1. Define `ItemKey{Repo, Number}` in `internal/app/itemkey.go`
2. Migrate all `map[int]` state maps to `map[ItemKey]`
3. Add repo/key to all message types
4. Update all command functions with repoName parameter
5. Update all handlers to use ItemKey lookups
6. Update reconciliation and helper functions

## Phase 4: Worktree Namespacing
1. Add `RepoName` to worktree Manager
2. Namespace paths: `{baseDir}/{repoName}/pr-{number}-{branch}/`
3. Update ScanExisting to scan under repo subdir

## Phase 5: UI + Fetching
1. Grouped list rendering with repo headers
2. Multi-repo fetch/poll in Run()/Init()
3. Parallel commands per repo

## Phase 6: Persistence
1. Use Repo field in tracked PRs/issues
2. Namespace review files: `reviews/{repoName}/review-{number}.json`
3. Backward compat for single-repo flat files
