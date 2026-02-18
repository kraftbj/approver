package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kraft/approver/internal/config"
	"github.com/kraft/approver/internal/ui"
)

// settingsCategory identifies which settings category is active.
type settingsCategory int

const (
	settingsCatGeneral settingsCategory = iota
	settingsCatAI
	settingsCatRepos
)

var settingsCategoryNames = []string{"General", "AI", "Repositories"}

// settingItem represents a single displayable row in the settings panel.
type settingItem struct {
	Key         string // Config field name (e.g. "poll_interval")
	Label       string // Display label
	Value       string // Current value as string
	Description string // Help text shown when selected
	Editable    bool   // false for section headers / info lines
	RestartReq  bool   // Shows "(restart required)" hint

	// Repo items: index into cfg.RepoSources, -1 for non-repo items
	RepoIdx int
	// SubField for expanded repo detail lines (e.g. "host", "features")
	SubField string
}

// settingsScreen manages the Settings screen state.
type settingsScreen struct {
	category     settingsCategory
	cursor       int
	items        []settingItem
	scrollOffset int

	// Tracks which field is being edited (for input overlay)
	editField string

	// Repo expand/collapse state: index into RepoSources → expanded
	expandedRepos map[int]bool

	// For confirmDeleteRepo: which repo index to delete
	deleteRepoIdx int
}

func newSettingsScreen() settingsScreen {
	return settingsScreen{
		expandedRepos: make(map[int]bool),
	}
}

func (s *settingsScreen) rebuildItems(cfg *config.Config) {
	s.items = nil

	switch s.category {
	case settingsCatGeneral:
		pollVal := "300"
		if cfg.PollInterval != nil {
			pollVal = strconv.Itoa(*cfg.PollInterval)
		}
		s.items = append(s.items,
			settingItem{
				Key: "poll_interval", Label: "Poll interval (seconds)", Value: pollVal,
				Description: "Seconds between background PR refreshes. 0 to disable.",
				Editable: true, RepoIdx: -1,
			},
			settingItem{
				Key: "pr_limit", Label: "PR limit", Value: strconv.Itoa(cfg.PRLimit),
				Description: "Maximum number of PRs/issues to fetch per repo.",
				Editable: true, RepoIdx: -1,
			},
			settingItem{
				Key: "worktree_dir", Label: "Worktree directory", Value: cfg.WorktreeDir,
				Description: "Base directory for git worktrees. Leave empty for default.",
				Editable: true, RestartReq: true, RepoIdx: -1,
			},
		)

	case settingsCatAI:
		reviewPrompt := cfg.ReviewPrompt
		if reviewPrompt == "" {
			reviewPrompt = "(default)"
		} else if len(reviewPrompt) > 60 {
			reviewPrompt = reviewPrompt[:57] + "..."
		}
		s.items = append(s.items,
			settingItem{
				Key: "review_prompt", Label: "Review prompt", Value: reviewPrompt,
				Description: "Custom prompt for the AI review agent. Leave empty for default.",
				Editable: true, RepoIdx: -1,
			},
			settingItem{
				Key: "allowed_tools", Label: "Review tools", Value: cfg.AllowedTools,
				Description: "Comma-separated tools for review agent (e.g. Read,Glob,Grep).",
				Editable: true, RepoIdx: -1,
			},
			settingItem{
				Key: "fix_allowed_tools", Label: "Fix tools", Value: cfg.FixAllowedTools,
				Description: "Comma-separated tools for fix agent (e.g. Read,Write,Edit,Glob,Grep).",
				Editable: true, RepoIdx: -1,
			},
		)

	case settingsCatRepos:
		if len(cfg.RepoSources) == 0 {
			s.items = append(s.items, settingItem{
				Label: "No repo sources configured (using CWD)",
				RepoIdx: -1,
			})
			return
		}
		for i, src := range cfg.RepoSources {
			// Repo header line
			label := src.Path
			if label == "" {
				label = src.ScanDir + " (scan)"
			}
			s.items = append(s.items, settingItem{
				Key: "repo", Label: label, Value: "",
				Editable: true, RestartReq: true, RepoIdx: i,
			})

			if s.expandedRepos[i] {
				host := src.Host
				if host == "" {
					host = "(auto-detect)"
				}
				features := strings.Join(src.Features, ",")
				if features == "" {
					features = strings.Join(config.DefaultFeatures, ",")
				}
				envStr := "(none)"
				if len(src.Env) > 0 {
					var parts []string
					for k, v := range src.Env {
						parts = append(parts, k+"="+v)
					}
					envStr = strings.Join(parts, ", ")
				}
				// Lookup setup_command from Repos config
				setupCmd := ""
				if cfg.Repos != nil {
					for pattern, rc := range cfg.Repos {
						if strings.Contains(src.Path, pattern) || strings.Contains(label, pattern) {
							setupCmd = rc.SetupCommand
							break
						}
					}
				}
				if setupCmd == "" {
					setupCmd = "(none)"
				}

				s.items = append(s.items,
					settingItem{
						Key: "host", Label: "  host", Value: host,
						Editable: true, RestartReq: true, RepoIdx: i, SubField: "host",
					},
					settingItem{
						Key: "features", Label: "  features", Value: features,
						Editable: true, RestartReq: true, RepoIdx: i, SubField: "features",
					},
					settingItem{
						Key: "env", Label: "  env", Value: envStr,
						Editable: false, RepoIdx: i, SubField: "env",
					},
					settingItem{
						Key: "setup_command", Label: "  setup", Value: setupCmd,
						Editable: true, RepoIdx: i, SubField: "setup_command",
					},
				)
			}
		}
	}
}

