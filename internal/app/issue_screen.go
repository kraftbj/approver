package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/claude"
	gh "github.com/kraft/approver/internal/github"
	"github.com/kraft/approver/internal/ui"
	"github.com/kraft/approver/internal/worktree"
)

// issueDetailMode controls what the right panel displays for issues.
type issueDetailMode int

const (
	issueDetailInfo issueDetailMode = iota
	issueDetailAI
)

// issueScreen handles the Issues screen (assigned issues).
type issueScreen struct {
	issueList  ui.IssueList
	detail     ui.IssueDetail
	detailMode issueDetailMode
	loaded     bool
}

func newIssueScreen() issueScreen {
	return issueScreen{
		issueList: ui.NewIssueList(),
		detail:    ui.NewIssueDetail(),
	}
}

func (s *issueScreen) HandleKey(h *home, key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit

	case "j", "down":
		s.issueList.MoveDown()

	case "k", "up":
		s.issueList.MoveUp()

	case "R":
		h.loading = true
		h.state = stateLoading
		return tea.Batch(h.spinner.Tick, fetchIssuesCmd(h.repoDir, h.cfg.PRLimit))

	case "o":
		issue := s.issueList.SelectedIssue()
		if issue != nil && issue.URL != "" {
			return openInBrowserCmd(issue.URL)
		}

	case "p":
		issue := s.issueList.SelectedIssue()
		if issue != nil && len(issue.LinkedPRs) > 0 {
			return openInBrowserCmd(issue.LinkedPRs[0].URL)
		} else if issue != nil {
			h.showError("No linked PR for this issue")
			return clearErrorAfter(3 * time.Second)
		}

	case "w":
		issue := s.issueList.SelectedIssue()
		if issue != nil {
			if _, exists := h.issueWorktrees[issue.Number]; exists {
				h.showError(fmt.Sprintf("Worktree already exists for issue #%d", issue.Number))
				return clearErrorAfter(3 * time.Second)
			}
			if h.creatingIssueWorktrees[issue.Number] {
				return nil
			}
			h.creatingIssueWorktrees[issue.Number] = true
			h.reconcileIssueWorktrees()
			return tea.Batch(h.spinner.Tick, createIssueWorktreeCmd(h.wtManager, issue.Number, issue.Title))
		}

	case "W":
		issue := s.issueList.SelectedIssue()
		if issue != nil {
			if _, exists := h.issueWorktrees[issue.Number]; !exists {
				h.showError(fmt.Sprintf("No worktree for issue #%d", issue.Number))
				return clearErrorAfter(3 * time.Second)
			}
			h.state = stateConfirm
			h.confirmAction = confirmDeleteIssueWorktree
			h.confirmMsg = fmt.Sprintf("Delete worktree for issue #%d? (y/n)", issue.Number)
		}

	case "a":
		h.state = stateInput
		h.inputAction = inputAddIssue
		h.inputPrompt = "Add issue (number or URL): "
		h.inputBuffer = ""

	case "d":
		issue := s.issueList.SelectedIssue()
		if issue != nil && issue.Source == "manual" {
			h.removeIssue(issue.Number)
			return removeTrackedIssueCmd(issue.Number)
		} else if issue != nil {
			h.showError("Can only remove manually-tracked issues")
			return clearErrorAfter(3 * time.Second)
		}

	case "c":
		issue := s.issueList.SelectedIssue()
		if issue == nil {
			return nil
		}
		if !claude.CheckTmux() {
			h.showError("tmux not found on PATH")
			return clearErrorAfter(3 * time.Second)
		}
		if !claude.CheckClaude() {
			h.showError("claude CLI not found on PATH")
			return clearErrorAfter(3 * time.Second)
		}
		wtPath, exists := h.issueWorktrees[issue.Number]
		if !exists {
			if h.creatingIssueWorktrees[issue.Number] {
				return nil
			}
			h.pendingIssue = pendingIssueTmux
			h.pendingIssueNum = issue.Number
			h.creatingIssueWorktrees[issue.Number] = true
			h.reconcileIssueWorktrees()
			return tea.Batch(h.spinner.Tick, createIssueWorktreeCmd(h.wtManager, issue.Number, issue.Title))
		}
		return h.startIssueTmux(issue.Number, wtPath)

	case "tab":
		switch s.detailMode {
		case issueDetailInfo:
			s.detailMode = issueDetailAI
		default:
			s.detailMode = issueDetailInfo
		}

	case "?":
		h.state = stateHelp
	}

	return nil
}

