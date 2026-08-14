package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	gh "github.com/kraft/approver/internal/github"
)

// PRList is the left panel showing the list of PRs.
type PRList struct {
	PRs          []gh.PR
	Selected     int
	Width        int
	Height       int
	Offset       int    // scroll offset (in PR indices)
	displayOrder []int  // maps display position → PR index (for grouped view navigation)
	LoadingMsg   string // when non-empty, shown at bottom of list panel
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
	l.RebuildDisplayOrder()
	l.clampScroll()
}

// RebuildDisplayOrder rebuilds the display-order mapping.
// In multi-repo mode, PRs are grouped by repo (alphabetical), so the display
// order differs from the insertion order in l.PRs.
func (l *PRList) RebuildDisplayOrder() {
	if !l.multiRepo() {
		l.displayOrder = nil
		return
	}
	repos := l.repoGroups()
	l.displayOrder = l.displayOrder[:0:0]
	for _, repo := range repos {
		for i, pr := range l.PRs {
			if pr.Repo == repo {
				l.displayOrder = append(l.displayOrder, i)
			}
		}
	}
}

// MoveUp moves selection up one item in display order.
func (l *PRList) MoveUp() {
	if len(l.displayOrder) > 0 {
		pos := l.displayPos()
		if pos > 0 {
			l.Selected = l.displayOrder[pos-1]
		}
	} else if l.Selected > 0 {
		l.Selected--
	}
	l.clampScroll()
}

// MoveDown moves selection down one item in display order.
func (l *PRList) MoveDown() {
	if len(l.displayOrder) > 0 {
		pos := l.displayPos()
		if pos < len(l.displayOrder)-1 {
			l.Selected = l.displayOrder[pos+1]
		}
	} else if l.Selected < len(l.PRs)-1 {
		l.Selected++
	}
	l.clampScroll()
}

// displayPos returns the current position of Selected within displayOrder.
func (l *PRList) displayPos() int {
	for i, idx := range l.displayOrder {
		if idx == l.Selected {
			return i
		}
	}
	return 0
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

// multiRepo returns true if the list contains PRs from more than one repo.
func (l *PRList) multiRepo() bool {
	if len(l.PRs) <= 1 {
		return false
	}
	first := l.PRs[0].Repo
	for _, pr := range l.PRs[1:] {
		if pr.Repo != first {
			return true
		}
	}
	return false
}

// repoGroups returns PRs grouped by repo, sorted alphabetically by repo name.
func (l *PRList) repoGroups() []string {
	seen := map[string]bool{}
	var repos []string
	for _, pr := range l.PRs {
		if !seen[pr.Repo] {
			seen[pr.Repo] = true
			repos = append(repos, pr.Repo)
		}
	}
	sort.Strings(repos)
	return repos
}

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

	multi := l.multiRepo()

	if !multi {
		return l.viewFlat(contentWidth)
	}
	return l.viewGrouped(contentWidth)
}

// viewFlat renders the list without repo headers (single-repo mode).
func (l *PRList) viewFlat(contentWidth int) string {
	availHeight := l.Height
	if l.LoadingMsg != "" {
		availHeight--
	}
	visibleItems := availHeight / linesPerItem
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

	if l.LoadingMsg != "" {
		lines = append(lines, DimStyle.Render("  "+l.LoadingMsg))
	}

	content := strings.Join(lines, "\n")
	return ListPanelStyle.Width(l.Width).Height(l.Height).Render(content)
}

// viewGrouped renders the list with repo headers (multi-repo mode).
func (l *PRList) viewGrouped(contentWidth int) string {
	repos := l.repoGroups()

	// Build ordered list of (repoName, prIndex) pairs
	type displayRow struct {
		isHeader bool
		repo     string
		prIdx    int
	}
	var rows []displayRow
	for _, repo := range repos {
		rows = append(rows, displayRow{isHeader: true, repo: repo})
		for i, pr := range l.PRs {
			if pr.Repo == repo {
				rows = append(rows, displayRow{prIdx: i})
			}
		}
	}

	// Find the row index of the selected PR
	selectedRow := 0
	for ri, row := range rows {
		if !row.isHeader && row.prIdx == l.Selected {
			selectedRow = ri
			break
		}
	}

	// Compute visible window in display lines
	availLines := l.Height
	if l.LoadingMsg != "" {
		availLines--
	}
	var lines []string

	// Scroll: start from a row such that the selected row is visible.
	// Walk backward from selectedRow to find a good start position.
	startRow := 0
	if len(rows) > 0 {
		// Calculate total lines from startRow to selectedRow
		startRow = selectedRow
		usedLines := linesPerItem // the selected item itself
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
	}

	usedLines := 0
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
			pr := l.PRs[row.prIdx]
			isSelected := row.prIdx == l.Selected
			line1, line2 := l.renderPRItem(pr, isSelected, contentWidth)
			lines = append(lines, line1, line2)
			usedLines += linesPerItem
		}
	}

	if l.LoadingMsg != "" {
		lines = append(lines, DimStyle.Render("  "+l.LoadingMsg))
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
	if pr.HasNotification {
		prefix += CIStyle("pending").Render("!")
	}
	if pr.Source == "manual" {
		prefix += ManualBadgeStyle.Render("+")
	}
	if pr.HasReview {
		prefix += CIStyle("pass").Render("R")
	} else if pr.IsReviewing {
		prefix += CIStyle("pending").Render("~")
	}
	if pr.HasTmux {
		prefix += ManualBadgeStyle.Render("T")
	}
	if pr.WorktreeSource == "conductor" {
		prefix += ManualBadgeStyle.Render("C")
	}
	if pr.IsCreatingWorktree {
		prefix += CIStyle("pending").Render("W")
	}
	if pr.State == "MERGED" {
		prefix += MergedBadgeStyle.Render("M")
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
