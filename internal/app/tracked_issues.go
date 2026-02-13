package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/kraft/approver/internal/github"
)

// trackedIssue is a minimal record for a manually-tracked issue.
type trackedIssue struct {
	Number int `json:"number"`
}

func trackedIssuesFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "approver", "tracked-issues.json"), nil
}

func loadTrackedIssues() ([]trackedIssue, error) {
	path, err := trackedIssuesFilePath()
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

	var tracked []trackedIssue
	if err := json.Unmarshal(data, &tracked); err != nil {
		return nil, err
	}
	return tracked, nil
}

func saveTrackedIssues(tracked []trackedIssue) error {
	path, err := trackedIssuesFilePath()
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

func addTrackedIssue(issueNumber int) error {
	tracked, err := loadTrackedIssues()
	if err != nil {
		return err
	}

	for _, t := range tracked {
		if t.Number == issueNumber {
			return nil
		}
	}

	tracked = append(tracked, trackedIssue{Number: issueNumber})
	return saveTrackedIssues(tracked)
}

func removeTrackedIssue(issueNumber int) error {
	tracked, err := loadTrackedIssues()
	if err != nil {
		return err
	}

	for i, t := range tracked {
		if t.Number == issueNumber {
			tracked = append(tracked[:i], tracked[i+1:]...)
			return saveTrackedIssues(tracked)
		}
	}
	return nil
}

func fetchTrackedIssuesCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		tracked, err := loadTrackedIssues()
		if err != nil {
			return issuesErrorMsg{err: fmt.Errorf("loading tracked issues: %w", err)}
		}

		var issues []gh.Issue
		for _, t := range tracked {
			issue, err := gh.FetchIssue(repoDir, fmt.Sprintf("%d", t.Number))
			if err != nil {
				continue
			}
			issue.Source = "manual"
			issues = append(issues, *issue)
		}

		return trackedIssuesLoadedMsg{issues: issues}
	}
}

