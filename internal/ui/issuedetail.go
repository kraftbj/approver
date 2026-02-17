package ui

import (
	"fmt"
	"strings"

	gh "github.com/kraft/approver/internal/github"
)

// IssueDetail is the right panel showing details of the selected issue.
type IssueDetail struct {
	Width  int
	Height int
}

func NewIssueDetail() IssueDetail {
	return IssueDetail{}
}

func (d *IssueDetail) SetSize(w, h int) {
	d.Width = w
	d.Height = h
}

// View renders the issue detail panel (info mode).
func (d *IssueDetail) View(issue *gh.Issue) string {
	if issue == nil {
		content := DimStyle.Render("No issue selected")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	var sections []string

	sections = append(sections, TitleStyle.Render(fmt.Sprintf("#%d %s", issue.Number, issue.Title)))
	sections = append(sections, "")

	// Metadata
	sections = append(sections, fmt.Sprintf("  Author:   @%s", issue.Author.Login))
	sections = append(sections, fmt.Sprintf("  State:    %s", issue.State))
	sections = append(sections, fmt.Sprintf("  URL:      %s", issue.URL))
	sections = append(sections, fmt.Sprintf("  Updated:  %s", issue.RelativeTime()))
	sections = append(sections, "")

	// Assignees
	if len(issue.Assignees) > 0 {
		logins := issue.AssigneeLogins()
		sections = append(sections, fmt.Sprintf("  Assigned: %s", strings.Join(logins, ", ")))
		sections = append(sections, "")
	}

	// Labels
	if len(issue.Labels) > 0 {
		names := issue.LabelNames()
		sections = append(sections, fmt.Sprintf("  Labels:   %s", strings.Join(names, ", ")))
		sections = append(sections, "")
	}

	// Source indicator
	if issue.Source == "manual" {
		sections = append(sections, ManualBadgeStyle.Render("  [added]"))
		sections = append(sections, "")
	}

	// Linked PRs
	if len(issue.LinkedPRs) > 0 {
		sections = append(sections, SectionHeaderStyle.Render("  Linked PRs"))
		for _, lpr := range issue.LinkedPRs {
			stateStyled := CIStyle("pass").Render(lpr.State)
			if lpr.State == "CLOSED" {
				stateStyled = CIStyle("fail").Render(lpr.State)
			} else if lpr.State == "OPEN" {
				stateStyled = CIStyle("pending").Render(lpr.State)
			}
			sections = append(sections, fmt.Sprintf("  #%d %s  %s", lpr.Number, lpr.Title, stateStyled))
			if lpr.HeadRefName != "" {
				sections = append(sections, DimStyle.Render(fmt.Sprintf("    branch: %s", lpr.HeadRefName)))
			}
		}
		sections = append(sections, DimStyle.Render("  Press p to open linked PR in browser"))
		sections = append(sections, "")
	}

	// Worktree status
	if issue.HasWorktree {
		sections = append(sections, CIStyle("pass").Render("  Worktree: active"))
	} else if issue.IsCreatingWorktree {
		sections = append(sections, CIStyle("pending").Render("  Worktree: creating..."))
	} else {
		sections = append(sections, DimStyle.Render("  Worktree: none"))
	}
	sections = append(sections, "")

	// Body
	if issue.Body != "" {
		sections = append(sections, SectionHeaderStyle.Render("  Description"))
		sections = append(sections, "")
		maxWidth := d.Width - 4
		wrapped := wrapText(issue.Body, maxWidth)
		for _, line := range strings.Split(wrapped, "\n") {
			sections = append(sections, "  "+line)
		}
	}

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}

// ViewWorking renders the "AI session in progress" view.
func (d *IssueDetail) ViewWorking(issue *gh.Issue, spinnerView, stepName string) string {
	if issue == nil {
		content := DimStyle.Render("No issue selected")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	var sections []string

	sections = append(sections, TitleStyle.Render(fmt.Sprintf("#%d AI Session", issue.Number)))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  %s %s", spinnerView, stepName))
	sections = append(sections, "")
	sections = append(sections, DimStyle.Render("  Press C to cancel."))

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}
