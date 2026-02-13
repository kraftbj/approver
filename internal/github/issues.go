package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// ghIssueFields is the set of fields we request from gh issue list/view.
const ghIssueFields = "number,title,author,labels,assignees,state,body,url,updatedAt"

// FetchIssues fetches all issues assigned to the current user.
func FetchIssues(repoDir string) ([]Issue, error) {
	cmd := exec.Command("gh", "issue", "list",
		"--assignee", "@me",
		"--json", ghIssueFields,
		"--limit", "50",
	)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh issue list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh issue list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	for i := range issues {
		issues[i].Source = "assigned"
	}

	return issues, nil
}

// FetchIssue fetches a single issue by number or URL.
func FetchIssue(repoDir string, numberOrURL string) (*Issue, error) {
	ref := numberOrURL
	if _, err := strconv.Atoi(numberOrURL); err != nil {
		ref = numberOrURL
	}

	cmd := exec.Command("gh", "issue", "view", ref,
		"--json", ghIssueFields,
	)
	if repoDir != "" {
		cmd.Dir = repoDir
	}

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh issue view failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh issue view failed: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	issue.Source = "manual"
	return &issue, nil
}
