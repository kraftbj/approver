package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/claude"
	"github.com/kraft/approver/internal/config"
	gh "github.com/kraft/approver/internal/github"
	"github.com/kraft/approver/internal/ui"
	"github.com/kraft/approver/internal/worktree"
)

type state int

const (
	stateDefault state = iota
	stateLoading
	stateConfirm
	stateHelp
	stateInput
)

// detailMode controls what the right panel displays.
type detailMode int

const (
	detailInfo     detailMode = iota // PR metadata
	detailReview                     // Review results
	detailComments                   // PR comments
)

// confirmAction tracks what the confirmation dialog is for.
type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDeleteWorktree
	confirmApprove
)

// inputAction distinguishes what the text input is being used for.
type inputAction int

const (
	inputAddPR          inputAction = iota // Adding a PR by number/URL
	inputRequestChanges                    // Entering reason for request-changes
)

// pendingAction tracks an action deferred until worktree creation completes.
type pendingAction int

const (
	pendingNone pendingAction = iota
	pendingReview
	pendingTmux
)

type home struct {
	width  int
	height int
	state  state

	// UI components
	prList   ui.PRList
	detail   ui.PRDetail
	menu     ui.Menu
	errBox   ui.ErrBox
	spinner  spinner.Model
	loading  bool

	// Confirmation state
	confirmMsg    string
	confirmAction confirmAction

	// Text input state
	inputBuffer string
	inputPrompt string
	inputAction inputAction

	// Working directory (git repo)
	repoDir string

	// Worktree state (pr number -> path)
	worktrees map[int]string

	// Worktree manager
	wtManager *worktree.Manager

	// Detail panel mode
	detailMode detailMode

	// Claude review state
	reviews    map[int]claude.ReviewResult
	reviewing  map[int]context.CancelFunc
	reviewStep map[int]string

	// PR comments cache (pr number -> comments)
	comments map[int][]gh.Comment

	// tmux session state
	tmuxSessions map[int]*claude.TmuxSession

	// Worktree creation in-progress tracking
	creatingWorktrees map[int]bool

	// Pending action (deferred until worktree creation completes)
	pending   pendingAction
	pendingPR int

	// Fun review spinner rotation
	reviewMsgIdx int

	// Config
	cfg *config.Config
}

func newHome() home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorCyan)

	return home{
		state:        stateLoading,
		prList:       ui.NewPRList(),
		detail:       ui.NewPRDetail(),
		menu:         ui.NewMenu(),
		errBox:       ui.NewErrBox(),
		spinner:      s,
		loading:      true,
		worktrees:    make(map[int]string),
		reviews:      make(map[int]claude.ReviewResult),
		reviewing:    make(map[int]context.CancelFunc),
		reviewStep:   make(map[int]string),
		comments:          make(map[int][]gh.Comment),
		tmuxSessions:      make(map[int]*claude.TmuxSession),
		creatingWorktrees: make(map[int]bool),
	}
}

func (h home) Init() tea.Cmd {
	return tea.Batch(
		h.spinner.Tick,
		fetchPRsCmd(h.repoDir),
		scanWorktreesCmd(h.wtManager),
	)
}