func (s *issueScreen) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "j/k", Desc: "navigate"},
		{Key: "w", Desc: "worktree"},
		{Key: "c", Desc: "tmux"},
		{Key: "p", Desc: "open PR"},
		{Key: "a", Desc: "add"},
		{Key: "o", Desc: "open"},
		{Key: "R", Desc: "refresh"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

func (s *issueScreen) View(h *home, height int) string {
	if !s.loaded && h.loading {
		return h.viewLoading(height)
	}
	if len(s.issueList.Issues) == 0 {
		content := ui.DimStyle.Render("\n\n  No assigned issues.\n\n  Press R to refresh, a to add an issue manually.")
		return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
	}
	return s.viewDashboard(h, height)
}

func (s *issueScreen) viewDashboard(h *home, height int) string {
	listWidth, detailWidth := screenLayout(h.width)

	s.issueList.SetSize(listWidth, height)
	s.detail.SetSize(detailWidth, height)

	listView := s.issueList.View()

	var detailView string
	issue := s.issueList.SelectedIssue()

	if s.detailMode == issueDetailAI {
		if issue != nil && h.issueTmuxSessions[issue.Number] != nil {
			detailView = s.detail.ViewWorking(issue, h.spinner.View(), "Claude session active")
		} else {
			if issue != nil {
				_, hasWT := h.issueWorktrees[issue.Number]
				msg := "No AI session. Press c to start a Claude tmux session."
				if !hasWT {
					msg = "No AI session. Press c to create a worktree and start Claude."
				}
				sections := fmt.Sprintf("%s\n\n  %s",
					ui.TitleStyle.Render(fmt.Sprintf("#%d AI Session", issue.Number)),
					ui.DimStyle.Render(msg))
				detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(sections)
			} else {
				detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(ui.DimStyle.Render("No issue selected"))
			}
		}
	} else {
		detailView = s.detail.View(issue)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

// Issue-specific pending action
type pendingIssueAction int

const (
	pendingIssueNone pendingIssueAction = iota
	pendingIssueTmux
)

// Issue commands

func fetchIssuesCmd(repoDir string, limit int) tea.Cmd {
	return func() tea.Msg {
		issues, err := gh.FetchIssues(repoDir, limit)
		if err != nil {
			return issuesErrorMsg{err: err}
		}
		return issuesLoadedMsg{issues: issues}
	}
}

func fetchSingleIssueCmd(repoDir, numberOrURL string) tea.Cmd {
	return func() tea.Msg {
		issue, err := gh.FetchIssue(repoDir, numberOrURL)
		if err != nil {
			return issueAddErrorMsg{err: err}
		}
		return issueAddedMsg{issue: *issue}
	}
}

func createIssueWorktreeCmd(mgr *worktree.Manager, issueNumber int, title string) tea.Cmd {
	return func() tea.Msg {
		path, err := mgr.CreateForIssue(issueNumber, title)
		if err != nil {
			return issueWorktreeErrorMsg{issueNumber: issueNumber, err: err}
		}
		return issueWorktreeCreatedMsg{issueNumber: issueNumber, path: path}
	}
}

func deleteIssueWorktreeCmd(mgr *worktree.Manager, issueNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.DeleteIssueWorktree(issueNumber); err != nil {
			return issueWorktreeErrorMsg{issueNumber: issueNumber, err: err}
		}
		return issueWorktreeDeletedMsg{issueNumber: issueNumber}
	}
}

func removeTrackedIssueCmd(issueNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := removeTrackedIssue(issueNumber); err != nil {
			return trackedRemoveErrorMsg{err: err}
		}
		return nil
	}
}
