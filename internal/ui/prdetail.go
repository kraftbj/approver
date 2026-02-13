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
	} else {
		sections = append(sections, DimStyle.Render("  Worktree: none"))
	}

	content := strings.Join(sections, "\n")
	return DetailPanelStyle.Width(d.Width).Height(d.Height).Render(content)
}

func checkIcon(status string) string {
	switch status {
	case "SUCCESS":
		return CIStyle("pass").Render("*")
	case "FAILURE", "ERROR", "TIMED_OUT", "CANCELLED":
		return CIStyle("fail").Render("x")
	case "IN_PROGRESS", "QUEUED", "PENDING":
		return CIStyle("pending").Render("~")
	default:
		return DimStyle.Render("-")
	}
}
