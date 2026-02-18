# GHE Support — Code References

## Key locations

- `internal/github/prs.go:CheckGH()` — the broken auth check (replaced with `CheckGHInstalled` + `CheckGHHosts`)
- `internal/github/prs.go:ParseRepoFromDir()` — existing function that gets `git remote get-url origin` output
- `internal/app/app.go:Run()` — startup flow where auth is checked and repos resolved
- `internal/config/repos.go:RepoSource` — config struct for user-defined repo locations
- `internal/config/repos.go:RepoEntry` — resolved repo struct used at runtime
- `internal/config/repos.go:ResolveRepoDirs()` — resolves `RepoSource` list into `RepoEntry` list

## How gh resolves hosts

All `gh` commands in the codebase use `cmd.Dir = repoDir`, which causes `gh` to read the repo's git remote and auto-resolve the correct GitHub host. This means PR fetching, approvals, and other operations already work with GHE — only the startup auth check was broken.

## Config format (after change)

```yaml
repo_sources:
  - path: ~/code/my-ghe-repo
    host: github.example.com
  - scan_dir: ~/code/oss
    # host omitted — auto-detected from git remote
```
