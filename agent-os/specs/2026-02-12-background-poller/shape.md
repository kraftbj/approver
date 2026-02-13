# Background Poller - Shape

## Problem
PRs go stale; must manually press R to see updates.

## Solution
- PollInterval config field (default: 300 seconds, 0 = disabled)
- pollTickMsg scheduled after initial prsLoadedMsg
- On tick: re-fetch silently (no loading state)
- pollPRsLoadedMsg merges results without spinner
- Schedule next tick after each poll
- Don't poll while in loading state

## Files Changed
- `internal/config/config.go` — PollInterval field + default
- `internal/app/app.go` — poll scheduling, pollPRsCmd, merge logic
- `internal/app/messages.go` — pollTickMsg, pollPRsLoadedMsg

## Test
- TestDefaultPollInterval in config_test.go
