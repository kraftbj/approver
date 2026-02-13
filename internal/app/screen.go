package app

import tea "github.com/charmbracelet/bubbletea"

// activeScreen identifies which screen is currently displayed.
type activeScreen int

const (
	screenReviews  activeScreen = iota
	screenIssues
	screenWatchlist
)

// Screen defines the interface for each top-level screen.
// Screens handle their own key events and rendering, but delegate
// domain state management to the home struct.
type Screen interface {
	// HandleKey processes a key press and returns a command.
	HandleKey(h *home, key string) tea.Cmd

	// View renders the screen content within the given height.
	View(h *home, height int) string
}
