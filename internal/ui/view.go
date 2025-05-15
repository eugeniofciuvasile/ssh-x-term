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

	content.WriteString(titleStyle.Render("SSH-X-Term"))
	content.WriteString("\n")

	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		defer func() {
			if r := recover(); r != nil {
				content.WriteString("\n")
				content.WriteString(errorStyle.Render("Component error: invalid UI state"))
			}
		}()
		content.WriteString(activeComponent.View())
	} else {
		content.WriteString("No active component")
	}

	if m.errorMessage != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMessage))
		m.errorMessage = ""
	}

	content.WriteString("\n")
	switch m.state {
	case StateConnectionList:
		content.WriteString("\nPress 'a' to add, 'e' to edit, 'd' to delete, 'o' to toggle open in new terminal, 'enter' to connect, 'ctrl+c' to quit")
	case StateSSHTerminal:
		content.WriteString("\nPress 'esc' to disconnect")
	}

	return appStyle.Render(content.String())
}
