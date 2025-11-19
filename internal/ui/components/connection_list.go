package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Table styling
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder).
				BorderBottom(true).
				Padding(0, 1)

	tableRowStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	tableRowSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorBackground).
				Background(ColorPrimary).
				Bold(true).
				Padding(0, 1)

	tableRowHoverStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Padding(0, 1)

	tableBorderStyle = lipgloss.NewStyle().
				Foreground(ColorBorder)

	checkboxStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	emptyStateStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted).
			Padding(2, 4).
			Align(lipgloss.Center)
)

type ToggleOpenInNewTerminalMsg struct{}

type ConnectionList struct {
	connections       []config.SSHConnection
	selectedIndex     int
	selectedConn      *config.SSHConnection
	highlightedConn   *config.SSHConnection
	openInNewTerminal bool
	width             int
	height            int
	scrollOffset      int
	filterText        string
	filtering         bool
}

func NewConnectionList(connections []config.SSHConnection) *ConnectionList {
	var highlighted *config.SSHConnection
	if len(connections) > 0 {
		highlighted = &connections[0]
	}

	return &ConnectionList{
		connections:       connections,
		selectedIndex:     0,
		highlightedConn:   highlighted,
		openInNewTerminal: config.IsTmuxAvailable,
	}
}

func sendRefresh() tea.Cmd {
	return tea.Tick(time.Millisecond, func(t time.Time) tea.Msg {
		return ToggleOpenInNewTerminalMsg{}
	})
}

func (cl *ConnectionList) Init() tea.Cmd { return nil }

func (cl *ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ToggleOpenInNewTerminalMsg:
		return cl, nil

	case tea.WindowSizeMsg:
		cl.SetSize(msg.Width, msg.Height)
		return cl, nil

	case tea.KeyMsg:
		if cl.filtering {
			switch msg.String() {
			case "esc":
				cl.filtering = false
				cl.filterText = ""
				return cl, nil
			case "enter":
				cl.filtering = false
				return cl, nil
			case "backspace":
				if len(cl.filterText) > 0 {
					cl.filterText = cl.filterText[:len(cl.filterText)-1]
				}
				return cl, nil
			default:
				if len(msg.String()) == 1 {
					cl.filterText += msg.String()
				}
				return cl, nil
			}
		}

		switch msg.String() {
		case "up", "k":
			if cl.selectedIndex > 0 {
				cl.selectedIndex--
				cl.updateHighlighted()
			}
		case "down", "j":
			if cl.selectedIndex < len(cl.connections)-1 {
				cl.selectedIndex++
				cl.updateHighlighted()
			}
		case "enter":
			if len(cl.connections) > 0 && cl.selectedIndex < len(cl.connections) {
				cl.selectedConn = &cl.connections[cl.selectedIndex]
				return cl, nil
			}
		case "o":
			cl.openInNewTerminal = !cl.openInNewTerminal
			return cl, sendRefresh()
		case "/":
			cl.filtering = true
			cl.filterText = ""
			return cl, nil
		}
	}

	return cl, nil
}