func (s *settingsScreen) HandleKey(h *home, key string) tea.Cmd {
	switch key {
	case "q":
		return tea.Quit

	case "h", "left":
		if s.category > 0 {
			s.category--
		} else {
			s.category = settingsCatRepos
		}
		s.cursor = 0
		s.scrollOffset = 0
		s.rebuildItems(h.cfg)

	case "l", "right":
		if s.category < settingsCatRepos {
			s.category++
		} else {
			s.category = settingsCatGeneral
		}
		s.cursor = 0
		s.scrollOffset = 0
		s.rebuildItems(h.cfg)

	case "j", "down":
		if s.cursor < len(s.items)-1 {
			s.cursor++
		}

	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
		}

	case "enter":
		if s.cursor < len(s.items) {
			item := s.items[s.cursor]
			if s.category == settingsCatRepos && item.RepoIdx >= 0 && item.SubField == "" {
				// Toggle expand/collapse for repo
				s.expandedRepos[item.RepoIdx] = !s.expandedRepos[item.RepoIdx]
				s.rebuildItems(h.cfg)
				return nil
			}
			if item.Editable {
				s.editField = item.Key
				if item.SubField != "" {
					s.editField = item.SubField
				}
				h.state = stateInput
				h.inputAction = inputSettingsEdit
				h.inputPrompt = fmt.Sprintf("%s: ", item.Label)
				// Pre-fill with current value (clean up display-only values)
				prefill := item.Value
				if prefill == "(default)" || prefill == "(auto-detect)" || prefill == "(none)" {
					prefill = ""
				}
				h.inputBuffer = prefill
			}
		}

	case "a":
		if s.category == settingsCatRepos {
			h.state = stateInput
			h.inputAction = inputSettingsAddRepo
			h.inputPrompt = "Repo path: "
			h.inputBuffer = ""
		}

	case "d":
		if s.category == settingsCatRepos && s.cursor < len(s.items) {
			item := s.items[s.cursor]
			if item.RepoIdx >= 0 && item.SubField == "" {
				s.deleteRepoIdx = item.RepoIdx
				h.state = stateConfirm
				h.confirmAction = confirmDeleteRepo
				h.confirmMsg = fmt.Sprintf("Remove %s from config? (y/n)", item.Label)
			}
		}

	case "?":
		h.state = stateHelp
	}

	return nil
}

