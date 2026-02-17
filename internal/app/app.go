package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	stateDefault   state = iota
	stateLoading
	stateConfirm
	stateHelp
	stateInput
	stateFixSelect // Finding selection overlay for fix agent
)

// detailMode controls what the right panel displays.
type detailMode int

const (
	detailInfo     detailMode = iota // PR metadata
	detailReview                     // Review results
	detailComments                   // PR comments
	detailFix                        // Fix agent results
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
	confirmFixCommitPush
)

// inputAction distinguishes what the text input is being used for.
type inputAction int

const (
	inputAddPR          inputAction = iota // Adding a PR by number/URL
	inputRequestChanges                    // Entering reason for request-changes
	inputAddIssue                          // Adding an issue by number/URL
	inputAddItem                           // Auto-detect PR vs issue
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
	issues issueScreen

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

	// Repository configuration
	repoDir    string               // Primary repo (CWD fallback for single-repo mode)
	repos      []config.RepoEntry   // All configured repositories
	wtManagers map[string]*worktree.Manager // Per-repo worktree managers (keyed by repo name)

	// Worktree state
	worktrees         map[ItemKey]string
	creatingWorktrees map[ItemKey]bool

	// Claude review state
	reviews    map[ItemKey]claude.ReviewResult
	reviewing  map[ItemKey]context.CancelFunc
	reviewStep map[ItemKey]string

	// PR comments cache
	comments map[ItemKey][]gh.Comment

	// Notification snapshots for detecting PR state changes
	prSnapshots map[ItemKey]prSnapshot

	// tmux session state
	tmuxSessions map[ItemKey]*claude.TmuxSession

	// Pending action (deferred until worktree creation completes)
	pending    pendingAction
	pendingKey ItemKey

	// Fun review spinner rotation
	reviewMsgIdx int

	// Issue worktree state
	issueWorktrees         map[ItemKey]string
	creatingIssueWorktrees map[ItemKey]bool

	// Issue tmux sessions
	issueTmuxSessions map[ItemKey]*claude.TmuxSession

	// Issue pending action
	pendingIssue    pendingIssueAction
	pendingIssueKey ItemKey


	// Fix agent state
	fixItems    []claude.FixItem              // Available findings for selection
	fixSelected []bool                        // Toggle state per item
	fixCursor   int                           // Cursor position in selection list
	fixing      map[ItemKey]context.CancelFunc // Fix agent in progress
	fixStep     map[ItemKey]string             // Current fix step text
	fixResults  map[ItemKey]string             // Fix agent output
	fixMsgIdx   int                           // Fun fix spinner rotation

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
		menu:                   ui.NewMenu(),
		errBox:                 ui.NewErrBox(),
		spinner:                s,
		loading:                true,
		wtManagers:             make(map[string]*worktree.Manager),
		worktrees:              make(map[ItemKey]string),
		creatingWorktrees:      make(map[ItemKey]bool),
		reviews:                make(map[ItemKey]claude.ReviewResult),
		reviewing:              make(map[ItemKey]context.CancelFunc),
		reviewStep:             make(map[ItemKey]string),
		comments:               make(map[ItemKey][]gh.Comment),
		prSnapshots:            make(map[ItemKey]prSnapshot),
		tmuxSessions:           make(map[ItemKey]*claude.TmuxSession),
		issueWorktrees:         make(map[ItemKey]string),
		creatingIssueWorktrees: make(map[ItemKey]bool),
		issueTmuxSessions:      make(map[ItemKey]*claude.TmuxSession),
		fixing:                 make(map[ItemKey]context.CancelFunc),
		fixStep:                make(map[ItemKey]string),
		fixResults:             make(map[ItemKey]string),
	}
}

