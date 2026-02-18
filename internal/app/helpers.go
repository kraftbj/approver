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

func (h *home) loadingProgress() string {
	total := 0
	for _, repo := range h.repos {
		if repo.HasFeature("prs") {
			total++
		}
		if repo.HasFeature("issues") {
			total++
		}
	}
	loaded := total - h.pendingFetches
	if total > 1 {
		return fmt.Sprintf("Loading... (%d/%d)", loaded, total)
	}
	return "Loading..."
}

func (h *home) viewLoading(height int) string {
	content := fmt.Sprintf("\n\n  %s %s", h.spinner.View(), h.loadingProgress())
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewEmpty(height int) string {
	content := ui.DimStyle.Render("\n\n  No PRs found.\n\n  Press R to refresh, a to add a PR or issue.")
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewHelp(height int) string {
	helpText := `
  Approver - GitHub AI Workflow Hub

  Screens:
    1              PRs (review-requested + authored + assigned)
    2              Issues (assigned + authored)
    3              Settings

  Navigation:
    j/k, up/down   Navigate list
    Tab            Cycle detail panel view

  PRs:
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

  Settings:
    h/l            Switch category
    j/k            Navigate items
    Enter          Edit field / expand repo
    a              Add repo source
    d              Remove repo source

  Common:
    a              Add PR or issue (auto-detects)
    d              Remove item from list
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
		if existing.Number == pr.Number && existing.Repo == pr.Repo {
			h.pr.prList.PRs[i] = pr
			return
		}
	}
	h.pr.prList.PRs = append(h.pr.prList.PRs, pr)
	h.pr.prList.RebuildDisplayOrder()
}

func (h *home) removePR(number int) {
	for i, pr := range h.pr.prList.PRs {
		if pr.Number == number {
			h.pr.prList.PRs = append(h.pr.prList.PRs[:i], h.pr.prList.PRs[i+1:]...)
			if h.pr.prList.Selected >= len(h.pr.prList.PRs) {
				h.pr.prList.Selected = max(0, len(h.pr.prList.PRs)-1)
			}
			h.pr.prList.RebuildDisplayOrder()
			return
		}
	}
}

func (h *home) addIssueToList(issue gh.Issue) {
	for i, existing := range h.issues.issueList.Issues {
		if existing.Number == issue.Number && existing.Repo == issue.Repo {
			h.issues.issueList.Issues[i] = issue
			return
		}
	}
	h.issues.issueList.Issues = append(h.issues.issueList.Issues, issue)
	h.issues.issueList.RebuildDisplayOrder()
}

func (h *home) removeIssue(number int) {
	for i, issue := range h.issues.issueList.Issues {
		if issue.Number == number {
			h.issues.issueList.Issues = append(h.issues.issueList.Issues[:i], h.issues.issueList.Issues[i+1:]...)
			if h.issues.issueList.Selected >= len(h.issues.issueList.Issues) {
				h.issues.issueList.Selected = max(0, len(h.issues.issueList.Issues)-1)
			}
			h.issues.issueList.RebuildDisplayOrder()
			return
		}
	}
}

func (h *home) startReview(key ItemKey, wtPath, repoDir string) tea.Cmd {
	h.pr.detailMode = detailReview
	ctx, cancel := context.WithCancel(context.Background())
	h.reviewing[key] = cancel
	h.reviewStep[key] = funReviewMessages[0]
	h.reviewMsgIdx = 0
	h.reconcileClaudeState()
	return tea.Batch(
		h.spinner.Tick,
		runClaudeReviewCmd(ctx, h.cfg, key, wtPath, repoDir),
		reviewSpinnerTick(),
	)
}

func (h *home) startTmux(key ItemKey, wtPath string) tea.Cmd {
	session, ok := h.tmuxSessions[key]
	if !ok || !session.Exists() {
		session = claude.NewTmuxSession(key.Number, wtPath)
		if err := session.Create(); err != nil {
			h.showError(fmt.Sprintf("Failed to create tmux session: %v", err))
			return clearErrorAfter(3 * time.Second)
		}
		h.tmuxSessions[key] = session
		h.reconcileClaudeState()
	}
	return tea.ExecProcess(session.AttachCmd(), func(err error) tea.Msg {
		if err != nil {
			return tmuxSessionErrorMsg{err: err}
		}
		return nil
	})
}

func (h *home) dispatchPending(key ItemKey, wtPath string) tea.Cmd {
	action := h.pending
	h.clearPending()
	repoDir := h.repoDirForKey(key)
	switch action {
	case pendingReview:
		return h.startReview(key, wtPath, repoDir)
	case pendingTmux:
		return h.startTmux(key, wtPath)
	}
	return nil
}

func (h *home) clearPending() {
	h.pending = pendingNone
	h.pendingKey = ItemKey{}
}

func (h *home) startIssueTmux(key ItemKey, wtPath string) tea.Cmd {
	session, ok := h.issueTmuxSessions[key]
	if !ok || !session.Exists() {
		session = claude.NewIssueTmuxSession(key.Number, wtPath)
		if err := session.Create(); err != nil {
			h.showError(fmt.Sprintf("Failed to create tmux session: %v", err))
			return clearErrorAfter(3 * time.Second)
		}
		h.issueTmuxSessions[key] = session
		h.reconcileIssueTmuxSessions()
	}
	return tea.ExecProcess(session.AttachCmd(), func(err error) tea.Msg {
		if err != nil {
			return tmuxSessionErrorMsg{err: err}
		}
		return nil
	})
}

func (h *home) dispatchIssuePending(key ItemKey, wtPath string) tea.Cmd {
	action := h.pendingIssue
	h.clearIssuePending()
	switch action {
	case pendingIssueTmux:
		return h.startIssueTmux(key, wtPath)
	}
	return nil
}

func (h *home) clearIssuePending() {
	h.pendingIssue = pendingIssueNone
	h.pendingIssueKey = ItemKey{}
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
		k := PRKey(pr)
		wtPath := h.worktrees[k]
		h.state = stateDefault
		return h.startFix(k, wtPath, selected)
	case "esc":
		h.state = stateDefault
	}
	return nil
}

func (h *home) startFix(key ItemKey, wtPath string, items []claude.FixItem) tea.Cmd {
	h.pr.detailMode = detailFix
	ctx, cancel := context.WithCancel(context.Background())
	h.fixing[key] = cancel
	h.fixStep[key] = funFixMessages[0]
	h.fixMsgIdx = 0
	return tea.Batch(
		h.spinner.Tick,
		runFixCmd(ctx, h.cfg, key, wtPath, items),
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

func (h *home) handleRepoSelectKey(key string) tea.Cmd {
	switch key {
	case "j", "down":
		if h.repoSelectCursor < len(h.repos)-1 {
			h.repoSelectCursor++
		}
	case "k", "up":
		if h.repoSelectCursor > 0 {
			h.repoSelectCursor--
		}
	case "enter":
		h.inputRepo = h.repos[h.repoSelectCursor]
		h.state = stateInput
		h.inputAction = h.repoSelectAction
		h.inputPrompt = fmt.Sprintf("[%s] Add PR or issue (number or URL): ", h.inputRepo.Name)
		h.inputBuffer = ""
	case "esc":
		h.state = stateDefault
	}
	return nil
}

func (h *home) viewRepoSelectOverlay(base string, height int) string {
	var lines []string
	lines = append(lines, ui.SectionHeaderStyle.Render("Select repository"))
	lines = append(lines, "")

	for i, repo := range h.repos {
		cursor := "  "
		if i == h.repoSelectCursor {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s", cursor, repo.Name)
		if i == h.repoSelectCursor {
			lines = append(lines, ui.SelectedStyle.Render(line))
		} else {
			lines = append(lines, line)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorCyan).
		Padding(1, 2).
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