func (s *settingsScreen) applyEdit(h *home, value string) tea.Cmd {
	cfg := h.cfg
	restartHint := false

	switch s.editField {
	case "poll_interval":
		v, err := strconv.Atoi(value)
		if err != nil {
			h.showError("Invalid number")
			return clearErrorAfter(3 * time.Second)
		}
		cfg.PollInterval = &v

	case "pr_limit":
		v, err := strconv.Atoi(value)
		if err != nil || v <= 0 {
			h.showError("Invalid number (must be > 0)")
			return clearErrorAfter(3 * time.Second)
		}
		cfg.PRLimit = v

	case "worktree_dir":
		cfg.WorktreeDir = value
		restartHint = true

	case "review_prompt":
		cfg.ReviewPrompt = value

	case "allowed_tools":
		cfg.AllowedTools = value

	case "fix_allowed_tools":
		cfg.FixAllowedTools = value

	// Repo sub-fields
	case "host":
		if s.cursor < len(s.items) {
			item := s.items[s.cursor]
			if item.RepoIdx >= 0 && item.RepoIdx < len(cfg.RepoSources) {
				cfg.RepoSources[item.RepoIdx].Host = value
				restartHint = true
			}
		}

	case "features":
		if s.cursor < len(s.items) {
			item := s.items[s.cursor]
			if item.RepoIdx >= 0 && item.RepoIdx < len(cfg.RepoSources) {
				var features []string
				for _, f := range strings.Split(value, ",") {
					f = strings.TrimSpace(f)
					if f != "" {
						features = append(features, f)
					}
				}
				cfg.RepoSources[item.RepoIdx].Features = features
				restartHint = true
			}
		}

	case "setup_command":
		if s.cursor < len(s.items) {
			item := s.items[s.cursor]
			if item.RepoIdx >= 0 && item.RepoIdx < len(cfg.RepoSources) {
				src := cfg.RepoSources[item.RepoIdx]
				repoKey := src.Path
				if repoKey == "" {
					repoKey = src.ScanDir
				}
				if cfg.Repos == nil {
					cfg.Repos = make(map[string]config.RepoConfig)
				}
				cfg.Repos[repoKey] = config.RepoConfig{SetupCommand: value}
			}
		}
	}

	s.rebuildItems(cfg)

	if restartHint {
		h.showInfo("Saved — restart to apply")
	} else {
		h.showInfo("Saved")
	}

	return tea.Batch(saveConfigCmd(cfg), clearErrorAfter(3*time.Second))
}

func (s *settingsScreen) applyAddRepo(h *home, path string) tea.Cmd {
	absPath, err := config.ExpandPath(path)
	if err != nil {
		h.showError(fmt.Sprintf("Invalid path: %v", err))
		return clearErrorAfter(3 * time.Second)
	}
	if !config.IsGitRepo(absPath) {
		h.showError(fmt.Sprintf("%s is not a git repo", absPath))
		return clearErrorAfter(3 * time.Second)
	}

	// Check for duplicates
	for _, src := range h.cfg.RepoSources {
		resolved, err := config.ExpandPath(src.Path)
		if err == nil && resolved == absPath {
			h.showError("Already tracked")
			return clearErrorAfter(3 * time.Second)
		}
	}

	h.cfg.RepoSources = append(h.cfg.RepoSources, config.RepoSource{Path: absPath})
	s.rebuildItems(h.cfg)
	h.showInfo("Repo added — restart to apply")
	return tea.Batch(saveConfigCmd(h.cfg), clearErrorAfter(3*time.Second))
}

