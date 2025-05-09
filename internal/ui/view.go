package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")).
			MarginTop(1)
)

// View renders the UI model
func (m *Model) View() string {
	var content strings.Builder

	// Render header
	content.WriteString(titleStyle.Render("SSH-X-Term"))
	content.WriteString("\n")

	// Render active component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		content.WriteString(activeComponent.View())
	} else {
		content.WriteString("No active component")
	}

	// Render error message if any
	if m.errorMessage != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMessage))
		// Clear error after displaying it
		m.errorMessage = ""
	}

	// Render footer with help text based on current state
	content.WriteString("\n")
	switch m.state {
	case StateConnectionList:
		content.WriteString("\nPress 'a' to add, 'e' to edit, 'd' to delete, 'o' to toggle open in new terminal, 'enter' to connect, 'ctrl+c' to quit")
	case StateSSHTerminal:
		content.WriteString("\nPress 'esc' to disconnect")
	}

	return appStyle.Render(content.String())
}
