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
	confirmUpdateBranch
	confirmPushBranch
)

// inputAction distinguishes what the text input is being used for.
type inputAction int

const (
	inputAddPR          inputAction = iota // Adding a PR by number/URL
	inputRequestChanges                    // Entering reason for request-changes
)

// prSnapshot captures PR state for change detection between polls.
type prSnapshot struct {
	commentCount   int
	ciStatus       string
	reviewDecision string
}

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

	// Screen management
	active activeScreen
	pr     prScreen

	// UI components
	menu    ui.Menu
	errBox  ui.ErrBox
	spinner spinner.Model
	loading bool

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

	// Claude review state
	reviews    map[int]claude.ReviewResult
	reviewing  map[int]context.CancelFunc
	reviewStep map[int]string

	// PR comments cache (pr number -> comments)
	comments map[int][]gh.Comment

	// Notification snapshots for detecting PR state changes
	prSnapshots map[int]prSnapshot

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
		state:             stateLoading,
		active:            screenReviews,
		pr:                newPRScreen(),
		menu:              ui.NewMenu(),
		errBox:            ui.NewErrBox(),
		spinner:           s,
		loading:           true,
		worktrees:         make(map[int]string),
		reviews:           make(map[int]claude.ReviewResult),
		reviewing:         make(map[int]context.CancelFunc),
		reviewStep:        make(map[int]string),
		comments:          make(map[int][]gh.Comment),
		prSnapshots:       make(map[int]prSnapshot),
		tmuxSessions:      make(map[int]*claude.TmuxSession),
		creatingWorktrees: make(map[int]bool),
	}
}

