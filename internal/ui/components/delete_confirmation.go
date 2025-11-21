package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DeleteConfirmation struct {
	connectionName string
	confirmed      bool
	canceled       bool
	width          int
	height         int
}

func NewDeleteConfirmation(connectionName string) *DeleteConfirmation {
	return &DeleteConfirmation{
		connectionName: connectionName,
	}
}

func (d *DeleteConfirmation) Init() tea.Cmd {
	return nil
}

func (d *DeleteConfirmation) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if d.confirmed || d.canceled {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.SetSize(msg.Width, msg.Height)
		return d, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			d.confirmed = true
			return d, nil
		case "n", "N", "esc":
			d.canceled = true
			return d, nil
		case "ctrl+c":
			d.canceled = true
			return d, nil
		}
	}

	return d, nil
}

func (d *DeleteConfirmation) View() string {
	if d.canceled {
		return ""
	}

	// Build the confirmation message
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")). // Red color for warning
		Render("⚠ Delete Connection")

	message := lipgloss.NewStyle().
		Foreground(colorSubText).
		Render("Are you sure you want to delete this connection?")

	connectionDisplay := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render(d.connectionName)

	prompt := lipgloss.NewStyle().
		Foreground(colorInactive).
		Render("Press Y to confirm, N or Esc to cancel")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"\n",
		message,
		"\n",
		connectionDisplay,
		"\n\n",
		prompt,
	)

	// Wrap in a bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border
		Padding(1, 3).
		Width(60).
		Align(lipgloss.Center).
		Render(content)

	// Center on screen
	availableHeight := max(d.height-3, 0)
	return lipgloss.Place(
		d.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (d *DeleteConfirmation) SetSize(width, height int) {
	d.width = width
	d.height = height
}

func (d *DeleteConfirmation) IsConfirmed() bool {
	return d.confirmed
}

func (d *DeleteConfirmation) IsCanceled() bool {
	return d.canceled
}
