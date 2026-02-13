package ui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorGreen   = lipgloss.Color("#22c55e")
	ColorRed     = lipgloss.Color("#ef4444")
	ColorYellow  = lipgloss.Color("#eab308")
	ColorBlue    = lipgloss.Color("#3b82f6")
	ColorGray    = lipgloss.Color("#6b7280")
	ColorDimGray = lipgloss.Color("#4b5563")
	ColorWhite   = lipgloss.Color("#f9fafb")
	ColorCyan    = lipgloss.Color("#06b6d4")
	ColorMagenta = lipgloss.Color("#a855f7")
)

// Panel styles
var (
	ListPanelStyle = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorDimGray)

	DetailPanelStyle = lipgloss.NewStyle().
				PaddingLeft(1)

	MenuBarStyle = lipgloss.NewStyle().
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorDimGray).
			Foreground(ColorGray)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	SelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e3a5f")).
			Foreground(ColorWhite)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	ManualBadgeStyle = lipgloss.NewStyle().
				Foreground(ColorCyan)
)

// CI status styling
func CIStyle(status string) lipgloss.Style {
	switch status {
	case "pass":
		return lipgloss.NewStyle().Foreground(ColorGreen)
	case "fail":
		return lipgloss.NewStyle().Foreground(ColorRed)
	case "pending":
		return lipgloss.NewStyle().Foreground(ColorYellow)
	default:
		return lipgloss.NewStyle().Foreground(ColorGray)
	}
}

// Review decision styling
func ReviewStyle(decision string) lipgloss.Style {
	switch decision {
	case "APPROVED":
		return lipgloss.NewStyle().Foreground(ColorGreen)
	case "CHANGES":
		return lipgloss.NewStyle().Foreground(ColorRed)
	case "REVIEW":
		return lipgloss.NewStyle().Foreground(ColorYellow)
	default:
		return lipgloss.NewStyle().Foreground(ColorGray)
	}
}
