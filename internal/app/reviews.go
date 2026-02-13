package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kraft/approver/internal/claude"
)

// reviewsDir returns the directory for persisted review results.
func reviewsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "approver", "reviews"), nil
}

// reviewFilePath returns the path for a specific PR's review result.
func reviewFilePath(prNumber int) (string, error) {
	dir, err := reviewsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "review-"+strconv.Itoa(prNumber)+".json"), nil
}

// saveReviewResult writes a review result to disk.
func saveReviewResult(prNumber int, result claude.ReviewResult) error {
	dir, err := reviewsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	path, err := reviewFilePath(prNumber)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// loadAllReviews reads all persisted review results from disk.
func loadAllReviews() map[int]claude.ReviewResult {
	reviews := make(map[int]claude.ReviewResult)

	dir, err := reviewsDir()
	if err != nil {
		return reviews
	}
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

// loadReviewsCmd is a tea.Cmd that loads all persisted reviews.
func loadReviewsCmd() tea.Cmd {
	return func() tea.Msg {
		reviews := loadAllReviews()
		return reviewsLoadedMsg{reviews: reviews}
	}
}

