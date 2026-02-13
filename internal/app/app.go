package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateDefault state = iota
	stateLoading
	stateConfirm
	stateHelp
)

type home struct {
	width  int
	height int
	state  state
}

func newHome() home {
	return home{
		state: stateDefault,
	}
}

func (h home) Init() tea.Cmd {
	return nil
}

func (h home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return h, tea.Quit
		}
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
	}
	return h, nil
}

func (h home) View() string {
	return fmt.Sprintf("Approver - PR Review Manager\n\nPress q to quit.\n")
}

func Run() error {
	p := tea.NewProgram(newHome(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
