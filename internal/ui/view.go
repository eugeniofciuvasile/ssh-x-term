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

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))
)

// View renders the UI model
func (m *Model) View() string {
	var content strings.Builder
	content.WriteString(titleStyle.Render("SSH-X-Term"))
	content.WriteString("\n")

	// If loading, show spinner but still show relevant content
	if m.loading {
		content.WriteString(loadingStyle.Render(m.spinner.View() + " Loading..."))
		content.WriteString("\n\n")
	}

	// Show current component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		defer func() {
			if r := recover(); r != nil {
				content.WriteString("\n")
				content.WriteString(errorStyle.Render("Component error: invalid UI state"))
			}
		}()

		// Don't render component details while loading for a cleaner look
		// except for terminal which should always be shown
		if !m.loading || m.state == StateSSHTerminal {
			content.WriteString(activeComponent.View())
		}
	} else if !m.loading {
		content.WriteString("No active component")
	}

	// Show error message if present
	if m.errorMessage != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMessage))
		// Clear error message after displaying it once
		m.errorMessage = ""
	}

	// Show help text based on current state
	content.WriteString("\n")
	if !m.loading || m.state == StateSSHTerminal {
		switch m.state {
		case StateConnectionList:
			content.WriteString("\nPress 'a' to add, 'e' to edit, 'd' to delete, 'o' to toggle open in new terminal, 'enter' to connect, 'ctrl+c' to quit")
		case StateSSHTerminal:
			content.WriteString("\nPress 'esc' to disconnect")
		case StateSelectStorage:
			content.WriteString("\nSelect storage type, press 'enter' to confirm, 'ctrl+c' to quit")
		case StateBitwardenConfig:
			content.WriteString("\nConfigure Bitwarden, press 'enter' to confirm, 'esc' to go back")
		case StateBitwardenLogin:
			content.WriteString("\nLogin to Bitwarden, press 'enter' to confirm, 'esc' to go back")
		case StateBitwardenUnlock:
			content.WriteString("\nUnlock Bitwarden, press 'enter' to confirm, 'esc' to go back")
		case StateOrganizationSelect:
			content.WriteString("\nSelect organization, 'o' for personal vault, press 'enter' to confirm, 'esc' to go back")
		case StateCollectionSelect:
			content.WriteString("\nSelect collection, press 'enter' to confirm, 'esc' to go back")
		case StateAddConnection, StateEditConnection:
			content.WriteString("\nEdit connection details, press 'enter' to save, 'esc' to cancel")
		}
	} else {
		// If loading, just show a minimal help message
		content.WriteString("\nPlease wait... (ctrl+c to cancel)")
	}

	return appStyle.Render(content.String())
}
