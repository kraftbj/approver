package claude

import (
	"regexp"
	"strconv"
	"strings"
)

// ReviewResult holds the output of the 3-agent review pipeline.
type ReviewResult struct {
	Summary   string  // Best-effort extracted summary
	Issues    []Issue // Best-effort parsed issues
	RawOutput string  // Full text output (always available)
	Agent1Out string  // Raw Agent 1 output
	Agent2Out string  // Raw Agent 2 output
	Agent3Out string  // Raw Agent 3 (checklist) output
}

// Issue represents a single confirmed issue from the review checklist.
type Issue struct {
	Number   int
	File     string
	Line     int
	Severity string // "high", "medium", "low"
	Problem  string
	Fix      string
}

// issueLineRegex matches numbered checklist items like "1." or "1)" at the start of a line.
var issueLineRegex = regexp.MustCompile(`(?m)^\s*(\d+)[.)]\s+(.*)`)

// severityRegex extracts severity markers like [high], [medium], [low].
var severityRegex = regexp.MustCompile(`(?i)\[(high|medium|low)\]`)

// fileLineRegex extracts file:line references like "path/to/file.go:42".
var fileLineRegex = regexp.MustCompile(`([\w/.\\-]+\.\w+):(\d+)`)

// parseChecklist attempts to extract structured issues from Agent 3 output.
// Falls back gracefully — issues may have missing fields.
func parseChecklist(output string) []Issue {
	matches := issueLineRegex.FindAllStringSubmatchIndex(output, -1)
	if len(matches) == 0 {
		return nil
	}

	var issues []Issue
	for i, match := range matches {
		num, _ := strconv.Atoi(output[match[2]:match[3]])

		// Get the full text for this item (up to next item or end)
		start := match[0]
		end := len(output)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		itemText := output[start:end]

		issue := Issue{
			Number:  num,
			Problem: strings.TrimSpace(output[match[4]:match[5]]),
		}

		// Extract severity
		if sevMatch := severityRegex.FindStringSubmatch(itemText); len(sevMatch) > 1 {
			issue.Severity = strings.ToLower(sevMatch[1])
		}

		// Extract file:line
		if flMatch := fileLineRegex.FindStringSubmatch(itemText); len(flMatch) > 2 {
			issue.File = flMatch[1]
			issue.Line, _ = strconv.Atoi(flMatch[2])
		}

		issues = append(issues, issue)
	}

	return issues
}

// IssueCount returns the number of issues, or 0 if none parsed.
func (r *ReviewResult) IssueCount() int {
	return len(r.Issues)
}

// HighSeverityCount returns the number of high-severity issues.
func (r *ReviewResult) HighSeverityCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == "high" {
			count++
		}
	}
	return count
}
