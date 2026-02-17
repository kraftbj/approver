package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/kraft/approver/internal/github"
)

type excludedItem struct {
	Number int    `json:"number"`
	Repo   string `json:"repo,omitempty"`
}

func excludedPRsFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "approver", "excluded-prs.json"), nil
}

func excludedIssuesFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, ".config", "approver", "excluded-issues.json"), nil
}

func loadExcludedItems(pathFunc func() (string, error)) ([]excludedItem, error) {
	path, err := pathFunc()
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
	var items []excludedItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func saveExcludedItems(pathFunc func() (string, error), items []excludedItem) error {
	path, err := pathFunc()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func addExcludedPR(number int, repo string) error {
	items, err := loadExcludedItems(excludedPRsFilePath)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Number == number && item.Repo == repo {
			return nil
		}
	}
	items = append(items, excludedItem{Number: number, Repo: repo})
	return saveExcludedItems(excludedPRsFilePath, items)
}

func addExcludedIssue(number int, repo string) error {
	items, err := loadExcludedItems(excludedIssuesFilePath)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Number == number && item.Repo == repo {
			return nil
		}
	}
	items = append(items, excludedItem{Number: number, Repo: repo})
	return saveExcludedItems(excludedIssuesFilePath, items)
}

func isExcludedPR(number int, repo string) bool {
	items, err := loadExcludedItems(excludedPRsFilePath)
	if err != nil {
		return false
	}
	for _, item := range items {
		if item.Number == number && item.Repo == repo {
			return true
		}
	}
	return false
}

func isExcludedIssue(number int, repo string) bool {
	items, err := loadExcludedItems(excludedIssuesFilePath)
	if err != nil {
		return false
	}
	for _, item := range items {
		if item.Number == number && item.Repo == repo {
			return true
		}
	}
	return false
}

func filterExcludedPRs(prs []gh.PR) []gh.PR {
	items, err := loadExcludedItems(excludedPRsFilePath)
	if err != nil || len(items) == 0 {
		return prs
	}
	excluded := make(map[ItemKey]bool, len(items))
	for _, item := range items {
		excluded[ItemKey{Repo: item.Repo, Number: item.Number}] = true
	}
	var filtered []gh.PR
	for _, pr := range prs {
		if !excluded[ItemKey{Repo: pr.Repo, Number: pr.Number}] {
			filtered = append(filtered, pr)
		}
	}
	return filtered
}

func filterExcludedIssues(issues []gh.Issue) []gh.Issue {
	items, err := loadExcludedItems(excludedIssuesFilePath)
	if err != nil || len(items) == 0 {
		return issues
	}
	excluded := make(map[ItemKey]bool, len(items))
	for _, item := range items {
		excluded[ItemKey{Repo: item.Repo, Number: item.Number}] = true
	}
	var filtered []gh.Issue
	for _, issue := range issues {
		if !excluded[ItemKey{Repo: issue.Repo, Number: issue.Number}] {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// Weekly cleanup types and functions

type cleanupDoneMsg struct{}

func lastCleanupPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "approver", "last-cleanup"), nil
}

func needsCleanup() bool {
	path, err := lastCleanupPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return true // Missing file = needs cleanup
	}
	return time.Since(info.ModTime()) > 7*24*time.Hour
}

func markCleanupDone() {
	path, err := lastCleanupPath()
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0o600)
}

func runExcludeCleanupCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		if !needsCleanup() {
			return cleanupDoneMsg{}
		}

		// Clean excluded PRs
		prItems, _ := loadExcludedItems(excludedPRsFilePath)
		var remainingPRs []excludedItem
		for _, item := range prItems {
			pr, err := gh.FetchPR(repoDir, fmt.Sprintf("%d", item.Number))
			if err != nil {
				remainingPRs = append(remainingPRs, item)
				continue
			}
			if pr.State != "CLOSED" && pr.State != "MERGED" {
				remainingPRs = append(remainingPRs, item)
			}
		}
		if len(prItems) > 0 {
			_ = saveExcludedItems(excludedPRsFilePath, remainingPRs)
		}

		// Clean excluded issues
		issueItems, _ := loadExcludedItems(excludedIssuesFilePath)
		var remainingIssues []excludedItem
		for _, item := range issueItems {
			issue, err := gh.FetchIssue(repoDir, fmt.Sprintf("%d", item.Number))
			if err != nil {
				remainingIssues = append(remainingIssues, item)
				continue
			}
			if issue.State != "CLOSED" {
				remainingIssues = append(remainingIssues, item)
			}
		}
		if len(issueItems) > 0 {
			_ = saveExcludedItems(excludedIssuesFilePath, remainingIssues)
		}

		markCleanupDone()
		return cleanupDoneMsg{}
	}
}
