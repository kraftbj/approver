# GitHub Enterprise Support

## Context

When Approver is pointed at a GHE repo, nothing resolves. The root cause is `CheckGH()` at `internal/github/prs.go:107` — it runs `gh auth token` which only checks the default host (github.com). All actual `gh` commands already work with GHE because they use `cmd.Dir = repoDir`, which lets `gh` auto-resolve the host from the repo's remote. The fix is: detect the host per repo, validate auth per host.

## Task 1: Save spec documentation

Create `agent-os/specs/2026-02-18-ghe-support/` with:
- `plan.md` — this plan
- `shape.md` — shaping notes
- `references.md` — code references

## Task 2: Add `Host` field to config types

**Files:** `internal/config/repos.go`

- Add `Host string` field to `RepoSource` (yaml tag: `host,omitempty`)
- Add `Host string` field to `RepoEntry`
- In `ResolveRepoDirs()`, propagate `src.Host` to each `RepoEntry`. For `ScanDir` entries, all child repos inherit the parent's `Host`. Leave empty if not set (auto-detection happens in app.go).

## Task 3: Add `ParseHostFromRemote` utility

**File:** `internal/github/prs.go`

Add `DefaultHost = "github.com"` constant and `ParseHostFromRemote(remoteURL string) string` that extracts hostname from SSH (`git@host:org/repo.git`) and HTTPS (`https://host/org/repo.git`) URLs. Returns `DefaultHost` on failure.

Reuses existing `ParseRepoFromDir()` at `prs.go:136` which already calls `git remote get-url origin`.

## Task 4: Replace `CheckGH()` with per-host auth

**Files:** `internal/github/prs.go`, `internal/app/app.go`

In `prs.go`:
- Add `CheckGHInstalled()` — just the `exec.LookPath("gh")` check
- Add `CheckGHHosts(hosts []string) error` — loops over hosts, runs `gh auth token --hostname <host>` for each, returns error naming the failing host
- Remove old `CheckGH()`

In `app.go` `Run()`:
1. Replace `gh.CheckGH()` with `gh.CheckGHInstalled()`
2. After repo resolution, add host-detection loop:
   - For each repo where `Host == ""`, call `gh.ParseRepoFromDir(repo.Dir)` then `gh.ParseHostFromRemote(url)` to populate it
   - For the CWD fallback path, also detect host
3. After host detection, collect unique hosts and call `gh.CheckGHHosts(hosts)`

## Task 5: Add tests

**Files:** `internal/github/prs_test.go`

Table-driven test for `ParseHostFromRemote`:
- SSH github.com, SSH GHE, HTTPS github.com, HTTPS GHE, HTTPS without .git suffix, empty string, garbage input

## Verification

1. `go build -o approver .` — compiles
2. `go test ./...` — all tests pass
3. Manual: run `approver` with a GHE repo in `repo_sources` with `host:` set — should fetch PRs/issues
4. Manual: run `approver` with a GHE repo without `host:` — should auto-detect from remote
5. Manual: existing github.com repos continue to work unchanged
6. Config backward-compatible: existing configs without `host:` work identically

## Files modified

| File | Changes |
|------|---------|
| `internal/config/repos.go` | Add `Host` to `RepoSource` and `RepoEntry`; propagate in `ResolveRepoDirs` |
| `internal/github/prs.go` | Add `DefaultHost`, `ParseHostFromRemote`, `CheckGHInstalled`, `CheckGHHosts`; remove `CheckGH` |
| `internal/app/app.go` | Replace `CheckGH()` with `CheckGHInstalled()`; add host-detection loop; add per-host auth check |
| `internal/github/prs_test.go` | Add `TestParseHostFromRemote` |
| `agent-os/specs/2026-02-18-ghe-support/` | Spec documentation (plan, shape, references) |

## What does NOT change

- All `gh` commands (fetch PRs/issues, approve, request changes) — `cmd.Dir` already handles host resolution
- All `git` commands — host-agnostic
- UI code — no display changes
- Worktree management — unaffected