func (h home) Init() tea.Cmd {
	return tea.Batch(
		h.spinner.Tick,
		fetchPRsCmd(h.repoDir),
		scanWorktreesCmd(h.wtManager),
		loadReviewsCmd(),
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
		h.pr.prList.SetPRs(msg.prs)
		h.reconcileWorktrees()
		h.takeSnapshots()
		// Also fetch tracked PRs to merge in
		cmds = append(cmds, fetchTrackedPRsCmd(h.repoDir))
		// Schedule background polling
		if h.cfg != nil && h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(h.cfg.PollInterval))
		}

	case trackedPRsLoadedMsg:
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
		if setupCmd := h.repoSetupCommand(); setupCmd != "" && h.pending != pendingNone && h.pendingPR == msg.prNumber {
			cmds = append(cmds, runSetupCmd(msg.path, setupCmd, msg.prNumber))
			return h, tea.Batch(cmds...)
		}
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
		addTrackedPR(msg.pr.Number)

	case prAddErrorMsg:
		h.state = stateDefault
		h.inputBuffer = ""
		h.showError(fmt.Sprintf("Failed to add PR: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case reviewsLoadedMsg:
		for prNumber, review := range msg.reviews {
			h.reviews[prNumber] = review
		}
		h.reconcileClaudeState()

	case claudeReviewDoneMsg:
		h.reviews[msg.prNumber] = msg.review
		delete(h.reviewing, msg.prNumber)
		delete(h.reviewStep, msg.prNumber)
		h.reconcileClaudeState()
		saveReviewResult(msg.prNumber, msg.review)

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

	case branchUpdatedMsg:
		h.showError(fmt.Sprintf("Branch updated for PR #%d", msg.prNumber))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		pr := h.pr.prList.SelectedPR()
		if pr != nil && pr.Number == msg.prNumber {
			h.state = stateConfirm
			h.confirmAction = confirmPushBranch
			h.confirmMsg = fmt.Sprintf("Push to origin? (y/n)")
		}

	case branchUpdateErrorMsg:
		h.showError(fmt.Sprintf("Failed to update branch for PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case branchPushedMsg:
		h.showError(fmt.Sprintf("Branch pushed for PR #%d", msg.prNumber))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case branchPushErrorMsg:
		h.showError(fmt.Sprintf("Failed to push branch for PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case pollTickMsg:
		if !h.loading {
			cmds = append(cmds, pollPRsCmd(h.repoDir))
		} else if h.cfg != nil && h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(h.cfg.PollInterval))
		}

	case pollPRsLoadedMsg:
		for _, pr := range msg.prs {
			h.addPRToList(pr)
		}
		h.reconcileWorktrees()
		h.reconcileClaudeState()
		h.reconcileNotifications()
		h.takeSnapshots()
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

	if key == "ctrl+c" {
		return tea.Quit
	}

	switch h.state {
	case stateHelp:
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
	// Screen switching (global, before dispatch)
	switch key {
	case "1":
		h.active = screenReviews
		return nil
	case "2":
		h.active = screenIssues
		return nil
	case "3":
		h.active = screenWatchlist
		return nil
	}

	// Dispatch to active screen
	switch h.active {
	case screenReviews:
		return h.pr.HandleKey(h, key)
	case screenIssues:
		return h.handlePlaceholderKey(key)
	case screenWatchlist:
		return h.handlePlaceholderKey(key)
	}
	return nil
}

// handlePlaceholderKey handles keys for screens not yet implemented.
func (h *home) handlePlaceholderKey(key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit
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
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				return deleteWorktreeCmd(h.wtManager, pr.Number)
			}
		case confirmApprove:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				return approvePRCmd(h.repoDir, pr.Number)
			}
		case confirmUpdateBranch:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				wtPath := h.worktrees[pr.Number]
				return updateBranchCmd(wtPath, pr.BaseRefName, pr.Number)
			}
		case confirmPushBranch:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				wtPath := h.worktrees[pr.Number]
				return pushBranchCmd(wtPath, pr.HeadRefName, pr.Number)
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
				pr := h.pr.prList.SelectedPR()
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

	// Main content area — dispatch to active screen
	var mainContent string
	switch h.active {
	case screenReviews:
		mainContent = h.pr.View(&h, panelHeight)
	case screenIssues:
		mainContent = h.viewPlaceholder("Issues", "Assigned issues will appear here.", panelHeight)
	case screenWatchlist:
		mainContent = h.viewPlaceholder("Watchlist", "Tracked PRs will appear here.", panelHeight)
	}

	// Overlay handling
	if h.state == stateHelp {
		mainContent = h.viewHelp(panelHeight)
	} else if h.state == stateConfirm {
		mainContent = h.viewConfirmOverlay(mainContent, panelHeight)
	} else if h.state == stateInput {
		mainContent = h.viewInputOverlay(mainContent, panelHeight)
	}

	// Menu bar with screen indicator
	screenIndicator := ui.ScreenIndicator(int(h.active))
	var hints []ui.KeyHint
	switch h.state {
	case stateConfirm:
		hints = ui.ConfirmHints()
	case stateInput:
		hints = ui.InputHints()
	default:
		hints = ui.DefaultHints()
	}
	menuView := h.menu.ViewWithScreen(hints, screenIndicator)

	// Error bar
	errView := h.errBox.View()

	// Compose
	parts := []string{mainContent, menuView}
	if errView != "" {
		parts = append(parts, errView)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (h *home) viewLoading(height int) string {
	content := fmt.Sprintf("\n\n  %s Fetching PRs...", h.spinner.View())
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewEmpty(height int) string {
	content := ui.DimStyle.Render("\n\n  No PRs requesting your review.\n\n  Press R to refresh.")
	return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
}

func (h *home) viewPlaceholder(title, subtitle string, height int) string {
	content := ui.DimStyle.Render(fmt.Sprintf("\n\n  %s\n\n  %s\n\n  Press 1/2/3 to switch screens.", title, subtitle))
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
    Tab            Cycle view: info/review/comments

  Actions:
    w              Create worktree for selected PR
    W              Delete worktree (with confirmation)
    c              Start/cancel Claude review (auto-creates worktree)
    t              Open Claude tmux session (auto-creates worktree)
                     (Ctrl+b d to detach back to Approver)
    A              Approve PR (with confirmation)
    X              Request changes (with reason)
    u              Update branch (merge base into worktree)
    o              Open in browser
    R              Refresh

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
	panelHeight := h.height - menuHeight

	listWidth := h.width * 30 / 100
	detailWidth := h.width - listWidth

	h.pr.prList.SetSize(listWidth, panelHeight)
	h.pr.detail.SetSize(detailWidth, panelHeight)
	h.menu.SetWidth(h.width)
	h.errBox.SetWidth(h.width)
}

func (h *home) showError(msg string) {
	h.errBox.SetError(msg)
}

func (h *home) reconcileWorktrees() {
	for i := range h.pr.prList.PRs {
		num := h.pr.prList.PRs[i].Number
		_, exists := h.worktrees[num]
		h.pr.prList.PRs[i].HasWorktree = exists
		h.pr.prList.PRs[i].IsCreatingWorktree = h.creatingWorktrees[num]
	}
}

func (h *home) reconcileClaudeState() {
	for i := range h.pr.prList.PRs {
		num := h.pr.prList.PRs[i].Number
		_, hasReview := h.reviews[num]
		_, isReviewing := h.reviewing[num]
		session, hasTmux := h.tmuxSessions[num]
		h.pr.prList.PRs[i].HasReview = hasReview
		h.pr.prList.PRs[i].IsReviewing = isReviewing
		h.pr.prList.PRs[i].HasTmux = hasTmux && session.Exists()
	}
}

func (h *home) takeSnapshots() {
	for _, pr := range h.pr.prList.PRs {
		h.prSnapshots[pr.Number] = prSnapshot{
			commentCount:   pr.CommentCount,
			ciStatus:       pr.CIStatus(),
			reviewDecision: pr.ReviewDecision,
		}
	}
}

func (h *home) reconcileNotifications() {
	for i := range h.pr.prList.PRs {
		pr := &h.pr.prList.PRs[i]
		snap, exists := h.prSnapshots[pr.Number]
		if !exists {
			continue
		}
		if pr.CommentCount != snap.commentCount ||
			pr.CIStatus() != snap.ciStatus ||
			pr.ReviewDecision != snap.reviewDecision {
			pr.HasNotification = true
		}
	}
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

func updateBranchCmd(wtPath, baseBranch string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		fetchCmd := exec.Command("git", "-C", wtPath, "fetch", "origin", baseBranch)
		if out, err := fetchCmd.CombinedOutput(); err != nil {
			return branchUpdateErrorMsg{prNumber: prNumber, err: fmt.Errorf("fetch: %s", string(out))}
		}
		mergeCmd := exec.Command("git", "-C", wtPath, "merge", fmt.Sprintf("origin/%s", baseBranch))
		if out, err := mergeCmd.CombinedOutput(); err != nil {
			return branchUpdateErrorMsg{prNumber: prNumber, err: fmt.Errorf("merge: %s", string(out))}
		}
		return branchUpdatedMsg{prNumber: prNumber}
	}
}

func pushBranchCmd(wtPath, headBranch string, prNumber int) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "-C", wtPath, "push", "origin", "HEAD")
		if out, err := cmd.CombinedOutput(); err != nil {
			return branchPushErrorMsg{prNumber: prNumber, err: fmt.Errorf("%s", string(out))}
		}
		return branchPushedMsg{prNumber: prNumber}
	}
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
	if err := gh.CheckGH(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

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
