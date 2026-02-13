package app

import (
	"context"
	"fmt"
	"os"
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
	confirmDeleteIssueWorktree
)

// inputAction distinguishes what the text input is being used for.
type inputAction int

const (
	inputAddPR          inputAction = iota // Adding a PR by number/URL
	inputRequestChanges                    // Entering reason for request-changes
	inputAddIssue                          // Adding an issue by number/URL
	inputAddWatchlistPR                    // Adding a PR to the watchlist
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
	active    activeScreen
	pr        prScreen
	issues    issueScreen
	watchlist watchlistScreen

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

	// Issue worktree state (issue number -> path)
	issueWorktrees         map[int]string
	creatingIssueWorktrees map[int]bool

	// Issue tmux sessions
	issueTmuxSessions map[int]*claude.TmuxSession

	// Issue pending action
	pendingIssue    pendingIssueAction
	pendingIssueNum int

	// Watchlist AI summaries (pr number -> summary text)
	watchlistSummaries map[int]string

	// Config
	cfg *config.Config
}

func newHome() home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorCyan)

	return home{
		state:                  stateLoading,
		active:                 screenReviews,
		pr:                     newPRScreen(),
		issues:                 newIssueScreen(),
		watchlist:              newWatchlistScreen(),
		menu:                   ui.NewMenu(),
		errBox:                 ui.NewErrBox(),
		spinner:                s,
		loading:                true,
		worktrees:              make(map[int]string),
		reviews:                make(map[int]claude.ReviewResult),
		reviewing:              make(map[int]context.CancelFunc),
		reviewStep:             make(map[int]string),
		comments:               make(map[int][]gh.Comment),
		prSnapshots:            make(map[int]prSnapshot),
		tmuxSessions:           make(map[int]*claude.TmuxSession),
		creatingWorktrees:      make(map[int]bool),
		issueWorktrees:         make(map[int]string),
		creatingIssueWorktrees: make(map[int]bool),
		issueTmuxSessions:      make(map[int]*claude.TmuxSession),
		watchlistSummaries:     make(map[int]string),
	}
}

