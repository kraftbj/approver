package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// trackedIssue is a minimal record for a manually-tracked issue.
type trackedIssue struct {
	Number int    `json:"number"`
	Repo   string `json:"repo,omitempty"`
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
