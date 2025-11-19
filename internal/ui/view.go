package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/ui/components"
)

// View renders the UI model with full-screen layout
func (m *Model) View() string {
	// Calculate available space
	// Header: 1 line for title + state
	// Footer: 1 line for help text
	// Content: remaining space

	headerHeightLines := 1
	footerHeightLines := 1
	contentHeight := max(m.height-headerHeightLines-footerHeightLines, 3)

	// Set widths for header and footer to match terminal width
	headerStyle := components.StyleHeader.Width(m.width)
	footerStyle := components.StyleFooter.Width(m.width)

	// Render header with app title and current state
	headerText := "SSH X TERM"
	stateText := m.getStateText()
	if stateText != "" {
		headerText += " | " + stateText
	}
	header := headerStyle.Render(headerText)

	// Render content
	var contentBuilder strings.Builder

	// Show error message if present
	if m.errorMessage != "" {
		contentBuilder.WriteString(components.StyleError.Padding(0, 2).Render(m.errorMessage))
		contentBuilder.WriteString("\n")
		// Clear error message after displaying it once
		m.errorMessage = ""
	}

	// If loading, show centered spinner
	if m.loading && m.state != StateSSHTerminal && m.state != StateSCPFileManager {
		spinnerText := components.StyleSpinner.Render(m.spinner.View() + " Loading...")
		content := components.CenterContent(spinnerText, m.width, contentHeight)
		contentBuilder.WriteString(content)
	} else if activeComponent := m.getActiveComponent(); activeComponent != nil {
		// Show current component
		defer func() {
			if r := recover(); r != nil {
				contentBuilder.WriteString(components.StyleError.Padding(0, 2).Render("Component error: invalid UI state"))
			}
		}()

		contentBuilder.WriteString(activeComponent.View())
	} else {
		contentBuilder.WriteString(components.StyleContainer.Render("No active component"))
	}

	content := contentBuilder.String()

	// For terminal and SCP manager states, ensure content fills the available space
	// These components already include their own headers
	if m.state == StateSSHTerminal || m.state == StateSCPFileManager {
		// Content should already fill the space, just ensure proper height
		content = lipgloss.NewStyle().
			Height(contentHeight).
			Width(m.width).
			Render(content)
	}

	// Render footer with help text
	footerText := m.getHelpText()
	footer := footerStyle.Render(footerText)

	// Combine header, content, and footer to fill entire terminal
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

// getHelpText returns context-appropriate help text
func (m *Model) getHelpText() string {
	if m.loading && m.state != StateSSHTerminal && m.state != StateSCPFileManager {
		return "Please wait... (ctrl+c to cancel)"
	}

	switch m.state {
	case StateConnectionList:
		return "a: add | e: edit | d: delete | s: scp | o: toggle new terminal | enter: connect | ctrl+c: quit"
	case StateSSHTerminal:
		// Get detailed help text from terminal component
		if m.terminal != nil {
			if m.terminal.IsSessionClosed() {
				return "Session closed - Press ESC to return"
			}

			// Show detailed help for wider terminals, compact for narrow
			if m.terminal.GetWidth() < 80 || m.width < 80 {
				return "ESC: Exit | CTRL+D: EOF | Scroll: PgUp/PgDn"
			}

			return "ESC: Exit | CTRL+D: EOF | PgUp/PgDn: Scroll Vertically | Tab: Complete Command | Mouse: Copy Text"
		}
		return "esc: disconnect"
	case StateSCPFileManager:
		return "↑/↓: navigate | enter: open | backspace: parent | tab: switch | g: get | u: upload | d: delete | n: create | r: rename | c: cd | /: search | esc: exit"
	case StateSelectStorage:
		return "↑/↓: navigate | enter: select | ctrl+c: quit"
	case StateBitwardenConfig:
		return "tab: next field | enter: confirm | esc: back"
	case StateBitwardenLogin:
		return "tab: next field | enter: login | esc: back"
	case StateBitwardenUnlock:
		return "enter: unlock | esc: back"
	case StateOrganizationSelect:
		return "↑/↓: navigate | o: personal vault | enter: select | esc: back"
	case StateCollectionSelect:
		return "↑/↓: navigate | enter: select | esc: back"
	case StateAddConnection, StateEditConnection:
		return "tab: next field | ctrl+p: toggle auth | enter: save | esc: cancel"
	default:
		return "ctrl+c: quit"
	}
}

// getStateText returns a human-readable string for the current state
func (m *Model) getStateText() string {
	switch m.state {
	case StateSelectStorage:
		return "Storage Selection"
	case StateBitwardenConfig:
		return "Bitwarden Setup"
	case StateConnectionList:
		return "Connections"
	case StateAddConnection:
		return "Add Connection"
	case StateEditConnection:
		return "Edit Connection"
	case StateSSHTerminal:
		return "SSH Terminal"
	case StateSCPFileManager:
		return "SCP File Manager"
	case StateBitwardenLogin:
		return "Bitwarden Login"
	case StateBitwardenUnlock:
		return "Bitwarden Unlock"
	case StateOrganizationSelect:
		return "Select Organization"
	case StateCollectionSelect:
		return "Select Collection"
	default:
		return ""
	}
}
