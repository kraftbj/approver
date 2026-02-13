# References

## Code

- `internal/app/app.go` — `prApprovedMsg` handler (line ~334): where the optimistic update is applied
- `internal/app/commands.go` — `fetchSinglePRCmd`: background re-fetch that confirms the state
- `internal/github/types.go` — `PullRequest.ReviewDecision`: the field being updated
- `internal/ui/styles.go` — review decision badge rendering (uses `ReviewDecision` value for color)

## Messages

- `prApprovedMsg` — sent when `gh pr review --approve` succeeds
- `fetchSinglePRMsg` — sent when the confirmation re-fetch completes
