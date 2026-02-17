package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/config"
	gh "github.com/kraft/approver/internal/github"
	"github.com/kraft/approver/internal/ui"
)

// watchlistDetailMode controls what the right panel displays for watchlist PRs.
type watchlistDetailMode int

const (
	watchlistDetailInfo    watchlistDetailMode = iota
	watchlistDetailSummary                     // AI summary
)

// watchlistScreen handles the Watchlist screen (manually tracked PRs).
type watchlistScreen struct {
	prList     ui.PRList
	detail     ui.PRDetail
	detailMode watchlistDetailMode
	loaded     bool
}

func newWatchlistScreen() watchlistScreen {
	return watchlistScreen{
		prList: ui.NewPRList(),
		detail: ui.NewPRDetail(),
	}
}

func (s *watchlistScreen) HandleKey(h *home, key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit

	case "j", "down":
		s.prList.MoveDown()

	case "k", "up":
		s.prList.MoveUp()

	case "R":
		h.loading = true
		h.state = stateLoading
		return tea.Batch(h.spinner.Tick, fetchWatchlistCmd(h.repos))

	case "o":
		pr := s.prList.SelectedPR()
		if pr != nil && pr.URL != "" {
			return openInBrowserCmd(pr.URL)
		}

	case "a":
		h.state = stateInput
		h.inputAction = inputAddWatchlistPR
		h.inputPrompt = "Add PR to watchlist (number or URL): "
		h.inputBuffer = ""

	case "d":
		pr := s.prList.SelectedPR()
		if pr != nil {
			h.removeWatchlistPR(pr.Number)
			return removeTrackedPRCmd(pr.Number)
		}

	case "tab":
		switch s.detailMode {
		case watchlistDetailInfo:
			s.detailMode = watchlistDetailSummary
		default:
			s.detailMode = watchlistDetailInfo
		}

	case "?":
		h.state = stateHelp
	}

	return nil
}

func (s *watchlistScreen) Hints() []ui.KeyHint {
	return []ui.KeyHint{
		{Key: "j/k", Desc: "navigate"},
		{Key: "a", Desc: "add"},
		{Key: "d", Desc: "remove"},
		{Key: "o", Desc: "open"},
		{Key: "R", Desc: "refresh"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

func (s *watchlistScreen) View(h *home, height int) string {
	if !s.loaded && h.loading {
		return h.viewLoading(height)
	}
	if len(s.prList.PRs) == 0 {
		content := ui.DimStyle.Render("\n\n  No watched PRs.\n\n  Press a to add a PR to the watchlist.")
		return lipgloss.NewStyle().Width(h.width).Height(height).Render(content)
	}
	return s.viewDashboard(h, height)
}

func (s *watchlistScreen) viewDashboard(h *home, height int) string {
	listWidth, detailWidth := screenLayout(h.width)

	s.prList.SetSize(listWidth, height)
	s.detail.SetSize(detailWidth, height)

	listView := s.prList.View()

	var detailView string
	pr := s.prList.SelectedPR()

	if s.detailMode == watchlistDetailSummary {
		if pr != nil {
			k := PRKey(pr)
			if summary, ok := h.watchlistSummaries[k]; ok {
				sections := fmt.Sprintf("%s\n\n  %s",
					ui.TitleStyle.Render(fmt.Sprintf("#%d Summary", pr.Number)),
					summary)
				detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(sections)
			} else {
				msg := "No AI summary. Press s to generate one."
				sections := fmt.Sprintf("%s\n\n  %s",
					ui.TitleStyle.Render(fmt.Sprintf("#%d Summary", pr.Number)),
					ui.DimStyle.Render(msg))
				detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(sections)
			}
		} else {
			detailView = ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(ui.DimStyle.Render("No PR selected"))
		}
	} else {
		detailView = s.detail.View(pr)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)
}

// Watchlist commands

func fetchWatchlistCmd(repos []config.RepoEntry) tea.Cmd {
	return func() tea.Msg {
		tracked, err := loadTrackedPRs()
		if err != nil {
			return watchlistErrorMsg{err: fmt.Errorf("loading watchlist: %w", err)}
		}

		// Use first repo as default
		repoDir := ""
		repoName := ""
		if len(repos) > 0 {
			repoDir = repos[0].Dir
			repoName = repos[0].Name
		}

		var prs []gh.PR
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
			pr, err := gh.FetchPR(fetchDir, fmt.Sprintf("%d", t.Number))
			if err != nil {
				continue
			}
			pr.Source = "manual"
			pr.Repo = fetchName
			pr.RepoDir = fetchDir
			prs = append(prs, *pr)
		}

		return watchlistLoadedMsg{prs: prs}
	}
}

func fetchSingleWatchlistPRCmd(repoDir string, key ItemKey, numberOrURL string) tea.Cmd {
	return func() tea.Msg {
		pr, err := gh.FetchPR(repoDir, numberOrURL)
		if err != nil {
			return watchlistAddErrorMsg{err: err}
		}
		pr.Source = "manual"
		pr.Repo = key.Repo
		pr.RepoDir = repoDir
		return watchlistPRAddedMsg{pr: *pr}
	}
}
