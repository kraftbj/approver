package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// KeyHint represents a single keybinding hint.
type KeyHint struct {
	Key  string
	Desc string
}

// Menu renders the bottom menu bar with context-sensitive key hints.
type Menu struct {
	Width int
}

func NewMenu() Menu {
	return Menu{}
}

// SetWidth updates the menu width.
func (m *Menu) SetWidth(w int) {
	m.Width = w
}

// screenNames maps screen index to display name.
var screenNames = []string{"Reviews", "Issues", "Watchlist"}

// ScreenIndicator renders the screen switcher like "[1:Reviews] 2:Issues 3:Watchlist".
func ScreenIndicator(activeIdx int) string {
	activeStyle := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(ColorDimGray)

	var parts []string
	for i, name := range screenNames {
		label := fmt.Sprintf("%d:%s", i+1, name)
		if i == activeIdx {
			parts = append(parts, activeStyle.Render("["+label+"]"))
		} else {
			parts = append(parts, inactiveStyle.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

// DefaultHints returns the standard keybindings for the default state.
func DefaultHints() []KeyHint {
	return []KeyHint{
		{"j/k", "navigate"},
		{"w", "worktree"},
		{"c", "review"},
		{"t", "claude session"},
		{"Tab", "cycle view"},
		{"R", "refresh"},
		{"A", "approve"},
		{"X", "request changes"},
		{"u", "update branch"},
		{"o", "open"},
		{"?", "help"},
		{"q", "quit"},
	}
}

// ConfirmHints returns keybindings during confirmation.
func ConfirmHints() []KeyHint {
	return []KeyHint{
		{"y", "confirm"},
		{"n/esc", "cancel"},
	}
}

// InputHints returns keybindings during text input.
func InputHints() []KeyHint {
	return []KeyHint{
		{"enter", "submit"},
		{"esc", "cancel"},
	}
}

// View renders the menu bar with the given hints (no screen indicator).
func (m *Menu) View(hints []KeyHint) string {
	return m.ViewWithScreen(hints, "")
}

// ViewWithScreen renders the menu bar with an optional screen indicator prefix.
func (m *Menu) ViewWithScreen(hints []KeyHint, screenIndicator string) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(ColorDimGray)

	var parts []string
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.Key)+": "+h.Desc)
	}

	hintsContent := strings.Join(parts, sepStyle.Render("  "))

	var content string
	if screenIndicator != "" {
		content = screenIndicator + sepStyle.Render("  |  ") + hintsContent
	} else {
		content = hintsContent
	}

	return MenuBarStyle.Width(m.Width).Render(content)
}