func (h home) Init() tea.Cmd {
	return tea.Batch(
		h.spinner.Tick,
		fetchPRsCmd(h.repoDir, h.cfg.PRLimit),
		scanWorktreesCmd(h.wtManager),
		scanIssueWorktreesCmd(h.wtManager),
		loadReviewsCmd(),
		fetchIssuesCmd(h.repoDir, h.cfg.PRLimit),
		fetchWatchlistCmd(h.repoDir),
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
		if h.loading || len(h.reviewing) > 0 || len(h.creatingWorktrees) > 0 || len(h.creatingIssueWorktrees) > 0 {
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
		// Schedule background polling
		if h.cfg != nil && *h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(*h.cfg.PollInterval))
		}

	case trackedPRsLoadedMsg:
		// Tracked PRs go to the watchlist screen, not reviews
		h.watchlist.loaded = true
		if h.active == screenWatchlist && h.loading {
			h.loading = false
			h.state = stateDefault
		}
		h.watchlist.prList.SetPRs(msg.prs)

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
		if session, ok := h.tmuxSessions[msg.prNumber]; ok {
			session.Kill()
			delete(h.tmuxSessions, msg.prNumber)
		}
		h.reconcileWorktrees()
		h.reconcileClaudeState()

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
		if err := addTrackedPR(msg.pr.Number); err != nil {
			h.showError(fmt.Sprintf("Failed to save tracked PR: %v", err))
			cmds = append(cmds, clearErrorAfter(3*time.Second))
		}

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
		if err := saveReviewResult(msg.prNumber, msg.review); err != nil {
			h.showError(fmt.Sprintf("Failed to save review: %v", err))
			cmds = append(cmds, clearErrorAfter(3*time.Second))
		}

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
		h.showInfo(fmt.Sprintf("PR #%d approved", msg.prNumber))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		cmds = append(cmds, fetchSinglePRCmd(h.repoDir, fmt.Sprintf("%d", msg.prNumber)))

	case prApproveErrorMsg:
		h.showError(fmt.Sprintf("Failed to approve PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case branchUpdatedMsg:
		h.showInfo(fmt.Sprintf("Branch updated for PR #%d", msg.prNumber))
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
		h.showInfo(fmt.Sprintf("Branch pushed for PR #%d", msg.prNumber))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case branchPushErrorMsg:
		h.showError(fmt.Sprintf("Failed to push branch for PR #%d: %v", msg.prNumber, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case pollTickMsg:
		if !h.loading {
			cmds = append(cmds, pollPRsCmd(h.repoDir, h.cfg.PRLimit))
		} else if h.cfg != nil && *h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(*h.cfg.PollInterval))
		}

	case pollPRsLoadedMsg:
		for _, pr := range msg.prs {
			h.addPRToList(pr)
		}
		h.reconcileWorktrees()
		h.reconcileClaudeState()
		h.reconcileNotifications()
		h.takeSnapshots()
		if h.cfg != nil && *h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(*h.cfg.PollInterval))
		}

	case prChangesRequestedMsg:
		h.showInfo(fmt.Sprintf("Changes requested on PR #%d", msg.prNumber))
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

	// Issue screen messages

	case issuesLoadedMsg:
		h.issues.loaded = true
		if h.active == screenIssues && h.loading {
			h.loading = false
			h.state = stateDefault
		}
		h.issues.issueList.SetIssues(msg.issues)
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()
		cmds = append(cmds, fetchTrackedIssuesCmd(h.repoDir))

	case trackedIssuesLoadedMsg:
		for _, issue := range msg.issues {
			h.addIssueToList(issue)
		}
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()

	case issuesErrorMsg:
		if h.active == screenIssues && h.loading {
			h.loading = false
			h.state = stateDefault
		}
		h.showError(fmt.Sprintf("Failed to fetch issues: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case issueAddedMsg:
		h.addIssueToList(msg.issue)
		h.state = stateDefault
		h.inputBuffer = ""
		if err := addTrackedIssue(msg.issue.Number); err != nil {
			h.showError(fmt.Sprintf("Failed to save tracked issue: %v", err))
			cmds = append(cmds, clearErrorAfter(3*time.Second))
		}

	case issueAddErrorMsg:
		h.state = stateDefault
		h.inputBuffer = ""
		h.showError(fmt.Sprintf("Failed to add issue: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case issueWorktreeCreatedMsg:
		h.issueWorktrees[msg.issueNumber] = msg.path
		delete(h.creatingIssueWorktrees, msg.issueNumber)
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()
		if h.pendingIssue != pendingIssueNone && h.pendingIssueNum == msg.issueNumber {
			cmd := h.dispatchIssuePending(msg.issueNumber, msg.path)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case issueWorktreeDeletedMsg:
		delete(h.issueWorktrees, msg.issueNumber)
		if session, ok := h.issueTmuxSessions[msg.issueNumber]; ok {
			session.Kill()
			delete(h.issueTmuxSessions, msg.issueNumber)
		}
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()

	case issueWorktreeErrorMsg:
		delete(h.creatingIssueWorktrees, msg.issueNumber)
		h.reconcileIssueWorktrees()
		h.clearIssuePending()
		h.showError(fmt.Sprintf("Issue worktree error: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case issueWorktreesScanMsg:
		h.issueWorktrees = msg.worktrees
		h.reconcileIssueWorktrees()

	// Watchlist screen messages

	case watchlistLoadedMsg:
		h.watchlist.loaded = true
		if h.active == screenWatchlist && h.loading {
			h.loading = false
			h.state = stateDefault
		}
		h.watchlist.prList.SetPRs(msg.prs)

	case watchlistErrorMsg:
		if h.active == screenWatchlist && h.loading {
			h.loading = false
			h.state = stateDefault
		}
		h.showError(fmt.Sprintf("Failed to fetch watchlist: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case watchlistPRAddedMsg:
		h.addWatchlistPR(msg.pr)
		h.state = stateDefault
		h.inputBuffer = ""
		if err := addTrackedPR(msg.pr.Number); err != nil {
			h.showError(fmt.Sprintf("Failed to save tracked PR: %v", err))
			cmds = append(cmds, clearErrorAfter(3*time.Second))
		}

	case watchlistAddErrorMsg:
		h.state = stateDefault
		h.inputBuffer = ""
		h.showError(fmt.Sprintf("Failed to add PR to watchlist: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case browserErrorMsg:
		h.showError(fmt.Sprintf("Failed to open browser: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case trackedRemoveErrorMsg:
		h.showError(fmt.Sprintf("Failed to update tracked list: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

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
		return h.issues.HandleKey(h, key)
	case screenWatchlist:
		return h.watchlist.HandleKey(h, key)
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
				return pushBranchCmd(wtPath, pr.Number)
			}
		case confirmDeleteIssueWorktree:
			issue := h.issues.issueList.SelectedIssue()
			if issue != nil {
				return deleteIssueWorktreeCmd(h.wtManager, issue.Number)
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
			case inputAddIssue:
				h.state = stateDefault
				return fetchSingleIssueCmd(h.repoDir, input)
			case inputAddWatchlistPR:
				h.state = stateDefault
				return fetchSingleWatchlistPRCmd(h.repoDir, input)
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
		mainContent = h.issues.View(&h, panelHeight)
	case screenWatchlist:
		mainContent = h.watchlist.View(&h, panelHeight)
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
		switch h.active {
		case screenReviews:
			hints = h.pr.Hints()
		case screenIssues:
			hints = h.issues.Hints()
		case screenWatchlist:
			hints = h.watchlist.Hints()
		default:
			hints = ui.DefaultHints()
		}
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

func Run() error {
	if err := gh.CheckGH(); err != nil {
		return err
	}

	repoDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if err := gh.CheckGitRepo(repoDir); err != nil {
		return err
	}

	cfg := config.LoadConfig()

	h := newHome()
	h.repoDir = repoDir
	worktreeDir := ""
	if cfg != nil {
		worktreeDir = cfg.WorktreeDir
	}
	wtm, err := worktree.NewManager(repoDir, worktreeDir)
	if err != nil {
		return fmt.Errorf("initializing worktree manager: %w", err)
	}
	h.wtManager = wtm
	h.cfg = cfg

	p := tea.NewProgram(h, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
