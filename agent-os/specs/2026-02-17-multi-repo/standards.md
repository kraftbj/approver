# Multi-Repo Support — Standards

## Backward Compatibility
- If `repo_sources` is empty/absent, fall back to CWD (single-repo mode)
- Single-repo mode: no repo headers in lists, behavior identical to before
- Old flat review files loaded as belonging to single configured repo
- Old worktree paths (no repo subdir) migrated on first run

## State Map Convention
- All state maps use `ItemKey{Repo, Number}` as key
- Helper functions `PRKey(pr)` and `IssueKey(issue)` for consistent key creation
- Messages carry `repo string` or `key ItemKey` field

## Naming
- `RepoSource` — config input (path or scan_dir)
- `RepoEntry` — resolved repo (name + absolute dir)
- `ItemKey` — composite key for state lookups
- Repo name = directory basename (e.g., "approver", "jetpack")

## Worktree Paths
- Multi-repo: `{baseDir}/{repoName}/pr-{number}-{branch}/`
- Single-repo migration: move old paths into repo subdir

## Config Format
```yaml
repo_sources:
  - path: ~/code/approver
  - path: ~/code/jetpack
  - scan_dir: ~/code/a8c/
```

## Testing
- Config resolution tested with temp directories
- Build verification: `go build -o approver .`
- Test suite: `go test ./...`
