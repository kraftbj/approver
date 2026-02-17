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
func reviewFilePath(key ItemKey) (string, error) {
	dir, err := reviewsDir()
	if err != nil {
		return "", err
	}
	if key.Repo != "" {
		return filepath.Join(dir, key.Repo, "review-"+strconv.Itoa(key.Number)+".json"), nil
	}
	return filepath.Join(dir, "review-"+strconv.Itoa(key.Number)+".json"), nil
}

// saveReviewResult writes a review result to disk.
func saveReviewResult(key ItemKey, result claude.ReviewResult) error {
	path, err := reviewFilePath(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// loadAllReviews reads all persisted review results from disk.
func loadAllReviews() map[ItemKey]claude.ReviewResult {
	reviews := make(map[ItemKey]claude.ReviewResult)

	dir, err := reviewsDir()
	if err != nil {
		return reviews
	}

	// Load flat review files (backward compat — no repo subdir)
	loadReviewsFromDir(dir, "", reviews)

	// Load namespaced review files (reviews/{repoName}/review-N.json)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return reviews
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoName := entry.Name()
		repoDir := filepath.Join(dir, repoName)
		loadReviewsFromDir(repoDir, repoName, reviews)
	}

	return reviews
}

func loadReviewsFromDir(dir, repoName string, reviews map[ItemKey]claude.ReviewResult) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "review-") || !strings.HasSuffix(name, ".json") {
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
		reviews[ItemKey{Repo: repoName, Number: prNumber}] = result
	}
}

// loadReviewsCmd is a tea.Cmd that loads all persisted reviews.
func loadReviewsCmd() tea.Cmd {
	return func() tea.Msg {
		reviews := loadAllReviews()
		return reviewsLoadedMsg{reviews: reviews}
	}
}
