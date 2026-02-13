package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kraft/approver/internal/claude"
)

// reviewsDir returns the directory for persisted review results.
func reviewsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "approver", "reviews")
}

// reviewFilePath returns the path for a specific PR's review result.
func reviewFilePath(prNumber int) string {
	return filepath.Join(reviewsDir(), "review-"+strconv.Itoa(prNumber)+".json")
}

// saveReviewResult writes a review result to disk.
func saveReviewResult(prNumber int, result claude.ReviewResult) error {
	dir := reviewsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reviewFilePath(prNumber), data, 0o644)
}

// loadAllReviews reads all persisted review results from disk.
func loadAllReviews() map[int]claude.ReviewResult {
	reviews := make(map[int]claude.ReviewResult)

	dir := reviewsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return reviews
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "review-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		numStr := strings.TrimPrefix(name, "review-")
		numStr = strings.TrimSuffix(numStr, ".json")
		prNumber, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}

		var result claude.ReviewResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		reviews[prNumber] = result
	}

	return reviews
}

// deleteReviewResult removes a persisted review result.
func deleteReviewResult(prNumber int) error {
	path := reviewFilePath(prNumber)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// loadReviewsCmd is a tea.Cmd that loads all persisted reviews.
func loadReviewsCmd() tea.Cmd {
	return func() tea.Msg {
		reviews := loadAllReviews()
		return reviewsLoadedMsg{reviews: reviews}
	}
}

// reviewsLoadedMsg carries persisted review results loaded from disk.
type reviewsLoadedMsg struct {
	reviews map[int]claude.ReviewResult
}
