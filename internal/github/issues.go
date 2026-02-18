package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ghIssueFields is the set of fields we request from gh issue list/view.
const ghIssueFields = "number,title,author,labels,assignees,state,body,url,updatedAt,closedByPullRequestsReferences"

// issueSearchQueries are run separately and merged because GHE doesn't support OR.
var issueSearchQueries = []string{
	"assignee:@me",
	"author:@me",
}

// FetchIssues fetches all issues assigned to the current user.
func FetchIssues(repoDir string, limit int) ([]Issue, error) {
	if limit <= 0 {
		limit = 50
	}

	seen := make(map[int]bool)
	var issues []Issue

	for _, query := range issueSearchQueries {
		cmd := GHCommand(repoDir, "issue", "list", "--search", query, "--json", ghIssueFields, "--limit", strconv.Itoa(limit))

		out, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("gh issue list failed: %s", string(exitErr.Stderr))
			}
			return nil, fmt.Errorf("gh issue list failed: %w", err)
		}

		var batch []Issue
		if err := json.Unmarshal(out, &batch); err != nil {
			return nil, fmt.Errorf("parsing gh output: %w", err)
		}

		for _, issue := range batch {
			if !seen[issue.Number] {
				seen[issue.Number] = true
				issue.Source = "auto"
				issues = append(issues, issue)
			}
		}
	}

	return issues, nil
}

// FetchIssue fetches a single issue by number or URL.
func FetchIssue(repoDir string, numberOrURL string) (*Issue, error) {
	numberOrURL = strings.TrimSpace(numberOrURL)
	cmd := GHCommand(repoDir, "issue", "view", numberOrURL, "--json", ghIssueFields)

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
