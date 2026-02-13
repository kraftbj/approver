# Notifications - Shape

## Problem
No visual indicator when PR state changes (new comments, CI, reviews).

## Solution
- HasNotification + CommentCount runtime fields on PR struct
- prSnapshot struct: commentCount, ciStatus, reviewDecision
- prSnapshots map on home struct
- After initial load: take snapshots (no notification on first load)
- On poll: compare new state to snapshot, set HasNotification if changed
- Yellow `!` badge in PR list prefix
- Clear HasNotification when user navigates to that PR

## Files Changed
- `internal/github/types.go` — HasNotification, CommentCount fields
- `internal/ui/prlist.go` — ! badge rendering
- `internal/app/app.go` — prSnapshot, takeSnapshots, reconcileNotifications, clearNotification
