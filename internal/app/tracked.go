package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kraft/approver/internal/config"
	gh "github.com/kraft/approver/internal/github"
)

// trackedPR is a minimal record for a manually-tracked PR.
type trackedPR struct {
	Number int    `json:"number"`
	Repo   string `json:"repo,omitempty"`
}

// trackedFilePath returns the path to the tracked PRs file.
func trackedFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "approver", "tracked.json"), nil
}

// loadTrackedPRs reads the tracked PR list from disk.
func loadTrackedPRs() ([]trackedPR, error) {
	path, err := trackedFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tracked []trackedPR
	if err := json.Unmarshal(data, &tracked); err != nil {
		return nil, err
	}
	return tracked, nil
}

// saveTrackedPRs writes the tracked PR list to disk.
func saveTrackedPRs(tracked []trackedPR) error {
	path, err := trackedFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tracked, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// addTrackedPR adds a PR number to the tracked list.
func addTrackedPR(prNumber int, repo string) error {
	tracked, err := loadTrackedPRs()
	if err != nil {
		return err
	}

	// Dedup
	for i, t := range tracked {
		if t.Number == prNumber && trackedRepoMatches(t.Repo, repo) {
			if t.Repo == "" && repo != "" {
				tracked[i].Repo = repo
				return saveTrackedPRs(tracked)
			}
			return nil
		}
	}

	tracked = append(tracked, trackedPR{Number: prNumber, Repo: repo})
	return saveTrackedPRs(tracked)
}

// removeTrackedPR removes a PR number from the tracked list.
func removeTrackedPR(prNumber int, repo string) error {
	tracked, err := loadTrackedPRs()
	if err != nil {
		return err
	}

	for i, t := range tracked {
		if t.Number == prNumber && trackedRepoMatches(t.Repo, repo) {
			tracked = append(tracked[:i], tracked[i+1:]...)
			return saveTrackedPRs(tracked)
		}
	}
	return nil
}

func trackedRepoMatches(storedRepo, repo string) bool {
	return storedRepo == repo || storedRepo == "" || repo == ""
}

// trackedPRsLoadedMsg carries the fetched tracked PRs to merge into the main list.
type trackedPRsLoadedMsg struct {
	prs []gh.PR
}

// fetchTrackedPRsCmd fetches all tracked PRs and merges them with the main PR list.
func fetchTrackedPRsCmd(repos []config.RepoEntry) tea.Cmd {
	return func() tea.Msg {
		tracked, err := loadTrackedPRs()
		if err != nil {
			return prsErrorMsg{err: fmt.Errorf("loading tracked PRs: %w", err)}
		}

		repoDir := ""
		repoName := ""
		if len(repos) > 0 {
			repoDir = repos[0].Dir
			repoName = repos[0].Name
		}

		var prs []gh.PR
		for _, t := range tracked {
			fetchDir := repoDir
			fetchName := repoName
			if t.Repo != "" {
				for _, r := range repos {
					if r.Name == t.Repo {
						fetchDir = r.Dir
						fetchName = r.Name
						break
					}
				}
			}
			pr, err := gh.FetchPR(fetchDir, fmt.Sprintf("%d", t.Number))
			if err != nil {
				continue
			}
			// Drop closed/merged tracked PRs
			if pr.State == "CLOSED" || pr.State == "MERGED" {
				_ = removeTrackedPR(t.Number, t.Repo)
				continue
			}
			pr.Source = "manual"
			pr.Repo = fetchName
			pr.RepoDir = fetchDir
			prs = append(prs, *pr)
		}

		return trackedPRsLoadedMsg{prs: prs}
	}
}
