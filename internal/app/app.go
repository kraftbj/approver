package app

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// confirmAction tracks what the confirmation dialog is for.
type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDeleteWorktree
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

	// Text input state (for adding PRs)
	inputBuffer string
	inputPrompt string

	// Working directory (git repo)
	repoDir string

	// Worktree state (pr number -> path)
	worktrees map[int]string

	// Worktree manager
	wtManager *worktree.Manager
}

func newHome() home {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorCyan)

	return home{
		state:     stateLoading,
		prList:    ui.NewPRList(),
		detail:    ui.NewPRDetail(),
		menu:      ui.NewMenu(),
		errBox:    ui.NewErrBox(),
		spinner:   s,
		loading:   true,
		worktrees: make(map[int]string),
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
		if h.loading {
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
		h.reconcileWorktrees()

	case worktreeDeletedMsg:
		delete(h.worktrees, msg.prNumber)
		h.reconcileWorktrees()

	case worktreesScanMsg:
		h.worktrees = msg.worktrees
		h.reconcileWorktrees()

	case worktreeErrorMsg:
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
		if pr != nil {
			return openInBrowserCmd(pr.Number, h.repoDir)
		}

	case "w":
		pr := h.prList.SelectedPR()
		if pr != nil {
			if _, exists := h.worktrees[pr.Number]; exists {
				h.showError(fmt.Sprintf("Worktree already exists for PR #%d", pr.Number))
				return clearErrorAfter(3 * time.Second)
			}
			return createWorktreeCmd(h.wtManager, pr.Number, pr.HeadRefName)
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

	case "?":
		h.state = stateHelp
	}

	return nil
}

func (h *home) handleConfirmKey(key string) tea.Cmd {
	switch key {
	case "y":
		h.state = stateDefault
		if h.confirmAction == confirmDeleteWorktree {
			pr := h.prList.SelectedPR()
			if pr != nil {
				return deleteWorktreeCmd(h.wtManager, pr.Number)
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
			return fetchSinglePRCmd(h.repoDir, input)
		}
		h.state = stateDefault
		return nil
	case "backspace":
		if len(h.inputBuffer) > 0 {
			h.inputBuffer = h.inputBuffer[:len(h.inputBuffer)-1]
		}
		return nil
	default:
		// Only accept printable characters
		if len(key) == 1 || key == "/" || key == ":" {
			h.inputBuffer += key
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
	detailView := h.detail.View(h.prList.SelectedPR())

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
    a              Add PR by number or URL
    d              Remove manually-tracked PR

  Actions:
    w              Create worktree for selected PR
    W              Delete worktree (with confirmation)
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
		_, exists := h.worktrees[h.prList.PRs[i].Number]
		h.prList.PRs[i].HasWorktree = exists
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

func openInBrowserCmd(prNumber int, repoDir string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--web")
		if repoDir != "" {
			cmd.Dir = repoDir
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		return nil
	}
}

func createWorktreeCmd(mgr *worktree.Manager, prNumber int, branch string) tea.Cmd {
	return func() tea.Msg {
		path, err := mgr.Create(prNumber, branch)
		if err != nil {
			return worktreeErrorMsg{err: err}
		}
		return worktreeCreatedMsg{prNumber: prNumber, path: path}
	}
}

func deleteWorktreeCmd(mgr *worktree.Manager, prNumber int) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.Delete(prNumber); err != nil {
			return worktreeErrorMsg{err: err}
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

func removeTrackedPRCmd(prNumber int) tea.Cmd {
	return func() tea.Msg {
		removeTrackedPR(prNumber)
		return nil
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

	h := newHome()
	h.repoDir = repoDir
	h.wtManager = worktree.NewManager(repoDir)

	p := tea.NewProgram(h, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
