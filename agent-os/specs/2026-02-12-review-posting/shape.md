# Review Posting (Request Changes) - Shape

## Problem
Can only approve; no way to request changes from the TUI.

## Solution
- `inputAction` type distinguishes input contexts: inputAddPR, inputRequestChanges
- `X` key sets stateInput + inputAction = inputRequestChanges + prompt "Reason:"
- handleInputKey dispatches based on inputAction on Enter
- `gh pr review <N> --request-changes --body "..."`
- On success: show message, refresh PR

## Files Changed
- `internal/app/app.go` — inputAction type, X key handler, handleInputKey dispatch
- `internal/app/messages.go` — prChangesRequestedMsg, prChangesRequestErrorMsg
- `internal/ui/menu.go` — add X: request changes hint
