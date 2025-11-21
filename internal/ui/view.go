package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Header style - always at top of screen
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("235")).
			Align(lipgloss.Center).
			Padding(0, 2).
			Width(0) // Will be set dynamically

	// Footer style - always at bottom of screen
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("235")).
			Align(lipgloss.Center).
			Padding(0, 2).
			Width(0) // Will be set dynamically

	// Content style - fills the middle space
	contentStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")).
			Align(lipgloss.Center).
			Padding(0, 2)
)

// View renders the UI model with full-screen layout
func (m *Model) View() string {
	// Calculate available space
	// Header: 1 line for title
	// Footer: 1 line for help text
	// Content: remaining space

	headerHeightLines := 1
	footerHeightLines := 1
	contentHeight := max(m.height-headerHeightLines-footerHeightLines, 3)

	// Set widths for header and footer to match terminal width
	headerStyle = headerStyle.Width(m.width)
	footerStyle = footerStyle.Width(m.width)

	// --- Dynamic Title Generation ---
	title := "SSH-X-Term"
	switch m.state {
	case StateSelectStorage:
		title = "Select Storage Provider"
	case StateBitwardenConfig:
		title = "Bitwarden Configuration"
	case StateBitwardenLogin:
		title = "Bitwarden Login"
	case StateBitwardenUnlock:
		title = "Unlock Bitwarden Vault"
	case StateOrganizationSelect:
		title = "Select Organization"
	case StateCollectionSelect:
		title = "Select Collection"
	case StateConnectionList:
		if m.connectionList != nil {
			checkboxStr := "( )"
			if m.connectionList.OpenInNewTerminal() {
				checkboxStr = "(✓)"
			}
			title = fmt.Sprintf("SSH Connections - Open in New Terminal %s", checkboxStr)
		} else {
			title = "SSH Connections"
		}
	case StateAddConnection:
		title = "Add New Connection"
	case StateEditConnection:
		title = "Edit Connection"
	case StateSSHTerminal:
		title = "Terminal Session"
	case StateSCPFileManager:
		title = "SCP File Manager"
	case StateSSHPassphrase:
		title = "SSH Authentication Required"
	}

	// Note: We removed the spinner from the header here
	header := headerStyle.Render(title)
	// --------------------------------

	// --- Render Content ---
	var content string

	// Check if we are in a blocking loading state
	// We don't block for Terminal or SCP as they handle their own async states/views
	if m.loading && m.state != StateSSHTerminal && m.state != StateSCPFileManager {

		// Create a centered container for the spinner
		spinnerView := fmt.Sprintf("%s Loading...", m.spinner.View())

		content = lipgloss.NewStyle().
			Width(m.width).
			Height(contentHeight-1).
			Align(lipgloss.Center, lipgloss.Center). // Center Horizontally and Vertically
			Render(spinnerView)

	} else {
		// Standard Component Rendering
		var contentBuilder strings.Builder

		// Show error message if present
		if m.errorMessage != "" {
			contentBuilder.WriteString(errorStyle.Render(m.errorMessage))
			contentBuilder.WriteString("\n")
			m.errorMessage = ""
		}

		if activeComponent := m.getActiveComponent(); activeComponent != nil {
			// Handle panic recovery for components
			defer func() {
				if r := recover(); r != nil {
					contentBuilder.WriteString(errorStyle.Render("Component error: invalid UI state"))
				}
			}()
			contentBuilder.WriteString(activeComponent.View())
		} else {
			contentBuilder.WriteString(contentStyle.Render("No active component"))
		}

		content = contentBuilder.String()

		// For specific states, ensure content fills the space manually
		// (Lipgloss styles inside the component usually handle this, but this is a safety net)
		if m.state == StateSSHTerminal || m.state == StateSCPFileManager {
			content = lipgloss.NewStyle().
				Height(contentHeight).
				Width(m.width).
				Render(content)
		}
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
		return "a: add | e: edit | d: delete | s: scp | / filter | o: toggle new terminal | enter: connect | ctrl+c: quit"
	case StateSSHTerminal:
		if m.terminal != nil {
			if m.terminal.IsSessionClosed() {
				return "Session closed - Press ESC to return"
			}
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
	case StateSSHPassphrase:
		return "enter: submit | esc: cancel"
	default:
		return "ctrl+c: quit"
	}
}