func (s *settingsScreen) removeRepo(h *home) tea.Cmd {
	idx := s.deleteRepoIdx
	if idx < 0 || idx >= len(h.cfg.RepoSources) {
		return nil
	}
	h.cfg.RepoSources = append(h.cfg.RepoSources[:idx], h.cfg.RepoSources[idx+1:]...)
	delete(s.expandedRepos, idx)
	// Re-key expanded repos above deleted index
	updated := make(map[int]bool)
	for k, v := range s.expandedRepos {
		if k > idx {
			updated[k-1] = v
		} else {
			updated[k] = v
		}
	}
	s.expandedRepos = updated
	s.rebuildItems(h.cfg)
	if s.cursor >= len(s.items) {
		s.cursor = max(0, len(s.items)-1)
	}
	h.showInfo("Repo removed — restart to apply")
	return tea.Batch(saveConfigCmd(h.cfg), clearErrorAfter(3*time.Second))
}

func (s *settingsScreen) Hints() []ui.KeyHint {
	return ui.SettingsHints()
}

func (s *settingsScreen) View(h *home, height int) string {
	listWidth, detailWidth := screenLayout(h.width)

	// Left panel: category list
	var catLines []string
	for i, name := range settingsCategoryNames {
		cursor := "  "
		if settingsCategory(i) == s.category {
			cursor = "> "
		}
		line := cursor + name
		if settingsCategory(i) == s.category {
			catLines = append(catLines, ui.SelectedStyle.Render(line))
		} else {
			catLines = append(catLines, line)
		}
	}
	catContent := lipgloss.JoinVertical(lipgloss.Left, catLines...)
	leftPanel := ui.ListPanelStyle.Width(listWidth).Height(height).Render(catContent)

	// Right panel: settings detail
	var lines []string
	lines = append(lines, ui.TitleStyle.Render(settingsCategoryNames[s.category]+" Settings"))
	lines = append(lines, "")

	// Scrolling
	visibleHeight := height - 4 // title + padding
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if s.cursor < s.scrollOffset {
		s.scrollOffset = s.cursor
	}
	if s.cursor >= s.scrollOffset+visibleHeight {
		s.scrollOffset = s.cursor - visibleHeight + 1
	}

	end := s.scrollOffset + visibleHeight
	if end > len(s.items) {
		end = len(s.items)
	}

	for i := s.scrollOffset; i < end; i++ {
		item := s.items[i]

		cursor := "  "
		if i == s.cursor {
			cursor = "> "
		}

		var line string
		if s.category == settingsCatRepos && item.RepoIdx >= 0 && item.SubField == "" {
			// Repo header: show expand indicator
			indicator := "+"
			if s.expandedRepos[item.RepoIdx] {
				indicator = "-"
			}
			line = fmt.Sprintf("%s%s %s", cursor, indicator, item.Label)
		} else if item.SubField != "" {
			// Repo sub-field
			line = fmt.Sprintf("%s%s: %s", cursor, item.Label, ui.DimStyle.Render(item.Value))
		} else if !item.Editable {
			// Non-editable info line
			line = fmt.Sprintf("%s%s", cursor, ui.DimStyle.Render(item.Label))
		} else {
			// Standard setting
			valDisplay := item.Value
			if valDisplay == "" {
				valDisplay = "(empty)"
			}
			line = fmt.Sprintf("%s%-24s %s", cursor, item.Label, valDisplay)
		}

		if item.RestartReq && item.Editable {
			line += ui.DimStyle.Render(" (restart required)")
		}

		if i == s.cursor {
			lines = append(lines, ui.SelectedStyle.Render(line))
		} else {
			lines = append(lines, line)
		}
	}

	// Description for selected item
	if s.cursor >= 0 && s.cursor < len(s.items) {
		item := s.items[s.cursor]
		if item.Description != "" {
			// Pad to fill remaining space then show description
			for len(lines) < visibleHeight+1 {
				lines = append(lines, "")
			}
			lines = append(lines, ui.DimStyle.Render("  "+item.Description))
		}
	}

	rightContent := lipgloss.JoinVertical(lipgloss.Left, lines...)
	rightPanel := ui.DetailPanelStyle.Width(detailWidth).Height(height).Render(rightContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}
