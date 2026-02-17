package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/claude"
	"github.com/kraft/approver/internal/config"
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
		var cmds []tea.Cmd
		cmds = append(cmds, h.spinner.Tick)
		for _, repo := range h.repos {
			cmds = append(cmds, fetchIssuesCmd(repo.Dir, repo.Name, h.cfg.PRLimit))
		}
		return tea.Batch(cmds...)

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
			k := IssueKey(issue)
			if _, exists := h.issueWorktrees[k]; exists {
				h.showError(fmt.Sprintf("Worktree already exists for issue #%d", issue.Number))
				return clearErrorAfter(3 * time.Second)
			}
			if h.creatingIssueWorktrees[k] {
				return nil
			}
			h.creatingIssueWorktrees[k] = true
			h.reconcileIssueWorktrees()
			mgr := h.wtManagerForKey(k)
			return tea.Batch(h.spinner.Tick, createIssueWorktreeCmd(mgr, k, issue.Title))
		}

	case "W":
		issue := s.issueList.SelectedIssue()
		if issue != nil {
			k := IssueKey(issue)
			if _, exists := h.issueWorktrees[k]; !exists {
				h.showError(fmt.Sprintf("No worktree for issue #%d", issue.Number))
				return clearErrorAfter(3 * time.Second)
			}
			h.state = stateConfirm
			h.confirmAction = confirmDeleteIssueWorktree
			h.confirmMsg = fmt.Sprintf("Delete worktree for issue #%d? (y/n)", issue.Number)
		}

	case "a":
		h.state = stateInput
		h.inputAction = inputAddItem
		h.inputPrompt = "Add PR or issue (number or URL): "
		h.inputBuffer = ""

	case "d":
		issue := s.issueList.SelectedIssue()
		if issue != nil {
			if issue.Source == "manual" {
				h.removeIssue(issue.Number)
				return removeTrackedIssueCmd(issue.Number)
			}
			// Auto-fetched: add to exclude list
			if err := addExcludedIssue(issue.Number, issue.Repo); err != nil {
				h.showError(fmt.Sprintf("Failed to exclude issue: %v", err))
				return clearErrorAfter(3 * time.Second)
			}
			h.removeIssue(issue.Number)
		}

	case "c":
		issue := s.issueList.SelectedIssue()
		if issue == nil {
			return nil
		}
		k := IssueKey(issue)
		if !claude.CheckTmux() {
			h.showError("tmux not found on PATH")
			return clearErrorAfter(3 * time.Second)
		}
		if !claude.CheckClaude() {
			h.showError("claude CLI not found on PATH")
			return clearErrorAfter(3 * time.Second)
		}
		wtPath, exists := h.issueWorktrees[k]
		if !exists {
			if h.creatingIssueWorktrees[k] {
				return nil
			}
			h.pendingIssue = pendingIssueTmux
			h.pendingIssueKey = k
			h.creatingIssueWorktrees[k] = true
			h.reconcileIssueWorktrees()
			mgr := h.wtManagerForKey(k)
			return tea.Batch(h.spinner.Tick, createIssueWorktreeCmd(mgr, k, issue.Title))
		}
		return h.startIssueTmux(k, wtPath)

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
		{Key: "d", Desc: "remove"},
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
		content := ui.DimStyle.Render("\n\n  No issues found.\n\n  Press R to refresh, a to add a PR or issue.")
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
		if issue != nil {
			k := IssueKey(issue)
			if h.issueTmuxSessions[k] != nil {
				detailView = s.detail.ViewWorking(issue, h.spinner.View(), "Claude session active")
			} else {
				_, hasWT := h.issueWorktrees[k]
				msg := "No AI session. Press c to start a Claude tmux session."
				if !hasWT {
					msg = "No AI session. Press c to create a worktree and start Claude."
				}
				sections := fmt.Sprintf("%s\n\n  %s",
					ui.TitleStyle.Render(fmt.Sprintf("#%d AI Session", issue.Number)),
					ui.DimStyle.Render(msg))
				detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(sections)
			}
		} else {
			detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(ui.DimStyle.Render("No issue selected"))
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

func fetchIssuesCmd(repoDir, repoName string, limit int) tea.Cmd {
	return func() tea.Msg {
		issues, err := gh.FetchIssues(repoDir, limit)
		if err != nil {
			return issuesErrorMsg{err: err}
		}
		for i := range issues {
			issues[i].Repo = repoName
			issues[i].RepoDir = repoDir
		}
		return issuesLoadedMsg{issues: issues}
	}
}

func fetchSingleIssueCmd(repoDir string, key ItemKey, numberOrURL string) tea.Cmd {
	return func() tea.Msg {
		issue, err := gh.FetchIssue(repoDir, numberOrURL)
		if err != nil {
			return issueAddErrorMsg{err: err}
		}
		issue.Repo = key.Repo
		issue.RepoDir = repoDir
		return issueAddedMsg{issue: *issue}
	}
}

func createIssueWorktreeCmd(mgr *worktree.Manager, key ItemKey, title string) tea.Cmd {
	return func() tea.Msg {
		path, err := mgr.CreateForIssue(key.Number, title)
		if err != nil {
			return issueWorktreeErrorMsg{key: key, err: err}
		}
		return issueWorktreeCreatedMsg{key: key, path: path}
	}
}

func deleteIssueWorktreeCmd(mgr *worktree.Manager, key ItemKey) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.DeleteIssueWorktree(key.Number); err != nil {
			return issueWorktreeErrorMsg{key: key, err: err}
		}
		return issueWorktreeDeletedMsg{key: key}
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

func fetchTrackedIssuesCmd(repos []config.RepoEntry) tea.Cmd {
	return func() tea.Msg {
		tracked, err := loadTrackedIssues()
		if err != nil {
			return issuesErrorMsg{err: fmt.Errorf("loading tracked issues: %w", err)}
		}

		// Use first repo as default for fetching tracked issues
		repoDir := ""
		repoName := ""
		if len(repos) > 0 {
			repoDir = repos[0].Dir
			repoName = repos[0].Name
		}

		var issues []gh.Issue
		for _, t := range tracked {
			fetchDir := repoDir
			fetchName := repoName
			if t.Repo != "" {
				for _, r := range repos {
					if r.Name == t.Repo {
						fetchDir = r.Dir
						fetchName = r.Name
						break
					}
				}
			}
			issue, err := gh.FetchIssue(fetchDir, fmt.Sprintf("%d", t.Number))
			if err != nil {
				continue
			}
			// Drop closed tracked issues
			if issue.State == "CLOSED" {
				_ = removeTrackedIssue(t.Number)
				continue
			}
			issue.Source = "manual"
			issue.Repo = fetchName
			issue.RepoDir = fetchDir
			issues = append(issues, *issue)
		}

		return trackedIssuesLoadedMsg{issues: issues}
	}
}
