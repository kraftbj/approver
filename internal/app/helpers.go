package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/claude"
	gh "github.com/kraft/approver/internal/github"
	"github.com/kraft/approver/internal/ui"
)

func (h *home) viewLoading(height int) string {
	content := fmt.Sprintf("\n\n  %s Fetching PRs...", h.spinner.View())
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewEmpty(height int) string {
	content := ui.DimStyle.Render("\n\n  No PRs requesting your review.\n\n  Press R to refresh.")
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewHelp(height int) string {
	helpText := `
  Approver - GitHub AI Workflow Hub

  Screens:
    1              Reviews (PRs requesting your review)
    2              Issues (assigned issues)
    3              Watchlist (tracked PRs)

  Navigation:
    j/k, up/down   Navigate list
    Tab            Cycle detail panel view

  Reviews:
    w              Create worktree for selected PR
    W              Delete worktree (with confirmation)
    c              Start/cancel Claude review (auto-creates worktree)
    t              Open Claude tmux session (auto-creates worktree)
    A              Approve PR (with confirmation)
    X              Request changes (with reason)
    u              Update branch (merge base into worktree)
    F              Fix review findings (select + agent)

  Issues:
    w              Create worktree for selected issue
    W              Delete worktree (with confirmation)
    c              Start Claude tmux session (auto-creates worktree)
    p              Open linked PR in browser
    a              Add issue by number or URL
    d              Remove manually-tracked issue

  Common:
    o              Open in browser
    R              Refresh
    ?              Show this help
    q              Quit
    ctrl+c         Force quit

  Press any key to dismiss.`

	style := lipgloss.NewStyle().
		Width(h.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorCyan).
		Padding(1, 2)

	return style.Render(boxStyle.Render(helpText))
}

func (h *home) viewConfirmOverlay(base string, height int) string {
	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorYellow).
		Padding(1, 2).
		Render(h.confirmMsg)

	return placeOverlay(h.width, height, base, overlay)
}

func (h *home) viewInputOverlay(base string, height int) string {
	content := fmt.Sprintf("%s%s_", h.inputPrompt, h.inputBuffer)
	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorCyan).
		Padding(0, 2).
		Render(content)

	return placeOverlay(h.width, height, base, overlay)
}

