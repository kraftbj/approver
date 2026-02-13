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
	sections = append(sections, fmt.Sprintf("  URL:      %s", pr.URL))
	sections = append(sections, fmt.Sprintf("  Updated:  %s", pr.RelativeTime()))
	sections = append(sections, fmt.Sprintf("  Size:     %s", pr.SizeString()))
	sections = append(sections, "")

	// Review decision
	reviewLabel := pr.ReviewIcon()
	reviewStyled := ReviewStyle(reviewLabel).Render(reviewLabel)
	sections = append(sections, fmt.Sprintf("  Review:   %s", reviewStyled))
	sections = append(sections, "")

	// CI status
	ciSummary := pr.CIStatus()
	ciStyled := CIStyle(ciSummary).Render(fmt.Sprintf("CI: %s", ciSummary))
	sections = append(sections, fmt.Sprintf("  %s", ciStyled))

	// Individual checks
	if len(pr.StatusChecks) > 0 {
		for _, check := range pr.StatusChecks {
			checkStatus := check.Conclusion
			if checkStatus == "" {
				checkStatus = check.Status
			}
			icon := checkIcon(checkStatus)
			sections = append(sections, fmt.Sprintf("    %s %s", icon, check.Name))
		}
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
