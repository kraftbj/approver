package ui

import (
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

// DefaultHints returns the standard keybindings for the default state.
func DefaultHints() []KeyHint {
	return []KeyHint{
		{"j/k", "navigate"},
		{"a", "add PR"},
		{"w", "worktree"},
		{"c", "review"},
		{"t", "claude session"},
		{"Tab", "toggle view"},
		{"R", "refresh"},
		{"A", "approve"},
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

// View renders the menu bar with the given hints.
func (m *Menu) View(hints []KeyHint) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(ColorDimGray)

	var parts []string
	for _, h := range hints {
		parts = append(parts, keyStyle.Render(h.Key)+": "+h.Desc)
	}

	content := strings.Join(parts, sepStyle.Render("  "))
	return MenuBarStyle.Width(m.Width).Render(content)
}