func placeOverlay(width, height int, base, overlay string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (h *home) layoutPanels() {
	menuHeight := 2
	errHeight := 0
	if h.errBox.Message != "" {
		errHeight = 1
	}
	panelHeight := h.height - menuHeight - errHeight

	listWidth, detailWidth := screenLayout(h.width)

	h.pr.prList.SetSize(listWidth, panelHeight)
	h.pr.detail.SetSize(detailWidth, panelHeight)

	h.issues.issueList.SetSize(listWidth, panelHeight)
	h.issues.detail.SetSize(detailWidth, panelHeight)

	h.watchlist.prList.SetSize(listWidth, panelHeight)
	h.watchlist.detail.SetSize(detailWidth, panelHeight)

	h.menu.SetWidth(h.width)
	h.errBox.SetWidth(h.width)
}

func screenLayout(totalWidth int) (listWidth, detailWidth int) {
	listWidth = totalWidth * 30 / 100
	detailWidth = totalWidth - listWidth
	return
}

func (h *home) showError(msg string) {
	h.errBox.SetError(msg)
}

func (h *home) showInfo(msg string) {
	h.errBox.SetInfo(msg)
}

func (h *home) addPRToList(pr gh.PR) {
	for i, existing := range h.pr.prList.PRs {
		if existing.Number == pr.Number {
			h.pr.prList.PRs[i] = pr
			return
		}
	}
	h.pr.prList.PRs = append(h.pr.prList.PRs, pr)
}

func (h *home) removePR(number int) {
	for i, pr := range h.pr.prList.PRs {
		if pr.Number == number {
			h.pr.prList.PRs = append(h.pr.prList.PRs[:i], h.pr.prList.PRs[i+1:]...)
			if h.pr.prList.Selected >= len(h.pr.prList.PRs) {
				h.pr.prList.Selected = max(0, len(h.pr.prList.PRs)-1)
			}
			return
		}
	}
}

func (h *home) addIssueToList(issue gh.Issue) {
	for i, existing := range h.issues.issueList.Issues {
		if existing.Number == issue.Number {
			h.issues.issueList.Issues[i] = issue
			return
		}
	}
	h.issues.issueList.Issues = append(h.issues.issueList.Issues, issue)
}

func (h *home) removeIssue(number int) {
	for i, issue := range h.issues.issueList.Issues {
		if issue.Number == number {
			h.issues.issueList.Issues = append(h.issues.issueList.Issues[:i], h.issues.issueList.Issues[i+1:]...)
			if h.issues.issueList.Selected >= len(h.issues.issueList.Issues) {
				h.issues.issueList.Selected = max(0, len(h.issues.issueList.Issues)-1)
			}
			return
		}
	}
}

func (h *home) addWatchlistPR(pr gh.PR) {
	for i, existing := range h.watchlist.prList.PRs {
		if existing.Number == pr.Number {
			h.watchlist.prList.PRs[i] = pr
			return
		}
	}
	h.watchlist.prList.PRs = append(h.watchlist.prList.PRs, pr)
}

func (h *home) removeWatchlistPR(number int) {
	for i, pr := range h.watchlist.prList.PRs {
		if pr.Number == number {
			h.watchlist.prList.PRs = append(h.watchlist.prList.PRs[:i], h.watchlist.prList.PRs[i+1:]...)
			if h.watchlist.prList.Selected >= len(h.watchlist.prList.PRs) {
				h.watchlist.prList.Selected = max(0, len(h.watchlist.prList.PRs)-1)
			}
			return
		}
	}
}

func (h *home) startReview(prNumber int, wtPath string) tea.Cmd {
	h.pr.detailMode = detailReview
	ctx, cancel := context.WithCancel(context.Background())
	h.reviewing[prNumber] = cancel
	h.reviewStep[prNumber] = funReviewMessages[0]
	h.reviewMsgIdx = 0
	h.reconcileClaudeState()
	return tea.Batch(
		h.spinner.Tick,
		runClaudeReviewCmd(ctx, h.cfg, prNumber, wtPath, h.repoDir),
		reviewSpinnerTick(),
	)
}

func (h *home) startTmux(prNumber int, wtPath string) tea.Cmd {
	session, ok := h.tmuxSessions[prNumber]
	if !ok || !session.Exists() {
		session = claude.NewTmuxSession(prNumber, wtPath)
		if err := session.Create(); err != nil {
			h.showError(fmt.Sprintf("Failed to create tmux session: %v", err))
			return clearErrorAfter(3 * time.Second)
		}
		h.tmuxSessions[prNumber] = session
		h.reconcileClaudeState()
	}
	return tea.ExecProcess(session.AttachCmd(), func(err error) tea.Msg {
		if err != nil {
			return tmuxSessionErrorMsg{err: err}
		}
		return nil
	})
}

func (h *home) dispatchPending(prNumber int, wtPath string) tea.Cmd {
	action := h.pending
	h.clearPending()
	switch action {
	case pendingReview:
		return h.startReview(prNumber, wtPath)
	case pendingTmux:
		return h.startTmux(prNumber, wtPath)
	}
	return nil
}

func (h *home) clearPending() {
	h.pending = pendingNone
	h.pendingPR = 0
}

func (h *home) startIssueTmux(issueNumber int, wtPath string) tea.Cmd {
	session, ok := h.issueTmuxSessions[issueNumber]
	if !ok || !session.Exists() {
		session = claude.NewIssueTmuxSession(issueNumber, wtPath)
		if err := session.Create(); err != nil {
			h.showError(fmt.Sprintf("Failed to create tmux session: %v", err))
			return clearErrorAfter(3 * time.Second)
		}
		h.issueTmuxSessions[issueNumber] = session
		h.reconcileIssueTmuxSessions()
	}
	return tea.ExecProcess(session.AttachCmd(), func(err error) tea.Msg {
		if err != nil {
			return tmuxSessionErrorMsg{err: err}
		}
		return nil
	})
}

func (h *home) dispatchIssuePending(issueNumber int, wtPath string) tea.Cmd {
	action := h.pendingIssue
	h.clearIssuePending()
	switch action {
	case pendingIssueTmux:
		return h.startIssueTmux(issueNumber, wtPath)
	}
	return nil
}

func (h *home) clearIssuePending() {
	h.pendingIssue = pendingIssueNone
	h.pendingIssueNum = 0
}

func (h *home) handleFixSelectKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		if h.fixCursor < len(h.fixItems)-1 {
			h.fixCursor++
		}
	case "k", "up":
		if h.fixCursor > 0 {
			h.fixCursor--
		}
	case " ":
		if h.fixCursor < len(h.fixSelected) {
			h.fixSelected[h.fixCursor] = !h.fixSelected[h.fixCursor]
		}
	case "a":
		allSelected := true
		for _, s := range h.fixSelected {
			if !s {
				allSelected = false
				break
			}
		}
		for i := range h.fixSelected {
			h.fixSelected[i] = !allSelected
		}
	case "enter":
		var selected []claude.FixItem
		for i, item := range h.fixItems {
			if h.fixSelected[i] {
				selected = append(selected, item)
			}
		}
		if len(selected) == 0 {
			h.state = stateDefault
			return nil
		}
		pr := h.pr.prList.SelectedPR()
		if pr == nil {
			h.state = stateDefault
			return nil
		}
		wtPath := h.worktrees[pr.Number]
		h.state = stateDefault
		return h.startFix(pr.Number, wtPath, selected)
	case "esc":
		h.state = stateDefault
	}
	return nil
}

