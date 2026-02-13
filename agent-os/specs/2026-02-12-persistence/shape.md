# Persistence (Review Results) - Shape

## Problem
Review results lost on restart.

## Solution
- Follow tracked.go pattern
- Storage: ~/.config/approver/reviews/review-{prNumber}.json
- saveReviewResult, loadAllReviews, deleteReviewResult functions
- On claudeReviewDoneMsg: save to disk after storing in memory
- On startup Init(): loadReviewsCmd in batch
- reviewsLoadedMsg populates h.reviews + reconcile

## Files Changed
- new `internal/app/reviews.go` — persistence functions
- `internal/app/app.go` — Init batch, reviewsLoadedMsg handler, save on review done

## Test
- TestSaveAndLoadReview in reviews_test.go
