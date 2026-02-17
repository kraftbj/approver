package ui

import (
	"fmt"
	"sort"
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

func (l *IssueList) multiRepo() bool {
	if len(l.Issues) <= 1 {
		return false
	}
	first := l.Issues[0].Repo
	for _, issue := range l.Issues[1:] {
		if issue.Repo != first {
			return true
		}
	}
	return false
}

func (l *IssueList) repoGroups() []string {
	seen := map[string]bool{}
	var repos []string
	for _, issue := range l.Issues {
		if !seen[issue.Repo] {
			seen[issue.Repo] = true
			repos = append(repos, issue.Repo)
		}
	}
	sort.Strings(repos)
	return repos
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

	multi := l.multiRepo()

	if !multi {
		return l.viewFlat(contentWidth)
	}
	return l.viewGrouped(contentWidth)
}

func (l *IssueList) viewFlat(contentWidth int) string {
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

func (l *IssueList) viewGrouped(contentWidth int) string {
	repos := l.repoGroups()

	type displayRow struct {
		isHeader bool
		repo     string
		issueIdx int
	}
	var rows []displayRow
	for _, repo := range repos {
		rows = append(rows, displayRow{isHeader: true, repo: repo})
		for i, issue := range l.Issues {
			if issue.Repo == repo {
				rows = append(rows, displayRow{issueIdx: i})
			}
		}
	}

	selectedRow := 0
	for ri, row := range rows {
		if !row.isHeader && row.issueIdx == l.Selected {
			selectedRow = ri
			break
		}
	}

	availLines := l.Height
	var lines []string

	startRow := selectedRow
	usedLines := linesPerItem
	for startRow > 0 {
		prev := startRow - 1
		prevLines := linesPerItem
		if rows[prev].isHeader {
			prevLines = 1
		}
		if usedLines+prevLines > availLines {
			break
		}
		usedLines += prevLines
		startRow = prev
	}

	usedLines = 0
	for ri := startRow; ri < len(rows) && usedLines < availLines; ri++ {
		row := rows[ri]
		if row.isHeader {
			if usedLines+1 > availLines {
				break
			}
			header := RepoHeaderStyle.Render(fmt.Sprintf("  %s", row.repo))
			headerLine := lipgloss.NewStyle().Width(contentWidth).Render(header)
			lines = append(lines, headerLine)
			usedLines++
		} else {
			if usedLines+linesPerItem > availLines {
				break
			}
			issue := l.Issues[row.issueIdx]
			isSelected := row.issueIdx == l.Selected
			line1, line2 := l.renderIssueItem(issue, isSelected, contentWidth)
			lines = append(lines, line1, line2)
			usedLines += linesPerItem
		}
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
	if issue.IsCreatingWorktree {
		prefix += CIStyle("pending").Render("W")
	}
	if len(issue.LinkedPRs) > 0 {
		prefix += CIStyle("pass").Render("P")
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