func (h *home) startFix(prNumber int, wtPath string, items []claude.FixItem) tea.Cmd {
	h.pr.detailMode = detailFix
	ctx, cancel := context.WithCancel(context.Background())
	h.fixing[prNumber] = cancel
	h.fixStep[prNumber] = funFixMessages[0]
	h.fixMsgIdx = 0
	return tea.Batch(
		h.spinner.Tick,
		runFixCmd(ctx, h.cfg, prNumber, wtPath, items),
		fixSpinnerTick(),
	)
}

func (h *home) viewFixSelectOverlay(base string, height int) string {
	var lines []string
	lines = append(lines, ui.SectionHeaderStyle.Render("Select findings to fix"))
	lines = append(lines, ui.DimStyle.Render("space=toggle  a=all  enter=start  esc=cancel"))
	lines = append(lines, "")

	maxItems := height - 8
	if maxItems < 5 {
		maxItems = 5
	}

	// Compute scroll window
	start := 0
	if h.fixCursor >= maxItems {
		start = h.fixCursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(h.fixItems) {
		end = len(h.fixItems)
	}

	for i := start; i < end; i++ {
		item := h.fixItems[i]
		check := "[ ]"
		if h.fixSelected[i] {
			check = "[x]"
		}
		cursor := "  "
		if i == h.fixCursor {
			cursor = "> "
		}

		sourceTag := fmt.Sprintf("[%s]", item.Source)
		line := fmt.Sprintf("%s%s %s %s", cursor, check, sourceTag, item.Summary)

		// Truncate to fit
		maxWidth := h.width - 10
		if maxWidth > 0 && len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}

		if i == h.fixCursor {
			lines = append(lines, ui.SelectedStyle.Render(line))
		} else {
			lines = append(lines, line)
		}
	}

	if len(h.fixItems) > maxItems {
		lines = append(lines, "")
		lines = append(lines, ui.DimStyle.Render(fmt.Sprintf("  %d/%d items", h.fixCursor+1, len(h.fixItems))))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorCyan).
		Padding(1, 2).
		Width(h.width - 10).
		Render(content)

	return placeOverlay(h.width, height, base, overlay)
}

func (h *home) repoSetupCommand() string {
	if h.cfg == nil {
		return ""
	}
	remoteURL, err := gh.ParseRepoFromDir(h.repoDir)
	if err != nil {
		return ""
	}
	return h.cfg.RepoSetupCommand(remoteURL)
}
