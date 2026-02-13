# Comment Viewer - Shape

## Problem
No way to see PR comments without leaving the TUI.

## Solution
- Tab cycles 3 modes: info -> review -> comments
- Comment struct: Author, Body, CreatedAt, URL
- FetchComments via `gh pr view <N> --json comments`
- Comments cached per PR number, fetched on demand when entering comments mode
- ViewComments renders author + relative time + body per comment

## Files Changed
- `internal/github/types.go` — Comment struct
- `internal/github/prs.go` — FetchComments function
- `internal/ui/prdetail.go` — ViewComments renderer
- `internal/app/app.go` — Tab cycling (3 modes), comments map, fetch cmd
- `internal/app/messages.go` — commentsLoadedMsg, commentsErrorMsg

## Test
- TestFetchComments in prs_test.go with JSON fixture