func (h home) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, h.spinner.Tick, loadReviewsCmd())

	for _, repo := range h.repos {
		repoName := repo.Name
		repoDir := repo.Dir
		cmds = append(cmds,
			fetchPRsCmd(repoDir, repoName, h.cfg.PRLimit),
			fetchIssuesCmd(repoDir, repoName, h.cfg.PRLimit),
		)
		if mgr, ok := h.wtManagers[repoName]; ok {
			cmds = append(cmds, scanWorktreesCmd(mgr, repoName), scanIssueWorktreesCmd(mgr, repoName))
		}
	}

	// Background weekly cleanup of exclude lists
	cmds = append(cmds, runExcludeCleanupCmd(h.repoDir))

	return tea.Batch(cmds...)
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
		if h.loading || len(h.reviewing) > 0 || len(h.fixing) > 0 || len(h.creatingWorktrees) > 0 || len(h.creatingIssueWorktrees) > 0 {
			var cmd tea.Cmd
			h.spinner, cmd = h.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case prsLoadedMsg:
		h.loading = false
		h.state = stateDefault
		h.pr.prList.SetPRs(filterExcludedPRs(msg.prs))
		h.reconcileWorktrees()
		h.takeSnapshots()
		// Fetch tracked PRs to merge into the list
		cmds = append(cmds, fetchTrackedPRsCmd(h.repos))
		// Schedule background polling
		if h.cfg != nil && *h.cfg.PollInterval > 0 {
			cmds = append(cmds, pollTick(*h.cfg.PollInterval))
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
		h.worktrees[msg.key] = msg.path
		delete(h.creatingWorktrees, msg.key)
		h.reconcileWorktrees()
		if setupCmd := h.repoSetupCommand(); setupCmd != "" && h.pending != pendingNone && h.pendingKey == msg.key {
			cmds = append(cmds, runSetupCmd(msg.path, setupCmd, msg.key))
			return h, tea.Batch(cmds...)
		}
		if h.pending != pendingNone && h.pendingKey == msg.key {
			cmd := h.dispatchPending(msg.key, msg.path)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case worktreeDeletedMsg:
		delete(h.worktrees, msg.key)
		if session, ok := h.tmuxSessions[msg.key]; ok {
			session.Kill()
			delete(h.tmuxSessions, msg.key)
		}
		h.reconcileWorktrees()
		h.reconcileClaudeState()

	case worktreesScanMsg:
		for k, v := range msg.worktrees {
			h.worktrees[k] = v
		}
		h.reconcileWorktrees()

	case worktreeErrorMsg:
		if msg.key.Number != 0 {
			delete(h.creatingWorktrees, msg.key)
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
		for key, review := range msg.reviews {
			h.reviews[key] = review
		}
		h.reconcileClaudeState()

	case claudeReviewDoneMsg:
		h.reviews[msg.key] = msg.review
		delete(h.reviewing, msg.key)
		delete(h.reviewStep, msg.key)
		h.reconcileClaudeState()
		if err := saveReviewResult(msg.key, msg.review); err != nil {
			h.showError(fmt.Sprintf("Failed to save review: %v", err))
			cmds = append(cmds, clearErrorAfter(3*time.Second))
		}

	case claudeReviewErrorMsg:
		delete(h.reviewing, msg.key)
		delete(h.reviewStep, msg.key)
		h.reconcileClaudeState()
		h.showError(fmt.Sprintf("Review failed for PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case claudeReviewProgressMsg:
		h.reviewStep[msg.key] = msg.step

	case tmuxSessionErrorMsg:
		h.showError(fmt.Sprintf("tmux error: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case reviewSpinnerTickMsg:
		if len(h.reviewing) > 0 {
			h.reviewMsgIdx++
			msg := funReviewMessages[h.reviewMsgIdx%len(funReviewMessages)]
			for k := range h.reviewing {
				h.reviewStep[k] = msg
			}
			cmds = append(cmds, reviewSpinnerTick())
		}

	case fixDoneMsg:
		h.fixResults[msg.key] = msg.output
		delete(h.fixing, msg.key)
		delete(h.fixStep, msg.key)
		h.pr.detailMode = detailFix
		h.state = stateConfirm
		h.confirmAction = confirmFixCommitPush
		h.confirmMsg = fmt.Sprintf("Commit and push fixes for PR #%d? (y/n)", msg.key.Number)

	case fixErrorMsg:
		delete(h.fixing, msg.key)
		delete(h.fixStep, msg.key)
		h.showError(fmt.Sprintf("Fix failed for PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case fixProgressMsg:
		h.fixStep[msg.key] = msg.step

	case fixSpinnerTickMsg:
		if len(h.fixing) > 0 {
			h.fixMsgIdx++
			msg := funFixMessages[h.fixMsgIdx%len(funFixMessages)]
			for k := range h.fixing {
				h.fixStep[k] = msg
			}
			cmds = append(cmds, fixSpinnerTick())
		}

	case fixCommitPushDoneMsg:
		h.showInfo(fmt.Sprintf("Fixes committed and pushed for PR #%d", msg.key.Number))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case fixCommitPushErrorMsg:
		h.showError(fmt.Sprintf("Fix commit/push failed for PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case prApprovedMsg:
		h.showInfo(fmt.Sprintf("PR #%d approved", msg.key.Number))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		// Optimistic update — show APPROVED immediately
		for i, pr := range h.pr.prList.PRs {
			if pr.Number == msg.key.Number && pr.Repo == msg.key.Repo {
				h.pr.prList.PRs[i].ReviewDecision = "APPROVED"
				break
			}
		}
		cmds = append(cmds, fetchSinglePRCmd(h.repoDirForKey(msg.key), msg.key, fmt.Sprintf("%d", msg.key.Number)))

	case prApproveErrorMsg:
		h.showError(fmt.Sprintf("Failed to approve PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case branchUpdatedMsg:
		h.showInfo(fmt.Sprintf("Branch updated for PR #%d", msg.key.Number))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		pr := h.pr.prList.SelectedPR()
		if pr != nil && PRKey(pr) == msg.key {
			h.state = stateConfirm
			h.confirmAction = confirmPushBranch
			h.confirmMsg = fmt.Sprintf("Push to origin? (y/n)")
		}

	case branchUpdateErrorMsg:
		h.showError(fmt.Sprintf("Failed to update branch for PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case branchPushedMsg:
		h.showInfo(fmt.Sprintf("Branch pushed for PR #%d", msg.key.Number))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case branchPushErrorMsg:
		h.showError(fmt.Sprintf("Failed to push branch for PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case pollTickMsg:
		if !h.loading {
			for _, repo := range h.repos {
				cmds = append(cmds, pollPRsCmd(repo.Dir, repo.Name, h.cfg.PRLimit))
			}
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
		h.showInfo(fmt.Sprintf("Changes requested on PR #%d", msg.key.Number))
		cmds = append(cmds, clearErrorAfter(3*time.Second))
		cmds = append(cmds, fetchSinglePRCmd(h.repoDirForKey(msg.key), msg.key, fmt.Sprintf("%d", msg.key.Number)))

	case prChangesRequestErrorMsg:
		h.showError(fmt.Sprintf("Failed to request changes on PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	case commentsLoadedMsg:
		h.comments[msg.key] = msg.comments

	case commentsErrorMsg:
		h.showError(fmt.Sprintf("Failed to fetch comments: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case setupDoneMsg:
		if h.pending != pendingNone && h.pendingKey == msg.key {
			cmd := h.dispatchPending(msg.key, msg.wtPath)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case setupErrorMsg:
		h.clearPending()
		h.showError(fmt.Sprintf("Setup failed for PR #%d: %v", msg.key.Number, msg.err))
		cmds = append(cmds, clearErrorAfter(5*time.Second))

	// Issue screen messages

	case issuesLoadedMsg:
		h.issues.loaded = true
		if h.active == screenIssues && h.loading {
			h.loading = false
			h.state = stateDefault
		}
		for _, issue := range filterExcludedIssues(msg.issues) {
			h.addIssueToList(issue)
		}
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()
		cmds = append(cmds, fetchTrackedIssuesCmd(h.repos))

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
		h.issueWorktrees[msg.key] = msg.path
		delete(h.creatingIssueWorktrees, msg.key)
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()
		if h.pendingIssue != pendingIssueNone && h.pendingIssueKey == msg.key {
			cmd := h.dispatchIssuePending(msg.key, msg.path)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case issueWorktreeDeletedMsg:
		delete(h.issueWorktrees, msg.key)
		if session, ok := h.issueTmuxSessions[msg.key]; ok {
			session.Kill()
			delete(h.issueTmuxSessions, msg.key)
		}
		h.reconcileIssueWorktrees()
		h.reconcileIssueTmuxSessions()

	case issueWorktreeErrorMsg:
		delete(h.creatingIssueWorktrees, msg.key)
		h.reconcileIssueWorktrees()
		h.clearIssuePending()
		h.showError(fmt.Sprintf("Issue worktree error: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case issueWorktreesScanMsg:
		for k, v := range msg.worktrees {
			h.issueWorktrees[k] = v
		}
		h.reconcileIssueWorktrees()

	case itemDetectedMsg:
		h.state = stateDefault
		h.inputBuffer = ""
		if msg.pr != nil {
			h.addPRToList(*msg.pr)
			h.active = screenReviews
			for i, p := range h.pr.prList.PRs {
				if p.Number == msg.pr.Number && p.Repo == msg.pr.Repo {
					h.pr.prList.Selected = i
					break
				}
			}
			if err := addTrackedPR(msg.pr.Number); err != nil {
				h.showError(fmt.Sprintf("Failed to save tracked PR: %v", err))
				cmds = append(cmds, clearErrorAfter(3*time.Second))
			}
		} else if msg.issue != nil {
			h.addIssueToList(*msg.issue)
			h.active = screenIssues
			for i, iss := range h.issues.issueList.Issues {
				if iss.Number == msg.issue.Number && iss.Repo == msg.issue.Repo {
					h.issues.issueList.Selected = i
					break
				}
			}
			if err := addTrackedIssue(msg.issue.Number); err != nil {
				h.showError(fmt.Sprintf("Failed to save tracked issue: %v", err))
				cmds = append(cmds, clearErrorAfter(3*time.Second))
			}
		}

	case itemDetectErrorMsg:
		h.state = stateDefault
		h.inputBuffer = ""
		h.showError(fmt.Sprintf("Failed to add item: %v", msg.err))
		cmds = append(cmds, clearErrorAfter(3*time.Second))

	case cleanupDoneMsg:
		// Weekly cleanup completed, nothing to do

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

	case stateFixSelect:
		return h.handleFixSelectKey(key)

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
	}

	// Dispatch to active screen
	switch h.active {
	case screenReviews:
		return h.pr.HandleKey(h, key)
	case screenIssues:
		return h.issues.HandleKey(h, key)
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
				k := PRKey(pr)
				mgr := h.wtManagerForKey(k)
				return deleteWorktreeCmd(mgr, k)
			}
		case confirmApprove:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				k := PRKey(pr)
				return approvePRCmd(pr.RepoDir, k)
			}
		case confirmUpdateBranch:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				k := PRKey(pr)
				wtPath := h.worktrees[k]
				return updateBranchCmd(wtPath, pr.BaseRefName, k)
			}
		case confirmPushBranch:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				k := PRKey(pr)
				wtPath := h.worktrees[k]
				return pushBranchCmd(wtPath, k)
			}
		case confirmDeleteIssueWorktree:
			issue := h.issues.issueList.SelectedIssue()
			if issue != nil {
				k := IssueKey(issue)
				mgr := h.wtManagerForKey(k)
				return deleteIssueWorktreeCmd(mgr, k)
			}
		case confirmFixCommitPush:
			pr := h.pr.prList.SelectedPR()
			if pr != nil {
				k := PRKey(pr)
				wtPath := h.worktrees[k]
				return fixCommitPushCmd(wtPath, k)
			}
		}
	case "n", "esc":
		if h.confirmAction == confirmFixCommitPush {
			h.state = stateDefault
			h.confirmAction = confirmNone
			h.showInfo("Changes left in worktree for manual inspection")
			return clearErrorAfter(3 * time.Second)
		}
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
			// Use first repo as default for manual adds
			defaultRepo := ""
			defaultDir := h.repoDir
			if len(h.repos) > 0 {
				defaultRepo = h.repos[0].Name
				defaultDir = h.repos[0].Dir
			}
			defaultKey := ItemKey{Repo: defaultRepo}
			switch h.inputAction {
			case inputRequestChanges:
				pr := h.pr.prList.SelectedPR()
				h.state = stateDefault
				if pr != nil {
					return requestChangesPRCmd(pr.RepoDir, PRKey(pr), input)
				}
				return nil
			case inputAddIssue:
				h.state = stateDefault
				return fetchSingleIssueCmd(defaultDir, defaultKey, input)
			case inputAddItem:
				h.state = stateDefault
				return detectAndFetchItemCmd(defaultDir, defaultKey, input)
			default:
				return fetchSinglePRCmd(defaultDir, defaultKey, input)
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
	}

	// Overlay handling
	if h.state == stateHelp {
		mainContent = h.viewHelp(panelHeight)
	} else if h.state == stateConfirm {
		mainContent = h.viewConfirmOverlay(mainContent, panelHeight)
	} else if h.state == stateInput {
		mainContent = h.viewInputOverlay(mainContent, panelHeight)
	} else if h.state == stateFixSelect {
		mainContent = h.viewFixSelectOverlay(mainContent, panelHeight)
	}

	// Menu bar with screen indicator
	screenIndicator := ui.ScreenIndicator(int(h.active))
	var hints []ui.KeyHint
	switch h.state {
	case stateConfirm:
		hints = ui.ConfirmHints()
	case stateInput:
		hints = ui.InputHints()
	case stateFixSelect:
		hints = ui.FixSelectHints()
	default:
		switch h.active {
		case screenReviews:
			hints = h.pr.Hints()
		case screenIssues:
			hints = h.issues.Hints()
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

func (h *home) repoDirForKey(key ItemKey) string {
	for _, repo := range h.repos {
		if repo.Name == key.Repo {
			return repo.Dir
		}
	}
	return h.repoDir
}

func (h *home) wtManagerForKey(key ItemKey) *worktree.Manager {
	if mgr, ok := h.wtManagers[key.Repo]; ok {
		return mgr
	}
	// Fallback for single-repo mode
	for _, mgr := range h.wtManagers {
		return mgr
	}
	return nil
}

func Run() error {
	if err := gh.CheckGH(); err != nil {
		return err
	}

	cfg := config.LoadConfig()
	h := newHome()
	h.cfg = cfg

	worktreeDir := ""
	if cfg != nil {
		worktreeDir = cfg.WorktreeDir
	}

	// Resolve repos: from config or fall back to CWD
	if cfg != nil && len(cfg.RepoSources) > 0 {
		repos, err := config.ResolveRepoDirs(cfg.RepoSources)
		if err != nil {
			return fmt.Errorf("resolving repo sources: %w", err)
		}
		if len(repos) == 0 {
			return fmt.Errorf("no valid git repositories found in repo_sources")
		}
		h.repos = repos
		h.repoDir = repos[0].Dir
	} else {
		repoDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		if err := gh.CheckGitRepo(repoDir); err != nil {
			return err
		}
		h.repoDir = repoDir
		h.repos = []config.RepoEntry{{Name: filepath.Base(repoDir), Dir: repoDir}}
	}

	// Create per-repo worktree managers
	for _, repo := range h.repos {
		wtm, err := worktree.NewManager(repo.Dir, worktreeDir, repo.Name)
		if err != nil {
			return fmt.Errorf("initializing worktree manager for %s: %w", repo.Name, err)
		}
		h.wtManagers[repo.Name] = wtm
	}

	p := tea.NewProgram(h, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
