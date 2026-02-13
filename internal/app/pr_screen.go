package app

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/claude"
	"github.com/kraft/approver/internal/ui"
)

// prScreen handles the Reviews screen (PRs requesting your review).
type prScreen struct {
	prList     ui.PRList
	detail     ui.PRDetail
	detailMode detailMode
}

func newPRScreen() prScreen {
	return prScreen{
		prList: ui.NewPRList(),
		detail: ui.NewPRDetail(),
	}
}

func (s *prScreen) HandleKey(h *home, key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit

	case "j", "down":
		s.prList.MoveDown()
		if pr := s.prList.SelectedPR(); pr != nil {
			pr.HasNotification = false
		}

	case "k", "up":
		s.prList.MoveUp()
		if pr := s.prList.SelectedPR(); pr != nil {
			pr.HasNotification = false
		}

	case "R":
		h.loading = true
		h.state = stateLoading
		return tea.Batch(h.spinner.Tick, fetchPRsCmd(h.repoDir, h.cfg.PRLimit))

	case "o":
		pr := s.prList.SelectedPR()
		if pr != nil && pr.URL != "" {
			return openInBrowserCmd(pr.URL)
		}

	case "w":
		pr := s.prList.SelectedPR()
		if pr != nil {
			if _, exists := h.worktrees[pr.Number]; exists {
				h.showError(fmt.Sprintf("Worktree already exists for PR #%d", pr.Number))
				return clearErrorAfter(3 * time.Second)
			}
			if h.creatingWorktrees[pr.Number] {
				return nil
			}
			h.creatingWorktrees[pr.Number] = true
			h.reconcileWorktrees()
			return tea.Batch(h.spinner.Tick, createWorktreeCmd(h.wtManager, pr.Number, pr.HeadRefName))
		}

	case "W":
		pr := s.prList.SelectedPR()
		if pr != nil {
			if _, exists := h.worktrees[pr.Number]; !exists {
				h.showError(fmt.Sprintf("No worktree for PR #%d", pr.Number))
				return clearErrorAfter(3 * time.Second)
			}
			h.state = stateConfirm
			h.confirmAction = confirmDeleteWorktree
			h.confirmMsg = fmt.Sprintf("Delete worktree for PR #%d? (y/n)", pr.Number)
		}

	case "tab":
		switch s.detailMode {
		case detailInfo:
			s.detailMode = detailReview
		case detailReview:
			s.detailMode = detailComments
			pr := s.prList.SelectedPR()
			if pr != nil {
				if _, ok := h.comments[pr.Number]; !ok {
					return fetchCommentsCmd(h.repoDir, pr.Number)
				}
			}
		default:
			s.detailMode = detailInfo
		}

	case "c":
		pr := s.prList.SelectedPR()
		if pr == nil {
			return nil
		}
		if !claude.CheckClaude() {
			h.showError("claude CLI not found on PATH")
			return clearErrorAfter(3 * time.Second)
		}
		if cancel, running := h.reviewing[pr.Number]; running {
			cancel()
			delete(h.reviewing, pr.Number)
			delete(h.reviewStep, pr.Number)
			h.reconcileClaudeState()
			return nil
		}
		wtPath, exists := h.worktrees[pr.Number]
		if !exists {
			if h.creatingWorktrees[pr.Number] {
				return nil
			}
			h.pending = pendingReview
			h.pendingPR = pr.Number
			h.creatingWorktrees[pr.Number] = true
			h.reconcileWorktrees()
			return tea.Batch(h.spinner.Tick, createWorktreeCmd(h.wtManager, pr.Number, pr.HeadRefName))
		}
		return h.startReview(pr.Number, wtPath)

	case "t":
		pr := s.prList.SelectedPR()
		if pr == nil {
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
		wtPath, exists := h.worktrees[pr.Number]
		if !exists {
			if h.creatingWorktrees[pr.Number] {
				return nil
			}
			h.pending = pendingTmux
			h.pendingPR = pr.Number
			h.creatingWorktrees[pr.Number] = true
			h.reconcileWorktrees()
			return tea.Batch(h.spinner.Tick, createWorktreeCmd(h.wtManager, pr.Number, pr.HeadRefName))
		}
		return h.startTmux(pr.Number, wtPath)

	case "u":
		pr := s.prList.SelectedPR()
		if pr != nil {
			if _, exists := h.worktrees[pr.Number]; !exists {
				h.showError(fmt.Sprintf("No worktree for PR #%d — create one first with w", pr.Number))
				return clearErrorAfter(3 * time.Second)
			}
			h.state = stateConfirm
			h.confirmAction = confirmUpdateBranch
			h.confirmMsg = fmt.Sprintf("Merge origin/%s into worktree for PR #%d? (y/n)", pr.BaseRefName, pr.Number)
		}

	case "X":
		pr := s.prList.SelectedPR()
		if pr != nil {
			h.state = stateInput
			h.inputAction = inputRequestChanges
			h.inputPrompt = fmt.Sprintf("Request changes on #%d — Reason: ", pr.Number)
			h.inputBuffer = ""
		}

	case "A":
		pr := s.prList.SelectedPR()
		if pr != nil {
			h.state = stateConfirm
			h.confirmAction = confirmApprove
			h.confirmMsg = fmt.Sprintf("Approve PR #%d? (y/n)", pr.Number)
		}

	case "?":
		h.state = stateHelp
	}

	return nil
}

func (s *prScreen) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "j/k", Desc: "navigate"},
		{Key: "w", Desc: "worktree"},
		{Key: "c", Desc: "review"},
		{Key: "t", Desc: "tmux"},
		{Key: "A", Desc: "approve"},
		{Key: "o", Desc: "open"},
		{Key: "R", Desc: "refresh"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

func (s *prScreen) View(h *home, height int) string {
	if h.loading && len(s.prList.PRs) == 0 {
		return h.viewLoading(height)
	}
	if len(s.prList.PRs) == 0 {
		return h.viewEmpty(height)
	}
	return s.viewDashboard(h, height)
}

func (s *prScreen) viewDashboard(h *home, height int) string {
	listWidth, detailWidth := screenLayout(h.width)

	s.prList.SetSize(listWidth, height)
	s.detail.SetSize(detailWidth, height)

	listView := s.prList.View()

	var detailView string
	pr := s.prList.SelectedPR()

	if s.detailMode == detailReview {
		if pr != nil {
			_, hasWT := h.worktrees[pr.Number]
			if _, reviewing := h.reviewing[pr.Number]; reviewing {
				step := h.reviewStep[pr.Number]
				detailView = s.detail.ViewReviewing(pr, h.spinner.View(), step)
			} else if review, ok := h.reviews[pr.Number]; ok {
				data := &ui.ReviewDisplayData{
					Checklist:  review.Agent3Out,
					RawOutput:  review.RawOutput,
					IssueCount: review.IssueCount(),
					HighCount:  review.HighSeverityCount(),
				}
				detailView = s.detail.ViewReview(pr, data, hasWT)
			} else {
				detailView = s.detail.ViewReview(pr, nil, hasWT)
			}
		} else {
			detailView = s.detail.ViewReview(nil, nil, false)
		}
	} else if s.detailMode == detailComments {
		if pr != nil {
			comments := h.comments[pr.Number]
			detailView = s.detail.ViewComments(pr, comments)
		} else {
			detailView = s.detail.ViewComments(nil, nil)
		}
	} else {
		detailView = s.detail.View(pr)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}
