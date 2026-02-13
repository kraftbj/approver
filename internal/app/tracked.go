package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/kraft/approver/internal/github"
)

// TrackedPR is a minimal record for a manually-tracked PR.
type TrackedPR struct {
	Number int    `json:"number"`
	Repo   string `json:"repo,omitempty"`
}

// trackedFilePath returns the path to the tracked PRs file.
func trackedFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "approver", "tracked.json")
}

// loadTrackedPRs reads the tracked PR list from disk.
func loadTrackedPRs() ([]TrackedPR, error) {
	path := trackedFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tracked []TrackedPR
	if err := json.Unmarshal(data, &tracked); err != nil {
		return nil, err
	}
	return tracked, nil
}

// saveTrackedPRs writes the tracked PR list to disk.
func saveTrackedPRs(tracked []TrackedPR) error {
	path := trackedFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tracked, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// addTrackedPR adds a PR number to the tracked list.
func addTrackedPR(prNumber int) error {
	tracked, err := loadTrackedPRs()
	if err != nil {
		return err
	}

	// Dedup
	for _, t := range tracked {
		if t.Number == prNumber {
			return nil
		}
	}

	tracked = append(tracked, TrackedPR{Number: prNumber})
	return saveTrackedPRs(tracked)
}

// removeTrackedPR removes a PR number from the tracked list.
func removeTrackedPR(prNumber int) error {
	tracked, err := loadTrackedPRs()
	if err != nil {
		return err
	}

	for i, t := range tracked {
		if t.Number == prNumber {
			tracked = append(tracked[:i], tracked[i+1:]...)
			return saveTrackedPRs(tracked)
		}
	}
	return nil
}

// fetchTrackedPRsCmd fetches all tracked PRs and merges them with the main list.
func fetchTrackedPRsCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		tracked, err := loadTrackedPRs()
		if err != nil {
			return prsErrorMsg{err: fmt.Errorf("loading tracked PRs: %w", err)}
		}

		var trackedPRs []gh.PR
		for _, t := range tracked {
			pr, err := gh.FetchPR(repoDir, fmt.Sprintf("%d", t.Number))
			if err != nil {
				// Skip PRs that can't be fetched (deleted, etc.)
				continue
			}
			pr.Source = "manual"
			trackedPRs = append(trackedPRs, *pr)
		}

		return trackedPRsLoadedMsg{prs: trackedPRs}
	}
}

// trackedPRsLoadedMsg carries the fetched tracked PRs.
type trackedPRsLoadedMsg struct {
	prs []gh.PR
}
