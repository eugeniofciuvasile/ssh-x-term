package components

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/eugeniofciuvasile/ssh-x-term/internal/config"
)

type ToggleOpenInNewTerminalMsg struct{}

type connectionItem struct {
	connection config.SSHConnection
}

func (i connectionItem) FilterValue() string { return i.connection.Name }

// connectionDelegate handles the rendering of each list item with dynamic widths
type connectionDelegate struct {
	nameWidth int
	hostWidth int
	userWidth int
	portWidth int
	authWidth int
}

func (d connectionDelegate) Height() int { return 1 }

func (d connectionDelegate) Spacing() int { return 0 }

func (d connectionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d connectionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(connectionItem)
	if !ok {
		return
	}

	conn := i.connection

	// Determine Authorization Method string
	authMethod := "Agent"
	if conn.UsePassword {
		authMethod = "Password"
	} else if conn.KeyFile != "" {
		authMethod = "Key File"
	}

	// Format columns using the dynamic widths stored in the delegate
	name := truncate(conn.Name, d.nameWidth)
	host := truncate(conn.Host, d.hostWidth)
	user := truncate(conn.Username, d.userWidth)
	port := truncate(fmt.Sprintf("%d", conn.Port), d.portWidth)
	auth := truncate(authMethod, d.authWidth)

	// Render row
	var style lipgloss.Style
	if index == m.Index() {
		style = selectedItemStyle
	} else {
		style = itemStyle
	}

	// Build the row string using Lipgloss for alignment
	row := lipgloss.JoinHorizontal(lipgloss.Left,
		lipgloss.NewStyle().Width(d.nameWidth).Render(name),
		lipgloss.NewStyle().Width(d.hostWidth).Render(host),
		lipgloss.NewStyle().Width(d.userWidth).Render(user),
		lipgloss.NewStyle().Width(d.portWidth).Render(port),
		lipgloss.NewStyle().Width(d.authWidth).Render(auth),
	)

	fmt.Fprint(w, style.Render(row))
}

// Helper to truncate strings that are too long
func truncate(s string, max int) string {
	if max < 3 {
		return ""
	}
	if len(s) > max-1 {
		return s[:max-2] + "…"
	}
	return s
}

type ConnectionList struct {
	list              list.Model
	connections       []config.SSHConnection
	selectedConn      *config.SSHConnection
	highlightedConn   *config.SSHConnection
	openInNewTerminal bool

	// layout stores the current column widths for header rendering
	layout connectionDelegate
}

func NewConnectionList(connections []config.SSHConnection) *ConnectionList {
	items := make([]list.Item, len(connections))
	for i, conn := range connections {
		items[i] = connectionItem{connection: conn}
	}

	// Initial delegate with default widths (will be resized immediately)
	defaultDelegate := connectionDelegate{
		nameWidth: 20, hostWidth: 20, userWidth: 15, portWidth: 8, authWidth: 10,
	}

	l := list.New(items, defaultDelegate, 80, 20)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.PaginationStyle = paginationStyle

	var highlighted *config.SSHConnection
	if len(connections) > 0 {
		highlighted = &connections[0]
	}

	cl := &ConnectionList{
		list:              l,
		connections:       connections,
		highlightedConn:   highlighted,
		openInNewTerminal: config.IsTmuxAvailable,
		layout:            defaultDelegate,
	}

	// Trigger an initial layout calculation
	cl.SetSize(80, 20)

	return cl
}

func (cl *ConnectionList) Init() tea.Cmd { return nil }

func (cl *ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ToggleOpenInNewTerminalMsg:
		return cl, nil

	case tea.WindowSizeMsg:
		cl.SetSize(msg.Width, msg.Height)
		return cl, nil

	case tea.KeyMsg:
		if cl.list.FilterState() == list.Filtering {
			newList, cmd := cl.list.Update(msg)
			cl.list = newList
			return cl, cmd
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if selectedItem := cl.list.SelectedItem(); selectedItem != nil {
				if connItem, ok := selectedItem.(connectionItem); ok {
					cl.selectedConn = &connItem.connection
					return cl, nil
				}
			}
		}
	}

	newList, cmd := cl.list.Update(msg)
	cl.list = newList
	if item := cl.list.SelectedItem(); item != nil {
		if connItem, ok := item.(connectionItem); ok {
			cl.highlightedConn = &connItem.connection
		}
	} else {
		cl.highlightedConn = nil
	}
	return cl, cmd
}

func (cl *ConnectionList) View() string {
	if len(cl.connections) == 0 {
		// Simplified message since global title handles context
		return "\n\n  No connections found. Press 'a' to add a connection.\n\n"
	}

	// Note: We no longer render the Title here because the main UI View() handles it.
	// We only render the Table Headers and the List itself.

	// Construct Table Headers using the DYNAMIC layout widths
	headers := lipgloss.NewStyle().PaddingLeft(2).Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			headerStyle.Width(cl.layout.nameWidth).Render("Name"),
			headerStyle.Width(cl.layout.hostWidth).Render("Host"),
			headerStyle.Width(cl.layout.userWidth).Render("User"),
			headerStyle.Width(cl.layout.portWidth).Render("Port"),
			headerStyle.Width(cl.layout.authWidth).Render("Auth Method"),
		),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		headers,
		cl.list.View(),
	)
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
	items := make([]list.Item, len(connections))
	for i, conn := range connections {
		items[i] = connectionItem{connection: conn}
	}
	cl.list.SetItems(items)
}

func (cl *ConnectionList) List() *list.Model { return &cl.list }

func (cl *ConnectionList) Reset() {
	cl.selectedConn = nil
	cl.list.Select(0)
	if len(cl.connections) > 0 {
		cl.highlightedConn = &cl.connections[0]
	} else {
		cl.highlightedConn = nil
	}
}

// SetSize recalculates the column widths based on available terminal space
func (cl *ConnectionList) SetSize(width, height int) {
	// Calculate height deduction:
	// 1. Global Header (managed by parent) -> 1 line
	// 2. Global Footer (managed by parent) -> 1 line
	// 3. Table Header (managed here) -> 1 line
	// Total overhead = 3 lines
	listHeight := max(height-2, 1)
	cl.list.SetHeight(listHeight)
	cl.list.SetWidth(width)

	// Calculate column widths
	cl.recalculateTableLayout(width)
}

func (cl *ConnectionList) recalculateTableLayout(totalWidth int) {
	// Subtract list padding (default is usually 2 for left padding)
	availableWidth := max(totalWidth-4, 0)

	// Define fixed or minimum widths
	const (
		minPortWidth = 12
		minAuthWidth = 16
	)

	// Strategy:
	// Port and Auth get fixed sizes if space is tight, or small slice of total
	portW := minPortWidth
	authW := minAuthWidth

	remaining := availableWidth - portW - authW

	// Distribute remaining space:
	// Name: 35%, Host: 35%, User: 30%
	nameW := int(float64(remaining) * 0.35)
	hostW := int(float64(remaining) * 0.35)
	userW := remaining - nameW - hostW // Give remainder to user to avoid rounding gaps

	// Ensure minimums
	if nameW < 10 {
		nameW = 10
	}
	if hostW < 10 {
		hostW = 10
	}
	if userW < 5 {
		userW = 5
	}

	// Create new delegate with calculated widths
	newLayout := connectionDelegate{
		nameWidth: nameW,
		hostWidth: hostW,
		userWidth: userW,
		portWidth: portW,
		authWidth: authW,
	}

	// Store layout for Header rendering
	cl.layout = newLayout

	// Update the list's delegate so rows render with new widths
	cl.list.SetDelegate(newLayout)
}