func (cl *ConnectionList) View() string {
	if len(cl.connections) == 0 {
		emptyMsg := "No connections found.\n\nPress 'a' to add your first connection."
		return StyleContainer.Render(
			StyleTitle.Render("SSH Connections") + "\n\n" +
				emptyStateStyle.Render(emptyMsg),
		)
	}

	var b strings.Builder

	// Title with checkbox for new terminal option
	title := "SSH Connections"
	checkbox := "[ ]"
	if cl.openInNewTerminal {
		checkbox = checkboxStyle.Render("[✓]")
	} else {
		checkbox = StyleTextMuted.Render("[ ]")
	}
	titleLine := StyleTitle.Render(title) + " " +
		StyleTextMuted.Render("Open in new terminal:") + " " + checkbox

	b.WriteString(titleLine)
	b.WriteString("\n\n")

	// Filter indicator
	if cl.filtering {
		filterPrompt := StyleInfo.Render("Filter: ") + StyleFocused.Render(cl.filterText + "█")
		b.WriteString(filterPrompt)
		b.WriteString("\n\n")
	}

	// Table header
	nameWidth := 25
	hostWidth := 30
	userWidth := 15
	portWidth := 8

	// Adjust widths if terminal is narrow
	if cl.width < 100 {
		nameWidth = 20
		hostWidth = 25
		userWidth = 12
		portWidth = 6
	}

	headerName := tableHeaderStyle.Width(nameWidth).Render("NAME")
	headerHost := tableHeaderStyle.Width(hostWidth).Render("HOST")
	headerUser := tableHeaderStyle.Width(userWidth).Render("USER")
	headerPort := tableHeaderStyle.Width(portWidth).Render("PORT")
	headerAuth := tableHeaderStyle.Width(15).Render("AUTH")

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		headerName,
		tableBorderStyle.Render("│"),
		headerHost,
		tableBorderStyle.Render("│"),
		headerUser,
		tableBorderStyle.Render("│"),
		headerPort,
		tableBorderStyle.Render("│"),
		headerAuth,
	))
	b.WriteString("\n")

	// Table rows
	visibleHeight := cl.height - 10 // Account for header, title, and footer
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Calculate scroll offset
	if cl.selectedIndex < cl.scrollOffset {
		cl.scrollOffset = cl.selectedIndex
	}
	if cl.selectedIndex >= cl.scrollOffset+visibleHeight {
		cl.scrollOffset = cl.selectedIndex - visibleHeight + 1
	}

	endIndex := cl.scrollOffset + visibleHeight
	if endIndex > len(cl.connections) {
		endIndex = len(cl.connections)
	}

	for i := cl.scrollOffset; i < endIndex; i++ {
		conn := cl.connections[i]

		// Determine row style
		var rowStyle lipgloss.Style
		if i == cl.selectedIndex {
			rowStyle = tableRowSelectedStyle
		} else {
			rowStyle = tableRowStyle
		}

		// Format fields
		name := truncate(conn.Name, nameWidth-2)
		host := truncate(conn.Host, hostWidth-2)
		user := truncate(conn.Username, userWidth-2)
		port := fmt.Sprintf("%d", conn.Port)
		if conn.Port == 0 {
			port = "22"
		}
		
		auth := "Key"
		if conn.UsePassword {
			auth = "Password"
		}

		rowName := rowStyle.Width(nameWidth).Render(name)
		rowHost := rowStyle.Width(hostWidth).Render(host)
		rowUser := rowStyle.Width(userWidth).Render(user)
		rowPort := rowStyle.Width(portWidth).Render(port)
		rowAuth := rowStyle.Width(15).Render(auth)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			rowName,
			tableBorderStyle.Render("│"),
			rowHost,
			tableBorderStyle.Render("│"),
			rowUser,
			tableBorderStyle.Render("│"),
			rowPort,
			tableBorderStyle.Render("│"),
			rowAuth,
		))
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(cl.connections) > visibleHeight {
		scrollInfo := fmt.Sprintf("\n%s Showing %d-%d of %d connections",
			StyleTextMuted.Render("↕"),
			cl.scrollOffset+1,
			endIndex,
			len(cl.connections))
		b.WriteString(StyleHelp.Render(scrollInfo))
	}

	return StyleContainer.Render(b.String())
}

func (cl *ConnectionList) SelectedConnection() *config.SSHConnection { return cl.selectedConn }

func (cl *ConnectionList) HighlightedConnection() *config.SSHConnection {
	return cl.highlightedConn
}

func (cl *ConnectionList) OpenInNewTerminal() bool { return cl.openInNewTerminal }

func (cl *ConnectionList) ToggleOpenInNewTerminal() {
	cl.openInNewTerminal = !cl.openInNewTerminal
}

func (cl *ConnectionList) SetConnections(connections []config.SSHConnection) {
	cl.connections = connections
	if cl.selectedIndex >= len(connections) {
		cl.selectedIndex = len(connections) - 1
	}
	if cl.selectedIndex < 0 {
		cl.selectedIndex = 0
	}
	cl.updateHighlighted()
}

func (cl *ConnectionList) List() interface{} { return nil } // Kept for compatibility

func (cl *ConnectionList) Reset() {
	cl.selectedConn = nil
	cl.selectedIndex = 0
	cl.scrollOffset = 0
	cl.updateHighlighted()
}

func (cl *ConnectionList) SetSize(width, height int) {
	if width <= 0 {
		width = 60
	}
	if height <= 0 {
		height = 20
	}
	cl.width = width
	cl.height = height
}

func (cl *ConnectionList) updateHighlighted() {
	if len(cl.connections) > 0 && cl.selectedIndex < len(cl.connections) {
		cl.highlightedConn = &cl.connections[cl.selectedIndex]
	} else {
		cl.highlightedConn = nil
	}
}

// truncate truncates a string to the specified length, adding "..." if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
