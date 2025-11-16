package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Header style - always at top of screen
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("235")).
			Padding(0, 2).
			Width(0) // Will be set dynamically

	// Footer style - always at bottom of screen
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("235")).
			Padding(0, 2).
			Width(0) // Will be set dynamically

	// Content style - fills the middle space
	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")).
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

	// Render header
	headerText := "SSH-X-Term"
	if m.loading {
		headerText += " " + m.spinner.View() + " Loading..."
	}
	header := headerStyle.Render(headerText)

	// Render content
	var contentBuilder strings.Builder

	// Show error message if present
	if m.errorMessage != "" {
		contentBuilder.WriteString(errorStyle.Render(m.errorMessage))
		contentBuilder.WriteString("\n")
		// Clear error message after displaying it once
		m.errorMessage = ""
	}

	// Show current component
	if activeComponent := m.getActiveComponent(); activeComponent != nil {
		defer func() {
			if r := recover(); r != nil {
				contentBuilder.WriteString(errorStyle.Render("Component error: invalid UI state"))
			}
		}()

		// Don't render component details while loading for a cleaner look
		// except for terminal which should always be shown
		if !m.loading || m.state == StateSSHTerminal {
			contentBuilder.WriteString(activeComponent.View())
		}
	} else if !m.loading {
		contentBuilder.WriteString(contentStyle.Render("No active component"))
	}

	content := contentBuilder.String()

	// For terminal state, ensure content fills the available space
	// The terminal component already includes its own header and footer
	if m.state == StateSSHTerminal {
		// Terminal content should already fill the space, just ensure proper height
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
	if m.loading && m.state != StateSSHTerminal {
		return "Please wait... (ctrl+c to cancel)"
	}

	switch m.state {
	case StateConnectionList:
		return "a: add | e: edit | d: delete | o: toggle new terminal | enter: connect | ctrl+c: quit"
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
