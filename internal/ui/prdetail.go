package ui

import (
	"fmt"
	"strings"

	gh "github.com/kraft/approver/internal/github"
)

// PRDetail is the right panel showing details of the selected PR.
type PRDetail struct {
	Width  int
	Height int
}

func NewPRDetail() PRDetail {
	return PRDetail{}
}

// SetSize updates the detail panel dimensions.
func (d *PRDetail) SetSize(w, h int) {
	d.Width = w
	d.Height = h
}

// View renders the detail panel for the given PR. Pass nil for empty state.
func (d *PRDetail) View(pr *gh.PR) string {
	if pr == nil {
		content := DimStyle.Render("No PR selected")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	var sections []string

	// Title
	sections = append(sections, TitleStyle.Render(fmt.Sprintf("#%d %s", pr.Number, pr.Title)))
	sections = append(sections, "")

	// Metadata
	sections = append(sections, fmt.Sprintf("  Author:   @%s", pr.Author.Login))
	sections = append(sections, fmt.Sprintf("  Branch:   %s", pr.HeadRefName))
	if pr.BaseRefName != "" {
		sections = append(sections, fmt.Sprintf("  Base:     %s", pr.BaseRefName))
	}
	sections = append(sections, fmt.Sprintf("  URL:      %s", pr.URL))
	sections = append(sections, fmt.Sprintf("  Updated:  %s", pr.RelativeTime()))
	sections = append(sections, fmt.Sprintf("  Size:     %s", pr.SizeString()))
	sections = append(sections, "")

	// Review decision
	reviewLabel := pr.ReviewIcon()
	reviewStyled := ReviewStyle(reviewLabel).Render(reviewLabel)
	sections = append(sections, fmt.Sprintf("  Review:   %s", reviewStyled))
	sections = append(sections, "")

	// Reviewers
	reviewers := pr.ReviewerSummary()
	if len(reviewers) > 0 {
		for _, r := range reviewers {
			name := r.Name
			if r.Team {
				name = name + " (team)"
			}
			var styled string
			switch r.State {
			case "approved":
				styled = CIStyle("pass").Render(fmt.Sprintf("  %s: approved", name))
			case "changes":
				styled = CIStyle("fail").Render(fmt.Sprintf("  %s: changes requested", name))
			case "commented":
				styled = CIStyle("pending").Render(fmt.Sprintf("  %s: commented", name))
			default:
				styled = DimStyle.Render(fmt.Sprintf("  %s: pending", name))
			}
			sections = append(sections, styled)
		}
		sections = append(sections, "")
	}

	// CI status - summary counts instead of listing every check
	ciSummary := pr.CIStatus()
	if len(pr.StatusChecks) > 0 {
		passed, failed, pending, skipped := 0, 0, 0, 0
		var failedNames []string
		for _, check := range pr.StatusChecks {
			switch check.EffectiveState() {
			case "pass":
				passed++
			case "fail":
				failed++
				failedNames = append(failedNames, check.DisplayName())
			case "skipped":
				skipped++
			default:
				pending++
			}
		}

		parts := []string{}
		if passed > 0 {
			parts = append(parts, CIStyle("pass").Render(fmt.Sprintf("%d pass", passed)))
		}
		if failed > 0 {
			parts = append(parts, CIStyle("fail").Render(fmt.Sprintf("%d fail", failed)))
		}
		if pending > 0 {
			parts = append(parts, CIStyle("pending").Render(fmt.Sprintf("%d pending", pending)))
		}
		if skipped > 0 {
			parts = append(parts, DimStyle.Render(fmt.Sprintf("%d skipped", skipped)))
		}

		ciLine := fmt.Sprintf("  CI:       %s", strings.Join(parts, ", "))
		sections = append(sections, ciLine)

		// Only show names of failed checks (the ones you actually care about)
		for _, name := range failedNames {
			sections = append(sections, CIStyle("fail").Render(fmt.Sprintf("            x %s", name)))
		}
	} else {
		sections = append(sections, fmt.Sprintf("  CI:       %s", CIStyle(ciSummary).Render("none")))
	}
	sections = append(sections, "")

	// Labels
	if len(pr.Labels) > 0 {
		var labelNames []string
		for _, l := range pr.Labels {
			labelNames = append(labelNames, l.Name)
		}
		sections = append(sections, fmt.Sprintf("  Labels:   %s", strings.Join(labelNames, ", ")))
		sections = append(sections, "")
	}

	// Source indicator
	if pr.Source == "manual" {
		sections = append(sections, ManualBadgeStyle.Render("  [manually tracked]"))
		sections = append(sections, "")
	}

	// Worktree status
	if pr.HasWorktree {
		sections = append(sections, CIStyle("pass").Render("  Worktree: active"))
	} else if pr.IsCreatingWorktree {
		sections = append(sections, CIStyle("pending").Render("  Worktree: creating..."))
	} else {
		sections = append(sections, DimStyle.Render("  Worktree: none"))
	}

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}

// ReviewDisplayData holds formatted review data for display.
type ReviewDisplayData struct {
	Checklist string // Agent 3's full numbered checklist (primary display)
	RawOutput string // Full combined output
	IssueCount int
	HighCount  int
}

// ViewReview renders the review results panel.
func (d *PRDetail) ViewReview(pr *gh.PR, review *ReviewDisplayData, hasWorktree bool) string {
	if pr == nil {
		content := DimStyle.Render("No PR selected")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	var sections []string

	sections = append(sections, TitleStyle.Render(fmt.Sprintf("#%d Review Results", pr.Number)))
	sections = append(sections, "")

	if review == nil {
		if hasWorktree {
			sections = append(sections, DimStyle.Render("  No review available. Press c to start a review."))
		} else {
			sections = append(sections, DimStyle.Render("  No review available. Press w to create a worktree first."))
		}
		content := strings.Join(sections, "\n")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	// Summary
	if review.IssueCount > 0 {
		summary := fmt.Sprintf("  %d issues found", review.IssueCount)
		if review.HighCount > 0 {
			summary += fmt.Sprintf(" (%d high severity)", review.HighCount)
		}
		sections = append(sections, IssueSeverityStyle("high").Render(summary))
	} else {
		sections = append(sections, CIStyle("pass").Render("  Review complete (see output below)"))
	}
	sections = append(sections, "")

	// Checklist (primary content)
	if review.Checklist != "" {
		sections = append(sections, SectionHeaderStyle.Render("  Confirmed Issues"))
		sections = append(sections, "")

		// Display checklist lines, wrapping each to panel width
		maxWidth := d.Width - 4
		for _, line := range strings.Split(review.Checklist, "\n") {
			if len(line) > maxWidth && maxWidth > 0 {
				line = line[:maxWidth]
			}
			sections = append(sections, "  "+line)
		}
	} else if review.RawOutput != "" {
		sections = append(sections, SectionHeaderStyle.Render("  Review Output"))
		sections = append(sections, "")
		maxWidth := d.Width - 4
		for _, line := range strings.Split(review.RawOutput, "\n") {
			if len(line) > maxWidth && maxWidth > 0 {
				line = line[:maxWidth]
			}
			sections = append(sections, "  "+line)
		}
	}

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}

// ViewReviewing renders the "review in progress" view.
func (d *PRDetail) ViewReviewing(pr *gh.PR, spinnerView, stepName string) string {
	if pr == nil {
		content := DimStyle.Render("No PR selected")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	var sections []string

	sections = append(sections, TitleStyle.Render(fmt.Sprintf("#%d Review in Progress", pr.Number)))
	sections = append(sections, "")
	sections = append(sections, fmt.Sprintf("  %s %s", spinnerView, stepName))
	sections = append(sections, "")
	sections = append(sections, DimStyle.Render("  Press c to cancel."))

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}

// ViewComments renders the PR comments panel.
func (d *PRDetail) ViewComments(pr *gh.PR, comments []gh.Comment) string {
	if pr == nil {
		content := DimStyle.Render("No PR selected")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	var sections []string

	sections = append(sections, TitleStyle.Render(fmt.Sprintf("#%d Comments", pr.Number)))
	sections = append(sections, "")

	if len(comments) == 0 {
		sections = append(sections, DimStyle.Render("  No comments on this PR."))
		content := strings.Join(sections, "\n")
		return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
	}

	sections = append(sections, DimStyle.Render(fmt.Sprintf("  %d comments", len(comments))))
	sections = append(sections, "")

	maxWidth := d.Width - 4
	for _, c := range comments {
		// Author + relative time
		header := fmt.Sprintf("  @%s  %s", c.Author.Login, gh.FormatRelativeTime(c.CreatedAt))
		sections = append(sections, SectionHeaderStyle.Render(header))

		// Body (truncate long lines)
		for _, line := range strings.Split(c.Body, "\n") {
			if len(line) > maxWidth && maxWidth > 0 {
				line = line[:maxWidth]
			}
			sections = append(sections, "  "+line)
		}
		sections = append(sections, "")
	}

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}