func (h home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
		h.layoutPanels()
		return h, nil

	case spinner.TickMsg:
		if h.loading || len(h.reviewing) > 0 || len(h.creatingWorktrees) > 0 {
			var cmd tea.Cmd
			h.spinner, cmd = h.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case prsLoadedMsg:
		h.loading = false
		h.state = stateDefault
		h.prList.SetPRs(msg.prs)
		h.reconcileWorktrees()
		// Also fetch tracked PRs to merge in
		cmds = append(cmds, fetchTrackedPRsCmd(h.repoDir))
		// Schedule background polling
		if h.cfg != nil && h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(h.cfg.PollInterval))
		}

	case trackedPRsLoadedMsg:
		// Merge tracked PRs into the list (dedup by number)
		for _, pr := range msg.prs {
			h.addPRToList(pr)
		}
		h.reconcileWorktrees()

	case prsErrorMsg:
		h.loading = false
		h.state = stateDefault
		h.showError(fmt.Sprintf("Failed to fetch PRs: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case hideErrMsg:
		h.errBox.Clear()

	case worktreeCreatedMsg:
		h.worktrees[msg.prNumber] = msg.path
		delete(h.creatingWorktrees, msg.prNumber)
		h.reconcileWorktrees()
		// Check if per-repo setup is configured
		if setupCmd := h.repoSetupCommand(); setupCmd != "" && h.pending != pendingNone && h.pendingPR == msg.prNumber {
			cmds = append(cmds, runSetupCmd(msg.path, setupCmd, msg.prNumber))
			return h, tea.Batch(cmds...)
		}
		// Dispatch any pending action
		if h.pending != pendingNone && h.pendingPR == msg.prNumber {
			cmd := h.dispatchPending(msg.prNumber, msg.path)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case worktreeDeletedMsg:
		delete(h.worktrees, msg.prNumber)
		h.reconcileWorktrees()

	case worktreesScanMsg:
		h.worktrees = msg.worktrees
		h.reconcileWorktrees()

	case worktreeErrorMsg:
		if msg.prNumber != 0 {
			delete(h.creatingWorktrees, msg.prNumber)
			h.reconcileWorktrees()
		}
		h.clearPending()
		h.showError(fmt.Sprintf("Worktree error: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case prAddedMsg:
		h.addPRToList(msg.pr)
		h.state = stateDefault
		h.inputBuffer = ""
		// Persist the tracked PR
		addTrackedPR(msg.pr.Number)

	case prAddErrorMsg:
		h.state = stateDefault
		h.inputBuffer = ""
		h.showError(fmt.Sprintf("Failed to add PR: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case claudeReviewDoneMsg:
		h.reviews[msg.prNumber] = msg.review
		delete(h.reviewing, msg.prNumber)
		delete(h.reviewStep, msg.prNumber)
		h.reconcileClaudeState()

	case claudeReviewErrorMsg:
		delete(h.reviewing, msg.prNumber)
		delete(h.reviewStep, msg.prNumber)
		h.reconcileClaudeState()
		h.showError(fmt.Sprintf("Review failed for PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case claudeReviewProgressMsg:
		h.reviewStep[msg.prNumber] = msg.step

	case tmuxSessionErrorMsg:
		h.showError(fmt.Sprintf("tmux error: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case reviewSpinnerTickMsg:
		if len(h.reviewing) > 0 {
			h.reviewMsgIdx++
			msg := funReviewMessages[h.reviewMsgIdx%len(funReviewMessages)]
			for prNum := range h.reviewing {
				h.reviewStep[prNum] = msg
			}
			cmds = append(cmds, reviewSpinnerTick())
		}

	case prApprovedMsg:
		h.showError(fmt.Sprintf("PR #%d approved", msg.prNumber))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		cmds = append(cmds, fetchSinglePRCmd(h.repoDir, fmt.Sprintf("%d", msg.prNumber)))

	case prApproveErrorMsg:
		h.showError(fmt.Sprintf("Failed to approve PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case pollTickMsg:
		// Don't poll while loading to avoid stacking requests
		if !h.loading {
			cmds = append(cmds, pollPRsCmd(h.repoDir))
		} else if h.cfg != nil && h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(h.cfg.PollInterval))
		}

	case pollPRsLoadedMsg:
		// Silently merge updated PRs without resetting loading state
		for _, pr := range msg.prs {
			h.addPRToList(pr)
		}
		h.reconcileWorktrees()
		h.reconcileClaudeState()
		// Schedule next poll
		if h.cfg != nil && h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(h.cfg.PollInterval))
		}

	case prChangesRequestedMsg:
		h.showError(fmt.Sprintf("Changes requested on PR #%d", msg.prNumber))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		cmds = append(cmds, fetchSinglePRCmd(h.repoDir, fmt.Sprintf("%d", msg.prNumber)))

	case prChangesRequestErrorMsg:
		h.showError(fmt.Sprintf("Failed to request changes on PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case commentsLoadedMsg:
		h.comments[msg.prNumber] = msg.comments

	case commentsErrorMsg:
		h.showError(fmt.Sprintf("Failed to fetch comments: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case setupDoneMsg:
		if h.pending != pendingNone && h.pendingPR == msg.prNumber {
			cmd := h.dispatchPending(msg.prNumber, msg.wtPath)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case setupErrorMsg:
		h.clearPending()
		h.showError(fmt.Sprintf("Setup failed for PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case tea.KeyMsg:
		cmd := h.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return h, tea.Batch(cmds...)
}

func (h *home) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Global keys
	if key == "ctrl+c" {
		return tea.Quit
	}

	switch h.state {
	case stateHelp:
		// Any key dismisses help
		h.state = stateDefault
		return nil

	case stateConfirm:
		return h.handleConfirmKey(key)

	case stateInput:
		return h.handleInputKey(msg)

	case stateLoading:
		if key == "q" {
			return tea.Quit
		}
		return nil

	default:
		return h.handleDefaultKey(key)
	}
}

func (h *home) handleDefaultKey(key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit

	case "j", "down":
		h.prList.MoveDown()

	case "k", "up":
		h.prList.MoveUp()

	case "R":
		h.loading = true
		h.state = stateLoading
		return tea.Batch(h.spinner.Tick, fetchPRsCmd(h.repoDir))

	case "o":
		pr := h.prList.SelectedPR()
		if pr != nil && pr.URL != "" {
			return openInBrowserCmd(pr.URL)
		}

	case "w":
		pr := h.prList.SelectedPR()
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
		pr := h.prList.SelectedPR()
		if pr != nil {
			if _, exists := h.worktrees[pr.Number]; !exists {
				h.showError(fmt.Sprintf("No worktree for PR #%d", pr.Number))
				return clearErrorAfter(3 * time.Second)
			}
			h.state = stateConfirm
			h.confirmAction = confirmDeleteWorktree
			h.confirmMsg = fmt.Sprintf("Delete worktree for PR #%d? (y/n)", pr.Number)
		}

	case "a":
		h.state = stateInput
		h.inputAction = inputAddPR
		h.inputPrompt = "Add PR (number or URL): "
		h.inputBuffer = ""

	case "d":
		pr := h.prList.SelectedPR()
		if pr != nil && pr.Source == "manual" {
			h.removePR(pr.Number)
			return removeTrackedPRCmd(pr.Number)
		} else if pr != nil {
			h.showError("Can only remove manually-tracked PRs")
			return clearErrorAfter(3 * time.Second)
		}

	case "tab":
		switch h.detailMode {
		case detailInfo:
			h.detailMode = detailReview
		case detailReview:
			h.detailMode = detailComments
			// Fetch comments if not cached
			pr := h.prList.SelectedPR()
			if pr != nil {
				if _, ok := h.comments[pr.Number]; !ok {
					return fetchCommentsCmd(h.repoDir, pr.Number)
				}
			}
		default:
			h.detailMode = detailInfo
		}

	case "c":
		pr := h.prList.SelectedPR()
		if pr == nil {
			return nil
		}
		if !claude.CheckClaude() {
			h.showError("claude CLI not found on PATH")
			return clearErrorAfter(3 * time.Second)
		}
		// If already reviewing, cancel
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
		pr := h.prList.SelectedPR()
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

	case "X":
		pr := h.prList.SelectedPR()
		if pr != nil {
			h.state = stateInput
			h.inputAction = inputRequestChanges
			h.inputPrompt = fmt.Sprintf("Request changes on #%d — Reason: ", pr.Number)
			h.inputBuffer = ""
		}

	case "A":
		pr := h.prList.SelectedPR()
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

func (h *home) handleConfirmKey(key string) tea.Cmd {
	switch key {
	case "y":
		h.state = stateDefault
		action := h.confirmAction
		h.confirmAction = confirmNone
		switch action {
		case confirmDeleteWorktree:
			pr := h.prList.SelectedPR()
			if pr != nil {
				return deleteWorktreeCmd(h.wtManager, pr.Number)
			}
		case confirmApprove:
			pr := h.prList.SelectedPR()
			if pr != nil {
				return approvePRCmd(h.repoDir, pr.Number)
			}
		}
	case "n", "esc":
		h.state = stateDefault
		h.confirmAction = confirmNone
	}
	return nil
}

func (h *home) handleInputKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	switch key {
	case "esc":
		h.state = stateDefault
		h.inputBuffer = ""
		return nil
	case "enter":
		if h.inputBuffer != "" {
			input := h.inputBuffer
			h.inputBuffer = ""
			switch h.inputAction {
			case inputRequestChanges:
				pr := h.prList.SelectedPR()
				h.state = stateDefault
				if pr != nil {
					return requestChangesPRCmd(h.repoDir, pr.Number, input)
				}
				return nil
			default:
				return fetchSinglePRCmd(h.repoDir, input)
			}
		}
		h.state = stateDefault
		return nil
	case "backspace":
		if len(h.inputBuffer) > 0 {
			h.inputBuffer = h.inputBuffer[:len(h.inputBuffer)-1]
		}
		return nil
	default:
		// Accept printable characters and pasted text
		if len(msg.Runes) > 0 {
			h.inputBuffer += string(msg.Runes)
		}
		return nil
	}
}

func (h home) View() string {
	if h.width == 0 || h.height == 0 {
		return ""
	}

	// Reserve space for menu (2 lines) and error (1 line if present)
	menuHeight := 2
	errHeight := 0
	if h.errBox.Message != "" {
		errHeight = 1
	}
	panelHeight := h.height - menuHeight - errHeight

	// Main content area
	var mainContent string
	if h.loading && len(h.prList.PRs) == 0 {
		mainContent = h.viewLoading(panelHeight)
	} else if len(h.prList.PRs) == 0 {
		mainContent = h.viewEmpty(panelHeight)
	} else {
		mainContent = h.viewDashboard(panelHeight)
	}

	// Overlay handling
	if h.state == stateHelp {
		mainContent = h.viewHelp(panelHeight)
	} else if h.state == stateConfirm {
		mainContent = h.viewConfirmOverlay(mainContent, panelHeight)
	} else if h.state == stateInput {
		mainContent = h.viewInputOverlay(mainContent, panelHeight)
	}

	// Menu bar
	var hints []ui.KeyHint
	switch h.state {
	case stateConfirm:
		hints = ui.ConfirmHints()
	case stateInput:
		hints = ui.InputHints()
	default:
		hints = ui.DefaultHints()
	}
	menuView := h.menu.View(hints)

	// Error bar
	errView := h.errBox.View()

	// Compose
	parts := []string{mainContent, menuView}
	if errView != "" {
		parts = append(parts, errView)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (h *home) viewDashboard(height int) string {
	listWidth := h.width * 30 / 100
	detailWidth := h.width - listWidth

	h.prList.SetSize(listWidth, height)
	h.detail.SetSize(detailWidth, height)

	listView := h.prList.View()

	var detailView string
	pr := h.prList.SelectedPR()

	if h.detailMode == detailReview {
		if pr != nil {
			_, hasWT := h.worktrees[pr.Number]
			if _, reviewing := h.reviewing[pr.Number]; reviewing {
				step := h.reviewStep[pr.Number]
				detailView = h.detail.ViewReviewing(pr, h.spinner.View(), step)
			} else if review, ok := h.reviews[pr.Number]; ok {
				data := &ui.ReviewDisplayData{
					Checklist:  review.Agent3Out,
					RawOutput:  review.RawOutput,
					IssueCount: review.IssueCount(),
					HighCount:  review.HighSeverityCount(),
				}
				detailView = h.detail.ViewReview(pr, data, hasWT)
			} else {
				detailView = h.detail.ViewReview(pr, nil, hasWT)
			}
		} else {
			detailView = h.detail.ViewReview(nil, nil, false)
		}
	} else if h.detailMode == detailComments {
		if pr != nil {
			comments := h.comments[pr.Number]
			detailView = h.detail.ViewComments(pr, comments)
		} else {
			detailView = h.detail.ViewComments(nil, nil)
		}
	} else {
		detailView = h.detail.View(pr)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

func (h *home) viewLoading(height int) string {
	content := fmt.Sprintf("\n\n  %s Fetching PRs...", h.spinner.View())
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewEmpty(height int) string {
	content := ui.DimStyle.Render("\n\n  No PRs requesting your review.\n\n  Press R to refresh, a to add a PR manually.")
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewHelp(height int) string {
	helpText := `
  Approver - PR Review Manager

  Navigation:
    j/k, up/down   Navigate PR list
    Tab            Cycle view: info/review/comments
    a              Add PR by number or URL
    d              Remove manually-tracked PR

  Actions:
    w              Create worktree for selected PR
    W              Delete worktree (with confirmation)
    c              Start/cancel Claude review (auto-creates worktree)
    t              Open Claude tmux session (auto-creates worktree)
                     (Ctrl+b d to detach back to Approver)
    A              Approve PR (with confirmation)
    X              Request changes (with reason)
    o              Open PR in browser
    R              Refresh PR list

  General:
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

	// Center the overlay on top of the base
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

// placeOverlay renders an overlay centered on a base view.
func placeOverlay(width, height int, base, overlay string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (h *home) layoutPanels() {
	menuHeight := 2
	panelHeight := h.height - menuHeight

	listWidth := h.width * 30 / 100
	detailWidth := h.width - listWidth

	h.prList.SetSize(listWidth, panelHeight)
	h.detail.SetSize(detailWidth, panelHeight)
	h.menu.SetWidth(h.width)
	h.errBox.SetWidth(h.width)
}

func (h *home) showError(msg string) {
	h.errBox.SetError(msg)
}

func (h *home) reconcileWorktrees() {
	for i := range h.prList.PRs {
		num := h.prList.PRs[i].Number
		_, exists := h.worktrees[num]
		h.prList.PRs[i].HasWorktree = exists
		h.prList.PRs[i].IsCreatingWorktree = h.creatingWorktrees[num]
	}
}

func (h *home) reconcileClaudeState() {
	for i := range h.prList.PRs {
		num := h.prList.PRs[i].Number
		_, hasReview := h.reviews[num]
		_, isReviewing := h.reviewing[num]
		session, hasTmux := h.tmuxSessions[num]
		h.prList.PRs[i].HasReview = hasReview
		h.prList.PRs[i].IsReviewing = isReviewing
		h.prList.PRs[i].HasTmux = hasTmux && session.Exists()
	}
}

func (h *home) addPRToList(pr gh.PR) {
	// Check if already in list (dedup by number)
	for i, existing := range h.prList.PRs {
		if existing.Number == pr.Number {
			h.prList.PRs[i] = pr
			return
		}
	}
	h.prList.PRs = append(h.prList.PRs, pr)
}

func (h *home) removePR(number int) {
	for i, pr := range h.prList.PRs {
		if pr.Number == number {
			h.prList.PRs = append(h.prList.PRs[:i], h.prList.PRs[i+1:]...)
			if h.prList.Selected >= len(h.prList.PRs) {
				h.prList.Selected = max(0, len(h.prList.PRs)-1)
			}
			return
		}
	}
}

// Commands

func fetchPRsCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		prs, err := gh.FetchPRs(repoDir)
		if err != nil {
			return prsErrorMsg{err: err}
		}
		return prsLoadedMsg{prs: prs}
	}
}

func fetchSinglePRCmd(repoDir, numberOrURL string) tea.Cmd {
	return func() tea.Msg {
		pr, err := gh.FetchPR(repoDir, numberOrURL)
		if err != nil {
			return prAddErrorMsg{err: err}
		}
		return prAddedMsg{pr: *pr}
	}
}

func clearErrorAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return hideErrMsg{}
	})
}

func openInBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		// Use macOS "open" directly - gh pr view --web doesn't work
		// inside alt screen since Bubble Tea owns stdout/stderr.
		exec.Command("open", url).Run()
		return nil
	}
}

func createWorktreeCmd(mgr *worktree.Manager, prNumber int, branch string) tea.Cmd {
	return func() tea.Msg {
		path, err := mgr.Create(prNumber, branch)
		if err != nil {
			return worktreeErrorMsg{prNumber: prNumber, err: err}
		}
		return worktreeCreatedMsg{prNumber: prNumber, path: path}
	}
}

func deleteWorktreeCmd(mgr *worktree.Manager, prNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.Delete(prNumber); err != nil {
			return worktreeErrorMsg{prNumber: prNumber, err: err}
		}
		return worktreeDeletedMsg{prNumber: prNumber}
	}
}

func scanWorktreesCmd(mgr *worktree.Manager) tea.Cmd {
	return func() tea.Msg {
		wts, _ := mgr.ScanExisting()
		return worktreesScanMsg{worktrees: wts}
	}
}

// worktreesScanMsg carries the initial worktree scan results.
type worktreesScanMsg struct {
	worktrees map[int]string
}

func pollTick(intervalSeconds int) tea.Cmd {
	return tea.Tick(time.Duration(intervalSeconds)*time.Second, func(time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func pollPRsCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		prs, err := gh.FetchPRs(repoDir)
		if err != nil {
			// Silently ignore poll errors
			return nil
		}
		return pollPRsLoadedMsg{prs: prs}
	}
}

func fetchCommentsCmd(repoDir string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		comments, err := gh.FetchComments(repoDir, prNumber)
		if err != nil {
			return commentsErrorMsg{prNumber: prNumber, err: err}
		}
		return commentsLoadedMsg{prNumber: prNumber, comments: comments}
	}
}

func removeTrackedPRCmd(prNumber int) tea.Cmd {
	return func() tea.Msg {
		removeTrackedPR(prNumber)
		return nil
	}
}

func runClaudeReviewCmd(ctx context.Context, cfg *config.Config, prNumber int, worktreePath, repoDir string) tea.Cmd {
	return func() tea.Msg {
		prompt := ""
		allowedTools := ""
		if cfg != nil {
			prompt = cfg.ReviewPrompt
			allowedTools = cfg.AllowedTools
		}
		result, err := claude.RunReviewPipeline(ctx, worktreePath, repoDir, prNumber, prompt, allowedTools, nil)
		if err != nil {
			return claudeReviewErrorMsg{prNumber: prNumber, err: err}
		}
		return claudeReviewDoneMsg{prNumber: prNumber, review: *result}
	}
}

// Fun review spinner messages, rotated every 4 seconds during review.
var funReviewMessages = []string{
	"Reading the diff with fresh eyes...",
	"Checking for off-by-one errors...",
	"Looking for forgotten TODOs...",
	"Sniffing out race conditions...",
	"Pondering variable names...",
	"Hunting for edge cases...",
	"Scrutinizing error handling...",
	"Questioning every nil check...",
	"Considering the blast radius...",
	"Almost there, double-checking...",
}

// startReview begins the Claude review pipeline for a PR.
func (h *home) startReview(prNumber int, wtPath string) tea.Cmd {
	h.detailMode = detailReview
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

// startTmux creates/reuses a tmux session and hands off the terminal.
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

// dispatchPending fires the deferred action after worktree/setup completes.
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

// clearPending resets the pending action state.
func (h *home) clearPending() {
	h.pending = pendingNone
	h.pendingPR = 0
}

// repoSetupCommand returns the per-repo setup command if configured.
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

func reviewSpinnerTick() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return reviewSpinnerTickMsg{}
	})
}

func requestChangesPRCmd(repoDir string, prNumber int, body string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "pr", "review", fmt.Sprintf("%d", prNumber), "--request-changes", "--body", body)
		if repoDir != "" {
			cmd.Dir = repoDir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return prChangesRequestErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s", string(out))}
		}
		return prChangesRequestedMsg{prNumber: prNumber}
	}
}

func approvePRCmd(repoDir string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "pr", "review", fmt.Sprintf("%d", prNumber), "--approve")
		if repoDir != "" {
			cmd.Dir = repoDir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return prApproveErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s", string(out))}
		}
		return prApprovedMsg{prNumber: prNumber}
	}
}

func runSetupCmd(wtPath, setupCommand string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", setupCommand)
		cmd.Dir = wtPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return setupErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s: %s", err, string(out))}
		}
		return setupDoneMsg{prNumber: prNumber, wtPath: wtPath}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Run() error {
	// Check prerequisites
	if err := gh.CheckGH(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine repo directory
	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := gh.CheckGitRepo(repoDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := config.LoadConfig()

	h := newHome()
	h.repoDir = repoDir
	h.wtManager = worktree.NewManager(repoDir)
	h.cfg = cfg

	p := tea.NewProgram(h, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
