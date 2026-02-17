package app

import (
	gh "github.com/kraft/approver/internal/github"
)

// ItemKey uniquely identifies a PR or issue across multiple repositories.
type ItemKey struct {
	Repo   string
	Number int
}

// PRKey returns the ItemKey for a PR.
func PRKey(pr *gh.PR) ItemKey {
	return ItemKey{Repo: pr.Repo, Number: pr.Number}
}

// IssueKey returns the ItemKey for an Issue.
func IssueKey(issue *gh.Issue) ItemKey {
	return ItemKey{Repo: issue.Repo, Number: issue.Number}
}
