package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	gh "github.com/kraft/approver/internal/github"
)

// PRList is the left panel showing the list of PRs.
type PRList struct {
	PRs      []gh.PR
	Selected int
	Width    int
	Height   int
	Offset   int // scroll offset
}

func NewPRList() PRList {
	return PRList{}
}

// SetSize updates the list dimensions.
func (l *PRList) SetSize(w, h int) {
	l.Width = w
	l.Height = h
}

// SetPRs replaces the PR list, clamping selection.
func (l *PRList) SetPRs(prs []gh.PR) {
	l.PRs = prs
	if l.Selected >= len(prs) {
		l.Selected = max(0, len(prs)-1)
	}
	l.clampScroll()
}

// MoveUp moves selection up one item.
func (l *PRList) MoveUp() {
	if l.Selected > 0 {
		l.Selected--
		l.clampScroll()
	}
}

// MoveDown moves selection down one item.
func (l *PRList) MoveDown() {
	if l.Selected < len(l.PRs)-1 {
		l.Selected++
		l.clampScroll()
	}
}

// SelectedPR returns the currently selected PR, or nil if empty.
func (l *PRList) SelectedPR() *gh.PR {
	if len(l.PRs) == 0 || l.Selected >= len(l.PRs) {
		return nil
	}
	return &l.PRs[l.Selected]
}

// linesPerItem is the number of display lines per PR entry.
const linesPerItem = 2

// clampScroll ensures the selected item is visible.
func (l *PRList) clampScroll() {
	if l.Height <= 0 {
		return
	}

	visibleItems := l.Height / linesPerItem
	if visibleItems < 1 {
		visibleItems = 1
	}

	if l.Selected < l.Offset {
		l.Offset = l.Selected
	}
	if l.Selected >= l.Offset+visibleItems {
		l.Offset = l.Selected - visibleItems + 1
	}
}

// View renders the PR list.
func (l *PRList) View() string {
	if len(l.PRs) == 0 {
		return ""
	}

	contentWidth := l.Width - 2 // account for border + padding
	if contentWidth < 10 {
		contentWidth = 10
	}

	visibleItems := l.Height / linesPerItem
	if visibleItems < 1 {
		visibleItems = 1
	}

	var lines []string
	end := min(l.Offset+visibleItems, len(l.PRs))
	for i := l.Offset; i < end; i++ {
		pr := l.PRs[i]
		isSelected := i == l.Selected

		line1, line2 := l.renderPRItem(pr, isSelected, contentWidth)
		lines = append(lines, line1, line2)
	}

	content := strings.Join(lines, "\n")
	return ListPanelStyle.Width(l.Width).Height(l.Height).Render(content)
}

// renderPRItem renders a single PR as two lines.
func (l *PRList) renderPRItem(pr gh.PR, selected bool, width int) (string, string) {
	// Line 1: selection indicator + PR number + title
	indicator := "  "
	if selected {
		indicator = "> "
	}

	// Status prefix indicators
	prefix := ""
	if pr.Source == "manual" {
		prefix = ManualBadgeStyle.Render("+")
	}
	if pr.HasReview {
		prefix += CIStyle("pass").Render("R")
	} else if pr.IsReviewing {
		prefix += CIStyle("pending").Render("~")
	}
	if pr.HasTmux {
		prefix += ManualBadgeStyle.Render("T")
	}
	if pr.IsCreatingWorktree {
		prefix += CIStyle("pending").Render("W")
	}

	numberStr := fmt.Sprintf("#%d", pr.Number)
	titleWidth := width - len(indicator) - len(numberStr) - len(prefix) - 2
	title := pr.Title
	if len(title) > titleWidth && titleWidth > 3 {
		title = title[:titleWidth-3] + "..."
	}

	line1 := fmt.Sprintf("%s%s%s %s", indicator, prefix, numberStr, title)

	// Line 2: author + size + CI + review status
	ciLabel := fmt.Sprintf("CI:%s", pr.CIStatus())
	ciStyled := CIStyle(pr.CIStatus()).Render(ciLabel)

	reviewStyled := ReviewStyle(pr.ReviewIcon()).Render(pr.ReviewIcon())

	line2 := fmt.Sprintf("   @%-8s %8s  %s  %s",
		truncate(pr.Author.Login, 8),
		pr.SizeString(),
		ciStyled,
		reviewStyled,
	)

	if selected {
		// Apply selection styling to the raw text parts
		line1Style := SelectedStyle.Width(width)
		line2Style := SelectedStyle.Width(width)
		return line1Style.Render(line1), line2Style.Render(line2)
	}

	dimLine1 := lipgloss.NewStyle().Width(width).Render(line1)
	dimLine2 := lipgloss.NewStyle().Width(width).Render(line2)
	return dimLine1, dimLine2
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
