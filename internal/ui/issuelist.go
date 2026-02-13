package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	gh "github.com/kraft/approver/internal/github"
)

// IssueList is the left panel showing the list of issues.
type IssueList struct {
	Issues   []gh.Issue
	Selected int
	Width    int
	Height   int
	Offset   int
}

func NewIssueList() IssueList {
	return IssueList{}
}

func (l *IssueList) SetSize(w, h int) {
	l.Width = w
	l.Height = h
}

func (l *IssueList) SetIssues(issues []gh.Issue) {
	l.Issues = issues
	if l.Selected >= len(issues) {
		l.Selected = max(0, len(issues)-1)
	}
	l.clampScroll()
}

func (l *IssueList) MoveUp() {
	if l.Selected > 0 {
		l.Selected--
		l.clampScroll()
	}
}

func (l *IssueList) MoveDown() {
	if l.Selected < len(l.Issues)-1 {
		l.Selected++
		l.clampScroll()
	}
}

func (l *IssueList) SelectedIssue() *gh.Issue {
	if len(l.Issues) == 0 || l.Selected >= len(l.Issues) {
		return nil
	}
	return &l.Issues[l.Selected]
}

func (l *IssueList) clampScroll() {
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

func (l *IssueList) View() string {
	if len(l.Issues) == 0 {
		return ""
	}

	contentWidth := l.Width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	visibleItems := l.Height / linesPerItem
	if visibleItems < 1 {
		visibleItems = 1
	}

	var lines []string
	end := min(l.Offset+visibleItems, len(l.Issues))
	for i := l.Offset; i < end; i++ {
		issue := l.Issues[i]
		isSelected := i == l.Selected

		line1, line2 := l.renderIssueItem(issue, isSelected, contentWidth)
		lines = append(lines, line1, line2)
	}

	content := strings.Join(lines, "\n")
	return ListPanelStyle.Width(l.Width).Height(l.Height).Render(content)
}

func (l *IssueList) renderIssueItem(issue gh.Issue, selected bool, width int) (string, string) {
	indicator := "  "
	if selected {
		indicator = "> "
	}

	// Status prefix indicators
	prefix := ""
	if issue.Source == "manual" {
		prefix += ManualBadgeStyle.Render("+")
	}
	if issue.HasTmux {
		prefix += ManualBadgeStyle.Render("T")
	}
	if issue.IsWorking {
		prefix += CIStyle("pending").Render("~")
	}
	if issue.IsCreatingWorktree {
		prefix += CIStyle("pending").Render("W")
	}

	numberStr := fmt.Sprintf("#%d", issue.Number)
	titleWidth := width - len(indicator) - len(numberStr) - len(prefix) - 2
	title := issue.Title
	if len(title) > titleWidth && titleWidth > 3 {
		title = title[:titleWidth-3] + "..."
	}

	line1 := fmt.Sprintf("%s%s%s %s", indicator, prefix, numberStr, title)

	// Line 2: author + state + labels
	stateLabel := strings.ToLower(issue.State)
	var stateStyled string
	if stateLabel == "open" {
		stateStyled = CIStyle("pass").Render(stateLabel)
	} else {
		stateStyled = CIStyle("fail").Render(stateLabel)
	}

	labelStr := ""
	labels := issue.LabelNames()
	if len(labels) > 0 {
		if len(labels) > 2 {
			labels = labels[:2]
		}
		labelStr = strings.Join(labels, ",")
		if len(labelStr) > 20 {
			labelStr = labelStr[:17] + "..."
		}
	}

	line2 := fmt.Sprintf("   @%-8s  %s", truncate(issue.Author.Login, 8), stateStyled)
	if labelStr != "" {
		line2 += "  " + DimStyle.Render(labelStr)
	}

	if selected {
		line1Style := SelectedStyle.Width(width)
		line2Style := SelectedStyle.Width(width)
		return line1Style.Render(line1), line2Style.Render(line2)
	}

	dimLine1 := lipgloss.NewStyle().Width(width).Render(line1)
	dimLine2 := lipgloss.NewStyle().Width(width).Render(line2)
	return dimLine1, dimLine2
}
