# Optimistic Approval State

## Problem

After approving a PR with `A`, the info toast says "PR #N approved" but the review decision badge stays stale until the background `fetchSinglePRCmd` round-trip completes. This makes the UI feel laggy — the user confirmed the action, the `gh` call succeeded, but the badge doesn't reflect it.

## Solution

Set `ReviewDecision = "APPROVED"` on the local PR object immediately when the `prApprovedMsg` arrives (i.e., the `gh pr review --approve` call succeeded). The existing `fetchSinglePRCmd` stays as a confirmation fetch — it will reconcile any other fields that changed server-side.

## Non-goals

- Removal logic: approved PRs naturally disappear from the `review-requested:@me` query on the next full refresh (`R` or app restart).
- Optimistic updates for other actions (request-changes, comment posting).

## Risks

None meaningful. The `gh` call already succeeded before `prApprovedMsg` fires, so the optimistic state matches reality. If the re-fetch somehow returns a different state, it overwrites the optimistic value — self-correcting.
