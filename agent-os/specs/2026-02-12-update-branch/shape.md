# Update Branch to Trunk - Shape

## Problem
No way to merge base branch into worktree without leaving TUI.

## Solution
- `u` key requires worktree to exist
- Shows confirm: "Merge origin/{base} into worktree for PR #{N}? (y/n)"
- confirmUpdateBranch + confirmPushBranch actions
- updateBranchCmd: `git -C {wtPath} fetch origin {base} && git -C {wtPath} merge origin/{base}`
- On success: "Branch updated" then confirm "Push to origin? (y/n)"
- pushBranchCmd: `git -C {wtPath} push origin HEAD`

## Files Changed
- `internal/app/app.go` — u key handler, confirm actions, update/push cmds
- `internal/app/messages.go` — branchUpdatedMsg, branchUpdateErrorMsg, branchPushedMsg, branchPushErrorMsg
- `internal/ui/menu.go` — u: update branch hint
