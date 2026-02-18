# GHE Support — Shaping Notes

## Problem

Approver fails silently when configured with GitHub Enterprise repos. The startup auth check (`gh auth token`) only validates the default host (github.com), so GHE repos pass the check but then fail when fetching PRs.

## Appetite

Small batch — the fix is surgical. All `gh` commands already work with GHE via `cmd.Dir`; only the auth validation is broken.

## Solution

1. **Per-repo host tracking**: Add `Host` field to config types so users can explicitly set the GHE hostname, or leave it empty for auto-detection.
2. **Auto-detection**: Parse the git remote URL to extract the hostname when not explicitly configured.
3. **Per-host auth check**: Replace the single `gh auth token` call with per-host checks using `--hostname`.

## Rabbit holes

- Don't try to make `gh` commands host-aware — they already resolve via `cmd.Dir`.
- Don't add host to PR/issue types — the repo dir already carries that context.
- Don't support multiple remotes per repo — origin is sufficient.

## No-gos

- No UI changes for host display.
- No host configuration wizard.
- No support for non-git-based host detection.
