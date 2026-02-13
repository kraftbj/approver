# Plan: Optimistic Approval State

## Changes

### `internal/app/app.go` — `prApprovedMsg` handler

Add a loop after the toast to find the PR by number and set `ReviewDecision = "APPROVED"`:

```go
for i, pr := range h.pr.prList.PRs {
    if pr.Number == msg.prNumber {
        h.pr.prList.PRs[i].ReviewDecision = "APPROVED"
        break
    }
}
```

The `fetchSinglePRCmd` call remains unchanged — it confirms the state from GitHub.

### `agent-os/product/roadmap.md`

Mark the "Optimistic Approval State" bullet as done.

## Verification

1. `go build ./...` compiles
2. `go test ./...` passes
3. `go vet ./...` clean
4. Manual: run approver, press `A`, confirm — badge flips to green APPROVED instantly
